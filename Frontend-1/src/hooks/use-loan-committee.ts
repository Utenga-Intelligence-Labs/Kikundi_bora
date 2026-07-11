import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { loanCommitteeApi } from "@/api/loan-committee";
import type { LoanCommitteeMember } from "@/api/loan-committee";

export const committeeKeys = {
  all: ["loan-committee"] as const,
  check: () => [...committeeKeys.all, "check"] as const,
  members: () => [...committeeKeys.all, "members"] as const,
  loans: (params?: Record<string, unknown>) =>
    [...committeeKeys.all, "loans", params] as const,
  loanDetail: (id: string) => [...committeeKeys.all, "loan", id] as const,
  dashboard: () => [...committeeKeys.all, "dashboard"] as const,
  history: (params?: Record<string, unknown>) =>
    [...committeeKeys.all, "history", params] as const,
  report: () => [...committeeKeys.all, "report"] as const,
  pendingCount: () => [...committeeKeys.all, "pending-count"] as const,
};

export function useIsCommitteeMember() {
  return useQuery({
    queryKey: committeeKeys.check(),
    queryFn: () => loanCommitteeApi.check(),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useCommitteeMembers() {
  return useQuery({
    queryKey: committeeKeys.members(),
    queryFn: () => loanCommitteeApi.listMembers(),
  });
}

export function useAppointCommitteeMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => loanCommitteeApi.appointMember(userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: committeeKeys.members() });
      qc.invalidateQueries({ queryKey: committeeKeys.dashboard() });
    },
  });
}

export function useRemoveCommitteeMember() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => loanCommitteeApi.removeMember(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: committeeKeys.members() });
      qc.invalidateQueries({ queryKey: committeeKeys.dashboard() });
    },
  });
}

export function useCommitteeLoans(params?: {
  page?: number;
  limit?: number;
  status?: string;
}) {
  return useQuery({
    queryKey: committeeKeys.loans(params as Record<string, unknown>),
    queryFn: () => loanCommitteeApi.listLoans(params),
  });
}

export function useCommitteeLoanDetail(id: string) {
  return useQuery({
    queryKey: committeeKeys.loanDetail(id),
    queryFn: () => loanCommitteeApi.getLoan(id),
    enabled: !!id,
  });
}

export function useSubmitLoanReview() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      loanId,
      data,
    }: {
      loanId: string;
      data: { decision: "APPROVE" | "REJECT"; comments?: string };
    }) => loanCommitteeApi.submitReview(loanId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: committeeKeys.all });
    },
  });
}

export function useCommitteeDashboard() {
  return useQuery({
    queryKey: committeeKeys.dashboard(),
    queryFn: () => loanCommitteeApi.getDashboard(),
  });
}

export function useCommitteeHistory(params?: {
  page?: number;
  limit?: number;
}) {
  return useQuery({
    queryKey: committeeKeys.history(params as Record<string, unknown>),
    queryFn: () => loanCommitteeApi.getHistory(params),
  });
}

export function useCommitteeReport() {
  return useQuery({
    queryKey: committeeKeys.report(),
    queryFn: () => loanCommitteeApi.getReport(),
  });
}

export function usePendingLoansCount() {
  return useQuery({
    queryKey: committeeKeys.pendingCount(),
    queryFn: () => loanCommitteeApi.getPendingCount(),
    staleTime: 30 * 1000, // 30 seconds
  });
}
