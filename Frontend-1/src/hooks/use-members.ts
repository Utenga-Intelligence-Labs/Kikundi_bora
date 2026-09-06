import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { membersApi } from "@/api/members";
import type { CreateMemberRequest, UpdateMemberRequest } from "@/api/types";

export const memberKeys = {
  all: ["members"] as const,
  list: (params?: Record<string, unknown>) =>
    [...memberKeys.all, "list", params] as const,
  detail: (id: string) => [...memberKeys.all, "detail", id] as const,
};

export function useMembers(params?: {
  page?: number;
  limit?: number;
  q?: string;
  user_id?: string;
}) {
  return useQuery({
    queryKey: memberKeys.list(params as Record<string, unknown>),
    queryFn: () => membersApi.list(params),
  });
}

export function useMember(id: string) {
  return useQuery({
    queryKey: memberKeys.detail(id),
    queryFn: () => membersApi.get(id),
    enabled: !!id,
  });
}

export function useCreateMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateMemberRequest) => membersApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.all }),
  });
}

export function useUpdateMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateMemberRequest }) =>
      membersApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.all }),
  });
}

export function useDeleteMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => membersApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.all }),
  });
}

/** Chair creates a login account for a member that has none; returns temp_password once */
export function useChairCreateLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => membersApi.createLogin(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.all }),
  });
}
