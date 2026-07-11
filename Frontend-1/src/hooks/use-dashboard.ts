import { useQuery } from "@tanstack/react-query";
import { dashboardApi } from "@/api/dashboard";

export const dashboardKeys = {
  summary: ["dashboard", "summary"] as const,
};

export function useDashboard() {
  return useQuery({
    queryKey: dashboardKeys.summary,
    queryFn: () => dashboardApi.summary(),
  });
}
