import { describe, it, expect, vi, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { api, ApiError } from "../client";
import { tokenStorage } from "../../lib/auth-storage";

const BASE = "http://localhost:8080/api/v1";

const server = setupServer(
  http.get(`${BASE}/test-ok`, () => HttpResponse.json({ ok: true })),
  http.get(`${BASE}/test-401`, () =>
    HttpResponse.json({ message: "Unauthorized" }, { status: 401 })
  ),
  http.get(`${BASE}/test-401-token`, ({ request }) => {
    const auth = request.headers.get("Authorization");
    if (auth) {
      return HttpResponse.json({ message: "Session expired" }, { status: 401 });
    }
    return HttpResponse.json({ message: "No auth" }, { status: 401 });
  }),
  http.get(`${BASE}/test-500`, () =>
    HttpResponse.json({ message: "Server error" }, { status: 500 })
  ),
  http.post(`${BASE}/test-post`, async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json({ received: body });
  }),
  http.put(`${BASE}/test-put`, async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json({ updated: body });
  }),
  http.delete(`${BASE}/test-delete`, () =>
    HttpResponse.json({ deleted: true })
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  tokenStorage.clear();
  vi.restoreAllMocks();
});
afterAll(() => server.close());

describe("ApiClient", () => {
  it("GET returns JSON response", async () => {
    const result = await api.get<{ ok: boolean }>("/test-ok");
    expect(result.ok).toBe(true);
  });

  it("GET throws ApiError on 401 without token", async () => {
    try {
      await api.get("/test-401");
      expect.fail("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(401);
    }
  });

  it("GET throws ApiError with message on 500", async () => {
    try {
      await api.get("/test-500");
      expect.fail("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(500);
      expect((e as ApiError).message).toBe("Server error");
    }
  });

  it("POST sends JSON body and returns response", async () => {
    const result = await api.post<{ received: unknown }>("/test-post", { name: "test" });
    expect(result.received).toEqual({ name: "test" });
  });

  it("PUT sends JSON body", async () => {
    const result = await api.put<{ updated: unknown }>("/test-put", { id: 1 });
    expect(result.updated).toEqual({ id: 1 });
  });

  it("DELETE sends request", async () => {
    const result = await api.delete<{ deleted: boolean }>("/test-delete");
    expect(result.deleted).toBe(true);
  });

  it("appends query params to GET", async () => {
    server.use(
      http.get(`${BASE}/test-query`, ({ request }) => {
        const url = new URL(request.url);
        return HttpResponse.json({ q: url.searchParams.get("q") });
      })
    );
    const result = await api.get<{ q: string | null }>("/test-query", { q: "search" });
    expect(result.q).toBe("search");
  });

  it("sends Authorization header when token is configured", async () => {
    tokenStorage.set("test-bearer-token");

    let receivedAuth = "";
    server.use(
      http.get(`${BASE}/test-auth-header`, ({ request }) => {
        receivedAuth = request.headers.get("Authorization") || "";
        return HttpResponse.json({ ok: true });
      })
    );

    api.configure({
      getToken: () => tokenStorage.get(),
      onUnauthorized: vi.fn(),
    });

    await api.get("/test-auth-header");
    expect(receivedAuth).toBe("Bearer test-bearer-token");
  });
});
