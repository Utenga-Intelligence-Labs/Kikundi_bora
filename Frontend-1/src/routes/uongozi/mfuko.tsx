import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { Field } from "@/components/Field";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth, requireRole, blockAdminFromPage } from "@/lib/role-guards";
import { useAppModal } from "@/components/AppModal";
import { tzs } from "@/lib/format";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  useWelfareEvents,
  useWelfareEvent,
  useCreateWelfareEvent,
  useDisburseWelfareEvent,
  welfareKeys,
} from "@/hooks/use-welfare";
import { welfareApi, type WelfareEventType, type WelfareFundingSource } from "@/api/welfare";
import { useMembers } from "@/hooks/use-members";
import {
  HeartHandshake,
  Plus,
  Check,
  Ban,
  Trash2,
  Wallet,
  Banknote,
  Loader2,
  X,
} from "lucide-react";

export const Route = createFileRoute("/uongozi/mfuko")({
  head: () => ({
    meta: [
      { title: "Mfuko wa Kijamii (Usimamizi) — Money Seeking" },
      { name: "description", content: "Usimamizi wa mifuko ya kijamii kwa Mweka Hazina: matukio, michango ya kila mwanachama, malipo." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    requireRole("treasurer");
    blockAdminFromPage();
  },
  component: MfukoManagementPage,
});

function useTreasuryWelfareAction() {
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const invalidate = () => qc.invalidateQueries({ queryKey: welfareKeys.all });
  const onError = (e: Error) =>
    showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" });
  return {
    approve: useMutation({
      mutationFn: (id: string) => welfareApi.approveContribution(id),
      onSuccess: invalidate,
      onError,
    }),
    record: useMutation({
      mutationFn: (v: { eventId: string; memberId: string; amount: number }) =>
        welfareApi.recordPayment(v.eventId, v.memberId, { amount: v.amount }),
      onSuccess: invalidate,
      onError,
    }),
    waive: useMutation({
      mutationFn: (v: { eventId: string; memberId: string }) =>
        welfareApi.waiveContribution(v.eventId, v.memberId),
      onSuccess: invalidate,
      onError,
    }),
    remove: useMutation({
      mutationFn: (id: string) => welfareApi.removeContribution(id),
      onSuccess: invalidate,
      onError,
    }),
  };
}

function MfukoManagementPage() {
  const { user } = useAuth();
  const [eventId, setEventId] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const { data: eventsData } = useWelfareEvents({ limit: 100 });
  const events = eventsData?.data ?? [];

  if (user?.role !== "treasurer") return null;

  return (
    <AppShell
      title="Mfuko wa Kijamii — Usimamizi"
      subtitle="Matukio, michango ya kila mwanachama, na malipo (Mweka Hazina)"
      action={
        <button
          onClick={() => setShowCreate(true)}
          className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground"
        >
          <Plus className="h-4 w-4" /> Unda Tukio
        </button>
      }
    >
      <div className="card-surface p-4 mb-4">
        <label className="block text-xs font-medium text-muted-foreground mb-1">Chagua tukio
          <select
            value={eventId}
            onChange={(e) => setEventId(e.target.value)}
            className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="">— Chagua tukio —</option>
            {events.map((ev) => (
              <option key={ev.id} value={ev.id}>
                {ev.event_type} — {ev.member?.full_name ?? ""} ({ev.status})
              </option>
            ))}
          </select>
        </label>
      </div>

      {showCreate && <CreateEventCard onClose={() => setShowCreate(false)} />}
      {eventId ? (
        <EventManageCard key={eventId} eventId={eventId} />
      ) : (
        <div className="card-surface p-8 text-center">
          <HeartHandshake className="mx-auto h-10 w-10 text-muted-foreground/50" />
          <p className="mt-3 text-sm text-muted-foreground">Chagua tukio kuona michango ya kila mwanachama</p>
        </div>
      )}
    </AppShell>
  );
}

function CreateEventCard({ onClose }: { onClose: () => void }) {
  const { showModal } = useAppModal();
  const createEvent = useCreateWelfareEvent();
  // Social-fund attribution: approved members only (server also rejects
  // with 403 — this keeps the dropdown honest too).
  const { data: membersData } = useMembers({ limit: 500, status: "approved" });
  const members = (membersData?.data ?? []).filter((m) => m.approval_status == null || m.approval_status === "approved");
  const [f, setF] = useState({
    memberId: "",
    eventType: "" as WelfareEventType | "",
    description: "",
    amount: "",
    fundingSource: "" as WelfareFundingSource | "",
    treasuryAmount: "",
    memberAmount: "",
  });

  const submit = async () => {
    if (!f.memberId || !f.eventType || !f.description || !f.amount || !f.fundingSource) return;
    let treasuryAmount = Number(f.treasuryAmount) || 0;
    let memberAmount = Number(f.memberAmount) || 0;
    if (f.fundingSource === "TREASURY" && treasuryAmount === 0) treasuryAmount = Number(f.amount);
    if (f.fundingSource === "MEMBER_CONTRIBUTION" && memberAmount === 0) memberAmount = Number(f.amount);
    try {
      await createEvent.mutateAsync({
        member_id: f.memberId,
        event_type: f.eventType as WelfareEventType,
        description: f.description,
        amount_requested: Number(f.amount),
        funding_source: f.fundingSource as WelfareFundingSource,
        treasury_amount: treasuryAmount,
        member_amount: memberAmount,
      });
      onClose();
    } catch (e) {
      showModal({ title: "Hitilafu", message: e instanceof Error ? e.message : "Imeshindikana", variant: "error", primaryLabel: "Sawa" });
    }
  };

  return (
    <div className="card-surface p-5 mb-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-display text-base font-semibold">Tukio Jipya</h3>
        <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted" aria-label="Funga">
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block text-xs">Mwanachama Aliyeathiriwa
          <select value={f.memberId} onChange={(e) => setF({ ...f, memberId: e.target.value })} className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm">
            <option value="">— Chagua —</option>
            {members.map((m) => (
              <option key={m.id} value={String(m.id)}>{m.full_name} ({m.member_no})</option>
            ))}
          </select>
        </label>
        <label className="block text-xs">Aina ya Tukio
          <select value={f.eventType} onChange={(e) => setF({ ...f, eventType: e.target.value as WelfareEventType })} className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm">
            <option value="">— Chagua —</option>
            {["MSIBA", "HARUSI", "DHARURA", "MATIBABU", "KUZALIWA", "ELIMU"].map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
        </label>
        <div className="sm:col-span-2">
          <Field label="Maelezo" value={f.description} onChange={(v) => setF({ ...f, description: v })} />
        </div>
        <Field label="Kiasi Kinachoombwa (TZS)" type="number" value={f.amount} onChange={(v) => setF({ ...f, amount: v })} />
        <label className="block text-xs">Chanzo cha Fedha
          <select value={f.fundingSource} onChange={(e) => setF({ ...f, fundingSource: e.target.value as WelfareFundingSource })} className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm">
            <option value="">— Chagua —</option>
            <option value="TREASURY">Hazina</option>
            <option value="MEMBER_CONTRIBUTION">Michango ya Wanachama</option>
            <option value="BOTH">Vyote</option>
          </select>
        </label>
      </div>
      <button
        onClick={submit}
        disabled={createEvent.isPending}
        className="mt-4 w-full rounded-xl bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50"
      >
        {createEvent.isPending ? "Inatengenezwa..." : "Tengeneza Tukio"}
      </button>
    </div>
  );
}

function EventManageCard({ eventId }: { eventId: string }) {
  const { showModal } = useAppModal();
  const qc = useQueryClient();
  const { data, isLoading } = useWelfareEvent(eventId);
  const actions = useTreasuryWelfareAction();
  const disburseEvent = useDisburseWelfareEvent();
  const [payRow, setPayRow] = useState<string | null>(null);
  const [payAmount, setPayAmount] = useState("");
  const [removing, setRemoving] = useState<{ id: string; name: string } | null>(null);

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    );
  }
  const event = data?.data;
  if (!event) return null;
  const contributions = data?.contributions ?? [];
  const stats = data?.stats ?? { paid_count: 0, pending_count: 0, total_paid: 0, total_pending: 0 };
  const canDisburse =
    event.status === "APPROVED" || (event.status === "COMPLETED" && !event.disbursed_at)
      ? stats.pending_count === 0 && !event.disbursed_at
      : false;

  const doRemove = () => {
    if (!removing) return;
    actions.remove.mutate(removing.id, {
      onSuccess: () => {
        setRemoving(null);
        qc.invalidateQueries({ queryKey: welfareKeys.all });
      },
    });
  };

  return (
    <div className="space-y-3">
      <div className="card-surface p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="font-semibold text-sm">
              {event.event_type} — {event.member?.full_name}
            </p>
            <p className="text-xs text-muted-foreground">
              Imelipwa: {stats.paid_count} ({tzs(stats.total_paid)}) · Inasubiri: {stats.pending_count} ({tzs(stats.total_pending)})
            </p>
          </div>
          <span className="chip text-[10px]">{event.status}</span>
        </div>
        {canDisburse && (
          <button
            onClick={() => disburseEvent.mutate(event.id)}
            disabled={disburseEvent.isPending}
            className="mt-3 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-success px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"
          >
            {disburseEvent.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Banknote className="h-4 w-4" />
            )}
            Toa Fedha kwa Mlengwa ({tzs(event.amount_approved ?? event.amount_requested)})
          </button>
        )}
        {event.disbursed_at && (
          <p className="mt-2 text-xs text-success">
            ✓ Fedha zimetolewa {new Date(event.disbursed_at).toLocaleDateString("sw-TZ")}
            {event.received_at
              ? ` · mlengwa amethibitisha kupokea ${new Date(event.received_at).toLocaleDateString("sw-TZ")}`
              : " · inasubiri uthibitisho wa mlengwa"}
          </p>
        )}
      </div>

      <div className="card-surface p-4">
        <h3 className="font-display text-sm font-semibold mb-3">
          Michango ya Kila Mwanachama ({contributions.length})
        </h3>
        <div className="max-h-[480px] overflow-y-auto space-y-1.5">
          {contributions.map((c) => (
            <div key={c.id} className="rounded-lg border border-border/60 px-3 py-2.5">
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-sm font-semibold truncate">
                    {c.member?.full_name ?? `Mwanachama #${c.member_id}`}
                  </p>
                  <p className="text-[11px] text-muted-foreground">{c.member?.member_no}</p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-sm font-semibold">{tzs(c.amount)}</span>
                  {c.status === "PAID" ? (
                    <span className="chip text-[10px] bg-success/15 text-success">Imelipwa</span>
                  ) : c.status === "WAIVED" ? (
                    <span className="chip text-[10px] bg-muted text-muted-foreground">Imesamehewa</span>
                  ) : (
                    <span className="chip text-[10px] bg-warning/25 text-foreground">Inasubiri</span>
                  )}
                </div>
              </div>
              {c.status === "PENDING" && (
                <div className="mt-2">
                  {payRow === c.id ? (
                    <div className="flex gap-2">
                      <input
                        type="number"
                        value={payAmount}
                        onChange={(e) => setPayAmount(e.target.value)}
                        placeholder="Kiasi"
                        className="flex-1 rounded-lg border border-border bg-background px-2 py-1.5 text-xs"
                      />
                      <button
                        onClick={() => {
                          actions.record.mutate(
                            { eventId: event.id, memberId: c.member_id, amount: Number(payAmount) || Number(c.amount) },
                            { onSuccess: () => { setPayRow(null); setPayAmount(""); } }
                          );
                        }}
                        className="rounded-lg bg-success px-2.5 py-1.5 text-xs font-semibold text-white"
                      >
                        Hifadhi
                      </button>
                      <button
                        onClick={() => { setPayRow(null); setPayAmount(""); }}
                        className="rounded-lg border border-border px-2.5 py-1.5 text-xs"
                      >
                        Ghairi
                      </button>
                    </div>
                  ) : (
                    <div className="flex flex-wrap gap-1.5">
                      <button
                        onClick={() => actions.approve.mutate(c.id)}
                        title="Thibitisha"
                        className="inline-flex items-center gap-1 rounded-lg bg-success/10 px-2 py-1 text-[11px] font-semibold text-success hover:bg-success/20"
                      >
                        <Check className="h-3 w-3" /> Thibitisha
                      </button>
                      <button
                        onClick={() => { setPayRow(c.id); setPayAmount(String(c.amount)); }}
                        title="Rekodi malipo"
                        className="inline-flex items-center gap-1 rounded-lg bg-primary/10 px-2 py-1 text-[11px] font-semibold text-primary hover:bg-primary/20"
                      >
                        <Wallet className="h-3 w-3" /> Rekodi
                      </button>
                      <button
                        onClick={() => {
                          actions.waive.mutate(
                            { eventId: event.id, memberId: c.member_id },
                            {
                              onError: (e: Error) =>
                                showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
                            }
                          );
                        }}
                        title="Samehe"
                        className="inline-flex items-center gap-1 rounded-lg bg-muted px-2 py-1 text-[11px] font-semibold text-muted-foreground hover:bg-muted/80"
                      >
                        <Ban className="h-3 w-3" /> Samehe
                      </button>
                      <button
                        onClick={() => setRemoving({ id: c.id, name: c.member?.full_name ?? c.member_id })}
                        title="Ondoa mwanachama kwenye tukio (hajahusika)"
                        className="inline-flex items-center gap-1 rounded-lg bg-destructive/10 px-2 py-1 text-[11px] font-semibold text-destructive hover:bg-destructive/20"
                      >
                        <Trash2 className="h-3 w-3" /> Ondoa
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
          {contributions.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-4">Hakuna michango kwenye tukio hili.</p>
          )}
        </div>
      </div>

      {removing && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center sm:p-4"
          onClick={() => setRemoving(null)}
        >
          <div
            className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="font-display text-base font-semibold">Ondoa mwanachama?</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              <strong>{removing.name}</strong> ataondolewa kwenye tukio hili kabisa (kwa wale ambao
              hawahusiki). Kitendo hiki hakiwezi kutenduliwa.
            </p>
            <div className="mt-4 flex gap-2">
              <button
                onClick={() => setRemoving(null)}
                className="flex-1 rounded-xl border border-border px-4 py-2.5 text-sm font-medium hover:bg-muted"
              >
                Ghairi
              </button>
              <button
                onClick={doRemove}
                disabled={actions.remove.isPending}
                className="flex-1 rounded-xl bg-destructive px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-50"
              >
                {actions.remove.isPending ? "Inaondoa..." : "Ondoa"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
