import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi, type AdminUserListParams } from "@/api/admin";
import type { AdminOverrideRequest, AdminResetPasswordRequest } from "@/api/types";

export const adminKeys = {
  all: ["admin"] as const,
  users: (params?: Record<string, unknown>) =>
    [...adminKeys.all, "users", params] as const,
  logs: (params?: Record<string, unknown>) =>
    [...adminKeys.all, "logs", params] as const,
  health: () => [...adminKeys.all, "health"] as const,
};

export function useAdminUsers(params?: AdminUserListParams) {
  return useQuery({
    queryKey: adminKeys.users(params as Record<string, unknown>),
    queryFn: () => adminApi.listUsers(params),
  });
}

export function useAdminLogs(params?: { page?: number; limit?: number; enabled?: boolean }) {
  const { enabled = true, ...queryParams } = params ?? {};
  return useQuery({
    queryKey: adminKeys.logs(queryParams as Record<string, unknown>),
    queryFn: () => adminApi.getLogs(queryParams),
    enabled,
  });
}

export function useSystemHealth() {
  return useQuery({
    queryKey: adminKeys.health(),
    queryFn: () => adminApi.getHealth(),
  });
}

export function useOverrideUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: AdminOverrideRequest }) =>
      adminApi.overrideUser(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: adminKeys.all });
    },
  });
}

export function useAdminResetPassword() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data?: AdminResetPasswordRequest }) =>
      adminApi.resetPassword(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: adminKeys.all });
    },
  });
}
