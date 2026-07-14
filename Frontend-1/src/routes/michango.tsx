import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useContributions, useCreateContribution } from "@/hooks/use-contributions";
import { useMembers } from "@/hooks/use-members";
import { Field } from "@/components/Field";
import { tzs, mwezi } from "@/lib/format";
import { useAuth } from "@/lib/auth-provider";
import { blockAdminFromPage, hasRole, requireAuth } from "@/lib/role-guards";
import { Skeleton } from "@/components/ui/skeleton";
import { Plus, X, CheckCircle2, AlertCircle, Loader2 } from "lucide-react";
import { z } from "zod";

export const Route = createFileRoute("/michango")({
  head: () => ({
    meta: [
      { title: "Michango — Money Seeking" },
      { name: "description", content: "Ingiza na fuatilia michango ya kila mwezi." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    blockAdminFromPage();
  },
  component: MichangoPage,
});

function generateMonthOptions(n: number) {
  const now = new Date();
  const months: string[] = [];
  for (let i = 0; i < n; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    months.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`);
  }
  return months;
}

function MichangoPage() {
  const { user } = useAuth();
  // Backend requires treasurer position for create; UI matches
  const canRecord = hasRole(user, "treasurer");
  const [open, setOpen] = useState(false);
  const [quickPayMember, setQuickPayMember] = useState<{ id: string; full_name: string } | null>(null);
  const monthOptions = generateMonthOptions(24);
  const [mk, setMk] = useState(monthOptions[0]);

  const { data: contribsData, isLoading: contribsLoading, error: contribsError, refetch } = useContributions({ month: mk, limit: 200 });
  const { data: membersData, isLoading: membersLoading } = useMembers({ limit: 500 });
  const createContribution = useCreateContribution();

  const contributions = contribsData?.data ?? [];
  const members = membersData?.data ?? [];
  const activeMembers = members.filter((m) => m.is_active);

  const memberMap = new Map(members.map((m) => [m.id, m] as const));

  const jumla = contributions.reduce((s, c) => s + c.amount, 0);
  const walipaCount = contributions.length;

  const walipaIds = new Set(contributions.map((c) => c.member_id));
  const wadaiwa = activeMembers.filter((m) => !walipaIds.has(m.id));

  const isLoading = contribsLoading || membersLoading;

  return (
    <AppShell
      title="Michango"
      subtitle="Kumbuka michango ya kila mwezi"
      action={
        canRecord ? (
          <button onClick={() => setOpen(true)} className="inline-flex items-center gap-1.5 rounded-xl bg-accent px-3.5 py-2 text-sm font-semibold text-accent-foreground">
            <Plus className="h-4 w-4" /> Ingiza
          </button>
        ) : undefined
      }
    >
      <div className="card-surface p-5">
        <div className="flex items-end justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs text-muted-foreground">Jumla ya mwezi</p>
            <p className="mt-1 font-display text-3xl font-extrabold">{tzs(jumla)}</p>
            <p className="mt-1 text-xs text-muted-foreground">{mwezi(mk + "-01")}</p>
          </div>
          <select value={mk} onChange={(e) => setMk(e.target.value)} className="rounded-xl border border-input bg-card px-3 py-2 text-sm">
            {monthOptions.map((m) => <option key={m} value={m}>{mwezi(m + "-01")}</option>)}
          </select>
        </div>
        <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
          <div className="rounded-xl bg-success/10 p-3">
            <p className="text-xs text-success">Wamelipa</p>
            <p className="font-display text-xl font-bold text-success">{walipaCount}</p>
          </div>
          <div className="rounded-xl bg-warning/20 p-3">
            <p className="text-xs">Hawajalipa</p>
            <p className="font-display text-xl font-bold">{wadaiwa.length}</p>
          </div>
        </div>
      </div>

      {isLoading && (
        <div className="mt-4 space-y-2.5">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="card-surface flex items-center gap-3 p-3.5">
              <Skeleton className="h-10 w-10 shrink-0 rounded-full" />
              <div className="flex-1 space-y-1.5">
                <Skeleton className="h-4 w-36" />
                <Skeleton className="h-3 w-24" />
              </div>
              <Skeleton className="h-4 w-20" />
            </div>
          ))}
        </div>
      )}

      {contribsError && (
        <div className="card-surface mt-4 p-6 text-center">
          <p className="text-sm text-destructive mb-3">{contribsError.message}</p>
          <button
            onClick={() => refetch()}
            className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground"
          >
            <Loader2 className="h-4 w-4" /> Jaribu tena
          </button>
        </div>
      )}

      {!isLoading && !contribsError && (
        <>
          <h2 className="mt-6 mb-2 font-display text-sm font-semibold">Waliolipa</h2>
          <div className="card-surface divide-y divide-border">
            {contributions.map((c) => {
              const m = memberMap.get(c.member_id);
              return (
                <div key={c.id} className="flex items-center justify-between px-4 py-3">
                  <div className="flex items-center gap-2.5 min-w-0">
                    <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{m?.full_name ?? c.member?.full_name ?? `#${c.member_id}`}</p>
                      <p className="text-xs text-muted-foreground">{c.paid_at}</p>
                    </div>
                  </div>
                  <p className="shrink-0 text-sm font-semibold">{tzs(c.amount)}</p>
                </div>
              );
            })}
            {contributions.length === 0 && <p className="px-4 py-6 text-center text-sm text-muted-foreground">Hakuna michango mwezi huu bado.</p>}
          </div>

          <h2 className="mt-6 mb-2 font-display text-sm font-semibold">Hawajalipa</h2>
          <div className="card-surface divide-y divide-border">
            {wadaiwa.map((w) => (
              <div key={w.id} className="flex items-center justify-between px-4 py-3">
                <div className="flex items-center gap-2.5 min-w-0">
                  <AlertCircle className="h-4 w-4 shrink-0 text-warning" />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{w.full_name}</p>
                    <p className="text-xs text-muted-foreground">{w.phone}</p>
                  </div>
                </div>
                {canRecord && (
                  <button
                    onClick={() => setQuickPayMember(w)}
                    disabled={createContribution.isPending}
                    className="text-xs font-semibold text-primary disabled:opacity-50"
                  >
                    + Lipa
                  </button>
                )}
              </div>
            ))}
            {wadaiwa.length === 0 && <p className="px-4 py-6 text-center text-sm text-muted-foreground">Wote wamelipa mwezi huu 🎉</p>}
          </div>
        </>
      )}

      {canRecord && open && <Form onClose={() => setOpen(false)} defaultMonth={mk} members={activeMembers} />}

      {canRecord && quickPayMember && (
        <QuickPayDialog
          member={quickPayMember}
          defaultMonth={mk}
          onClose={() => setQuickPayMember(null)}
          onPaid={(data) => {
            createContribution.mutate(data);
            setQuickPayMember(null);
          }}
          isPending={createContribution.isPending}
        />
      )}
    </AppShell>
  );
}

