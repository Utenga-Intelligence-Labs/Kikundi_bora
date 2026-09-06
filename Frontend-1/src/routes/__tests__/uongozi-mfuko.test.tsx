/**
 * /uongozi/mfuko (treasurer management) tests:
 * - event selector + per-member rows render with status chips
 * - PENDING rows expose Thibitisha / Rekodi / Samehe / Ondoa actions
 * - Ondoa asks for confirmation, then DELETEs the obligation
 * - paid rows show no action buttons
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { Route as MfukoRoute } from "../uongozi/mfuko";

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

const authState = { role: "treasurer" as string };
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({
    user: { id: "u-9", name: "Hazina", role: authState.role, member_id: "m-9" },
  }),
}));

vi.mock("@/lib/role-guards", () => ({
  requireAuth: () => {},
  requireRole: () => {},
  hasRole: (_u: unknown, ...roles: string[]) =>
    roles.includes(authState.role) || authState.role === "admin",
  blockAdminFromPage: () => {},
}));

vi.mock("@/components/AppShell", () => ({
  AppShell: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { api } from "@/api/client";

const eventsPayload = {
  data: [
    {
      id: "e-1",
      event_type: "MATIBABU",
      description: "Matibabu",
      status: "APPROVED",
      member: { id: "m-1", member_no: "KKK-0001", full_name: "Asha Mwakalinga" },
    },
  ],
  total: 1,
};

const detailPayload = {
  data: {
    id: "e-1",
    event_type: "MATIBABU",
    status: "APPROVED",
    amount_approved: 60000,
    amount_requested: 60000,
    member: { id: "m-1", member_no: "KKK-0001", full_name: "Asha Mwakalinga" },
  },
  contributions: [
    {
      id: "c-pending",
      event_id: "e-1",
      member_id: "m-2",
      amount: 30000,
      status: "PENDING",
      member: { id: "m-2", member_no: "KKK-0002", full_name: "Juma Kibwana" },
    },
    {
      id: "c-paid",
      event_id: "e-1",
      member_id: "m-3",
      amount: 30000,
      status: "PAID",
      member: { id: "m-3", member_no: "KKK-0003", full_name: "Neema Mhagama" },
    },
  ],
  stats: { paid_count: 1, pending_count: 1, total_paid: 30000, total_pending: 30000 },
};

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const C = (MfukoRoute as unknown as { component: React.ComponentType }).component;
  return render(
    <QueryClientProvider client={qc}>
      <AppModalProvider>
        <C />
      </AppModalProvider>
    </QueryClientProvider>
  );
}

describe("Treasurer /uongozi/mfuko", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.role = "treasurer";
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.startsWith("/welfare/events") && !url.includes("/welfare/events/")) {
        return Promise.resolve(JSON.parse(JSON.stringify({ data: eventsPayload.data, total: 1 })));
      }
      if (url.includes("/welfare/events/e-1")) {
        return Promise.resolve(JSON.parse(JSON.stringify(detailPayload)));
      }
      if (url.startsWith("/members")) {
        return Promise.resolve({ data: [], total: 0 });
      }
      return Promise.reject(new Error("unexpected GET " + url));
    });
    vi.mocked(api.post).mockResolvedValue({ message: "ok", data: {} } as never);
    vi.mocked(api.delete).mockResolvedValue({ message: "removed" } as never);
  });

  it("lists per-member rows with actions on PENDING only", async () => {
    renderPage();
    const sel = await screen.findByLabelText(/Chagua tukio/i);
    await screen.findByText("MATIBABU — Asha Mwakalinga (APPROVED)");
    fireEvent.change(sel, { target: { value: "e-1" } });
    expect(await screen.findByText("Juma Kibwana")).toBeTruthy();
    expect(screen.getByText("Neema Mhagama")).toBeTruthy();
    // PENDING row: all four actions
    expect(screen.getByTitle("Thibitisha")).toBeTruthy();
    expect(screen.getByTitle("Rekodi malipo")).toBeTruthy();
    expect(screen.getByTitle("Samehe")).toBeTruthy();
    expect(screen.getByTitle(/Ondoa mwanachama/)).toBeTruthy();
  });

  it("Ondoa confirms then DELETEs the obligation", async () => {
    renderPage();
    const sel = await screen.findByLabelText(/Chagua tukio/i);
    await screen.findByText("MATIBABU — Asha Mwakalinga (APPROVED)");
    fireEvent.change(sel, { target: { value: "e-1" } });
    fireEvent.click(await screen.findByTitle(/Ondoa mwanachama/));
    expect(await screen.findByText(/ataondolewa kwenye tukio hili kabisa/)).toBeTruthy();
    fireEvent.click(screen.getByText("Ondoa", { selector: "button.bg-destructive" }) ?? screen.getAllByText("Ondoa")[1]);
    await waitFor(() =>
      expect(api.delete).toHaveBeenCalledWith("/welfare/contributions/c-pending")
    );
  });

  it("approve posts to the contribution approve endpoint", async () => {
    renderPage();
    const sel = await screen.findByLabelText(/Chagua tukio/i);
    await screen.findByText("MATIBABU — Asha Mwakalinga (APPROVED)");
    fireEvent.change(sel, { target: { value: "e-1" } });
    fireEvent.click(await screen.findByTitle("Thibitisha"));
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/welfare/contributions/c-pending/approve")
    );
  });
});
