import { createFileRoute, Link } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { AppShell } from "@/components/AppShell";
import { requireAuth, requireRole } from "@/lib/role-guards";
import { tzs } from "@/lib/format";
import {
  useTrialBalance,
  useLedgerBalance,
  useLedgerStatement,
  useTransactionDetail,
} from "@/hooks/use-ledger";
import {
  Scale,
  Loader2,
  AlertTriangle,
  ArrowDownLeft,
  ArrowUpRight,
  X,
  ReceiptText,
} from "lucide-react";

export const Route = createFileRoute("/kitabu")({
  head: () => ({
    meta: [
      { title: "Kitabu cha Fedha — Money Seeking" },
      { name: "description", content: "Kitabu cha double-entry (kusoma tu): trial balance na mwenendo wa pesa." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    // Viewing only — chair, secretary, treasurer. Admin bypasses.
    requireRole("chair", "secretary", "treasurer");
  },
  component: KitabuPage,
});

type Tab = "muhtasari" | "mwenendo";

function KitabuPage() {
  const [tab, setTab] = useState<Tab>("muhtasari");

  const tabs: { id: Tab; label: string }[] = [
    { id: "muhtasari", label: "Muhtasari" },
    { id: "mwenendo", label: "Pesa Ilivyoingia/Kutoka" },
  ];

  return (
    <AppShell
      title="Kitabu cha Fedha"
      subtitle="Mtazamo wa vitabu (kusoma tu) — miamala inaingia kiotomatiki"
    >
      <div className="mb-4 flex flex-wrap gap-2">
        {tabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`rounded-full px-4 py-1.5 text-sm font-medium ${
                tab === t.id
                  ? "bg-primary text-primary-foreground"
                  : "border border-border text-muted-foreground"
              }`}
            >
              {t.label}
            </button>
          ))}
      </div>

      {tab === "muhtasari" && <TrialBalanceCard />}
      {tab === "mwenendo" && <MovementCard />}
    </AppShell>
  );
}

function Card({ title, sub, icon: Icon, children }: { title: string; sub?: string; icon: any; children: React.ReactNode }) {
  return (
    <section className="card-surface overflow-hidden">
      <header className="flex items-center gap-2.5 border-b border-border px-4 py-3">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
          <Icon className="h-4 w-4" />
        </span>
        <div>
          <h3 className="font-display text-sm font-semibold">{title}</h3>
          {sub && <p className="text-[11px] text-muted-foreground">{sub}</p>}
        </div>
      </header>
      <div className="p-4">{children}</div>
    </section>
  );
}

function ErrorNote({ message }: { message: string }) {
  return (
    <p className="flex items-center gap-2 rounded-xl bg-destructive/10 px-3 py-2.5 text-sm text-destructive">
      <AlertTriangle className="h-4 w-4 shrink-0" /> {message}
    </p>
  );
}

