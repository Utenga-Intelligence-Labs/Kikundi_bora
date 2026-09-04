import { api } from "./client";

export type LedgerDirection = "debit" | "credit";
export type LedgerAccountType =
  | "asset"
  | "liability"
  | "income"
  | "expense"
  | "equity";

export interface LedgerEntryInput {
  account_name: string;
  direction: LedgerDirection;
  amount_minor: number; // whole TZS shillings as integer
}

export interface TrialBalanceLine {
  AccountName: string;
  Type: string;
  DebitMinor: number;
  CreditMinor: number;
}

export interface TrialBalance {
  GroupID: string;
  AsOf: string | null;
  Lines: TrialBalanceLine[] | null;
  TotalDebitMinor: number;
  TotalCreditMinor: number;
  Balanced: boolean;
}

export interface StatementLine {
  TransactionID: string;
  EventType: string;
  Direction: string;
  AmountMinor: number;
  Currency: string;
  OccurredAt: string;
  Memo: string;
}

export interface LedgerBalance {
  account: string;
  amount_minor: number;
  currency: string;
}

// Standard chart of accounts (Swahili naming, see backend/ledger/README.md).
// Member/bank accounts take a suffix, e.g. akiba_ya_mwanachama:KKK-0001.
export const STANDARD_ACCOUNTS: { name: string; type: LedgerAccountType }[] = [
  { name: "hazina_taslimu", type: "asset" },
  { name: "hazina_benki", type: "asset" },
  { name: "dai_la_mkopo", type: "asset" },
  { name: "akiba_ya_mwanachama", type: "liability" },
  { name: "mapato_ya_riba", type: "income" },
  { name: "mapato_ya_faini", type: "income" },
  { name: "hifadhi_ya_hasara_ya_mkopo", type: "expense" },
  { name: "mtaji_wa_kikundi", type: "equity" },
  { name: "faida_tulizo", type: "equity" },
];

export const ledgerApi = {
  openAccount: (data: {
    name: string;
    type: LedgerAccountType;
    owner_member_ref?: string;
  }) =>
    api.post<{ account_name: string; id: string }>(
      "/admin/ledger/accounts",
      data
    ),
  recordTransaction: (data: {
    memo: string;
    occurred_at?: string;
    entries: LedgerEntryInput[];
  }) =>
    api.post<{ transaction_id: string }>(
      "/admin/ledger/transactions",
      data
    ),
  reverseTransaction: (id: string, reason?: string) =>
    api.post<{ reversal_event_id: string }>(
      `/admin/ledger/transactions/${id}/reverse`,
      { reason }
    ),
  getBalance: (account: string) =>
    api.get<LedgerBalance>("/admin/ledger/balance", { account }),
  getStatement: (account: string, from?: string, to?: string) => {
    const q: Record<string, string> = { account };
    if (from) q.from = from;
    if (to) q.to = to;
    return api.get<{ statement: StatementLine[] | null }>(
      "/admin/ledger/statement",
      q
    );
  },
  getTrialBalance: () => api.get<TrialBalance>("/admin/ledger/trial-balance"),
};
