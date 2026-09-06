import { createFileRoute } from "@tanstack/react-router";
import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { AppShell } from "@/components/AppShell";
import { useMembers, useCreateMember, useUpdateMember, useChairCreateLogin } from "@/hooks/use-members";
import { useChairResetPassword } from "@/hooks/use-user-management";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, blockAdminFromPage, requireAuth } from "@/lib/role-guards";
import { tokenStorage } from "@/lib/auth-storage";
import { tarehe } from "@/lib/format";
import { Field } from "@/components/Field";
import { useDebounce } from "@/hooks/use-debounce";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
  PaginationEllipsis,
} from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import { Search, UserPlus, Phone, X, Loader2, Pencil, KeyRound, Clock, Send } from "lucide-react";

export const Route = createFileRoute("/wanachama")({
  head: () => ({
    meta: [
      { title: "Wanachama — Kikundi" },
      { name: "description", content: "Sajili na simamia wanachama wa kikundi." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    blockAdminFromPage();
  },
  component: WanachamaPage,
});

function WanachamaPage() {
  const { user } = useAuth();
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const limit = 20;
  const debouncedQ = useDebounce(q, 300);
  const [open, setOpen] = useState(false);
  const [editMember, setEditMember] = useState<typeof members[number] | null>(null);
  const [resetMember, setResetMember] = useState<typeof members[number] | null>(null);
  const [lifecycleMember, setLifecycleMember] = useState<{ id: string; full_name: string; is_active: boolean } | null>(null);
  const isChair = hasRole(user, "chair");
  const resetPwd = useChairResetPassword();
  const createLogin = useChairCreateLogin();
  const [resetLoading, setResetLoading] = useState(false);
  const [resetMsg, setResetMsg] = useState<string | null>(null);
  const [resetTempPassword, setResetTempPassword] = useState<string | null>(null);
  const [resetFailed, setResetFailed] = useState(false);

  const { data, isLoading, error, refetch } = useMembers({
    q: debouncedQ || undefined,
    page,
    limit,
  });
  const members = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / limit);
  const updateMember = useUpdateMember();

  const handlePageChange = useCallback((p: number) => {
    setPage(p);
  }, []);

  const closeResetModal = () => {
    setResetMember(null);
    setResetMsg(null);
    setResetTempPassword(null);
    setResetFailed(false);
  };

  const handleResetPassword = async () => {
    if (!resetMember) return;
    setResetLoading(true);
    setResetMsg(null);
    setResetTempPassword(null);
    setResetFailed(false);
    try {
      // Members with a login account get a password reset; members without
      // one get a fresh login account (both return a one-time temp password).
      const res = resetMember.user_id
        ? await resetPwd.mutateAsync(resetMember.user_id)
        : await createLogin.mutateAsync(resetMember.id);
      setResetMsg(res.message || "Nenosiri limewekwa upya.");
      if (res.temp_password) setResetTempPassword(res.temp_password);
    } catch (e: unknown) {
      setResetFailed(true);
      setResetMsg(e instanceof Error ? e.message : "Imeshindikana kuweka upya nenosiri");
    } finally {
      setResetLoading(false);
    }
  };

  return (
    <AppShell
      title="Wanachama"
      subtitle={`${total} wameandikishwa`}
      action={
        <div className="flex gap-2">
          <button
            onClick={() => setOpen(true)}
            className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground"
          >
            <UserPlus className="h-4 w-4" /> Ongeza Mwanachama
          </button>
        </div>
      }
    >
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          value={q}
          onChange={(e) => { setQ(e.target.value); setPage(1); }}
          placeholder="Tafuta kwa jina, simu au namba…"
          className="w-full rounded-xl border border-input bg-card pl-9 pr-3 py-3 text-sm outline-none focus:border-primary"
        />
      </div>

      {isLoading && (
        <div className="mt-4 space-y-2.5">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="card-surface flex items-center gap-3 p-3.5">
              <Skeleton className="h-11 w-11 shrink-0 rounded-full" />
              <div className="min-w-0 flex-1 space-y-1.5">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-48" />
              </div>
            </div>
          ))}
        </div>
      )}

      {error && (
        <div className="card-surface mt-4 p-6 text-center">
          <p className="text-sm text-destructive mb-3">{error.message}</p>
          <button
            onClick={() => refetch()}
            className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground"
          >
            <Loader2 className="h-4 w-4" /> Jaribu tena
          </button>
        </div>
      )}

      {!isLoading && !error && (
        <>
          <div className="mt-4 space-y-2.5">
            {members.map((w) => (
              <div key={w.id} className="card-surface flex items-center gap-3 p-3.5">
                <div className="grid h-11 w-11 shrink-0 place-items-center rounded-full bg-primary/10 font-display font-bold text-primary">
                  {w.full_name.split(" ").map((x) => x[0]).slice(0, 2).join("")}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <p className="truncate font-semibold">{w.full_name}</p>
                    <span className={`chip text-[10px] ${w.is_active ? "bg-success/15 text-success" : "bg-destructive/10 text-destructive"}`}>
                      {w.is_active ? "Hai" : "Hahai"}
                    </span>
                  </div>
                  <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                    <span className="inline-flex items-center gap-1"><Phone className="h-3 w-3" />{w.phone}</span>
                    <span>· {w.member_no}</span>
                  </div>
                  <p className="mt-0.5 text-[11px] text-muted-foreground">Alijiunga {tarehe(w.joined_at)}</p>
                </div>
                <div className="flex items-center gap-1.5">
                  {isChair && w.user_id && (
                    <button
                      onClick={() => { setResetMember(w); setResetMsg(null); }}
                      title="Weka upya nenosiri"
                      className="rounded-lg p-1.5 text-amber-600 hover:bg-amber-50"
                    >
                      <KeyRound className="h-4 w-4" />
                    </button>
                  )}
                  {isChair && !w.user_id && (
                    <button
                      onClick={() => { setResetMember(w); setResetMsg(null); }}
                      title="Tengeneza akaunti ya kuingia"
                      className="rounded-lg p-1.5 text-sky-600 hover:bg-sky-50"
                    >
                      <UserPlus className="h-4 w-4" />
                    </button>
                  )}
                  {isChair && (
                    <button
                      onClick={() => setEditMember(w)}
                      title="Hariri mwanachama"
                      className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                  )}
                  {isChair && (
                    <button
                      onClick={() => {
                        if (w.is_active) {
                          setLifecycleMember({ id: w.id, full_name: w.full_name, is_active: w.is_active });
                        } else {
                          updateMember.mutate({ id: w.id, data: { is_active: true } });
                        }
                      }}
                      disabled={updateMember.isPending}
                      className={`text-xs font-medium disabled:opacity-50 rounded-lg px-2.5 py-1.5 ${
                        w.is_active
                          ? "text-destructive hover:bg-destructive/10"
                          : "text-success hover:bg-success/10"
                      }`}
                    >
                      {w.is_active ? "Zima" : "Amilisha"}
                    </button>
                  )}
                </div>
              </div>
            ))}
            {members.length === 0 && (
              <div className="card-surface p-8 text-center text-sm text-muted-foreground">
                Hakuna mwanachama aliyepatikana.
              </div>
            )}
          </div>

          {totalPages > 1 && (
            <div className="mt-4">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => handlePageChange(page - 1)}
                      className={page <= 1 ? "pointer-events-none opacity-50" : ""}
                    />
                  </PaginationItem>
                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
                    .map((p, i, arr) => (
                      <PaginationItem key={p}>
                        {i > 0 && arr[i - 1] !== p - 1 && <PaginationEllipsis />}
                        <PaginationLink
                          onClick={() => handlePageChange(p)}
                          isActive={p === page}
                        >
                          {p}
                        </PaginationLink>
                      </PaginationItem>
                    ))}
                  <PaginationItem>
                    <PaginationNext
                      onClick={() => handlePageChange(page + 1)}
                      className={page >= totalPages ? "pointer-events-none opacity-50" : ""}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </>
      )}

      {open && <FormDialog onClose={() => setOpen(false)} />}
      {editMember && (
        <EditMemberDialog
          member={editMember}
          onClose={() => setEditMember(null)}
        />
      )}

      {/* Deactivate confirmation (chair only) */}
      {lifecycleMember && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={() => setLifecycleMember(null)}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-lg font-semibold">Zima mwanachama?</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Una uhakika unataka kuzima <span className="font-semibold text-foreground">{lifecycleMember.full_name}</span>?
              Hataweza kushiriki shughuli za kikundi hadi aamilishwe tena.
            </p>
            <div className="mt-5 flex gap-2">
              <button
                type="button"
                onClick={() => setLifecycleMember(null)}
                className="flex-1 rounded-xl border border-border py-2.5 text-sm font-semibold"
              >
                Ghairi
              </button>
              <button
                type="button"
                disabled={updateMember.isPending}
                onClick={() => {
                  updateMember.mutate(
                    { id: lifecycleMember.id, data: { is_active: false } },
                    { onSettled: () => setLifecycleMember(null) },
                  );
                }}
                className="flex-1 rounded-xl bg-destructive py-2.5 text-sm font-semibold text-white disabled:opacity-50"
              >
                {updateMember.isPending ? "Inafanyika..." : "Ndio, zima"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reset Password Confirmation Modal (chair → POST /users/:id/reset-password) */}
      {resetMember && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={resetTempPassword ? undefined : closeResetModal}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="grid h-10 w-10 place-items-center rounded-full bg-amber-100">
                  <KeyRound className="h-5 w-5 text-amber-600" />
                </div>
                <div>
                  <h3 className="font-display text-lg font-semibold">{resetMember.user_id ? "Weka Upya Nenosiri" : "Tengeneza Akaunti ya Kuingia"}</h3>
                  <p className="text-xs text-muted-foreground">{resetMember.full_name}</p>
                </div>
              </div>
              <button onClick={closeResetModal} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-4 w-4" /></button>
            </div>
            {resetTempPassword ? (
              <div className="mb-4 space-y-3">
                {resetMsg && <p className="text-sm text-success">{resetMsg}</p>}
                <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2">
                  <p className="text-xs font-semibold text-amber-900">Nenosiri la muda (onyesha mara moja):</p>
                  <p className="mt-1 font-mono text-base tracking-wide text-amber-950 select-all">{resetTempPassword}</p>
                  <p className="mt-1 text-[11px] text-amber-800">Nakili sasa — haitaonekana tena baada ya kufunga.</p>
                </div>
              </div>
            ) : resetMsg ? (
              <div className={`mb-4 rounded-xl px-4 py-3 text-sm ${resetFailed ? "bg-destructive/10 text-destructive" : "bg-success/10 text-success"}`}>
                {resetMsg}
              </div>
            ) : (
              <p className="mb-4 text-sm text-muted-foreground">
                {resetMember.user_id ? (
                  <>Nenosiri la muda nasibu litatolewa kwa <strong>{resetMember.full_name}</strong>. Litaonyeshwa mara moja baada ya kuthibitisha — mpe mtumiaji moja kwa moja. Atakazwa kuweka nenosiri jipya atakapoingia.</>
                ) : (
                  <><strong>{resetMember.full_name}</strong> hana akaunti ya kuingia. Akaunti mpya itatengenezwa na nenosiri la muda litaonyeshwa mara moja — mpe mwanachama moja kwa moja. Atakazwa kuweka nenosiri jipya atakapoingia.</>
                )}
              </p>
            )}
            <div className="flex gap-3">
              <button
                onClick={closeResetModal}
                className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted"
              >
                {resetMsg || resetTempPassword ? "Funga" : "Ghairi"}
              </button>
              {!resetMsg && !resetTempPassword && (
                <button
                  onClick={handleResetPassword}
                  disabled={resetLoading}
                  className="flex-1 rounded-xl bg-amber-500 py-2.5 text-sm font-semibold text-white disabled:opacity-60 inline-flex items-center justify-center gap-2"
                >
                  {resetLoading && <Loader2 className="h-4 w-4 animate-spin" />}
                  {resetMember.user_id ? "Weka Upya Nenosiri" : "Tengeneza Akaunti"}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}

function FormDialog({ onClose }: { onClose: () => void }) {
  const createMember = useCreateMember();
  const qc = useQueryClient();
  const [f, setF] = useState({
    full_name: "",
    phone: "",
    address: "",
    gender: "",
    occupation: "",
    email: "",
    next_of_kin_name: "",
    next_of_kin_phone: "",
    joined_at: new Date().toISOString().slice(0, 10),
  });
  const [err, setErr] = useState<string | null>(null);
  const [submitted, setSubmitted] = useState(false);
  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [photoUploading, setPhotoUploading] = useState(false);
  const [photoUrl, setPhotoUrl] = useState<string | null>(null);

  const handlePhoto = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setPhotoFile(file);
    setPhotoUploading(true);
    setErr(null);
    try {
      const fd = new FormData();
      fd.append("file", file);
      const token = tokenStorage.get();
      const res = await fetch(
        `${import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1"}/upload/avatar`,
        { method: "POST", headers: { Authorization: `Bearer ${token}` }, body: fd }
      );
      if (!res.ok) throw new Error("Imeshindikana kupakia picha");
      const data = await res.json();
      setPhotoUrl(data.url);
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "Imeshindikana kupakia picha");
      setPhotoFile(null);
    } finally {
      setPhotoUploading(false);
    }
  };

  const handleSubmit = async () => {
    setErr(null);
    try {
      await createMember.mutateAsync({
        full_name: f.full_name,
        phone: f.phone,
        address: f.address || undefined,
        gender: (f.gender || undefined) as "MME" | "MKE" | undefined,
        occupation: f.occupation || undefined,
        email: f.email || undefined,
        next_of_kin_name: f.next_of_kin_name || undefined,
        next_of_kin_phone: f.next_of_kin_phone || undefined,
        photo_url: photoUrl || undefined,
        joined_at: f.joined_at,
      });
      qc.invalidateQueries({ queryKey: ["members"] });
      setSubmitted(true);
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "Imeshindikana kusajili");
    }
  };

  // Success view — clear "Pending approval" state, NOT fully active yet
  if (submitted) {
    return (
      <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={onClose}>
        <div className="w-full max-w-md rounded-t-3xl bg-card p-6 sm:rounded-2xl text-center" onClick={(e) => e.stopPropagation()}>
          <Clock className="mx-auto h-12 w-12 text-amber-500" />
          <h3 className="mt-3 font-display text-lg font-bold">Mwanachama Amesajiliwa</h3>
          <span className="chip bg-amber-100 text-amber-700 mt-2 inline-flex items-center gap-1">
            <Clock className="h-3 w-3" /> Pending approval
          </span>
          <p className="mt-3 text-sm text-muted-foreground">
            Mwanachama hajiunganishwa kikamilifu mpaka <strong>Katibu</strong> aidhinishе.
          </p>
          <button
            onClick={onClose}
            className="mt-5 w-full rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground"
          >
            Sawa
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center overflow-y-auto" onClick={onClose}>
      <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl my-6 max-h-[85dvh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-display text-lg font-semibold">Sajili Mwanachama Mpya</h3>
          <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-4 w-4" /></button>
        </div>
        {err && <p className="mb-3 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{err}</p>}
        <div className="space-y-3">
          <Field label="Jina kamili" value={f.full_name} onChange={(v) => setF({ ...f, full_name: v })} />
          <Field label="Namba ya simu" value={f.phone} onChange={(v) => setF({ ...f, phone: v })} type="tel" />
          <div>
            <label className="block text-xs text-muted-foreground mb-1">Jinsia</label>
            <select
              value={f.gender}
              onChange={(e) => setF({ ...f, gender: e.target.value })}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
            >
              <option value="">— Chagua —</option>
              <option value="MME">Mwanamume</option>
              <option value="MKE">Mwanamke</option>
            </select>
          </div>
          <Field label="Kazi / Taaluma (hiari)" value={f.occupation} onChange={(v) => setF({ ...f, occupation: v })} />
          <Field label="Barua pepe (hiari)" value={f.email} onChange={(v) => setF({ ...f, email: v })} type="email" />
          <Field label="Mlezi / Ndugu wa karibu (hiari)" value={f.next_of_kin_name} onChange={(v) => setF({ ...f, next_of_kin_name: v })} />
          <Field label="Simu ya mlezi (hiari)" value={f.next_of_kin_phone} onChange={(v) => setF({ ...f, next_of_kin_phone: v })} type="tel" />
          <Field label="Anwani (hiari)" value={f.address} onChange={(v) => setF({ ...f, address: v })} />
          <Field label="Tarehe ya kujiunga" value={f.joined_at} onChange={(v) => setF({ ...f, joined_at: v })} type="date" />
          <div>
            <label className="block text-xs text-muted-foreground mb-1">Picha ya mwanachama (hiari)</label>
            <input
              type="file"
              accept="image/*"
              onChange={handlePhoto}
              className="w-full text-xs text-muted-foreground file:mr-3 file:rounded-lg file:border-0 file:bg-primary/10 file:px-3 file:py-1.5 file:text-xs file:font-semibold file:text-primary"
            />
            {photoUploading && (
              <p className="mt-1 text-xs text-muted-foreground flex items-center gap-1">
                <Loader2 className="h-3 w-3 animate-spin" /> Inapakia picha...
              </p>
            )}
            {photoUrl && <p className="mt-1 text-xs text-success">Picha imepakiwa ✓</p>}
          </div>
        </div>
        <button
          disabled={!f.full_name || !f.phone || createMember.isPending || photoUploading}
          onClick={handleSubmit}
          className="mt-5 w-full rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-50 inline-flex items-center justify-center gap-2"
        >
          {createMember.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
          Tuma kwa Katibu kuidhinisha
        </button>
      </div>
    </div>
  );
}


function EditMemberDialog({ member, onClose }: { member: { id: string; full_name: string; phone: string; address?: string; is_active: boolean; member_no: string; user_id?: string }; onClose: () => void }) {
  const updateMember = useUpdateMember();
  const resetPwd = useChairResetPassword();
  const [f, setF] = useState({
    full_name: member.full_name,
    phone: member.phone,
    address: member.address ?? "",
    is_active: member.is_active,
  });
  const [err, setErr] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [tempPassword, setTempPassword] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const handleSubmit = async () => {
    setErr(null);
    setSuccess(null);
    try {
      await updateMember.mutateAsync({
        id: member.id,
        data: {
          full_name: f.full_name,
          phone: f.phone,
          address: f.address || undefined,
          is_active: f.is_active,
        },
      });
      setSuccess("Mwanachama amebadilishwa.");
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "Imeshindikana kubadilisha");
    }
  };

  const handleToggleStatus = async () => {
    setErr(null);
    setSuccess(null);
    setActionLoading("status");
    try {
      await updateMember.mutateAsync({
        id: member.id,
        data: { is_active: !f.is_active },
      });
      setF({ ...f, is_active: !f.is_active });
      setSuccess(f.is_active ? "Mwanachama amezimwa." : "Mwanachama ameilishwa.");
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "Imeshindikana kubadilisha hali");
    } finally {
      setActionLoading(null);
    }
  };

  const handleResetPassword = async () => {
    if (!member.user_id) return;
    setErr(null);
    setSuccess(null);
    setTempPassword(null);
    setActionLoading("resetPwd");
    try {
      const res = await resetPwd.mutateAsync(member.user_id);
      setSuccess(res.message || "Nenosiri limewekwa upya.");
      if (res.temp_password) setTempPassword(res.temp_password);
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "Imeshindikana kuweka upya nenosiri");
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={onClose}>
      <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h3 className="font-display text-lg font-semibold">Hariri Mwanachama</h3>
            <p className="text-xs text-muted-foreground">{member.member_no}</p>
          </div>
          <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-4 w-4" /></button>
        </div>
        {err && <p className="mb-3 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{err}</p>}
        {success && <p className="mb-3 rounded-lg bg-success/10 px-3 py-2 text-sm text-success">{success}</p>}
        {tempPassword && (
          <div className="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm">
            <p className="font-semibold text-amber-900">Nenosiri la muda (onyesha mara moja):</p>
            <p className="mt-1 font-mono text-base tracking-wide text-amber-950 select-all">{tempPassword}</p>
            <p className="mt-1 text-xs text-amber-800">Nakili sasa — haitaonekana tena baada ya kufunga.</p>
          </div>
        )}
        <div className="space-y-3">
          <Field label="Jina kamili" value={f.full_name} onChange={(v) => setF({ ...f, full_name: v })} />
          <Field label="Namba ya simu" value={f.phone} onChange={(v) => setF({ ...f, phone: v })} type="tel" />
          <Field label="Anwani" value={f.address} onChange={(v) => setF({ ...f, address: v })} />
          <div className="flex items-center justify-between rounded-xl border border-input px-3 py-2.5">
            <span className="text-sm">Hali: <span className={`font-semibold ${f.is_active ? "text-success" : "text-destructive"}`}>{f.is_active ? "Hai" : "Hahai"}</span></span>
            <button
              onClick={handleToggleStatus}
              disabled={actionLoading === "status"}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium disabled:opacity-50 ${
                f.is_active ? "bg-destructive/10 text-destructive" : "bg-success/10 text-success"
              }`}
            >
              {actionLoading === "status" ? "Inashughulikiwa..." : f.is_active ? "Zima" : "Amilisha"}
            </button>
          </div>
          {member.user_id && (
            <div className="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2.5">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <KeyRound className="h-4 w-4 text-amber-600" />
                  <span className="text-sm font-medium text-amber-800">Nenosiri</span>
                </div>
                <button
                  onClick={handleResetPassword}
                  disabled={actionLoading === "resetPwd"}
                  className="rounded-lg bg-amber-100 px-3 py-1.5 text-xs font-medium text-amber-700 hover:bg-amber-200 disabled:opacity-50"
                >
                  {actionLoading === "resetPwd" ? "Inashughulikiwa..." : "Weka Upya"}
                </button>
              </div>
              <p className="mt-1 text-[11px] text-amber-600">Nenosiri la muda nasibu litatolewa na kuonyeshwa mara moja; mtumiaji atakazwa kuweka jipya.</p>
            </div>
          )}
        </div>
        <div className="mt-5 flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted"
          >
            Ghairi
          </button>
          <button
            disabled={!f.full_name || !f.phone || updateMember.isPending}
            onClick={handleSubmit}
            className="flex-1 rounded-xl bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50 inline-flex items-center justify-center gap-2"
          >
            {updateMember.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            Hifadhi
          </button>
        </div>
      </div>
    </div>
  );
}
