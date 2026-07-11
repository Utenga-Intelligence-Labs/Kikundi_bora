import { describe, it, expect, beforeEach } from "vitest";
import { tokenStorage } from "../auth-storage";

beforeEach(() => {
  sessionStorage.clear();
  tokenStorage.clear();
});

describe("tokenStorage", () => {
  describe("set and get", () => {
    it("sets and gets a token", () => {
      tokenStorage.set("my-token-123");
      expect(tokenStorage.get()).toBe("my-token-123");
    });

    it("persists across calls within the same session", () => {
      tokenStorage.set("token-a");
      expect(tokenStorage.get()).toBe("token-a");
      expect(tokenStorage.get()).toBe("token-a");
    });
  });

  describe("clear", () => {
    it("removes the token", () => {
      tokenStorage.set("token-b");
      tokenStorage.clear();
      expect(tokenStorage.get()).toBeNull();
    });
  });

  describe("exists", () => {
    it("returns true when token is set", () => {
      tokenStorage.set("token-c");
      expect(tokenStorage.exists()).toBe(true);
    });

    it("returns false when no token", () => {
      expect(tokenStorage.exists()).toBe(false);
    });

    it("returns false after clear", () => {
      tokenStorage.set("token-d");
      tokenStorage.clear();
      expect(tokenStorage.exists()).toBe(false);
    });
  });

  describe("getRole", () => {
    it("decodes role from JWT payload", () => {
      const header = btoa(JSON.stringify({ alg: "HS256" }));
      const payload = btoa(JSON.stringify({ user_id: "abc", role: "chair" }));
      const token = `${header}.${payload}.sig`;
      tokenStorage.set(token);
      expect(tokenStorage.getRole()).toBe("chair");
    });

    it("returns null for invalid token", () => {
      tokenStorage.set("garbage");
      expect(tokenStorage.getRole()).toBeNull();
    });

    it("returns null when no token stored", () => {
      expect(tokenStorage.getRole()).toBeNull();
    });
  });
});
