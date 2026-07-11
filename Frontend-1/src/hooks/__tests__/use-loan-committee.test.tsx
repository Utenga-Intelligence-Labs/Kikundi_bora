import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useIsCommitteeMember, useCommitteeMembers, useCommitteeDashboard, committeeKeys } from "../use-loan-committee";
import { loanCommitteeApi } from "@/api/loan-committee";

vi.mock("@/api/loan-committee", () => ({
  loanCommitteeApi: {
    check: vi.fn(),
    listMembers: vi.fn(),
    listLoans: vi.fn(),
    getLoan: vi.fn(),
    submitReview: vi.fn(),
    getDashboard: vi.fn(),
    getHistory: vi.fn(),
    getReport: vi.fn(),
    getPendingLoansCount: vi.fn(),
    appointMember: vi.fn(),
    removeMember: vi.fn(),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    {children}
  </QueryClientProvider>
);

describe("committeeKeys", () => {
  it("has correct keys", () => {
    expect(committeeKeys.all).toEqual(["loan-committee"]);
    expect(committeeKeys.check()).toEqual(["loan-committee", "check"]);
    expect(committeeKeys.pendingCount()).toEqual(["loan-committee", "pending-count"]);
    expect(committeeKeys.loanDetail("loan-1")).toEqual(["loan-committee", "loan", "loan-1"]);
  });
});

describe("useIsCommitteeMember", () => {
  it("returns is_member true when user is committee", async () => {
    vi.mocked(loanCommitteeApi.check).mockResolvedValueOnce({ is_member: true });
    const { result } = renderHook(() => useIsCommitteeMember(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.is_member).toBe(true);
  });

  it("returns is_member false for non-committee", async () => {
    vi.mocked(loanCommitteeApi.check).mockResolvedValueOnce({ is_member: false });
    const { result } = renderHook(() => useIsCommitteeMember(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.is_member).toBe(false);
  });
});

describe("useCommitteeMembers", () => {
  it("returns members list", async () => {
    vi.mocked(loanCommitteeApi.listMembers).mockResolvedValueOnce({
      data: [{ user_id: "1", user_name: "Juma", user_role: "chairperson" }],
    });
    const { result } = renderHook(() => useCommitteeMembers(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.data).toHaveLength(1);
  });
});

describe("useCommitteeDashboard", () => {
  it("returns dashboard stats", async () => {
    vi.mocked(loanCommitteeApi.getDashboard).mockResolvedValueOnce({
      total_loans: 10, pending_loans: 3, approved_loans: 5, rejected_loans: 2,
    });
    const { result } = renderHook(() => useCommitteeDashboard(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.pending_loans).toBe(3);
  });
});
