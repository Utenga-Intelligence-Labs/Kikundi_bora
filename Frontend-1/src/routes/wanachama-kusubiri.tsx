import { createFileRoute, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { usePendingUsers, useApproveUser, useRejectUser } from "@/hooks/use-user-management";
import { roleMap } from "@/api/types";
import type { User } from "@/api/types";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, blockAdminFromPage } from "@/lib/role-guards";
import { tokenStorage } from "@/lib/auth-storage";
import { Clock, CheckCircle2, XCircle, User as UserIcon, Phone, Calendar } from "lucide-react";

export const Route = createFileRoute("/wanachama-kusubiri")({
  head: () => ({
    meta: [
      { title: "Wanaosubiri — Kikundi" },
      { name: "description", content: "Orodha ya watumiaji wanaosubiri kuidhinishwa." },
    ],
  }),
  beforeLoad: () => {
    if (typeof window !== "undefined" && !tokenStorage.exists()) {
      throw redirect({ to: "/ingia" });
    }
    blockAdminFromPage();
  },
  component: PendingUsersPage,
});

function PendingUsersPage() {
  const { user } = useAuth();
  const { data, isLoading, error, refetch } = usePendingUsers({ limit: 50 });
  const approveMutation = useApproveUser();
  const rejectMutation = useRejectUser();

  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [actionType, setActionType] = useState<"approve" | "reject" | null>(null);
  const [remarks, setRemarks] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  if (!hasRole(user, "secretary", "chair")) {
    return (
      <AppShell title="Wanaosubiri">
        <div className="flex items-center justify-center py-20">
          <p className="text-muted-foreground">Huna ruhusa ya kuona ukurasa huu.</p>
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

  const pendingUsers = data?.data ?? [];

  return (
    <AppShell title="Wanaosubiri Kuidhinishwa" subtitle="Watumiaji walioundwa na Mwenyekiti">
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

      {/* Approve/Reject Modal */}
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
