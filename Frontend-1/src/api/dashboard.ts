import { api } from "./client";
import type {
  DashboardSummary,
  MemberDashboardSummary,
  GroupDashboardSummary,
  KatibuDashboardSummary,
  HazinaDashboardSummary,
} from "./types";

export const dashboardApi = {
  // Legacy endpoint
  summary: () => api.get<DashboardSummary>("/dashboard"),

  // Role-scoped endpoints (Task A backend)
  memberSummary: (memberId: string) =>
    api.get<MemberDashboardSummary>(`/members/${memberId}/dashboard-summary`),

  groupSummary: (groupId: string) =>
    api.get<GroupDashboardSummary>(`/groups/${groupId}/dashboard-summary`),

  groupSummaryKatibu: (groupId: string) =>
    api.get<KatibuDashboardSummary>(`/groups/${groupId}/dashboard-summary/katibu`),

  groupSummaryHazina: (groupId: string) =>
    api.get<HazinaDashboardSummary>(`/groups/${groupId}/dashboard-summary/mweka-hazina`),

};
