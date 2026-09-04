import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AppShell } from "@/components/AppShell";
import { Field } from "@/components/Field";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth, requireRole, hasRole } from "@/lib/role-guards";
import {
  useOffenceTypes,
  useCreateOffenceType,
  useUpdateOffenceType,
  useDecideOffenceType,
  useMeetings,
  useCreateMeeting,
  useAttendance,
  useSetAttendance,
  useTriggerMeetingFines,
  useFines,
  useProposeWaiver,
  useDecideWaiver,
} from "@/hooks/use-obligations";
import { groupsApi } from "@/api/groups";
import { useMembers } from "@/hooks/use-members";
import { tzs } from "@/lib/format";
import { useAppModal } from "@/components/AppModal";
import {
  Gavel,
  CalendarDays,
  ReceiptText,
  Loader2,
  Check,
  X,
  Plus,
} from "lucide-react";

export const Route = createFileRoute("/mikutano")({
  head: () => ({
    meta: [
      { title: "Mikutano na Makosa — Money Seeking" },
      { name: "description", content: "Aina za makosa, mikutano, mahudhurio na kuidhinisha misamaha." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    requireRole("chair", "secretary");
  },
  component: MikutanoPage,
});

type Tab = "makosa" | "mikutano" | "faini";

const KIND_LABEL: Record<string, string> = {
  late_contribution: "Kuchelewa mchango",
  meeting_absence: "Kutohudhuria mkutano",
  meeting_late: "Kuchelewa mkutanoni",
  other: "Nyingine",
};

function MikutanoPage() {
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("makosa");
  const isChair = hasRole(user, "chair", "admin");
  const isSecretary = hasRole(user, "secretary", "admin");

  return (
    <AppShell
      title="Mikutano na Makosa"
      subtitle={isChair ? "Pendekeza aina za makosa — Katibu anaidhinisha" : "Idhinisha, andikisha mahudhurio, toa faini"}
    >
      <div className="mb-4 flex flex-wrap gap-2">
        {([
          { id: "makosa", label: "Aina za Makosa" },
          { id: "mikutano", label: "Mikutano" },
          { id: "faini", label: "Faini na Misamaha" },
        ] as { id: Tab; label: string }[]).map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`rounded-full px-4 py-1.5 text-sm font-medium ${
              tab === t.id ? "bg-primary text-primary-foreground" : "border border-border text-muted-foreground"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>
      {tab === "makosa" && <OffenceTab isChair={isChair} isSecretary={isSecretary} />}
      {tab === "mikutano" && <MeetingsTab isSecretary={isSecretary} />}
      {tab === "faini" && <FinesTab isChair={isChair} isSecretary={isSecretary} />}
    </AppShell>
  );
}

function useGroupId() {
  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  return (gs?.data.id ?? null) as string | null;
}

function Card({ title, sub, icon: Icon, children }: { title: string; sub?: string; icon: any; children: React.ReactNode }) {
  return (
    <section className="card-surface overflow-hidden">
      <header className="flex items-center gap-2.5 border-b border-border px-4 py-3">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
          <Icon className="h-4 w-4" />
        </span>
        <div>
          <h3 className="font-display text-sm font-semibold">{title}</h3>
          {sub && <p className="text-[11px] text-muted-foreground">{sub}</p>}
        </div>
      </header>
      <div className="space-y-3 p-4">{children}</div>
    </section>
  );
}

// ── Aina za makosa ───────────────────────────────────────────────────────────
function OffenceTab({ isChair, isSecretary }: { isChair: boolean; isSecretary: boolean }) {
  const groupId = useGroupId();
  const { showModal } = useAppModal();
  const { data, isLoading } = useOffenceTypes(groupId);
  const create = useCreateOffenceType();
  const update = useUpdateOffenceType();
  const decide = useDecideOffenceType();
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState({ kind: "late_contribution", name: "", fine_type: "fixed", amount: "", grace: "0" });

  const submit = () => {
    const payload: any = {
      kind: form.kind,
      name: form.name.trim(),
      fine_type: form.fine_type,
      grace_period_days: parseInt(form.grace || "0", 10),
    };
    if (form.fine_type === "fixed") payload.fine_amount = parseFloat(form.amount);
    else payload.fine_percentage = parseFloat(form.amount);
    const done = (msg: string) => showModal({ title: "Imefanikiwa", message: msg, variant: "success", primaryLabel: "Sawa" });
    const fail = (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" });
    if (editing && groupId) {
      update.mutate({ groupId, id: editing, data: payload }, { onSuccess: () => { done("Mabadiliko yametumwa kwa Katibu"); setFormOpen(false); setEditing(null); }, onError: fail });
    } else if (groupId) {
      create.mutate({ groupId, data: payload }, { onSuccess: (r: any) => { done(r.message); setFormOpen(false); }, onError: fail });
    }
  };

  return (
    <Card title="Aina za Makosa" sub="Mwenyekiti anapendekeza · Katibu anaidhinisha" icon={Gavel}>
      {isChair && !formOpen && (
        <button onClick={() => { setEditing(null); setForm({ kind: "late_contribution", name: "", fine_type: "fixed", amount: "", grace: "0" }); setFormOpen(true); }} className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground">
          <Plus className="h-3.5 w-3.5" /> Pendekeza Kosa
        </button>
      )}
      {isChair && formOpen && (
        <div className="space-y-2 rounded-lg border p-3">
          <div className="grid gap-2 sm:grid-cols-2">
            <label className="block text-xs">Aina
              <select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })} className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm">
                {Object.entries(KIND_LABEL).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
              </select>
            </label>
            <Field label="Jina (mf. Kuchelewesha Mchango)" value={form.name} onChange={(v) => setForm({ ...form, name: v })} />
            <label className="block text-xs">Aina ya faini
              <select value={form.fine_type} onChange={(e) => setForm({ ...form, fine_type: e.target.value })} className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm">
                <option value="fixed">Kiasi maalum (TZS)</option>
                <option value="percentage">Asilimia ya mchango (%)</option>
              </select>
            </label>
            <Field label={form.fine_type === "fixed" ? "Kiasi (TZS)" : "Asilimia (%)"} type="number" value={form.amount} onChange={(v) => setForm({ ...form, amount: v })} />
            <Field label="Siku za neema" type="number" value={form.grace} onChange={(v) => setForm({ ...form, grace: v })} />
          </div>
          <div className="flex gap-2">
            <button onClick={submit} disabled={!form.name.trim() || !(parseFloat(form.amount) > 0)} className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50">Tuma Pendekezo</button>
            <button onClick={() => { setFormOpen(false); setEditing(null); }} className="rounded-lg border border-border px-3 py-2 text-xs">Ghairi</button>
          </div>
        </div>
      )}
      {isLoading ? <Loader2 className="h-5 w-5 animate-spin text-primary" /> : (
        <div className="space-y-2">
          {(data?.data ?? []).map((o) => (
            <div key={o.id} className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2">
              <div className="text-sm">
                <p className="font-semibold">{o.name}</p>
                <p className="text-xs text-muted-foreground">
                  {KIND_LABEL[o.kind] ?? o.kind} · {o.fine_type === "fixed" ? tzs(Number(o.fine_amount ?? 0)) : `${o.fine_percentage}%`} · neema siku {o.grace_period_days}
                </p>
                <StatusChip status={o.status} />
              </div>
              <div className="flex shrink-0 gap-1.5">
                {isChair && o.status !== "inactive" && (
                  <button onClick={() => { setEditing(o.id); setForm({ kind: o.kind, name: o.name, fine_type: o.fine_type, amount: String(o.fine_type === "fixed" ? o.fine_amount ?? "" : o.fine_percentage ?? ""), grace: String(o.grace_period_days) }); setFormOpen(true); }} className="rounded-lg border border-border px-2 py-1 text-xs">Badilisha</button>
                )}
                {isSecretary && o.status === "pending" && groupId && (
                  <button onClick={() => decide.mutate({ groupId, id: o.id, approve: true })} className="inline-flex items-center gap-1 rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground">
                    <Check className="h-3 w-3" /> Idhinisha
                  </button>
                )}
                {(isSecretary || isChair) && o.status === "active" && groupId && (
                  <button onClick={() => decide.mutate({ groupId, id: o.id, approve: false })} className="rounded-lg border border-destructive/40 px-2 py-1 text-xs text-destructive">Zima</button>
                )}
              </div>
            </div>
          ))}
          {(data?.data.length ?? 0) === 0 && <p className="text-sm text-muted-foreground">Hakuna aina za makosa bado.</p>}
        </div>
      )}
    </Card>
  );
}

function StatusChip({ status }: { status: string }) {
  const map: Record<string, string> = {
    pending: "bg-amber-100 text-amber-800",
    active: "bg-emerald-500/15 text-emerald-700",
    inactive: "bg-muted text-muted-foreground",
  };
  const label: Record<string, string> = { pending: "Inasubiri", active: "Hai", inactive: "Zima" };
  return <span className={`mt-1 inline-block rounded-full px-2 py-0.5 text-[10px] font-semibold ${map[status] ?? map.inactive}`}>{label[status] ?? status}</span>;
}

// ── Mikutano + mahudhurio ────────────────────────────────────────────────────
function MeetingsTab({ isSecretary }: { isSecretary: boolean }) {
  const groupId = useGroupId();
  const { showModal } = useAppModal();
  const { data, isLoading } = useMeetings(groupId);
  const create = useCreateMeeting();
  const trigger = useTriggerMeetingFines();
  const [title, setTitle] = useState("");
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [selected, setSelected] = useState<string | null>(null);

  return (
    <div className="space-y-3">
      <Card title="Mikutano" sub="Unda mkutano, andikisha mahudhurio" icon={CalendarDays}>
        <div className="grid gap-2 sm:grid-cols-3">
          <Field label="Kichwa" value={title} onChange={setTitle} placeholder="Mkutano Mkuu" />
          <Field label="Tarehe" type="date" value={date} onChange={setDate} />
          <div className="flex items-end">
            <button
              disabled={!groupId || !title.trim() || create.isPending}
              onClick={() => groupId && create.mutate({ groupId, data: { title: title.trim(), meeting_date: date } }, {
                onSuccess: () => setTitle(""),
                onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
              })}
              className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50"
            >Unda Mkutano</button>
          </div>
        </div>
        {isLoading ? <Loader2 className="h-5 w-5 animate-spin text-primary" /> : (
          <div className="space-y-2">
            {(data?.data ?? []).map((m) => (
              <div key={m.id} className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2">
                <div className="text-sm">
                  <p className="font-semibold">{m.title}</p>
                  <p className="text-xs text-muted-foreground">{new Date(m.meeting_date).toLocaleDateString()}</p>
                </div>
                <div className="flex gap-1.5">
                  <button onClick={() => setSelected(selected === m.id ? null : m.id)} className="rounded-lg border border-border px-2 py-1 text-xs">
                    {selected === m.id ? "Funga" : "Mahudhurio"}
                  </button>
                  {isSecretary && (
                    <button
                      onClick={() => trigger.mutate(m.id, {
                        onSuccess: (r: any) => showModal({ title: "Imekamilika", message: `${r.message} (${r.created} zimetolewa)`, variant: "success", primaryLabel: "Sawa" }),
                        onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
                      })}
                      disabled={trigger.isPending}
                      className="rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground disabled:opacity-50"
                    >Toa Faini</button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
      {selected && <AttendanceCard meetingId={selected} canEdit={isSecretary} />}
    </div>
  );
}

function AttendanceCard({ meetingId, canEdit }: { meetingId: string; canEdit: boolean }) {
  const { showModal } = useAppModal();
  const { data, isLoading } = useAttendance(meetingId);
  const { data: membersData } = useMembers({ limit: 500 });
  const save = useSetAttendance();
  const [marks, setMarks] = useState<Record<string, string>>({});

  const members = membersData?.data ?? [];
  const existing: Record<string, string> = {};
  (data?.data ?? []).forEach((a) => { existing[a.member_id] = a.status; });
  const statusOf = (id: string) => marks[id] ?? existing[id] ?? "present";

  return (
    <Card title="Mahudhurio" sub={canEdit ? "Katibu: chagua hali kwa kila mwanachama" : "Kusoma tu"} icon={CalendarDays}>
      {isLoading ? <Loader2 className="h-5 w-5 animate-spin text-primary" /> : (
        <div className="space-y-1.5">
          {members.filter((m: any) => m.is_active).map((m: any) => (
            <div key={m.id} className="flex items-center justify-between gap-2 rounded-lg border px-3 py-1.5 text-sm">
              <span>{m.full_name} <span className="text-xs text-muted-foreground">({m.member_no})</span></span>
              {canEdit ? (
                <select
                  value={statusOf(m.id)}
                  onChange={(e) => setMarks((p) => ({ ...p, [m.id]: e.target.value }))}
                  className="rounded-lg border border-border bg-background px-2 py-1 text-xs"
                >
                  <option value="present">Alihudhuria</option>
                  <option value="absent">Hayupo</option>
                  <option value="late">Aliachelewa</option>
                </select>
              ) : (
                <span className="text-xs text-muted-foreground">{statusOf(m.id) === "present" ? "Alihudhuria" : statusOf(m.id) === "absent" ? "Hayupo" : "Aliachelewa"}</span>
              )}
            </div>
          ))}
        </div>
      )}
      {canEdit && (
        <button
          disabled={save.isPending}
          onClick={() => {
            const rows = members.filter((m: any) => m.is_active).map((m: any) => ({ member_id: m.id, status: statusOf(m.id) }));
            save.mutate({ meetingId, rows }, {
              onSuccess: () => { setMarks({}); showModal({ title: "Imefanikiwa", message: "Mahudhurio yamehifadhiwa. Sasa bofya 'Toa Faini'.", variant: "success", primaryLabel: "Sawa" }); },
              onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
            });
          }}
          className="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50"
        >Hifadhi Mahudhurio</button>
      )}
    </Card>
  );
}

// ── Faini + misamaha ─────────────────────────────────────────────────────────
function FinesTab({ isChair, isSecretary }: { isChair: boolean; isSecretary: boolean }) {
  const { showModal } = useAppModal();
  const { data, isLoading } = useFines({});
  const propose = useProposeWaiver();
  const decide = useDecideWaiver();
  const [waiveFor, setWaiveFor] = useState<string | null>(null);
  const [reason, setReason] = useState("");

  return (
    <Card title="Faini na Misamaha" sub={isChair ? "Pendekeza msamaha — Katibu anaamua" : "Amua maombi ya misamaha"} icon={ReceiptText}>
      {isLoading ? <Loader2 className="h-5 w-5 animate-spin text-primary" /> : (
        <div className="space-y-2">
          {(data?.data ?? []).map((f) => (
            <div key={f.id} className="rounded-lg border px-3 py-2 text-sm">
              <div className="flex items-center justify-between gap-2">
                <div>
                  <p className="font-semibold">{f.offence_type?.name ?? f.reason}</p>
                  <p className="text-xs text-muted-foreground">
                    {f.member?.full_name} ({f.member?.member_no}) · {new Date(f.occurrence_date).toLocaleDateString()} · {f.status === "unpaid" ? "Haijalipwa" : f.status === "paid" ? "Imelipwa" : "Imesamehewa"}
                    {f.waiver_status === "pending" ? " · Msamaha unaosubiri" : ""}
                  </p>
                </div>
                <span className="shrink-0 font-bold">{tzs(Number(f.amount))}</span>
              </div>
              {isChair && f.status === "unpaid" && f.waiver_status === "none" && (
                waiveFor === f.id ? (
                  <div className="mt-2 flex gap-2">
                    <input
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      placeholder="Sababu ya msamaha"
                      className="flex-1 rounded-lg border border-border bg-background px-2 py-1.5 text-xs"
                    />
                    <button
                      onClick={() => propose.mutate({ id: f.id, reason }, {
                        onSuccess: (r: any) => { setWaiveFor(null); setReason(""); showModal({ title: "Imetumwa", message: r.message, variant: "success", primaryLabel: "Sawa" }); },
                        onError: (e: Error) => showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
                      })}
                      disabled={!reason.trim() || propose.isPending}
                      className="rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground disabled:opacity-50"
                    >Tuma</button>
                    <button onClick={() => { setWaiveFor(null); setReason(""); }} className="rounded-lg border border-border px-2 py-1 text-xs"><X className="h-3 w-3" /></button>
                  </div>
                ) : (
                  <button onClick={() => setWaiveFor(f.id)} className="mt-2 rounded-lg border border-border px-2 py-1 text-xs">Pendekeza Msamaha</button>
                )
              )}
              {isSecretary && f.waiver_status === "pending" && (
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">Sababu: {f.waiver_request_reason ?? f.reason_note ?? "—"}</span>
                  <button onClick={() => decide.mutate({ id: f.id, approve: true })} className="inline-flex items-center gap-1 rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground"><Check className="h-3 w-3" /> Kubali</button>
                  <button onClick={() => decide.mutate({ id: f.id, approve: false })} className="rounded-lg border border-destructive/40 px-2 py-1 text-xs text-destructive">Kataa</button>
                </div>
              )}
            </div>
          ))}
          {(data?.data.length ?? 0) === 0 && <p className="text-sm text-muted-foreground">Hakuna faini.</p>}
        </div>
      )}
    </Card>
  );
}
