import { api } from "./client";
import type {
  AdminOverrideRequest,
  AdminResetPasswordRequest,
  AdminLog,
  User,
  MessageResponse,
  PaginatedResponse,
  SystemHealth,
} from "./types";

export interface AdminUserListParams {
  page?: number;
  limit?: number;
  status?: string;
  role?: string;
  q?: string;
}

export const adminApi = {
  listUsers: (params?: AdminUserListParams) => {
    const searchParams = new URLSearchParams();
    if (params?.page) searchParams.set("page", String(params.page));
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.status) searchParams.set("status", params.status);
    if (params?.role) searchParams.set("role", params.role);
    if (params?.q) searchParams.set("q", params.q);
    const qs = searchParams.toString();
    return api.get<PaginatedResponse<User>>(`/admin/users${qs ? `?${qs}` : ""}`);
  },

  getLogs: (params?: { page?: number; limit?: number }) => {
    const searchParams = new URLSearchParams();
    if (params?.page) searchParams.set("page", String(params.page));
    if (params?.limit) searchParams.set("limit", String(params.limit));
    const qs = searchParams.toString();
    return api.get<PaginatedResponse<AdminLog>>(`/admin/logs${qs ? `?${qs}` : ""}`);
  },

  overrideUser: (id: string, data: AdminOverrideRequest) =>
    api.post<MessageResponse>(`/admin/users/${id}/override`, data),

  resetPassword: (id: string, data?: AdminResetPasswordRequest) =>
    api.post<MessageResponse & { temp_password?: string }>(`/admin/users/${id}/reset-password`, data ?? {}),

  getHealth: () => api.get<SystemHealth>("/admin/health"),
};
