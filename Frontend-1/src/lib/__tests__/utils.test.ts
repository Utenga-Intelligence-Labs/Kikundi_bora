import { describe, it, expect, beforeEach } from "vitest";
import { cn, initials, decodeToken, isTokenExpired, getTokenExpiry, getTokenRole } from "../utils";
import { tokenStorage } from "../auth-storage";

beforeEach(() => {
  sessionStorage.clear();
  tokenStorage.clear();
});

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("px-4", "py-2")).toBe("px-4 py-2");
  });

  it("handles conditional classes", () => {
    expect(cn("base", false && "hidden", "extra")).toBe("base extra");
  });

  it("resolves tailwind conflicts", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });
});

describe("initials", () => {
  it("returns initials from full name", () => {
    expect(initials("Juma Kibwana")).toBe("JK");
  });

  it("handles single name", () => {
    expect(initials("Juma")).toBe("J");
  });

  it("handles double-space names", () => {
    expect(initials("Asha Mwakalinga")).toBe("AM");
  });

  it("returns ? for empty string", () => {
    expect(initials("")).toBe("?");
  });
});

describe("decodeToken", () => {
  it("decodes a valid JWT payload", () => {
    const header = btoa(JSON.stringify({ alg: "HS256" }));
    const payload = btoa(JSON.stringify({ user_id: "abc", role: "chair" }));
    const token = `${header}.${payload}.sig`;
    expect(decodeToken(token)).toEqual({ user_id: "abc", role: "chair" });
  });

  it("returns null for invalid token", () => {
    expect(decodeToken("not.a.token")).toBeNull();
  });

  it("returns null for garbage string", () => {
    expect(decodeToken("garbage")).toBeNull();
  });
});

describe("isTokenExpired", () => {
  it("returns false for a valid future token", () => {
    const exp = Math.floor(Date.now() / 1000) + 3600;
    const payload = btoa(JSON.stringify({ exp }));
    const token = `header.${payload}.sig`;
    expect(isTokenExpired(token)).toBe(false);
  });

  it("returns true for an expired token", () => {
    const exp = Math.floor(Date.now() / 1000) - 3600;
    const payload = btoa(JSON.stringify({ exp }));
    const token = `header.${payload}.sig`;
    expect(isTokenExpired(token)).toBe(true);
  });

  it("returns false when no exp claim", () => {
    const payload = btoa(JSON.stringify({ user_id: "abc" }));
    const token = `header.${payload}.sig`;
    expect(isTokenExpired(token)).toBe(false);
  });
});

describe("getTokenExpiry", () => {
  it("returns the exp timestamp", () => {
    const exp = 1720000000;
    const payload = btoa(JSON.stringify({ exp }));
    const token = `header.${payload}.sig`;
    expect(getTokenExpiry(token)).toBe(exp);
  });

  it("returns null when no exp claim", () => {
    const payload = btoa(JSON.stringify({ role: "member" }));
    const token = `header.${payload}.sig`;
    expect(getTokenExpiry(token)).toBeNull();
  });
});

describe("getTokenRole", () => {
  it("reads role from a valid JWT in sessionStorage", () => {
    const payload = btoa(JSON.stringify({ user_id: "abc", role: "chair" }));
    const token = `header.${payload}.sig`;
    sessionStorage.setItem("kikundi-token", token);
    tokenStorage.set(token);
    expect(getTokenRole()).toBe("chair");
  });

  it("returns null when no token stored", () => {
    expect(getTokenRole()).toBeNull();
  });
});
