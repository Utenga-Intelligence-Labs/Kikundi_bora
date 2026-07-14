import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useAdminLogs } from "@/hooks/use-admin";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, requireRole } from "@/lib/role-guards";
import { Activity, ChevronLeft, ChevronRight, Clock } from "lucide-react";

export const Route = createFileRoute("/admin-logs")({
  head: () => ({
    meta: [
      { title: "Kumbukumbu za Mfumo — Kikundi" },
      { name: "description", content: "Kumbukumbu za vitendo vya msimamizi." },
    ],
  }),
  beforeLoad: () => {
    requireRole("admin");
  },
  component: AdminLogsPage,
});

function AdminLogsPage() {
  const { user } = useAuth();
  const [page, setPage] = useState(1);
  const isAdmin = hasRole(user, "admin");
  // Do not fetch privileged data until auth confirms admin
  const { data, isLoading } = useAdminLogs({ page, limit: 30, enabled: isAdmin });

  if (!isAdmin) {
    return (
      <AppShell title="Kumbukumbu za Mfumo">
        <div className="flex items-center justify-center py-20">
          <p className="text-muted-foreground">Huna ruhusa ya kuona ukurasa huu.</p>
        </div>
      </AppShell>
    );
  }

  const logs = data?.data ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 30);

  const actionLabels: Record<string, string> = {
    USER_OVERRIDE: "Kubadilisha hali ya mtumiaji",
    PASSWORD_RESET: "Weka upya nenosiri",
  };

  return (
    <AppShell title="Kumbukumbu za Mfumo" subtitle="Vitendo vyote vya msimamizi">
      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="card-surface animate-pulse px-4 py-3">
              <div className="h-4 w-2/3 rounded bg-muted" />
            </div>
          ))}
        </div>
      ) : logs.length === 0 ? (
        <div className="card-surface flex flex-col items-center px-4 py-12 text-center">
          <Activity className="mb-3 h-10 w-10 text-muted-foreground" />
          <p className="text-sm font-medium">Hakuna kumbukumbu bado.</p>
          <p className="text-xs text-muted-foreground">Vitendo vya msimamizi vitaonekana hapa.</p>
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {logs.map((log) => (
            <div key={log.id} className="px-4 py-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium">{actionLabels[log.action] ?? log.action}</p>
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  {new Date(log.created_at).toLocaleString("sw-TZ")}
                </span>
              </div>
              <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                {log.admin && <span>Msimamizi: {log.admin.name}</span>}
                {log.target_user && <span>→ {log.target_user.name} ({log.target_user.phone})</span>}
                {log.ip_address && <span>IP: {log.ip_address}</span>}
              </div>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page <= 1}
            className="flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm disabled:opacity-40"
          >
            <ChevronLeft className="h-4 w-4" /> Nyuma
          </button>
          <span className="text-sm text-muted-foreground">Ukurasa {page} wa {totalPages}</span>
          <button
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page >= totalPages}
            className="flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm disabled:opacity-40"
          >
            Mbele <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}
    </AppShell>
  );
}
