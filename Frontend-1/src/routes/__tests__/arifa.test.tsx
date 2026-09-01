/**
 * ArifaPage component tests — notification detail modal, mark-as-read
 * (inline + modal + mark-all), and unread-badge sync.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { Route } from "../arifa";

const navigateMock = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return {
    ...actual,
    Link: (props: { to: string; children?: React.ReactNode }) => (
      <a href={props.to} onClick={(e) => e.preventDefault()}>
        {props.children}
      </a>
    ),
    useNavigate: () => navigateMock,
    createFileRoute: (path: string) => (opts: Record<string, unknown>) => ({
      ...opts,
      fullPath: path,
    }),
  };
});

vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({
    user: {
      id: "u-100",
      name: "Asha Mwakalinga",
      role: "member",
      member_id: "m-1",
    },
  }),
}));

vi.mock("@/lib/role-guards", () => ({
  requireAuth: () => {},
}));

vi.mock("@/components/AppShell", () => ({
  AppShell: ({
    children,
    action,
    subtitle,
  }: {
    children?: React.ReactNode;
    action?: React.ReactNode;
    subtitle?: string;
  }) => (
    <div>
      <div data-testid="subtitle">{subtitle}</div>
      <div data-testid="action">{action}</div>
      {children}
    </div>
  ),
}));

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

import { api } from "@/api/client";

const listPayload = {
  data: [
    {
      id: "u1",
      title: "Kumbusho la Mchango",
      message: "Mchango wa TZS 10,000 unatarajiwa ifikapo 2026-09-03.",
      type: "CONTRIBUTION_DUE",
      created_at: "2026-08-31T09:00:00Z",
    },
    {
      id: "r1",
      title: "Mchango Umethibitishwa",
      message: "Mchango wako wa TZS 5,000 umethibitishwa.",
      type: "CONTRIBUTION",
      read_at: "2026-08-31T10:00:00Z",
      created_at: "2026-08-31T08:00:00Z",
    },
  ],
  total: 2,
  unread: 1,
  page: 1,
  limit: 20,
};

const RouteAny = Route as unknown as {
  component?: React.ComponentType;
  options?: { component?: React.ComponentType };
};
const ArifaComponent = RouteAny.component ?? RouteAny.options?.component ?? (() => null);

describe("ArifaPage — notifications", () => {
  let qc: QueryClient;
  let db: typeof listPayload;
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <AppModalProvider>{children}</AppModalProvider>
    </QueryClientProvider>
  );

  beforeEach(() => {
    vi.clearAllMocks();
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    // Tiny stateful "server": GET returns current state; POST mutates it,
    // so refetches after mark-as-read return the updated truth (like the
    // real backend).
    db = JSON.parse(JSON.stringify(listPayload));
    vi.mocked(api.get).mockImplementation(async () => JSON.parse(JSON.stringify(db)));
    vi.mocked(api.post).mockImplementation(async (_path: string, body?: unknown) => {
      const ids = (body as { ids?: string[] } | undefined)?.ids;
      const stamp = new Date().toISOString();
      if (Array.isArray(ids)) {
        for (const n of db.data) {
          if (ids.includes(n.id) && !n.read_at) {
            n.read_at = stamp;
            db.unread -= 1;
          }
        }
      } else {
        for (const n of db.data) {
          if (!n.read_at) {
            n.read_at = stamp;
            db.unread -= 1;
          }
        }
      }
      return { message: "ok" };
    });
  });

  it("renders notifications with read/unread distinction", async () => {
    render(<ArifaComponent />, { wrapper });
    const unreadCard = await screen.findByTestId("notif-card-u1");
    expect(unreadCard.getAttribute("data-unread")).toBe("true");
    expect(screen.getByTestId("notif-card-r1").getAttribute("data-unread")).toBe("false");
    // subtitle shows unread count
    expect(screen.getByTestId("subtitle").textContent).toContain("arifa 1 mpya");
  });

  it("clicking a card opens the detail modal with full content and marks it read (optimistic)", async () => {
    render(<ArifaComponent />, { wrapper });
    const card = await screen.findByTestId("notif-card-u1");
    fireEvent.click(card);

    // Modal shows full title + full message + timestamp (scoped to dialog —
    // the title also appears on the card behind it)
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Kumbusho la Mchango")).toBeTruthy();
    expect(within(dialog).getByText(/unatarajiwa ifikapo 2026-09-03/)).toBeTruthy();
    expect(within(dialog).getByText(/Muda:/)).toBeTruthy();

    // Opening an unread notification marks it read immediately
    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/notifications/read", { ids: ["u1"] })
    );
    await waitFor(() =>
      expect(screen.getByTestId("notif-card-u1").getAttribute("data-unread")).toBe("false")
    );
    // Modal status chip + primary button now say Imesomwa
    await waitFor(() =>
      expect(within(dialog).getAllByText("Imesomwa").length).toBeGreaterThanOrEqual(1)
    );
  });

  it("inline 'Soma' button marks one notification read via the batch endpoint", async () => {
    render(<ArifaComponent />, { wrapper });
    const btn = await screen.findByTestId("mark-read-u1");
    fireEvent.click(btn);

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/notifications/read", { ids: ["u1"] })
    );
    await waitFor(() =>
      expect(screen.getByTestId("notif-card-u1").getAttribute("data-unread")).toBe("false")
    );
    // read notification (r1) has no inline mark button
    expect(screen.queryByTestId("mark-read-r1")).toBeNull();
  });

  it("'Soma Zote' calls read-all, clears unread styling and resets the badge", async () => {
    render(<ArifaComponent />, { wrapper });
    const markAll = await screen.findByText("Soma Zote");
    fireEvent.click(markAll);

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/notifications/read-all")
    );
    await waitFor(() => {
      expect(screen.getByTestId("notif-card-u1").getAttribute("data-unread")).toBe("false");
      expect(screen.getByTestId("notif-card-r1").getAttribute("data-unread")).toBe("false");
    });
    // badge cache (single source of truth for the header bell) reset to 0
    await waitFor(() => expect(qc.getQueryData(["notifications", "unread"])).toEqual({ unread: 0 }));
    // subtitle no longer reports unread
    expect(screen.getByTestId("subtitle").textContent).toBe("Angalia arifa zako");
  });

  it("rolls back optimistic read state and shows an error when the request fails", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("network down"));
    render(<ArifaComponent />, { wrapper });
    const btn = await screen.findByTestId("mark-read-u1");
    fireEvent.click(btn);

    // error modal appears
    expect(await screen.findByText("Hitilafu")).toBeTruthy();
    // optimistic update rolled back — card is unread again
    await waitFor(() =>
      expect(screen.getByTestId("notif-card-u1").getAttribute("data-unread")).toBe("true")
    );
    expect(screen.getByTestId("subtitle").textContent).toContain("arifa 1 mpya");
  });
});
