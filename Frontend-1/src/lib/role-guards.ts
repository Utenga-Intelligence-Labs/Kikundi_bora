import { redirect } from "@tanstack/react-router";
import type { User } from "@/api/types";
import { tokenStorage } from "./auth-storage";
import { getTokenRole, isTokenExpired } from "./utils";

/**
 * beforeLoad guard: require a stored JWT (client-side).
 * Optionally pass a resolved user (e.g. from /me) to also reject null user.
 * Rejects missing or expired tokens for faster redirect before API 401.
 */
export function requireAuth(user?: User | null) {
  if (typeof window !== "undefined") {
    if (!tokenStorage.exists()) {
      throw redirect({ to: "/ingia" });
    }
    const token = tokenStorage.get();
    if (token && isTokenExpired(token)) {
      tokenStorage.clear();
      throw redirect({ to: "/ingia" });
    }
  }
  if (user === null) {
    throw redirect({ to: "/ingia" });
  }
}

/**
 * beforeLoad guard: require auth + one of the given roles (from JWT payload).
 * Admin bypasses role checks. Used for /admin, money pages, etc.
 */
export function requireRole(...roles: string[]) {
  requireAuth();
  if (typeof window === "undefined") return;
  const role = getTokenRole();
  if (role === "admin") return;
  if (!role || !roles.includes(role)) {
    throw redirect({ to: "/dashibodi" });
  }
}

/**
 * Component-level check against a resolved User from /me.
 */
export function requireUserRole(
  user: User | null | undefined,
  ...roles: string[]
) {
  requireAuth(user);
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
