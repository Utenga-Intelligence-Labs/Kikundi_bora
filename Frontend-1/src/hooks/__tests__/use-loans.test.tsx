import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useLoans, useLoan, loanKeys } from "../use-loans";
import { loansApi } from "@/api/loans";

vi.mock("@/api/loans", () => ({
  loansApi: {
    list: vi.fn(),
    get: vi.fn(),
    apply: vi.fn(),
    approve: vi.fn(),
    reject: vi.fn(),
    disburse: vi.fn(),
    outstandingReport: vi.fn(),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
};

describe("loanKeys", () => {
  it("has correct all key", () => {
    expect(loanKeys.all).toEqual(["loans"]);
  });

  it("has correct list key with params", () => {
    expect(loanKeys.list({ status: "PENDING" })).toEqual([
      "loans", "list", { status: "PENDING" },
    ]);
  });

  it("has correct detail key", () => {
    expect(loanKeys.detail("loan-1")).toEqual(["loans", "detail", "loan-1"]);
  });

  it("has correct outstanding key", () => {
    expect(loanKeys.outstanding()).toEqual(["loans", "outstanding"]);
  });
});

describe("useLoans", () => {
  it("calls loansApi.list with correct params", async () => {
    vi.mocked(loansApi.list).mockResolvedValueOnce({ data: [], total: 0 });
    renderHook(() => useLoans({ status: "OUTSTANDING", page: 1 }), { wrapper });
    await waitFor(() =>
      expect(loansApi.list).toHaveBeenCalledWith({
        status: "OUTSTANDING", page: 1,
      })
    );
  });

  it("returns loan data on success", async () => {
    vi.mocked(loansApi.list).mockResolvedValueOnce({
      data: [{ id: "1", amount: 50000, status: "PENDING" }],
      total: 1,
    });
    const { result } = renderHook(() => useLoans({}), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.total).toBe(1);
  });
});

describe("useLoan", () => {
  it("fetches loan detail by id", async () => {
    vi.mocked(loansApi.get).mockResolvedValueOnce({
      data: { id: "loan-1", amount: 100000, status: "OUTSTANDING" },
      repayments: [],
    });
    const { result } = renderHook(() => useLoan("loan-1"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.data.id).toBe("loan-1");
  });
});
