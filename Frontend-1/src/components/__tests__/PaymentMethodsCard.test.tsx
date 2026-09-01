/**
 * PaymentMethodsCard tests — read-only for members (copy button, no
 * management), management form for mwenyekiti/hazina.
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AppModalProvider } from "@/components/AppModal";
import { PaymentMethodsCard } from "@/components/PaymentMethodsCard";

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
  };
});

const authState = { role: "member" as string };
vi.mock("@/lib/auth-provider", () => ({
  useAuth: () => ({
    user: { id: "u-1", name: "Test User", role: authState.role, member_id: "m-1" },
  }),
}));

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
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

const methodsPayload = {
  data: [
    {
      id: "pm-1",
      group_id: "g-1",
      type: "lipa_namba",
      provider_name: "M-Pesa",
      account_number: "255700000000",
      account_name: "Money Seeking Group",
      is_active: true,
    },
    {
      id: "pm-2",
      group_id: "g-1",
      type: "bank",
      provider_name: "CRDB",
      account_number: "0150000000000",
      account_name: "Money Seeking Group",
      is_active: true,
    },
  ],
  total: 2,
};

const PaymentMethodsCardWithRole = () => {
  const Route = { component: PaymentMethodsCard } as const;
  return <Route.component />;
};

describe("PaymentMethodsCard", () => {
  let qc: QueryClient;
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={qc}>
      <AppModalProvider>{children}</AppModalProvider>
    </QueryClientProvider>
  );

  beforeEach(() => {
    vi.clearAllMocks();
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    vi.mocked(groupsApi.current).mockResolvedValue({
      data: { id: "g-1", name: "Kikundi", contribution_interval: "monthly" },
      pending_proposal: null,
      next_due_date: null,
    });
    vi.mocked(api.get).mockResolvedValue(JSON.parse(JSON.stringify(methodsPayload)));
  });

  it("member sees read-only payment info with copy buttons and no management form", async () => {
    render(<PaymentMethodsCardWithRole />, { wrapper });

    expect(await screen.findByText(/M-Pesa/)).toBeTruthy();
    expect(screen.getByText("255700000000")).toBeTruthy();
    expect(screen.getByText(/CRDB/)).toBeTruthy();
    expect(screen.getAllByText("Money Seeking Group").length).toBe(2);

    // copy button available
    expect(screen.getByTestId("copy-pm-1")).toBeTruthy();

    // no management UI for members
    expect(screen.queryByText("Ongeza")).toBeNull();
    expect(screen.queryByLabelText("Badilisha")).toBeNull();
    expect(screen.queryByLabelText("Futa")).toBeNull();
  });

  it("copy button writes the account number to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<PaymentMethodsCardWithRole />, { wrapper });

    const btn = await screen.findByTestId("copy-pm-2");
    fireEvent.click(btn);

    await waitFor(() => expect(writeText).toHaveBeenCalledWith("0150000000000"));
    await waitFor(() => expect(screen.getByText("Imenakiliwa")).toBeTruthy());
  });

  it("mwenyekiti (chair) gets the management form and can add a payment method", async () => {
    authState.role = "chair";
    vi.mocked(api.post).mockResolvedValue({ message: "ok", data: {} });
    render(<PaymentMethodsCardWithRole />, { wrapper });

    const addBtn = await screen.findByText("Ongeza");
    fireEvent.click(addBtn);

    // management form fields appear
    const selects = screen.getAllByRole("combobox");
    fireEvent.change(selects[0], { target: { value: "lipa_namba" } });
    const inputs = screen.getAllByRole("textbox");
    fireEvent.change(inputs[0], { target: { value: "Tigo Pesa" } });
    fireEvent.change(inputs[1], { target: { value: "255710000000" } });
    fireEvent.change(inputs[2], { target: { value: "Money Seeking" } });

    fireEvent.click(screen.getByText("Hifadhi"));

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/groups/g-1/payment-methods", {
        type: "lipa_namba",
        provider_name: "Tigo Pesa",
        account_number: "255710000000",
        account_name: "Money Seeking",
      })
    );
    authState.role = "member";
  });

  it("management edit/deactivate/delete controls appear for treasurer", async () => {
    authState.role = "treasurer";
    render(<PaymentMethodsCardWithRole />, { wrapper });

    // wait for rows to render before asserting management controls
    expect(await screen.findByText(/M-Pesa/)).toBeTruthy();
    expect(screen.getByText("Ongeza")).toBeTruthy();
    expect(screen.getAllByLabelText("Badilisha").length).toBe(2);
    expect(screen.getAllByLabelText("Zima").length).toBe(2);
    expect(screen.getAllByLabelText("Futa").length).toBe(2);
    authState.role = "member";
  });
});
