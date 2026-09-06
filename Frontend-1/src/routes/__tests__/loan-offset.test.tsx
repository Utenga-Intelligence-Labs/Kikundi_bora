/**
 * Loan offset (savings → overdue debt) frontend tests:
 * - portfolio: overdue loan row opens detail modal with "Offset na Akiba"
 *   section showing outstanding / available / capped amount + chair propose button
 * - member dashboard: offset history entries render with the distinct
 *   Swahili label, never as a normal contribution row
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { Route as PortfolioRoute } from "../uongozi/portfolio";
import { Route as DashibodiRoute } from "../dashibodi";

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: (props: { to: string; children?: React.ReactNode }) => (
      <a href={props.to} onClick={(e) => e.preventDefault()}>
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

const authState = {
  role: "chair" as string,
  user: { id: "u-1", name: "Mwenyekiti", role: "chair", member_id: "m-1" } as Record<string, unknown>,
};
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({ user: authState.user }),
}));

vi.mock("@/lib/role-guards", () => ({
  requireAuth: () => {},
  requireRole: () => {},
  hasRole: (_u: unknown, ...roles: string[]) =>
    roles.includes(authState.role) || authState.role === "admin",
  blockAdminFromPage: () => {},
}));

vi.mock("@/components/AppShell", () => ({
  AppShell: ({ children, action }: { children?: React.ReactNode; action?: React.ReactNode }) => (
    <div>
      {action}
      {children}
    </div>
  ),
}));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { api } from "@/api/client";

function renderWithClient(Comp: React.ComponentType) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <AppModalProvider>
        <Comp />
      </AppModalProvider>
    </QueryClientProvider>,
  );
}

const overdueLoan = {
  id: "loan-1",
  member_id: "m-2",
  member_no: "KKK-0002",
  full_name: "Juma Mdeni",
  principal: "100000",
  amount_repaid: "20000",
  outstanding: "80000",
  status: "OUTSTANDING",
  is_overdue: true,
  disbursed_at: "2025-01-01",
  due_date: "2025-06-01",
};

const currentLoan = {
  id: "loan-2",
  member_id: "m-3",
  member_no: "KKK-0003",
  full_name: "Neema Mlipaji",
  principal: "50000",
  amount_repaid: "10000",
  outstanding: "40000",
  status: "OUTSTANDING",
  is_overdue: false,
  disbursed_at: "2026-01-01",
  due_date: "2027-01-01",
};

describe("portfolio offset action", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.role = "chair";
    authState.user = { id: "u-1", name: "Mwenyekiti", role: "chair", member_id: "m-1" };
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path.startsWith("/loans/portfolio")) {
        return Promise.resolve({
          data: {
            total_disbursed: "150000", total_outstanding: "120000", total_overdue: "80000",
            count_outstanding: 2, count_closed: 0, count_overdue: 1,
            status_counts: {}, loans: [overdueLoan, currentLoan],
          },
        });
      }
      if (path === `/loans/${overdueLoan.id}/offset-preview`) {
        return Promise.resolve({
          data: {
            eligible: true, outstanding: "80000", gross_savings: "50000",
            offsets_applied: "0", available_savings: "50000", offset_amount: "50000",
          },
        });
      }
      if (path === "/loan-offsets") {
        return Promise.resolve({ data: [], total: 0 });
      }
      return Promise.resolve({ data: [], total: 0 });
    });
  });

  it("shows Offset section with capped figures for overdue loans (chair proposes)", async () => {
    const C = (PortfolioRoute as unknown as { component: React.ComponentType }).component;
    renderWithClient(C);
    await waitFor(() => expect(screen.getByText("Juma Mdeni")).toBeTruthy());
    fireEvent.click(screen.getByText("Juma Mdeni"));
    await waitFor(() => expect(screen.getByText("Offset na Akiba — mkopo umechelewa")).toBeTruthy());
    await waitFor(() => expect(screen.getByText("Kiasi kitakachokatwa")).toBeTruthy());
    // capped at savings (50k), not the full 80k outstanding
    const section = screen.getByText("Offset na Akiba — mkopo umechelewa").parentElement!;
    expect(section.textContent).toContain("50,000");
    expect(screen.getByText("Pendekeza Offset na Akiba")).toBeTruthy();
  });

  it("does not offer offset for non-overdue loans", async () => {
    const C = (PortfolioRoute as unknown as { component: React.ComponentType }).component;
    renderWithClient(C);
    await waitFor(() => expect(screen.getByText("Neema Mlipaji")).toBeTruthy());
    fireEvent.click(screen.getByText("Neema Mlipaji"));
    await waitFor(() => expect(screen.getByText("Historia ya Mkopo")).toBeTruthy());
    expect(screen.queryByText("Offset na Akiba — mkopo umechelewa")).toBeNull();
  });
});

describe("member dashboard offset history", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.role = "member";
    authState.user = { id: "u-2", name: "Asha", role: "member", member_id: "m-2", phone: "0710000004" };
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/groups/current") {
        return Promise.resolve({ data: { id: "g-1", fixed_contribution_amount: "10000" } });
      }
      if (path === "/members/m-2/dashboard-summary") {
        return Promise.resolve({
          data: {
            member_id: "m-2", member_no: "KKK-0002", full_name: "Asha",
            total_contributions: "50000", contributions_count: 5,
            total_offsets_applied: "30000", available_savings: "20000",
            welfare_contributions_total: "0", welfare_contributions_count: 0,
            pending_contributions_count: 0, rejected_contributions_count: 0,
            outstanding_loans_count: 1, outstanding_loans_balance: "70000",
            closed_loans_count: 0,
            recent_contributions: [
              {
                id: "off-1", source: "loan_offset", contribution_type: "OFFSET",
                period_label: "2026-08", amount: "30000", status: "OFFSET_APPLIED",
                created_at: "2026-08-15T10:00:00Z",
              },
              {
                id: "c-1", source: "contribution", contribution_type: "AKIBA",
                period_label: "2026-07", amount: "10000", status: "PAID",
                paid_at: "2026-07-05", created_at: "2026-07-05T10:00:00Z",
              },
            ],
          },
        });
      }
      if (path === "/welfare/contribute-events") {
        return Promise.resolve({ data: [], total: 0 });
      }
      return Promise.resolve({ data: [], total: 0 });
    });
  });

  it("labels the offset distinctly and shows net Akiba", async () => {
    const C = (DashibodiRoute as unknown as { component: React.ComponentType }).component;
    renderWithClient(C);
    await waitFor(() =>
      expect(screen.getByText(/Akiba yako imetumika kulipa mkopo uliochelewa/)).toBeTruthy(),
    );
    expect(screen.getByText("OFFSET")).toBeTruthy();
    expect(screen.getByText(/Akiba iliyotumika kulipa mkopo uliochelewa/)).toBeTruthy();
  });
});
