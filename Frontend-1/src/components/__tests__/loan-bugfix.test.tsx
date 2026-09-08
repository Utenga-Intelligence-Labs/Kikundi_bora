/**
 * Bug-fix frontend tests (loans + committee + borrower confirmation):
 * - BUG-2: role-scoped "Mikopo Inayosubiri Idhini" dashboard card renders for
 *   leadership + bodi members, shows loans at MY stage, hidden for others
 * - BUG-2: /uongozi/mikopo page admits bodi members and shows "Zamu yako"
 * - BUG-3: "Unda Bodi Ya Mikopo" is on mwenyekiti's view, NOT katibu's
 * - BUG-5: Mikopo Yangu shows the distinct "Imetolewa — Inasubiri Uthibitisho
 *   Wako" state with a confirm action; confirmed loans show "Imethibitishwa"
 * - BUG-4: dashboard banner offers the inline "Nimepokea" action
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { api } from "@/api/client";

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
    Link: (props: {
      to?: string;
      children?: React.ReactNode;
      params?: unknown;
      className?: string;
      "data-testid"?: string;
    }) => (
      <a
        href={props.to ?? "#"}
        data-testid={props["data-testid"]}
        onClick={(e) => e.preventDefault()}
      >
        {props.children}
      </a>
    ),
    useNavigate: () => vi.fn(),
    createFileRoute: (path: string) => (opts: Record<string, unknown>) => ({
      ...opts,
      fullPath: path,
    }),
  };
});

const authState: { user: Record<string, unknown> | null } = { user: null };
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({
    user: authState.user,
    isLeadership: ["chair", "secretary", "treasurer"].includes(
      String(authState.user?.role ?? ""),
    ),
  }),
}));

vi.mock("@/lib/role-guards", () => ({
  requireAuth: () => {},
  requireRole: () => {},
  blockAdminFromPage: () => {},
}));

vi.mock("@/components/AppShell", () => ({
  AppShell: ({
    children,
    action,
  }: {
    children?: React.ReactNode;
    action?: React.ReactNode;
    title?: string;
  }) => (
    <div>
      {action}
      {children}
    </div>
  ),
}));

// Capture showModal calls; when autoConfirm is on, invoke onPrimary directly
// to simulate the user tapping the confirm button in the modal.
const modalCalls: Array<Record<string, unknown>> = [];
let autoConfirm = false;
vi.mock("@/components/AppModal", () => ({
  AppModalProvider: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useAppModal: () => ({
    showModal: (opts: Record<string, unknown>) => {
      modalCalls.push(opts);
      if (autoConfirm && typeof opts.onPrimary === "function") {
        void (opts.onPrimary as () => void | Promise<void>)();
      }
    },
  }),
}));

vi.mock("@/lib/auth-storage", () => ({ tokenStorage: { get: () => "tok" } }));
vi.mock("@/lib/format", () => ({
  tzs: (n: number) => `TZS ${Number(n).toLocaleString()}`,
  tarehe: (s: string) => String(s),
}));
vi.mock("@/lib/roles", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/roles")>();
  return { ...actual, getDualPlaneNav: () => ({ member: [], leadership: [] }) };
});

import { LoanApprovalsCard } from "../LoanApprovalsCard";
import { Route as MikopoRoute } from "../../routes/uongozi/mikopo";
import { Route as MyLoansRoute } from "../../routes/mikopo";

function renderWithQC(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

function routeComponent(route: unknown) {
  return (route as { component: React.ComponentType }).component;
}

const chairUser = { id: "u-1", name: "Juma", role: "chair", member_id: "m-1" };
const chairLoan = {
  id: "L1",
  amount: "150000",
  awaiting_role: "mwenyekiti",
  my_turn: true,
  member: { id: "m-9", member_no: "KKK-0009", full_name: "Asha", phone: "07" },
};

beforeEach(() => {
  vi.clearAllMocks();
  modalCalls.length = 0;
  autoConfirm = false;
  authState.user = { ...chairUser };
});

describe("BUG-2: LoanApprovalsCard", () => {
  it("shows the role-scoped queue with count for leadership", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/loan-committee/check")
        return Promise.resolve({ is_committee_member: true });
      if (path.startsWith("/uongozi/mikopo/pending")) {
        return Promise.resolve({
          data: [chairLoan, { ...chairLoan, id: "L2" }],
          total: 2,
        });
      }
      return Promise.resolve({ data: [] });
    });
    renderWithQC(<LoanApprovalsCard />);
    expect(await screen.findByTestId("loan-approvals-card")).toBeDefined();
    const count = await screen.findByTestId("loan-approvals-count");
    await waitFor(() => expect(count.textContent).toBe("2"));
    expect(screen.getAllByText(/Asha/).length).toBeGreaterThan(0);
    expect(screen.getByText(/Fungua mgawanyo wote/)).toBeDefined();
  });

  it("shows the empty state when no loans are at my stage", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/loan-committee/check")
        return Promise.resolve({ is_committee_member: true });
      if (path.startsWith("/uongozi/mikopo/pending"))
        return Promise.resolve({ data: [], total: 0 });
      return Promise.resolve({ data: [] });
    });
    renderWithQC(<LoanApprovalsCard />);
    expect(
      await screen.findByText(/Hakuna mkopo unasubiri hatua yako/),
    ).toBeDefined();
  });

  it("renders nothing for a plain member who is not on the bodi", async () => {
    authState.user = { ...chairUser, role: "member" };
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/loan-committee/check")
        return Promise.resolve({ is_committee_member: false });
      return Promise.resolve({ data: [] });
    });
    const { container } = renderWithQC(<LoanApprovalsCard />);
    await waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(
      container.querySelector('[data-testid="loan-approvals-card"]'),
    ).toBeNull();
  });
});

// ---------- BUG-2 + BUG-3: /uongozi/mikopo page ----------

function renderUongoziMikopo() {
  const Comp = (MikopoRoute as unknown as { component: React.ComponentType })
    .component;
  return renderWithQC(<Comp />);
}

const loanAtMyStage = {
  id: "L-1",
  member_id: "m-9",
  amount: "100000",
  due_date: "2026-12-31",
  status: "PENDING",
  applied_at: "2026-09-01",
  hazina_approved_at: "2026-09-01T08:00:00Z",
  katibu_approved_at: "2026-09-02T08:00:00Z",
  bodi_approved_at: null,
  mwenyekiti_approved_at: null,
  awaiting_role: "mwenyekiti",
  my_turn: true,
  member: { id: "m-9", member_no: "KKK-0009", full_name: "Asha", phone: "07" },
};

function stubFetchLoans(loans: unknown[]) {
  const fetchMock = vi.fn(async (_url: string | URL) => ({
    ok: true,
    json: async () => ({ data: loans, total: loans.length }),
  }));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("BUG-2/3: /uongozi/mikopo", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] });
  });

  it("shows 'Zamu yako' for a loan at the viewer's stage (chair)", async () => {
    authState.user = { ...chairUser };
    stubFetchLoans([loanAtMyStage]);
    renderUongoziMikopo();
    expect(await screen.findByText(/Asha/)).toBeDefined();
    expect(screen.getByTestId("my-turn-chip")).toBeDefined();
  });

  it("admits an appointed bodi member and does NOT show 'Huna ruhusa'", async () => {
    authState.user = { ...chairUser, role: "member" };
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/loan-committee/check")
        return Promise.resolve({ is_committee_member: true });
      return Promise.resolve({ data: [] });
    });
    stubFetchLoans([]);
    renderUongoziMikopo();
    await waitFor(() => expect(screen.queryByText(/Huna ruhusa/)).toBeNull());
  });

  it("BUG-3: 'Unda Bodi Ya Mikopo' appears for mwenyekiti, NOT for katibu", () => {
    authState.user = { ...chairUser, role: "secretary" };
    stubFetchLoans([]);
    const { unmount } = renderUongoziMikopo();
    // (this button was previously shown to katibu — must be gone)
    expect(screen.queryByText(/Unda Bodi Ya Mikopo/)).toBeNull();
    unmount();

    authState.user = { ...chairUser, role: "chair" };
    stubFetchLoans([]);
    renderUongoziMikopo();
    expect(screen.getByText(/Unda Bodi Ya Mikopo/)).toBeDefined();
  });
});

// ---------- BUG-5: Mikopo Yangu borrower receipt confirmation ----------

function renderMyLoans() {
  const Comp = (MyLoansRoute as unknown as { component: React.ComponentType })
    .component;
  return renderWithQC(<Comp />);
}

const outstandingLoan = {
  id: "L-9",
  member_id: "m-1",
  amount: 200000,
  approved_amount: 200000,
  balance_remaining: 200000,
  status: "OUTSTANDING",
  due_date: "2026-12-31",
  applied_at: "2026-09-01",
  borrower_confirmed_at: null as string | null,
};

describe("BUG-5: Mikopo Yangu borrower receipt confirmation", () => {
  beforeEach(() => {
    modalCalls.length = 0;
    autoConfirm = false;
    authState.user = { ...chairUser, role: "member", member_id: "m-1" };
  });

  it("shows the distinct unconfirmed state + confirm action", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: [outstandingLoan],
      total: 1,
    });
    renderMyLoans();
    expect(
      await screen.findByText(/Imetolewa — Inasubiri Uthibitisho Wako/),
    ).toBeDefined();
    expect(screen.getByTestId("confirm-loan-received")).toBeDefined();
  });

  it("confirm action PATCHes /loans/:id/confirm-received", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: [outstandingLoan],
      total: 1,
    });
    (api.patch as ReturnType<typeof vi.fn>).mockResolvedValue({
      message: "ok",
    });
    autoConfirm = true;
    renderMyLoans();
    const btn = await screen.findByTestId("confirm-loan-received");
    fireEvent.click(btn);
    await waitFor(() =>
      expect(api.patch).toHaveBeenCalledWith("/loans/L-9/confirm-received"),
    );
  });

  it("confirmed loans show 'Imethibitishwa' and no confirm button", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: [
        { ...outstandingLoan, borrower_confirmed_at: "2026-09-08T09:00:00Z" },
      ],
      total: 1,
    });
    renderMyLoans();
    expect(await screen.findByText(/Imetolewa — Imethibitishwa/)).toBeDefined();
    expect(screen.queryByTestId("confirm-loan-received")).toBeNull();
  });
});

// ---------- BUG-4: dashboard "Nimepokea" banner ----------
import { Route as DashibodiRoute } from "../../routes/dashibodi";

describe("BUG-4: beneficiary dashboard banner", () => {
  beforeEach(() => {
    modalCalls.length = 0;
    authState.user = { ...chairUser, role: "member", member_id: "m-1" };
  });

  it("offers inline 'Nimepokea' for a disbursed, unconfirmed welfare event", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/groups/current")
        return Promise.resolve({
          data: { id: "g1", contribution_interval: "monthly" },
          my_contribution: { status: "none" },
        });
      if (path === "/members/m-1/dashboard-summary")
        return Promise.resolve({
          data: {
            total_contributions: "0",
            outstanding_loans_count: 0,
            closed_loans_count: 0,
          },
        });
      if (path.startsWith("/welfare/events")) {
        return Promise.resolve({
          data: [
            {
              id: "W-1",
              member_id: "m-1",
              member: {
                id: "m-1",
                member_no: "KKK-0009",
                full_name: "Asha",
                phone: "07",
              },
              status: "COMPLETED",
              disbursed_at: "2026-09-01",
              received_at: null,
              amount_approved: "50000",
            },
          ],
          total: 1,
        });
      }
      return Promise.resolve({ data: [] });
    });
    (api.post as ReturnType<typeof vi.fn>).mockResolvedValue({ message: "ok" });
    autoConfirm = true;

    const Comp = (
      DashibodiRoute as unknown as { component: React.ComponentType }
    ).component;
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={qc}>
        <Comp />
      </QueryClientProvider>,
    );

    const btn = await screen.findByTestId("confirm-welfare-received");
    fireEvent.click(btn);
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        "/welfare/events/W-1/confirm-receipt",
      ),
    );
  });
});
