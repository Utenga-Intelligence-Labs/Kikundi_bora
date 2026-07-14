import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useSystemHealth, useAdminUsers, useAdminLogs, useOverrideUser, useAdminResetPassword } from "@/hooks/use-admin";
import { roleMap } from "@/api/types";
import type { User, AdminLog, UserStatus } from "@/api/types";
import { useAuth } from "@/lib/auth-provider";
import { hasRole, requireRole } from "@/lib/role-guards";
import { useDebounce } from "@/hooks/use-debounce";
import {
  Settings, Users as UsersIcon, Activity, ShieldCheck,
  Search, ChevronLeft, ChevronRight,
  UserCheck, UserX, Pause, KeyRound, Clock,
} from "lucide-react";

export const Route = createFileRoute("/admin")({
  head: () => ({
    meta: [
      { title: "Msimamizi — Kikundi" },
      { name: "description", content: "Paneli ya msimamizi wa mfumo." },
    ],
  }),
  beforeLoad: () => {
    requireRole("admin");
  },
  component: AdminPage,
});

type Tab = "dashboard" | "users" | "logs";

const statusColors: Record<string, string> = {
  ACTIVE: "bg-success/15 text-success",
  PENDING: "bg-amber-100 text-amber-700",
  REJECTED: "bg-destructive/10 text-destructive",
  SUSPENDED: "bg-muted text-muted-foreground",
};

