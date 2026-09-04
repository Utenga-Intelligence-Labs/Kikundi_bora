import { api } from "./client";

export interface ArrearsItem {
  cycle_label: string;
  due_date: string;
  expected_amount: string;
  paid_amount: string;
  owed: string;
}

export interface FineItem {
  id: string;
  offence_name: string;
  offence_kind: string;
  amount: string;
  occurrence_date: string;
  cycle_label: string;
  status: string;
  waiver_status: string;
}

export interface MemberObligations {
  member_id: string;
  member_no: string;
  full_name: string;
  total_arrears: string;
  current_cycle_due: string;
  current_cycle_label: string;
  total_fines_unpaid: string;
  grand_total_owed: string;
  itemized_arrears: ArrearsItem[];
  itemized_fines: FineItem[];
}

export interface MemberObligationRollup {
  member_id: string;
  member_no: string;
  full_name: string;
  total_arrears: string;
  current_cycle_due: string;
  total_fines_unpaid: string;
  grand_total_owed: string;
}

export interface GroupObligations {
  total_arrears_outstanding: string;
  total_fines_outstanding: string;
  member_count_owing: number;
  members: MemberObligationRollup[];
}

export interface FineOffenceType {
  id: string;
  group_id: string;
  kind: string;
  name: string;
  fine_type: string;
  fine_amount?: string | null;
  fine_percentage?: string | null;
  grace_period_days: number;
  status: string;
  created_by: string;
  approved_by?: string | null;
}

export interface OffenceTypeInput {
  kind: string;
  name: string;
  fine_type: string;
  fine_amount?: number;
  fine_percentage?: number;
  grace_period_days?: number;
}

export interface Fine {
  id: string;
  group_id: string;
  member_id: string;
  offence_type_id: string;
  contribution_cycle_label: string;
  occurrence_date: string;
  due_date: string;
  amount: string;
  reason: string;
  reason_note?: string | null;
  status: string;
  waiver_status: string;
  waiver_request_reason?: string | null;
  offence_type?: { id: string; name: string; kind: string } | null;
  member?: { id: string; full_name: string; member_no: string } | null;
}

export interface Meeting {
  id: string;
  group_id: string;
  title: string;
  meeting_date: string;
  notes?: string | null;
}

export interface AttendanceRow {
  id?: string;
  meeting_id?: string;
  member_id: string;
  status: string;
  fined?: boolean;
  member?: { id: string; full_name: string; member_no: string } | null;
}

export interface QueueEntry {
  member: MemberObligationRollup;
  arrears: ArrearsItem[];
  fines: FineItem[];
}

export const obligationsApi = {
  memberSummary: (memberId: string) =>
    api.get<{ data: MemberObligations }>(
      `/members/${memberId}/obligations/summary`
    ),
  groupSummary: (groupId: string) =>
    api.get<{ data: GroupObligations }>(
      `/groups/${groupId}/obligations/summary`
    ),
  collectionQueue: (groupId: string) =>
    api.get<{ data: QueueEntry[]; total: number }>(
      `/groups/${groupId}/collection-queue`
    ),
};

export const finesApi = {
  list: (params?: { member_id?: string; status?: string }) => {
    const q: Record<string, string> = {};
    if (params?.member_id) q.member_id = params.member_id;
    if (params?.status) q.status = params.status;
    return api.get<{ data: Fine[]; total: number }>("/fines", q);
  },
  collect: (id: string) =>
    api.post<{ message: string; data: Fine }>(`/fines/${id}/collect`),
  proposeWaiver: (id: string, reason: string) =>
    api.post<{ message: string; data: Fine }>(`/fines/${id}/waive-propose`, {
      reason,
    }),
  approveWaiver: (id: string) =>
    api.post<{ message: string; data: Fine }>(`/fines/${id}/waive-approve`),
  rejectWaiver: (id: string) =>
    api.post<{ message: string; data: Fine }>(`/fines/${id}/waive-reject`),
};

export const offenceTypesApi = {
  list: (groupId: string) =>
    api.get<{ data: FineOffenceType[]; total: number }>(
      `/groups/${groupId}/fine-offence-types`
    ),
  create: (groupId: string, data: OffenceTypeInput) =>
    api.post<{ message: string; data: FineOffenceType }>(
      `/groups/${groupId}/fine-offence-types`,
      data
    ),
  update: (groupId: string, id: string, data: Partial<OffenceTypeInput>) =>
    api.patch<{ message: string; data: FineOffenceType }>(
      `/groups/${groupId}/fine-offence-types/${id}`,
      data
    ),
  approve: (groupId: string, id: string) =>
    api.post<{ message: string; data: FineOffenceType }>(
      `/groups/${groupId}/fine-offence-types/${id}/approve`
    ),
  deactivate: (groupId: string, id: string) =>
    api.post<{ message: string; data: FineOffenceType }>(
      `/groups/${groupId}/fine-offence-types/${id}/deactivate`
    ),
};

export const meetingsApi = {
  list: (groupId: string) =>
    api.get<{ data: Meeting[]; total: number }>(
      `/groups/${groupId}/meetings`
    ),
  create: (groupId: string, data: { title: string; meeting_date: string; notes?: string }) =>
    api.post<{ message: string; data: Meeting }>(
      `/groups/${groupId}/meetings`,
      data
    ),
  attendance: (meetingId: string) =>
    api.get<{ data: AttendanceRow[]; total: number }>(
      `/meetings/${meetingId}/attendance`
    ),
  setAttendance: (meetingId: string, rows: { member_id: string; status: string }[]) =>
    api.put<{ message: string }>(`/meetings/${meetingId}/attendance`, rows),
  triggerFines: (meetingId: string) =>
    api.post<{ message: string; created: number }>(
      `/meetings/${meetingId}/trigger-fines`
    ),
};
