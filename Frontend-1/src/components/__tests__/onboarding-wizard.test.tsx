/**
 * Mwenyekiti setup-wizard tests:
 * - chair of a group with onboarding_completed=false sees the wizard on load;
 *   with onboarding_completed=true does not (unless re-opened via the link)
 * - wizard resumes at the correct step when left partway (localStorage) and
 *   when the server reports steps already done
 * - skipping/finishing calls onboarding-complete and never blocks navigation
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { api } from "@/api/client";
import { ONBOARDING_PROGRESS_KEY } from "@/api/onboarding";
import { OnboardingGate } from "../OnboardingGate";
import { OPEN_SETUP_EVENT } from "../OnboardingWizard";

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

const authState: { user: Record<string, unknown> | null } = {
  user: null,
};
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({ user: authState.user }),
}));

vi.mock("@/components/AppModal", () => ({
  AppModalProvider: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useAppModal: () => ({ showModal: vi.fn() }),
}));

const GROUP_ID = "g-1";
const chairUser = {
  id: "u-1",
  name: "Mwenyekiti",
  role: "chair",
  status: "ACTIVE",
  member_id: "m-1",
};

function mockApiResponses({
  completed = false,
  steps,
}: {
  completed?: boolean;
  steps?: Record<string, boolean>;
}) {
  (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
    if (path === "/groups/current") {
      return Promise.resolve({
        data: { id: GROUP_ID, name: "Kikundi" },
        pending_proposal: null,
      });
    }
    if (path === `/groups/${GROUP_ID}/onboarding-status`) {
      return Promise.resolve({
        data: {
          onboarding_completed: completed,
          steps: {
            contribution_proposed: false,
            contribution_approved: false,
            payment_method_added: false,
            members_added: false,
            ...steps,
          },
        },
      });
    }
    return Promise.resolve({ data: [] });
  });
}

function renderGate() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <OnboardingGate />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  authState.user = { ...chairUser };
  localStorage.clear();
});

describe("OnboardingGate — wizard", () => {
  it("shows the wizard on load when onboarding_completed=false", async () => {
    mockApiResponses({ completed: false });
    renderGate();
    expect(await screen.findByTestId("onboarding-wizard")).toBeDefined();
    expect(screen.getByText(/Karibu kwenye Kikundi chako/)).toBeDefined();
    expect(screen.getByText(/Hatua 1 kati ya 5/)).toBeDefined();
  });

  it("does NOT show the wizard when onboarding_completed=true", async () => {
    mockApiResponses({ completed: true });
    renderGate();
    await waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(screen.queryByTestId("onboarding-wizard")).toBeNull();
  });

  it("re-opens via the Setup ya Kikundi event even when completed", async () => {
    mockApiResponses({ completed: true });
    renderGate();
    await waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(screen.queryByTestId("onboarding-wizard")).toBeNull();
    fireEvent(window, new CustomEvent(OPEN_SETUP_EVENT));
    expect(await screen.findByTestId("onboarding-wizard")).toBeDefined();
  });

  it("resumes at the saved step when left partway (localStorage)", async () => {
    mockApiResponses({ completed: false });
    localStorage.setItem(
      ONBOARDING_PROGRESS_KEY,
      JSON.stringify({ step: 2, done: [] }),
    );
    renderGate();
    await screen.findByTestId("onboarding-wizard");
    // step index 2 = Njia za Malipo → "Hatua 3 kati ya 5"
    expect(await screen.findByText(/Hatua 3 kati ya 5/)).toBeDefined();
    expect(screen.getByText(/Njia za Malipo/)).toBeDefined();
  });

  it("resumes at the first incomplete step the server reports", async () => {
    mockApiResponses({
      completed: false,
      steps: {
        contribution_proposed: true,
        payment_method_added: true,
        members_added: false,
      },
    });
    renderGate();
    await screen.findByTestId("onboarding-wizard");
    // settings proposed + payment added → member step (index 3 → "Hatua 4")
    expect(await screen.findByText(/Hatua 4 kati ya 5/)).toBeDefined();
  });

  it("skip-all calls onboarding-complete and closes (never blocks)", async () => {
    mockApiResponses({ completed: false });
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue({
      message: "ok",
    });
    renderGate();
    await screen.findByTestId("onboarding-wizard");
    fireEvent.click(screen.getByTestId("wizard-skip-all"));
    await waitFor(() =>
      expect(api.patch).toHaveBeenCalledWith(
        `/groups/${GROUP_ID}/onboarding-complete`,
      ),
    );
    await waitFor(() =>
      expect(screen.queryByTestId("onboarding-wizard")).toBeNull(),
    );
    // progress storage is cleared so the wizard doesn't reopen mid-flow
    expect(localStorage.getItem(ONBOARDING_PROGRESS_KEY)).toBeNull();
  });

  it("non-chair users never see the wizard", async () => {
    authState.user = { ...chairUser, role: "treasurer" };
    mockApiResponses({ completed: false });
    renderGate();
    // treasurer triggers no onboarding queries — assert the overlay is absent
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.queryByTestId("onboarding-wizard")).toBeNull();
  });
});
