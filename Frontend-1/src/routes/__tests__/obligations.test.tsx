/**
 * DeniLangu + Ukusanyaji + Mikutano role-gating tests:
 * - grand total equals the sum of the three section subtotals
 * - treasurer sees "Mark as Collected" (Pokea); chair/secretary do not
 * - chair's waiver action proposes (waive-propose), never immediate waive
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { Route as DeniRoute } from "../deni-langu";
import { Route as UkusanyajiRoute } from "../ukusanyaji";
import { Route as MikutanoRoute } from "../mikutano";

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

const authState = { role: "member" as string };
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({
    user: {
      id: "u-1",
      name: "Test User",
      role: authState.role,
      member_id: "m-1",
    },
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

vi.mock("@/api/groups", () => ({
  groupsApi: { current: vi.fn() },
}));

import { api } from "@/api/client";
import { groupsApi } from "@/api/groups";

const obligationsPayload = {
  data: {
    member_id: "m-1",
    member_no: "KKK-0001",
    full_name: "Asha Mwakalinga",
    total_arrears: "20000",
    current_cycle_due: "10000",
    current_cycle_label: "2026-08",
    total_fines_unpaid: "5000",
    grand_total_owed: "35000",
    itemized_arrears: [
      { cycle_label: "2026-06", due_date: "2026-06-30", expected_amount: "10000", paid_amount: "0", owed: "10000" },
      { cycle_label: "2026-07", due_date: "2026-07-31", expected_amount: "10000", paid_amount: "0", owed: "10000" },
    ],
    itemized_fines: [
      { id: "f-1", offence_name: "Kuchelewa", offence_kind: "late_contribution", amount: "5000", occurrence_date: "2026-07-31", cycle_label: "2026-07", status: "unpaid", waiver_status: "none" },
    ],
  },
};

const queuePayload = {
  data: [
    {
      member: {
        member_id: "m-1", member_no: "KKK-0001", full_name: "Asha Mwakalinga",
        total_arrears: "20000", current_cycle_due: "10000",
        total_fines_unpaid: "5000", grand_total_owed: "35000",
      },
      arrears: obligationsPayload.data.itemized_arrears,
      fines: obligationsPayload.data.itemized_fines,
    },
  ],
  total: 1,
};

function renderWith(route: unknown) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const r = route as {
    component?: React.ComponentType;
    options?: { component?: React.ComponentType };
  };
  const C = r.component ?? r.options?.component ?? (() => null);
  return render(
    <QueryClientProvider client={qc}>
      <AppModalProvider>
        <C />
      </AppModalProvider>
    </QueryClientProvider>
  );
}

describe("DeniLangu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.role = "member";
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.includes("/obligations/summary")) return Promise.resolve(JSON.parse(JSON.stringify(obligationsPayload)));
      return Promise.resolve({ data: null });
    });
  });

  it("grand total equals the sum of the three section subtotals", async () => {
    renderWith(DeniRoute);
    const grand = await screen.findByTestId("grand-total");
    const arrears = await screen.findByTestId("arrears-section-total");
    const current = await screen.findByTestId("current-section-total");
    const fines = await screen.findByTestId("fines-section-total");
    const num = (el: HTMLElement) => Number(el.textContent?.replace(/[^0-9]/g, ""));
    expect(num(grand)).toBe(num(arrears) + num(current) + num(fines));
    expect(num(grand)).toBe(35000);
  });
});

describe("Ukusanyaji role gating", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(groupsApi.current).mockResolvedValue({ data: { id: "g-1" } } as never);
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.includes("/collection-queue")) return Promise.resolve(JSON.parse(JSON.stringify(queuePayload)));
      return Promise.resolve({ data: null });
    });
  });

  it("treasurer sees Mark as Collected (Pokea) and collects", async () => {
    authState.role = "treasurer";
    vi.mocked(api.post).mockResolvedValue({ message: "ok", data: {} });
    renderWith(UkusanyajiRoute);
    expect(await screen.findByText("Asha Mwakalinga")).toBeTruthy();
    const btn = await screen.findByLabelText("Mark as Collected Kuchelewa");
    fireEvent.click(btn);
    // confirm in modal
    const confirm = await screen.findByText("Nimepokea");
    fireEvent.click(confirm);
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/fines/f-1/collect")
    );
    authState.role = "member";
  });

  it("chair never sees the collect action", async () => {
    authState.role = "chair";
    renderWith(UkusanyajiRoute);
    expect(await screen.findByText("Ukurasa huu ni kwa Mweka Hazina tu.")).toBeTruthy();
    expect(screen.queryByLabelText("Mark as Collected Kuchelewa")).toBeNull();
    authState.role = "member";
  });
});

describe("Mikutano waiver propose flow", () => {
  const finesPayload = {
    data: [
      {
        id: "f-9", group_id: "g-1", member_id: "m-1",
        offence_type_id: "o-1", contribution_cycle_label: "2026-07",
        occurrence_date: "2026-07-31", due_date: "2026-07-31",
        amount: "5000", reason: "Kuchelewa", status: "unpaid",
        waiver_status: "none",
        offence_type: { id: "o-1", name: "Kuchelewa", kind: "late_contribution" },
        member: { id: "m-1", full_name: "Asha Mwakalinga", member_no: "KKK-0001" },
      },
    ],
    total: 1,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    authState.role = "chair";
    vi.mocked(groupsApi.current).mockResolvedValue({ data: { id: "g-1" } } as never);
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.startsWith("/fines")) return Promise.resolve(JSON.parse(JSON.stringify(finesPayload)));
      if (url.includes("fine-offence-types")) return Promise.resolve({ data: [], total: 0 });
      if (url.includes("/meetings")) return Promise.resolve({ data: [], total: 0 });
      return Promise.resolve({ data: null });
    });
    vi.mocked(api.post).mockResolvedValue({ message: "Ombi limetumwa", data: {} });
  });

  it("chair waiver action proposes (waive-propose), never immediate waive/collect", async () => {
    renderWith(MikutanoRoute);
    fireEvent.click(await screen.findByText("Faini na Misamaha"));
    expect(await screen.findByText("Kuchelewa")).toBeTruthy();
    // no collect button anywhere for chair
    expect(screen.queryByLabelText(/Mark as Collected/)).toBeNull();
    fireEvent.click(screen.getByText("Pendekeza Msamaha"));
    fireEvent.change(screen.getByPlaceholderText("Sababu ya msamaha"), {
      target: { value: "Mgonge" },
    });
    fireEvent.click(screen.getByText("Tuma"));
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/fines/f-9/waive-propose", {
        reason: "Mgonge",
      })
    );
    const urls = vi.mocked(api.post).mock.calls.map((c) => String(c[0]));
    expect(urls.some((u) => u.includes("waive-approve") || u.includes("/collect"))).toBe(false);
    authState.role = "member";
  });
});
