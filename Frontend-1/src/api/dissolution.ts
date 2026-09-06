import { api } from "./client";

export interface DissolutionProposal {
  id: string;
  group_id: string;
  proposed_by: string;
  cycle_span_years: number;
  period_start: string;
  period_end: string;
  status: string;
  voting_deadline: string;
  executed_at?: string | null;
  created_at: string;
}

export interface DissolutionTally {
  yes: number; no: number; total: number; approved: boolean;
}

export interface DissolutionPayout {
  id: string;
  proposal_id: string;
  member_id: string;
  total_contributed: string;
  total_owed: string;
  amount_owed: string;
  status: string;
  calculated_at: string;
  paid_at?: string | null;
  member?: { id: string; full_name: string; member_no: string };
}

export const dissolutionApi = {
  propose: (groupId: string, data: { cycle_span_years: number; voting_deadline: string }) =>
    api.post<{ message: string; data: DissolutionProposal }>(`/groups/${groupId}/dissolution-proposals`, data),
  listByGroup: (groupId: string) => api.get<{ data: DissolutionProposal[] }>(`/groups/${groupId}/dissolution-proposals`),
  get: (id: string) => api.get<{ data: DissolutionProposal; tally: DissolutionTally; my_vote: string | null }>(`/dissolution-proposals/${id}`),
  vote: (id: string, vote: string) => api.post<{ message: string; data: any }>(`/dissolution-proposals/${id}/vote`, { vote }),
  execute: (id: string) => api.post<{ message: string; data: any }>(`/dissolution-proposals/${id}/execute`),
  payouts: (id: string) => api.get<{ data: DissolutionPayout[] }>(`/dissolution-proposals/${id}/payouts`),
  markPaid: (id: string) => api.patch<{ message: string; data: any }>(`/dissolution-payouts/${id}/mark-paid`),
  myPayouts: () => api.get<{ data: DissolutionPayout[] }>(`/dissolution-payouts/me`),
};
