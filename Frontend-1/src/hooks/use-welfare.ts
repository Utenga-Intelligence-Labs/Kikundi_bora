import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { welfareApi } from "@/api/welfare";
import type {
  CreateWelfareEventRequest,
  ApproveWelfareEventRequest,
  RejectWelfareEventRequest,
  RecordWelfarePaymentRequest,
} from "@/api/welfare";

export const welfareKeys = {
  all: ["welfare"] as const,
  dashboard: () => [...welfareKeys.all, "dashboard"] as const,
  events: (params?: Record<string, unknown>) =>
    [...welfareKeys.all, "events", params] as const,
  eventDetail: (id: string) => [...welfareKeys.all, "event", id] as const,
  contributions: (params?: Record<string, unknown>) =>
    [...welfareKeys.all, "contributions", params] as const,
  myContributions: (params?: Record<string, unknown>) =>
    [...welfareKeys.all, "my-contributions", params] as const,
};

export function useWelfareDashboard() {
  return useQuery({
    queryKey: welfareKeys.dashboard(),
    queryFn: () => welfareApi.getDashboard(),
  });
}

export function useWelfareEvents(params?: {
  page?: number;
  limit?: number;
  status?: string;
  event_type?: string;
}) {
  return useQuery({
    queryKey: welfareKeys.events(params as Record<string, unknown>),
    queryFn: () => welfareApi.listEvents(params),
  });
}

export function useWelfareEvent(id: string) {
  return useQuery({
    queryKey: welfareKeys.eventDetail(id),
    queryFn: () => welfareApi.getEvent(id),
    enabled: !!id,
  });
}

export function useCreateWelfareEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateWelfareEventRequest) =>
      welfareApi.createEvent(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: welfareKeys.all }),
  });
}

export function useApproveWelfareEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: ApproveWelfareEventRequest;
    }) => welfareApi.approveEvent(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: welfareKeys.all }),
  });
}

export function useRejectWelfareEvent() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: RejectWelfareEventRequest;
    }) => welfareApi.rejectEvent(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: welfareKeys.all }),
  });
}

export function useWelfareContributions(params?: {
  page?: number;
  limit?: number;
  status?: string;
  event_id?: string;
}) {
  return useQuery({
    queryKey: welfareKeys.contributions(params as Record<string, unknown>),
    queryFn: () => welfareApi.listContributions(params),
  });
}

export function useMyWelfareContributions(params?: {
  page?: number;
  limit?: number;
  status?: string;
}) {
  return useQuery({
    queryKey: welfareKeys.myContributions(params as Record<string, unknown>),
    queryFn: () => welfareApi.myContributions(params),
  });
}

export function useRecordWelfarePayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      eventId,
      memberId,
      data,
    }: {
      eventId: string;
      memberId: string;
      data: RecordWelfarePaymentRequest;
    }) => welfareApi.recordPayment(eventId, memberId, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: welfareKeys.all }),
  });
}

export function useWaiveWelfareContribution() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      eventId,
      memberId,
    }: {
      eventId: string;
      memberId: string;
    }) => welfareApi.waiveContribution(eventId, memberId),
    onSuccess: () => qc.invalidateQueries({ queryKey: welfareKeys.all }),
  });
}
