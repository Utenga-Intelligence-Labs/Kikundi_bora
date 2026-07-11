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
