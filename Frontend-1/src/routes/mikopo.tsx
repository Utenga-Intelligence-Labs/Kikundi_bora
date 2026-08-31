import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useLoans, useApplyLoan } from "@/hooks/use-loans";
import { Field } from "@/components/Field";
import { tzs, tarehe } from "@/lib/format";
import { X, Send, Loader2 } from "lucide-react";
import { useAuth } from "@/lib/auth-provider";
import { blockAdminFromPage, requireAuth } from "@/lib/role-guards";
import { type LoanStatus } from "@/api/types";

export const Route = createFileRoute("/mikopo")({
  head: () => ({
    meta: [
      { title: "Mikopo — Money Seeking" },
      { name: "description", content: "Toa mikopo na fuatilia mizania ya wanachama." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    blockAdminFromPage();
  },
  component: MikopoPage,
});

type Tab = "PENDING" | "UNDER_REVIEW" | "APPROVED" | "OUTSTANDING" | "CLOSED" | "REJECTED";

const tabLabels: Record<Tab, string> = {
  PENDING: "Ombi",
  UNDER_REVIEW: "Inapitiwa",
  APPROVED: "Imeidhinishwa",
  OUTSTANDING: "Wazi",
  CLOSED: "Imefungwa",
  REJECTED: "Imekataliwa",
};

function MikopoPage() {
  const { user } = useAuth();
  if (!user) return null;

  const [tab, setTab] = useState<Tab>("OUTSTANDING");
  const [openRequest, setOpenRequest] = useState(false);

  const { data: loansData, isLoading } = useLoans({ limit: 200 });
  const loans = loansData?.data ?? [];
  // Personal view — EVERY user (including leadership, dual plane) sees only
  // their own loans here. Group-wide management lives on /uongozi/mikopo.
  const myMemberId = user.member_id || null;
  const visible = myMemberId ? loans.filter((l) => l.member_id === myMemberId) : [];
  const list = visible.filter((l) => l.status === tab);
  const tabs: Tab[] = ["PENDING", "UNDER_REVIEW", "APPROVED", "OUTSTANDING", "CLOSED", "REJECTED"];

  const jumlaWazi = visible
    .filter((l) => l.status === "OUTSTANDING")
    .reduce((s, l) => s + Number(l.balance_remaining ?? 0), 0);

  return (
    <AppShell
      title="Mikopo Yangu"
      subtitle="Omba na fuatilia mikopo yako"
      action={
        myMemberId ? (
          <button
            onClick={() => setOpenRequest(true)}
            className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground"
          >
            <Send className="h-4 w-4" /> Omba Mkopo
          </button>
        ) : null
      }
    >
      {!myMemberId && (
        <div className="card-surface mb-5 p-4">
          <p className="text-sm">Hujasajiliwa kama mwanachama bado.</p>
          <Link to="/wasifu" className="mt-2 inline-block text-sm font-semibold text-primary">Jisajili sasa →</Link>
        </div>
      )}

      <div className="hero-surface px-5 py-5">
        <p className="text-xs text-primary-foreground/70">Salio la mikopo yangu</p>
        <p className="mt-1 font-display text-3xl font-extrabold">{tzs(jumlaWazi)}</p>
        <p className="mt-1 text-xs text-primary-foreground/70">{visible.filter((l) => l.status === "OUTSTANDING").length} mikopo wazi</p>
        {myMemberId && (
          <button
            onClick={() => setOpenRequest(true)}
            className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-primary shadow-lg transition-transform hover:scale-[1.01] sm:w-auto"
          >
            <Send className="h-4 w-4" /> Omba Mkopo Mpya
          </button>
        )}
      </div>

      {isLoading && (
        <div className="mt-8 flex justify-center"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      )}

      <div className="mt-5 -mx-1 flex gap-1 overflow-x-auto pb-1">
        {tabs.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`shrink-0 rounded-lg px-4 py-1.5 text-xs font-semibold ${tab === t ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}
          >
            {tabLabels[t]} ({visible.filter((l) => l.status === t).length})
          </button>
        ))}
      </div>

      <div className="mt-3 space-y-2.5">
        {list.map((l) => {
          const bal = Number(l.balance_remaining ?? (l.status === "APPROVED" ? (l.approved_amount ?? l.amount) : 0));
          const pct = l.approved_amount ? Math.min(100, ((Number(l.approved_amount) - bal) / Number(l.approved_amount)) * 100) : 0;
          return (
            <div key={l.id} className="card-surface p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate font-semibold">{l.purpose || "Mkopo wa kikundi"}</p>
                  <p className="text-xs text-muted-foreground">Kilitumwa {tarehe(l.applied_at || l.due_date)}</p>
                </div>
                <span className={`chip text-[10px] ${statusClass(l.status)}`}>{tabLabels[l.status as Tab] ?? l.status}</span>
              </div>
              <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
                <div><p className="text-muted-foreground">Kiasi</p><p className="font-semibold">{tzs(l.amount)}</p></div>
                <div>
                  <p className="text-muted-foreground">{l.status === "APPROVED" ? "Idhinishwa" : "Salio"}</p>
                  <p className={`font-semibold ${l.status !== "APPROVED" ? "text-warning" : ""}`}>{tzs(bal)}</p>
                </div>
                <div><p className="text-muted-foreground">Mwisho</p><p className="font-semibold">{tarehe(l.due_date)}</p></div>
              </div>
              {l.status === "OUTSTANDING" && (
                <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-muted">
                  <div className="h-full bg-success transition-all" style={{ width: `${pct}%` }} />
                </div>
              )}
              {l.rejection_reason && <p className="mt-2 text-xs text-destructive">Sababu: {l.rejection_reason}</p>}
              {l.status === "PENDING" && (
                <p className="mt-2 text-xs text-muted-foreground">
                  Ombi lako linapitiwa: Hazina → Katibu → Bodi → Mwenyekiti (uonekano wa mfuatano: /uongozi/mikopo).
                </p>
              )}
            </div>
          );
        })}
        {list.length === 0 && !isLoading && <div className="card-surface p-8 text-center text-sm text-muted-foreground">Hakuna mikopo katika hali hii.</div>}
      </div>

      {openRequest && myMemberId && <RequestForm memberId={myMemberId} onClose={() => setOpenRequest(false)} />}
    </AppShell>
  );
}

function statusClass(s: LoanStatus) {
  switch (s) {
    case "PENDING": return "bg-muted text-foreground";
    case "UNDER_REVIEW": return "bg-primary/15 text-primary";
    case "APPROVED": return "bg-primary/15 text-primary";
    case "OUTSTANDING": return "bg-warning/25 text-foreground";
    case "CLOSED": return "bg-success/15 text-success";
    case "REJECTED": return "bg-destructive/10 text-destructive";
    default: return "bg-muted text-foreground";
  }
}

function RequestForm({ memberId, onClose }: { memberId: string; onClose: () => void }) {
  const applyLoan = useApplyLoan();
  const due = new Date(); due.setMonth(due.getMonth() + 6);
  const [f, setF] = useState({ kiasi: "200000", tareheMwisho: due.toISOString().slice(0, 10), maelezo: "" });

  const handleSubmit = async () => {
    try {
      await applyLoan.mutateAsync({
        member_id: memberId,
        amount: Number(f.kiasi),
        due_date: f.tareheMwisho,
        purpose: f.maelezo || undefined,
      });
      onClose();
    } catch { /* handled by RQ */ }
  };

  return (
    <Modal title="Omba Mkopo" onClose={onClose}>
      <Field label="Kiasi unachoomba (TZS)" value={f.kiasi} onChange={(v) => setF({ ...f, kiasi: v })} type="number" />
      <div className="mt-3"><Field label="Tarehe ya kurudisha" value={f.tareheMwisho} onChange={(v) => setF({ ...f, tareheMwisho: v })} type="date" /></div>
      <div className="mt-3"><Field label="Madhumuni" value={f.maelezo} onChange={(v) => setF({ ...f, maelezo: v })} /></div>
      <button
        disabled={applyLoan.isPending}
        onClick={handleSubmit}
        className="mt-5 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-50"
      >
        {applyLoan.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} Tuma Ombi
      </button>
      <p className="mt-2 text-center text-xs text-muted-foreground">Mwenyekiti ataidhinisha kisha Mweka Hazina atakutolea fedha.</p>
    </Modal>
  );
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={onClose}>
      <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-display text-lg font-semibold">{title}</h3>
          <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-4 w-4" /></button>
        </div>
        {children}
      </div>
    </div>
  );
}
