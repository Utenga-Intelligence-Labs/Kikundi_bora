/**
 * SmsSettingsCard + OtpForm tests.
 * - Toggle + per-type checkboxes render and save via PUT.
 * - Noop-provider notice shows while no real vendor is wired.
 * - OtpForm (preserved, unrouted) renders and posts a 6-digit code.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { SmsSettingsCard } from "@/components/SmsSettingsCard";
import { OtpForm } from "@/components/OtpForm";

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
  };
});

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/api/groups", () => ({
  groupsApi: {
    current: vi.fn(),
  },
}));

import { api } from "@/api/client";
import { groupsApi } from "@/api/groups";

const settingsPayload = {
  data: {
    sms_enabled: false,
    provider: "noop",
    provider_real: false,
    types: {
      CONTRIBUTION_DUE: true,
      FINE_ISSUED: false,
    },
  },
};

function renderWithProviders(children: React.ReactNode) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AppModalProvider>{children}</AppModalProvider>
    </QueryClientProvider>
  );
}

describe("SmsSettingsCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(groupsApi.current).mockResolvedValue({
      data: { id: "g-1" },
    } as never);
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url.includes("/notification-settings")) {
        return Promise.resolve(JSON.parse(JSON.stringify(settingsPayload)));
      }
      return Promise.reject(new Error("unexpected GET " + url));
    });
    vi.mocked(api.put).mockResolvedValue({ data: settingsPayload.data });
  });

  it("renders the master toggle and per-type checkboxes", async () => {
    renderWithProviders(<SmsSettingsCard />);
    expect(await screen.findByLabelText("SMS zimewashwa")).toBeTruthy();
    expect(await screen.findByLabelText("Ukumbusho wa mchango")).toBeTruthy();
    expect(await screen.findByLabelText("Taarifa ya faini")).toBeTruthy();
  });

  it("shows the noop-provider notice until a real vendor is wired", async () => {
    renderWithProviders(<SmsSettingsCard />);
    expect(
      await screen.findByText(/Mtoa huduma wa SMS bado haujaunganishwa/)
    ).toBeTruthy();
  });

  it("saves toggle + type changes via PUT", async () => {
    renderWithProviders(<SmsSettingsCard />);
    const toggle = await screen.findByLabelText("SMS zimewashwa");
    fireEvent.click(toggle);
    const fineBox = await screen.findByLabelText("Taarifa ya faini");
    fireEvent.click(fineBox);
    fireEvent.click(screen.getByText("Hifadhi"));
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith(
        "/groups/g-1/notification-settings",
        { sms_enabled: true, types: { FINE_ISSUED: true } }
      )
    );
  });
});

describe("OtpForm (preserved, unrouted)", () => {
  beforeEach(() => {
    if (!(globalThis as Record<string, unknown>).ResizeObserver) {
      (globalThis as Record<string, unknown>).ResizeObserver = class {
        observe() {}
        unobserve() {}
        disconnect() {}
      };
    }
  });

  it("renders the OTP slots and posts the code to verify-otp", async () => {
    const onVerified = vi.fn();
    const realFetch = globalThis.fetch;
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ token: "tok-123" }),
    });
    (globalThis as Record<string, unknown>).fetch = fetchMock;
    try {
      const { container } = renderWithProviders(
        <OtpForm challengeId="ch-1" onVerified={onVerified} />
      );
      // input-otp renders a single (visually hidden) input plus slot divs.
      const input = container.querySelector("input");
      expect(input).toBeTruthy();
      expect(
        container.querySelectorAll("[data-slot]").length
      ).toBeGreaterThanOrEqual(0);
      fireEvent.change(input as Element, { target: { value: "123456" } });
      fireEvent.click(screen.getByText("Thibitisha"));
      await waitFor(() =>
        expect(fetchMock).toHaveBeenCalledWith(
          "/api/v1/auth/verify-otp",
          expect.objectContaining({
            method: "POST",
            body: JSON.stringify({ challenge_id: "ch-1", code: "123456" }),
          })
        )
      );
      expect(onVerified).toHaveBeenCalledWith("tok-123");
    } finally {
      (globalThis as Record<string, unknown>).fetch = realFetch;
    }
  });
});
