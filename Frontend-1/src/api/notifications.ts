import { api } from "./client";
import type {
  Notification,
  NotificationReadRequest,
  NotificationListResponse,
  MessageResponse,
} from "./types";

export const notificationsApi = {
  list: (params?: {
    page?: number;
    limit?: number;
    unread?: boolean;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.unread) q.unread = "true";
    return api.get<NotificationListResponse>("/notifications", q);
  },
  markRead: (data: NotificationReadRequest) =>
    api.post<MessageResponse>("/notifications/read", data),
};
