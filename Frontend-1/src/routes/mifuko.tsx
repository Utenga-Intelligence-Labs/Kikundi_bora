import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { api } from "@/api/client";
import { AppShell } from "@/components/AppShell";
import { AppModal, useAppModal } from "@/components/AppModal";
import { groupsApi } from "@/api/groups";
import {
  socialFundsApi,
  type SocialFund,
  type SocialFundStatus,
} from "@/api/social-funds";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Layers, Plus, Loader2, Check, X, ChevronRight, Target, Landmark,
} from "lucide-react";

export const Route = createFileRoute("/mifuko")({
  head: () => ({
    meta: [
      { title: "Mifuko — Money Seeking" },
      { name: "description", content: "Mifuko ya kijamii ya kikundi na salio zake." },
    ],
  }),
  beforeLoad: () => requireAuth(),
  component: MifukoPage,
});

const statusChip: Record<SocialFundStatus, { label: string; cls: string }> = {
  PENDING_APPROVAL: { label: "Inasubiri Katibu", cls: "bg-amber-100 text-amber-700" },
  ACTIVE: { label: "Inatumika", cls: "bg-success/15 text-success" },
  REJECTED: { label: "Imekataliwa", cls: "bg-destructive/10 text-destructive" },
  CLOSED: { label: "Imefungwa", cls: "bg-muted text-muted-foreground" },
};

function MifukoPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const isChair = user?.role === "chair";
  const isKatibu = user?.role === "secretary";

  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const groupId = gs?.data.id;

  const { data, isLoading } = useQuery({
    queryKey: ["social-funds", groupId],
    queryFn: () => socialFundsApi.list(groupId!),
    enabled: !!groupId,
  });
  const funds: SocialFund[] = data?.data ?? [];
  const pendingFunds = funds.filter((f) => f.status === "PENDING_APPROVAL");
  const activeFunds = funds.filter((f) => f.status === "ACTIVE");
  const otherFunds = funds.filter(
    (f) => f.status !== "ACTIVE" && f.status !== "PENDING_APPROVAL"
  );

  // ---- create form (mwenyekiti) ----
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: "", description: "", target_amount: "" });
  const createMutation = useMutation({
    mutationFn: () =>
      socialFundsApi.create(groupId!, {
        name: form.name,
        description: form.description,
        target_amount: form.target_amount ? parseFloat(form.target_amount) : undefined,
      }),
    onSuccess: () => {
      showModal({
        title: "Imefanikiwa",
        message: "Mfuko umetundwa. Unasubiri idhini ya Katibu.",
        variant: "success",
        primaryLabel: "Sawa",
      });
      setForm({ name: "", description: "", target_amount: "" });
      setShowCreate(false);
      qc.invalidateQueries({ queryKey: ["social-funds"] });
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  // ---- katibu: approve / reject pending ----
  const approveMutation = useMutation({
    mutationFn: (fundId: string) => socialFundsApi.approve(groupId!, fundId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["social-funds"] });
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });
  const [rejecting, setRejecting] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const rejectMutation = useMutation({
    mutationFn: (vars: { fundId: string; reason: string }) =>
      socialFundsApi.rejectFund(groupId!, vars.fundId, vars.reason),
    onSuccess: () => {
      setRejecting(null);
      setRejectReason("");
      qc.invalidateQueries({ queryKey: ["social-funds"] });
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const renderFundCard = (f: SocialFund, showActions: boolean) => (
    <div key={f.id} className="card-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2 mb-1.5 flex-wrap">
            <h3 className="font-display text-lg font-semibold truncate">{f.name}</h3>
            <span className={`chip text-[10px] px-2 py-0.5 rounded ${statusChip[f.status].cls}`}>
              {statusChip[f.status].label}
            </span>
          </div>
          {f.description && (
            <p className="text-sm text-muted-foreground line-clamp-2">{f.description}</p>
          )}
        </div>
        <Link
          to="/mifuko/$fundId"
          params={{ fundId: f.id }}
          className="shrink-0 inline-flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs font-medium hover:bg-muted"
        >
          Fungua <ChevronRight className="h-3 w-3" />
        </Link>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
        <div>
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <Landmark className="h-3 w-3" /> Salio
          </p>
          <p className="font-display text-lg font-bold text-success">
            TZS {Number(f.current_balance).toLocaleString()}
          </p>
        </div>
        {f.target_amount && (
          <div>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Target className="h-3 w-3" /> Lengo
            </p>
            <p className="font-display text-lg font-bold">
              TZS {Number(f.target_amount).toLocaleString()}
            </p>
          </div>
        )}
      </div>
      {f.target_amount && Number(f.target_amount) > 0 && (
        <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-muted">
          <div
            className="h-full bg-success transition-all"
            style={{
              width: `${Math.min(100, (Number(f.current_balance) / Number(f.target_amount)) * 100)}%`,
            }}
          />
        </div>
      )}
      {showActions && (
        <div className="mt-3 flex gap-2">
          <button
            onClick={() => approveMutation.mutate(f.id)}
            disabled={approveMutation.isPending}
            className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-success px-3 py-2 text-xs font-semibold text-white disabled:opacity-50"
          >
            <Check className="h-3.5 w-3.5" /> Idhinisha
          </button>
          <button
            onClick={() => setRejecting(f.id)}
            className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-destructive/10 px-3 py-2 text-xs font-semibold text-destructive"
          >
            <X className="h-3.5 w-3.5" /> Kataa
          </button>
        </div>
      )}
      {f.rejection_reason && (
        <p className="mt-2 text-xs text-destructive">Sababu: {f.rejection_reason}</p>
      )}
    </div>
  );

  return (
    <AppShell
      title="Mifuko ya Kijamii"
      subtitle="Voshiria vya kikundi — msiba, harusi, dharura na vingine"
      action={
        isChair && !showCreate ? (
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground"
          >
            <Plus className="h-4 w-4" /> Unda Mfuko
          </button>
        ) : undefined
      }
    >
      {/* Create form — mwenyekiti */}
      {showCreate && isChair && (
        <div className="card-surface p-5 space-y-3 mb-6">
          <div className="flex items-center justify-between">
            <h3 className="font-display text-lg font-semibold">Unda Mfuko Mpya</h3>
            <button onClick={() => setShowCreate(false)} className="rounded-lg p-1 hover:bg-muted" aria-label="Funga">
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="text-xs text-muted-foreground">
            Mfuko utaanza kutumika baada ya Katibu kuidhinisha.
          </p>
          <div>
            <label className="block text-xs text-muted-foreground mb-1">Jina la mfuko *</label>
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Mfuko wa Msiba"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-muted-foreground mb-1">Maelezo</label>
            <textarea
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              placeholder="Maelezo ya matumizi ya mfuko huu..."
              rows={2}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-muted-foreground mb-1">Kiasi lengwa (TZS — hiari)</label>
            <input
              type="number"
              min="0"
              value={form.target_amount}
              onChange={(e) => setForm({ ...form, target_amount: e.target.value })}
              placeholder="1000000"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
            />
          </div>
          <button
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending || !form.name.trim()}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground disabled:opacity-50"
          >
            {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            Tuma kwa Katibu kwa idhini
          </button>
        </div>
      )}

      {/* Katibu: pending approvals */}
      {isKatibu && pendingFunds.length > 0 && (
        <>
          <div className="mb-3 flex items-center gap-2">
            <h2 className="font-display text-base font-semibold">Zinasubiri Idhini Yako</h2>
            <span className="chip bg-amber-100 text-amber-700 text-[10px] px-2 py-0.5 rounded">
              {pendingFunds.length}
            </span>
          </div>
          <div className="space-y-3">
            {pendingFunds.map((f) => renderFundCard(f, true))}
          </div>
        </>
      )}

      {/* Active funds */}
      <div className="mt-2 mb-3 flex items-center gap-2">
        <h2 className="font-display text-base font-semibold flex items-center gap-1.5">
          <Layers className="h-4 w-4 text-primary" /> Mifuko Inayotumika
        </h2>
        <span className="chip bg-primary/10 text-primary text-[10px] px-2 py-0.5 rounded">
          {activeFunds.length}
        </span>
      </div>
      {isLoading ? (
        <div className="flex justify-center py-10">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : activeFunds.length === 0 ? (
        <div className="card-surface p-10 text-center">
          <Layers className="mx-auto h-10 w-10 text-muted-foreground/50" />
          <p className="mt-3 text-muted-foreground">Hakuna mifuko inayotumika kwa sasa</p>
          {isChair && (
            <p className="text-sm text-muted-foreground mt-1">Bofya "Unda Mfuko" kuanza</p>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {activeFunds.map((f) => renderFundCard(f, false))}
        </div>
      )}

      {/* Closed / rejected (history) */}
      {otherFunds.length > 0 && (
        <>
          <div className="mt-7 mb-3">
            <h2 className="font-display text-base font-semibold">Historia</h2>
          </div>
          <div className="space-y-3 opacity-80">
            {otherFunds.map((f) => renderFundCard(f, false))}
          </div>
        </>
      )}

      {/* Reject reason modal (katibu) */}
      <AppModal
        open={!!rejecting}
        onOpenChange={(open) => {
          if (!open) setRejecting(null);
        }}
        title="Kataa Mfuko"
        message="Eleza sababu ya kukataa mfuko huu."
        variant="warning"
        primaryLabel="Thibitisha Kukataa"
        onPrimary={() => {
          if (rejecting && rejectReason.trim()) {
            rejectMutation.mutate({ fundId: rejecting, reason: rejectReason.trim() });
          }
        }}
      >
        <textarea
          value={rejectReason}
          onChange={(e) => setRejectReason(e.target.value)}
          placeholder="Sababu..."
          rows={2}
          className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
        />
      </AppModal>
    </AppShell>
  );
}
