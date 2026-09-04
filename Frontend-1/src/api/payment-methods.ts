import { api } from "./client";

// --- Types ---

export type PaymentMethodType = "lipa_namba" | "bank";

export interface PaymentMethod {
  id: string;
  group_id: string;
  type: PaymentMethodType;
  provider_name: string;
  account_number: string;
  account_name: string;
  is_active: boolean;
  status: string; // pending | approved (missing = approved, pre-workflow rows)
  approved_by?: string | null;
  approved_at?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface PaymentMethodInput {
  type: PaymentMethodType;
  provider_name: string;
  account_number: string;
  account_name: string;
  is_active?: boolean;
}

// --- API ---

export const paymentMethodsApi = {
  list: (groupId: string) =>
    api.get<{ data: PaymentMethod[]; total: number }>(
      `/groups/${groupId}/payment-methods`
    ),

  /** Mwenyekiti / Mweka Hazina only */
  create: (groupId: string, data: PaymentMethodInput) =>
    api.post<{ message: string; data: PaymentMethod }>(
      `/groups/${groupId}/payment-methods`,
      data
    ),

  /** Mwenyekiti / Mweka Hazina only — partial update incl. is_active toggle */
  update: (groupId: string, pmId: string, data: Partial<PaymentMethodInput>) =>
    api.patch<{ message: string; data: PaymentMethod }>(
      `/groups/${groupId}/payment-methods/${pmId}`,
      data
    ),

  /** Mwenyekiti / Mweka Hazina only */
  remove: (groupId: string, pmId: string) =>
    api.delete<{ message: string }>(
      `/groups/${groupId}/payment-methods/${pmId}`
    ),

  /** Mwenyekiti only — approve a treasurer-submitted (pending) method */
  approve: (groupId: string, pmId: string) =>
    api.post<{ message: string; data: PaymentMethod }>(
      `/groups/${groupId}/payment-methods/${pmId}/approve`
    ),
};
