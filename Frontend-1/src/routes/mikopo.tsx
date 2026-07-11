import { createFileRoute, redirect, Link } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useLoans, useApplyLoan, useApproveLoan, useRejectLoan, useDisburseLoan } from "@/hooks/use-loans";
import { useMembers } from "@/hooks/use-members";
import { Field } from "@/components/Field";
import { tzs, tarehe } from "@/lib/format";
import { X, Check, Ban, Send, Wallet, Loader2 } from "lucide-react";
import { useAuth } from "@/lib/auth-provider";
import { tokenStorage } from "@/lib/auth-storage";
import { blockAdminFromPage } from "@/lib/role-guards";
import { roleMap, type LoanStatus } from "@/api/types";

export const Route = createFileRoute("/mikopo")({
  head: () => ({
    meta: [
      { title: "Mikopo — Money Seeking" },
      { name: "description", content: "Toa mikopo na fuatilia mizania ya wanachama." },
    ],
  }),
  beforeLoad: () => {
    if (typeof window !== "undefined" && !tokenStorage.exists()) throw redirect({ to: "/ingia" });
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

  const jukumu = roleMap[user.role] ?? "Mwanachama";
  const isMember = jukumu === "Mwanachama";
  const isChair = jukumu === "Mwenyekiti";
  const isTreasurer = jukumu === "Mweka Hazina";

  const [tab, setTab] = useState<Tab>(isMember ? "OUTSTANDING" : isChair ? "PENDING" : isTreasurer ? "APPROVED" : "OUTSTANDING");
  const [openRequest, setOpenRequest] = useState(false);
  const [rejecting, setRejecting] = useState<string | null>(null);
  const [disbursing, setDisbursing] = useState<string | null>(null);

  const { data: loansData, isLoading } = useLoans({ limit: 200 });
  const { data: membersData } = useMembers({ limit: 200 });
  const approveLoan = useApproveLoan();
  const rejectLoan = useRejectLoan();
  const disburseLoan = useDisburseLoan();

  const loans = loansData?.data ?? [];
  const members = membersData?.data ?? [];
  const me = members.find((m) => m.user_id === user.id) ||
    members.find((m) => m.full_name.toLowerCase() === user.name.toLowerCase());

  const visible = isMember && me
    ? loans.filter((l) => l.member_id === me.id)
    : loans;
  const list = visible.filter((l) => l.status === tab);
  const tabs: Tab[] = ["PENDING", "UNDER_REVIEW", "APPROVED", "OUTSTANDING", "CLOSED", "REJECTED"];

  const jumlaWazi = visible
    .filter((l) => l.status === "OUTSTANDING")
    .reduce((s, l) => s + (l.balance_remaining ?? 0), 0);

  return (
    <AppShell
      title={isMember ? "Mikopo Yangu" : "Mikopo"}
      subtitle={isMember ? "Omba na fuatilia mikopo yako" : "Simamia mikopo ya kikundi"}
      action={
        isMember ? (
          <button
            onClick={() => setOpenRequest(true)}
            disabled={!me}
            className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground disabled:opacity-50"
          >
            <Send className="h-4 w-4" /> Omba Mkopo
          </button>
        ) : null
      }
    >
      {isMember && !me && (
        <div className="card-surface mb-5 p-4">
          <p className="text-sm">Hujasajiliwa kama mwanachama bado.</p>
          <Link to="/wasifu" className="mt-2 inline-block text-sm font-semibold text-primary">Jisajili sasa →</Link>
        </div>
      )}

      <div className="hero-surface px-5 py-5">
        <p className="text-xs text-primary-foreground/70">{isMember ? "Salio la mikopo yangu" : "Mizania ya mikopo wazi"}</p>
        <p className="mt-1 font-display text-3xl font-extrabold">{tzs(jumlaWazi)}</p>
        <p className="mt-1 text-xs text-primary-foreground/70">{visible.filter((l) => l.status === "OUTSTANDING").length} mikopo wazi</p>
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
          const bal = l.balance_remaining ?? (l.status === "APPROVED" ? (l.approved_amount ?? l.amount) : 0);
          const pct = l.approved_amount ? Math.min(100, (((l.approved_amount ?? 0) - bal) / (l.approved_amount ?? 1)) * 100) : 0;
          return (
            <div key={l.id} className="card-surface p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate font-semibold">{l.member?.full_name ?? `Mwanachama #${l.member_id}`}</p>
                  <p className="text-xs text-muted-foreground">{l.purpose || "—"}</p>
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

              {isChair && l.status === "PENDING" && (
                <div className="mt-3 flex gap-2">
                  <button
                    onClick={() => approveLoan.mutate({ id: l.id, data: { approved_amount: l.amount } })}
                    disabled={approveLoan.isPending}
                    className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-success px-3 py-2 text-xs font-semibold text-white disabled:opacity-50"
                  >
                    <Check className="h-3.5 w-3.5" /> Idhinisha
                  </button>
                  <button
                    onClick={() => setRejecting(l.id)}
                    className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-destructive/10 px-3 py-2 text-xs font-semibold text-destructive"
                  >
                    <Ban className="h-3.5 w-3.5" /> Kataa
                  </button>
                </div>
              )}
              {isTreasurer && l.status === "APPROVED" && (
                <button
                  onClick={() => setDisbursing(l.id)}
                  disabled={disburseLoan.isPending}
                  className="mt-3 inline-flex w-full items-center justify-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50"
                >
                  <Wallet className="h-3.5 w-3.5" /> Toa Fedha
                </button>
              )}
            </div>
          );
        })}
        {list.length === 0 && !isLoading && <div className="card-surface p-8 text-center text-sm text-muted-foreground">Hakuna mikopo katika hali hii.</div>}
      </div>

      {openRequest && me && <RequestForm memberId={me.id} onClose={() => setOpenRequest(false)} />}
      {rejecting != null && <RejectDialog onClose={() => setRejecting(null)} onConfirm={(reason) => { rejectLoan.mutate({ id: rejecting, data: { reason } }); setRejecting(null); }} />}
      {disbursing != null && (() => {
        const l = loans.find((x) => x.id === disbursing);
        return (
          <DisburseDialog
            loan={l}
            onClose={() => setDisbursing(null)}
            onConfirm={() => { disburseLoan.mutate(disbursing); setDisbursing(null); }}
            isPending={disburseLoan.isPending}
            error={disburseLoan.error ? (disburseLoan.error.message) : null}
          />
        );
      })()}
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

function DisburseDialog({ loan, onClose, onConfirm, isPending, error }: { loan?: { id: string; amount: number; approved_amount?: number; member?: { full_name?: string } }; onClose: () => void; onConfirm: () => void; isPending: boolean; error: string | null }) {
  return (
    <Modal title="Toa Fedha za Mkopo" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-sm">
          Unatoa mkopo wa <span className="font-semibold">{tzs(loan?.approved_amount ?? loan?.amount ?? 0)}</span>
          {loan?.member?.full_name ? <> kwa <span className="font-semibold">{loan.member.full_name}</span></> : null}
        </p>
        <p className="text-xs text-muted-foreground">Hakikisha umemkabidhi mwanachama fedha kabla ya kubofya.</p>
      </div>
      {error && <p className="mt-2 text-sm text-destructive">{error}</p>}
      <div className="mt-4 flex gap-3">
        <button onClick={onClose} className="flex-1 rounded-xl border border-border py-2.5 text-sm font-semibold">Ghairi</button>
        <button onClick={onConfirm} disabled={isPending} className="flex-1 rounded-xl bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50 inline-flex items-center justify-center gap-2">
          {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wallet className="h-4 w-4" />}
          Thibitisha Kutolea
        </button>
      </div>
    </Modal>
  );
}

function RejectDialog({ onClose, onConfirm }: { onClose: () => void; onConfirm: (sababu: string) => void }) {
  const [s, setS] = useState("");
  return (
    <Modal title="Kataa Ombi" onClose={onClose}>
      <Field label="Sababu" value={s} onChange={setS} />
      <button
        onClick={() => onConfirm(s || "Haijatolewa sababu")}
        className="mt-5 w-full rounded-xl bg-destructive py-3 text-sm font-semibold text-white"
      >
        Thibitisha Kukataa
      </button>
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
