import { api } from "./client";
import type {
  Contribution,
  CreateContributionRequest,
  EditContributionRequest,
  PaginatedResponse,
  MonthlyReportResponse,
  MessageResponse,
} from "./types";

export const contributionsApi = {
  list: (params?: {
    page?: number;
    limit?: number;
    member_id?: string;
    month?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.member_id) q.member_id = String(params.member_id);
    if (params?.month) q.month = params.month;
    return api.get<PaginatedResponse<Contribution>>("/contributions", q);
  },
  create: (data: CreateContributionRequest) =>
    api.post<{ message: string; data: Contribution }>("/contributions", data),
  edit: (id: string, data: EditContributionRequest) =>
    api.put<{ message: string; data: Contribution }>(
      `/contributions/${id}`,
      data
    ),
  monthlyReport: (month?: string) =>
    api.get<MonthlyReportResponse>(
      "/contributions/monthly-report",
      month ? { month } : undefined
    ),
};
