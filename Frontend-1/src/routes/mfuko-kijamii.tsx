import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { blockAdminFromPage, requireAuth } from "@/lib/role-guards";
import { useAuth } from "@/lib/auth-provider";
import { useMembers } from "@/hooks/use-members";
import {
  useWelfareDashboard,
  useWelfareEvents,
  useWelfareEvent,
  useCreateWelfareEvent,
  useApproveWelfareEvent,
  useRejectWelfareEvent,
  useMyWelfareContributions,
  useWelfareContributions,
  useRecordWelfarePayment,
  useWaiveWelfareContribution,
  useDisburseWelfareEvent,
} from "@/hooks/use-welfare";
import type {
  WelfareEventType,
  WelfareFundingSource,
  WelfareEvent,
  WelfareContribution,
  CreateWelfareEventRequest,
} from "@/api/welfare";
import { tzs, tarehe } from "@/lib/format";
import {
  Heart,
  Plus,
  Check,
  Ban,
  X,
  Loader2,
  Wallet,
  Users,
  Clock,
  CheckCircle,
  XCircle,
  Search,
  FileText,
  HandCoins,
  Eye,
} from "lucide-react";

export const Route = createFileRoute("/mfuko-kijamii")({
  head: () => ({
    meta: [
      { title: "Mfuko wa Kijamii — Money Seeking" },
      { name: "description", content: "Simamia mfuko wa kijamii — misiba, harusi, dharura na mengineyo." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    blockAdminFromPage();
  },
  component: MfukoKijamiiPage,
});

type Tab = "matukio" | "michango" | "ripoti";

const tabLabels: Record<Tab, string> = {
  matukio: "Matukio",
  michango: "Michango",
  ripoti: "Ripoti",
};

const eventTypeLabels: Record<WelfareEventType, string> = {
  MSIBA: "Misiba",
  HARUSI: "Harusi",
  DHARURA: "Dharura",
  MATIBABU: "Matibabu",
  KUZALIWA: "Kuzaliwa",
  ELIMU: "Elimu",
};

const fundingLabels: Record<WelfareFundingSource, string> = {
  TREASURY: "Hazina",
  MEMBER_CONTRIBUTION: "Michango ya Wanachama",
  BOTH: "Hazina + Wanachama",
};

function MfukoKijamiiPage() {
  const { user } = useAuth();
  if (!user) return null;

  const isTreasurer = user.role === "treasurer";
  const isChair = user.role === "chair";
  const isMember = user.role === "member";
  const isSecretary = user.role === "secretary";

  const [tab, setTab] = useState<Tab>("matukio");

  return (
    <AppShell
      title="Mfuko wa Kijamii"
      subtitle={
        isTreasurer
          ? "Simamia matukio na michango ya kijamii"
          : isChair
          ? "Idhinisha matukio ya kijamii"
          : isMember
          ? "Michango yako ya kijamii"
          : "Ripoti za mfuko wa kijamii"
      }
      action={isTreasurer ? <CreateEventButton /> : null}
    >
      <DashboardStats />

      <div className="mt-5 -mx-1 flex gap-1 overflow-x-auto pb-1">
        {(["matukio", "michango", "ripoti"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`shrink-0 rounded-lg px-4 py-1.5 text-xs font-semibold ${
              tab === t ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
            }`}
          >
            {tabLabels[t]}
          </button>
        ))}
      </div>

      <div className="mt-3">
        {tab === "matukio" && <EventsTab />}
        {tab === "michango" && <ContributionsTab />}
        {tab === "ripoti" && <ReportsTab />}
      </div>
    </AppShell>
  );
}

// ---------- Dashboard Stats ----------

function DashboardStats() {
  const { user } = useAuth();
  const { data: dashData } = useWelfareDashboard();
  const dash = dashData?.data;
  if (!dash) return null;

  const isMember = user?.role === "member";

  const stats = isMember
    ? [
        { label: "Michango Inayosubiri", value: String(dash.my_pending_contributions), icon: Clock, color: "text-warning" },
        { label: "Michango Imelipwa", value: String(dash.my_paid_contributions), icon: CheckCircle, color: "text-success" },
      ]
    : [
        { label: "Matukio Yote", value: String(dash.total_events), icon: Heart, color: "text-primary" },
        { label: "Yanasubiri Idhini", value: String(dash.pending_approval), icon: Clock, color: "text-warning" },
        { label: "Yaliyoidhinishwa", value: String(dash.active_events), icon: CheckCircle, color: "text-success" },
        { label: "Yaliyokataliwa", value: String(dash.rejected_events), icon: XCircle, color: "text-destructive" },
      ];

  return (
    <div className="grid gap-3 grid-cols-2 lg:grid-cols-4">
      {stats.map((s) => (
        <div key={s.label} className="card-surface p-3">
          <div className="flex items-center gap-2.5">
            <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10">
              <s.icon className={`h-4 w-4 ${s.color}`} />
            </span>
            <div>
              <p className="text-[10px] text-muted-foreground">{s.label}</p>
              <p className="text-lg font-bold">{s.value}</p>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// ---------- Events Tab ----------

function EventsTab() {
  const { user } = useAuth();
  const isChair = user?.role === "chair";
  const isTreasurer = user?.role === "treasurer";

  const [statusFilter, setStatusFilter] = useState("");
  const { data: eventsData, isLoading } = useWelfareEvents({
    limit: 100,
    status: statusFilter || undefined,
  });

  const [viewingEvent, setViewingEvent] = useState<string | null>(null);
  const [approvingEvent, setApprovingEvent] = useState<string | null>(null);
  const [rejectingEvent, setRejectingEvent] = useState<string | null>(null);

  const events = eventsData?.data ?? [];

  return (
    <div>
      <div className="flex gap-2 mb-4 overflow-x-auto">
        {[
          { value: "", label: "Yote" },
          { value: "PENDING", label: "Yanasubiri" },
          { value: "APPROVED", label: "Yaliyoidhinishwa" },
          { value: "COMPLETED", label: "Yaliyokamilika" },
          { value: "REJECTED", label: "Yaliyokataliwa" },
        ].map((f) => (
          <button
            key={f.value}
            onClick={() => setStatusFilter(f.value)}
            className={`shrink-0 rounded-lg px-3 py-1.5 text-xs font-semibold ${
              statusFilter === f.value ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {isLoading && (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      )}

      <div className="space-y-2.5">
        {events.map((ev) => (
          <EventCard
            key={ev.id}
            event={ev}
            onView={() => setViewingEvent(ev.id)}
            onApprove={isChair && ev.status === "PENDING" ? () => setApprovingEvent(ev.id) : undefined}
            onReject={isChair && ev.status === "PENDING" ? () => setRejectingEvent(ev.id) : undefined}
          />
        ))}
        {events.length === 0 && !isLoading && (
          <div className="card-surface p-8 text-center text-sm text-muted-foreground">
            Hakuna matukio ya kijamii.
          </div>
        )}
      </div>

      {viewingEvent != null && (
        <EventDetailDialog eventId={viewingEvent} onClose={() => setViewingEvent(null)} />
      )}
      {approvingEvent != null && (
        <ApproveDialog eventId={approvingEvent} onClose={() => setApprovingEvent(null)} />
      )}
      {rejectingEvent != null && (
        <RejectDialog eventId={rejectingEvent} onClose={() => setRejectingEvent(null)} />
      )}
    </div>
  );
}

function EventCard({
  event,
  onView,
  onApprove,
  onReject,
}: {
  event: WelfareEvent;
  onView: () => void;
  onApprove?: () => void;
  onReject?: () => void;
}) {
  const statusMap: Record<string, { label: string; cls: string }> = {
    PENDING: { label: "Inasubiri", cls: "bg-warning/25 text-foreground" },
    APPROVED: { label: "Imeidhinishwa", cls: "bg-primary/15 text-primary" },
    REJECTED: { label: "Imekataliwa", cls: "bg-destructive/10 text-destructive" },
    COMPLETED: { label: "Imekamilika", cls: "bg-success/15 text-success" },
  };
  const st = statusMap[event.status] ?? { label: event.status, cls: "bg-muted text-foreground" };

  return (
    <div className="card-surface p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate font-semibold">
            {event.member?.full_name ?? `Mwanachama #${event.member_id}`}
          </p>
          <p className="text-xs text-muted-foreground">
            {eventTypeLabels[event.event_type] ?? event.event_type} • {event.member?.member_no}
          </p>
        </div>
        <span className={`chip text-[10px] ${st.cls}`}>{st.label}</span>
      </div>

      <p className="mt-2 text-xs text-muted-foreground line-clamp-2">{event.description}</p>

      <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
        <div>
          <p className="text-muted-foreground">Kiasi</p>
          <p className="font-semibold">{tzs(event.amount_requested)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Chanzo</p>
          <p className="font-semibold">{fundingLabels[event.funding_source]}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Tarehe</p>
          <p className="font-semibold">{tarehe(event.created_at)}</p>
        </div>
      </div>

      {event.rejection_reason && (
        <p className="mt-2 text-xs text-destructive">Sababu: {event.rejection_reason}</p>
      )}

      <div className="mt-3 flex gap-2">
        <button
          onClick={onView}
          className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs font-semibold hover:bg-muted"
        >
          <Eye className="h-3.5 w-3.5" /> Angalia
        </button>
        {onApprove && (
          <button
            onClick={onApprove}
            className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-success px-3 py-2 text-xs font-semibold text-white"
          >
            <Check className="h-3.5 w-3.5" /> Idhinisha
          </button>
        )}
        {onReject && (
          <button
            onClick={onReject}
            className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-destructive/10 px-3 py-2 text-xs font-semibold text-destructive"
          >
            <Ban className="h-3.5 w-3.5" /> Kataa
          </button>
        )}
      </div>
    </div>
  );
}

// ---------- Event Detail Dialog ----------

function EventDetailDialog({ eventId, onClose }: { eventId: string; onClose: () => void }) {
  const { user } = useAuth();
  const { data, isLoading } = useWelfareEvent(eventId);
  const recordPayment = useRecordWelfarePayment();
  const waiveContrib = useWaiveWelfareContribution();
  const disburseEvent = useDisburseWelfareEvent();
  const [payingMember, setPayingMember] = useState<string | null>(null);
  const [payAmount, setPayAmount] = useState("");

  if (isLoading || !data) {
    return (
      <Modal title="Taarifa za Tukio" onClose={onClose}>
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      </Modal>
    );
  }

  const { data: event, contributions, stats } = data;
  const isTreasurer = user?.role === "treasurer";
  const canDisburse = isTreasurer && event.status === "APPROVED" && stats.pending_count === 0;

  return (
    <Modal title="Taarifa za Tukio" onClose={onClose}>
      <div className="space-y-4">
        {/* Event info */}
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <p className="text-muted-foreground">Mwanachama</p>
            <p className="font-semibold">{event.member?.full_name}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Aina ya Tukio</p>
            <p className="font-semibold">{eventTypeLabels[event.event_type]}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Kiasi Kilichoombwa</p>
            <p className="font-semibold">{tzs(event.amount_requested)}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Kiasi Kilichoidhinishwa</p>
            <p className="font-semibold">{event.amount_approved ? tzs(event.amount_approved) : "—"}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Chanzo cha Fedha</p>
            <p className="font-semibold">{fundingLabels[event.funding_source]}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Hali</p>
            <p className="font-semibold">{event.status}</p>
          </div>
        </div>

        <p className="text-xs text-muted-foreground">{event.description}</p>

        {/* Contributions */}
        {contributions.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-2">Michango ya Wanachama</h4>
            <div className="text-xs mb-2 flex gap-4">
              <span className="text-success">Imelipwa: {stats.paid_count} ({tzs(stats.total_paid)})</span>
              <span className="text-warning">Inasubiri: {stats.pending_count} ({tzs(stats.total_pending)})</span>
            </div>
            <div className="max-h-48 overflow-y-auto space-y-1.5">
              {contributions.map((c) => (
                <div key={c.id} className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2">
                  <div className="min-w-0">
                    <p className="text-xs font-semibold truncate">{c.member?.full_name ?? `Mwanachama #${c.member_id}`}</p>
                    <p className="text-[10px] text-muted-foreground">{c.member?.member_no}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold">{tzs(c.amount)}</span>
                    {c.status === "PAID" ? (
                      <span className="chip text-[10px] bg-success/15 text-success">Imelipwa</span>
                    ) : c.status === "WAIVED" ? (
                      <span className="chip text-[10px] bg-muted text-muted-foreground">Imesamehewa</span>
                    ) : (
                      <span className="chip text-[10px] bg-warning/25 text-foreground">Inasubiri</span>
                    )}
                    {isTreasurer && c.status === "PENDING" && (
                      <div className="flex gap-1">
                        <button
                          onClick={() => {
                            setPayingMember(c.member_id);
                            setPayAmount(String(c.amount));
                          }}
                          className="rounded p-1 text-success hover:bg-success/10"
                          title="Rekodi Malipo"
                        >
                          <Wallet className="h-3 w-3" />
                        </button>
                        <button
                          onClick={() => waiveContrib.mutate({ eventId: event.id, memberId: c.member_id })}
                          className="rounded p-1 text-muted-foreground hover:bg-muted"
                          title="Samehe"
                        >
                          <Ban className="h-3 w-3" />
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Pay dialog */}
      {payingMember != null && (
        <div className="mt-4 border-t border-border pt-4">
          <p className="text-sm font-semibold mb-2">Rekodi Malipo</p>
          <input
            type="number"
            value={payAmount}
            onChange={(e) => setPayAmount(e.target.value)}
            placeholder="Kiasi"
            className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <div className="mt-2 flex gap-2">
            <button onClick={() => setPayingMember(null)} className="flex-1 rounded-lg border border-border py-2 text-xs font-semibold">
              Ghairi
            </button>
            <button
              onClick={() => {
                recordPayment.mutate({
                  eventId: event.id,
                  memberId: payingMember,
                  data: { amount: Number(payAmount) },
                });
                setPayingMember(null);
              }}
              disabled={recordPayment.isPending}
              className="flex-1 rounded-lg bg-success py-2 text-xs font-semibold text-white disabled:opacity-50"
            >
              {recordPayment.isPending ? <Loader2 className="h-3 w-3 animate-spin mx-auto" /> : "Thibitisha"}
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}

// ---------- Approve Dialog ----------

function ApproveDialog({ eventId, onClose }: { eventId: string; onClose: () => void }) {
  const { data } = useWelfareEvent(eventId);
  const approveEvent = useApproveWelfareEvent();
  const [amount, setAmount] = useState("");

  const event = data?.data;

  return (
    <Modal title="Idhinisha Tukio la Kijamii" onClose={onClose}>
      {event && (
        <div className="space-y-3">
          <p className="text-sm">
            <span className="font-semibold">{eventTypeLabels[event.event_type]}</span> — {event.member?.full_name}
          </p>
          <p className="text-xs text-muted-foreground">Kiasi kilichoombwa: {tzs(event.amount_requested)}</p>
          <div>
            <label className="text-sm font-medium">Kiasi Kilichoidhinishwa (TZS)</label>
            <input
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder={String(event.amount_requested)}
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>
          <button
            onClick={() => {
              approveEvent.mutate({
                id: eventId,
                data: { approved_amount: Number(amount) || event.amount_requested },
              });
              onClose();
            }}
            disabled={approveEvent.isPending}
            className="w-full rounded-xl bg-success py-3 text-sm font-semibold text-white disabled:opacity-50 inline-flex items-center justify-center gap-2"
          >
            {approveEvent.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
            Thibitisha Kuidhinisha
          </button>
        </div>
      )}
    </Modal>
  );
}

// ---------- Reject Dialog ----------

function RejectDialog({ eventId, onClose }: { eventId: string; onClose: () => void }) {
  const rejectEvent = useRejectWelfareEvent();
  const [reason, setReason] = useState("");

  return (
    <Modal title="Kataa Tukio la Kijamii" onClose={onClose}>
      <div>
        <label className="text-sm font-medium">Sababu ya Kukataa</label>
        <textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Andika sababu..."
          className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          rows={3}
        />
      </div>
      <button
        onClick={() => {
          rejectEvent.mutate({ id: eventId, data: { reason: reason || "Haijatolewa sababu" } });
          onClose();
        }}
        disabled={rejectEvent.isPending}
        className="mt-3 w-full rounded-xl bg-destructive py-3 text-sm font-semibold text-white disabled:opacity-50 inline-flex items-center justify-center gap-2"
      >
        {rejectEvent.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />}
        Thibitisha Kukataa
      </button>
    </Modal>
  );
}

// ---------- Contributions Tab ----------

function ContributionsTab() {
  const { user } = useAuth();
  const isMember = user?.role === "member";

  const [statusFilter, setStatusFilter] = useState("");

  const { data: myData, isLoading: myLoading } = useMyWelfareContributions({
    limit: 100,
    status: statusFilter || undefined,
  });
  const { data: allData, isLoading: allLoading } = useWelfareContributions({
    limit: 100,
    status: statusFilter || undefined,
  });

  const contribs = isMember ? (myData?.data ?? []) : (allData?.data ?? []);
  const isLoading = isMember ? myLoading : allLoading;

  return (
    <div>
      <div className="flex gap-2 mb-4 overflow-x-auto">
        {[
          { value: "", label: "Zote" },
          { value: "PENDING", label: "Zinasubiri" },
          { value: "PAID", label: "Zimelipwa" },
          { value: "WAIVED", label: "Zimesamehewa" },
        ].map((f) => (
          <button
            key={f.value}
            onClick={() => setStatusFilter(f.value)}
            className={`shrink-0 rounded-lg px-3 py-1.5 text-xs font-semibold ${
              statusFilter === f.value ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {isLoading && (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      )}

      <div className="space-y-2">
        {contribs.map((c) => (
          <div key={c.id} className="card-surface p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-sm font-semibold">
                  {c.member?.full_name ?? `Mwanachama #${c.member_id}`}
                </p>
                <p className="text-xs text-muted-foreground">
                  {c.event ? `${eventTypeLabels[c.event.event_type] ?? c.event.event_type} — ${c.event.description}` : `Tukio #${c.event_id}`}
                </p>
              </div>
              <span
                className={`chip text-[10px] ${
                  c.status === "PAID"
                    ? "bg-success/15 text-success"
                    : c.status === "WAIVED"
                    ? "bg-muted text-muted-foreground"
                    : "bg-warning/25 text-foreground"
                }`}
              >
                {c.status === "PAID" ? "Imelipwa" : c.status === "WAIVED" ? "Imesamehewa" : "Inasubiri"}
              </span>
            </div>
            <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
              <div>
                <p className="text-muted-foreground">Kiasi</p>
                <p className="font-semibold">{tzs(c.amount)}</p>
              </div>
              {c.paid_at && (
                <div>
                  <p className="text-muted-foreground">Tarehe ya Malipo</p>
                  <p className="font-semibold">{tarehe(c.paid_at)}</p>
                </div>
              )}
            </div>
          </div>
        ))}
        {contribs.length === 0 && !isLoading && (
          <div className="card-surface p-8 text-center text-sm text-muted-foreground">
            Hakuna michango ya kijamii.
          </div>
        )}
      </div>
    </div>
  );
}

// ---------- Reports Tab ----------

function ReportsTab() {
  const { data: dashData } = useWelfareDashboard();
  const { data: eventsData } = useWelfareEvents({ limit: 500 });
  const dash = dashData?.data;
  const events = eventsData?.data ?? [];

  const completedCount = events.filter((e) => e.status === "COMPLETED").length;
  const totalRequested = events.reduce((s, e) => s + Number(e.amount_requested), 0);
  const totalApproved = events.reduce((s, e) => s + Number(e.amount_approved ?? 0), 0);

  return (
    <div className="space-y-4">
      <div className="card-surface p-4">
        <h3 className="font-display text-sm font-semibold mb-3">Muhtasari wa Mfuko</h3>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Matukio Yote</span>
            <span className="font-semibold">{dash?.total_events ?? 0}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Yaliyokamilika</span>
            <span className="font-semibold text-success">{completedCount}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Jumla Iliyoombwa</span>
            <span className="font-semibold">{tzs(totalRequested)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Jumla Iliyoidhinishwa</span>
            <span className="font-semibold">{tzs(totalApproved)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Michango ya Wanachama</span>
            <span className="font-semibold">{tzs(dash?.total_collected ?? 0)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Fedha za Hazina</span>
            <span className="font-semibold">{tzs(dash?.total_from_treasury ?? 0)}</span>
          </div>
        </div>
      </div>

      {/* By event type */}
      <div className="card-surface p-4">
        <h3 className="font-display text-sm font-semibold mb-3">Kwa Aina ya Tukio</h3>
        <div className="space-y-2">
          {(["MSIBA", "HARUSI", "DHARURA", "MATIBABU", "KUZALIWA", "ELIMU"] as WelfareEventType[]).map((et) => {
            const count = events.filter((e) => e.event_type === et).length;
            if (count === 0) return null;
            const total = events.filter((e) => e.event_type === et).reduce((s, e) => s + Number(e.amount_requested), 0);
            return (
              <div key={et} className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2 text-sm">
                <span>{eventTypeLabels[et]}</span>
                <div className="text-right">
                  <span className="font-semibold">{count} matukio</span>
                  <span className="ml-2 text-muted-foreground">({tzs(total)})</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ---------- Create Event Button ----------

function CreateEventButton() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="inline-flex items-center gap-1.5 rounded-xl bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground"
      >
        <Plus className="h-4 w-4" /> Unda Tukio
      </button>
      {open && <CreateEventForm onClose={() => setOpen(false)} />}
    </>
  );
}

// ---------- Create Event Form ----------

function CreateEventForm({ onClose }: { onClose: () => void }) {
  const createEvent = useCreateWelfareEvent();
  const { data: membersData } = useMembers({ limit: 500 });
  const members = membersData?.data ?? [];

  const [f, setF] = useState({
    memberId: "",
    eventType: "" as WelfareEventType | "",
    description: "",
    amount: "",
    fundingSource: "" as WelfareFundingSource | "",
    treasuryAmount: "",
    memberAmount: "",
  });

  const handleSubmit = async () => {
    if (!f.memberId || !f.eventType || !f.description || !f.amount || !f.fundingSource) return;

    // Set default amounts based on funding source
    let treasuryAmount = Number(f.treasuryAmount) || 0;
    let memberAmount = Number(f.memberAmount) || 0;
    
    if (f.fundingSource === "TREASURY" && treasuryAmount === 0) {
      treasuryAmount = Number(f.amount);
    } else if (f.fundingSource === "MEMBER_CONTRIBUTION" && memberAmount === 0) {
      memberAmount = Number(f.amount);
    }

    const data: CreateWelfareEventRequest = {
      member_id: f.memberId,
      event_type: f.eventType as WelfareEventType,
      description: f.description,
      amount_requested: Number(f.amount),
      funding_source: f.fundingSource as WelfareFundingSource,
      treasury_amount: treasuryAmount,
      member_amount: memberAmount,
    };

    try {
      await createEvent.mutateAsync(data);
      onClose();
    } catch { /* handled by RQ */ }
  };

  return (
    <Modal title="Tunda Tukio la Kijamii" onClose={onClose}>
      <div className="space-y-3">
        <Field
          label="Mwanachama Aliyeathiriwa"
          value={f.memberId}
          onChange={(v) => setF({ ...f, memberId: v })}
          type="select"
          options={members.map((m) => ({ value: String(m.id), label: `${m.full_name} (${m.member_no})` }))}
        />
        <Field
          label="Aina ya Tukio"
          value={f.eventType}
          onChange={(v) => setF({ ...f, eventType: v as WelfareEventType })}
          type="select"
          options={[
            { value: "MSIBA", label: "Misiba" },
            { value: "HARUSI", label: "Harusi" },
            { value: "DHARURA", label: "Dharura" },
            { value: "MATIBABU", label: "Matibabu" },
            { value: "KUZALIWA", label: "Kuzaliwa" },
            { value: "ELIMU", label: "Elimu" },
          ]}
        />
        <div>
          <label className="text-sm font-medium">Maelezo</label>
          <textarea
            value={f.description}
            onChange={(e) => setF({ ...f, description: e.target.value })}
            placeholder="Maelezo ya tukio..."
            className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
            rows={3}
          />
        </div>
        <Field label="Kiasi Kinachohitajika (TZS)" value={f.amount} onChange={(v) => setF({ ...f, amount: v })} type="number" />
        <Field
          label="Chanzo cha Fedha"
          value={f.fundingSource}
          onChange={(v) => setF({ ...f, fundingSource: v as WelfareFundingSource })}
          type="select"
          options={[
            { value: "TREASURY", label: "Hazina ya Kikundi" },
            { value: "MEMBER_CONTRIBUTION", label: "Michango ya Wanachama" },
            { value: "BOTH", label: "Hazina + Wanachama" },
          ]}
        />
        {f.fundingSource === "TREASURY" && (
          <Field label="Kiasi kutoka Hazina (TZS)" value={f.treasuryAmount || f.amount} onChange={(v) => setF({ ...f, treasuryAmount: v })} type="number" />
        )}
        {f.fundingSource === "MEMBER_CONTRIBUTION" && (
          <Field label="Kiasi kutoka Wanachama (TZS)" value={f.memberAmount || f.amount} onChange={(v) => setF({ ...f, memberAmount: v })} type="number" />
        )}
        {f.fundingSource === "BOTH" && (
          <>
            <Field label="Kiasi kutoka Hazina (TZS)" value={f.treasuryAmount} onChange={(v) => setF({ ...f, treasuryAmount: v })} type="number" />
            <Field label="Kiasi kutoka Wanachama (TZS)" value={f.memberAmount} onChange={(v) => setF({ ...f, memberAmount: v })} type="number" />
          </>
        )}
      </div>
      <button
        onClick={handleSubmit}
        disabled={createEvent.isPending}
        className="mt-5 w-full rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-50 inline-flex items-center justify-center gap-2"
      >
        {createEvent.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
        Tunda Tukio
      </button>
    </Modal>
  );
}

// ---------- Shared Components ----------

function Field({
  label,
  value,
  onChange,
  type = "text",
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  options?: { value: string; label: string }[];
}) {
  if (type === "select" && options) {
    return (
      <div>
        <label className="text-sm font-medium">{label}</label>
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        >
          <option value="">Chagua...</option>
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>
    );
  }
  return (
    <div>
      <label className="text-sm font-medium">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
      />
    </div>
  );
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={onClose}>
      <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-display text-lg font-semibold">{title}</h3>
          <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted">
            <X className="h-4 w-4" />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
