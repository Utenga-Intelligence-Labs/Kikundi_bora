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
  type SocialFundContribution,
} from "@/api/social-funds";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft, Loader2, Landmark, Target, HandCoins, Check, X, Plus,
} from "lucide-react";

export const Route = createFileRoute("/mifuko/$fundId")({
  beforeLoad: () => requireAuth(),
  component: FundDetailPage,
});

const statusChip: Record<string, { label: string; cls: string }> = {
  PENDING_APPROVAL: { label: "Inasubiri Katibu", cls: "bg-amber-100 text-amber-700" },
  ACTIVE: { label: "Inatumika", cls: "bg-success/15 text-success" },
  REJECTED: { label: "Imekataliwa", cls: "bg-destructive/10 text-destructive" },
  CLOSED: { label: "Imefungwa", cls: "bg-muted text-muted-foreground" },
};

function FundDetailPage() {
  const { user } = useAuth();
  const { fundId } = Route.useParams();
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const isHazina = user?.role === "treasurer";
  const isChair = user?.role === "chair";

  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const groupId = gs?.data.id;

  const { data, isLoading, error } = useQuery({
    queryKey: ["social-fund", groupId, fundId],
    queryFn: () => socialFundsApi.detail(groupId!, fundId),
    enabled: !!groupId,
  });
  const fund = data?.data;
  const contributions: SocialFundContribution[] = data?.contributions ?? [];

  const [showContribute, setShowContribute] = useState(false);
  const [amount, setAmount] = useState("");
  const contributeMutation = useMutation({
    mutationFn: () =>
      socialFundsApi.contribute(groupId!, fundId, parseFloat(amount)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["social-fund", groupId, fundId] });
      setShowContribute(false);
      setAmount("");
      showModal({
        title: "Imefanikiwa",
        message: "Mchango umetumwa. Unasubiri uthibitisho wa Mweka Hazina.",
        variant: "success",
        primaryLabel: "Sawa",
      });
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const confirmMutation = useMutation({
    mutationFn: (cid: string) => socialFundsApi.confirm(groupId!, fundId, cid),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["social-fund", groupId, fundId] });
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });
  const [rejecting, setRejecting] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const rejectMutation = useMutation({
    mutationFn: (vars: { cid: string; reason: string }) =>
      socialFundsApi.rejectContribution(groupId!, fundId, vars.cid, vars.reason),
    onSuccess: () => {
      setRejecting(null);
      setRejectReason("");
      qc.invalidateQueries({ queryKey: ["social-fund", groupId, fundId] });
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const closeMutation = useMutation({
    mutationFn: () => socialFundsApi.close(groupId!, fundId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["social-fund", groupId, fundId] }),
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  if (isLoading) {
    return (
      <AppShell title="Mfuko">
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      </AppShell>
    );
  }

  if (error || !fund) {
    return (
      <AppShell title="Mfuko">
        <div className="card-surface p-10 text-center">
          <p className="text-muted-foreground">Mfuko haujapatikana.</p>
          <Link to="/mifuko" className="mt-3 inline-block text-sm font-semibold text-primary">
            ← Rudi kwenye Mifuko
          </Link>
        </div>
      </AppShell>
    );
  }

  const chip = statusChip[fund.status] ?? { label: fund.status, cls: "bg-muted text-foreground" };
  const pendingContribs = contributions.filter((c) => c.status === "PENDING");
  const history = contributions.filter((c) => c.status !== "PENDING");
  const targetPct =
    fund.target_amount && Number(fund.target_amount) > 0
      ? Math.min(100, (Number(fund.current_balance) / Number(fund.target_amount)) * 100)
      : null;
  const canContribute = fund.status === "ACTIVE" && !!user?.member_id;

  return (
    <AppShell
      title={fund.name}
      subtitle={fund.description || "Mfuko wa kijamii"}
      action={
        <Link
          to="/mifuko"
          className="inline-flex items-center gap-1.5 rounded-xl border border-border px-3.5 py-2 text-sm font-medium hover:bg-muted"
        >
          <ArrowLeft className="h-4 w-4" /> Rudi
        </Link>
      }
    >
      {/* Balance hero */}
      <div className="hero-surface px-5 py-6">
        <div className="flex items-center justify-between gap-3">
          <p className="text-xs font-medium uppercase tracking-wider text-primary-foreground/80 flex items-center gap-1.5">
            <Landmark className="h-3.5 w-3.5" /> Salio la mfuko
          </p>
          <span className={`chip text-[10px] px-2 py-0.5 rounded ${chip.cls}`}>{chip.label}</span>
        </div>
        <p className="mt-2 font-display text-4xl font-extrabold">
          TZS {Number(fund.current_balance).toLocaleString()}
        </p>
        {fund.target_amount && (
          <p className="mt-2 text-xs text-primary-foreground/80 flex items-center gap-1.5">
            <Target className="h-3 w-3" /> Lengo: TZS {Number(fund.target_amount).toLocaleString()}
          </p>
        )}
        {targetPct != null && (
          <div className="mt-3 h-2 overflow-hidden rounded-full bg-white/20">
            <div className="h-full bg-white transition-all" style={{ width: `${targetPct}%` }} />
          </div>
        )}
        {canContribute && (
          <button
            onClick={() => setShowContribute(true)}
            className="mt-4 inline-flex items-center gap-2 rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-primary shadow-lg transition-transform hover:scale-[1.01]"
          >
            <HandCoins className="h-4 w-4" /> Changia
          </button>
        )}
        {isChair && fund.status === "ACTIVE" && (
          <button
            onClick={() =>
              showModal({
                title: "Funga mfuko?",
                message: `Mfuko "${fund.name}" hautakubali michango zaidi baada ya kufungwa.`,
                variant: "warning",
                primaryLabel: "Funga Mfuko",
                onPrimary: () => closeMutation.mutate(),
              })
            }
            disabled={closeMutation.isPending}
            className="mt-4 ml-2 inline-flex items-center gap-2 rounded-xl border border-white/40 px-4 py-2.5 text-sm font-semibold text-white hover:bg-white/10 disabled:opacity-50"
          >
            <X className="h-4 w-4" /> Funga Mfuko
          </button>
        )}
      </div>

      {/* Hazina: pending contributions */}
      {isHazina && pendingContribs.length > 0 && fund.status === "ACTIVE" && (
        <>
          <div className="mt-6 mb-3 flex items-center gap-2">
            <h2 className="font-display text-base font-semibold">Zinasubiri Uthibitisho</h2>
            <span className="chip bg-amber-100 text-amber-700 text-[10px] px-2 py-0.5 rounded">
              {pendingContribs.length}
            </span>
          </div>
          <div className="space-y-2.5">
            {pendingContribs.map((c) => (
              <div key={c.id} className="card-surface flex items-center justify-between gap-3 p-4">
                <div className="min-w-0">
                  <p className="text-sm font-semibold">{c.member?.full_name ?? `Mwanachama #${c.member_id}`}</p>
                  <p className="text-xs text-muted-foreground">
                    TZS {Number(c.amount).toLocaleString()} · {new Date(c.created_at).toLocaleDateString("sw-TZ")}
                  </p>
                </div>
                <div className="flex gap-2 shrink-0">
                  <button
                    onClick={() => confirmMutation.mutate(c.id)}
                    disabled={confirmMutation.isPending}
                    className="inline-flex items-center gap-1 rounded-lg bg-success px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50"
                  >
                    {confirmMutation.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Check className="h-3 w-3" />}
                    Thibitisha
                  </button>
                  <button
                    onClick={() => setRejecting(c.id)}
                    className="inline-flex items-center gap-1 rounded-lg bg-destructive/10 px-3 py-1.5 text-xs font-semibold text-destructive"
                  >
                    <X className="h-3 w-3" /> Kataa
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {/* History */}
      <div className="mt-6 mb-3">
        <h2 className="font-display text-base font-semibold">Historia ya Michango</h2>
      </div>
      {history.length === 0 ? (
        <div className="card-surface p-8 text-center text-sm text-muted-foreground">
          Hakuna michango iliyothibitishwa bado.
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {history.map((c) => (
            <div key={c.id} className="flex items-center justify-between px-4 py-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{c.member?.full_name ?? `Mwanachama #${c.member_id}`}</p>
                <p className="text-xs text-muted-foreground">
                  {c.status === "CONFIRMED" && c.contributed_at
                    ? `Alichanga ${new Date(c.contributed_at).toLocaleDateString("sw-TZ")}`
                    : `Kilitumwa ${new Date(c.created_at).toLocaleDateString("sw-TZ")}`}
                  {c.rejection_reason && ` · Sababu: ${c.rejection_reason}`}
                </p>
              </div>
              <div className="text-right shrink-0">
                <p className={`text-sm font-semibold ${c.status === "CONFIRMED" ? "text-success" : "text-destructive"}`}>
                  TZS {Number(c.amount).toLocaleString()}
                </p>
                <span
                  className={`chip text-[10px] px-1.5 py-0.5 rounded ${
                    c.status === "CONFIRMED" ? "bg-success/15 text-success" : "bg-destructive/10 text-destructive"
                  }`}
                >
                  {c.status === "CONFIRMED" ? "Imethibitishwa" : "Imekataliwa"}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Contribute modal (members) */}
      <AppModal
        open={showContribute}
        onOpenChange={(open) => {
          if (!open) setShowContribute(false);
        }}
        title={`Changia kwenye ${fund.name}`}
        message="Weka kiasi unachotaka kuchanga. Mweka Hazina atathibitisha."
        variant="info"
        primaryLabel="Tuma Mchango"
        onPrimary={() => {
          if (parseFloat(amount) > 0) contributeMutation.mutate();
        }}
      >
        <input
          type="number"
          min="0"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="Kiasi (TZS)"
          className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm"
        />
      </AppModal>

      {/* Reject reason modal (hazina) */}
      <AppModal
        open={!!rejecting}
        onOpenChange={(open) => {
          if (!open) setRejecting(null);
        }}
        title="Kataa mchango"
        message="Eleza sababu ya kukataa mchango huu."
        variant="warning"
        primaryLabel="Thibitisha Kukataa"
        onPrimary={() => {
          if (rejecting && rejectReason.trim()) {
            rejectMutation.mutate({ cid: rejecting, reason: rejectReason.trim() });
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
