import { redirect } from "@tanstack/react-router";
import type { User, Jukumu } from "@/api/types";
import { tokenStorage } from "./auth-storage";
import { getTokenRole } from "./utils";

export function requireAuth(user: User | null | undefined) {
  if (!user) {
    throw redirect({ to: "/ingia" });
  }
}

export function requireRole(
  user: User | null | undefined,
  ...roles: string[]
) {
  requireAuth(user);
  // Admin bypasses all role checks
  if (user && user.role === "admin") return;
  if (user && !roles.includes(user.role)) {
    throw redirect({ to: "/dashibodi" });
  }
}

export function hasRole(
  user: User | null | undefined,
  ...roles: string[]
): boolean {
  if (!user) return false;
  if (user.role === "admin") return true;
  return roles.includes(user.role);
}

// For use in beforeLoad — redirects admin to dashboard
export function blockAdminFromPage() {
  if (typeof window === "undefined") return;
  const role = getTokenRole();
  if (role === "admin") {
    throw redirect({ to: "/dashibodi" });
  }
}

const isDev = import.meta.env.DEV;

export const DEMO_ACCOUNTS: Record<
  string,
  { email: string; password: string }
> = isDev
  ? {
      Mwenyekiti: { email: "juma@kikundi.tz", password: "demo123" },
      "Mweka Hazina": { email: "fatuma@kikundi.tz", password: "demo123" },
      Katibu: { email: "rashidi@kikundi.tz", password: "demo123" },
      Msimamizi: { email: "0000000000", password: "123456789" },
    }
  : {};
