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
