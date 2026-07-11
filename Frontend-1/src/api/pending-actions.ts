import { api } from "./client";
import type { PendingAction, PaginatedResponse, MessageResponse } from "./types";

export interface PendingActionListParams {
  page?: number;
  limit?: number;
  status?: string;
  type?: string;
}

export const pendingActionsApi = {
  list: (params?: PendingActionListParams) => {
    const searchParams = new URLSearchParams();
    if (params?.page) searchParams.set("page", String(params.page));
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.status) searchParams.set("status", params.status);
    if (params?.type) searchParams.set("type", params.type);
    const qs = searchParams.toString();
    return api.get<PaginatedResponse<PendingAction>>(`/pending-actions${qs ? `?${qs}` : ""}`);
  },

  get: (id: string) => api.get<PendingAction>(`/pending-actions/${id}`),

  approve: (id: string, remarks?: string) =>
    api.post<MessageResponse>(`/pending-actions/${id}/approve`, { remarks: remarks ?? "" }),

  reject: (id: string, remarks?: string) =>
    api.post<MessageResponse>(`/pending-actions/${id}/reject`, { remarks: remarks ?? "" }),
};
