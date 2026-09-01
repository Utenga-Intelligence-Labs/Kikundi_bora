import { describe, it, expect } from "vitest";
import { getSidebarNav, mobileNav, sidebarNav, roleSubtitle } from "../roles";

describe("sidebarNav", () => {
  it("Mwenyekiti has 10 items", () => {
    expect(sidebarNav["Mwenyekiti"]).toHaveLength(10);
  });

  it("Mweka Hazina has 9 items", () => {
    expect(sidebarNav["Mweka Hazina"]).toHaveLength(9);
  });

  it("Katibu has 8 items", () => {
    expect(sidebarNav["Katibu"]).toHaveLength(8);
  });

  it("Mwanachama has 5 items", () => {
    expect(sidebarNav["Mwanachama"]).toHaveLength(5);
  });

  it("Msimamizi has 3 items", () => {
    expect(sidebarNav["Msimamizi"]).toHaveLength(3);
  });

  it("every item has required properties", () => {
    for (const role of Object.keys(sidebarNav)) {
      for (const item of sidebarNav[role as keyof typeof sidebarNav]) {
        expect(item).toHaveProperty("to");
        expect(item).toHaveProperty("label");
        expect(item).toHaveProperty("icon");
        expect(typeof item.to).toBe("string");
        expect(typeof item.label).toBe("string");
      }
    }
  });
});

describe("getSidebarNav", () => {
  it("returns same items for Mwenyekiti regardless of committee", () => {
    expect(getSidebarNav("Mwenyekiti", false)).toHaveLength(10);
    expect(getSidebarNav("Mwenyekiti", true)).toHaveLength(10);
  });

  it("inserts committee item for Mwanachama who is committee member", () => {
    const items = getSidebarNav("Mwanachama", true);
    expect(items).toHaveLength(6);
    expect(items[1].label).toBe("Kamati ya Mikopo");
    expect(items[1].to).toBe("/kamati-mikopo");
  });

  it("does not insert committee item for non-committee Mwanachama", () => {
    const items = getSidebarNav("Mwanachama", false);
    expect(items).toHaveLength(5);
    const labels = items.map((i) => i.label);
    expect(labels).not.toContain("Kamati ya Mikopo");
  });
});

describe("mobileNav", () => {
  it("always returns at least 4 items, at most 5", () => {
    for (const role of Object.keys(sidebarNav)) {
      const nav = mobileNav(role as keyof typeof sidebarNav, false);
      expect(nav.length).toBeGreaterThanOrEqual(4);
      expect(nav.length).toBeLessThanOrEqual(5);
    }
  });

  it("last item is always Wasifu", () => {
    for (const role of Object.keys(sidebarNav)) {
      const items = mobileNav(role as keyof typeof sidebarNav, false);
      expect(items[items.length - 1].label).toBe("Wasifu");
      expect(items[items.length - 1].to).toBe("/wasifu");
    }
  });

  it("limits to 4 main items + Wasifu", () => {
    const items = mobileNav("Mwenyekiti", true);
    expect(items[0].label).toBe("Dashibodi");
    expect(items[4].label).toBe("Wasifu");
  });
});

describe("roleSubtitle", () => {
  it("has subtitles for all roles", () => {
    const roles = Object.keys(sidebarNav);
    for (const role of roles) {
      expect(roleSubtitle[role as keyof typeof roleSubtitle]).toBeTruthy();
    }
  });
});
