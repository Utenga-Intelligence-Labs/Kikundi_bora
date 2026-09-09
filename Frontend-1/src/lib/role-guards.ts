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
 * Admin is a distinct system-level role and passes ONLY when explicitly
 * listed (e.g. requireRole("admin") for /admin). No blanket bypass.
 */
export function requireRole(...roles: string[]) {
  requireAuth();
  if (typeof window === "undefined") return;
  const role = getTokenRole();
  if (!role || !roles.includes(role)) {
    throw redirect({ to: "/dashibodi" });
  }
}

/**
 * Component-level check against a resolved User from /me.
 * Admin passes ONLY when explicitly listed — no blanket bypass.
 */
export function requireUserRole(
  user: User | null | undefined,
  ...roles: string[]
) {
  requireAuth(user);
  if (!user) return; // still loading, requireAuth already checked token exists
  if (!roles.includes(user.role)) {
    throw redirect({ to: "/dashibodi" });
  }
}

export function hasRole(
  user: User | null | undefined,
  ...roles: string[]
): boolean {
  if (!user) return false;
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

// Dual plane guard: requires user to have a linked member row.
// Admin has no member row (system-level role) — no bypass.
export function requireMember(user: User | null | undefined) {
  requireAuth(user);
  if (user && !user.member_id) {
    throw redirect({ to: "/dashibodi" });
  }
}

// Dual plane guard: requires user to hold at least one of the given leadership roles.
// Admin is not group leadership — no bypass.
export function requireLeadership(user: User | null | undefined, ...roles: string[]) {
  requireAuth(user);
  if (user && (!user.leadership || !user.leadership.some((r) => roles.includes(r)))) {
    throw redirect({ to: "/dashibodi" });
  }
}
