import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { pendingActionsApi, type PendingActionListParams } from "@/api/pending-actions";

export const pendingActionKeys = {
  all: ["pending-actions"] as const,
  list: (params?: Record<string, unknown>) =>
    [...pendingActionKeys.all, "list", params] as const,
  detail: (id: string) => [...pendingActionKeys.all, "detail", id] as const,
};

export function usePendingActions(params?: PendingActionListParams) {
  return useQuery({
    queryKey: pendingActionKeys.list(params as Record<string, unknown>),
    queryFn: () => pendingActionsApi.list(params),
  });
}

export function usePendingAction(id: string) {
  return useQuery({
    queryKey: pendingActionKeys.detail(id),
    queryFn: () => pendingActionsApi.get(id),
    enabled: !!id,
  });
}

export function useApprovePendingAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, remarks }: { id: string; remarks?: string }) =>
      pendingActionsApi.approve(id, remarks),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pendingActionKeys.all });
    },
  });
}

export function useRejectPendingAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, remarks }: { id: string; remarks?: string }) =>
      pendingActionsApi.reject(id, remarks),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pendingActionKeys.all });
    },
  });
}
