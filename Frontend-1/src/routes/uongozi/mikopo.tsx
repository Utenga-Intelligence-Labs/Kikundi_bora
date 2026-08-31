import { createFileRoute, Link } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { tokenStorage } from "@/lib/auth-storage";
import { requireAuth } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { ShieldCheck, Check, X, Clock, Banknote, UserPlus, Users, ChevronRight } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useIsCommitteeMember } from "@/hooks/use-loan-committee";
import { useMembers } from "@/hooks/use-members";
import { useState } from "react";
import { useAppModal } from "@/components/AppModal";

export const Route = createFileRoute("/uongozi/mikopo")({
  beforeLoad: () => { requireAuth(); },
  component: MikopoPage,
});

interface Loan {
  id: string; member_id: string; amount: string; approved_amount?: string;
  balance_remaining?: string; purpose?: string; due_date: string; status: string;
  rejection_reason?: string; applied_at: string;
  hazina_approved_at?: string; katibu_approved_at?: string; bodi_approved_at?: string; mwenyekiti_approved_at?: string;
  member?: { id: string; member_no: string; full_name: string; phone: string };
}

function MikopoPage() {
  const { user, isLeadership } = useAuth();
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [showBodiPopup, setShowBodiPopup] = useState(false);
  const [selectedMember, setSelectedMember] = useState("");

  const isHazina = user?.role === "treasurer";
  const isKatibu = user?.role === "secretary";
  const isMwenyekiti = user?.role === "chair";
  const { data: committeeCheck } = useIsCommitteeMember();
  const isBodi = user?.role === "member" && committeeCheck?.is_committee_member;

  if (!user || !isLeadership) {
    return <AppShell title="Idhinisha Mikopo"><div className="card-surface p-12 text-center"><p className="text-muted-foreground">Huna ruhusa</p></div></AppShell>;
  }

  const { data: loans, isLoading } = useQuery({
    queryKey: ["uongozi", "mikopo", "pending"],
    queryFn: async () => {
      const token = tokenStorage.get();
      const base = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";
      const res = await fetch(`${base}/uongozi/mikopo/pending`, { headers: { Authorization: `Bearer ${token}` } });
      if (!res.ok) throw new Error("Imeshindikana");
      const data = await res.json();
      return data.data as Loan[];
    },
  });

  const { data: membersData } = useMembers({ limit: 50 });

  const approveMutation = useMutation({
    mutationFn: async ({ loanId, amount }: { loanId: string; amount: string | number }) => {
      const token = tokenStorage.get();
      const base = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";
      const res = await fetch(`${base}/uongozi/mikopo/${loanId}/approve`, {
        method: "POST", headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ approved_amount: amount }),
      });
      if (!res.ok) throw new Error((await res.json()).message || "Imeshindikana");
      return res.json();
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["uongozi", "mikopo"] }); },
    onError: (err: Error) => showModal({ title: "Hitilafu", message: err.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const appointMutation = useMutation({
    mutationFn: async (userId: string) => {
      const token = tokenStorage.get();
      const base = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";
      const res = await fetch(`${base}/loan-committee/members`, {
        method: "POST", headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ user_id: userId }),
      });
      if (!res.ok) throw new Error((await res.json()).message || "Imeshindikana");
      return res.json();
    },
    onSuccess: () => { setShowBodiPopup(false); setSelectedMember(""); showModal({ title: "Imefanikiwa", message: "Mwanachama ameongezwa kwenye Bodi ya Mikopo!", variant: "success", primaryLabel: "Sawa" }); },
    onError: (err: Error) => showModal({ title: "Hitilafu", message: err.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const handleApprove = (loan: Loan) => {
    showModal({
      title: "Thibitisha",
      message: `Idhinisha mkopo wa TZS ${Number(loan.amount).toLocaleString()}?`,
      variant: "warning",
      primaryLabel: "Thibitisha",
      secondaryLabel: "Ghairi",
      onPrimary: async () => {
        setActionLoading(loan.id);
        try {
          await approveMutation.mutateAsync({ loanId: loan.id, amount: loan.amount });
          showModal({ title: "Imefanikiwa", message: "Umeidhinisha!", variant: "success", primaryLabel: "Sawa" });
        } catch { /* error handled by mutation */ }
        finally { setActionLoading(null); }
      },
    });
  };

  const getApprovalStep = (loan: Loan) => {
    if (loan.mwenyekiti_approved_at) return { step: 4, label: "Mwenyekiti ameidhinisha", done: true };
    if (loan.bodi_approved_at) return { step: 4, label: "Anasubiri Mwenyekiti", done: false };
    if (loan.katibu_approved_at) return { step: 3, label: "Anasubiri Bodi", done: false };
    if (loan.hazina_approved_at) return { step: 2, label: "Anasubiri Katibu", done: false };
    return { step: 1, label: "Anasubiri Hazina", done: false };
  };

  const canApprove = (loan: Loan) => {
    if (isHazina && !loan.hazina_approved_at) return true;
    if (isKatibu && loan.hazina_approved_at && !loan.katibu_approved_at) return true;
    if (isBodi && loan.hazina_approved_at && loan.katibu_approved_at && !loan.bodi_approved_at) return true;
    if (isMwenyekiti && loan.hazina_approved_at && loan.katibu_approved_at && loan.bodi_approved_at && !loan.mwenyekiti_approved_at) return true;
    return false;
  };

  const pendingLoans = loans ?? [];
  const availableMembers = (membersData?.data ?? []).filter((m: any) => m.is_active);

  return (
    <AppShell title="Idhinisha Mikopo" subtitle="Mfuatano: Hazina → Katibu → Bodi → Mwenyekiti" action={
      isKatibu && (
        <button onClick={() => setShowBodiPopup(true)} className="inline-flex items-center gap-1.5 rounded-xl bg-accent px-3.5 py-2 text-sm font-semibold">
          <UserPlus className="h-4 w-4" /> Unda Bodi Ya Mikopo
        </button>
      )
    }>
      {isLoading ? (
        <div className="card-surface animate-pulse p-6"><div className="h-4 w-1/3 rounded bg-muted" /></div>
      ) : pendingLoans.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <ShieldCheck className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-muted-foreground">Hakuna mikopo inayosubiri idhini</p>
        </div>
      ) : (
        <div className="space-y-4">
          {pendingLoans.map((loan) => {
            const step = getApprovalStep(loan);
            return (
              <div key={loan.id} className="card-surface p-5">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="chip bg-amber-100 text-amber-700 text-[10px] font-semibold px-2 py-0.5 rounded">{loan.member?.member_no}</span>
                      <span className="chip bg-blue-100 text-blue-700 text-[10px] font-semibold px-2 py-0.5 rounded inline-flex items-center gap-1"><Clock className="h-3 w-3" /> {step.label}</span>
                    </div>
                    <h3 className="font-display text-lg font-semibold">{loan.member?.full_name}</h3>
                    <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
                      <div><p className="text-muted-foreground">Kiasi</p><p className="font-semibold">TZS {Number(loan.amount).toLocaleString()}</p></div>
                      <div><p className="text-muted-foreground">Tarehe ya Kulipa</p><p className="font-semibold">{new Date(loan.due_date).toLocaleDateString("sw-TZ")}</p></div>
                    </div>

                    {/* Approval progress bar */}
                    <div className="mt-3 flex items-center gap-1 text-xs flex-wrap">
                      <span className={`chip px-2 py-0.5 rounded-full ${loan.hazina_approved_at ? 'bg-success text-success-foreground' : 'bg-muted text-muted-foreground'}`}>Hazina {loan.hazina_approved_at ? '✓' : ''}</span>
                      <ChevronRight className="h-3 w-3 text-muted-foreground" />
                      <span className={`chip px-2 py-0.5 rounded-full ${loan.katibu_approved_at ? 'bg-success text-success-foreground' : 'bg-muted text-muted-foreground'}`}>Katibu {loan.katibu_approved_at ? '✓' : ''}</span>
                      <ChevronRight className="h-3 w-3 text-muted-foreground" />
                      <span className={`chip px-2 py-0.5 rounded-full ${loan.bodi_approved_at ? 'bg-success text-success-foreground' : 'bg-muted text-muted-foreground'}`}>Bodi {loan.bodi_approved_at ? '✓' : ''}</span>
                      <ChevronRight className="h-3 w-3 text-muted-foreground" />
                      <span className={`chip px-2 py-0.5 rounded-full ${loan.mwenyekiti_approved_at ? 'bg-success text-success-foreground' : 'bg-muted text-muted-foreground'}`}>Mwenyekiti {loan.mwenyekiti_approved_at ? '✓' : ''}</span>
                    </div>
                  </div>
                  <div className="flex flex-col gap-2">
                    {canApprove(loan) && (
                      <button onClick={() => handleApprove(loan)} disabled={actionLoading === loan.id}
                        className="inline-flex items-center gap-1.5 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700 disabled:opacity-50">
                        <Check className="h-4 w-4" /> {actionLoading === loan.id ? "..." : "Idhinisha"}
                      </button>
                    )}
                    <Link to={`/ukaguzi-mkopo/${loan.id}`} className="inline-flex items-center gap-1.5 rounded-lg border px-4 py-2 text-sm font-medium hover:bg-muted">Angalia</Link>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Unda Bodi Ya Mikopo Popup */}
      {showBodiPopup && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center sm:p-4" onClick={() => setShowBodiPopup(false)}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold flex items-center gap-2"><Users className="h-5 w-5" /> Unda Bodi Ya Mikopo</h3>
              <button onClick={() => setShowBodiPopup(false)} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-4 w-4" /></button>
            </div>
            <p className="text-sm text-muted-foreground mb-4">Viongozi wote (Mwenyekiti, Katibu, Hazina) tayari ni wajumbe. Chagua mwanachama wa kawaida kama mwakilishi wa wanachama wengine.</p>
            <label className="block">
              <span className="mb-1 block text-xs font-medium text-muted-foreground">Chagua Mwanachama</span>
              <select value={selectedMember} onChange={(e) => setSelectedMember(e.target.value)}
                className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm">
                <option value="">— Chagua mwanachama —</option>
                {availableMembers.map((m: any) => (
                  <option key={m.id} value={m.user_id || m.id}>{m.full_name} ({m.member_no})</option>
                ))}
              </select>
            </label>
            <button onClick={() => selectedMember && appointMutation.mutate(selectedMember)} disabled={!selectedMember || appointMutation.isPending}
              className="mt-4 w-full rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-50">
              {appointMutation.isPending ? "Inaongeza..." : "Ongeza Kwenye Bodi"}
            </button>
          </div>
        </div>
      )}
    </AppShell>
  );
}
