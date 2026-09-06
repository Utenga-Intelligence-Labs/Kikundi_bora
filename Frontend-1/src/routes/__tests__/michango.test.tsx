/**
 * Consolidated /michango tests:
 * - treasurer sees Thibitisha/Kataa on PENDING AKIBA rows (row + modal)
 * - MFUKO rows show the chair-note instead of treasurer actions
 * - "Mifuko ya Kijamii" tab lists pending welfare obligations with
 *   treasurer approve/reject; secretary sees none of the action buttons
 * - approve posts to the right endpoint; reject requires a reason
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { Route as MichangoRoute } from "../michango";

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

vi.mock("@/api/upload", () => ({
  withUploadToken: (u: string) => u,
}));

import { api } from "@/api/client";

const michangoPayload = {
  data: [
    {
      id: "c-akiba",
      member_id: "m-1",
      contribution_type: "AKIBA",
      period_label: "2026-09",
      amount: "10000",
      status: "PENDING_VERIFICATION",
      created_at: "2026-09-01T10:00:00Z",
      member: { id: "m-1", member_no: "KKK-0001", full_name: "Asha Mwakalinga", phone: "0710000001" },
    },
    {
      id: "c-mfuko",
      member_id: "m-2",
      contribution_type: "MFUKO_WA_KIJAMII",
      period_label: "2026-09",
      amount: "5000",
      status: "PENDING_VERIFICATION",
      created_at: "2026-09-01T11:00:00Z",
      member: { id: "m-2", member_no: "KKK-0002", full_name: "Juma Kibwana", phone: "0710000002" },
    },
  ],
  total: 2,
};

const welfarePendingPayload = {
  data: [
    {
      id: "w-1",
      event_id: "e-1",
      member_id: "m-1",
      amount: "15000",
      status: "PENDING",
      created_at: "2026-09-02T10:00:00Z",
      member: { id: "m-1", member_no: "KKK-0001", full_name: "Asha Mwakalinga", phone: "0710000001" },
      event: { id: "e-1", event_type: "MATIBABU", description: "Matibabu", status: "APPROVED" },
    },
  ],
  total: 1,
};

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const C = (MichangoRoute as unknown as { component: React.ComponentType }).component;
  return render(
    <QueryClientProvider client={qc}>
      <AppModalProvider>
        <C />
      </AppModalProvider>
    </QueryClientProvider>
  );
}

describe("Consolidated /michango", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.role = "treasurer";
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/michango") return Promise.resolve(JSON.parse(JSON.stringify(michangoPayload)));
      if (url.startsWith("/welfare/contributions")) {
        return Promise.resolve(JSON.parse(JSON.stringify(welfarePendingPayload)));
      }
      return Promise.reject(new Error("unexpected GET " + url));
    });
    vi.mocked(api.post).mockResolvedValue({ message: "ok", data: {} } as never);
  });

  it("treasurer sees approve/reject on PENDING AKIBA rows, chair-note on MFUKO", async () => {
    renderPage();
    expect(await screen.findByText(/Asha Mwakalinga/)).toBeTruthy();
    const approveBtns = await screen.findAllByText("Thibitisha");
    expect(approveBtns.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Inasubiri Mwenyekiti.")).toBeTruthy();
  });

  it("approve posts to michango confirm endpoint", async () => {
    renderPage();
    const approveBtns = await screen.findAllByText("Thibitisha");
    fireEvent.click(approveBtns[0]);
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/michango/c-akiba/confirm")
    );
  });

  it("reject requires a reason before posting", async () => {
    renderPage();
    const rejectBtns = await screen.findAllByText("Kataa");
    fireEvent.click(rejectBtns[0]);
    expect(screen.getByPlaceholderText("Andika sababu...")).toBeTruthy();
    // Confirm button disabled with empty reason
    expect(screen.getByText("Thibitisha Kukataa")).toBeDisabled();
    fireEvent.change(screen.getByPlaceholderText("Andika sababu..."), {
      target: { value: "Picha haijulikani" },
    });
    fireEvent.click(screen.getByText("Thibitisha Kukataa"));
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/michango/c-akiba/reject", {
        reason: "Picha haijulikani",
      })
    );
  });

  it("mifuko tab lists pending welfare obligations with treasurer actions", async () => {
    renderPage();
    fireEvent.click(await screen.findByText("Michango ya Mifuko ya Kijamii"));
    expect(await screen.findByText("MATIBABU")).toBeTruthy();
    expect(screen.getByText("TZS 15,000")).toBeTruthy();
    const approveBtns = await screen.findAllByText("Thibitisha");
    fireEvent.click(approveBtns[0]);
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/welfare/contributions/w-1/approve")
    );
  });

  it("secretary sees no approve/reject actions anywhere", async () => {
    authState.role = "secretary";
    renderPage();
    expect(await screen.findByText(/Asha Mwakalinga/)).toBeTruthy();
    expect(screen.queryByText("Thibitisha")).toBeNull();
    expect(screen.queryByText("Kataa")).toBeNull();
    fireEvent.click(screen.getByText("Michango ya Mifuko ya Kijamii"));
    expect(await screen.findByText("MATIBABU")).toBeTruthy();
    expect(screen.queryByText("Thibitisha")).toBeNull();
    authState.role = "treasurer";
  });
});
