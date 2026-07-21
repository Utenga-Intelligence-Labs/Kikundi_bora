import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireRole } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCircle, XCircle, Clock, Loader2, Eye, ImageIcon, X } from "lucide-react";

export const Route = createFileRoute("/michango-inayosubiri")({
  beforeLoad: () => {
    requireRole("chair", "treasurer", "secretary");
  },
  component: MichangoInayosubiriPage,
});

interface MemberContribution {
  id: string;
  member_id: string;
  contribution_type: "AKIBA" | "MFUKO_WA_KIJAMII";
  period_label: string;
  amount: number;
  proof_image_url?: string;
  proof_message?: string;
  status: "PENDING_VERIFICATION" | "CONFIRMED" | "REJECTED";
  created_at: string;
  member?: {
    id: string;
    member_no: string;
    full_name: string;
    phone: string;
  };
}

function MichangoInayosubiriPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [viewingContrib, setViewingContrib] = useState<MemberContribution | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [showRejectInput, setShowRejectInput] = useState(false);

  const { data, isLoading } = useQuery<{ data: MemberContribution[]; total: number }>({
    queryKey: ["michango", "pending"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/michango/pending", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata michango");
      return res.json();
    },
  });

  const confirmMutation = useMutation({
    mutationFn: async (id: string) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch(`/api/v1/michango/${id}/confirm`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.message || "Imeshindikana kuthibitisha");
      }
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      setViewingContrib(null);
      alert("Mchango umethibitishwa!");
    },
    onError: (err: Error) => {
      alert(err.message);
    },
  });

  const rejectMutation = useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason: string }) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch(`/api/v1/michango/${id}/reject`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ reason }),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.message || "Imeshindikana kukataa");
      }
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      setViewingContrib(null);
      setRejectReason("");
      setShowRejectInput(false);
      alert("Mchango umekataliwa");
    },
    onError: (err: Error) => {
      alert(err.message);
    },
  });

  if (!user) return null;

  const contributions = data?.data ?? [];

  const canConfirm = (contrib: MemberContribution) => {
    if (contrib.contribution_type === "AKIBA" && user.role === "treasurer") return true;
    if (contrib.contribution_type === "MFUKO_WA_KIJAMII" && user.role === "chair") return true;
    return false;
  };

  const handleReject = () => {
    if (!viewingContrib) return;
    if (!rejectReason.trim()) {
      alert("Andika sababu ya kukataa");
      return;
    }
    rejectMutation.mutate({ id: viewingContrib.id, reason: rejectReason });
  };

  const handleConfirm = (id: string) => {
    if (confirm("Thibitisha mchango huu?")) {
      confirmMutation.mutate(id);
    }
  };

  return (
    <AppShell title="Michango Inayosubiri" subtitle="Thibitisha au kataa michango ya wanachama">
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : contributions.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <CheckCircle className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <p className="mt-4 text-muted-foreground">Hakuna michango inayosubiri</p>
        </div>
      ) : (
        <div className="space-y-3">
          {contributions.map((contrib) => (
            <div key={contrib.id} className="card-surface p-4">
              <div className="flex gap-4">
                {/* Thumbnail */}
                {contrib.proof_image_url ? (
                  <div
                    className="h-20 w-20 shrink-0 overflow-hidden rounded-lg border bg-muted cursor-pointer"
                    onClick={() => setViewingContrib(contrib)}
                  >
                    <img
                      src={contrib.proof_image_url}
                      alt="Uthibitisho"
                      className="h-full w-full object-cover"
                    />
                  </div>
                ) : (
                  <div className="h-20 w-20 shrink-0 flex items-center justify-center rounded-lg border bg-muted/50">
                    <ImageIcon className="h-6 w-6 text-muted-foreground/50" />
                  </div>
                )}

                {/* Details */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={`chip ${contrib.contribution_type === "AKIBA" ? "bg-blue-100 text-blue-700" : "bg-purple-100 text-purple-700"} text-[10px] font-semibold px-2 py-0.5 rounded`}>
                      {contrib.contribution_type === "AKIBA" ? "Akiba" : "Mfuko wa Kijamii"}
                    </span>
                    <span className="chip bg-warning/25 text-foreground text-[10px] font-semibold px-2 py-0.5 rounded inline-flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      Inasubiri
                    </span>
                  </div>
                  <p className="font-display text-lg font-bold">TZS {contrib.amount.toLocaleString()}</p>
                  <p className="text-sm text-muted-foreground mt-0.5">
                    {contrib.member?.full_name} ({contrib.member?.member_no})
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Kipindi: {contrib.period_label} • {new Date(contrib.created_at).toLocaleDateString("sw-TZ")}
                  </p>
                  {contrib.proof_message && (
                    <p className="text-xs text-muted-foreground mt-1 truncate">
                      💬 {contrib.proof_message}
                    </p>
                  )}
                </div>
              </div>

              {/* Action buttons */}
              <div className="flex gap-2 mt-3 pt-3 border-t border-border">
                <button
                  onClick={() => setViewingContrib(contrib)}
                  className="flex-1 inline-flex items-center justify-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground/80 hover:bg-muted"
                >
                  <Eye className="h-4 w-4" />
                  Angalia
                </button>
                {canConfirm(contrib) && (
                  <>
                    <button
                      onClick={() => handleConfirm(contrib.id)}
                      disabled={confirmMutation.isPending}
                      className="flex-1 inline-flex items-center justify-center gap-1.5 rounded-lg bg-green-600 px-3 py-2 text-sm font-semibold text-white hover:bg-green-700 disabled:opacity-50"
                    >
                      {confirmMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <CheckCircle className="h-4 w-4" />
                      )}
                      Thibitisha
                    </button>
                    <button
                      onClick={() => {
                        setViewingContrib(contrib);
                        setShowRejectInput(true);
                      }}
                      className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-destructive px-3 py-2 text-sm font-semibold text-destructive-foreground hover:bg-destructive/90"
                    >
                      <XCircle className="h-4 w-4" />
                      Kataa
                    </button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* View Details Modal */}
      {viewingContrib && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center sm:p-4"
          onClick={() => {
            setViewingContrib(null);
            setRejectReason("");
            setShowRejectInput(false);
          }}
        >
          <div
            className="w-full max-w-lg rounded-t-3xl bg-card sm:rounded-2xl max-h-[90vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-card px-5 py-4">
              <h3 className="font-display text-lg font-semibold">Maelezo ya Mchango</h3>
              <button
                onClick={() => {
                  setViewingContrib(null);
                  setRejectReason("");
                  setShowRejectInput(false);
                }}
                className="rounded-lg p-1.5 hover:bg-muted"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="p-5 space-y-4">
              {/* Proof Image — Prominent */}
              {viewingContrib.proof_image_url && (
                <div className="rounded-xl overflow-hidden border bg-muted">
                  <img
                    src={viewingContrib.proof_image_url}
                    alt="Picha ya uthibitisho"
                    className="w-full max-h-80 object-contain"
                  />
                </div>
              )}

              {/* Contribution Info */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs text-muted-foreground">Mwanachama</p>
                  <p className="font-semibold">{viewingContrib.member?.full_name}</p>
                  <p className="text-xs text-muted-foreground">{viewingContrib.member?.member_no}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Kiasi</p>
                  <p className="font-display text-2xl font-bold">TZS {viewingContrib.amount.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Aina ya Mchango</p>
                  <span className={`chip ${viewingContrib.contribution_type === "AKIBA" ? "bg-blue-100 text-blue-700" : "bg-purple-100 text-purple-700"} text-xs font-semibold px-2 py-0.5 rounded`}>
                    {viewingContrib.contribution_type === "AKIBA" ? "Akiba" : "Mfuko wa Kijamii"}
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

              {/* Reject Reason Input */}
              {showRejectInput && (
                <div className="p-4 bg-destructive/5 border border-destructive/20 rounded-lg space-y-3">
                  <label className="block text-sm font-medium text-destructive">
                    Sababu ya kukataa
                  </label>
                  <textarea
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    placeholder="Andika sababu ya kukataa mchango huu..."
                    rows={3}
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-destructive"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={() => {
                        setShowRejectInput(false);
                        setRejectReason("");
                      }}
                      className="flex-1 rounded-lg border border-border py-2 text-sm font-medium hover:bg-muted"
                    >
                      Ghairi
                    </button>
                    <button
                      onClick={handleReject}
                      disabled={rejectMutation.isPending || !rejectReason.trim()}
                      className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-destructive py-2 text-sm font-semibold text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
                    >
                      {rejectMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <XCircle className="h-4 w-4" />
                      )}
                      Kataa Mchango
                    </button>
                  </div>
                </div>
              )}

              {/* Action Buttons */}
              {!showRejectInput && (
                <div className="flex gap-2 pt-2">
                  {canConfirm(viewingContrib) && (
                    <>
                      <button
                        onClick={() => handleConfirm(viewingContrib.id)}
                        disabled={confirmMutation.isPending}
                        className="flex-1 inline-flex items-center justify-center gap-2 rounded-xl bg-green-600 px-4 py-3 font-semibold text-white hover:bg-green-700 disabled:opacity-50"
                      >
                        {confirmMutation.isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <CheckCircle className="h-4 w-4" />
                        )}
                        Thibitisha
                      </button>
                      <button
                        onClick={() => setShowRejectInput(true)}
                        className="flex-1 inline-flex items-center justify-center gap-2 rounded-xl bg-destructive px-4 py-3 font-semibold text-destructive-foreground hover:bg-destructive/90"
                      >
                        <XCircle className="h-4 w-4" />
                        Kataa
                      </button>
                    </>
                  )}
                  <button
                    onClick={() => {
                      setViewingContrib(null);
                      setRejectReason("");
                      setShowRejectInput(false);
                    }}
                    className="flex-1 inline-flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-3 font-medium text-foreground/80 hover:bg-muted"
                  >
                    Funga
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
