import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useMemberDashboardSummary, useGroupDashboardSummary, useKatibuDashboardSummary, useHazinaDashboardSummary, useUserRoles } from "@/hooks/use-scoped-dashboard";
import { AppShell } from "@/components/AppShell";
import { useAuth } from "@/lib/auth-provider";
import { useRoleSwitch } from "@/lib/role-context";
import { MemberAkibaCard, GroupBalanceCard, PersonalMemberStatsCard } from "@/components/DashboardCards";
import { roleMap, type Jukumu } from "@/api/types";
import { roleSubtitle } from "@/lib/roles";
import { requireAuth } from "@/lib/role-guards";
import { tzs } from "@/lib/format";
import { Users, ShieldCheck, TrendingUp, Receipt, PiggyBank, Wallet, ClipboardList, AlertCircle, Loader2, Crown } from "lucide-react";

export const Route = createFileRoute("/dashibodi")({
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
  const { currentRole } = useRoleSwitch();

  if (!user) return null;

  const jina = user.name.split(" ")[0] || "rafiki";
  const displayRole = currentRole || roleMap[user.role] || "Mwanachama";

  return (
    <AppShell
      title={`Habari, ${jina}`}
      subtitle={roleSubtitle[displayRole] || ""}
    >
      <div className="mb-5 flex items-center gap-2">
        <span className="chip bg-primary/10 text-primary">
          {displayRole}
        </span>
        <span className="text-xs text-muted-foreground">
          Mwonekano umetengenezwa kwa jukumu lako
        </span>
      </div>

      {displayRole === "Mwenyekiti" && <ChairmanView userId={user.id} groupId={user.group_id} memberId={user.member_id} />}
      {displayRole === "Mweka Hazina" && <TreasurerView groupId={user.group_id} />}
      {displayRole === "Katibu" && <SecretaryView groupId={user.group_id} />}
      {displayRole === "Mwanachama" && <MemberView memberId={user.member_id} />}
      {displayRole === "Msimamizi" && <AdminView />}
    </AppShell>
  );
}

// ============================================================================
// MWENYEKITI - Uses groupSummary + memberSummary
// ============================================================================

function ChairmanView({ userId, groupId, memberId }: { userId: string; groupId?: string; memberId?: string }) {
  const groupIdVal = groupId || "1"; // TODO: Get from context/auth
  const { data: groupData, isLoading: groupLoading, error: groupError } = useGroupDashboardSummary(groupIdVal);

  if (groupLoading) {
    return (
      <div className="space-y-5">
        <div className="card-surface animate-pulse">
          <div className="h-12 bg-muted rounded w-40 mb-4" />
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-16 bg-muted rounded" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (groupError || !groupData) {
    return (
      <div className="card-surface border border-destructive/50 bg-destructive/5">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1">Imeshindikana kupakia</h3>
            <p className="text-xs text-muted-foreground">
              Haijapatikana data ya kikundi. Tafadhali jaribu tena.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Group balance card */}
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        <GroupBalanceCard groupId={groupIdVal} />

        {/* Personal member stats (if user is also a member) */}
        {memberId && <PersonalMemberStatsCard memberId={memberId} />}
      </div>

      {/* Quick actions */}
      <div className="mt-6">
        <SectionTitle>Kazi zako kuu</SectionTitle>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <QuickAction to="/mikopo" icon={ShieldCheck} label="Idhinisha Mikopo" />
          <QuickAction to="/wanachama" icon={Users} label="Wanachama" />
          <QuickAction to="/ripoti" icon={TrendingUp} label="Tazama Ripoti" />
          <QuickAction to="/marejesho" icon={Receipt} label="Marejesho" />
        </div>
      </div>
    </>
  );
}

// ============================================================================
// MWEKA HAZINA - Uses hazinaSummary
// ============================================================================

function TreasurerView({ groupId }: { groupId?: string }) {
  const groupIdVal = groupId || "1"; // TODO: Get from context/auth
  const { data: hazinData, isLoading, error } = useHazinaDashboardSummary(groupIdVal);

  if (isLoading) {
    return (
      <div className="card-surface animate-pulse">
        <div className="h-12 bg-muted rounded w-40 mb-4" />
        <div className="grid grid-cols-2 gap-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-16 bg-muted rounded" />
          ))}
        </div>
      </div>
    );
  }

  if (error || !hazinData) {
    return (
      <div className="card-surface border border-destructive/50 bg-destructive/5">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1">Imeshindikana kupakia</h3>
            <p className="text-xs text-muted-foreground">
              Haijapatikana data ya hazina. Tafadhali jaribu tena.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Mapato section */}
      <div className="card-surface">
        <h3 className="font-semibold text-sm mb-4">Mapato ya Mwezi Huu</h3>
        <p className="text-3xl font-bold text-primary mb-4">
          {tzs(Number(hazinData.cash_in_this_period))}
        </p>
        <div className="grid grid-cols-2 gap-3">
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Imechukuniwa</p>
            <p className="font-semibold">{tzs(Number(hazinData.cash_in_confirmed))}</p>
          </div>
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Inasub.</p>
            <p className="font-semibold">
              {tzs(Number(hazinData.cash_in_pending))} ({hazinData.cash_in_pending_count})
            </p>
          </div>
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Malipo</p>
            <p className="font-semibold">{tzs(Number(hazinData.repayments_this_month))}</p>
          </div>
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Salio</p>
            <p className="font-semibold text-primary">{tzs(Number(hazinData.available_balance))}</p>
          </div>
        </div>
      </div>

      {/* Quick actions */}
      <div className="mt-6">
        <SectionTitle>Kazi zako za leo</SectionTitle>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          <QuickAction to="/michango" icon={PiggyBank} label="Pokea Mchango" />
          <QuickAction to="/marejesho" icon={Receipt} label="Pokea Marejesho" />
          <QuickAction to="/mikopo" icon={Wallet} label="Simamia Mikopo" />
        </div>
      </div>

      {/* Recent disbursements */}
      {hazinData.recent_disbursements.length > 0 && (
        <>
          <SectionTitle className="mt-6">Mikopo iliyopewa karibuni</SectionTitle>
          <div className="card-surface divide-y divide-border">
            {hazinData.recent_disbursements.slice(0, 5).map((d) => (
              <div key={d.loan_id} className="flex items-center justify-between px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{d.full_name}</p>
                  <p className="text-xs text-muted-foreground">{d.member_no}</p>
                </div>
                <p className="shrink-0 font-semibold">{tzs(Number(d.amount))}</p>
              </div>
            ))}
          </div>
        </>
      )}
    </>
  );
}

