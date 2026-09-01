import { api } from "./client";

// --- Types ---

export type SocialFundStatus =
  | "PENDING_APPROVAL"
  | "ACTIVE"
  | "REJECTED"
  | "CLOSED";

export type SocialFundContributionStatus = "PENDING" | "CONFIRMED" | "REJECTED";

export interface SocialFund {
  id: string;
  group_id: string;
  name: string;
  description: string;
  target_amount?: string | null;
  current_balance: string;
  status: SocialFundStatus;
  created_by: string;
  approved_by?: string | null;
  approved_at?: string | null;
  rejection_reason?: string | null;
  closed_at?: string | null;
  created_at: string;
  creator?: { id: string; name: string; role: string };
}

export interface SocialFundContribution {
  id: string;
  fund_id: string;
  member_id: string;
  amount: string;
  status: SocialFundContributionStatus;
  contributed_at?: string | null;
  rejection_reason?: string | null;
  created_at: string;
  member?: Pick<
    import("./types").Member,
    "id" | "member_no" | "full_name"
  >;
}

export interface SocialFundDetail {
  data: SocialFund;
  contributions: SocialFundContribution[];
  stats: {
    confirmed_count: number;
    pending_count: number;
    rejected_count: number;
  };
}

// --- API ---

export const socialFundsApi = {
  list: (groupId: string) =>
    api.get<{ data: SocialFund[]; total: number }>(
      `/groups/${groupId}/social-funds`
    ),

  /** Mwenyekiti only */
  create: (
    groupId: string,
    data: { name: string; description: string; target_amount?: number }
  ) =>
    api.post<{ message: string; data: SocialFund }>(
      `/groups/${groupId}/social-funds`,
      data
    ),

  /** Katibu only */
  approve: (groupId: string, fundId: string) =>
    api.post<{ message: string; data: SocialFund }>(
      `/groups/${groupId}/social-funds/${fundId}/approve`
    ),

  /** Katibu only */
  rejectFund: (groupId: string, fundId: string, reason: string) =>
    api.post<{ message: string; data: SocialFund }>(
      `/groups/${groupId}/social-funds/${fundId}/reject`,
      { reason }
    ),

  /** Mwenyekiti only */
  close: (groupId: string, fundId: string) =>
    api.post<{ message: string; data: SocialFund }>(
      `/groups/${groupId}/social-funds/${fundId}/close`
    ),

  detail: (groupId: string, fundId: string) =>
    api.get<SocialFundDetail>(`/groups/${groupId}/social-funds/${fundId}`),

  /** Any member with a member row — declares a contribution (PENDING) */
  contribute: (groupId: string, fundId: string, amount: number) =>
    api.post<{ message: string; data: SocialFundContribution }>(
      `/groups/${groupId}/social-funds/${fundId}/contribute`,
      { amount }
    ),

  /** Mweka Hazina only */
  confirm: (groupId: string, fundId: string, cid: string) =>
    api.post<{ message: string; data: SocialFundContribution }>(
      `/groups/${groupId}/social-funds/${fundId}/contributions/${cid}/confirm`
    ),

  /** Mweka Hazina only */
  rejectContribution: (
    groupId: string,
    fundId: string,
    cid: string,
    reason: string
  ) =>
    api.post<{ message: string; data: SocialFundContribution }>(
      `/groups/${groupId}/social-funds/${fundId}/contributions/${cid}/reject`,
      { reason }
    ),
};
