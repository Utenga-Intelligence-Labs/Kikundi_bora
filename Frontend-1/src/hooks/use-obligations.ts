import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  obligationsApi,
  finesApi,
  offenceTypesApi,
  meetingsApi,
  type OffenceTypeInput,
} from "@/api/obligations";

export const obligationKeys = {
  member: (id: string) => ["obligations", "member", id] as const,
  group: (id: string) => ["obligations", "group", id] as const,
  queue: (id: string) => ["obligations", "queue", id] as const,
  fines: (params?: Record<string, string>) =>
    ["obligations", "fines", params] as const,
  offences: (groupId: string) =>
    ["obligations", "offences", groupId] as const,
  meetings: (groupId: string) =>
    ["obligations", "meetings", groupId] as const,
  attendance: (meetingId: string) =>
    ["obligations", "attendance", meetingId] as const,
};

export function useMemberObligations(memberId: string | null) {
  return useQuery({
    queryKey: obligationKeys.member(memberId ?? ""),
    queryFn: () => obligationsApi.memberSummary(memberId as string),
    enabled: !!memberId,
  });
}

export function useGroupObligations(groupId: string | null) {
  return useQuery({
    queryKey: obligationKeys.group(groupId ?? ""),
    queryFn: () => obligationsApi.groupSummary(groupId as string),
    enabled: !!groupId,
  });
}

export function useCollectionQueue(groupId: string | null) {
  return useQuery({
    queryKey: obligationKeys.queue(groupId ?? ""),
    queryFn: () => obligationsApi.collectionQueue(groupId as string),
    enabled: !!groupId,
  });
}

export function useFines(params?: { member_id?: string; status?: string }) {
  return useQuery({
    queryKey: obligationKeys.fines(params as Record<string, string> | undefined),
    queryFn: () => finesApi.list(params),
  });
}

export function useCollectFine() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => finesApi.collect(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useProposeWaiver() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      finesApi.proposeWaiver(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useDecideWaiver() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, approve }: { id: string; approve: boolean }) =>
      approve ? finesApi.approveWaiver(id) : finesApi.rejectWaiver(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useOffenceTypes(groupId: string | null) {
  return useQuery({
    queryKey: obligationKeys.offences(groupId ?? ""),
    queryFn: () => offenceTypesApi.list(groupId as string),
    enabled: !!groupId,
  });
}

export function useCreateOffenceType() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ groupId, data }: { groupId: string; data: OffenceTypeInput }) =>
      offenceTypesApi.create(groupId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useUpdateOffenceType() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      groupId,
      id,
      data,
    }: {
      groupId: string;
      id: string;
      data: Partial<OffenceTypeInput>;
    }) => offenceTypesApi.update(groupId, id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useDecideOffenceType() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      groupId,
      id,
      approve,
    }: {
      groupId: string;
      id: string;
      approve: boolean;
    }) =>
      approve
        ? offenceTypesApi.approve(groupId, id)
        : offenceTypesApi.deactivate(groupId, id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useMeetings(groupId: string | null) {
  return useQuery({
    queryKey: obligationKeys.meetings(groupId ?? ""),
    queryFn: () => meetingsApi.list(groupId as string),
    enabled: !!groupId,
  });
}

export function useCreateMeeting() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      groupId,
      data,
    }: {
      groupId: string;
      data: { title: string; meeting_date: string; notes?: string };
    }) => meetingsApi.create(groupId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}

export function useAttendance(meetingId: string | null) {
  return useQuery({
    queryKey: obligationKeys.attendance(meetingId ?? ""),
    queryFn: () => meetingsApi.attendance(meetingId as string),
    enabled: !!meetingId,
  });
}

export function useSetAttendance() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      meetingId,
      rows,
    }: {
      meetingId: string;
      rows: { member_id: string; status: string }[];
    }) => meetingsApi.setAttendance(meetingId, rows),
    onSuccess: (_d, v) => {
      qc.invalidateQueries({
        queryKey: obligationKeys.attendance(v.meetingId),
      });
    },
  });
}

export function useTriggerMeetingFines() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (meetingId: string) => meetingsApi.triggerFines(meetingId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["obligations"] });
    },
  });
}
