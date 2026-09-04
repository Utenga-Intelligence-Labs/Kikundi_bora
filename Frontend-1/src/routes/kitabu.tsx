import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { AppShell } from "@/components/AppShell";
import { Field } from "@/components/Field";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth, requireRole, hasRole } from "@/lib/role-guards";
import { tzs } from "@/lib/format";
import {
  useTrialBalance,
  useLedgerBalance,
  useLedgerStatement,
  useOpenAccount,
  useRecordTransaction,
  useReverseTransaction,
} from "@/hooks/use-ledger";
import {
  STANDARD_ACCOUNTS,
  type LedgerDirection,
  type LedgerAccountType,
  type LedgerEntryInput,
} from "@/api/ledger";
import {
  BookOpen,
  Scale,
  Search,
  PlusCircle,
  Landmark,
  Undo2,
  Loader2,
  AlertTriangle,
  CheckCircle2,
} from "lucide-react";

export const Route = createFileRoute("/kitabu")({
  head: () => ({
    meta: [
      { title: "Kitabu cha Fedha — Money Seeking" },
      { name: "description", content: "Kitabu cha double-entry: trial balance, akaunti, na uingizaji wa miamala (mweka hazina)." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    // Chair + secretary view-only; treasurer posts. Admin bypasses (full access).
    requireRole("chair", "secretary", "treasurer");
  },
  component: KitabuPage,
});

type Tab = "muhtasari" | "akaunti" | "muamala" | "akaunti-mpya" | "batilisha";

function KitabuPage() {
  const { user } = useAuth();
  const canWrite = hasRole(user, "treasurer", "admin");
  const [tab, setTab] = useState<Tab>("muhtasari");

  const tabs: { id: Tab; label: string; write?: boolean }[] = [
    { id: "muhtasari", label: "Muhtasari" },
    { id: "akaunti", label: "Akaunti" },
    { id: "muamala", label: "Ingiza Muamala", write: true },
    { id: "akaunti-mpya", label: "Fungua Akaunti", write: true },
    { id: "batilisha", label: "Batilisha", write: true },
  ];

  return (
    <AppShell
      title="Kitabu cha Fedha"
      subtitle={
        canWrite
          ? "Double-entry: ingiza miamala na fuatilia vitabu"
          : "Mtazamo wa vitabu (kusoma tu)"
      }
    >
      <div className="mb-4 flex flex-wrap gap-2">
        {tabs
          .filter((t) => !t.write || canWrite)
          .map((t) => (
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
      {tab === "akaunti" && <AccountCard />}
      {tab === "muamala" && canWrite && <RecordForm />}
      {tab === "akaunti-mpya" && canWrite && <OpenAccountForm />}
      {tab === "batilisha" && canWrite && <ReverseForm />}
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

function SuccessNote({ message }: { message: string }) {
  return (
    <p className="flex items-center gap-2 rounded-xl bg-emerald-500/10 px-3 py-2.5 text-sm text-emerald-700 dark:text-emerald-400">
      <CheckCircle2 className="h-4 w-4 shrink-0" /> {message}
    </p>
  );
}

const submitBtn =
  "rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50";

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

// ---- Akaunti (balance + statement): everyone with access ----
function AccountCard() {
  const [account, setAccount] = useState("hazina_taslimu");
  const [query, setQuery] = useState<string | null>(null);
  const balance = useLedgerBalance(query);
  const statement = useLedgerStatement(query);

  return (
    <Card title="Akaunti" sub="Salio na statement kwa akaunti moja" icon={Search}>
      <div className="mb-3 flex flex-col gap-2 sm:flex-row">
        <div className="flex-1">
          <Field label="Jina la akaunti" value={account} onChange={setAccount} placeholder="hazina_taslimu" />
        </div>
        <button onClick={() => setQuery(account.trim())} className={`${submitBtn} sm:self-end`}>
          Tafuta
        </button>
      </div>
      <datalist id="kitabu-accounts">
        {STANDARD_ACCOUNTS.map((a) => (
          <option key={a.name} value={a.name} />
        ))}
      </datalist>

      {balance.data && (
        <p className="mb-3 rounded-xl bg-primary/5 px-3 py-2.5 text-sm">
          Salio la <strong>{balance.data.account}</strong>:{" "}
          <strong>{tzs(balance.data.amount_minor)}</strong>
        </p>
      )}
      {(balance.error || statement.error) && query && (
        <ErrorNote message="Akaunti haijapatikana. Fungua akaunti kwanza." />
      )}
      {(statement.data?.statement ?? []).length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-muted-foreground">
                <th className="py-2 pr-2">Tarehe</th>
                <th className="py-2 pr-2">Maelezo</th>
                <th className="py-2 pr-2">Mwelekeo</th>
                <th className="py-2 text-right">Kiasi</th>
              </tr>
            </thead>
            <tbody>
              {(statement.data?.statement ?? []).map((s) => (
                <tr key={`${s.TransactionID}-${s.OccurredAt}`} className="border-b border-border/50">
                  <td className="py-2 pr-2 text-muted-foreground">{new Date(s.OccurredAt).toLocaleDateString()}</td>
                  <td className="py-2 pr-2">{s.Memo}</td>
                  <td className="py-2 pr-2">{s.Direction === "debit" ? "Debit" : "Credit"}</td>
                  <td className="py-2 text-right">{tzs(s.AmountMinor)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

// ---- Ingiza Muamala (treasurer/admin only) ----
function RecordForm() {
  const [memo, setMemo] = useState("");
  const [entries, setEntries] = useState<(LedgerEntryInput & { amount: string })[]>([
    { account_name: "hazina_taslimu", direction: "debit", amount_minor: 0, amount: "" },
    { account_name: "akiba_ya_mwanachama:", direction: "credit", amount_minor: 0, amount: "" },
  ]);
  const m = useRecordTransaction();
  const [done, setDone] = useState<string | null>(null);

  const totals = useMemo(() => {
    let d = 0, c = 0;
    for (const e of entries) {
      const v = Math.round(Number(e.amount) || 0);
      if (e.direction === "debit") d += v; else c += v;
    }
    return { d, c, balanced: d > 0 && d === c };
  }, [entries]);

  const setEntry = (i: number, patch: Partial<LedgerEntryInput & { amount: string }>) =>
    setEntries((prev) => prev.map((e, j) => (j === i ? { ...e, ...patch } : e)));

  const submit = () => {
    setDone(null);
    m.reset();
    m.mutate(
      {
        memo: memo.trim(),
        entries: entries.map((e) => ({
          account_name: e.account_name.trim(),
          direction: e.direction as LedgerDirection,
          amount_minor: Math.round(Number(e.amount) || 0),
        })),
      },
      { onSuccess: (r) => { setDone(`Muamala umerekodiwa (${r.transaction_id.slice(0, 8)}…)`); setMemo(""); } }
    );
  };

  const valid = memo.trim().length > 0 && entries.length >= 1 && totals.balanced &&
    entries.every((e) => e.account_name.trim().length >= 2 && (Number(e.amount) || 0) > 0);

  return (
    <Card title="Ingiza Muamala" sub="Debit lazima iwe sawa na credit — vinginevyo seva itakataa" icon={PlusCircle}>
      <div className="mb-3"><Field label="Maelezo (memo)" value={memo} onChange={setMemo} placeholder="Mfano: Michango Septemba — KKK-0001" /></div>
      {entries.map((e, i) => (
        <div key={i} className="mb-2 grid grid-cols-12 gap-2">
          <div className="col-span-6">
            <Field label={i === 0 ? "Akaunti" : ""} value={e.account_name} onChange={(v) => setEntry(i, { account_name: v })} placeholder="hazina_taslimu" />
          </div>
          <div className="col-span-3">
            <label className="block">
              {i === 0 && <span className="mb-1 block text-xs font-medium text-muted-foreground">Mwelekeo</span>}
              <select
                value={e.direction}
                onChange={(ev) => setEntry(i, { direction: ev.target.value as LedgerDirection })}
                className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none"
              >
                <option value="debit">Debit</option>
                <option value="credit">Credit</option>
              </select>
            </label>
          </div>
          <div className="col-span-2">
            <Field label={i === 0 ? "TZS" : ""} type="number" value={e.amount} onChange={(v) => setEntry(i, { amount: v })} placeholder="0" />
          </div>
          <div className="col-span-1 flex items-end">
            <button
              onClick={() => setEntries((prev) => prev.filter((_, j) => j !== i))}
              disabled={entries.length <= 1}
              className="rounded-xl border border-border px-2 py-2.5 text-sm text-destructive disabled:opacity-30"
              title="Ondoa"
            >×</button>
          </div>
        </div>
      ))}
      <div className="mb-3 flex gap-2">
        <button
          onClick={() => setEntries((p) => [...p, { account_name: "", direction: "debit", amount_minor: 0, amount: "" }])}
          className="rounded-xl border border-border px-3 py-1.5 text-sm"
        >+ Mstari</button>
        <span className={`self-center text-sm ${totals.balanced ? "text-emerald-600" : "text-muted-foreground"}`}>
          Debit {tzs(totals.d)} · Credit {tzs(totals.c)}{totals.balanced ? " · Inasawazishwa" : ""}
        </span>
      </div>
      {m.isError && <div className="mb-2"><ErrorNote message={(m.error as Error)?.message ?? "Imeshindikana kurekodi"} /></div>}
      {done && <div className="mb-2"><SuccessNote message={done} /></div>}
      <button onClick={submit} disabled={!valid || m.isPending} className={submitBtn}>
        {m.isPending ? "Inarekodi…" : "Rekodi Muamala"}
      </button>
    </Card>
  );
}

// ---- Fungua Akaunti (treasurer/admin only) ----
function OpenAccountForm() {
  const [name, setName] = useState("");
  const [type, setType] = useState<LedgerAccountType>("asset");
  const [owner, setOwner] = useState("");
  const m = useOpenAccount();
  const [done, setDone] = useState<string | null>(null);

  return (
    <Card title="Fungua Akaunti" sub="Akaunti za wanachama: akiba_ya_mwanachama:KKK-0001" icon={Landmark}>
      <div className="mb-3 grid gap-3 sm:grid-cols-2">
        <Field label="Jina la akaunti" value={name} onChange={setName} placeholder="hazina_taslimu" />
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-muted-foreground">Aina</span>
          <select value={type} onChange={(e) => setType(e.target.value as LedgerAccountType)} className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none">
            <option value="asset">asset (mali)</option>
            <option value="liability">liability (deni/akiba)</option>
            <option value="income">income (mapato)</option>
            <option value="expense">expense (matumizi)</option>
            <option value="equity">equity (mtaji)</option>
          </select>
        </label>
      </div>
      <div className="mb-3"><Field label="Mmiliki (hiari — namba ya mwanachama)" value={owner} onChange={setOwner} placeholder="KKK-0001" /></div>
      <div className="mb-3 flex flex-wrap gap-1.5">
        {STANDARD_ACCOUNTS.map((a) => (
          <button key={a.name} onClick={() => { setName(a.name); setType(a.type); }} className="rounded-full border border-border px-2.5 py-1 text-xs text-muted-foreground">
            {a.name}
          </button>
        ))}
      </div>
      {m.isError && <div className="mb-2"><ErrorNote message={(m.error as Error)?.message ?? "Imeshindikana"} /></div>}
      {done && <div className="mb-2"><SuccessNote message={done} /></div>}
      <button
        onClick={() => { setDone(null); m.reset(); m.mutate({ name: name.trim(), type, owner_member_ref: owner.trim() || undefined }, { onSuccess: (r) => { setDone(`Akaunti "${r.account_name}" imefunguliwa`); setName(""); setOwner(""); } }); }}
        disabled={name.trim().length < 2 || m.isPending}
        className={submitBtn}
      >{m.isPending ? "Inafungua…" : "Fungua Akaunti"}</button>
    </Card>
  );
}

// ---- Batilisha (treasurer/admin only) ----
function ReverseForm() {
  const [id, setId] = useState("");
  const [reason, setReason] = useState("");
  const m = useReverseTransaction();
  const [done, setDone] = useState<string | null>(null);

  return (
    <Card title="Batilisha Muamala" sub="Historia haifutwi — ubatilishaji ni event mpya" icon={Undo2}>
      <div className="mb-3"><Field label="Transaction ID" value={id} onChange={setId} placeholder="uuid ya muamala" /></div>
      <div className="mb-3"><Field label="Sababu" value={reason} onChange={setReason} placeholder="Mfano: kiasi kilikosewa" /></div>
      {m.isError && <div className="mb-2"><ErrorNote message={(m.error as Error)?.message ?? "Imeshindikana"} /></div>}
      {done && <div className="mb-2"><SuccessNote message={done} /></div>}
      <button
        onClick={() => { setDone(null); m.reset(); m.mutate({ id: id.trim(), reason: reason.trim() || undefined }, { onSuccess: () => { setDone("Muamala umebatilishwa"); setId(""); setReason(""); } }); }}
        disabled={id.trim().length < 8 || m.isPending}
        className={submitBtn}
      >{m.isPending ? "Inabatilisha…" : "Batilisha"}</button>
      <p className="mt-3 flex items-start gap-2 text-xs text-muted-foreground">
        <BookOpen className="mt-0.5 h-3.5 w-3.5 shrink-0" />
        ID ya muamala unaipata kwenye statement ya akaunti husika (safu ya kwanza).
      </p>
    </Card>
  );
}
