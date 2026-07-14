import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { contributionsApi } from "@/api/contributions";
import type { CreateContributionRequest, EditContributionRequest } from "@/api/types";

export const contributionKeys = {
  all: ["contributions"] as const,
  list: (params?: Record<string, unknown>) =>
    [...contributionKeys.all, "list", params] as const,
  monthlyReport: (month?: string) =>
    [...contributionKeys.all, "monthly-report", month] as const,
};

export function useContributions(params?: {
  page?: number;
  limit?: number;
  member_id?: string;
  month?: string;
  enabled?: boolean;
}) {
  const { enabled = true, ...queryParams } = params ?? {};
  return useQuery({
    queryKey: contributionKeys.list(queryParams as Record<string, unknown>),
    queryFn: () => contributionsApi.list(queryParams),
    enabled,
  });
}

export function useMonthlyReport(month?: string) {
  return useQuery({
    queryKey: contributionKeys.monthlyReport(month),
    queryFn: () => contributionsApi.monthlyReport(month),
  });
}

export function useCreateContribution() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateContributionRequest) =>
      contributionsApi.create(data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: contributionKeys.all }),
  });
}

export function useEditContribution() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: EditContributionRequest }) =>
      contributionsApi.edit(id, data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: contributionKeys.all }),
  });
}
