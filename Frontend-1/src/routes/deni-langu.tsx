import { createFileRoute } from "@tanstack/react-router";
import { AppShell } from "@/components/AppShell";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { useMemberObligations } from "@/hooks/use-obligations";
import { tzs } from "@/lib/format";
import { Wallet, CalendarClock, ReceiptText, Loader2 } from "lucide-react";

export const Route = createFileRoute("/deni-langu")({
  head: () => ({
    meta: [
      { title: "Deni Langu — Money Seeking" },
      { name: "description", content: "Jumla unayodaiwa: malimbikizo, mchango wa sasa na faini." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
  },
  component: DeniLanguPage,
});

function DeniLanguPage() {
  const { user } = useAuth();
  const memberId = user?.member_id ?? null;
  const { data, isLoading, error } = useMemberObligations(memberId);

  if (isLoading) {
    return (
      <AppShell title="Deni Langu" subtitle="Inapakia...">
        <div className="mt-8 flex justify-center"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      </AppShell>
    );
  }
  if (error || !data?.data) {
    return (
      <AppShell title="Deni Langu" subtitle="Imeshindikana kupakia">
        <p className="text-sm text-muted-foreground">Jaribu tena baadaye.</p>
      </AppShell>
    );
  }

  const ob = data.data;
  const grand = Number(ob.grand_total_owed);

  return (
    <AppShell title="Deni Langu" subtitle="Jumla unayodaiwa na kikundi">
      <div className="max-w-3xl space-y-4">
        <section className="card-surface overflow-hidden" data-testid="grand-total-card">
          <div className="bg-primary px-4 py-5 text-primary-foreground">
            <p className="text-xs uppercase tracking-wide opacity-80">Jumla Unayodaiwa</p>
            <p className="font-display text-3xl font-bold" data-testid="grand-total">
              {tzs(grand)}
            </p>
            <p className="mt-1 text-xs opacity-80">{ob.full_name} · {ob.member_no}</p>
          </div>
        </section>

        <Section
          testId="arrears-section"
          icon={CalendarClock}
          title="Michango Iliyopita Isiyolipwa"
          total={Number(ob.total_arrears)}
          empty="Huna malimbikizo. Hongera!"
        >
          {ob.itemized_arrears.map((a) => (
            <Row
              key={a.cycle_label}
              label={`Mzunguko ${a.cycle_label}`}
              sub={`Inadaiwa ${new Date(a.due_date).toLocaleDateString()}`}
              value={tzs(Number(a.owed))}
            />
          ))}
        </Section>

        <Section
          testId="current-section"
          icon={Wallet}
          title={`Mchango wa Sasa${ob.current_cycle_label ? ` (${ob.current_cycle_label})` : ""}`}
          total={Number(ob.current_cycle_due)}
          empty="Mchango wa sasa umelipwa."
        >
          {Number(ob.current_cycle_due) > 0 && (
            <Row label="Inayodaiwa sasa" sub="Mzunguko ulio wazi" value={tzs(Number(ob.current_cycle_due))} />
          )}
        </Section>

        <Section
          testId="fines-section"
          icon={ReceiptText}
          title="Faini"
          total={Number(ob.total_fines_unpaid)}
          empty="Huna faini."
        >
          {ob.itemized_fines.map((f) => (
            <Row
              key={f.id}
              label={f.offence_name}
              sub={`${new Date(f.occurrence_date).toLocaleDateString()} · ${f.status === "unpaid" ? "Haijalipwa" : f.status}${f.waiver_status === "pending" ? " · Msamaha unaosubiri" : ""}`}
              value={tzs(Number(f.amount))}
            />
          ))}
        </Section>
      </div>
    </AppShell>
  );
}

function Section({ testId, icon: Icon, title, total, empty, children }: {
  testId: string;
  icon: any;
  title: string;
  total: number;
  empty: string;
  children: React.ReactNode;
}) {
  const hasItems = Array.isArray(children)
    ? (children as unknown[]).length > 0
    : !!children;
  return (
    <section className="card-surface overflow-hidden" data-testid={testId}>
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <span className="flex items-center gap-2.5">
          <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
            <Icon className="h-4 w-4" />
          </span>
          <h3 className="font-display text-sm font-semibold">{title}</h3>
        </span>
        <span className="text-sm font-bold" data-testid={`${testId}-total`}>{tzs(total)}</span>
      </header>
      <div>
        {hasItems ? children : (
          <p className="px-4 py-3 text-sm text-muted-foreground">{empty}</p>
        )}
      </div>
    </section>
  );
}

function Row({ label, sub, value }: { label: string; sub: string; value: string }) {
  return (
    <div className="flex items-center justify-between border-t border-border px-4 py-2.5 first:border-t-0">
      <span>
        <span className="block text-sm font-medium">{label}</span>
        <span className="block text-xs text-muted-foreground">{sub}</span>
      </span>
      <span className="text-sm font-semibold">{value}</span>
    </div>
  );
}
