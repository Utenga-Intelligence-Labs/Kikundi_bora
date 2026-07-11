import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "@/api/notifications";
import type { NotificationReadRequest } from "@/api/types";

export const notificationKeys = {
  all: ["notifications"] as const,
  list: (params?: Record<string, unknown>) =>
    [...notificationKeys.all, "list", params] as const,
};

export function useNotifications(params?: {
  page?: number;
  limit?: number;
  unread?: boolean;
}) {
  return useQuery({
    queryKey: notificationKeys.list(params as Record<string, unknown>),
    queryFn: () => notificationsApi.list(params),
  });
}

export function useMarkNotificationsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: NotificationReadRequest) =>
      notificationsApi.markRead(data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: notificationKeys.all }),
  });
}
