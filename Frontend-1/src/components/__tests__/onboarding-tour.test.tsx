/**
 * New-member app tour tests:
 * - approved member with onboarding_seen=false sees the tour exactly once
 * - skipping or completing calls /members/:id/onboarding-seen and the tour
 *   never shows again automatically
 * - onboarding_seen=true → tour never renders
 * - tour dismissal never blocks navigation (plain overlay, closes cleanly)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { api } from "@/api/client";
import { OnboardingTour } from "../OnboardingTour";
import { OnboardingGate } from "../OnboardingGate";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: (props: { to: string; children?: React.ReactNode }) => (
      <a href={props.to} onClick={(e) => e.preventDefault()}>
        {props.children}
      </a>
    ),
    useNavigate: () => vi.fn(),
  };
});

vi.mock("@/components/AppModal", () => ({
  AppModalProvider: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useAppModal: () => ({ showModal: vi.fn() }),
}));

const authState: { user: Record<string, unknown> | null } = { user: null };
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({ user: authState.user }),
}));

const MEMBER_ID = "m-42";
const memberUser = {
  id: "u-42",
  name: "Asha",
  role: "member",
  status: "ACTIVE",
  member_id: MEMBER_ID,
  onboarding_seen: false,
};

describe("OnboardingTour (direct render)", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the first step and advances through all steps", () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <OnboardingTour open memberId={MEMBER_ID} onClose={() => {}} />
      </QueryClientProvider>,
    );
    expect(screen.getByTestId("onboarding-tour")).toBeDefined();
    expect(screen.getByText(/1 kati ya 5/)).toBeDefined();
    expect(screen.getByText("Dashibodi")).toBeDefined();
    fireEvent.click(screen.getByRole("button", { name: /Endelea/ }));
    expect(screen.getByText("Weka Mchango")).toBeDefined();
  });

  it("last step shows 'Sawa, nimeelewa' and marks seen on finish", async () => {
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue({
      message: "ok",
    });
    const onClose = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <OnboardingTour open memberId={MEMBER_ID} onClose={onClose} />
      </QueryClientProvider>,
    );
    // advance to the last step
    for (let i = 0; i < 4; i++)
      fireEvent.click(screen.getByRole("button", { name: /Endelea/ }));
    expect(screen.getByText(/Sawa, nimeelewa/)).toBeDefined();
    fireEvent.click(screen.getByTestId("tour-finish"));
    await waitFor(() =>
      expect(api.patch).toHaveBeenCalledWith(
        `/members/${MEMBER_ID}/onboarding-seen`,
      ),
    );
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(screen.queryByTestId("onboarding-tour")).toBeNull();
  });

  it("'Ruka' (skip) marks seen immediately — never blocks the user", async () => {
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue({
      message: "ok",
    });
    const onClose = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <OnboardingTour open memberId={MEMBER_ID} onClose={onClose} />
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByTestId("tour-skip"));
    await waitFor(() =>
      expect(api.patch).toHaveBeenCalledWith(
        `/members/${MEMBER_ID}/onboarding-seen`,
      ),
    );
    await waitFor(() =>
      expect(screen.queryByTestId("onboarding-tour")).toBeNull(),
    );
    expect(onClose).toHaveBeenCalled();
  });

  it("hides even if the mark-seen request fails (never traps the member)", async () => {
    (api.patch as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("offline"),
    );
    const onClose = vi.fn();
    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { mutations: { retry: false } } })
        }
      >
        <OnboardingTour open memberId={MEMBER_ID} onClose={onClose} />
      </QueryClientProvider>,
    );
    fireEvent.click(screen.getByTestId("tour-skip"));
    await waitFor(() =>
      expect(screen.queryByTestId("onboarding-tour")).toBeNull(),
    );
    expect(onClose).toHaveBeenCalled();
  });
});

describe("OnboardingGate — tour", () => {
  function gateUser(onboardingSeen: boolean | undefined) {
    return { ...memberUser, onboarding_seen: onboardingSeen };
  }

  function mockMeRelatedApi() {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] });
  }

  beforeEach(() => {
    vi.clearAllMocks();
    authState.user = null;
  });

  it("shows the tour once for an approved member with onboarding_seen=false", async () => {
    mockMeRelatedApi();
    authState.user = gateUser(false);
    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <OnboardingGate />
      </QueryClientProvider>,
    );
    expect(await screen.findByTestId("onboarding-tour")).toBeDefined();
  });

  it("does not render the tour when onboarding_seen=true", async () => {
    mockMeRelatedApi();
    authState.user = gateUser(true);
    render(
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <OnboardingGate />
      </QueryClientProvider>,
    );
    // members trigger no onboarding queries — assert the overlay is absent
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.queryByTestId("onboarding-tour")).toBeNull();
  });
});