function AdminPage() {
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("dashboard");

  if (!hasRole(user, "admin")) {
    return (
      <AppShell title="Msimamizi">
        <div className="flex items-center justify-center py-20">
          <p className="text-muted-foreground">Huna ruhusa ya kuona ukurasa huu.</p>
        </div>
      </AppShell>
    );
  }

  return (
    <AppShell title="Paneli ya Msimamizi" subtitle="Simamia mfumo wote na watumiaji">
      {/* Tab bar */}
      <div className="mb-6 flex gap-1 rounded-xl bg-muted p-1">
        {([
          { key: "dashboard" as Tab, label: "Dashibodi", icon: ShieldCheck },
          { key: "users" as Tab, label: "Watumiaji", icon: UsersIcon },
          { key: "logs" as Tab, label: "Kumbukumbu", icon: Activity },
        ]).map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
              tab === key ? "bg-card text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {tab === "dashboard" && <DashboardTab />}
      {tab === "users" && <UsersTab />}
      {tab === "logs" && <LogsTab />}
    </AppShell>
  );
}

function DashboardTab() {
  const { data: health, isLoading } = useSystemHealth();

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="card-surface animate-pulse p-5">
            <div className="h-4 w-1/2 rounded bg-muted" />
            <div className="mt-3 h-8 w-1/3 rounded bg-muted" />
          </div>
        ))}
      </div>
    );
  }

  if (!health) return null;

  const stats = [
    { label: "Watumiaji Wote", value: health.total_users, color: "text-foreground" },
    { label: "Wanaosubiri", value: health.pending_users, color: "text-amber-600" },
    { label: "Wanaotumia", value: health.active_users, color: "text-success" },
    { label: "Waliokataliwa", value: health.rejected_users, color: "text-destructive" },
    { label: "Waliopigwa marufuku", value: health.suspended_users, color: "text-muted-foreground" },
    { label: "Waliingia leo", value: health.recent_logins_24h, color: "text-primary" },
  ];

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {stats.map((s) => (
          <div key={s.label} className="card-surface p-5">
            <p className="text-xs font-medium text-muted-foreground">{s.label}</p>
            <p className={`mt-1 font-display text-3xl font-bold ${s.color}`}>{s.value}</p>
          </div>
        ))}
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="card-surface p-5">
          <h3 className="mb-3 text-sm font-semibold">Watumiaji kwa Jukumu</h3>
          <div className="space-y-2">
            {health.users_by_role.map((r) => (
              <div key={r.role} className="flex items-center justify-between">
                <span className="text-sm">{roleMap[r.role] ?? r.role}</span>
                <span className="text-sm font-semibold">{r.count}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="card-surface p-5">
          <h3 className="mb-3 text-sm font-semibold">Takwimu za Mfumo</h3>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm">Wanachama</span>
              <span className="text-sm font-semibold">{health.total_members}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function UsersTab() {
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, 300);
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminUsers({ page, limit: 20, q: debouncedSearch || undefined, status: statusFilter || undefined });
  const overrideMutation = useOverrideUser();
  const resetPwdMutation = useAdminResetPassword();

  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [actionModal, setActionModal] = useState<"override" | "reset" | null>(null);
  const [overrideAction, setOverrideAction] = useState<"activate" | "deactivate" | "suspend">("activate");
  const [reason, setReason] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [resetTempPassword, setResetTempPassword] = useState<string | null>(null);
  const [resetError, setResetError] = useState<string | null>(null);

  const handleOverride = async () => {
    if (!selectedUser) return;
    setActionLoading(true);
    try {
      await overrideMutation.mutateAsync({ id: selectedUser.id, data: { action: overrideAction, reason } });
      setSelectedUser(null);
      setActionModal(null);
      setReason("");
    } catch { /* handled */ } finally {
      setActionLoading(false);
    }
  };

  const handleResetPassword = async () => {
    if (!selectedUser) return;
    setActionLoading(true);
    setResetError(null);
    setResetTempPassword(null);
    try {
      const res = await resetPwdMutation.mutateAsync({ id: selectedUser.id });
      const temp = (res as { temp_password?: string }).temp_password;
      if (temp) {
        setResetTempPassword(temp);
      } else {
        // Provided password path or already shown — allow close
        setSelectedUser(null);
        setActionModal(null);
      }
    } catch (e: unknown) {
      setResetError(e instanceof Error ? e.message : "Imeshindikana kuweka upya nenosiri");
    } finally {
      setActionLoading(false);
    }
  };

  const closeResetModal = () => {
    setSelectedUser(null);
    setActionModal(null);
    setResetTempPassword(null);
    setResetError(null);
  };

  const users = data?.data ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex flex-col gap-3 sm:flex-row">
        <div className="flex flex-1 items-center gap-2 rounded-xl border border-input bg-background px-3 py-2">
          <Search className="h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            placeholder="Tafuta jina, simu, email..."
            className="w-full bg-transparent text-sm outline-none"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
          className="rounded-xl border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">Hali Zote</option>
          <option value="ACTIVE">Inayotumika</option>
          <option value="PENDING">Inasubiri</option>
          <option value="REJECTED">Imekataliwa</option>
          <option value="SUSPENDED">Imesimamishwa</option>
        </select>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="card-surface animate-pulse px-4 py-3">
              <div className="h-4 w-1/3 rounded bg-muted" />
            </div>
          ))}
        </div>
      ) : users.length === 0 ? (
        <div className="card-surface px-4 py-8 text-center">
          <p className="text-sm text-muted-foreground">Hakuna watumiaji walopatikana.</p>
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {users.map((u) => (
            <div key={u.id} className="flex items-center justify-between px-4 py-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
                    {u.name.charAt(0).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{u.name}</p>
                    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span>{u.phone}</span>
                      <span className="chip text-[10px] bg-primary/10 text-primary">{roleMap[u.role] ?? u.role}</span>
                      <span className={`chip text-[10px] ${statusColors[u.status] ?? ""}`}>{u.status}</span>
                      {u.must_change_password && (
                        <span className="chip text-[10px] bg-amber-100 text-amber-700">Nenosiri la mfumo</span>
                      )}
                    </div>
                  </div>
                </div>
              </div>
              <div className="flex gap-1.5">
                <button
                  onClick={() => { setSelectedUser(u); setActionModal("override"); }}
                  title="Badilisha hali"
                  className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {u.status === "ACTIVE" ? <UserX className="h-4 w-4" /> : <UserCheck className="h-4 w-4" />}
                </button>
                <button
                  onClick={() => { setSelectedUser(u); setActionModal("reset"); }}
                  title="Weka upya nenosiri"
                  className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  <KeyRound className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
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

      {/* Override Modal */}
      {selectedUser && actionModal === "override" && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={() => { setSelectedUser(null); setActionModal(null); }}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-lg font-bold">Badilisha Hali ya Mtumiaji</h3>
            <p className="mt-1 text-sm text-muted-foreground">{selectedUser.name} — {selectedUser.phone}</p>
            <div className="mt-4 space-y-3">
              <div className="flex gap-2">
                {(["activate", "deactivate", "suspend"] as const).map((a) => (
                  <button
                    key={a}
                    onClick={() => setOverrideAction(a)}
                    className={`flex-1 rounded-lg border px-3 py-2 text-xs font-medium ${
                      overrideAction === a ? "border-primary bg-primary/10 text-primary" : "border-border"
                    }`}
                  >
                    {a === "activate" ? "Amilisha" : a === "deactivate" ? "Zima" : "Simamisha"}
                  </button>
                ))}
              </div>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Sababu (si lazima)"
                className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none focus:border-primary"
                rows={2}
              />
            </div>
            <div className="mt-4 flex gap-3">
              <button onClick={() => { setSelectedUser(null); setActionModal(null); setReason(""); }} className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted">Ghairi</button>
              <button onClick={handleOverride} disabled={actionLoading} className="flex-1 rounded-xl bg-primary py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-60">
                {actionLoading ? "Inashughulikiwa..." : "Thibitisha"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reset Password Modal */}
      {selectedUser && actionModal === "reset" && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center" onClick={resetTempPassword ? undefined : closeResetModal}>
          <div className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-lg font-bold">Weka Upya Nenosiri</h3>
            {resetTempPassword ? (
              <div className="mt-3 space-y-3">
                <p className="text-sm text-muted-foreground">
                  Nenosiri la muda la <strong>{selectedUser.name}</strong> limetengenezwa. Liandike sasa — halitaonekana tena.
                </p>
                <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2">
                  <p className="text-xs font-semibold text-amber-900">Nenosiri la muda:</p>
                  <p className="mt-1 font-mono text-base tracking-wide text-amber-950 select-all">{resetTempPassword}</p>
                </div>
                <button onClick={closeResetModal} className="w-full rounded-xl bg-primary py-2.5 text-sm font-semibold text-primary-foreground">
                  Nimeinakili — Funga
                </button>
              </div>
            ) : (
              <>
                <p className="mt-1 text-sm text-muted-foreground">
                  Nenosiri la muda nasibu litatolewa kwa <strong>{selectedUser.name}</strong>. Mtumiaji atakazwa kuweka nenosiri jipya atakapoingia. Onyesha nenosiri la muda mara moja baada ya kuthibitisha.
                </p>
                {resetError && <p className="mt-2 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{resetError}</p>}
                <div className="mt-4 flex gap-3">
                  <button onClick={closeResetModal} className="flex-1 rounded-xl border border-border py-2.5 text-sm font-medium hover:bg-muted">Ghairi</button>
                  <button onClick={handleResetPassword} disabled={actionLoading} className="flex-1 rounded-xl bg-destructive py-2.5 text-sm font-semibold text-white disabled:opacity-60">
                    {actionLoading ? "Inashughulikiwa..." : "Weka Upya"}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function LogsTab() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminLogs({ page, limit: 30 });

  const logs = data?.data ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 30);

  const actionLabels: Record<string, string> = {
    USER_OVERRIDE: "Kubadilisha hali",
    PASSWORD_RESET: "Weka upya nenosiri",
  };

  return (
    <div className="space-y-4">
      {isLoading ? (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="card-surface animate-pulse px-4 py-3">
              <div className="h-4 w-2/3 rounded bg-muted" />
            </div>
          ))}
        </div>
      ) : logs.length === 0 ? (
        <div className="card-surface px-4 py-8 text-center">
          <Activity className="mx-auto mb-3 h-10 w-10 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Hakuna kumbukumbu bado.</p>
        </div>
      ) : (
        <div className="card-surface divide-y divide-border">
          {logs.map((log) => (
            <div key={log.id} className="px-4 py-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium">{actionLabels[log.action] ?? log.action}</p>
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  {new Date(log.created_at).toLocaleString("sw-TZ")}
                </span>
              </div>
              <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                {log.admin && <span>Msimamizi: {log.admin.name}</span>}
                {log.target_user && <span>→ {log.target_user.name} ({log.target_user.phone})</span>}
                {log.ip_address && <span>IP: {log.ip_address}</span>}
              </div>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
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
    </div>
  );
}
