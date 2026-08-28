const API_BASE =
  import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";

export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number
  ) {
    super(message);
    this.name = "ApiError";
  }
}

class ApiClient {
  private getToken: (() => string | null) | null = null;
  private onUnauthorized: (() => void) | null = null;

  configure(opts: { getToken: () => string | null; onUnauthorized: () => void }) {
    this.getToken = opts.getToken;
    this.onUnauthorized = opts.onUnauthorized;
  }

  private headers(): HeadersInit {
    const h: HeadersInit = { "Content-Type": "application/json" };
    const token = this.getToken?.();
    if (token) h["Authorization"] = `Bearer ${token}`;
    return h;
  }

  async request<T>(path: string, opts: RequestInit = {}): Promise<T> {
    const merged: Record<string, string> = { "Content-Type": "application/json" };
    const token = this.getToken?.();
    if (token) merged["Authorization"] = `Bearer ${token}`;
    if (opts.headers) {
      for (const [k, v] of Object.entries(opts.headers)) {
        if (v !== undefined && v !== null) merged[k] = v;
        else delete merged[k];
      }
    }
    const res = await fetch(`${API_BASE}${path}`, {
      ...opts,
      headers: merged,
    });

    if (res.status === 401) {
      // Only treat as session expiry if user had a token (protected request)
      const hasToken = !!this.getToken?.();
      if (hasToken) {
        this.onUnauthorized?.();
        throw new ApiError("Session imeisha. Tafadhali ingia tena.", 401);
      }
      // No token = login/register attempt with wrong credentials
      const body = await res.json().catch(() => ({}));
      throw new ApiError(
        body.message || "Nambari ya simu/barua pepe au nenosiri si sahihi",
        401
      );
    }

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new ApiError(
        body.message || `Hitilafu ya seva (${res.status})`,
        res.status
      );
    }

    if (res.status === 204) return {} as T;
    return res.json();
  }

  get<T>(path: string, params?: Record<string, string>) {
    const qs = params ? "?" + new URLSearchParams(params).toString() : "";
    return this.request<T>(path + qs);
  }

  post<T>(path: string, body?: unknown) {
    return this.request<T>(path, {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  put<T>(path: string, body?: unknown) {
    return this.request<T>(path, {
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  delete<T>(path: string) {
    return this.request<T>(path, { method: "DELETE" });
  }
}

export const api = new ApiClient();
