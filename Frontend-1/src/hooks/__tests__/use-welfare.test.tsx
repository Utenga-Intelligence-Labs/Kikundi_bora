import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useWelfareDashboard, useWelfareEvents, welfareKeys } from "../use-welfare";
import { welfareApi } from "@/api/welfare";

vi.mock("@/api/welfare", () => ({
  welfareApi: {
    getDashboard: vi.fn(),
    listEvents: vi.fn(),
    getEvent: vi.fn(),
    createEvent: vi.fn(),
    approveEvent: vi.fn(),
    rejectEvent: vi.fn(),
    listContributions: vi.fn(),
    myContributions: vi.fn(),
    recordPayment: vi.fn(),
    waiveContribution: vi.fn(),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    {children}
  </QueryClientProvider>
);

describe("welfareKeys", () => {
  it("has correct structure", () => {
    expect(welfareKeys.all).toEqual(["welfare"]);
    expect(welfareKeys.dashboard()).toEqual(["welfare", "dashboard"]);
    expect(welfareKeys.eventDetail("ev-1")).toEqual(["welfare", "event", "ev-1"]);
    expect(welfareKeys.events({ status: "PENDING" })).toEqual(["welfare", "events", { status: "PENDING" }]);
  });
});

describe("useWelfareDashboard", () => {
  it("returns dashboard data", async () => {
    vi.mocked(welfareApi.getDashboard).mockResolvedValueOnce({
      total_events: 5, pending_approval: 2, completed_events: 3,
      active_events: 0, rejected_events: 0, total_collected: 150000, total_from_treasury: 50000,
    });
    const { result } = renderHook(() => useWelfareDashboard(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.total_events).toBe(5);
    expect(result.current.data?.pending_approval).toBe(2);
  });

  it("calls welfareApi.getDashboard", async () => {
    vi.mocked(welfareApi.getDashboard).mockResolvedValueOnce({});
    renderHook(() => useWelfareDashboard(), { wrapper });
    await waitFor(() => expect(welfareApi.getDashboard).toHaveBeenCalled());
  });
});

describe("useWelfareEvents", () => {
  it("filters by status", async () => {
    vi.mocked(welfareApi.listEvents).mockResolvedValueOnce({ data: [], total: 0 });
    renderHook(() => useWelfareEvents({ status: "APPROVED" }), { wrapper });
    await waitFor(() =>
      expect(welfareApi.listEvents).toHaveBeenCalledWith({ status: "APPROVED" })
    );
  });
});
