import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "../auth-provider";
import { authApi } from "@/api/auth";
import type { User } from "@/api/types";
import { tokenStorage } from "../auth-storage";

vi.mock("@/api/auth", () => ({
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    me: vi.fn().mockRejectedValue(new Error("no token")),
    changePassword: vi.fn(),
    firstLoginSetup: vi.fn(),
  },
}));

beforeEach(() => {
  sessionStorage.clear();
  tokenStorage.clear();
  vi.clearAllMocks();
});

const createWrapper = () => {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  );
};

function AuthConsumer() {
  const auth = useAuth();
  return (
    <div>
      <span data-testid="user-name">{auth.user?.name ?? "none"}</span>
      <button
        data-testid="login-btn"
        onClick={() => auth.login({ email: "test@test.com", password: "pass123" })}
      >
        Login
      </button>
      <button data-testid="logout-btn" onClick={() => auth.logout()}>
        Logout
      </button>
    </div>
  );
}

describe("AuthProvider", () => {
  it("displays user name after successful login", async () => {
    const mockUser: User = {
      id: "2",
      name: "Test User",
      phone: "07123",
      role: "member",
      status: "ACTIVE",
      is_active: true,
      must_change_password: false,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    vi.mocked(authApi.login).mockResolvedValueOnce({
      token: "new-token",
      user: mockUser,
      expires_at: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
      first_login_required: false,
    });
    // After login the token exists, so AuthProvider refetches /me —
    // mock a successful response (module default rejects with "no token")
    vi.mocked(authApi.me).mockResolvedValueOnce(mockUser);

    const user = userEvent.setup();
    render(<AuthConsumer />, { wrapper: createWrapper() });

    await waitFor(
      () => expect(screen.getByTestId("login-btn")).toBeDefined(),
      { timeout: 5000 }
    );

    await user.click(screen.getByTestId("login-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("user-name").textContent).toBe("Test User");
      expect(tokenStorage.exists()).toBe(true);
    });
  });

  it("clears user on logout", async () => {
    tokenStorage.set("some-token");

    const user = userEvent.setup();
    render(<AuthConsumer />, { wrapper: createWrapper() });

    await waitFor(
      () => expect(screen.getByTestId("logout-btn")).toBeDefined(),
      { timeout: 5000 }
    );

    await user.click(screen.getByTestId("logout-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("user-name").textContent).toBe("none");
    });
  });
});
