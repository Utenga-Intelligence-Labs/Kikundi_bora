import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireRole } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCircle, XCircle, Clock, Loader2, Eye, ImageIcon, X, AlertTriangle, UserCheck } from "lucide-react";

export const Route = createFileRoute("/michango-inayosubiri")({
  beforeLoad: () => {
    requireRole("chair", "treasurer", "secretary");
  },
  component: TaarifaWanaosubiriPage,
});

interface MemberRow {
  member_id: string;
  full_name: string;
  member_no: string;
  phone: string;
  status: string;
  amount: number;
  period_label: string;
  contribution_type: string;
  contribution_id: string;
  proof_image_url: string;
  proof_message: string;
  submitted_at: string;
}

const STATUS_CONFIG: Record<string, { label: string; color: string; icon: any }> = {
  PENDING_VERIFICATION: { label: "Inasubiri", color: "bg-warning/25 text-foreground", icon: Clock },
  CONFIRMED: { label: "Imethibitishwa", color: "bg-success/25 text-success", icon: CheckCircle },
  REJECTED: { label: "Imekataliwa", color: "bg-destructive/25 text-destructive", icon: XCircle },
  HAJACHANGIA: { label: "Hajachangia", color: "bg-muted text-muted-foreground", icon: AlertTriangle },
};

function TaarifaWanaosubiriPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [viewing, setViewing] = useState<MemberRow | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [showRejectInput, setShowRejectInput] = useState(false);

  const isHazina = user?.role === "treasurer";

  const { data, isLoading } = useQuery<{ data: MemberRow[]; total: number }>({
    queryKey: ["michango", "members-summary"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/michango/members-summary", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata taarifa");
      return res.json();
    },
  });

  const confirmMutation = useMutation({
    mutationFn: async (id: string) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch(`/api/v1/michango/${id}/confirm`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error((await res.json()).message || "Imeshindikana kuthibitisha");
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      setViewing(null);
    },
    onError: (err: Error) => alert(err.message),
  });

  const rejectMutation = useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason: string }) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch(`/api/v1/michango/${id}/reject`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ reason }),
      });
      if (!res.ok) throw new Error((await res.json()).message || "Imeshindikana kukataa");
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      setViewing(null);
      setRejectReason("");
      setShowRejectInput(false);
    },
    onError: (err: Error) => alert(err.message),
  });

  if (!user) return null;

  const members = data?.data ?? [];

  return (
    <AppShell title="Taarifa za Wanaosubiri" subtitle={isHazina ? "Thibitisha au kataa michango ya wanachama" : "Angalia hali ya michango ya wanachama"}>
      {isLoading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      ) : members.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <UserCheck className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-muted-foreground">Hakuna wanachama wanaosubiri</p>
        </div>
      ) : (
        <div className="space-y-3">
          {members.map((m) => {
            const cfg = STATUS_CONFIG[m.status] || STATUS_CONFIG.HAJACHANGIA;
            const Icon = cfg.icon;
            const canAct = isHazina && m.status === "PENDING_VERIFICATION";

            return (
              <div key={m.member_id} className="card-surface p-4">
                <div className="flex items-center gap-3">
                  <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-muted font-bold text-sm">
                    {m.full_name.charAt(0)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="font-semibold text-sm">{m.full_name}</p>
                      <span className="text-xs text-muted-foreground">{m.member_no}</span>
                    </div>
                    <span className={`chip mt-1 inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded ${cfg.color}`}>
                      <Icon className="h-3 w-3" />
                      {cfg.label}
                    </span>
                    {m.amount > 0 && (
                      <span className="ml-2 text-sm font-bold">TZS {m.amount.toLocaleString()}</span>
                    )}
                  </div>

                  {m.proof_image_url && (
                    <button onClick={() => setViewing(m)} className="shrink-0 h-12 w-12 overflow-hidden rounded-lg border bg-muted">
                      <img src={m.proof_image_url} alt="Proof" className="h-full w-full object-cover" />
                    </button>
                  )}

                  {canAct && (
                    <div className="flex gap-2 shrink-0">
                      <button
                        onClick={() => confirmMutation.mutate(m.contribution_id)}
                        disabled={confirmMutation.isPending}
                        className="rounded-lg bg-success px-3 py-2 text-xs font-semibold text-success-foreground hover:bg-success/90"
                      >
                        {confirmMutation.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : "Thibitisha"}
                      </button>
                      <button
                        onClick={() => { setViewing(m); setShowRejectInput(true); }}
                        className="rounded-lg bg-destructive px-3 py-2 text-xs font-semibold text-destructive-foreground hover:bg-destructive/90"
                      >
                        Kataa
                      </button>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Detail Modal */}
      {viewing && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center sm:p-4" onClick={() => { setViewing(null); setRejectReason(""); setShowRejectInput(false); }}>
          <div className="w-full max-w-lg rounded-t-3xl bg-card sm:rounded-2xl max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-card px-5 py-4">
              <h3 className="font-display text-lg font-semibold">{viewing.full_name}</h3>
              <button onClick={() => { setViewing(null); setRejectReason(""); setShowRejectInput(false); }} className="rounded-lg p-1.5 hover:bg-muted"><X className="h-5 w-5" /></button>
            </div>
            <div className="p-5 space-y-4">
              {viewing.proof_image_url && (
                <div className="rounded-xl overflow-hidden border bg-muted">
                  <img src={viewing.proof_image_url} alt="Proof" className="w-full max-h-80 object-contain" />
                </div>
              )}
              <div className="grid grid-cols-2 gap-4">
                <div><p className="text-xs text-muted-foreground">Mwanachama</p><p className="font-semibold">{viewing.full_name}</p><p className="text-xs text-muted-foreground">{viewing.member_no}</p></div>
                <div><p className="text-xs text-muted-foreground">Kiasi</p><p className="font-display text-2xl font-bold">TZS {viewing.amount.toLocaleString()}</p></div>
                <div><p className="text-xs text-muted-foreground">Kipindi</p><p className="font-semibold">{viewing.period_label || "—"}</p></div>
                <div><p className="text-xs text-muted-foreground">Hali</p><span className={`chip inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded ${STATUS_CONFIG[viewing.status]?.color}`}>{STATUS_CONFIG[viewing.status]?.label}</span></div>
              </div>
              {viewing.proof_message && <div className="p-3 bg-muted/50 rounded-lg"><p className="text-xs text-muted-foreground mb-1">Ujumbe</p><p className="text-sm">{viewing.proof_message}</p></div>}
              {showRejectInput && isHazina && (
                <div className="p-4 bg-destructive/5 border border-destructive/20 rounded-lg space-y-3">
                  <label className="block text-sm font-medium text-destructive">Sababu ya kukataa</label>
                  <textarea value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} placeholder="Andika sababu..." rows={3} className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
                  <div className="flex gap-2">
                    <button onClick={() => { setShowRejectInput(false); setRejectReason(""); }} className="flex-1 rounded-lg border py-2 text-sm">Ghairi</button>
                    <button onClick={() => rejectMutation.mutate({ id: viewing.contribution_id, reason: rejectReason })} disabled={rejectMutation.isPending || !rejectReason.trim()} className="flex-1 rounded-lg bg-destructive py-2 text-sm font-semibold text-destructive-foreground">Kataa</button>
                  </div>
                </div>
              )}
              <button onClick={() => { setViewing(null); setRejectReason(""); setShowRejectInput(false); }} className="w-full rounded-xl border py-3 text-sm">Funga</button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
