/**
 * Role-specific dashboard card components — styled to match dashibodi-old.tsx
 * hero-surface / card-surface / chip with same colours.
 */

import { useQuery } from "@tanstack/react-query";
import { dashboardApi } from "@/api/dashboard";
import { tzs } from "@/lib/format";
import { AlertCircle, Loader2, Heart, Clock } from "lucide-react";

// ---------- shared bits (mirrors dashibodi-old.tsx) ----------
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

function LoadingSkeleton() {
  return (
    <div className="flex justify-center py-12">
      <Loader2 className="h-8 w-8 animate-spin text-primary" />
    </div>
  );
}

// ============================================================================
// MEMBER DASHBOARD CARD - "Akiba Yangu" (for Mwanachama + leadership personal)
// ============================================================================

export function MemberAkibaCard({ memberId }: { memberId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["member-dashboard", memberId],
    queryFn: () => dashboardApi.memberSummary(memberId),
    enabled: !!memberId,
  });

  if (isLoading) return <LoadingSkeleton />;

  if (error || !data) {
    return (
      <div className="card-surface border border-destructive/50 bg-destructive/5">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1">Imeshindikana kupakia</h3>
            <p className="text-xs text-muted-foreground">Haijapatikana data ya akiba yako. Tafadhali jaribu tena.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      <HeroBalance
        label="Akiba Yangu"
        value={tzs(Number(data.total_contributions ?? 0))}
        stats={[
          ["Michango", String(data.contributions_count ?? 0)],
          ["Mikopo wazi", String(data.outstanding_loans_count ?? 0)],
          ["Deni bado", tzs(data.outstanding_loans_balance ?? 0)],
          ["Mikopo iliyofungwa", String(data.closed_loans_count ?? 0)],
        ]}
      />
      {/* extra details — same card-surface / chip colours as old */}
      {(data.pending_contributions_count > 0 || data.rejected_contributions_count > 0) && (
        <div className="mt-3 flex flex-wrap gap-2">
          {data.pending_contributions_count > 0 && (
            <span className="chip bg-warning/30 text-foreground flex items-center gap-1">
              <Clock className="h-3.5 w-3.5" /> {data.pending_contributions_count} inasubiri
            </span>
          )}
          {data.rejected_contributions_count > 0 && (
            <span className="chip bg-destructive/15 text-destructive">{data.rejected_contributions_count} yalikataliwa</span>
          )}
        </div>
      )}
      {data.welfare_contributions_total !== "0" && (
        <div className="card-surface mt-3 flex items-center justify-between p-4">
          <span className="flex items-center gap-2 text-sm font-medium">
            <Heart className="h-4 w-4 text-pink-500" /> Mfuko wa Kijamii
          </span>
          <span className="text-sm font-bold">{tzs(Number(data.welfare_contributions_total))}</span>
        </div>
      )}
    </>
  );
}

// ============================================================================
// LEADERSHIP DASHBOARD CARD - "Salio la Kikundi" (for Mwenyekiti/Katibu/Hazina)
// ============================================================================

export function GroupBalanceCard({ groupId }: { groupId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["group-dashboard", groupId],
    queryFn: () => dashboardApi.groupSummary(groupId),
    enabled: !!groupId,
  });

  if (isLoading) return <LoadingSkeleton />;

  if (error || !data) {
    return (
      <div className="card-surface border border-destructive/50 bg-destructive/5">
        <div className="flex items-start gap-3">
          <AlertCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
          <div>
            <h3 className="font-semibold text-sm mb-1">Imeshindikana kupakia</h3>
            <p className="text-xs text-muted-foreground">Haijapatikana data ya kikundi. Tafadhali jaribu tena.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <HeroBalance
      label="Salio la Kikundi"
      value={tzs(Number(data.available_balance ?? 0))}
      stats={[
        ["Wanachama hai", String(data.total_active_members ?? 0)],
        ["Mikopo wazi", String(data.outstanding_loans_count ?? 0)],
        ["Deni bado", tzs(data.outstanding_loans_balance ?? 0)],
        ["Michango", tzs(data.total_contributions ?? 0)],
      ]}
    />
  );
}

// ============================================================================
// MEMBER PERSONAL CARD (for leadership to view their own member stats)
// Matches dashibodi-old.tsx Member info cards: card-surface + CheckCircle2
// ============================================================================

export function PersonalMemberStatsCard({ memberId }: { memberId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["member-personal-stats", memberId],
    queryFn: () => dashboardApi.memberSummary(memberId),
    enabled: !!memberId,
  });

  if (isLoading) return <LoadingSkeleton />;

  if (error || !data) return null;

  // Same hero-surface design as GroupBalanceCard ("Salio la Kikundi") — content unchanged
  return (
    <HeroBalance
      label="Akiba Yangu Binafsi"
      value={tzs(Number(data.total_contributions ?? 0))}
      stats={[
        ["Michango", String(data.contributions_count ?? 0)],
        ["Mikopo wazi", String(data.outstanding_loans_count ?? 0)],
        ["Deni bado", tzs(data.outstanding_loans_balance ?? 0)],
        ["Inasub.", String(data.pending_contributions_count ?? 0)],
      ]}
    />
  );
}
