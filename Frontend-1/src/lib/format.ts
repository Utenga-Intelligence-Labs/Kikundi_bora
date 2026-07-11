export const tzs = (n: number) =>
  new Intl.NumberFormat("sw-TZ", { maximumFractionDigits: 0 }).format(n || 0) + " TZS";

export const tarehe = (d: string | Date) => {
  const x = typeof d === "string" ? new Date(d) : d;
  return x.toLocaleDateString("sw-TZ", { day: "2-digit", month: "short", year: "numeric" });
};

export const mwezi = (d: string | Date) => {
  const x = typeof d === "string" ? new Date(d) : d;
  return x.toLocaleDateString("sw-TZ", { month: "long", year: "numeric" });
};
