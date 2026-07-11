import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { loansApi } from "@/api/loans";
import type {
  ApplyLoanRequest,
  ApproveLoanRequest,
  RejectLoanRequest,
} from "@/api/types";

export const loanKeys = {
  all: ["loans"] as const,
  list: (params?: Record<string, unknown>) =>
    [...loanKeys.all, "list", params] as const,
  detail: (id: string) => [...loanKeys.all, "detail", id] as const,
  outstanding: () => [...loanKeys.all, "outstanding"] as const,
};

export function useLoans(params?: {
  page?: number;
  limit?: number;
  status?: string;
  member_id?: string;
}) {
  return useQuery({
    queryKey: loanKeys.list(params as Record<string, unknown>),
    queryFn: () => loansApi.list(params),
  });
}

export function useLoan(id: string) {
  return useQuery({
    queryKey: loanKeys.detail(id),
    queryFn: () => loansApi.get(id),
    enabled: !!id,
  });
}

export function useOutstandingReport() {
  return useQuery({
    queryKey: loanKeys.outstanding(),
    queryFn: () => loansApi.outstandingReport(),
  });
}

export function useApplyLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ApplyLoanRequest) => loansApi.apply(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useApproveLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ApproveLoanRequest }) =>
      loansApi.approve(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useRejectLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: RejectLoanRequest }) =>
      loansApi.reject(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

export function useDisburseLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => loansApi.disburse(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}
