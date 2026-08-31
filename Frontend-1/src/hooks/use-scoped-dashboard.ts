/**
 * Hooks for role-scoped dashboard endpoints (Task A backend).
 */

import { useQuery } from "@tanstack/react-query";
import { dashboardApi } from "@/api/dashboard";

export const roleScopedDashboardKeys = {
  memberSummary: (memberId: string) => ["member-dashboard-summary", memberId],
  groupSummary: (groupId: string) => ["group-dashboard-summary", groupId],
  groupSummaryKatibu: (groupId: string) => ["group-dashboard-katibu", groupId],
  groupSummaryHazina: (groupId: string) => ["group-dashboard-hazina", groupId],
};

/**
 * Fetch member personal dashboard (Akiba Yangu view)
 */
export function useMemberDashboardSummary(memberId?: string) {
  return useQuery({
    queryKey: roleScopedDashboardKeys.memberSummary(memberId || ""),
    queryFn: () => dashboardApi.memberSummary(memberId!),
    enabled: !!memberId,
  });
}

/**
 * Fetch group-wide dashboard (Uongozi view)
 */
export function useGroupDashboardSummary(groupId?: string) {
  return useQuery({
    queryKey: roleScopedDashboardKeys.groupSummary(groupId || ""),
    queryFn: () => dashboardApi.groupSummary(groupId!),
    enabled: !!groupId,
  });
}

/**
 * Fetch secretary-specific dashboard (Katibu view)
 */
export function useKatibuDashboardSummary(groupId?: string) {
  return useQuery({
    queryKey: roleScopedDashboardKeys.groupSummaryKatibu(groupId || ""),
    queryFn: () => dashboardApi.groupSummaryKatibu(groupId!),
    enabled: !!groupId,
  });
}

/**
 * Fetch treasurer-specific dashboard (Mweka Hazina view)
 */
export function useHazinaDashboardSummary(groupId?: string) {
  return useQuery({
    queryKey: roleScopedDashboardKeys.groupSummaryHazina(groupId || ""),
    queryFn: () => dashboardApi.groupSummaryHazina(groupId!),
    enabled: !!groupId,
  });
}
