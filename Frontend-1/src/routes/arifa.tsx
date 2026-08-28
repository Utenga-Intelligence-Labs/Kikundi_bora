import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { tokenStorage } from "@/lib/auth-storage";
import { AppShell } from "@/components/AppShell";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Bell, Check, Loader2, Banknote, PiggyBank, Megaphone,
  AlertCircle, CheckCircle, XCircle, Info, Wallet
} from "lucide-react";

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
  read_at?: string;
  created_at: string;
}

function getNotifIcon(type: string) {
  switch (type) {
    case "LOAN_REQUEST":
    case "LOAN_APPROVED":
    case "LOAN_REJECTED":
    case "LOAN_DISBURSED":
    case "LOAN_UNDER_REVIEW":
      return Banknote;
    case "CONTRIBUTION":
      return PiggyBank;
    case "WELFARE_CREATED":
    case "WELFARE_APPROVED":
    case "WELFARE_REJECTED":
    case "WELFARE_PAYMENT":
    case "WELFARE_COMPLETED":
      return Wallet;
    case "SYSTEM":
      return Megaphone;
    default:
      return Bell;
  }
}

function getNotifColor(type: string) {
  switch (type) {
    case "LOAN_APPROVED":
    case "WELFARE_APPROVED":
      return "bg-green-100 text-green-700";
    case "LOAN_REJECTED":
    case "WELFARE_REJECTED":
      return "bg-red-100 text-red-700";
    case "LOAN_DISBURSED":
    case "WELFARE_PAYMENT":
      return "bg-blue-100 text-blue-700";
    case "LOAN_REQUEST":
    case "LOAN_UNDER_REVIEW":
      return "bg-amber-100 text-amber-700";
    case "CONTRIBUTION":
      return "bg-purple-100 text-purple-700";
    case "SYSTEM":
      return "bg-primary/10 text-primary";
    default:
      return "bg-muted text-foreground";
  }
}

function ArifaPage() {
  const { user } = useAuth();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery<{ data: Notification[]; total: number; unread: number }>({
    queryKey: ["notifications"],
    queryFn: async () => {
      const token = tokenStorage.get();
      const res = await fetch("/api/v1/notifications", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata arifa");
      return res.json();
    },
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  const markReadMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      const token = tokenStorage.get();
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
  const unreadCount = data?.unread ?? 0;

  const markAllAsRead = () => {
    const unreadIds = notifications.filter((n) => !n.read_at).map((n) => n.id);
    if (unreadIds.length > 0) {
      markReadMutation.mutate(unreadIds);
    }
  };

  return (
    <AppShell
      title="Arifa Zangu"
      subtitle={unreadCount > 0 ? `Una arifa ${unreadCount} mpya` : "Angalia arifa zako"}
      action={
        unreadCount > 0 ? (
          <button
            onClick={markAllAsRead}
            disabled={markReadMutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Check className="h-4 w-4" />
            Soma Zote
          </button>
        ) : undefined
      }
    >
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : notifications.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <Bell className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-muted-foreground">Huna arifa bado</p>
          <p className="text-sm text-muted-foreground mt-1">Arifa zitaonekana hapa unapopata taarifa mpya</p>
        </div>
      ) : (
        <div className="space-y-3">
          {notifications.map((notif) => {
            const isUnread = !notif.read_at;
            const Icon = getNotifIcon(notif.type);
            const colorClass = getNotifColor(notif.type);

            return (
              <div
                key={notif.id}
                className={`card-surface p-4 transition-colors ${
                  isUnread ? "border-l-4 border-l-primary bg-primary/5" : ""
                }`}
              >
                <div className="flex items-start gap-3">
                  <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-xl ${colorClass}`}>
                    <Icon className="h-5 w-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <h3 className="font-semibold text-sm">{notif.title}</h3>
                        <p className="text-sm text-foreground/80 mt-1">{notif.message}</p>
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
                      {isUnread && (
                        <button
                          onClick={() => markReadMutation.mutate([notif.id])}
                          className="shrink-0 inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs font-medium hover:bg-muted"
                        >
                          <Check className="h-3 w-3" />
                          Soma
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </AppShell>
  );
}
