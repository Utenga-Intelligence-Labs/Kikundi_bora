// SECURITY NOTE: Token storage uses sessionStorage (not httpOnly cookies).
// Tradeoffs:
// - sessionStorage: cleared when tab closes, but vulnerable to XSS token theft
// - httpOnly cookies: immune to XSS, but vulnerable to CSRF and harder to use
//   with SPAs that need to read the token for API calls
//
// Mitigations in place:
// - Content-Security-Policy headers prevent inline script injection
// - X-Content-Type-Options prevents MIME sniffing
// - Token expiry reduced to 30 minutes with auto-refresh
// - Server verifies role from DB on every request (stale token can't escalate)
//
// To switch to httpOnly cookies, the backend would need to set the cookie
// and the frontend would need to use credentials: 'include' on all fetch calls.

let accessToken: string | null = null;

export const tokenStorage = {
  get: (): string | null => {
    if (accessToken) return accessToken;
    if (typeof window !== "undefined") {
      const stored = sessionStorage.getItem("kikundi-token");
      if (stored) accessToken = stored;
    }
    return accessToken;
  },
  set: (token: string) => {
    accessToken = token;
    if (typeof window !== "undefined") {
      sessionStorage.setItem("kikundi-token", token);
    }
  },
  clear: () => {
    accessToken = null;
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("kikundi-token");
    }
  },
  exists: (): boolean => {
    if (accessToken) return true;
    if (typeof window !== "undefined") {
      return !!sessionStorage.getItem("kikundi-token");
    }
    return false;
  },
  // Decode JWT payload for UI role gating only.
  // SECURITY: This is NOT used for authorization — the server verifies
  // the role from the database on every request (see middleware/auth.go).
  // Client-side role checks are purely for UX (showing/hiding UI elements).
  getRole: (): string | null => {
    const token = tokenStorage.get();
    if (!token) return null;
    try {
      const parts = token.split(".");
      if (parts.length !== 3) return null;
      const payload = JSON.parse(atob(parts[1]));
      return payload.role || null;
    } catch {
      return null;
    }
  },
};
