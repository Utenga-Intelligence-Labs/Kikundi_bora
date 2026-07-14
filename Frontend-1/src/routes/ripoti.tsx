import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { AppShell } from "@/components/AppShell";
import { useDashboard } from "@/hooks/use-dashboard";
import { useMembers } from "@/hooks/use-members";
import { useLoans } from "@/hooks/use-loans";
import { tzs } from "@/lib/format";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, blockAdminFromPage, requireAuth, requireRole } from "@/lib/role-guards";
import { Users, PiggyBank, Banknote, Receipt, AlertCircle, Wallet, Loader2 } from "lucide-react";

export const Route = createFileRoute("/ripoti")({
  head: () => ({
    meta: [
      { title: "Ripoti — Money Seeking" },
      { name: "description", content: "Ripoti za kifedha za kikundi: wanachama, michango, mikopo, marejesho." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    requireRole("chair", "secretary", "treasurer");
    blockAdminFromPage();
  },
  component: RipotiPage,
});

function RipotiPage() {
  const { user } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (hasRole(user, "admin")) {
      navigate({ to: "/admin-logs", replace: true });
    }
  }, [user, navigate]);

  if (hasRole(user, "admin")) {
    return (
      <AppShell title="Ripoti" subtitle="Hali ya kifedha ya kikundi">
        <div className="mt-8 flex justify-center"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      </AppShell>
    );
  }

  const { data: dash, isLoading: dashLoading } = useDashboard();
  const { data: membersData } = useMembers({ limit: 500 });
  const { data: loansData } = useLoans({ status: "OUTSTANDING", limit: 100 });

  const members = membersData?.data ?? [];
  const activeMembers = members.filter((m) => m.is_active).length;
  const inactiveMembers = members.filter((m) => !m.is_active).length;
  const openLoans = loansData?.data ?? [];

  if (dashLoading) {
    return (
      <AppShell title="Ripoti" subtitle="Hali ya kifedha ya kikundi">
        <div className="mt-8 flex justify-center"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      </AppShell>
    );
  }

  return (
    <AppShell title="Ripoti" subtitle="Hali ya kifedha ya kikundi">
      <div className="grid gap-4 lg:grid-cols-2">
      <Card title="R-06 · Muhtasari wa Kifedha" icon={Wallet}>
        <Row label="Wanachama hai" value={String(dash?.total_active_members ?? activeMembers)} />
        <Row label="Jumla ya michango" value={tzs(dash?.total_contributions ?? 0)} />
        <Row label="Jumla ya mikopo iliyotolewa" value={tzs(dash?.total_loans_issued ?? 0)} />
        <Row label="Jumla ya marejesho" value={tzs(dash?.total_repayments ?? 0)} />
        <Row label="Salio la mikopo wazi" value={tzs(dash?.total_outstanding_balance ?? 0)} />
        <Row label="Michango mwezi huu" value={tzs(dash?.total_contributions_this_month ?? 0)} strong />
      </Card>

      <Card title="R-02 · Michango ya Mwezi" icon={PiggyBank}>
        <Row label="Walipa mwezi huu" value={`${dash?.members_paid_this_month ?? 0} wanachama`} />
        <Row label="Hawajalipa" value={`${dash?.members_defaulted_this_month ?? 0} wanachama`} />
        <Row label="Mikopo inayosubiri" value={`${dash?.count_pending_loans ?? 0} mikopo`} />
      </Card>

      <Card title="R-04 · Mikopo Isiyolipwa" icon={AlertCircle}>
        {openLoans.length === 0 && <p className="px-4 py-3 text-sm text-muted-foreground">Hakuna mikopo iliyo wazi.</p>}
        {openLoans.map((l) => (
          <div key={l.id} className="flex items-center justify-between px-4 py-3 border-t border-border first:border-t-0">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{l.member?.full_name ?? `Mwanachama #${l.member_id}`}</p>
              <p className="text-xs text-muted-foreground">Mkopo {tzs(l.approved_amount ?? l.amount)}</p>
            </div>
            <p className="text-sm font-semibold text-warning">{tzs(l.balance_remaining ?? 0)}</p>
          </div>
        ))}
      </Card>

      <Card title="R-01 · Wanachama" icon={Users}>
        <Row label="Hai" value={String(activeMembers)} />
        <Row label="Hahai" value={String(inactiveMembers)} />
        <Row label="Jumla" value={String(members.length)} strong />
      </Card>

      <Card title="R-03 / R-05 · Shughuli za Mikopo" icon={Banknote}>
        <Row label="Mikopo wazi" value={String(dash?.count_outstanding_loans ?? 0)} />
        <Row label="Malipo yote" value={String(dash?.total_repayments ? "—" : "—")} />
      </Card>
      </div>

      <p className="mt-6 text-center text-xs text-muted-foreground">
        Kupakua ripoti kwa PDF/Excel kutapatikana katika Awamu inayofuata.
      </p>
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
      <div>{children}</div>
    </section>
  );
}

function Row({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="flex items-center justify-between border-t border-border px-4 py-2.5 first:border-t-0">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className={`text-sm ${strong ? "font-display text-base font-bold" : "font-semibold"}`}>{value}</span>
    </div>
  );
}
