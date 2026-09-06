import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { AppShell } from "@/components/AppShell";
import { usePendingUsers, useApproveUser, useRejectUser } from "@/hooks/use-user-management";
import { roleMap } from "@/api/types";
import type { User } from "@/api/types";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, blockAdminFromPage, requireAuth, requireRole } from "@/lib/role-guards";
import { api } from "@/api/client";
import { tzs } from "@/lib/format";
import {
  Clock, CheckCircle2, XCircle, Phone, Calendar, Loader2, User as UserIcon,
} from "lucide-react";

export const Route = createFileRoute("/wanachama-kusubiri")({
  head: () => ({
    meta: [
      { title: "Wanaosubiri — Kikundi" },
      { name: "description", content: "Orodha ya watumiaji na wanachama wanaosubiri kuidhinishwa." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    requireRole("secretary");
    blockAdminFromPage();
  },
  component: PendingUsersPage,
});

interface PendingMember {
  id: string;
  member_no: string;
  full_name: string;
  phone: string;
  gender?: string | null;
  occupation?: string | null;
  email?: string | null;
  next_of_kin_name?: string | null;
  next_of_kin_phone?: string | null;
  photo_url?: string | null;
  approval_status: "pending" | "approved" | "rejected";
  registered_by: string;
  created_at: string;
  registrar?: { id: string; name: string; role: string };
}

function PendingUsersPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { data, isLoading, error, refetch } = usePendingUsers({ limit: 50 });
  const approveMutation = useApproveUser();
  const rejectMutation = useRejectUser();

  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [actionType, setActionType] = useState<"approve" | "reject" | null>(null);
  const [remarks, setRemarks] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  // Pending MEMBERS (submitted by mwenyekiti, awaiting katibu)
  const isKatibu = user?.role === "secretary";
  const { data: pendingMembersData, isLoading: membersLoading } = useQuery({
    queryKey: ["members", "pending"],
    queryFn: () => api.get<{ data: PendingMember[] }>("/members?status=pending&limit=50"),
  });
  const pendingMembers: PendingMember[] = pendingMembersData?.data ?? [];

  const memberActionMutation = useMutation({
    mutationFn: (vars: { id: string; action: "approve" | "reject"; reason?: string }) => {
      if (vars.action === "approve") {
        return api.patch(`/members/${vars.id}/approve`);
      }
      return api.patch(`/members/${vars.id}/reject`, { reason: vars.reason });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["members"] });
    },
    onError: (e: Error) => {
      alert(e.message);
    },
  });

  const [selectedMember, setSelectedMember] = useState<PendingMember | null>(null);
  const [memberAction, setMemberAction] = useState<"approve" | "reject" | null>(null);
  const [memberReason, setMemberReason] = useState("");

  if (!hasRole(user, "secretary")) {
    return (
      <AppShell title="Wanaosubiri">
        <div className="flex items-center justify-center py-20">
          <p className="text-muted-foreground">Ukurasa huu ni kwa Katibu tu — ndiye anayeidhinisha wanachama.</p>
        </div>
      </AppShell>
    );
  }

  const handleAction = async () => {
    if (!selectedUser || !actionType) return;
    setActionLoading(true);
    try {
      if (actionType === "approve") {
        await approveMutation.mutateAsync({ id: selectedUser.id, data: { remarks } });
      } else {
        await rejectMutation.mutateAsync({ id: selectedUser.id, data: { remarks } });
      }
      setSelectedUser(null);
      setActionType(null);
      setRemarks("");
    } catch {
      // error handled by mutation
    } finally {
      setActionLoading(false);
    }
  };

  const submitMemberAction = () => {
    if (!selectedMember || !memberAction) return;
    if (memberAction === "reject" && !memberReason.trim()) return; // reason required
    memberActionMutation.mutate({
      id: selectedMember.id,
      action: memberAction,
      reason: memberAction === "reject" ? memberReason.trim() : undefined,
    });
    setSelectedMember(null);
    setMemberAction(null);
    setMemberReason("");
  };

  const pendingUsers = data?.data ?? [];

  const renderMemberRow = (m: PendingMember) => (
    <div key={m.id} className="px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-start gap-3">
          {m.photo_url ? (
            <img
              src={m.photo_url}
              alt={m.full_name}
              className="h-10 w-10 shrink-0 rounded-full border object-cover"
            />
          ) : (
            <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-primary/10 text-sm font-bold text-primary">
              {m.full_name.charAt(0).toUpperCase()}
            </div>
          )}
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold">
              {m.full_name}{" "}
              <span className="text-xs font-normal text-muted-foreground">({m.member_no})</span>
            </p>
            <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <Phone className="h-3 w-3" /> {m.phone}
              </span>
              {m.gender && <span>Jinsia: {m.gender === "MME" ? "Mwanamume" : "Mwanamke"}</span>}
              {m.occupation && <span>Kazi: {m.occupation}</span>}
              {m.email && <span>{m.email}</span>}
            </div>
            {(m.next_of_kin_name || m.next_of_kin_phone) && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                Mlezi: {m.next_of_kin_name ?? "—"} {m.next_of_kin_phone ? `(${m.next_of_kin_phone})` : ""}
              </p>
            )}
            {/* Audit trail */}
            <p className="mt-1 flex items-center gap-1 text-[10px] text-muted-foreground">
              <Calendar className="h-3 w-3" />
              Imeongezwa na {m.registrar?.name ?? "—"} ·{" "}
              {new Date(m.created_at).toLocaleDateString("sw-TZ", {
                day: "numeric",
                month: "short",
                year: "numeric",
              })}
            </p>
          </div>
        </div>
        {isKatibu && (
          <div className="flex shrink-0 gap-2">
            <button
              onClick={() => {
                setSelectedMember(m);
                setMemberAction("approve");
              }}
              className="rounded-lg bg-success/10 px-3 py-1.5 text-xs font-medium text-success hover:bg-success/20"
            >
              <CheckCircle2 className="mr-1 inline h-3.5 w-3.5" />
              Idhinisha
            </button>
            <button
              onClick={() => {
                setSelectedMember(m);
                setMemberAction("reject");
              }}
              className="rounded-lg bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive hover:bg-destructive/20"
            >
              <XCircle className="mr-1 inline h-3.5 w-3.5" />
              Kataa
            </button>
          </div>
        )}
      </div>
    </div>
  );

  return (
    <AppShell title="Wanaosubiri Kuidhinishwa" subtitle="Watumiaji na wanachama walioundwa na Mwenyekiti">
      {/* ---- Pending MEMBERS (katibu approval queue) ---- */}
      <div className="mb-3 flex items-center gap-2">
        <h2 className="font-display text-base font-semibold">Wanachama Wanaosubiri</h2>
        {pendingMembers.length > 0 && (
          <span className="chip bg-amber-100 text-amber-700 text-[10px]">{pendingMembers.length}</span>
        )}
      </div>
      {membersLoading ? (
        <div className="card-surface animate-pulse p-4">
          <div className="h-4 w-1/3 rounded bg-muted" />
        </div>
      ) : pendingMembers.length === 0 ? (
        <div className="card-surface flex flex-col items-center px-4 py-8 text-center">
          <CheckCircle2 className="mb-2 h-8 w-8 text-success" />
          <p className="text-sm text-muted-foreground">Hakuna mwanachama anayesubiri idhini.</p>
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {pendingMembers.map(renderMemberRow)}
        </div>
      )}

      {/* ---- Pending USERS (account approval) ---- */}
      <div className="mb-3 mt-7 flex items-center gap-2">
        <h2 className="font-display text-base font-semibold">Akaunti za Watumiaji</h2>
        {pendingUsers.length > 0 && (
          <span className="chip bg-amber-100 text-amber-700 text-[10px]">{pendingUsers.length}</span>
        )}
      </div>
      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <div key={i} className="card-surface animate-pulse px-4 py-4">
              <div className="h-4 w-1/3 rounded bg-muted" />
              <div className="mt-2 h-3 w-1/2 rounded bg-muted" />
            </div>
          ))}
        </div>
      ) : error ? (
        <div className="card-surface px-4 py-8 text-center">
          <p className="text-sm text-destructive">Imeshindikana kupakua data.</p>
          <button onClick={() => refetch()} className="mt-2 text-sm font-medium text-primary">Jaribu tena</button>
        </div>
      ) : pendingUsers.length === 0 ? (
        <div className="card-surface flex flex-col items-center px-4 py-12 text-center">
          <CheckCircle2 className="mb-3 h-10 w-10 text-success" />
          <p className="text-sm font-medium">Hakuna mtumiaji anayesubiri.</p>
          <p className="text-xs text-muted-foreground">Watumiaji wote wamekaguliwa.</p>
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {pendingUsers.map((u) => (
            <div key={u.id} className="flex items-center justify-between px-4 py-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                    {u.name.charAt(0).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{u.name}</p>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1"><Phone className="h-3 w-3" />{u.phone}</span>
                      <span className="chip bg-amber-100 text-amber-700 text-[10px]">{roleMap[u.role] ?? u.role}</span>
                    </div>
                  </div>
                </div>
                <p className="mt-1 flex items-center gap-1 text-[10px] text-muted-foreground">
                  <Calendar className="h-3 w-3" />
                  {new Date(u.created_at).toLocaleDateString("sw-TZ", { day: "numeric", month: "short", year: "numeric" })}
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => { setSelectedUser(u); setActionType("approve"); }}
                  className="rounded-lg bg-success/10 px-3 py-1.5 text-xs font-medium text-success hover:bg-success/20"
                >
                  <CheckCircle2 className="inline h-3.5 w-3.5 mr-1" />
                  Idhinisha
                </button>
                <button
                  onClick={() => { setSelectedUser(u); setActionType("reject"); }}
                  className="rounded-lg bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive hover:bg-destructive/20"
                >
                  <XCircle className="inline h-3.5 w-3.5 mr-1" />
                  Kataa
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Member approve/reject modal — reason REQUIRED on reject */}
      {selectedMember && memberAction && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center"
          onClick={() => { setSelectedMember(null); setMemberAction(null); }}
        >
          <div
            className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="font-display text-lg font-bold">
              {memberAction === "approve" ? "Idhinisha Mwanachama" : "Kataa Mwanachama"}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {selectedMember.full_name} — {selectedMember.member_no}
            </p>
            {memberAction === "approve" ? (
              <p className="mt-3 text-sm">
                Mwanachama atahesabiwa kwenye jumla ya wanachama na atapata ufikiaji wa dashibodi mara moja.
              </p>
            ) : (
              <div className="mt-4">
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-destructive">
                    Sababu ya kukataa (lazima) *
                  </span>
                  <textarea
                    value={memberReason}
                    onChange={(e) => setMemberReason(e.target.value)}
                    placeholder="Eleza sababu..."
                    rows={3}
                    className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-ring/20"
                  />
                </label>
              </div>
            )}
            <div className="mt-4 flex gap-3">
              <button
                onClick={() => { setSelectedMember(null); setMemberAction(null); setMemberReason(""); }}
                className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted"
              >
                Ghairi
              </button>
              <button
                onClick={submitMemberAction}
                disabled={
                  memberActionMutation.isPending ||
                  (memberAction === "reject" && !memberReason.trim())
                }
                className={`flex-1 rounded-xl py-2.5 text-sm font-semibold text-white disabled:opacity-60 ${
                  memberAction === "approve" ? "bg-success" : "bg-destructive"
                }`}
              >
                {memberActionMutation.isPending ? (
                  <Loader2 className="mx-auto h-4 w-4 animate-spin" />
                ) : memberAction === "approve" ? (
                  "Idhinisha"
                ) : (
                  "Kataa"
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* User approve/reject Modal */}
      {selectedUser && actionType && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={() => { setSelectedUser(null); setActionType(null); }}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-lg font-bold">
              {actionType === "approve" ? "Idhinisha Mtumiaji" : "Kataa Mtumiaji"}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {selectedUser.name} — {selectedUser.phone}
            </p>
            <div className="mt-4">
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-muted-foreground">Maoni (si lazima)</span>
                <textarea
                  value={remarks}
                  onChange={(e) => setRemarks(e.target.value)}
                  placeholder={actionType === "approve" ? "Maoni ya uidhinishaji..." : "Sababu ya kukataa..."}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-ring/20"
                  rows={3}
                />
              </label>
            </div>
            <div className="mt-4 flex gap-3">
              <button
                onClick={() => { setSelectedUser(null); setActionType(null); setRemarks(""); }}
                className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted"
              >
                Ghairi
              </button>
              <button
                onClick={handleAction}
                disabled={actionLoading}
                className={`flex-1 rounded-xl py-2.5 text-sm font-semibold text-white disabled:opacity-60 ${
                  actionType === "approve" ? "bg-success" : "bg-destructive"
                }`}
              >
                {actionLoading ? "Inashughulikiwa..." : actionType === "approve" ? "Idhinisha" : "Kataa"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
