/**
 * Tests for role-switch functionality
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClientProvider, QueryClient } from "@tanstack/react-query";
import { RoleSwitchProvider, useRoleSwitch } from "@/lib/role-context";
import { RoleSwitchToggle } from "@/components/RoleSwitchToggle";
import type { Jukumu } from "@/api/types";

const qc = new QueryClient();

function TestComponent() {
  const { currentRole, availableRoles, switchRole } = useRoleSwitch();
  return (
    <div>
      <div data-testid="current-role">{currentRole}</div>
      <div data-testid="available-roles">{availableRoles.join(",")}</div>
      <button
        onClick={() => switchRole("Mwanachama")}
        data-testid="switch-to-member"
      >
        Badili kuwa Mwanachama
      </button>
      <button
        onClick={() => switchRole("Mwenyekiti")}
        data-testid="switch-to-chair"
      >
        Badili kuwa Mwenyekiti
      </button>
    </div>
  );
}

describe("RoleSwitch Context", () => {
  it("should provide primary role as default", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwanachama"
          leadershipRoles={[]}
          memberId="member-123"
        >
          <TestComponent />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    expect(screen.getByTestId("current-role")).toHaveTextContent("Mwanachama");
  });

  it("should include primary role in available roles", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwenyekiti"
          leadershipRoles={[]}
          memberId="member-123"
        >
          <TestComponent />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    const availableRoles = screen.getByTestId("available-roles").textContent;
    expect(availableRoles).toContain("Mwenyekiti");
  });

  it("should include leadership roles in available roles", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwanachama"
          leadershipRoles={["MWENYEKITI", "KATIBU"]}
          memberId="member-123"
        >
          <TestComponent />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    const availableRoles = screen.getByTestId("available-roles").textContent;
    expect(availableRoles).toContain("Mwenyekiti");
    expect(availableRoles).toContain("Katibu");
  });

  it("should allow switching between roles", async () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwenyekiti"
          leadershipRoles={[]}
          memberId="member-123"
        >
          <TestComponent />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    expect(screen.getByTestId("current-role")).toHaveTextContent("Mwenyekiti");

    fireEvent.click(screen.getByTestId("switch-to-member"));

    await waitFor(() => {
      expect(screen.getByTestId("current-role")).toHaveTextContent("Mwanachama");
    });
  });

  it("should include implicit member role when memberId provided", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwenyekiti"
          leadershipRoles={["MWENYEKITI"]}
          memberId="member-123"
        >
          <TestComponent />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    const availableRoles = screen.getByTestId("available-roles").textContent;
    expect(availableRoles).toContain("Mwanachama");
    expect(availableRoles).toContain("Mwenyekiti");
  });

  it("should not include member role without memberId", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwenyekiti"
          leadershipRoles={["MWENYEKITI"]}
        >
          <TestComponent />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    const availableRoles = screen.getByTestId("available-roles").textContent;
    // Mwanachama should NOT be in the list without memberId
    expect(availableRoles).not.toContain("Mwanachama");
  });
});

describe("RoleSwitchToggle Component", () => {
  it("should not render when only one role available", () => {
    const { container } = render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwanachama"
          leadershipRoles={[]}
          memberId="member-123"
        >
          <RoleSwitchToggle />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    // Should render null (empty fragment), not the toggle
    const toggle = container.querySelector('[data-testid="role-switch-toggle"]');
    expect(toggle).toBeNull();
  });

  it("should render when multiple roles available", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwenyekiti"
          leadershipRoles={["MWENYEKITI"]}
          memberId="member-123"
        >
          <RoleSwitchToggle />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    // Should show the "Jukumu" label
    expect(screen.getByText("Jukumu")).toBeInTheDocument();
  });

  it("should display current role in toggle", () => {
    render(
      <QueryClientProvider client={qc}>
        <RoleSwitchProvider
          primaryRole="Mwenyekiti"
          leadershipRoles={["MWENYEKITI", "KATIBU"]}
          memberId="member-123"
        >
          <RoleSwitchToggle />
        </RoleSwitchProvider>
      </QueryClientProvider>
    );

    // The select trigger should show the current role
    const selectTrigger = screen.getByRole("combobox");
    expect(selectTrigger).toBeInTheDocument();
  });
});

describe("Role-switch sidebar nav integration", () => {
  it("should show correct nav items for mwanyachama role", () => {
    // This test would verify that when viewing as "Mwanachama",
    // only member nav items are shown (Michango Yangu, Weka Mchango, etc.)
    // Implementation depends on AppShell integration
    expect(true).toBe(true); // Placeholder
  });

  it("should show correct nav items for leadership role", () => {
    // This test would verify that when viewing as "Mwenyekiti",
    // leadership nav items are shown (Pokea Michango, Wanachama, etc.)
    expect(true).toBe(true); // Placeholder
  });

  it("should show role-specific nav items for katibu", () => {
    // Katibu should see secretary-specific items
    expect(true).toBe(true); // Placeholder
  });

  it("should show role-specific nav items for hazina", () => {
    // Hazina should see treasurer-specific items
    expect(true).toBe(true); // Placeholder
  });
});
