/**
 * Role-switch context for managing multi-role user views.
 * When a user has multiple roles (e.g., mwenyekiti + mwanachama),
 * they can switch between role-specific dashboards without reloading.
 */

import {
  createContext,
  useContext,
  useState,
  useEffect,
  type ReactNode,
} from "react";
import type { Jukumu } from "@/api/types";
import { roleMap } from "@/api/types";

interface RoleSwitchContextValue {
  // The role currently being viewed
  currentRole: Jukumu | null;
  // All available roles for the user
  availableRoles: Jukumu[];
  // Switch to a different role
  switchRole: (role: Jukumu) => void;
  // Is the user viewing a different role than their primary?
  isViewingAltRole: boolean;
}

const RoleSwitchContext = createContext<RoleSwitchContextValue | null>(null);

export function RoleSwitchProvider({
  children,
  primaryRole,
  leadershipRoles,
  memberId,
}: {
  children: ReactNode;
  primaryRole: Jukumu;
  leadershipRoles: string[];
  memberId?: string;
}) {
  const [currentRole, setCurrentRole] = useState<Jukumu>(primaryRole);

  // Build available roles list
  const availableRoles: Jukumu[] = [];

  // Leadership positions first (if any)
  if (leadershipRoles.length > 0) {
    if (leadershipRoles.includes("MWENYEKITI")) {
      availableRoles.push("Mwenyekiti");
    }
    if (leadershipRoles.includes("KATIBU")) {
      availableRoles.push("Katibu");
    }
    if (leadershipRoles.includes("HAZINA")) {
      availableRoles.push("Mweka Hazina");
    }
  }

  // Then primary role (if not already in list)
  if (!availableRoles.includes(primaryRole)) {
    availableRoles.push(primaryRole);
  }

  // Then implicit member role (if linked member exists)
  if (memberId && !availableRoles.includes("Mwanachama")) {
    availableRoles.push("Mwanachama");
  }

  const isViewingAltRole = currentRole !== primaryRole;

  return (
    <RoleSwitchContext.Provider
      value={{
        currentRole,
        availableRoles,
        switchRole: setCurrentRole,
        isViewingAltRole,
      }}
    >
      {children}
    </RoleSwitchContext.Provider>
  );
}

export function useRoleSwitch() {
  const ctx = useContext(RoleSwitchContext);
  if (!ctx) {
    throw new Error("useRoleSwitch must be used within RoleSwitchProvider");
  }
  return ctx;
}
