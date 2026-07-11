import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { backupApi, type BackupSettings } from "@/api/backup";

export const backupKeys = {
  all: ["backup"] as const,
  history: (params?: Record<string, unknown>) =>
    [...backupKeys.all, "history", params] as const,
  settings: () => [...backupKeys.all, "settings"] as const,
};

export function useBackupHistory(params?: { page?: number; limit?: number }) {
  return useQuery({
    queryKey: backupKeys.history(params as Record<string, unknown>),
    queryFn: () => backupApi.getHistory(params),
  });
}

export function useBackupSettings() {
  return useQuery({
    queryKey: backupKeys.settings(),
    queryFn: () => backupApi.getSettings(),
  });
}

export function useGenerateBackup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (backupType?: string) => backupApi.generate(backupType),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.all }),
  });
}

export function useSaveBackupSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: BackupSettings) => backupApi.saveSettings(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: backupKeys.settings() }),
  });
}
