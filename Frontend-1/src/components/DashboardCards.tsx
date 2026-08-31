/**
 * Role-specific dashboard card components.
 * These cards bind to the role-scoped API endpoints from Task A backend.
 */

import { useQuery } from "@tanstack/react-query";
import { dashboardApi } from "@/api/dashboard";
import { tzs } from "@/lib/format";
import {
  PiggyBank,
  Heart,
  AlertCircle,
  Loader2,
  TrendingDown,
  DollarSign,
  Clock,
} from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";

// ============================================================================
// MEMBER DASHBOARD CARD - "Akiba Yangu"
// ============================================================================

export function MemberAkibaCard({ memberId }: { memberId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["member-dashboard", memberId],
    queryFn: () => dashboardApi.memberSummary(memberId),
    enabled: !!memberId,
  });

  if (isLoading) {
    return (
      <div className="card-surface animate-pulse">
        <div className="flex items-center justify-between mb-4">
          <div className="h-5 bg-muted rounded w-32" />
          <PiggyBank className="h-6 w-6 text-muted" />
        </div>
        <div className="h-8 bg-muted rounded w-40 mb-3" />
        <div className="h-4 bg-muted rounded w-24" />
      </div>
    );
  }

  if (error || !data) {
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
    <div className="card-surface">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-sm">Akiba Yangu</h3>
        <PiggyBank className="h-5 w-5 text-primary" />
      </div>

      {/* Main amount */}
      <div className="mb-4">
        <p className="text-3xl font-bold text-primary">
          {tzs(Number(data.total_contributions))}
        </p>
        <p className="text-xs text-muted-foreground">
          Michango {data.contributions_count}
        </p>
      </div>

      {/* Pending contributions warning */}
      {data.pending_contributions_count > 0 && (
        <Alert className="mb-3 border-amber-200 bg-amber-50">
          <Clock className="h-4 w-4 text-amber-700" />
          <AlertDescription className="text-xs text-amber-700">
            {data.pending_contributions_count} michango inasubiri uthibitisho
          </AlertDescription>
        </Alert>
      )}

      {/* Rejected contributions alert */}
      {data.rejected_contributions_count > 0 && (
        <Alert className="mb-3 border-destructive/50 bg-destructive/5">
          <AlertCircle className="h-4 w-4 text-destructive" />
          <AlertDescription className="text-xs text-destructive">
            {data.rejected_contributions_count} michango yalikataliwa
          </AlertDescription>
        </Alert>
      )}

      {/* Welfare contributions */}
      {data.welfare_contributions_total !== "0" && (
        <div className="pt-3 border-t border-border">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <Heart className="h-4 w-4 text-pink-500" />
              <span className="text-xs font-medium">Mfuko wa Kijamii</span>
            </div>
            <span className="text-sm font-semibold">
              {tzs(Number(data.welfare_contributions_total))}
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            Michango {data.welfare_contributions_count}
          </p>
        </div>
      )}

      {/* Loans summary */}
      {(data.outstanding_loans_count > 0 || data.closed_loans_count > 0) && (
        <div className="pt-3 border-t border-border">
          <h4 className="text-xs font-semibold mb-2">Mikopo Yangu</h4>
          {data.outstanding_loans_count > 0 && (
            <div className="flex items-center justify-between mb-1">
              <span className="text-xs text-muted-foreground">Deni bado:</span>
              <span className="text-sm font-semibold text-destructive">
                {tzs(Number(data.outstanding_loans_balance))}
              </span>
            </div>
          )}
          {data.closed_loans_count > 0 && (
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground">Imefungwa:</span>
              <span className="text-sm font-semibold text-green-600">
                {data.closed_loans_count}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// LEADERSHIP DASHBOARD CARD - "Salio la Kikundi"
// ============================================================================

export function GroupBalanceCard({ groupId }: { groupId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["group-dashboard", groupId],
    queryFn: () => dashboardApi.groupSummary(groupId),
    enabled: !!groupId,
  });

  if (isLoading) {
    return (
      <div className="card-surface animate-pulse">
        <div className="flex items-center justify-between mb-4">
          <div className="h-5 bg-muted rounded w-32" />
          <DollarSign className="h-6 w-6 text-muted" />
        </div>
        <div className="h-8 bg-muted rounded w-40 mb-3" />
        <div className="grid grid-cols-2 gap-3">
          <div className="h-10 bg-muted rounded" />
          <div className="h-10 bg-muted rounded" />
        </div>
      </div>
    );
  }

  if (error || !data) {
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
    <div className="card-surface">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-sm">Salio la Kikundi</h3>
        <DollarSign className="h-5 w-5 text-primary" />
      </div>

      {/* Main balance */}
      <div className="mb-4">
        <p className="text-3xl font-bold text-primary">
          {tzs(Number(data.available_balance))}
        </p>
        <p className="text-xs text-muted-foreground">
          = Michango + Malipo − Zimepewa
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-3">
        <div className="bg-muted/30 rounded p-2">
          <p className="text-xs text-muted-foreground mb-1">Michango</p>
          <p className="font-semibold text-sm">
            {tzs(Number(data.total_contributions))}
          </p>
        </div>
        <div className="bg-muted/30 rounded p-2">
          <p className="text-xs text-muted-foreground mb-1">Malipo</p>
          <p className="font-semibold text-sm">
            {tzs(Number(data.total_repayments))}
          </p>
        </div>
        <div className="bg-muted/30 rounded p-2">
          <p className="text-xs text-muted-foreground mb-1">Zimepewa</p>
          <p className="font-semibold text-sm text-destructive">
            -{tzs(Number(data.total_disbursed))}
          </p>
        </div>
        <div className="bg-muted/30 rounded p-2">
          <p className="text-xs text-muted-foreground mb-1">Wanachama</p>
          <p className="font-semibold text-sm">
            {data.total_active_members}
          </p>
        </div>
      </div>

      {/* Loans alert */}
      {data.outstanding_loans_count > 0 && (
        <div className="mt-3 pt-3 border-t border-border">
          <div className="flex items-center gap-2">
            <TrendingDown className="h-4 w-4 text-amber-600" />
            <div className="text-xs">
              <span className="font-medium text-amber-700">Deni bado:</span>
              <span className="ml-1 text-amber-700">
                {tzs(Number(data.outstanding_loans_balance))} ({data.outstanding_loans_count})
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Pending items alert */}
      {(data.pending_contributions_count > 0 || data.pending_loans_count > 0) && (
        <Alert className="mt-3 border-blue-200 bg-blue-50">
          <Clock className="h-4 w-4 text-blue-700" />
          <AlertDescription className="text-xs text-blue-700">
            {data.pending_contributions_count} michango + {data.pending_loans_count} mikopo inasub.
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

// ============================================================================
// MEMBER PERSONAL CARD (for leadership to view their own member stats)
// ============================================================================

export function PersonalMemberStatsCard({ memberId }: { memberId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["member-personal-stats", memberId],
    queryFn: () => dashboardApi.memberSummary(memberId),
    enabled: !!memberId,
  });

  if (isLoading) {
    return (
      <div className="card-surface animate-pulse">
        <div className="h-5 bg-muted rounded w-40 mb-4" />
        <div className="h-6 bg-muted rounded w-32 mb-2" />
        <div className="h-4 bg-muted rounded w-24" />
      </div>
    );
  }

  if (error || !data) return null;

  return (
    <div className="card-surface bg-blue-50 border border-blue-200">
      <h4 className="font-semibold text-sm mb-3">Michango Yangu Binafsi</h4>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">Akiba:</span>
          <span className="font-semibold">
            {tzs(Number(data.total_contributions))}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">Idadi:</span>
          <span className="font-semibold">{data.contributions_count}</span>
        </div>
        {data.pending_contributions_count > 0 && (
          <div className="flex items-center justify-between pt-1 border-t border-blue-200">
            <span className="text-xs text-blue-700">Inasub.:</span>
            <span className="font-semibold text-blue-700">
              {data.pending_contributions_count}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
