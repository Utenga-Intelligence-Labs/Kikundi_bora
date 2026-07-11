import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useMembers, useMember, memberKeys } from "../use-members";
import { membersApi } from "@/api/members";

vi.mock("@/api/members", () => ({
  membersApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
};

describe("memberKeys", () => {
  it("has correct all key", () => {
    expect(memberKeys.all).toEqual(["members"]);
  });

  it("has correct list key", () => {
    expect(memberKeys.list({ page: 1 })).toEqual(["members", "list", { page: 1 }]);
  });

  it("has correct detail key", () => {
    expect(memberKeys.detail("abc-123")).toEqual(["members", "detail", "abc-123"]);
  });
});

describe("useMembers", () => {
  it("returns loading state initially", () => {
    vi.mocked(membersApi.list).mockResolvedValueOnce({ data: [], total: 0 });
    const { result } = renderHook(
      () => useMembers({ page: 1, q: "Juma" }),
      { wrapper }
    );
    expect(result.current.isLoading).toBe(true);
  });

  it("returns data and total on success", async () => {
    vi.mocked(membersApi.list).mockResolvedValueOnce({
      data: [{ id: "1", full_name: "Juma", phone: "07123" }],
      total: 1,
    });
    const { result } = renderHook(() => useMembers({}), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({
      data: [{ id: "1", full_name: "Juma", phone: "07123" }],
      total: 1,
    });
  });
});

describe("useMember", () => {
  it("does not fetch when no id", () => {
    vi.mocked(membersApi.get).mockResolvedValueOnce({});
    const { result } = renderHook(() => useMember(""), { wrapper });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("fetches member by id", async () => {
    vi.mocked(membersApi.get).mockResolvedValueOnce({
      id: "abc", full_name: "Asha", phone: "0712345678",
    });
    const { result } = renderHook(() => useMember("abc"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(membersApi.get).toHaveBeenCalledWith("abc");
  });
});
