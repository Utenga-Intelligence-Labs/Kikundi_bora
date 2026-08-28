// Backend decimal.Decimal amounts arrive as JSON strings — coerce before
// formatting. (Intl.NumberFormat also accepts strings, but be explicit.)
export const tzs = (n: number | string) =>
  new Intl.NumberFormat("sw-TZ", { maximumFractionDigits: 0 }).format(Number(n) || 0) + " TZS";

export const tarehe = (d: string | Date) => {
  const x = typeof d === "string" ? new Date(d) : d;
  if (isNaN(x.getTime())) return "—";
  return x.toLocaleDateString("sw-TZ", { day: "2-digit", month: "short", year: "numeric" });
};

export const mwezi = (d: string | Date) => {
  const x = typeof d === "string" ? new Date(d) : d;
  if (isNaN(x.getTime())) return "—";
  return x.toLocaleDateString("sw-TZ", { month: "long", year: "numeric" });
};