// ============================================================================
// KATIBU - Uses katibuSummary
// ============================================================================

function SecretaryView({ groupId }: { groupId?: string }) {
  const groupIdVal = groupId || "1"; // TODO: Get from context/auth
  const { data: katibuData, isLoading, error } = useKatibuDashboardSummary(groupIdVal);

  if (isLoading) {
    return (
      <div className="card-surface animate-pulse">
        <div className="h-12 bg-muted rounded w-40 mb-4" />
        <div className="grid grid-cols-2 gap-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-16 bg-muted rounded" />
          ))}
        </div>
      </div>
    );
  }

  if (error || !katibuData) {
    return (
      <div className="card-surface border border-destructive/50 bg-destructive/5">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1">Imeshindikana kupakia</h3>
            <p className="text-xs text-muted-foreground">
              Haijapatikana data ya katibu. Tafadhali jaribu tena.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Members card */}
      <div className="card-surface">
        <h3 className="font-semibold text-sm mb-4">Wanachama wa Kikundi</h3>
        <p className="text-3xl font-bold text-primary mb-4">
          {katibuData.total_active_members}
        </p>
        <div className="grid grid-cols-2 gap-3">
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Wapya mwezi huu</p>
            <p className="font-semibold">{katibuData.members_joined_this_month}</p>
          </div>
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Waliondoka</p>
            <p className="font-semibold">{katibuData.members_left_this_month}</p>
          </div>
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Inasub. idhinisho</p>
            <p className="font-semibold">{katibuData.pending_user_approvals}</p>
          </div>
          <div className="bg-muted/30 rounded p-3">
            <p className="text-xs text-muted-foreground mb-1">Walicho chelezo</p>
            <p className="font-semibold text-destructive">{katibuData.late_payments_count}</p>
          </div>
        </div>
      </div>

      {/* Quick actions */}
      <div className="mt-6">
        <SectionTitle>Kazi zako</SectionTitle>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          <QuickAction to="/wanachama" icon={ClipboardList} label="Sajili Mwanachama" />
          <QuickAction to="/michango" icon={PiggyBank} label="Kumbukumbu" />
          <QuickAction to="/ripoti" icon={TrendingUp} label="Andaa Ripoti" />
        </div>
      </div>

      {/* Late payments list */}
      {katibuData.late_payments.length > 0 && (
        <>
          <SectionTitle className="mt-6">Wanachama walio chelezo</SectionTitle>
          <div className="card-surface divide-y divide-border">
            {katibuData.late_payments.slice(0, 8).map((p, i) => (
              <div key={`${p.member_id}-${i}`} className="flex items-center justify-between px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{p.full_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {p.member_no} • {p.period_label}
                  </p>
                </div>
                <span className="chip bg-destructive/30 text-destructive">Chelezo</span>
              </div>
            ))}
          </div>
        </>
      )}
    </>
  );
}

