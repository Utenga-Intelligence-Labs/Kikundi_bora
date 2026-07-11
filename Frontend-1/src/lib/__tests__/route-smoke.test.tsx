import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "../auth-provider";
import { authApi } from "@/api/auth";
import { tokenStorage } from "../auth-storage";

vi.mock("@/api/auth", () => ({
  authApi: {
    login: vi.fn(),
    me: vi.fn().mockRejectedValue(new Error("no token")),
    logout: vi.fn(),
    register: vi.fn(),
    changePassword: vi.fn(),
    firstLoginSetup: vi.fn(),
  },
}));

beforeEach(() => {
  sessionStorage.clear();
  tokenStorage.clear();
  vi.clearAllMocks();
});

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return (
    <QueryClientProvider client={qc}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  );
};

describe("Route Component Smoke (Auth Integration)", () => {
  it("unauthenticated user sees login prompt", async () => {
    function ProtectedPage() {
      const { user, isLoading } = useAuth();
      if (isLoading) return <div>Loading...</div>;
      if (!user) return <div data-testid="unauthorized">Tafadhali ingia</div>;
      return <div data-testid="authorized">Karibu {user.name}</div>;
    }

    render(<ProtectedPage />, { wrapper });

    await waitFor(
      () => expect(screen.getByTestId("unauthorized")).toBeDefined(),
      { timeout: 5000 }
    );
    expect(screen.getByTestId("unauthorized").textContent).toBe("Tafadhali ingia");
  });

  it("role-based conditional rendering works", async () => {
    // Simulate logged-in chair user
    tokenStorage.set("fake-chair-token");
    vi.mocked(authApi.me).mockResolvedValueOnce({
      id: "1",
      name: "Mwenyekiti Juma",
      phone: "0710000001",
      role: "chair",
      status: "ACTIVE",
      is_active: true,
      email: null,
    });

    function RoleGatedContent() {
      const { user, isLoading } = useAuth();
      if (isLoading) return <div>Loading...</div>;
      if (!user) return <div>No user</div>;
      if (user.role === "chair") return <div data-testid="chair-view">Mwenyekiti Dashboard</div>;
      if (user.role === "member") return <div data-testid="member-view">Member Dashboard</div>;
      return <div>Unknown</div>;
    }

    render(<RoleGatedContent />, { wrapper });

    await waitFor(
      () => expect(screen.getByTestId("chair-view")).toBeDefined(),
      { timeout: 5000 }
    );
    expect(screen.getByTestId("chair-view").textContent).toBe("Mwenyekiti Dashboard");
  });

  it("member role sees different content than chair", async () => {
    tokenStorage.set("fake-member-token");
    vi.mocked(authApi.me).mockResolvedValueOnce({
      id: "2",
      name: "Asha Mwakalinga",
      phone: "0710000004",
      role: "member",
      status: "ACTIVE",
      is_active: true,
      email: null,
    });

    function RoleGatedContent() {
      const { user, isLoading } = useAuth();
      if (isLoading) return <div>Loading...</div>;
      if (!user) return <div>No user</div>;
      if (user.role === "chair") return <div>Chair Dashboard</div>;
      if (user.role === "member") return <div data-testid="member-view">Member Dashboard</div>;
      return <div>Unknown</div>;
    }

    render(<RoleGatedContent />, { wrapper });

    await waitFor(
      () => expect(screen.getByTestId("member-view")).toBeDefined(),
      { timeout: 5000 }
    );
  });
});
