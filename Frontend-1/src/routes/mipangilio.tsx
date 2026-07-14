import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { AppShell } from "@/components/AppShell";
import { requireAuth } from "@/lib/role-guards";
import { useAuth } from "@/lib/auth-provider";
import { useMembers } from "@/hooks/use-members";
import {
  useCommitteeMembers,
  useAppointCommitteeMember,
  useRemoveCommitteeMember,
} from "@/hooks/use-loan-committee";
import { useBackupHistory, useBackupSettings, useGenerateBackup, useSaveBackupSettings } from "@/hooks/use-backup";
import { useDownloadReport } from "@/hooks/use-reports";
import { backupApi } from "@/api/backup";
import {
  Settings as Cog,
  Bell,
  Globe,
  UserPlus,
  Users,
  Search,
  Loader2,
  X,
  Trash,
  Database,
  Download,
  FileText,
  Mail,
  Clock,
  HardDrive,
  CheckCircle2,
  XCircle,
} from "lucide-react";

export const Route = createFileRoute("/mipangilio")({
  head: () => ({
    meta: [
      { title: "Mipangilio — Money Seeking" },
      { name: "description", content: "Mipangilio ya mfumo wa Money Seeking." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
  },
  component: MipangilioPage,
});

function MipangilioPage() {
  const { user } = useAuth();
  const isChair = user?.role === "chair";
  const isAdmin = user?.role === "admin";

  const [mchango, setMchango] = useState(() => {
    if (typeof window !== "undefined") {
      return localStorage.getItem("kikundi-mchango") || "50,000";
    }
    return "50,000";
  });
  const [editingMchango, setEditingMchango] = useState(false);
  const [mchangoInput, setMchangoInput] = useState(mchango);

  const saveMchango = () => {
    const cleaned = mchangoInput.replace(/[^0-9]/g, "");
    if (cleaned) {
      const formatted = Number(cleaned).toLocaleString("en-TZ");
      setMchango(formatted);
      if (typeof window !== "undefined") {
        localStorage.setItem("kikundi-mchango", formatted);
      }
    }
    setEditingMchango(false);
  };

  return (
    <AppShell title="Mipangilio" subtitle="Sanidi mfumo kulingana na kikundi chako">
      <div className="grid gap-4 lg:grid-cols-2">
        <Card icon={Cog} title="Taarifa za Kikundi">
          <Row k="Jina la kikundi" v="Money Seeking" />
          <Row k="Mahali" v="Iringa, Tanzania" />
          <Row k="Mwaka ulioanzishwa" v="2024" />
          {!isChair && !isAdmin && <Row k="Mchango wa kawaida" v={`${mchango} TZS / mwezi`} />}
          {isChair && editingMchango ? (
            <div className="flex items-center gap-2 px-4 py-2.5">
              <span className="text-sm text-muted-foreground">Mchango wa kawaida</span>
              <div className="flex items-center gap-1.5 ml-auto">
                <input
                  type="text"
                  value={mchangoInput}
                  onChange={(e) => setMchangoInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && saveMchango()}
                  className="w-28 rounded-lg border border-input bg-background px-2 py-1 text-sm text-right font-semibold outline-none focus:border-primary"
                  autoFocus
                />
                <span className="text-sm text-muted-foreground">TZS / mwezi</span>
                <button onClick={saveMchango} className="rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground">Hifadhi</button>
                <button onClick={() => setEditingMchango(false)} className="rounded-lg px-2 py-1 text-xs text-muted-foreground hover:bg-muted">Ghairi</button>
              </div>
            </div>
          ) : isChair ? (
            <div
              className="flex items-center justify-between px-4 py-2.5 cursor-pointer hover:bg-muted/50"
              onClick={() => {
                setMchangoInput(mchango.replace(/,/g, ""));
                setEditingMchango(true);
              }}
            >
              <span className="text-sm text-muted-foreground">Mchango wa kawaida</span>
              <div className="flex items-center gap-2">
                <span className="text-sm font-semibold">{mchango} TZS / mwezi</span>
                <Cog className="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </div>
          ) : null}
        </Card>

        {!isAdmin && (
          <Card icon={Bell} title="Arifa">
            <Toggle label="Arifa ya mchango wa mwezi" defaultChecked />
            <Toggle label="Arifa ya mkopo unaokaribia kuisha" defaultChecked />
            <Toggle label="Arifa za SMS / WhatsApp" />
          </Card>
        )}

        <Card icon={Globe} title="Lugha & Sarafu">
          <Row k="Lugha" v="Kiswahili" />
          <Row k="Sarafu" v="Shilingi ya Tanzania (TZS)" />
          <Row k="Eneo la wakati" v="Africa/Dar_es_Salaam" />
        </Card>

        {/* Admin Backup Center */}
        {isAdmin && <BackupCenter />}

        {/* Chair Reports Center */}
        {isChair && <ReportsCenter />}

        {/* Loan Committee Management — Chair only */}
        {isChair && <LoanCommitteeManagement />}
      </div>
    </AppShell>
  );
}

// ==================== ADMIN BACKUP CENTER ====================
function BackupCenter() {
  const { data: settingsData } = useBackupSettings();
  const { data: historyData } = useBackupHistory({ limit: 5 });
  const generateBackup = useGenerateBackup();
  const saveSettings = useSaveBackupSettings();

  const [email, setEmail] = useState(settingsData?.email ?? "");
  const [backupType, setBackupType] = useState(settingsData?.backup_type ?? "database_only");
  const [frequency, setFrequency] = useState(settingsData?.frequency ?? "manual");
  const [settingsMsg, setSettingsMsg] = useState<string | null>(null);

  // Sync from API when settings load
  useEffect(() => {
    if (settingsData) {
      setEmail(settingsData.email ?? "");
      setBackupType(settingsData.backup_type ?? "database_only");
      setFrequency(settingsData.frequency ?? "manual");
    }
  }, [settingsData]);

  const handleSaveSettings = async () => {
    try {
      await saveSettings.mutateAsync({ email, backup_type: backupType, frequency });
      setSettingsMsg("Mipangilio imehifadhiwa ✓");
      setTimeout(() => setSettingsMsg(null), 3000);
    } catch {
      setSettingsMsg("Imeshindikana kuhifadhi");
    }
  };

  const handleGenerate = async () => {
    try {
      await generateBackup.mutateAsync(backupType);
    } catch { /* handled by RQ */ }
  };

  const history = historyData?.data ?? [];

  return (
    <>
      <Card icon={Database} title="Hifadhi Nakala ya Mfumo">
        <div className="px-4 py-3">
          <p className="text-xs text-muted-foreground mb-4">
            Unda nakala salama ya mfumo na kuituma kwa barua pepe.
          </p>

          <div className="space-y-3">
            <label className="block">
              <span className="mb-1 block text-xs font-medium text-muted-foreground">Barua pepe ya backup</span>
              <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5">
                <Mail className="h-4 w-4 text-muted-foreground" />
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@example.com"
                  className="w-full bg-transparent text-sm outline-none"
                />
              </div>
            </label>

            <label className="block">
              <span className="mb-1 block text-xs font-medium text-muted-foreground">Aina ya backup</span>
              <select
                value={backupType}
                onChange={(e) => setBackupType(e.target.value)}
                className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none"
              >
                <option value="database_only">Database pekee</option>
                <option value="database_files">Database + Faili zilizopakiwa</option>
                <option value="full_system">Mfumo mzima</option>
              </select>
            </label>

            <label className="block">
              <span className="mb-1 block text-xs font-medium text-muted-foreground">Mzunguko wa backup</span>
              <select
                value={frequency}
                onChange={(e) => setFrequency(e.target.value)}
                className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none"
              >
                <option value="manual">Mwenyewe (Manual)</option>
                <option value="daily">Kila siku</option>
                <option value="weekly">Kila wiki</option>
                <option value="monthly">Kila mwezi</option>
              </select>
            </label>

            {settingsMsg && (
              <p className={`text-sm ${settingsMsg.includes("✓") ? "text-success" : "text-destructive"}`}>{settingsMsg}</p>
            )}

            <div className="flex gap-2 pt-1">
              <button
                onClick={handleSaveSettings}
                disabled={saveSettings.isPending}
                className="flex-1 inline-flex items-center justify-center gap-2 rounded-xl border border-border px-3 py-2.5 text-sm font-semibold hover:bg-muted disabled:opacity-50"
              >
                {saveSettings.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Cog className="h-4 w-4" />}
                Hifadhi Mipangilio
              </button>
              <button
                onClick={handleGenerate}
                disabled={generateBackup.isPending}
                className="flex-1 inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-3 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50"
              >
                {generateBackup.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <HardDrive className="h-4 w-4" />}
                Tengeneza Backup Sasa
              </button>
            </div>
          </div>
        </div>
      </Card>

      {/* Backup History */}
      <Card icon={Clock} title="Historia ya Backup">
        <div className="px-4 py-3">
          {history.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">Hakuna backup bado.</p>
          ) : (
            <div className="space-y-2">
              {history.map((h) => (
                <div key={h.id} className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2">
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{h.filename}</p>
                    <p className="text-[10px] text-muted-foreground">
                      {new Date(h.created_at).toLocaleString("sw-TZ")} · {(h.size_bytes / 1024).toFixed(1)} KB
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    {h.status === "completed" ? (
                      <CheckCircle2 className="h-4 w-4 text-success" />
                    ) : h.status === "failed" ? (
                      <XCircle className="h-4 w-4 text-destructive" />
                    ) : (
                      <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                    )}
                    <button
                      onClick={() => {
                        const url = backupApi.downloadUrl(h.id);
                        const token = sessionStorage.getItem("kikundi-token");
                        fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
                          .then((r) => r.blob())
                          .then((blob) => {
                            const a = document.createElement("a");
                            a.href = URL.createObjectURL(blob);
                            a.download = h.filename;
                            document.body.appendChild(a);
                            a.click();
                            a.remove();
                          })
                          .catch(() => {});
                      }}
                      className="text-xs text-primary hover:underline"
                    >
                      Pakua
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </Card>
    </>
  );
}

// ==================== CHAIR REPORTS CENTER ====================
function ReportsCenter() {
  const downloadReport = useDownloadReport();
  const [month, setMonth] = useState("");
  const [loanStatus, setLoanStatus] = useState("");

  const handleDownload = (type: string, param?: string) => {
    downloadReport.mutate({ type, param });
  };

  const reports = [
    {
      id: "wanachama",
      title: "Wanachama",
      desc: "Orodha ya wanachama, tarehe za kujiunga, hali",
      icon: Users,
    },
    {
      id: "michango",
      title: "Michango",
      desc: "Michango, jumla, salio",
      icon: FileText,
    },
    {
      id: "mikopo",
      title: "Mikopo",
      desc: "Mikopo hai, iliyolipwa, iliyopita muda",
      icon: FileText,
    },
    {
      id: "mapato",
      title: "Mapato na Matumizi",
      desc: "Mapato, matumizi, faida/hasara",
      icon: FileText,
    },
    {
      id: "muhtasari",
      title: "Muhtasari wa Kikundi",
      desc: "Ripoti ya muhtasari wa uongozi",
      icon: FileText,
    },
  ];

  return (
    <Card icon={Download} title="Ripoti za Kikundi">
      <div className="px-4 py-3">
        <p className="text-xs text-muted-foreground mb-4">
          Pakua ripoti mbalimbali za kikundi.
        </p>

        {/* Filters */}
        <div className="grid grid-cols-2 gap-2 mb-4">
          <label className="block">
            <span className="mb-1 block text-[10px] font-medium text-muted-foreground">Mwezi (michango)</span>
            <input
              type="month"
              value={month}
              onChange={(e) => setMonth(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-2 py-1.5 text-xs outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-[10px] font-medium text-muted-foreground">Hali ya mkopo</span>
            <select
              value={loanStatus}
              onChange={(e) => setLoanStatus(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-2 py-1.5 text-xs outline-none"
            >
              <option value="">Zote</option>
              <option value="OUTSTANDING">Haijapitwa muda</option>
              <option value="CLOSED">Imefungwa</option>
              <option value="PENDING">Inasubiri</option>
            </select>
          </label>
        </div>

        <div className="space-y-2">
          {reports.map((r) => (
            <div key={r.id} className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2.5">
              <div className="min-w-0 flex items-center gap-2.5">
                <span className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
                  <r.icon className="h-4 w-4" />
                </span>
                <div>
                  <p className="text-sm font-semibold">{r.title}</p>
                  <p className="text-[10px] text-muted-foreground">{r.desc}</p>
                </div>
              </div>
              <button
                onClick={() => {
                  const param = r.id === "michango" ? month : r.id === "mikopo" ? loanStatus : undefined;
                  handleDownload(r.id, param || undefined);
                }}
                disabled={downloadReport.isPending}
                className="shrink-0 inline-flex items-center gap-1 rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground disabled:opacity-50"
              >
                <Download className="h-3 w-3" /> Pakua
              </button>
            </div>
          ))}
        </div>

        {downloadReport.isError && (
          <p className="mt-2 text-sm text-destructive">{downloadReport.error?.message ?? "Imeshindikana kupakua"}</p>
        )}
      </div>
    </Card>
  );
}

// ==================== LOAN COMMITTEE ====================
function LoanCommitteeManagement() {
  const { data: membersData, isLoading: membersLoading } = useCommitteeMembers();
  const { data: allMembersData } = useMembers({ limit: 500 });
  const appointMember = useAppointCommitteeMember();
  const removeMember = useRemoveCommitteeMember();

  const [searchQuery, setSearchQuery] = useState("");
  const [showAppointDialog, setShowAppointDialog] = useState(false);
  const [removing, setRemoving] = useState<string | null>(null);

  const committeeMembers = membersData?.data ?? [];
  const allMembers = allMembersData?.data ?? [];

  const committeeUserIds = new Set(committeeMembers.map((m) => m.user_id));
  const availableMembers = allMembers.filter(
    (m) =>
      m.is_active &&
      m.user_id &&
      !committeeUserIds.has(m.user_id) &&
      (searchQuery === "" ||
        m.full_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        m.member_no.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const handleAppoint = async (userId: string) => {
    try {
      await appointMember.mutateAsync(userId);
      setShowAppointDialog(false);
      setSearchQuery("");
    } catch { /* handled by RQ */ }
  };

  const handleRemove = async (id: string) => {
    try {
      await removeMember.mutateAsync(id);
      setRemoving(null);
    } catch { /* handled by RQ */ }
  };

  return (
    <Card icon={UserPlus} title="Kamati ya Mikopo">
      <div className="px-4 py-3">
        <p className="text-xs text-muted-foreground mb-3">
          Simamia wanachama wa kamati ya mikopo. Mwenyekiti, Katibu na Mweka Hazina ni wanachama otomatiki.
        </p>

        {membersLoading ? (
          <div className="flex justify-center py-4">
            <Loader2 className="h-5 w-5 animate-spin text-primary" />
          </div>
        ) : (
          <div className="space-y-2 mb-4">
            {committeeMembers.map((m) => (
              <div
                key={`${m.user_id}-${m.id || "auto"}`}
                className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="text-sm font-semibold truncate">{m.user_name}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {m.user_role} {m.appointed_by ? `• Ameteuliwa na ${m.appointed_by}` : "• Otomatiki"}
                  </p>
                </div>
                {m.id !== "" && (
                  <button
                    onClick={() => setRemoving(m.id)}
                    className="shrink-0 rounded-lg p-1.5 text-destructive hover:bg-destructive/10"
                    title="Ondoa"
                  >
                    <Trash className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        <button
          onClick={() => setShowAppointDialog(true)}
          className="flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border px-3 py-2.5 text-sm font-medium text-primary hover:bg-primary/5"
        >
          <UserPlus className="h-4 w-4" /> Teua Mwanachama Mpya
        </button>
      </div>

      {showAppointDialog && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center"
          onClick={() => { setShowAppointDialog(false); setSearchQuery(""); }}
        >
          <div
            className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl max-h-[80vh] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center justify-between">
              <h3 className="font-display text-lg font-semibold">Teua Mwanachama</h3>
              <button onClick={() => { setShowAppointDialog(false); setSearchQuery(""); }} className="rounded-lg p-1.5 hover:bg-muted">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="relative mb-3">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                placeholder="Tafuta mwanachama..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full rounded-lg border border-border bg-background py-2 pl-9 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>

            <div className="flex-1 overflow-y-auto space-y-1.5">
              {availableMembers.length === 0 ? (
                <p className="py-4 text-center text-sm text-muted-foreground">
                  {searchQuery ? "Hakuna mwanachama anayefanana na utafutaji." : "Wanachama wote tayari ni wa kamati ya mikopo."}
                </p>
              ) : (
                availableMembers.map((m) => (
                  <button
                    key={m.id}
                    onClick={() => handleAppoint(m.user_id!)}
                    disabled={appointMember.isPending}
                    className="flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-left hover:bg-muted disabled:opacity-50"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-semibold truncate">{m.full_name}</p>
                      <p className="text-[10px] text-muted-foreground">{m.member_no}</p>
                    </div>
                    <UserPlus className="h-4 w-4 shrink-0 text-primary" />
                  </button>
                ))
              )}
            </div>
          </div>
        </div>
      )}

      {removing != null && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center"
          onClick={() => setRemoving(null)}
        >
          <div className="w-full max-w-sm rounded-t-3xl bg-card p-5 sm:rounded-2xl" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-lg font-semibold mb-2">Ondoa Mwanachama</h3>
            <p className="text-sm text-muted-foreground mb-4">Una uhakika unataka kumuondoa mwanachama huyu kutoka kamati ya mikopo?</p>
            <div className="flex gap-3">
              <button onClick={() => setRemoving(null)} className="flex-1 rounded-xl border border-border py-2.5 text-sm font-semibold">Ghairi</button>
              <button
                onClick={() => handleRemove(removing)}
                disabled={removeMember.isPending}
                className="flex-1 inline-flex items-center justify-center gap-2 rounded-xl bg-destructive py-2.5 text-sm font-semibold text-white disabled:opacity-50"
              >
                {removeMember.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash className="h-4 w-4" />}
                Ondoa
              </button>
            </div>
          </div>
        </div>
      )}
    </Card>
  );
}

// ==================== SHARED COMPONENTS ====================
function Card({ icon: Icon, title, children }: { icon: any; title: string; children: React.ReactNode }) {
  return (
    <section className="card-surface overflow-hidden">
      <header className="flex items-center gap-2.5 border-b border-border px-4 py-3">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
          <Icon className="h-4 w-4" />
        </span>
        <h3 className="font-display text-sm font-semibold">{title}</h3>
      </header>
      <div className="divide-y divide-border">{children}</div>
    </section>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-center justify-between px-4 py-2.5">
      <span className="text-sm text-muted-foreground">{k}</span>
      <span className="text-sm font-semibold">{v}</span>
    </div>
  );
}

function Toggle({ label, defaultChecked, disabled }: { label: string; defaultChecked?: boolean; disabled?: boolean }) {
  return (
    <label className={`flex items-center justify-between px-4 py-3 ${disabled ? "opacity-50" : ""}`}>
      <span className="text-sm">{label}</span>
      <input type="checkbox" defaultChecked={defaultChecked} disabled={disabled} className="h-5 w-9 appearance-none rounded-full bg-muted transition-colors checked:bg-primary relative cursor-pointer before:absolute before:top-0.5 before:left-0.5 before:h-4 before:w-4 before:rounded-full before:bg-white before:transition-transform checked:before:translate-x-4" />
    </label>
  );
}
