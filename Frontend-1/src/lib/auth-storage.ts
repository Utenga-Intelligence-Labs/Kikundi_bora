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
