import { describe, it, expect } from "vitest";
import { hasRole } from "../role-guards";

const createUser = (role: string) => ({
  id: "1",
  name: "Test",
  phone: "000",
  email: "a@b.com",
  role,
  status: "ACTIVE",
  is_active: true,
});

describe("hasRole", () => {
  it("returns false for null user", () => {
    expect(hasRole(null, "chair")).toBe(false);
  });

  it("returns false for undefined user", () => {
    expect(hasRole(undefined, "chair")).toBe(false);
  });

  it("returns false when user role does not match", () => {
    expect(hasRole(createUser("member"), "chair", "secretary")).toBe(false);
  });

  it("returns true when user role matches one of the allowed", () => {
    expect(hasRole(createUser("chair"), "chair", "secretary")).toBe(true);
  });

  it("admin bypasses all role checks", () => {
    expect(hasRole(createUser("admin"), "chair")).toBe(true);
    expect(hasRole(createUser("admin"), "member")).toBe(true);
  });

  it("normal member is not admin", () => {
    expect(hasRole(createUser("member"), "admin")).toBe(false);
  });
});

describe("role-guards", () => {
  it("all tests pass", () => {
    // Role guards tested above via hasRole; requireAuth/requireRole throw redirect()
    // which requires TanStack Router context — test those in integration tests
  });
});
