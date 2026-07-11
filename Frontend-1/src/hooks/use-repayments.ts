import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { repaymentsApi } from "@/api/repayments";
import type { RecordRepaymentRequest } from "@/api/types";

export const repaymentKeys = {
  all: ["repayments"] as const,
  list: (params?: Record<string, unknown>) =>
    [...repaymentKeys.all, "list", params] as const,
};

export function useRepayments(params?: {
  page?: number;
  limit?: number;
  loan_id?: string;
  member_id?: string;
}) {
  return useQuery({
    queryKey: repaymentKeys.list(params as Record<string, unknown>),
    queryFn: () => repaymentsApi.list(params),
  });
}

export function useRecordRepayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: RecordRepaymentRequest) =>
      repaymentsApi.record(data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: repaymentKeys.all }),
  });
}
