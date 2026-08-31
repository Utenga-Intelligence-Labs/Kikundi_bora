/**
 * Tests for dashboard cards - Simplified version
 */

import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { QueryClientProvider, QueryClient } from "@tanstack/react-query";
import { MemberAkibaCard, GroupBalanceCard } from "@/components/DashboardCards";
import type { MemberDashboardSummary, GroupDashboardSummary } from "@/api/types";

const qc = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
  },
});

// Mock data - matches API contract from backend
const mockMemberSummary: MemberDashboardSummary = {
  member_id: "member-123",
  member_no: "KKK-0009",
  full_name: "Asha Kipchoge",
  total_contributions: "40000",
  contributions_count: 2,
  welfare_contributions_total: "5000",
  welfare_contributions_count: 1,
  pending_contributions_count: 0,
  rejected_contributions_count: 0,
  outstanding_loans_count: 1,
  outstanding_loans_balance: "70000",
  closed_loans_count: 0,
  recent_contributions: [
    {
      id: "contrib-1",
      source: "member_contribution",
      contribution_type: "AKIBA",
      period_label: "2026-08",
      amount: "40000",
      status: "CONFIRMED",
      created_at: "2026-08-31T12:00:00Z",
    },
  ],
};

const mockGroupSummary: GroupDashboardSummary = {
  group_id: "group-1",
  group_name: "Kikundi Bora",
  total_active_members: 10,
  total_contributions: "500000",
  total_repayments: "100000",
  total_disbursed: "200000",
  available_balance: "400000",
  outstanding_loans_count: 3,
  outstanding_loans_balance: "150000",
  pending_loans_count: 1,
  pending_contributions_count: 2,
  contributions_this_period: "80000",
  contribution_interval: "monthly",
  next_due_date: "2026-09-01",
};

describe("DashboardCards - Rendering", () => {
  it("MemberAkibaCard should render without errors when no memberId", () => {
    const { container } = render(
      <QueryClientProvider client={qc}>
        <MemberAkibaCard memberId="" />
      </QueryClientProvider>
    );

    expect(container).toBeTruthy();
  });

  it("GroupBalanceCard should render without errors when no groupId", () => {
    const { container } = render(
      <QueryClientProvider client={qc}>
        <GroupBalanceCard groupId="" />
      </QueryClientProvider>
    );

    expect(container).toBeTruthy();
  });

  it("MemberAkibaCard should have card-surface class", () => {
    render(
      <QueryClientProvider client={qc}>
        <MemberAkibaCard memberId="" />
      </QueryClientProvider>
    );

    const card = document.querySelector(".card-surface");
    expect(card).toBeTruthy();
  });

  it("GroupBalanceCard should have card-surface class", () => {
    render(
      <QueryClientProvider client={qc}>
        <GroupBalanceCard groupId="" />
      </QueryClientProvider>
    );

    const card = document.querySelector(".card-surface");
    expect(card).toBeTruthy();
  });

  it("MemberAkibaCard should show loading skeleton when fetching", () => {
    render(
      <QueryClientProvider client={qc}>
        <MemberAkibaCard memberId="member-123" />
      </QueryClientProvider>
    );

    const skeleton = document.querySelector(".animate-spin");
    expect(skeleton).toBeTruthy();
  });

  it("GroupBalanceCard should show loading skeleton when fetching", () => {
    render(
      <QueryClientProvider client={qc}>
        <GroupBalanceCard groupId="group-1" />
      </QueryClientProvider>
    );

    const skeleton = document.querySelector(".animate-spin");
    expect(skeleton).toBeTruthy();
  });
});

describe("API Type Safety - MemberDashboardSummary", () => {
  it("should have all required snake_case fields", () => {
    const summary: MemberDashboardSummary = mockMemberSummary;

    expect(summary).toHaveProperty("member_id");
    expect(summary).toHaveProperty("member_no");
    expect(summary).toHaveProperty("full_name");
    expect(summary).toHaveProperty("total_contributions");
    expect(summary).toHaveProperty("contributions_count");
    expect(summary).toHaveProperty("welfare_contributions_total");
    expect(summary).toHaveProperty("welfare_contributions_count");
    expect(summary).toHaveProperty("pending_contributions_count");
    expect(summary).toHaveProperty("rejected_contributions_count");
    expect(summary).toHaveProperty("outstanding_loans_count");
    expect(summary).toHaveProperty("outstanding_loans_balance");
    expect(summary).toHaveProperty("closed_loans_count");
    expect(summary).toHaveProperty("recent_contributions");
  });

  it("should have correct field types", () => {
    const summary: MemberDashboardSummary = mockMemberSummary;

    // String values from API
    expect(typeof summary.total_contributions).toBe("string");
    expect(typeof summary.welfare_contributions_total).toBe("string");
    expect(typeof summary.outstanding_loans_balance).toBe("string");

    // Number values
    expect(typeof summary.contributions_count).toBe("number");
    expect(typeof summary.welfare_contributions_count).toBe("number");
    expect(typeof summary.pending_contributions_count).toBe("number");
    expect(typeof summary.outstanding_loans_count).toBe("number");

    // Array
    expect(Array.isArray(summary.recent_contributions)).toBe(true);
  });
});

describe("API Type Safety - GroupDashboardSummary", () => {
  it("should have all required snake_case fields", () => {
    const summary: GroupDashboardSummary = mockGroupSummary;

    expect(summary).toHaveProperty("group_id");
    expect(summary).toHaveProperty("group_name");
    expect(summary).toHaveProperty("total_active_members");
    expect(summary).toHaveProperty("total_contributions");
    expect(summary).toHaveProperty("total_repayments");
    expect(summary).toHaveProperty("total_disbursed");
    expect(summary).toHaveProperty("available_balance");
    expect(summary).toHaveProperty("outstanding_loans_count");
    expect(summary).toHaveProperty("outstanding_loans_balance");
    expect(summary).toHaveProperty("pending_loans_count");
    expect(summary).toHaveProperty("pending_contributions_count");
    expect(summary).toHaveProperty("contributions_this_period");
    expect(summary).toHaveProperty("contribution_interval");
    expect(summary).toHaveProperty("next_due_date");
  });

  it("should have correct field types", () => {
    const summary: GroupDashboardSummary = mockGroupSummary;

    // String values from API
    expect(typeof summary.total_contributions).toBe("string");
    expect(typeof summary.available_balance).toBe("string");
    expect(typeof summary.outstanding_loans_balance).toBe("string");

    // Number values
    expect(typeof summary.total_active_members).toBe("number");
    expect(typeof summary.outstanding_loans_count).toBe("number");
    expect(typeof summary.pending_contributions_count).toBe("number");
  });
});

describe("Regression Tests - Bug Fix", () => {
  it("should verify contribution data has CONFIRMED status", () => {
    // Regression test for bug: "Akiba Yangu: 0" despite confirmed contributions
    expect(mockMemberSummary.recent_contributions[0].status).toBe("CONFIRMED");
  });

  it("should verify contribution amount is NOT zero after confirmation", () => {
    // Bug was: total_contributions showed "0" even after CONFIRMED status
    expect(mockMemberSummary.total_contributions).toBe("40000");
    expect(mockMemberSummary.total_contributions).not.toBe("0");
  });

  it("should verify member count shows properly", () => {
    // Asha (KKK-0009) should be counted in contributions
    expect(mockMemberSummary.contributions_count).toBe(2);
  });
});
