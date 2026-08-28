import { api } from "./client";

// --- Types ---

export type ContributionInterval = "weekly" | "monthly" | "semi_annual" | "yearly";

export interface GroupInfo {
  id: string;
  name: string;
  contribution_interval: ContributionInterval;
  contribution_due_date?: string | null;
  fixed_contribution_amount?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface GroupSettingProposal {
  id: string;
  group_id: string;
  contribution_interval: ContributionInterval;
  contribution_due_date?: string | null;
  fixed_contribution_amount?: string | null;
  status: "PENDING" | "APPROVED" | "REJECTED";
  proposed_by: string;
  approved_by?: string | null;
  rejection_reason?: string | null;
  created_at: string;
  reviewed_at?: string | null;
  proposer?: { id: string; name: string; role: string };
}

export interface GroupSettingsResponse {
  data: GroupInfo;
  pending_proposal: GroupSettingProposal | null;
  next_due_date: string | null;
}

export interface ProposeSettingsRequest {
  contribution_interval: ContributionInterval;
  contribution_due_date: string;
  fixed_contribution_amount?: number;
}

export const INTERVAL_LABELS: Record<ContributionInterval, string> = {
  weekly: "Mara kwa wiki",
  monthly: "Mara kwa mwezi",
  semi_annual: "Mara mbili kwa mwaka",
  yearly: "Mara moja kwa mwaka",
};

// --- API ---

export const groupsApi = {
  /** Single-group deployment: resolves the group + settings + pending proposal. */
  current: () => api.get<GroupSettingsResponse>("/groups/current"),

  getSettings: (groupId: string) =>
    api.get<GroupSettingsResponse>(`/groups/${groupId}/contribution-settings`),

  /** Mwenyekiti (chair) only. */
  propose: (groupId: string, data: ProposeSettingsRequest) =>
    api.post<{ message: string; data: GroupSettingProposal }>(
      `/groups/${groupId}/contribution-settings/propose`,
      data
    ),

  /** Katibu (secretary) only — applies the pending proposal. */
  approve: (groupId: string) =>
    api.post<{ message: string; data: GroupInfo }>(
      `/groups/${groupId}/contribution-settings/approve`
    ),

  /** Katibu (secretary) only. */
  reject: (groupId: string, reason: string) =>
    api.post<{ message: string; data: GroupSettingProposal }>(
      `/groups/${groupId}/contribution-settings/reject`,
      { reason }
    ),
};
