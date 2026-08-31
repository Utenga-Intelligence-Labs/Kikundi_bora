import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Clock, CalendarDays, CheckCircle2, XCircle, Loader2, PiggyBank, Send } from "lucide-react";
import { useAuth } from "@/lib/auth-provider";
import { useAppModal } from "@/components/AppModal";
import {
  groupsApi,
  INTERVAL_LABELS,
  type ContributionInterval,
} from "@/api/groups";

const INTERVALS: ContributionInterval[] = ["weekly", "monthly", "semi_annual", "yearly"];

/**
 * Group contribution interval settings card for /mipangilio:
 *  - everyone (leadership) sees the approved settings + next due date
 *  - Mwenyekiti (chair) gets the propose form (blocked while a proposal is pending)
 *  - Katibu (secretary) gets Approve/Reject on the pending proposal
 */
export function ContributionSettingsCard() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const isChair = user?.role === "chair";
  const isSecretary = user?.role === "secretary";

  const { data, isLoading } = useQuery({
    queryKey: ["groups", "settings"],
    queryFn: () => groupsApi.current(),
  });

  const group = data?.data;
  const pending = data?.pending_proposal ?? null;

  const [form, setForm] = useState({
    interval: "monthly" as ContributionInterval,
    due: "",
    amount: "",
  });
  const [rejectMode, setRejectMode] = useState(false);
  const [rejectReason, setRejectReason] = useState("");

  useEffect(() => {
    if (group) {
      setForm((f) => ({ ...f, interval: group.contribution_interval }));
    }
  }, [group?.id]);

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["groups", "settings"] });
  };

  const proposeMutation = useMutation({
    mutationFn: () =>
      groupsApi.propose(group!.id, {
        contribution_interval: form.interval,
        contribution_due_date: form.due,
        fixed_contribution_amount: form.amount ? parseFloat(form.amount) : undefined,
      }),
    onSuccess: () => {
      showModal({ title: "Imefanikiwa", message: "Pendekezo limetumwa kwa Katibu kwa idhini.", variant: "success", primaryLabel: "Sawa" });
      invalidate();
    },
    onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const approveMutation = useMutation({
    mutationFn: () => groupsApi.approve(group!.id),
    onSuccess: () => {
      showModal({ title: "Imefanikiwa", message: "Mipangilio imeidhinishwa na sasa inatumika.", variant: "success", primaryLabel: "Sawa" });
      invalidate();
    },
    onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const rejectMutation = useMutation({
    mutationFn: () => groupsApi.reject(group!.id, rejectReason),
    onSuccess: () => {
      showModal({ title: "Imefanikiwa", message: "Pendekezo limekataliwa.", variant: "success", primaryLabel: "Sawa" });
      setRejectMode(false);
      setRejectReason("");
      invalidate();
    },
    onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  if (isLoading) {
    return (
      <div className="card-surface p-6 flex items-center gap-2 text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" /> Inapakia mipangilio...
      </div>
    );
  }
  if (!group) return null;

  const fixedAmount = group.fixed_contribution_amount
    ? Number(group.fixed_contribution_amount).toLocaleString()
    : null;

  return (
    <div className="card-surface p-6 space-y-4">
      <div className="flex items-center gap-2">
        <PiggyBank className="h-5 w-5 text-primary" />
        <h3 className="font-display text-lg font-semibold">Mipangilio ya Michango</h3>
      </div>

      {/* Approved settings */}
      <div className="grid sm:grid-cols-3 gap-3">
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-xs text-muted-foreground">Kipindi</p>
          <p className="font-semibold">{INTERVAL_LABELS[group.contribution_interval]}</p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <CalendarDays className="h-3 w-3" /> Tarehe ya mchango
          </p>
          <p className="font-semibold">{group.contribution_due_date ?? "— Haijawekwa"}</p>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <p className="text-xs text-muted-foreground flex items-center gap-1">
            <CheckCircle2 className="h-3 w-3" /> Kiasi kilichopangwa
          </p>
          <p className="font-semibold">{fixedAmount ? `TZS ${fixedAmount}` : "— Haijawekwa"}</p>
        </div>
      </div>
      {data?.next_due_date && (
        <p className="text-sm text-muted-foreground flex items-center gap-1">
          <Clock className="h-3.5 w-3.5" /> Mchango ujao unatarajiwa:{" "}
          <span className="font-semibold text-foreground">{data.next_due_date}</span>
        </p>
      )}

      {/* Pending proposal */}
      {pending && (
        <div className="rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-950/30 p-4 space-y-2">
          <p className="text-sm font-semibold flex items-center gap-2">
            <Clock className="h-4 w-4 text-amber-600" /> Pendekezo lililosubiri idhini
          </p>
          <p className="text-sm">
            {INTERVAL_LABELS[pending.contribution_interval]} · Tarehe:{" "}
            {pending.contribution_due_date ?? "—"} · Kiasi:{" "}
            {pending.fixed_contribution_amount
              ? `TZS ${Number(pending.fixed_contribution_amount).toLocaleString()}`
              : "—"}
          </p>
          {isSecretary && !rejectMode && (
            <div className="flex gap-2 pt-1">
              <button
                onClick={() => approveMutation.mutate()}
                disabled={approveMutation.isPending}
                className="inline-flex items-center gap-1 rounded-lg bg-success px-3 py-1.5 text-sm font-semibold text-white hover:bg-success/90 disabled:opacity-50"
              >
                {approveMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle2 className="h-4 w-4" />}
                Idhinisha
              </button>
              <button
                onClick={() => setRejectMode(true)}
                className="inline-flex items-center gap-1 rounded-lg bg-destructive px-3 py-1.5 text-sm font-semibold text-white hover:bg-destructive/90"
              >
                <XCircle className="h-4 w-4" /> Kataa
              </button>
            </div>
          )}
          {isSecretary && rejectMode && (
            <div className="space-y-2 pt-1">
              <textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Sababu ya kukataa..."
                rows={2}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => rejectReason.trim() && rejectMutation.mutate()}
                  disabled={rejectMutation.isPending || !rejectReason.trim()}
                  className="rounded-lg bg-destructive px-3 py-1.5 text-sm font-semibold text-white disabled:opacity-50"
                >
                  {rejectMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Thibitisha kukataa"}
                </button>
                <button onClick={() => setRejectMode(false)} className="rounded-lg border px-3 py-1.5 text-sm">
                  Ghairi
                </button>
              </div>
            </div>
          )}
          {isChair && (
            <p className="text-xs text-muted-foreground">
              Unaweza kutuma pendekezo jipya baada ya Katibu kujibu hili.
            </p>
          )}
        </div>
      )}

      {/* Chair: propose form */}
      {isChair && (
        <div className="space-y-3 border-t pt-4">
          <p className="text-sm font-medium">
            {pending ? "Pendekezo mpya (limezuiwa — kuna linalosubiri)" : "Pendekeza mabadiliko mapya"}
          </p>
          <div className="grid sm:grid-cols-3 gap-3">
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Kipindi</label>
              <select
                value={form.interval}
                onChange={(e) => setForm({ ...form, interval: e.target.value as ContributionInterval })}
                disabled={!!pending}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm disabled:opacity-50"
              >
                {INTERVALS.map((i) => (
                  <option key={i} value={i}>
                    {INTERVAL_LABELS[i]}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1">
                Tarehe ya mchango {form.interval === "weekly" ? "(1-7, 1=Jumatatu)" : form.interval === "monthly" ? "(siku 1-31)" : "(MM-DD)"}
              </label>
              <input
                value={form.due}
                onChange={(e) => setForm({ ...form, due: e.target.value })}
                placeholder={form.interval === "weekly" ? "3" : form.interval === "monthly" ? "5" : "01-15"}
                disabled={!!pending}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm disabled:opacity-50"
              />
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Kiasi (TZS — hiari)</label>
              <input
                type="number"
                min="0"
                step="0.01"
                value={form.amount}
                onChange={(e) => setForm({ ...form, amount: e.target.value })}
                placeholder="Bila kiasi maalum"
                disabled={!!pending}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm disabled:opacity-50"
              />
            </div>
          </div>
          <button
            onClick={() => proposeMutation.mutate()}
            disabled={!!pending || proposeMutation.isPending || !form.due.trim()}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {proposeMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            Tuma kwa Katibu kwa idhini
          </button>
          <p className="text-xs text-muted-foreground">
            Mabadiliko hairatumiwa mpaka Katibu ai-idhinishе.
          </p>
        </div>
      )}
    </div>
  );
}
