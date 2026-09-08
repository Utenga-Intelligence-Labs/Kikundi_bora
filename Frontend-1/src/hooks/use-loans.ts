import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { loansApi } from "@/api/loans";
import type { ApplyLoanRequest } from "@/api/types";

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
  enabled?: boolean;
}) {
  const { enabled = true, ...queryParams } = params ?? {};
  return useQuery({
    queryKey: loanKeys.list(queryParams as Record<string, unknown>),
    queryFn: () => loansApi.list(queryParams),
    enabled,
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

export function useDisburseLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => loansApi.disburse(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

/** BUG-5: borrower confirms receiving the disbursed loan. */
export function useConfirmLoanReceived() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => loansApi.confirmReceived(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: loanKeys.all }),
  });
}

/** Sequential-chain queue (BUG-2): loans awaiting the caller's stage. */
export function usePendingApprovalLoans(myTurnOnly = false, enabled = true) {
  return useQuery({
    queryKey: [...loanKeys.all, "pending-approval", { myTurnOnly }],
    queryFn: () => loansApi.pendingApproval(myTurnOnly),
    enabled,
  });
}

/** Sign off at the caller's stage of the sequential chain. */
export function useChainApproveLoan() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      approved_amount,
    }: {
      id: string;
      approved_amount?: number;
    }) =>
      loansApi.chainApprove(
        id,
        approved_amount ? { approved_amount } : undefined,
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: loanKeys.all });
      qc.invalidateQueries({ queryKey: ["loans", "pending-approval"] });
    },
  });
}
