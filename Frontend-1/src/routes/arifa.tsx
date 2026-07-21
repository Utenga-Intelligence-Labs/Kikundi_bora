import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Bell, Check, Loader2 } from "lucide-react";

export const Route = createFileRoute("/arifa")({
  beforeLoad: () => {
    requireAuth();
  },
  component: ArifaPage,
});

interface Notification {
  id: string;
  title: string;
  message: string;
  type: string;
  is_read: boolean;
  created_at: string;
}

function ArifaPage() {
  const { user } = useAuth();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery<{ data: Notification[]; total: number; unread: number }>({
    queryKey: ["notifications"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/notifications", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata arifa");
      return res.json();
    },
  });

  const markReadMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/notifications/read", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ ids }),
      });
      if (!res.ok) throw new Error("Imeshindikana");
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  if (!user) return null;

  const notifications = data?.data ?? [];

  const markAllAsRead = () => {
    const unreadIds = notifications.filter((n) => !n.is_read).map((n) => n.id);
    if (unreadIds.length > 0) {
      markReadMutation.mutate(unreadIds);
    }
  };

  return (
    <AppShell title="Arifa Zangu" subtitle="Angalia arifa zako">
      {notifications.length > 0 && (
        <div className="mb-4 flex justify-end">
          <button
            onClick={markAllAsRead}
            disabled={markReadMutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Check className="h-4 w-4" />
            Soma Zote
          </button>
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : notifications.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <Bell className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-muted-foreground">Huna arifa</p>
        </div>
      ) : (
        <div className="space-y-3">
          {notifications.map((notif) => (
            <div
              key={notif.id}
              className={`card-surface p-5 ${!notif.is_read ? "border-l-4 border-l-primary" : ""}`}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <Bell className={`h-4 w-4 ${!notif.is_read ? "text-primary" : "text-muted-foreground"}`} />
                    <h3 className="font-semibold">{notif.title}</h3>
                    {!notif.is_read && (
                      <span className="chip bg-primary text-primary-foreground text-[9px] font-bold px-2 py-0.5 rounded">
                        MPYA
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-foreground/80">{notif.message}</p>
                  <p className="text-xs text-muted-foreground mt-2">
                    {new Date(notif.created_at).toLocaleDateString("sw-TZ", {
                      year: "numeric",
                      month: "long",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </p>
                </div>
                {!notif.is_read && (
                  <button
                    onClick={() => markReadMutation.mutate([notif.id])}
                    className="inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs font-medium hover:bg-muted"
                  >
                    <Check className="h-3 w-3" />
                    Soma
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </AppShell>
  );
}
