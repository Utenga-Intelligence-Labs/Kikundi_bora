import { api } from "./client";
import type {
  Loan,
  ApplyLoanRequest,
  ApproveLoanRequest,
  RejectLoanRequest,
  LoanWithRepayments,
  PaginatedResponse,
  OutstandingReportResponse,
  MessageResponse,
} from "./types";

// --- Loan Disbursement Portfolio (leadership) ---

export interface PortfolioLoan {
  id: string;
  member_id: string;
  member_no: string;
  full_name: string;
  principal: string;
  amount_repaid: string;
  outstanding: string;
  status: "OUTSTANDING" | "CLOSED";
  is_overdue: boolean;
  disbursed_at?: string;
  due_date: string;
}

export interface LoanPortfolioSummary {
  total_disbursed: string;
  total_repaid: string;
  total_outstanding: string;
  total_overdue: string;
  count_outstanding: number;
  count_closed: number;
  count_overdue: number;
  status_counts: Record<string, number>;
  loans: PortfolioLoan[];
}

export const loansApi = {
  portfolio: (params?: {
    status?: string;
    member_id?: string;
    from?: string;
    to?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.status) q.status = params.status;
    if (params?.member_id) q.member_id = params.member_id;
    if (params?.from) q.from = params.from;
    if (params?.to) q.to = params.to;
    const qs = new URLSearchParams(q).toString();
    return api.get<{ data: LoanPortfolioSummary }>(
      `/loans/portfolio${qs ? `?${qs}` : ""}`
    );
  },

  list: (params?: {
    page?: number;
    limit?: number;
    status?: string;
    member_id?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.status) q.status = params.status;
    if (params?.member_id) q.member_id = String(params.member_id);
    return api.get<PaginatedResponse<Loan>>("/loans", q);
  },
  get: (id: string) => api.get<LoanWithRepayments>(`/loans/${id}`),
  apply: (data: ApplyLoanRequest) =>
    api.post<{ message: string; data: Loan }>("/loans/apply", data),
  approve: (id: string, data: ApproveLoanRequest) =>
    api.post<{ message: string; data: Loan }>(`/loans/${id}/approve`, data),
  reject: (id: string, data: RejectLoanRequest) =>
    api.post<{ message: string; data: Loan }>(`/loans/${id}/reject`, data),
  disburse: (id: string) =>
    api.post<{ message: string; data: Loan }>(`/loans/${id}/disburse`),
  outstandingReport: () =>
    api.get<OutstandingReportResponse>("/loans/outstanding-report"),
};

// --- Loan offset (overdue debt paid from member savings) ---
// Three-role flow: chair proposes → secretary approves/rejects → treasurer executes.

export interface OffsetPreview {
  eligible: boolean;
  reason?: string;
  outstanding: string;
  gross_savings: string;
  offsets_applied: string;
  available_savings: string;
  offset_amount: string;
  existing_proposal?: LoanOffset | null;
}

export interface LoanOffset {
  id: string;
  loan_id: string;
  member_id: string;
  proposed_amount: string;
  amount: string;
  outstanding_before: string;
  savings_before: string;
  status: "PROPOSED" | "APPROVED" | "EXECUTED" | "REJECTED";
  proposed_by: string;
  approved_by?: string | null;
  executed_by?: string | null;
  proposed_at: string;
  approved_at?: string | null;
  executed_at?: string | null;
  reason?: string | null;
  member?: { id: string; full_name: string; member_no: string };
}

export const loanOffsetApi = {
  preview: (loanId: string) =>
    api.get<{ data: OffsetPreview }>(`/loans/${loanId}/offset-preview`),
  propose: (loanId: string, reason?: string) =>
    api.post<{ message: string; data: LoanOffset }>(`/loans/${loanId}/offset-propose`, reason ? { reason } : {}),
  list: (params?: { status?: string; loan_id?: string; member_id?: string }) => {
    const q: Record<string, string> = {};
    if (params?.status) q.status = params.status;
    if (params?.loan_id) q.loan_id = params.loan_id;
    if (params?.member_id) q.member_id = params.member_id;
    return api.get<{ data: LoanOffset[]; total: number }>(`/loan-offsets`, q);
  },
  approve: (id: string) =>
    api.post<{ message: string; data: LoanOffset }>(`/loan-offsets/${id}/approve`),
  reject: (id: string, reason?: string) =>
    api.post<{ message: string; data: LoanOffset }>(`/loan-offsets/${id}/reject`, reason ? { reason } : {}),
  execute: (id: string) =>
    api.post<{ message: string; data: LoanOffset }>(`/loan-offsets/${id}/execute`),
};
