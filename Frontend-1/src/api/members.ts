import { api } from "./client";
import type {
  Member,
  CreateMemberRequest,
  UpdateMemberRequest,
  PaginatedResponse,
  MessageResponse,
} from "./types";

export const membersApi = {
  list: (params?: { page?: number; limit?: number; q?: string; user_id?: string }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.q) q.q = params.q;
    if (params?.user_id) q.user_id = params.user_id;
    return api.get<PaginatedResponse<Member>>("/members", q);
  },
  get: (id: string) => api.get<Member>(`/members/${id}`),
  create: (data: CreateMemberRequest) =>
    api.post<{ message: string; data: Member }>("/members", data),
  update: (id: string, data: UpdateMemberRequest) =>
    api.put<{ message: string; data: Member }>(`/members/${id}`, data),
  delete: (id: string) =>
    api.delete<MessageResponse>(`/members/${id}`),
};
