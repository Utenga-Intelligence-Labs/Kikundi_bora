import { api } from "./client";
import type { PaginatedResponse, MessageResponse, Member } from "./types";

// --- Types ---

export type WelfareEventType =
  | "MSIBA"
  | "HARUSI"
  | "DHARURA"
  | "MATIBABU"
  | "KUZALIWA"
  | "ELIMU";

export type WelfareFundingSource =
  | "TREASURY"
  | "MEMBER_CONTRIBUTION"
  | "BOTH";

export type WelfareEventStatus =
  | "PENDING"
  | "APPROVED"
  | "REJECTED"
  | "COMPLETED";

export type WelfareContributionStatus =
  | "PENDING"
  | "PAID"
  | "WAIVED";

export interface WelfareEvent {
  id: string;
  member_id: string;
  event_type: WelfareEventType;
  description: string;
  amount_requested: number;
  amount_approved?: number;
  funding_source: WelfareFundingSource;
  treasury_amount: number;
  member_amount: number;
  status: WelfareEventStatus;
  created_by: string;
  approved_by?: string;
  rejected_by?: string;
  rejection_reason?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  member?: Pick<Member, "id" | "member_no" | "full_name" | "phone">;
  creator?: { id: string; name: string; role: string };
  approver?: { id: string; name: string; role: string };
}

export interface WelfareContribution {
  id: string;
  event_id: string;
  member_id: string;
  amount: number;
  status: WelfareContributionStatus;
  paid_at?: string;
  recorded_by?: string;
  created_at: string;
  updated_at: string;
  event?: Pick<WelfareEvent, "id" | "event_type" | "description" | "status">;
  member?: Pick<Member, "id" | "member_no" | "full_name" | "phone">;
  recorder?: { id: string; name: string; role: string };
}

export interface WelfareDashboard {
  total_events: number;
  pending_approval: number;
  active_events: number;
  completed_events: number;
  rejected_events: number;
  total_collected: number;
  total_from_treasury: number;
  my_pending_contributions: number;
  my_paid_contributions: number;
}

export interface CreateWelfareEventRequest {
  member_id: string;
  event_type: WelfareEventType;
  description: string;
  amount_requested: number;
  funding_source: WelfareFundingSource;
  treasury_amount: number;
  member_amount: number;
}

export interface ApproveWelfareEventRequest {
  approved_amount: number;
}

export interface RejectWelfareEventRequest {
  reason: string;
}

export interface RecordWelfarePaymentRequest {
  amount: number;
}

export interface WelfareEventDetail {
  data: WelfareEvent;
  contributions: WelfareContribution[];
  stats: {
    total_paid: number;
    total_pending: number;
    paid_count: number;
    pending_count: number;
  };
}

// --- API ---

export const welfareApi = {
  // Dashboard
  getDashboard: () =>
    api.get<{ data: WelfareDashboard }>("/welfare/dashboard"),

  // Events
  listEvents: (params?: {
    page?: number;
    limit?: number;
    status?: string;
    event_type?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.status) q.status = params.status;
    if (params?.event_type) q.event_type = params.event_type;
    return api.get<PaginatedResponse<WelfareEvent>>("/welfare/events", q);
  },

  getEvent: (id: string) =>
    api.get<WelfareEventDetail>(`/welfare/events/${id}`),

  createEvent: (data: CreateWelfareEventRequest) =>
    api.post<{ message: string; data: WelfareEvent }>("/welfare/events", data),

  approveEvent: (id: string, data: ApproveWelfareEventRequest) =>
    api.post<{ message: string; data: WelfareEvent }>(
      `/welfare/events/${id}/approve`,
      data
    ),

  rejectEvent: (id: string, data: RejectWelfareEventRequest) =>
    api.post<{ message: string; data: WelfareEvent }>(
      `/welfare/events/${id}/reject`,
      data
    ),

  // Contributions
  listContributions: (params?: {
    page?: number;
    limit?: number;
    status?: string;
    event_id?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.status) q.status = params.status;
    if (params?.event_id) q.event_id = params.event_id;
    return api.get<PaginatedResponse<WelfareContribution>>(
      "/welfare/contributions",
      q
    );
  },

  myContributions: (params?: {
    page?: number;
    limit?: number;
    status?: string;
  }) => {
    const q: Record<string, string> = {};
    if (params?.page) q.page = String(params.page);
    if (params?.limit) q.limit = String(params.limit);
    if (params?.status) q.status = params.status;
    return api.get<PaginatedResponse<WelfareContribution>>(
      "/welfare/my-contributions",
      q
    );
  },

  recordPayment: (
    eventId: string,
    memberId: string,
    data: RecordWelfarePaymentRequest
  ) =>
    api.post<MessageResponse>(
      `/welfare/events/${eventId}/contributions/${memberId}/pay`,
      data
    ),

  waiveContribution: (eventId: string, memberId: string) =>
    api.post<MessageResponse>(
      `/welfare/events/${eventId}/contributions/${memberId}/waive`
    ),
};
