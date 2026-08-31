import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useDashboard } from "@/hooks/use-dashboard";
import { useMembers } from "@/hooks/use-members";
import { useLoans } from "@/hooks/use-loans";
import { useContributions, useMonthlyReport } from "@/hooks/use-contributions";
import { AppShell } from "@/components/AppShell";
import { tzs } from "@/lib/format";
import { useAuth } from "@/lib/auth-provider";
import { roleMap } from "@/api/types";
import { groupsApi, INTERVAL_LABELS } from "@/api/groups";
import { roleSubtitle } from "@/lib/roles";
import { requireAuth } from "@/lib/role-guards";
import { useSystemHealth } from "@/hooks/use-admin";
import { ArrowRight, Users, PiggyBank, Banknote, Receipt, TrendingUp, AlertCircle, ShieldCheck, Wallet, ClipboardList, CheckCircle2, Loader2, Shield, Activity, Settings, CalendarDays } from "lucide-react";

export const Route = createFileRoute("/dashibodi-old")({
  head: () => ({
    meta: [
      { title: "Dashibodi — Money Seeking" },
      { name: "description", content: "Muhtasari wa hali ya kikundi: wanachama, michango, mikopo na marejesho." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
  },
  component: Dashibodi,
});

function Dashibodi() {
  const { user } = useAuth();
  if (!user) return null;
  const jina = user.name.split(" ")[0] || "rafiki";
  const jukumu = roleMap[user.role] ?? "Mwanachama";
  return (
    <AppShell title={`Habari, ${jina}`} subtitle={roleSubtitle[jukumu]}>
      <div className="mb-5 flex items-center gap-2">
        <span className="chip bg-primary/10 text-primary">{jukumu}</span>
        <span className="text-xs text-muted-foreground">Mwonekano umetengenezwa kwa jukumu lako</span>
      </div>
      {jukumu === "Mwenyekiti" && <ChairmanView />}
      {jukumu === "Mweka Hazina" && <TreasurerView />}
      {jukumu === "Katibu" && <SecretaryView />}
      {jukumu === "Mwanachama" && <MemberView userId={user.id} userName={user.name} memberId={user.member_id || null} memberCode={user.member_code || null} userPhone={user.phone} />}
      {jukumu === "Msimamizi" && <AdminView />}
    </AppShell>
  );
}

// ---------- MWENYEKITI ----------
function ChairmanView() {
  const { data: dash, isLoading } = useDashboard();
  const { data: loansData } = useLoans({ status: "OUTSTANDING", limit: 10 });

  if (isLoading) return <LoadingSkeleton />;

  const openLoans = loansData?.data ?? [];

  return (
    <>
      <HeroBalance label="Salio la Kikundi" value={tzs(Number(dash?.total_contributions ?? 0) + Number(dash?.total_repayments ?? 0) - Number(dash?.total_loans_issued ?? 0))} stats={[
        ["Wanachama hai", String(dash?.total_active_members ?? 0)],
        ["Mikopo wazi", String(dash?.count_outstanding_loans ?? 0)],
        ["Deni bado", tzs(dash?.total_outstanding_balance ?? 0)],
        ["Michango", tzs(dash?.total_contributions ?? 0)],
      ]} />
      <SectionTitle>Kazi zako kuu</SectionTitle>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <QuickAction to="/mikopo" icon={ShieldCheck} label="Idhinisha Mikopo" />
        <QuickAction to="/wanachama" icon={Users} label="Wanachama" />
        <QuickAction to="/ripoti" icon={TrendingUp} label="Tazama Ripoti" />
        <QuickAction to="/marejesho" icon={Receipt} label="Marejesho" />
      </div>
      <SectionTitle>Mikopo inayohitaji uangalizi</SectionTitle>
      <div className="card-surface divide-y divide-border">
        {openLoans.slice(0, 6).map((l) => (
          <div key={l.id} className="flex items-center justify-between px-4 py-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{l.member?.full_name ?? `Mwanachama #${l.member_id}`}</p>
              <p className="text-xs text-muted-foreground">Mwisho: {l.due_date}</p>
            </div>
            <p className="shrink-0 text-sm font-semibold">{tzs(l.balance_remaining ?? 0)}</p>
          </div>
        ))}
        {openLoans.length === 0 && <p className="px-4 py-6 text-center text-sm text-muted-foreground">Hakuna mikopo wazi.</p>}
      </div>
    </>
  );
}

// ---------- MWEKA HAZINA ----------
function TreasurerView() {
  const { data: dash, isLoading } = useDashboard();
  const { data: reportData } = useMonthlyReport();

  if (isLoading) return <LoadingSkeleton />;

  const report = reportData?.data ?? [];
  const walipa = report.filter((r) => r.status === "AMELIPA");
  const wadaiwa = report.filter((r) => r.status === "HAJALIPA");

  return (
    <>
      <HeroBalance label="Mapato ya Mwezi Huu" value={tzs(Number(dash?.total_contributions_this_month ?? 0) + Number(dash?.total_repayments_this_month ?? 0))} stats={[
        ["Michango", tzs(dash?.total_contributions_this_month ?? 0)],
        ["Marejesho", tzs(dash?.total_repayments_this_month ?? 0)],
        ["Walipa", String(walipa.length)],
        ["Wadaiwa", String(wadaiwa.length)],
      ]} />
      <SectionTitle>Kazi zako za leo</SectionTitle>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
        <QuickAction to="/michango" icon={PiggyBank} label="Pokea Mchango" />
        <QuickAction to="/marejesho" icon={Receipt} label="Pokea Marejesho" />
        <QuickAction to="/mikopo" icon={Wallet} label="Simamia Mikopo" />
      </div>
      <SectionTitle>Wadaiwa Mwezi Huu</SectionTitle>
      <div className="card-surface divide-y divide-border">
        {wadaiwa.slice(0, 8).map((r) => (
          <div key={r.member_no} className="flex items-center justify-between px-4 py-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{r.full_name}</p>
              <p className="text-xs text-muted-foreground">{r.phone}</p>
            </div>
            <Link to="/michango" className="chip bg-warning/30 text-foreground">Pokea</Link>
          </div>
        ))}
        {wadaiwa.length === 0 && <p className="px-4 py-6 text-center text-sm text-success">Wote wamelipa 🎉</p>}
      </div>
    </>
  );
}

// ---------- KATIBU ----------
function SecretaryView() {
  const { data: dash, isLoading } = useDashboard();
  const { data: membersData } = useMembers({ limit: 5 });

  if (isLoading) return <LoadingSkeleton />;

  const members = membersData?.data ?? [];
  const wapya = [...members].sort((a, b) => b.joined_at.localeCompare(a.joined_at)).slice(0, 5);

  return (
    <>
      <HeroBalance label="Wanachama wa Kikundi" value={String(dash?.total_active_members ?? 0)} stats={[
        ["Hai", String(dash?.total_active_members ?? 0)],
        ["Hawajalipa mwezi huu", String(dash?.members_defaulted_this_month ?? 0)],
        ["Mikopo wazi", String(dash?.count_outstanding_loans ?? 0)],
        ["Mikopo inayosubiri", String(dash?.count_pending_loans ?? 0)],
      ]} />
      <SectionTitle>Kazi zako</SectionTitle>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
        <QuickAction to="/wanachama" icon={ClipboardList} label="Sajili Mwanachama" />
        <QuickAction to="/michango" icon={PiggyBank} label="Kumbukumbu za Michango" />
        <QuickAction to="/ripoti" icon={TrendingUp} label="Andaa Ripoti" />
      </div>
      <SectionTitle>Wanachama wapya</SectionTitle>
      <div className="card-surface divide-y divide-border">
        {wapya.map((w) => (
          <div key={w.id} className="flex items-center justify-between px-4 py-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{w.full_name}</p>
              <p className="text-xs text-muted-foreground">{w.member_no} · {w.joined_at}</p>
            </div>
            <span className={`chip ${w.is_active ? "bg-success/15 text-success" : "bg-muted text-muted-foreground"}`}>{w.is_active ? "Hai" : "Hahai"}</span>
          </div>
        ))}
      </div>
    </>
  );
}

// ---------- MWANACHAMA ----------
function MemberView({ userId, userName, memberId, memberCode, userPhone }: { userId: string; userName: string; memberId: string | null; memberCode: string | null; userPhone: string }) {
  const me = memberId ? { id: memberId, member_no: memberCode, phone: userPhone } : null;
  const memberIdVal = me?.id;

  // Group contribution settings — due-date banner for members
  const { data: settingsData } = useQuery({
    queryKey: ["groups", "settings"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const fixedAmount = settingsData?.data?.fixed_contribution_amount != null
    ? Number(settingsData.data.fixed_contribution_amount)
    : null;
  const nextDue = settingsData?.next_due_date ?? null;

  // Personal totals only — never show group dashboard as "Akiba Yangu"
  const { data: contribsData, isLoading: contribsLoading } = useContributions({
    member_id: memberIdVal,
    limit: 200,
    enabled: !!memberIdVal,
  });
  const { data: loansData, isLoading: loansLoading } = useLoans({
    member_id: memberIdVal,
    limit: 200,
    enabled: !!memberIdVal,
  });

  if (!!memberIdVal && (contribsLoading || loansLoading)) {
    return <LoadingSkeleton />;
  }

  if (!me) {
    return (
      <div className="card-surface p-6 text-center">
        <AlertCircle className="mx-auto h-10 w-10 text-warning" />
        <h2 className="mt-3 font-display text-lg font-bold">Jisajili kama mwanachama</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Karibu <span className="font-semibold">{userName}</span> — kamilisha usajili wako kupitia ukurasa wa wasifu ili uweze kuomba mikopo na kuona michango yako.
        </p>
        <Link to="/wasifu" className="mt-4 inline-flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground">
          Nenda kwenye Wasifu <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    );
  }

  const myContributions = contribsData?.data ?? [];
  const myLoans = loansData?.data ?? [];
  const totalContributions = myContributions.reduce((s, c) => s + Number(c.amount), 0);
  const outstandingLoans = myLoans.filter((l) => l.status === "OUTSTANDING");
  const outstandingBalance = outstandingLoans.reduce((s, l) => s + Number(l.balance_remaining ?? 0), 0);
  const closedLoans = myLoans.filter((l) => l.status === "CLOSED").length;

  return (
    <>
      {(fixedAmount != null || nextDue) && (
        <div className="card-surface p-4 mb-4 border-l-4 border-l-primary flex items-center gap-3">
          <CalendarDays className="h-5 w-5 shrink-0 text-primary" />
          <div>
            <p className="text-sm font-semibold">
              Mchango ujao
              {fixedAmount != null && <>: TZS {fixedAmount.toLocaleString()}</>}
              {nextDue && <> · ifikapo {nextDue}</>}
            </p>
            <p className="text-xs text-muted-foreground">
              Kipindi: {settingsData ? INTERVAL_LABELS[settingsData.data.contribution_interval] : "—"} · Wasilisha kupitia "Weka Mchango"
            </p>
          </div>
          <Link to="/weka-mchango" className="ml-auto shrink-0 rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:bg-primary/90">
            Weka Mchango
          </Link>
        </div>
      )}
      <HeroBalance label="Akiba Yangu" value={tzs(totalContributions)} stats={[
        ["Michango", String(myContributions.length)],
        ["Mikopo wazi", String(outstandingLoans.length)],
        ["Deni bado", tzs(outstandingBalance)],
        ["Mikopo iliyofungwa", String(closedLoans)],
      ]} />

      <div className="mt-5 grid gap-3 md:grid-cols-2">
        <div className="card-surface flex items-center gap-3 p-4">
          <CheckCircle2 className="h-8 w-8 text-success" />
          <div>
            <p className="text-sm font-semibold">{me.member_no ? `Mwanachama #${me.member_no}` : `Mwanachama #${me.id?.slice(0, 8)}`}</p>
            <p className="text-xs text-muted-foreground">Namba ya simu: {me.phone}</p>
          </div>
        </div>
        <div className="card-surface p-4">
          <p className="text-xs text-muted-foreground">Namba ya simu</p>
          <p className="font-display text-lg font-bold">{me.phone}</p>
          <p className="text-xs text-muted-foreground">{me.member_no ?? "—"}</p>
        </div>
      </div>

      <SectionTitle>Michango Yangu</SectionTitle>
      {myContributions.length === 0 ? (
        <div className="card-surface p-6 text-center text-sm text-muted-foreground">
          Hakuna michango iliyorekodiwa bado.
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {myContributions.slice(0, 6).map((c) => (
            <div key={c.id} className="flex items-center justify-between px-4 py-3">
              <div>
                <p className="text-sm font-medium">{tzs(c.amount)}</p>
                <p className="text-xs text-muted-foreground">{c.month?.slice?.(0, 7) ?? c.month}</p>
              </div>
              <span className="chip bg-success/15 text-success">{c.status}</span>
            </div>
          ))}
        </div>
      )}
    </>
  );
}

// ---------- MSIMAMIZI ----------
function AdminView() {
  const { data: health, isLoading } = useSystemHealth();

  if (isLoading) return <LoadingSkeleton />;

  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div className="card-surface flex items-center gap-4 p-5">
          <span className="grid h-12 w-12 place-items-center rounded-xl bg-primary/10 text-primary">
            <Users className="h-6 w-6" />
          </span>
          <div>
            <p className="text-2xl font-bold">{health?.total_users ?? 0}</p>
            <p className="text-xs text-muted-foreground">Watumiaji wote</p>
          </div>
        </div>
        <div className="card-surface flex items-center gap-4 p-5">
          <span className="grid h-12 w-12 place-items-center rounded-xl bg-amber-100 text-amber-700">
            <Activity className="h-6 w-6" />
          </span>
          <div>
            <p className="text-2xl font-bold">{health?.active_users ?? 0}</p>
            <p className="text-xs text-muted-foreground">Wanaotumia</p>
          </div>
        </div>
        <div className="card-surface flex items-center gap-4 p-5">
          <span className="grid h-12 w-12 place-items-center rounded-xl bg-destructive/10 text-destructive">
            <AlertCircle className="h-6 w-6" />
          </span>
          <div>
            <p className="text-2xl font-bold">{health?.pending_users ?? 0}</p>
            <p className="text-xs text-muted-foreground">Wanaosubiri</p>
          </div>
        </div>
        <div className="card-surface flex items-center gap-4 p-5">
          <span className="grid h-12 w-12 place-items-center rounded-xl bg-success/15 text-success">
            <Users className="h-6 w-6" />
          </span>
          <div>
            <p className="text-2xl font-bold">{health?.total_members ?? 0}</p>
            <p className="text-xs text-muted-foreground">Wanachama</p>
          </div>
        </div>
        <div className="card-surface flex items-center gap-4 p-5">
          <span className="grid h-12 w-12 place-items-center rounded-xl bg-amber-100 text-amber-700">
            <Activity className="h-6 w-6" />
          </span>
          <div>
            <p className="text-2xl font-bold">{health?.recent_logins_24h ?? 0}</p>
            <p className="text-xs text-muted-foreground">Waliingia leo</p>
          </div>
        </div>
      </div>

      <SectionTitle>Haraka</SectionTitle>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <QuickAction to="/admin" icon={Settings} label="Simamia Watumiaji" />
        <QuickAction to="/admin-logs" icon={Activity} label="Kumbukumbu" />
        <QuickAction to="/wanachama" icon={Users} label="Wanachama" />
        <QuickAction to="/wasifu" icon={Shield} label="Wasifu Wangu" />
      </div>
    </>
  );
}

// ---------- shared bits ----------
function HeroBalance({ label, value, stats }: { label: string; value: string; stats: [string, string][] }) {
  return (
    <section className="hero-surface px-5 py-6 lg:px-7 lg:py-8">
      <p className="text-xs font-medium uppercase tracking-wider text-primary-foreground/80">{label}</p>
      <p className="mt-2 font-display text-4xl font-extrabold lg:text-5xl">{value}</p>
      <div className="mt-5 grid grid-cols-2 gap-3 text-sm lg:grid-cols-4">
        {stats.map(([k, v]) => (
          <div key={k} className="rounded-xl bg-white/15 px-3 py-2.5">
            <p className="text-xs text-primary-foreground/80">{k}</p>
            <p className="font-semibold">{v}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h2 className="mb-3 mt-7 font-display text-base font-semibold">{children}</h2>;
}

function QuickAction({ to, icon: Icon, label }: { to: string; icon: any; label: string }) {
  return (
    <Link to={to} className="card-surface flex items-center gap-3 p-3.5 transition-colors hover:border-primary/40">
      <span className="grid h-10 w-10 place-items-center rounded-xl bg-primary text-primary-foreground">
        <Icon className="h-5 w-5" strokeWidth={2.25} />
      </span>
      <span className="text-sm font-semibold leading-tight">{label}</span>
    </Link>
  );
}

function LoadingSkeleton() {
  return (
    <div className="flex justify-center py-12">
      <Loader2 className="h-8 w-8 animate-spin text-primary" />
    </div>
  );
}
