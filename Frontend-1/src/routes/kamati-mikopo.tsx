import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { blockAdminFromPage, requireAuth } from "@/lib/role-guards";
import { useAuth } from "@/lib/auth-provider";
import { roleMap } from "@/api/types";
import {
  useCommitteeDashboard,
  useCommitteeLoans,
  useCommitteeMembers,
  useSubmitLoanReview,
  usePendingLoansCount,
  useCommitteeHistory,
} from "@/hooks/use-loan-committee";
import { useIsCommitteeMember } from "@/hooks/use-loan-committee";
import { tzs, tarehe } from "@/lib/format";
import {
  Users,
  FileSearch,
  CheckCircle,
  XCircle,
  Clock,
  UserCheck,
  Loader2,
  X,
  Check,
  Ban,
  MessageSquare,
  ChevronRight,
} from "lucide-react";

export const Route = createFileRoute("/kamati-mikopo")({
  head: () => ({
    meta: [
      { title: "Kamati ya Mikopo — Money Seeking" },
      { name: "description", content: "Kamati ya mikopo - simamia na kupitia maombi ya mikopo." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    blockAdminFromPage();
  },
  component: KamatiMikopoPage,
});

type Tab = "dashibodi" | "maombi" | "wanachama" | "historia";

const tabLabels: Record<Tab, string> = {
  dashibodi: "Dashibodi",
  maombi: "Maombi ya Mikopo",
  wanachama: "Wanachama wa Kamati",
  historia: "Historia ya Ukaguzi",
};

function KamatiMikopoPage() {
  // All hooks must run before any conditional return (Rules of Hooks)
  const { user } = useAuth();
  const { data: committeeCheck, isLoading: checkLoading } = useIsCommitteeMember();
  const [tab, setTab] = useState<Tab>("dashibodi");

  if (checkLoading) {
    return (
      <AppShell title="Kamati ya Mikopo">
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      </AppShell>
    );
  }

  if (!committeeCheck?.is_committee_member) {
    return (
      <AppShell title="Kamati ya Mikopo">
        <div className="card-surface p-8 text-center">
          <XCircle className="mx-auto h-12 w-12 text-destructive mb-3" />
          <p className="text-lg font-semibold">Huna ruhusa</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Wanachama wa kamati ya mikopo pekee ndio wanaweza kufikia ukurasa huu.
          </p>
          <Link to="/dashibodi" className="mt-4 inline-block text-sm font-semibold text-primary">
            Rudi dashibodi →
          </Link>
        </div>
      </AppShell>
    );
  }

  if (!user) return null;

  const isChair = user.role === "chair";

  return (
    <AppShell
      title="Kamati ya Mikopo"
      subtitle="Simamia na kupitia maombi ya mikopo"
    >
      <div className="mt-2 -mx-1 flex gap-1 overflow-x-auto pb-1">
        {(Object.keys(tabLabels) as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`shrink-0 rounded-lg px-4 py-1.5 text-xs font-semibold ${
              tab === t
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-muted-foreground"
            }`}
          >
            {tabLabels[t]}
          </button>
        ))}
      </div>

      <div className="mt-5">
        {tab === "dashibodi" && <DashboardTab />}
        {tab === "maombi" && <LoansTab />}
        {tab === "wanachama" && <MembersTab isChair={isChair} />}
        {tab === "historia" && <HistoryTab />}
      </div>
    </AppShell>
  );
}

// --- Dashboard Tab ---
function DashboardTab() {
  const { data: dashData, isLoading } = useCommitteeDashboard();

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    );
  }

  const dash = dashData?.data;

  const stats = [
    { label: "Maombi Yanayosubiri", value: dash?.pending_reviews ?? 0, icon: Clock, color: "text-warning" },
    { label: "Yaliyopo Chini ya Ukaguzi", value: dash?.loans_under_review ?? 0, icon: FileSearch, color: "text-primary" },
    { label: "Yaliyoidhinishwa", value: dash?.approved_loans ?? 0, icon: CheckCircle, color: "text-success" },
    { label: "Yaliyokataliwa", value: dash?.rejected_loans ?? 0, icon: XCircle, color: "text-destructive" },
    { label: "Ukaguzi Wangu", value: dash?.my_reviews ?? 0, icon: UserCheck, color: "text-primary" },
    { label: "Wanachama wa Kamati", value: dash?.committee_members ?? 0, icon: Users, color: "text-foreground" },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {stats.map((s) => (
        <div key={s.label} className="card-surface p-4">
          <div className="flex items-center gap-3">
            <span className="grid h-10 w-10 place-items-center rounded-lg bg-primary/10">
              <s.icon className={`h-5 w-5 ${s.color}`} />
            </span>
            <div>
              <p className="text-xs text-muted-foreground">{s.label}</p>
              <p className="text-2xl font-bold">{s.value}</p>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// --- Loans Tab ---
function LoansTab() {
  const [statusFilter, setStatusFilter] = useState<string>("");
  const { data: loansData, isLoading } = useCommitteeLoans({
    limit: 100,
    status: statusFilter || undefined,
  });
  const submitReview = useSubmitLoanReview();

  const [reviewingLoan, setReviewingLoan] = useState<string | null>(null);
  const [reviewDecision, setReviewDecision] = useState<"APPROVE" | "REJECT" | null>(null);
  const [reviewComments, setReviewComments] = useState("");

  const loans = loansData?.data ?? [];

  const handleSubmitReview = async () => {
    if (!reviewingLoan || !reviewDecision) return;
    try {
      await submitReview.mutateAsync({
        loanId: reviewingLoan,
        data: {
          decision: reviewDecision,
          comments: reviewComments || undefined,
        },
      });
      setReviewingLoan(null);
      setReviewDecision(null);
      setReviewComments("");
    } catch {
      /* handled by RQ */
    }
  };

  return (
    <div>
      <div className="flex gap-2 mb-4 overflow-x-auto">
        {[
          { value: "", label: "Yote" },
          { value: "PENDING", label: "Yanayosubiri" },
          { value: "UNDER_REVIEW", label: "Chini ya Ukaguzi" },
        ].map((f) => (
          <button
            key={f.value}
            onClick={() => setStatusFilter(f.value)}
            className={`shrink-0 rounded-lg px-3 py-1.5 text-xs font-semibold ${
              statusFilter === f.value
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-muted-foreground"
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
        {loans.map((l) => (
          <div key={l.id} className="card-surface p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate font-semibold">
                  {l.member?.full_name ?? `Mwanachama #${l.member_id}`}
                </p>
                <p className="text-xs text-muted-foreground">
                  {l.member?.member_no} • {l.purpose || "Hakuna madhumuni"}
                </p>
              </div>
              <span
                className={`chip text-[10px] ${
                  l.status === "PENDING"
                    ? "bg-warning/25 text-foreground"
                    : l.status === "UNDER_REVIEW"
                    ? "bg-primary/15 text-primary"
                    : l.status === "APPROVED"
                    ? "bg-success/15 text-success"
                    : "bg-destructive/10 text-destructive"
                }`}
              >
                {l.status === "PENDING"
                  ? "Inasubiri"
                  : l.status === "UNDER_REVIEW"
                  ? "Chini ya Ukaguzi"
                  : l.status === "APPROVED"
                  ? "Imeidhinishwa"
                  : "Imekataliwa"}
              </span>
            </div>

            <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
              <div>
                <p className="text-muted-foreground">Kiasi</p>
                <p className="font-semibold">{tzs(l.amount)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Tarehe ya Mwisho</p>
                <p className="font-semibold">{tarehe(l.due_date)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Iliyoombwa</p>
                <p className="font-semibold">{tarehe(l.applied_at)}</p>
              </div>
            </div>

            {(l.status === "PENDING" || l.status === "UNDER_REVIEW") && (
              <div className="mt-3 flex gap-2">
                <Link
                  to="/ukaguzi-mkopo/$loanId"
                  params={{ loanId: String(l.id) }}
                  className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-xs font-semibold hover:bg-muted"
                >
                  <FileSearch className="h-3.5 w-3.5" /> Angalia Kwa Undani
                </Link>
                <button
                  onClick={() => {
                    setReviewingLoan(l.id);
                    setReviewDecision("APPROVE");
                  }}
                  className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-success px-3 py-2 text-xs font-semibold text-white"
                >
                  <Check className="h-3.5 w-3.5" /> Idhinisha
                </button>
                <button
                  onClick={() => {
                    setReviewingLoan(l.id);
                    setReviewDecision("REJECT");
                  }}
                  className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-destructive/10 px-3 py-2 text-xs font-semibold text-destructive"
                >
                  <Ban className="h-3.5 w-3.5" /> Kataa
                </button>
              </div>
            )}
          </div>
        ))}

        {loans.length === 0 && !isLoading && (
          <div className="card-surface p-8 text-center text-sm text-muted-foreground">
            Hakuna maombi ya mikopo kwa sasa.
          </div>
        )}
      </div>

      {/* Review Dialog */}
      {reviewingLoan != null && reviewDecision && (
        <ReviewDialog
          decision={reviewDecision}
          comments={reviewComments}
          onCommentsChange={setReviewComments}
          onClose={() => {
            setReviewingLoan(null);
            setReviewDecision(null);
            setReviewComments("");
          }}
          onSubmit={handleSubmitReview}
          isPending={submitReview.isPending}
          error={submitReview.error?.message ?? null}
        />
      )}
    </div>
  );
}

// --- Members Tab ---
function MembersTab({ isChair }: { isChair: boolean }) {
  const { data: membersData, isLoading } = useCommitteeMembers();

  const members = membersData?.data ?? [];

  return (
    <div>
      {isChair && (
        <div className="mb-4">
          <Link
            to="/kamati-mikopo"
            className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-sm font-semibold text-primary-foreground"
          >
            <Users className="h-4 w-4" /> Simamia Wanachama
          </Link>
        </div>
      )}

      {isLoading && (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      )}

      <div className="space-y-2">
        {members.map((m) => (
          <div key={`${m.user_id}-${m.id || "auto"}`} className="card-surface p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-semibold">{m.user_name}</p>
                <p className="text-xs text-muted-foreground">
                  {m.user_email} • {m.user_role}
                </p>
              </div>
              <div className="text-right">
                {m.id === "" ? (
                  <span className="chip bg-primary/15 text-primary text-[10px]">Otomatiki</span>
                ) : (
                  <span className="chip bg-success/15 text-success text-[10px]">Ameteuliwa</span>
                )}
              </div>
            </div>
            {m.appointed_by && (
              <p className="mt-1 text-xs text-muted-foreground">
                Ameteuliwa na: {m.appointed_by} • {m.appointed_at}
              </p>
            )}
          </div>
        ))}

        {members.length === 0 && !isLoading && (
          <div className="card-surface p-8 text-center text-sm text-muted-foreground">
            Hakuna wanachama wa kamati.
          </div>
        )}
      </div>
    </div>
  );
}

// --- History Tab ---
function HistoryTab() {
  const { data: historyData, isLoading } = useCommitteeHistory({ limit: 100 });

  const rows: Array<{
    loan_id: string;
    applicant_name: string;
    member_no: string;
    amount: number;
    status: string;
    reviewed_by: string;
    decision: string;
    comments?: string;
    reviewed_at: string;
  }> = historyData?.data ?? [];

  return (
    <div>
      {isLoading && (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      )}

      <div className="space-y-2">
        {rows.map((r, i) => (
          <div key={`${r.loan_id}-${i}`} className="card-surface p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate font-semibold">{r.applicant_name}</p>
                <p className="text-xs text-muted-foreground">
                  {r.member_no} • Mkopo #{r.loan_id}
                </p>
              </div>
              <span
                className={`chip text-[10px] ${
                  r.decision === "APPROVE"
                    ? "bg-success/15 text-success"
                    : "bg-destructive/10 text-destructive"
                }`}
              >
                {r.decision === "APPROVE" ? "Imeidhinishwa" : "Imekataliwa"}
              </span>
            </div>

            <div className="mt-2 grid grid-cols-3 gap-2 text-xs">
              <div>
                <p className="text-muted-foreground">Kiasi</p>
                <p className="font-semibold">{tzs(r.amount)}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Mkaguzi</p>
                <p className="font-semibold">{r.reviewed_by}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Tarehe</p>
                <p className="font-semibold">{r.reviewed_at}</p>
              </div>
            </div>

            {r.comments && (
              <p className="mt-2 text-xs text-muted-foreground">
                <MessageSquare className="inline h-3 w-3 mr-1" />
                {r.comments}
              </p>
            )}
          </div>
        ))}

        {rows.length === 0 && !isLoading && (
          <div className="card-surface p-8 text-center text-sm text-muted-foreground">
            Hakuna historia ya ukaguzi bado.
          </div>
        )}
      </div>
    </div>
  );
}

// --- Review Dialog ---
function ReviewDialog({
  decision,
  comments,
  onCommentsChange,
  onClose,
  onSubmit,
  isPending,
  error,
}: {
  decision: "APPROVE" | "REJECT";
  comments: string;
  onCommentsChange: (v: string) => void;
  onClose: () => void;
  onSubmit: () => void;
  isPending: boolean;
  error: string | null;
}) {
  const isApprove = decision === "APPROVE";

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-foreground/40 sm:items-center"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-t-3xl bg-card p-5 sm:rounded-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-display text-lg font-semibold">
            {isApprove ? "Idhinisha Mkopo" : "Kataa Mkopo"}
          </h3>
          <button onClick={onClose} className="rounded-lg p-1.5 hover:bg-muted">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-3">
          <div>
            <label className="text-sm font-medium">Maoni (si lazima)</label>
            <textarea
              value={comments}
              onChange={(e) => onCommentsChange(e.target.value)}
              placeholder={
                isApprove
                  ? "Maoni yako kuhusu mkopo huu..."
                  : "Sababu ya kukataa mkopo huu..."
              }
              className="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              rows={3}
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <p className="text-xs text-muted-foreground">
            {isApprove
              ? "Ukaguzi wako utahifadhiwa. Mkopo utaidhinishwa tu pale wanachama wote wa kamati walipoidhinisha."
              : "Mkopo utakataliwa mara moja ukaguzi wako ukifikishwa."}
          </p>
        </div>

        <div className="mt-4 flex gap-3">
          <button
            onClick={onClose}
            className="flex-1 rounded-xl border border-border py-2.5 text-sm font-semibold"
          >
            Ghairi
          </button>
          <button
            onClick={onSubmit}
            disabled={isPending}
            className={`flex-1 inline-flex items-center justify-center gap-2 rounded-xl py-2.5 text-sm font-semibold text-white disabled:opacity-50 ${
              isApprove ? "bg-success" : "bg-destructive"
            }`}
          >
            {isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : isApprove ? (
              <Check className="h-4 w-4" />
            ) : (
              <Ban className="h-4 w-4" />
            )}
            {isApprove ? "Thibitisha Kuidhinisha" : "Thibitisha Kukataa"}
          </button>
        </div>
      </div>
    </div>
  );
}