// ============================================================================
// MWANACHAMA - Uses memberSummary
// ============================================================================

function MemberView({ memberId }: { memberId?: string }) {
  const { data: memberData, isLoading, error } = useMemberDashboardSummary(memberId);

  if (!memberId) {
    return (
      <div className="card-surface border border-amber-200 bg-amber-50">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-amber-700 shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1 text-amber-700">Umefungwa</h3>
            <p className="text-xs text-amber-700">
              Unaloanzisha kazi kama mwanachama, lakini baada haijapatikana kiunganisho cha Mwanachama. Tafadhali wasiliana na msimamizi.
            </p>
          </div>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="card-surface animate-pulse">
        <div className="h-12 bg-muted rounded w-40 mb-4" />
        <div className="space-y-3">
          <div className="h-16 bg-muted rounded" />
          <div className="h-16 bg-muted rounded" />
        </div>
      </div>
    );
  }

  if (error || !memberData) {
    return (
      <div className="card-surface border border-destructive/50 bg-destructive/5">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1">Imeshindikana kupakia</h3>
            <p className="text-xs text-muted-foreground">
              Haijapatikana data ya akiba yako. Tafadhali jaribu tena.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Main member card */}
      <MemberAkibaCard memberId={memberId} />

      {/* Quick actions */}
      <div className="mt-6">
        <SectionTitle>Kazi zako</SectionTitle>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          <QuickAction to="/weka-mchango" icon={PiggyBank} label="Weka Mchango" />
          <QuickAction to="/mikopo" icon={Wallet} label="Omba Mkopo" />
          <QuickAction to="/historia" icon={ClipboardList} label="Historia" />
        </div>
      </div>
    </>
  );
}

// ============================================================================
// MSIMAMIZI - Admin view
// ============================================================================

function AdminView() {
  return (
    <div className="card-surface">
      <h3 className="font-semibold text-sm">Zana za Msimamizi</h3>
      <p className="text-sm text-muted-foreground mt-2">
        Huduma za utawala ya mfumo - wanachama, watumiaji, na maandalizi.
      </p>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-3 mt-4">
        <QuickAction to="/huduma/watumiaji" icon={Users} label="Watumiaji" />
        <QuickAction to="/huduma/baadhi" icon={Crown} label="Mipangilio" />
        <QuickAction to="/huduma/sisi" icon={TrendingUp} label="Safu" />
      </div>
    </div>
  );
}

// ============================================================================
// HELPER COMPONENTS
// ============================================================================

function SectionTitle({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <h2 className={`font-semibold text-lg mb-4 mt-5 ${className}`}>{children}</h2>;
}

function QuickAction({ to, icon: Icon, label }: { to: string; icon: React.ComponentType<{ className?: string }>; label: string }) {
  return (
    <a
      href={to}
      className="flex flex-col items-center gap-2 rounded-xl border border-border p-3 text-center text-sm font-medium transition-colors hover:bg-muted"
    >
      <Icon className="h-5 w-5 text-primary" />
      <span>{label}</span>
    </a>
  );
}