// ---- Muhtasari (trial balance): everyone with access ----
function TrialBalanceCard() {
  const { data, isLoading, error } = useTrialBalance();
  if (isLoading) {
    return (
      <Card title="Trial Balance" icon={Scale}>
        <div className="flex justify-center py-6"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      </Card>
    );
  }
  if (error || !data) {
    return (
      <Card title="Trial Balance" icon={Scale}>
        <ErrorNote message="Vitabu havisawazishwi au bado hakuna miamala. Mweka hazina aanze kwa kufungua akaunti." />
      </Card>
    );
  }
  const lines = data.Lines ?? [];
  return (
    <Card
      title="Trial Balance"
      sub={data.Balanced ? "Debit = Credit · vitabu vimesawazishwa" : "ONYO: debit na credit hazisawazishwi"}
      icon={Scale}
    >
      <div className="mb-3">
        <span className={`rounded-full px-3 py-1 text-xs font-semibold ${data.Balanced ? "bg-emerald-500/15 text-emerald-700" : "bg-destructive/10 text-destructive"}`}>
          {data.Balanced ? "Imesawazishwa" : "Haijasawazishwa"}
        </span>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-xs text-muted-foreground">
              <th className="py-2 pr-2">Akaunti</th>
              <th className="py-2 pr-2">Aina</th>
              <th className="py-2 pr-2 text-right">Debit</th>
              <th className="py-2 text-right">Credit</th>
            </tr>
          </thead>
          <tbody>
            {lines.map((l) => (
              <tr key={l.AccountName} className="border-b border-border/50">
                <td className="py-2 pr-2 font-medium">{l.AccountName}</td>
                <td className="py-2 pr-2 text-muted-foreground">{l.Type}</td>
                <td className="py-2 pr-2 text-right">{tzs(l.DebitMinor)}</td>
                <td className="py-2 text-right">{tzs(l.CreditMinor)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="font-bold">
              <td className="py-2 pr-2" colSpan={2}>Jumla</td>
              <td className="py-2 pr-2 text-right">{tzs(data.TotalDebitMinor)}</td>
              <td className="py-2 text-right">{tzs(data.TotalCreditMinor)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </Card>
  );
}

// ---- Mwenendo wa pesa (cash in/out log): everyone with access ----
// hazina_taslimu ni asset: debit = pesa iliyoingia, credit = pesa iliyotoka.
function MovementCard() {
  const toISO = useMemo(() => new Date().toISOString(), []);
  const fromISO = useMemo(
    () => new Date(Date.now() - 90 * 86400000).toISOString(),
    []
  );
  const balance = useLedgerBalance("hazina_taslimu");
  const stmt = useLedgerStatement("hazina_taslimu", fromISO, toISO);

  const lines = useMemo(() => {
    const ls = [...(stmt.data?.statement ?? [])];
    ls.sort(
      (a, b) => +new Date(a.OccurredAt) - +new Date(b.OccurredAt)
    );
    return ls;
  }, [stmt.data]);

  const signed = (direction: string, minor: number) =>
    direction === "debit" ? minor : -minor;
  const inSum = lines
    .filter((l) => l.Direction === "debit")
    .reduce((s, l) => s + l.AmountMinor, 0);
  const outSum = lines
    .filter((l) => l.Direction !== "debit")
    .reduce((s, l) => s + l.AmountMinor, 0);
  const net = inSum - outSum;
  const current = balance.data?.amount_minor ?? null;
  const opening = current != null ? current - net : null;

  if (balance.isLoading || stmt.isLoading) {
    return (
      <Card title="Pesa Ilivyoingia/Kutoka" icon={ArrowDownLeft}>
        <div className="flex justify-center py-6"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      </Card>
    );
  }
  if ((balance.error && stmt.error) || lines.length === 0) {
    return (
      <Card title="Pesa Ilivyoingia/Kutoka" sub="hazina_taslimu · siku 90 zilizopita" icon={ArrowDownLeft}>
        <p className="text-sm text-muted-foreground">Hakuna mwenendo wa pesa katika kipindi hiki. Miamala inaingia kiotomatiki michango inapothibitishwa.</p>
      </Card>
    );
  }

  let running = opening ?? 0;
  const rows = lines.map((l) => {
    running += signed(l.Direction, l.AmountMinor);
    return { ...l, running };
  });

  const [selected, setSelected] = useState<string | null>(null);
  const topOut = useMemo(
    () =>
      lines
        .filter((l) => l.Direction !== "debit")
        .slice()
        .sort((a, b) => b.AmountMinor - a.AmountMinor)
        .slice(0, 5),
    [lines]
  );

  return (
    <Card title="Pesa Ilivyoingia/Kutoka" sub="hazina_taslimu · siku 90 zilizopita · bofya mstari kuona maelezo" icon={ArrowDownLeft}>
      <div className="mb-3 grid grid-cols-3 gap-2 text-center">
        <div className="rounded-xl bg-emerald-500/10 px-2 py-2">
          <p className="text-[11px] text-muted-foreground">Ingizo</p>
          <p className="text-sm font-bold text-emerald-600">+{tzs(inSum)}</p>
        </div>
        <div className="rounded-xl bg-destructive/10 px-2 py-2">
          <p className="text-[11px] text-muted-foreground">Matoleo</p>
          <p className="text-sm font-bold text-destructive">−{tzs(outSum)}</p>
        </div>
        <div className="rounded-xl bg-primary/5 px-2 py-2">
          <p className="text-[11px] text-muted-foreground">Salio sasa</p>
          <p className="text-sm font-bold">{current != null ? tzs(current) : "—"}</p>
        </div>
      </div>
      {topOut.length > 0 && (
        <div className="mb-3 rounded-xl border border-border p-3">
          <p className="mb-2 text-xs font-semibold text-muted-foreground">Matoleo makubwa zaidi (pesa ilikwenda wapi)</p>
          <div className="space-y-1.5">
            {topOut.map((l) => (
              <button
                key={`top-${l.TransactionID}-${l.OccurredAt}`}
                onClick={() => setSelected(l.TransactionID)}
                className="flex w-full items-center justify-between gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-muted"
              >
                <span className="min-w-0">
                  <span className="block truncate font-medium">
                    {l.Memo || (l.EventType === "TransactionReversed" ? `Ubatilishaji (${l.ReversesMemo || "—"})` : "—")}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {new Date(l.OccurredAt).toLocaleDateString()} · {l.ActorName || ""}
                  </span>
                </span>
                <span className="shrink-0 font-semibold text-destructive">−{tzs(l.AmountMinor)}</span>
              </button>
            ))}
          </div>
        </div>
      )}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-xs text-muted-foreground">
              <th className="py-2 pr-2">Tarehe</th>
              <th className="py-2 pr-2">Maelezo</th>
              <th className="py-2 pr-2">Nani</th>
              <th className="py-2 pr-2 text-right">Ingizo</th>
              <th className="py-2 pr-2 text-right">Toleo</th>
              <th className="py-2 text-right">Salio</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr
                key={`${r.TransactionID}-${r.OccurredAt}`}
                onClick={() => setSelected(r.TransactionID)}
                className="cursor-pointer border-b border-border/50 hover:bg-muted/50"
              >
                <td className="py-2 pr-2 text-muted-foreground">{new Date(r.OccurredAt).toLocaleDateString()}</td>
                <td className="py-2 pr-2">
                  {r.Memo || (r.EventType === "TransactionReversed" ? "Ubatilishaji" : "—")}
                  {r.EventType === "TransactionReversed" && (
                    <span className="ml-1.5 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-800">Ubatilishaji</span>
                  )}
                </td>
                <td className="py-2 pr-2 text-muted-foreground">{r.ActorName || "—"}</td>
                <td className="py-2 pr-2 text-right text-emerald-600">
                  {r.Direction === "debit" ? `+${tzs(r.AmountMinor)}` : "—"}
                </td>
                <td className="py-2 pr-2 text-right text-destructive">
                  {r.Direction !== "debit" ? `−${tzs(r.AmountMinor)}` : "—"}
                </td>
                <td className="py-2 text-right font-semibold">{tzs(r.running)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="mt-3 flex items-start gap-2 text-xs text-muted-foreground">
        <ArrowUpRight className="mt-0.5 h-3.5 w-3.5 shrink-0" />
        Kila mchango ukithibitishwa, kila marejesho na kila mkopo unaotolewa huingia hapa kiotomatiki — hakuna kuingiza manually.
      </p>
      {selected && <TransactionModal id={selected} onClose={() => setSelected(null)} />}
    </Card>
  );
}

// ---- Maelezo ya muamala (click-to-view): nani, lini, wapi ilikwenda ----
function TransactionModal({ id, onClose }: { id: string; onClose: () => void }) {
  const { data, isLoading, error } = useTransactionDetail(id);
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 p-4 sm:items-center" onClick={onClose}>
      <div
        className="card-surface max-h-[85vh] w-full max-w-lg overflow-y-auto p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-2 font-display text-base font-semibold">
            <ReceiptText className="h-4 w-4 text-primary" /> Maelezo ya Muamala
          </h3>
          <button onClick={onClose} aria-label="Funga" className="rounded-lg p-1 hover:bg-muted">
            <X className="h-4 w-4" />
          </button>
        </div>
        {isLoading && <div className="flex justify-center py-6"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>}
        {error && <ErrorNote message="Imeshindikana kupakia maelezo." />}
        {data && (
          <div className="space-y-3 text-sm">
            <div className="rounded-xl bg-muted/40 p-3">
              <p className="font-semibold">{data.memo || (data.event_type === "TransactionReversed" ? "Ubatilishaji wa muamala" : "—")}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {new Date(data.occurred_at).toLocaleString()} · Aliyeingiza: <strong>{data.actor_name || "—"}</strong>
              </p>
              {data.event_type === "TransactionReversed" && (
                <p className="mt-1 text-xs">
                  <span className="rounded-full bg-amber-100 px-1.5 py-0.5 font-semibold text-amber-800">Ubatilishaji</span>{" "}
                  {data.reverses_memo && <span className="text-muted-foreground">wa: {data.reverses_memo}</span>}
                  {data.reason && <span className="block text-muted-foreground">Sababu: {data.reason}</span>}
                </p>
              )}
            </div>
            <div>
              <p className="mb-1 text-xs font-semibold text-muted-foreground">Miguu ya muamala (double-entry)</p>
              {(data.entries ?? []).map((e, i) => (
                <div key={i} className="flex items-center justify-between border-t border-border/50 py-1.5 first:border-t-0">
                  <span>{e.account_name} <span className="text-xs text-muted-foreground">({e.direction})</span></span>
                  <span className="font-semibold">{tzs(e.amount_minor)}</span>
                </div>
              ))}
            </div>
            {data.loan && (
              <div className="rounded-xl border border-border p-3">
                <p className="mb-1 text-xs font-semibold text-muted-foreground">Mkopo husika</p>
                <p className="font-semibold">{data.loan.member_name} <span className="text-xs text-muted-foreground">({data.loan.member_no})</span></p>
                <p className="text-xs text-muted-foreground">Kiasi: {data.loan.amount} · Hali: {data.loan.status}</p>
                <p className="text-xs text-muted-foreground">Aliyeupitisha: <strong>{data.loan.reviewed_by || "—"}</strong>{data.loan.reviewed_at ? ` · ${new Date(data.loan.reviewed_at).toLocaleString()}` : ""}</p>
                <p className="text-xs text-muted-foreground">Uliotolewa: <strong>{data.loan.disbursed_at ? new Date(data.loan.disbursed_at).toLocaleString() : "—"}</strong>{data.loan.disbursed_by ? ` na ${data.loan.disbursed_by}` : ""}</p>
                <Link
                  to="/ukaguzi-mkopo/$loanId"
                  params={{ loanId: data.loan.id }}
                  className="mt-2 inline-block rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground"
                >
                  Fungua mkopo
                </Link>
              </div>
            )}
            <p className="font-mono text-[11px] text-muted-foreground">ID: {data.transaction_id}</p>
          </div>
        )}
      </div>
    </div>
  );
}
