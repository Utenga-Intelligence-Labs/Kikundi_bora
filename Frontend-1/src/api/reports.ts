const API_BASE =
  import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1";

function getToken(): string | null {
  try {
    return sessionStorage.getItem("kikundi-token");
  } catch {
    return null;
  }
}

async function downloadReport(path: string, filename: string) {
  const token = getToken();
  const res = await fetch(`${API_BASE}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message || `Hitilafu ya seva (${res.status})`);
  }

  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export const reportsApi = {
  wanachama: () => downloadReport("/reports/wanachama", "wanachama.csv"),
  michango: (month?: string) => {
    const qs = month ? `?month=${month}` : "";
    return downloadReport(`/reports/michango${qs}`, "michango.csv");
  },
  mikopo: (status?: string) => {
    const qs = status ? `?status=${status}` : "";
    return downloadReport(`/reports/mikopo${qs}`, "mikopo.csv");
  },
  mapato: () => downloadReport("/reports/mapato", "mapato_matumizi.csv"),
  muhtasari: () => downloadReport("/reports/muhtasari", "muhtasari.csv"),
};
