import { api } from "./client";

export interface BackupHistory {
  id: string;
  filename: string;
  size_bytes: number;
  backup_type: string;
  email_sent_to: string;
  status: string;
  error_message?: string;
  created_by: string;
  created_at: string;
  creator?: { id: string; name: string };
}

export interface BackupSettings {
  id?: string;
  email: string;
  backup_type: string;
  frequency: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}

export const backupApi = {
  generate: (backupType?: string) =>
    api.post<{ message: string; data: BackupHistory }>("/admin/backup/generate", {
      backup_type: backupType || "database_only",
    }),

  getHistory: (params?: { page?: number; limit?: number }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    return api.get<PaginatedResponse<BackupHistory>>("/admin/backup/history", q);
  },

  getSettings: () => api.get<BackupSettings>("/admin/backup/settings"),

  saveSettings: (data: BackupSettings) =>
    api.post<{ message: string; data: BackupSettings }>("/admin/backup/settings", data),

  downloadUrl: (id: string) => {
    const base = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";
    return `${base.replace("/api/v1", "")}/api/v1/admin/backup/download/${id}`;
  },
};
