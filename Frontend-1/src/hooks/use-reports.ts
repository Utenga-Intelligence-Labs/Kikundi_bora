import { useMutation } from "@tanstack/react-query";
import { reportsApi } from "@/api/reports";

export function useDownloadReport() {
  return useMutation({
    mutationFn: ({ type, param }: { type: string; param?: string }) => {
      switch (type) {
        case "wanachama":
          return reportsApi.wanachama();
        case "michango":
          return reportsApi.michango(param);
        case "mikopo":
          return reportsApi.mikopo(param);
        case "mapato":
          return reportsApi.mapato();
        case "muhtasari":
          return reportsApi.muhtasari();
        default:
          throw new Error("Ripoti haipatikana");
      }
    },
  });
}
