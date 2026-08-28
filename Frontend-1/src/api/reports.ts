import { api } from "./client";

export const reportsApi = {
  wanachama: () => api.download("/reports/wanachama", "wanachama.csv"),
  michango: (month?: string) => {
    const qs = month ? `?month=${month}` : "";
    return api.download(`/reports/michango${qs}`, "michango.csv");
  },
  mikopo: (status?: string) => {
    const qs = status ? `?status=${status}` : "";
    return api.download(`/reports/mikopo${qs}`, "mikopo.csv");
  },
  mapato: () => api.download("/reports/mapato", "mapato_matumizi.csv"),
  muhtasari: () => api.download("/reports/muhtasari", "muhtasari.csv"),
};
