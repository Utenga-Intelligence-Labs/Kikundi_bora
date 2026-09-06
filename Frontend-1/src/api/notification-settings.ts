import { api } from "./client";

export interface NotificationSettings {
  sms_enabled: boolean;
  provider: string;
  provider_real: boolean;
  types: Record<string, boolean>;
}

export interface NotificationSettingsUpdate {
  sms_enabled?: boolean;
  types?: Record<string, boolean>;
}

export const notificationSettingsApi = {
  /** Mwenyekiti / admin only. */
  get: (groupId: string) =>
    api.get<{ data: NotificationSettings }>(
      `/groups/${groupId}/notification-settings`
    ),

  /** Mwenyekiti / admin only. */
  update: (groupId: string, data: NotificationSettingsUpdate) =>
    api.put<{ data: NotificationSettings }>(
      `/groups/${groupId}/notification-settings`,
      data
    ),
};