function QuickPayDialog({
  member,
  defaultMonth,
  onClose,
  onPaid,
  isPending,
}: {
  member: { id: string; full_name: string };
  defaultMonth: string;
  onClose: () => void;
  onPaid: (data: { member_id: string; amount: number; month: string; paid_at: string; payment_method: "CASH" | "BANK" | "MOBILE_MONEY" }) => void;
  isPending: boolean;
}) {
  const [kiasi, setKiasi] = useState("50000");
  const [tarehe, setTarehe] = useState(new Date().toISOString().slice(0, 10));
  const [err, setErr] = useState<string | null>(null);

  const num = Number(kiasi);
  const valid = !isNaN(num) && isFinite(num) && num > 0;

  const submit = () => {
    if (!valid) { setErr("Kiasi lazima kiwe namba halali zaidi ya 0"); return; }
    onPaid({ member_id: member.id, amount: num, month: defaultMonth, paid_at: tarehe, payment_method: "CASH" });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={onClose}>
      <div className="w-full max-w-sm rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
        <h3 className="font-display text-lg font-semibold">Lipa kwa {member.full_name}</h3>
        <div className="mt-3 space-y-3">
          <Field label="Kiasi (TZS)" value={kiasi} onChange={setKiasi} type="number" />
          <Field label="Tarehe" value={tarehe} onChange={setTarehe} type="date" />
        </div>
        {err && <p className="mt-2 text-sm text-destructive">{err}</p>}
        <div className="mt-4 flex gap-3">
          <button onClick={onClose} className="flex-1 rounded-xl border border-border py-2.5 text-sm font-semibold">Ghairi</button>
          <button onClick={submit} disabled={isPending || !valid} className="flex-1 rounded-xl bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50 inline-flex items-center justify-center gap-2">
            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Lipa
          </button>
        </div>
      </div>
    </div>
  );
}

