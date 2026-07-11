import { api } from "./client";
import type {
  Repayment,
  RecordRepaymentRequest,
  RepaymentResponse,
  PaginatedResponse,
} from "./types";

export const repaymentsApi = {
  list: (params?: {
    page?: number;
    limit?: number;
    loan_id?: string;
    member_id?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.loan_id) q.loan_id = String(params.loan_id);
    if (params?.member_id) q.member_id = String(params.member_id);
    return api.get<PaginatedResponse<Repayment>>("/repayments", q);
  },
  record: (data: RecordRepaymentRequest) =>
    api.post<{ message: string; data: RepaymentResponse }>(
      "/repayments",
      data
    ),
};
