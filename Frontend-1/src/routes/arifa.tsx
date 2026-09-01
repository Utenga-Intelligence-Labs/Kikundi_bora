import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { api } from "@/api/client";
import { AppShell } from "@/components/AppShell";
import { AppModal, useAppModal } from "@/components/AppModal";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Bell, Check, Loader2, Banknote, PiggyBank, Megaphone,
  AlertCircle, CheckCircle, Info, Wallet, ArrowRight
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

const listKey = ["notifications"] as const;
const badgeKey = ["notifications", "unread"] as const;

function getNotifIcon(type: string) {
  switch (type) {
    case "LOAN_REQUEST":
    case "LOAN_APPROVED":
    case "LOAN_REJECTED":
    case "LOAN_DISBURSED":
    case "LOAN_UNDER_REVIEW":
      return Banknote;
    case "CONTRIBUTION":
    case "CONTRIBUTION_DUE":
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
    case "CONTRIBUTION_DUE":
      return "bg-amber-100 text-amber-700";
    case "CONTRIBUTION":
      return "bg-purple-100 text-purple-700";
    case "SYSTEM":
      return "bg-primary/10 text-primary";
    default:
      return "bg-muted text-foreground";
  }
}

/** Related page a notification points to (for the modal "Nenda" action). */
function linkForType(type: string): string | null {
  if (type.startsWith("LOAN")) return "/mikopo";
  if (type === "CONTRIBUTION" || type === "CONTRIBUTION_DUE") return "/weka-mchango";
  if (type.startsWith("WELFARE")) return "/mfuko-kijamii";
  return null;
}

function ArifaPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { showModal } = useAppModal();
  const [selected, setSelected] = useState<Notification | null>(null);

  const { data, isLoading } = useQuery<{ data: Notification[]; total: number; unread: number }>({
    queryKey: listKey,
    queryFn: async () => {
      return api.get("/notifications");
    },
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  // Optimistic read: update the list cache AND the header badge cache
  // (single source of truth — the badge is derived from the same server
  // count and both caches are invalidated together).
  const applyOptimisticRead = (ids: string[]) => {
    const prev = qc.getQueryData(listKey);
    qc.setQueryData(listKey, (old: { data: Notification[]; unread: number } | undefined) => {
      if (!old) return old;
      const nowIso = new Date().toISOString();
      const notifData = old.data.map((n) =>
        (ids.length === 0 || ids.includes(n.id)) && !n.read_at
          ? { ...n, read_at: nowIso }
          : n
      );
      const unread = notifData.filter((n) => !n.read_at).length;
      return { ...old, data: notifData, unread };
    });
    const updated = qc.getQueryData<{ unread: number } | undefined>(listKey);
    qc.setQueryData(badgeKey, { unread: updated?.unread ?? 0 });
    return prev;
  };

  const markReadMutation = useMutation({
    mutationFn: async (vars: { ids?: string[]; all?: boolean }) => {
      if (vars.all) return api.post("/notifications/read-all");
      return api.post("/notifications/read", { ids: vars.ids });
    },
    onMutate: (vars) => applyOptimisticRead(vars.all ? [] : vars.ids ?? []),
    onError: (_err, _vars, ctx) => {
      // Roll back the optimistic update, then refetch server truth
      if (ctx) qc.setQueryData(listKey, ctx);
      qc.invalidateQueries({ queryKey: ["notifications"] });
      showModal({
        title: "Hitilafu",
        message: "Imeshindikana kusoma arifa. Jaribu tena.",
        variant: "error",
        primaryLabel: "Sawa",
      });
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  if (!user) return null;

  const notifications = data?.data ?? [];
  const unreadCount = data?.unread ?? 0;

  // The selected notification, kept in sync with the (optimistically
  // updated) list so the modal reflects the current read state.
  const selectedLive = selected
    ? notifications.find((n) => n.id === selected.id) ?? selected
    : null;
  const selectedLink = selectedLive ? linkForType(selectedLive.type) : null;

  const openDetail = (notif: Notification) => {
    setSelected(notif);
    // Standard notification UX: opening an unread notification marks it
    // read immediately (optimistic, rolled back on failure).
    if (!notif.read_at) markReadMutation.mutate({ ids: [notif.id] });
  };

  const formatTs = (iso: string) =>
    new Date(iso).toLocaleDateString("sw-TZ", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });

  return (
    <AppShell
      title="Arifa Zangu"
      subtitle={unreadCount > 0 ? `Una arifa ${unreadCount} mpya` : "Angalia arifa zako"}
      action={
        unreadCount > 0 ? (
          <button
            onClick={() => markReadMutation.mutate({ all: true })}
            disabled={markReadMutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {markReadMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Check className="h-4 w-4" />
            )}
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
            const isRowPending =
              markReadMutation.isPending &&
              (markReadMutation.variables?.ids?.includes(notif.id) || !!markReadMutation.variables?.all);

            return (
              <div
                key={notif.id}
                role="button"
                tabIndex={0}
                data-testid={`notif-card-${notif.id}`}
                data-unread={isUnread}
                onClick={() => openDetail(notif)}
                onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && openDetail(notif)}
                className={`card-surface p-4 transition-colors cursor-pointer hover:border-primary/40 ${
                  isUnread ? "border-l-4 border-l-primary bg-primary/5" : "opacity-70"
                }`}
              >
                <div className="flex items-start gap-3">
                  <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-xl ${colorClass}`}>
                    <Icon className="h-5 w-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <h3 className={`text-sm ${isUnread ? "font-bold" : "font-medium text-muted-foreground"}`}>
                          {notif.title}
                        </h3>
                        <p className={`text-sm mt-1 line-clamp-2 ${isUnread ? "text-foreground/80" : "text-muted-foreground"}`}>
                          {notif.message}
                        </p>
                        <p className="text-xs text-muted-foreground mt-2">
                          {formatTs(notif.created_at)}
                        </p>
                      </div>
                      {isUnread && (
                        <button
                          data-testid={`mark-read-${notif.id}`}
                          onClick={(e) => {
                            e.stopPropagation();
                            markReadMutation.mutate({ ids: [notif.id] });
                          }}
                          disabled={isRowPending}
                          aria-label="Soma"
                          title="Soma"
                          className="shrink-0 inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs font-medium hover:bg-muted disabled:opacity-50"
                        >
                          {isRowPending ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <Check className="h-3 w-3" />
                          )}
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

      {/* Detail modal */}
      <AppModal
        open={!!selectedLive}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
        title={selectedLive?.title ?? ""}
        message={selectedLive?.message}
        variant="info"
        primaryLabel={
          selectedLive && !selectedLive.read_at ? (
            <span className="inline-flex items-center gap-1.5">
              <Loader2 className="h-4 w-4 animate-spin" /> Inasoma...
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5">
              <Check className="h-4 w-4" /> Imesomwa
            </span>
          )
        }
        onPrimary={() => {
          if (selectedLive && !selectedLive.read_at) {
            markReadMutation.mutate({ ids: [selectedLive.id] });
          }
        }}
        secondaryLabel={selectedLink ? "Nenda" : undefined}
        onSecondary={() => {
          if (selectedLink) navigate({ to: selectedLink as never });
        }}
      >
        {selectedLive && (
          <div className="space-y-2 text-xs text-muted-foreground">
            <p>
              Muda: <span className="font-medium text-foreground">{formatTs(selectedLive.created_at)}</span>
            </p>
            <p>
              Hali:{" "}
              {selectedLive.read_at ? (
                <span className="chip bg-success/15 text-success inline-flex items-center gap-1">
                  <CheckCircle className="h-3 w-3" /> Imesomwa
                </span>
              ) : (
                <span className="chip bg-amber-100 text-amber-700 inline-flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" /> Haijasomwa
                </span>
              )}
            </p>
            {selectedLink && (
              <p className="flex items-center gap-1 text-primary">
                <ArrowRight className="h-3 w-3" /> Endelea kwenye ukurasa husika kwa kutumia "Nenda"
              </p>
            )}
          </div>
        )}
      </AppModal>
    </AppShell>
  );
}
