import { api } from "./client";
import type {
  DashboardSummary,
  MemberDashboardSummary,
  GroupDashboardSummary,
  KatibuDashboardSummary,
  HazinaDashboardSummary,
} from "./types";

export const dashboardApi = {
  // Legacy endpoint (returns the summary object directly, no envelope)
  summary: () => api.get<DashboardSummary>("/dashboard"),

  // Role-scoped endpoints (Task A backend) — the server wraps each summary
  // in {"data": ...}, so unwrap here; consumers read the summary fields directly.
  memberSummary: async (memberId: string) => {
    const res = await api.get<{ data: MemberDashboardSummary }>(
      `/members/${memberId}/dashboard-summary`
    );
    return res.data;
  },

  groupSummary: async (groupId: string) => {
    const res = await api.get<{ data: GroupDashboardSummary }>(
      `/groups/${groupId}/dashboard-summary`
    );
    return res.data;
  },

  groupSummaryKatibu: async (groupId: string) => {
    const res = await api.get<{ data: KatibuDashboardSummary }>(
      `/groups/${groupId}/dashboard-summary/katibu`
    );
    return res.data;
  },

  groupSummaryHazina: async (groupId: string) => {
    const res = await api.get<{ data: HazinaDashboardSummary }>(
      `/groups/${groupId}/dashboard-summary/mweka-hazina`
    );
    return res.data;
  },
};
