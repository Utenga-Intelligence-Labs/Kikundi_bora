import { api } from "./client";
import type { PaginatedResponse, MessageResponse, Loan } from "./types";

// --- Types ---

export interface LoanCommitteeMember {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  user_role: string;
  appointed_by?: string;
  appointed_at: string;
  is_active: boolean;
}

export interface LoanReview {
  id: string;
  loan_id: string;
  reviewer_id: string;
  reviewer_name: string;
  decision: "PENDING" | "APPROVE" | "REJECT";
  comments?: string;
  reviewed_at?: string;
}

export interface LoanCommitteeDashboard {
  pending_reviews: number;
  loans_under_review: number;
  approved_loans: number;
  rejected_loans: number;
  my_reviews: number;
  committee_members: number;
}

export interface LoanCommitteeHistoryRow {
  loan_id: string;
  applicant_name: string;
  member_no: string;
  amount: number;
  status: string;
  reviewed_by: string;
  decision: string;
  comments?: string;
  reviewed_at: string;
}

export interface LoanDetailResponse {
  data: Loan;
  reviews: LoanReview[];
  contributions: unknown[];
  previous_loans: Loan[];
  outstanding_balance: number;
}

export interface CommitteeActivityReport {
  total_reviews: number;
  approval_rate: number;
  rejection_rate: number;
  reviews_by_member: {
    user_id: string;
    user_name: string;
    reviews: number;
    approvals: number;
    rejections: number;
  }[];
  committee_composition: {
    user_id: string;
    user_name: string;
    role: string;
    appointed_at: string;
    type: "automatic" | "appointed";
  }[];
  review_history: LoanCommitteeHistoryRow[];
}

// --- API ---

export const loanCommitteeApi = {
  // Check if current user is a committee member
  check: () => api.get<{ is_committee_member: boolean }>("/loan-committee/check"),

  // Members
  listMembers: () =>
    api.get<{ data: LoanCommitteeMember[] }>("/loan-committee/members"),

  appointMember: (userId: string) =>
    api.post<MessageResponse>("/loan-committee/members", { user_id: userId }),

  removeMember: (id: string) =>
    api.delete<MessageResponse>(`/loan-committee/members/${id}`),

  // Loans
  listLoans: (params?: { page?: number; limit?: number; status?: string }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.status) q.status = params.status;
    return api.get<PaginatedResponse<Loan>>("/loan-committee/loans", q);
  },

  getLoan: (id: string) =>
    api.get<LoanDetailResponse>(`/loan-committee/loans/${id}`),

  submitReview: (loanId: string, data: { decision: "APPROVE" | "REJECT"; comments?: string }) =>
    api.post<MessageResponse & { data: Loan }>(`/loan-committee/loans/${loanId}/review`, data),

  // Dashboard
  getDashboard: () =>
    api.get<{ data: LoanCommitteeDashboard }>("/loan-committee/dashboard"),

  // History
  getHistory: (params?: { page?: number; limit?: number }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    return api.get<PaginatedResponse<LoanCommitteeHistoryRow>>("/loan-committee/history", q);
  },

  // Report
  getReport: () =>
    api.get<{ data: CommitteeActivityReport }>("/loan-committee/report"),

  // Pending count for badges
  getPendingCount: () =>
    api.get<{ count: number }>("/loan-committee/pending-count"),
};
