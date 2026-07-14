import { api } from "./client";
import type {
  CreateUserRequest,
  ApproveUserRequest,
  RejectUserRequest,
  User,
  MessageResponse,
  PaginatedResponse,
} from "./types";

export interface UserListParams {
  page?: number;
  limit?: number;
  status?: string;
  role?: string;
  q?: string;
}

export interface CreateUserResponse extends MessageResponse {
  data?: User;
  /** Plaintext temp password returned once to the creating chair — never stored client-side long-term */
  temp_password?: string;
}

export const userManagementApi = {
  create: (data: CreateUserRequest) =>
    api.post<CreateUserResponse>("/users/create", data),

  listPending: (params?: { page?: number; limit?: number }) => {
    const searchParams = new URLSearchParams();
    if (params?.page) searchParams.set("page", String(params.page));
    if (params?.limit) searchParams.set("limit", String(params.limit));
    const qs = searchParams.toString();
    return api.get<PaginatedResponse<User>>(`/users/pending${qs ? `?${qs}` : ""}`);
  },

  list: (params?: UserListParams) => {
    const searchParams = new URLSearchParams();
    if (params?.page) searchParams.set("page", String(params.page));
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.status) searchParams.set("status", params.status);
    if (params?.role) searchParams.set("role", params.role);
    if (params?.q) searchParams.set("q", params.q);
    const qs = searchParams.toString();
    return api.get<PaginatedResponse<User>>(`/users${qs ? `?${qs}` : ""}`);
  },

  approve: (id: string, data?: ApproveUserRequest) =>
    api.post<MessageResponse>(`/users/${id}/approve`, data ?? {}),

  reject: (id: string, data?: RejectUserRequest) =>
    api.post<MessageResponse>(`/users/${id}/reject`, data ?? {}),
};