const formSchema = z.object({
  member_id: z.string().min(1, "Chagua mwanachama"),
  kiasi: z
    .string()
    .min(1, "Weka kiasi")
    .refine((v) => {
      const n = Number(v);
      return !isNaN(n) && isFinite(n) && n > 0;
    }, "Kiasi lazima kiwe namba halali zaidi ya 0"),
  mwezi: z.string().min(1, "Chagua mwezi"),
  tarehe: z.string().min(1, "Chagua tarehe"),
  maelezo: z.string().optional(),
});

type FormFields = z.infer<typeof formSchema>;

function Form({ onClose, defaultMonth, members }: { onClose: () => void; defaultMonth: string; members: { id: string; full_name: string }[] }) {
  const createContribution = useCreateContribution();
  const [f, setF] = useState<FormFields>({
    member_id: members[0]?.id || "",
    kiasi: "50000",
    mwezi: defaultMonth,
    tarehe: new Date().toISOString().slice(0, 10),
    maelezo: "",
  });
  const [errors, setErrors] = useState<Partial<Record<keyof FormFields, string>>>({});
  const [submitErr, setSubmitErr] = useState<string | null>(null);

  const handleSubmit = async () => {
    const parsed = formSchema.safeParse(f);
    if (!parsed.success) {
      const fieldErrors: Partial<Record<keyof FormFields, string>> = {};
      parsed.error.errors.forEach((e) => {
        const key = e.path[0] as keyof FormFields;
        if (!fieldErrors[key]) fieldErrors[key] = e.message;
      });
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    setSubmitErr(null);
    try {
      await createContribution.mutateAsync({
        member_id: f.member_id,
        amount: Number(f.kiasi),
        month: f.mwezi,
        paid_at: f.tarehe,
        payment_method: "CASH",
        notes: f.maelezo || undefined,
      });
      onClose();
    } catch (e: unknown) {
      setSubmitErr(e instanceof Error ? e.message : "Imeshindikana kurekodi");
    }
  };

  const set = <K extends keyof FormFields>(k: K, v: FormFields[K]) => {
    setF((p) => ({ ...p, [k]: v }));
    if (errors[k]) setErrors((p) => { const c = { ...p }; delete c[k]; return c; });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={onClose}>
      <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-display text-lg font-semibold">Ingiza Mchango</h3>
          <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-4 w-4" /></button>
        </div>
        {submitErr && <p className="mb-3 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{submitErr}</p>}
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-muted-foreground">Mwanachama</span>
          <select value={f.member_id} onChange={(e) => set("member_id", e.target.value)} className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm">
            {members.map((w) => <option key={w.id} value={w.id}>{w.full_name}</option>)}
          </select>
          {errors.member_id && <p className="mt-1 text-xs text-destructive">{errors.member_id}</p>}
        </label>
        <div className="mt-3">
          <Field label="Kiasi (TZS)" value={f.kiasi} onChange={(v) => set("kiasi", v)} type="number" />
          {errors.kiasi && <p className="mt-1 text-xs text-destructive">{errors.kiasi}</p>}
        </div>
        <div className="mt-3">
          <Field label="Mwezi" value={f.mwezi} onChange={(v) => set("mwezi", v)} type="month" />
          {errors.mwezi && <p className="mt-1 text-xs text-destructive">{errors.mwezi}</p>}
        </div>
        <div className="mt-3">
          <Field label="Tarehe ya malipo" value={f.tarehe} onChange={(v) => set("tarehe", v)} type="date" />
          {errors.tarehe && <p className="mt-1 text-xs text-destructive">{errors.tarehe}</p>}
        </div>
        <div className="mt-3">
          <Field label="Maelezo (hiari)" value={f.maelezo ?? ""} onChange={(v) => set("maelezo", v)} />
        </div>
        <button
          disabled={createContribution.isPending}
          onClick={handleSubmit}
          className="mt-5 w-full rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-50 inline-flex items-center justify-center gap-2"
        >
          {createContribution.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
          Hifadhi Mchango
        </button>
      </div>
    </div>
  );
}
