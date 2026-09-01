import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useAuth } from "@/lib/auth-provider";
import { blockAdminFromPage, requireAuth, requireRole } from "@/lib/role-guards";
import { api } from "@/api/client";
import { withUploadToken } from "@/api/upload";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@/components/ui/skeleton";
import { CheckCircle, XCircle, Clock, Loader2, Eye, ImageIcon, X, Filter } from "lucide-react";

export const Route = createFileRoute("/michango")({
  head: () => ({
    meta: [
      { title: "Michango — Money Seeking" },
      { name: "description", content: "Fuatilia michango yote ya wanachama." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    // "Pokea Michango" — mweka hazina (receipting) + katibu (records);
    // mwenyekiti hana tena ruhusa hii.
    requireRole("treasurer", "secretary");
    blockAdminFromPage();
  },
  component: MichangoPage,
});

interface MemberContribution {
  id: string;
  member_id: string;
  contribution_type: "AKIBA" | "MFUKO_WA_KIJAMII";
  period_label: string;
  amount: string; // backend decimal.Decimal serializes as string
  proof_image_url?: string;
  proof_message?: string;
  status: "PENDING_VERIFICATION" | "CONFIRMED" | "REJECTED";
  review_reason?: string;
  created_at: string;
  reviewed_at?: string;
  member?: {
    id: string;
    member_no: string;
    full_name: string;
    phone: string;
  };
}

type StatusFilter = "ALL" | "PENDING_VERIFICATION" | "CONFIRMED" | "REJECTED";

function MichangoPage() {
  const { user } = useAuth();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("ALL");
  const [typeFilter, setTypeFilter] = useState<"ALL" | "AKIBA" | "MFUKO_WA_KIJAMII">("ALL");
  const [viewingContrib, setViewingContrib] = useState<MemberContribution | null>(null);

  const { data, isLoading, error } = useQuery<{ data: MemberContribution[]; total: number }>({
    queryKey: ["michango", "all"],
    queryFn: async () => {
      return api.get("/michango");
    },
  });

  if (!user) return null;

  const contributions = data?.data ?? [];

  // Apply filters
  const filtered = contributions.filter((c) => {
    if (statusFilter !== "ALL" && c.status !== statusFilter) return false;
    if (typeFilter !== "ALL" && c.contribution_type !== typeFilter) return false;
    return true;
  });

  // Summary stats
  const totalAmount = filtered.reduce((sum, c) => sum + Number(c.amount), 0);
  const pendingCount = contributions.filter((c) => c.status === "PENDING_VERIFICATION").length;
  const confirmedCount = contributions.filter((c) => c.status === "CONFIRMED").length;

  const statusConfig = {
    PENDING_VERIFICATION: { icon: Clock, label: "Inasubiri", color: "text-amber-600", bgColor: "bg-amber-100" },
    CONFIRMED: { icon: CheckCircle, label: "Imethibitishwa", color: "text-green-600", bgColor: "bg-green-100" },
    REJECTED: { icon: XCircle, label: "Imekataliwa", color: "text-red-600", bgColor: "bg-red-100" },
  };

  return (
    <AppShell
      title="Michango Zote"
      subtitle="Fuatilia michango ya wanachama wote"
    >
      {/* Summary Cards */}
      <div className="grid grid-cols-3 gap-3 mb-6">
        <div className="card-surface p-4 text-center">
          <p className="text-xs text-muted-foreground">Jumla</p>
          <p className="font-display text-xl font-bold">{contributions.length}</p>
        </div>
        <div className="card-surface p-4 text-center">
          <p className="text-xs text-muted-foreground">Inasubiri</p>
          <p className="font-display text-xl font-bold text-amber-600">{pendingCount}</p>
        </div>
        <div className="card-surface p-4 text-center">
          <p className="text-xs text-muted-foreground">Imethibitishwa</p>
          <p className="font-display text-xl font-bold text-green-600">{confirmedCount}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="card-surface p-4 mb-6">
        <div className="flex items-center gap-2 mb-3">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Chuja</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {/* Status filters */}
          {[
            { value: "ALL", label: "Zote" },
            { value: "PENDING_VERIFICATION", label: "Inasubiri" },
            { value: "CONFIRMED", label: "Imethibitishwa" },
            { value: "REJECTED", label: "Imekataliwa" },
          ].map((s) => (
            <button
              key={s.value}
              onClick={() => setStatusFilter(s.value as StatusFilter)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                statusFilter === s.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-foreground/80 hover:bg-muted/80"
              }`}
            >
              {s.label}
            </button>
          ))}
          <div className="h-6 w-px bg-border mx-1" />
          {/* Type filters */}
          {[
            { value: "ALL", label: "Aina Zote" },
            { value: "AKIBA", label: "Akiba" },
            { value: "MFUKO_WA_KIJAMII", label: "Mfuko" },
          ].map((t) => (
            <button
              key={t.value}
              onClick={() => setTypeFilter(t.value as any)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                typeFilter === t.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-foreground/80 hover:bg-muted/80"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* Total */}
      <div className="card-surface p-4 mb-4">
        <p className="text-xs text-muted-foreground">Jumla ya michango iliyochujwa</p>
        <p className="font-display text-2xl font-bold">TZS {totalAmount.toLocaleString()}</p>
        <p className="text-xs text-muted-foreground mt-1">{filtered.length} michango</p>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="card-surface p-4">
              <div className="flex gap-3">
                <Skeleton className="h-16 w-16 rounded-lg" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-3 w-48" />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Error */}
      {error && !isLoading && (
        <div className="card-surface p-8 text-center">
          <p className="text-sm text-destructive">{error.message}</p>
        </div>
      )}

      {/* Contributions List */}
      {!isLoading && !error && (
        <div className="space-y-3">
          {filtered.length === 0 ? (
            <div className="card-surface p-8 text-center">
              <p className="text-muted-foreground">Hakuna michango kwa kichuja hiki</p>
            </div>
          ) : (
            filtered.map((contrib) => {
              const status = statusConfig[contrib.status];
              const StatusIcon = status.icon;
              return (
                <div
                  key={contrib.id}
                  className="card-surface p-4 cursor-pointer hover:bg-muted/30 transition-colors"
                  onClick={() => setViewingContrib(contrib)}
                >
                  <div className="flex gap-3">
                    {/* Thumbnail */}
                    {contrib.proof_image_url ? (
                      <div className="h-16 w-16 shrink-0 overflow-hidden rounded-lg border bg-muted">
                        <img
                          src={withUploadToken(contrib.proof_image_url)}
                          alt="Uthibitisho"
                          className="h-full w-full object-cover"
                        />
                      </div>
                    ) : (
                      <div className="h-16 w-16 shrink-0 flex items-center justify-center rounded-lg border bg-muted/50">
                        <ImageIcon className="h-6 w-6 text-muted-foreground/50" />
                      </div>
                    )}

                    {/* Details */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`chip ${contrib.contribution_type === "AKIBA" ? "bg-blue-100 text-blue-700" : "bg-purple-100 text-purple-700"} text-[10px] font-semibold px-2 py-0.5 rounded`}>
                          {contrib.contribution_type === "AKIBA" ? "Akiba" : "Mfuko"}
                        </span>
                        <span className={`chip ${status.bgColor} ${status.color} text-[10px] font-semibold px-2 py-0.5 rounded inline-flex items-center gap-1`}>
                          <StatusIcon className="h-3 w-3" />
                          {status.label}
                        </span>
                      </div>
                      <p className="font-display text-base font-bold">TZS {Number(contrib.amount).toLocaleString()}</p>
                      <p className="text-sm text-muted-foreground">
                        {contrib.member?.full_name} ({contrib.member?.member_no})
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {contrib.period_label} • {new Date(contrib.created_at).toLocaleDateString("sw-TZ")}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}

      {/* View Details Modal */}
      {viewingContrib && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center sm:p-4"
          onClick={() => setViewingContrib(null)}
        >
          <div
            className="w-full max-w-lg rounded-t-3xl bg-card sm:rounded-2xl max-h-[90vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-card px-5 py-4">
              <h3 className="font-display text-lg font-semibold">Maelezo ya Mchango</h3>
              <button
                onClick={() => setViewingContrib(null)}
                className="rounded-lg p-1.5 hover:bg-muted"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="p-5 space-y-4">
              {/* Proof Image */}
              {viewingContrib.proof_image_url && (
                <div className="rounded-xl overflow-hidden border bg-muted">
                  <img
                    src={withUploadToken(viewingContrib.proof_image_url)}
                    alt="Picha ya uthibitisho"
                    className="w-full max-h-80 object-contain"
                  />
                </div>
              )}

              {/* Info */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs text-muted-foreground">Mwanachama</p>
                  <p className="font-semibold">{viewingContrib.member?.full_name}</p>
                  <p className="text-xs text-muted-foreground">{viewingContrib.member?.member_no}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Kiasi</p>
                  <p className="font-display text-2xl font-bold">TZS {Number(viewingContrib.amount).toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Aina</p>
                  <span className={`chip ${viewingContrib.contribution_type === "AKIBA" ? "bg-blue-100 text-blue-700" : "bg-purple-100 text-purple-700"} text-xs font-semibold px-2 py-0.5 rounded`}>
                    {viewingContrib.contribution_type === "AKIBA" ? "Akiba" : "Mfuko wa Kijamii"}
                  </span>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Hali</p>
                  <span className={`chip ${statusConfig[viewingContrib.status].bgColor} ${statusConfig[viewingContrib.status].color} text-xs font-semibold px-2 py-0.5 rounded`}>
                    {statusConfig[viewingContrib.status].label}
                  </span>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Kipindi</p>
                  <p className="font-semibold">{viewingContrib.period_label}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Tarehe</p>
                  <p className="font-semibold">{new Date(viewingContrib.created_at).toLocaleDateString("sw-TZ")}</p>
                </div>
              </div>

              {/* Proof Message */}
              {viewingContrib.proof_message && (
                <div className="p-3 bg-muted/50 rounded-lg">
                  <p className="text-xs text-muted-foreground mb-1">Ujumbe wa Muamala</p>
                  <p className="text-sm">{viewingContrib.proof_message}</p>
                </div>
              )}

              {/* Rejection Reason */}
              {viewingContrib.status === "REJECTED" && viewingContrib.review_reason && (
                <div className="p-3 bg-destructive/10 rounded-lg">
                  <p className="text-xs text-destructive mb-1">Sababu ya Kukataliwa</p>
                  <p className="text-sm text-destructive">{viewingContrib.review_reason}</p>
                </div>
              )}

              {/* Close Button */}
              <button
                onClick={() => setViewingContrib(null)}
                className="w-full inline-flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-3 font-medium text-foreground/80 hover:bg-muted"
              >
                Funga
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
