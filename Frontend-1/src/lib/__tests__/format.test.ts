import { describe, it, expect } from "vitest";
import { tzs, tarehe, mwezi } from "../format";

describe("tzs", () => {
  it("formats zero", () => {
    expect(tzs(0)).toBe("0 TZS");
  });

  it("formats a whole number with no decimals", () => {
    expect(tzs(500000)).toBe("500,000 TZS");
  });

  it("formats a large number", () => {
    expect(tzs(10000000)).toBe("10,000,000 TZS");
  });

  it("removes decimal part", () => {
    expect(tzs(1234.567)).toBe("1,235 TZS");
  });
});

describe("tarehe", () => {
  it("formats a Date object", () => {
    const result = tarehe(new Date("2024-05-15"));
    expect(result).toBe("15 Mei 2024");
  });

  it("formats an ISO string", () => {
    const result = tarehe("2024-01-03");
    expect(result).toBe("03 Jan 2024");
  });
});

describe("mwezi", () => {
  it("formats month and year from Date", () => {
    const result = mwezi(new Date("2024-03-10"));
    expect(result).toBe("Machi 2024");
  });

  it("formats month and year from string", () => {
    const result = mwezi("2024-12-01");
    expect(result).toBe("Desemba 2024");
  });
});
