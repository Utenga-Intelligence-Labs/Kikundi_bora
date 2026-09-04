import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ledgerApi } from "@/api/ledger";

export const ledgerKeys = {
  all: ["ledger"] as const,
  trialBalance: () => [...ledgerKeys.all, "trial-balance"] as const,
  balance: (account: string) =>
    [...ledgerKeys.all, "balance", account] as const,
  statement: (account: string, from?: string, to?: string) =>
    [...ledgerKeys.all, "statement", account, from, to] as const,
  transaction: (id: string) =>
    [...ledgerKeys.all, "transaction", id] as const,
};

export function useTrialBalance() {
  return useQuery({
    queryKey: ledgerKeys.trialBalance(),
    queryFn: () => ledgerApi.getTrialBalance(),
    retry: false,
  });
}

export function useLedgerBalance(account: string | null) {
  return useQuery({
    queryKey: ledgerKeys.balance(account ?? ""),
    queryFn: () => ledgerApi.getBalance(account as string),
    enabled: !!account,
    retry: false,
  });
}

export function useLedgerStatement(
  account: string | null,
  from?: string,
  to?: string
) {
  return useQuery({
    queryKey: ledgerKeys.statement(account ?? "", from, to),
    queryFn: () =>
      ledgerApi.getStatement(account as string, from, to),
    enabled: !!account,
    retry: false,
  });
}

export function useTransactionDetail(id: string | null) {
  return useQuery({
    queryKey: ledgerKeys.transaction(id ?? ""),
    queryFn: () => ledgerApi.getTransaction(id as string),
    enabled: !!id,
    retry: false,
  });
}

export function useOpenAccount() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ledgerApi.openAccount,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ledgerKeys.all });
    },
  });
}

export function useRecordTransaction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ledgerApi.recordTransaction,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ledgerKeys.all });
    },
  });
}

export function useReverseTransaction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      ledgerApi.reverseTransaction(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ledgerKeys.all });
    },
  });
}
