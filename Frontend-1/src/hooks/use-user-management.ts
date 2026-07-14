import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userManagementApi, type UserListParams } from "@/api/user-management";
import type { CreateUserRequest, ApproveUserRequest, RejectUserRequest } from "@/api/types";

export const userMgmtKeys = {
  all: ["user-management"] as const,
  pending: (params?: Record<string, unknown>) =>
    [...userMgmtKeys.all, "pending", params] as const,
  list: (params?: Record<string, unknown>) =>
    [...userMgmtKeys.all, "list", params] as const,
};

export function usePendingUsers(params?: { page?: number; limit?: number }) {
  return useQuery({
    queryKey: userMgmtKeys.pending(params as Record<string, unknown>),
    queryFn: () => userManagementApi.listPending(params),
  });
}

export function useUserList(params?: UserListParams) {
  return useQuery({
    queryKey: userMgmtKeys.list(params as Record<string, unknown>),
    queryFn: () => userManagementApi.list(params),
  });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateUserRequest) => userManagementApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: userMgmtKeys.all });
    },
  });
}

export function useApproveUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data?: ApproveUserRequest }) =>
      userManagementApi.approve(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: userMgmtKeys.all });
    },
  });
}

export function useRejectUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data?: RejectUserRequest }) =>
      userManagementApi.reject(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: userMgmtKeys.all });
    },
  });
}

/** Chair resets a member/user password; response may include temp_password once */
export function useChairResetPassword() {
  return useMutation({
    mutationFn: (id: string) => userManagementApi.resetPassword(id),
  });
}
