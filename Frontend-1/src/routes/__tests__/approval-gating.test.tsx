/**
 * Approval-gating frontend tests:
 * - membersApi.list forwards the status filter (used to fetch approved-only lists)
 * - buildCreateMemberPayload sends backdate fields only when toggle on + chair
 * - /uongozi/mfuko member dropdown hides non-approved members
 * - /ingia shows the pending-approval error distinctly (amber) vs wrong-password
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { membersApi } from "@/api/members";
import { api } from "@/api/client";
import { buildCreateMemberPayload } from "../wanachama";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

describe("membersApi.list status filter", () => {
  beforeEach(() => vi.clearAllMocks());

  it("forwards status=approved to GET /members", async () => {
    (api.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [], total: 0 });
    await membersApi.list({ limit: 500, status: "approved" });
    expect(api.get).toHaveBeenCalledWith("/members", { limit: "500", status: "approved" });
  });
});

describe("buildCreateMemberPayload", () => {
  const base = { full_name: "Asha", phone: "0710000001", joined_at: "2026-01-01" };

  it("omits backdate fields when toggle is off", () => {
    const p = buildCreateMemberPayload(base, { backdateOn: false, backdateFrom: "2025-10-01", isChair: true });
    expect(p).toEqual(base);
    expect("backdate_arrears" in p).toBe(false);
  });

  it("omits backdate fields for non-chair even when toggle is on", () => {
    const p = buildCreateMemberPayload(base, { backdateOn: true, backdateFrom: "2025-10-01", isChair: false });
    expect("backdate_arrears" in p).toBe(false);
    expect("backdate_from_cycle" in p).toBe(false);
  });

  it("includes backdate fields when toggle is on and actor is chair", () => {
    const p = buildCreateMemberPayload(base, { backdateOn: true, backdateFrom: "2025-10-01", isChair: true });
    expect(p).toMatchObject({ backdate_arrears: true, backdate_from_cycle: "2025-10-01" });
  });
});

// ---------- /uongozi/mfuko dropdown ----------

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

const loginMock = vi.fn();
const authMockState = {
  role: "treasurer" as string,
  user: { id: "u-9", name: "Hazina", role: "treasurer", member_id: "m-9" } as Record<string, unknown> | null,
};
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({ login: loginMock, user: authMockState.user }),
}));

vi.mock("@/lib/role-guards", () => ({
  requireAuth: () => {},
  requireRole: () => {},
  hasRole: (_u: unknown, ...roles: string[]) =>
    roles.includes(authMockState.role) || authMockState.role === "admin",
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

vi.mock("@/components/AppModal", () => ({
  AppModalProvider: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  useAppModal: () => ({ showModal: vi.fn() }),
}));

import { Route as MfukoRoute } from "../uongozi/mfuko";

function renderMfuko() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const Comp = (MfukoRoute as unknown as { component: React.ComponentType }).component;
  return render(
    <QueryClientProvider client={qc}>
      <Comp />
    </QueryClientProvider>,
  );
}

describe("/uongozi/mfuko member dropdown", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authMockState.user = { id: "u-9", name: "Hazina", role: "treasurer", member_id: "m-9" };
    (api.get as ReturnType<typeof vi.fn>).mockImplementation((path: string) => {
      if (path === "/members") {
        return Promise.resolve({
          data: [
            { id: "m-1", full_name: "Asha Halali", member_no: "KKK-0001", approval_status: "approved" },
            { id: "m-2", full_name: "Juma Anayesubiri", member_no: "KKK-0002", approval_status: "pending" },
          ],
          total: 2,
        });
      }
      return Promise.resolve({ data: [], total: 0 });
    });
  });

  it("hides non-approved members from the affected-member dropdown", async () => {
    renderMfuko();
    fireEvent.click(screen.getByText("Unda Tukio"));
    await waitFor(() => {
      expect(screen.getByText("Mwanachama Aliyeathiriwa")).toBeTruthy();
    });
    let memberSelect: HTMLSelectElement | undefined;
    await waitFor(() => {
      const selects = document.querySelectorAll("select");
      memberSelect = Array.from(selects).find((s) =>
        Array.from(s.options).some((o) => o.text.includes("Asha Halali")),
      ) as HTMLSelectElement | undefined;
      expect(memberSelect).toBeTruthy();
    });
    const labels = Array.from(memberSelect!.options).map((o) => o.text);
    expect(labels.some((t) => t.includes("Asha Halali"))).toBe(true);
    expect(labels.some((t) => t.includes("Juma Anayesubiri"))).toBe(false);
  });
});

// ---------- /ingia pending error ----------

import { Route as IngiaRoute } from "../ingia";

vi.mock("@/components/AuthLayout", () => ({
  AuthLayout: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/Field", () => ({
  Field: ({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) => (
    <label>
      {label}
      <input aria-label={label} value={value} onChange={(e) => onChange(e.target.value)} />
    </label>
  ),
}));

describe("/ingia pending-approval error", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authMockState.user = null;
  });

  function renderIngia() {
    const Comp = (IngiaRoute as unknown as { component: React.ComponentType }).component;
    return render(<Comp />);
  }

  it("shows pending-approval rejection in an amber box", async () => {
    loginMock.mockRejectedValueOnce(
      new Error("Akaunti yako bado inasubiri idhini ya katibu. Tafadhali subiri kuidhinishwa — si tatizo la nenosiri."),
    );
    renderIngia();
    fireEvent.change(screen.getByLabelText("Nambari ya simu au Barua pepe"), { target: { value: "0710000001" } });
    fireEvent.click(screen.getByText("Ingia"));
    await waitFor(() => {
      expect(screen.getByText(/bado inasubiri idhini ya katibu/)).toBeTruthy();
    });
    const box = screen.getByText(/bado inasubiri idhini ya katibu/);
    expect(box.className).toContain("amber");
  });

  it("shows wrong-password errors in red, not amber", async () => {
    loginMock.mockRejectedValueOnce(new Error("Nambari ya simu/barua pepe au nenosiri si sahihi"));
    renderIngia();
    fireEvent.change(screen.getByLabelText("Nambari ya simu au Barua pepe"), { target: { value: "0710000001" } });
    fireEvent.click(screen.getByText("Ingia"));
    await waitFor(() => {
      expect(screen.getByText(/nenosiri si sahihi/)).toBeTruthy();
    });
    const box = screen.getByText(/nenosiri si sahihi/);
    expect(box.className).not.toContain("amber");
    expect(box.className).toContain("destructive");
  });
});
