import { createFileRoute, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { usePendingActions, useApprovePendingAction, useRejectPendingAction } from "@/hooks/use-pending-actions";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, blockAdminFromPage } from "@/lib/role-guards";
import { tokenStorage } from "@/lib/auth-storage";
import type { PendingAction } from "@/api/types";
import { Clock, CheckCircle2, XCircle, ChevronLeft, ChevronRight, FileText } from "lucide-react";

export const Route = createFileRoute("/vitendo-vinavyosubiri")({
  head: () => ({
    meta: [
      { title: "Vitendo Vinavyosubiri — Kikundi" },
      { name: "description", content: "Vitendo vinavyohitaji idhinisho lako." },
    ],
  }),
  beforeLoad: () => {
    if (typeof window !== "undefined" && !tokenStorage.exists()) {
      throw redirect({ to: "/ingia" });
    }
    blockAdminFromPage();
  },
  component: PendingActionsPage,
});

const actionTypeLabels: Record<string, string> = {
  CONTRIBUTION_EDIT: "Kubadilisha Mchango",
  WELFARE_CREATE: "Kuunda Mfuko wa Kijamii",
  LOAN_DISBURSE: "Kutoa Mkopo",
};

function PendingActionsPage() {
  const { user } = useAuth();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("PENDING");
  const { data, isLoading } = usePendingActions({ page, limit: 20, status: statusFilter || undefined });
  const approveMutation = useApprovePendingAction();
  const rejectMutation = useRejectPendingAction();

  const [selectedAction, setSelectedAction] = useState<PendingAction | null>(null);
  const [actionType, setActionType] = useState<"approve" | "reject" | null>(null);
  const [remarks, setRemarks] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  if (!hasRole(user, "chair")) {
    return (
      <AppShell title="Vitendo Vinavyosubiri">
        <div className="flex items-center justify-center py-20">
          <p className="text-muted-foreground">Huna ruhusa ya kuona ukurasa huu.</p>
        </div>
      </AppShell>
    );
  }

  const handleAction = async () => {
    if (!selectedAction || !actionType) return;
    setActionLoading(true);
    try {
      if (actionType === "approve") {
        await approveMutation.mutateAsync({ id: selectedAction.id, remarks });
      } else {
        await rejectMutation.mutateAsync({ id: selectedAction.id, remarks });
      }
      setSelectedAction(null);
      setActionType(null);
      setRemarks("");
    } catch {
      // handled by mutation
    } finally {
      setActionLoading(false);
    }
  };

  const actions = data?.data ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  return (
    <AppShell title="Vitendo Vinavyosubiri" subtitle="Vitendo vinavyohitaji idhinisho lako">
      {/* Status filter */}
      <div className="mb-4 flex gap-2">
        {(["PENDING", "APPROVED", "REJECTED"] as const).map((s) => (
          <button
            key={s}
            onClick={() => { setStatusFilter(s); setPage(1); }}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium ${
              statusFilter === s ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:text-foreground"
            }`}
          >
            {s === "PENDING" ? "Inasubiri" : s === "APPROVED" ? "Imeidhinishwa" : "Imekataliwa"}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="card-surface animate-pulse px-4 py-4">
              <div className="h-4 w-1/3 rounded bg-muted" />
            </div>
          ))}
        </div>
      ) : actions.length === 0 ? (
        <div className="card-surface flex flex-col items-center px-4 py-12 text-center">
          <CheckCircle2 className="mb-3 h-10 w-10 text-success" />
          <p className="text-sm font-medium">Hakuna vitendo vinavyosubiri.</p>
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {actions.map((a) => (
            <div key={a.id} className="flex items-center justify-between px-4 py-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <FileText className="h-4 w-4 text-muted-foreground" />
                  <p className="text-sm font-medium">{actionTypeLabels[a.action_type] ?? a.action_type}</p>
                  <span className={`chip text-[10px] ${
                    a.status === "PENDING" ? "bg-amber-100 text-amber-700" :
                    a.status === "APPROVED" ? "bg-success/15 text-success" :
                    "bg-destructive/10 text-destructive"
                  }`}>
                    {a.status === "PENDING" ? "Inasubiri" : a.status === "APPROVED" ? "Imeidhinishwa" : "Imekataliwa"}
                  </span>
                </div>
                <div className="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                  {a.requester && <span>Na: {a.requester.name}</span>}
                  <span className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {new Date(a.created_at).toLocaleString("sw-TZ")}
                  </span>
                </div>
                {a.remarks && (
                  <p className="mt-1 text-xs text-muted-foreground">Maoni: {a.remarks}</p>
                )}
              </div>
              {a.status === "PENDING" && (
                <div className="flex gap-2">
                  <button
                    onClick={() => { setSelectedAction(a); setActionType("approve"); }}
                    className="rounded-lg bg-success/10 px-3 py-1.5 text-xs font-medium text-success hover:bg-success/20"
                  >
                    <CheckCircle2 className="inline h-3.5 w-3.5 mr-1" />
                    Idhinisha
                  </button>
                  <button
                    onClick={() => { setSelectedAction(a); setActionType("reject"); }}
                    className="rounded-lg bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive hover:bg-destructive/20"
                  >
                    <XCircle className="inline h-3.5 w-3.5 mr-1" />
                    Kataa
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
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

      {/* Approve/Reject Modal */}
      {selectedAction && actionType && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={() => { setSelectedAction(null); setActionType(null); }}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-lg font-bold">
              {actionType === "approve" ? "Idhinisha Kitendo" : "Kataa Kitendo"}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {actionTypeLabels[selectedAction.action_type] ?? selectedAction.action_type}
            </p>
            <div className="mt-4">
              <textarea
                value={remarks}
                onChange={(e) => setRemarks(e.target.value)}
                placeholder={actionType === "approve" ? "Maoni (si lazima)" : "Sababu ya kukataa..."}
                className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-ring/20"
                rows={3}
              />
            </div>
            <div className="mt-4 flex gap-3">
              <button
                onClick={() => { setSelectedAction(null); setActionType(null); setRemarks(""); }}
                className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted"
              >
                Ghairi
              </button>
              <button
                onClick={handleAction}
                disabled={actionLoading}
                className={`flex-1 rounded-xl py-2.5 text-sm font-semibold text-white disabled:opacity-60 ${
                  actionType === "approve" ? "bg-success" : "bg-destructive"
                }`}
              >
                {actionLoading ? "Inashughulikiwa..." : actionType === "approve" ? "Idhinisha" : "Kataa"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
