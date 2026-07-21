import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireLeadership } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCircle, XCircle, Clock, Loader2, Eye } from "lucide-react";

export const Route = createFileRoute("/michango-inayosubiri")({
  beforeLoad: () => {
    requireLeadership("MWENYEKITI", "HAZINA", "KATIBU");
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
            <div key={contrib.id} className="card-surface p-5">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <span className={`chip ${contrib.contribution_type === "AKIBA" ? "bg-blue-100 text-blue-700" : "bg-purple-100 text-purple-700"} text-[10px] font-semibold px-2 py-0.5 rounded`}>
                      {contrib.contribution_type === "AKIBA" ? "Akiba" : "Mfuko"}
                    </span>
                    <span className="chip bg-warning/25 text-foreground text-[10px] font-semibold px-2 py-0.5 rounded inline-flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      Inasubiri
                    </span>
                  </div>
                  <p className="font-display text-lg font-bold">TZS {contrib.amount.toLocaleString()}</p>
                  <p className="text-sm text-muted-foreground mt-1">
                    {contrib.member?.full_name} ({contrib.member?.member_no})
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Kipindi: {contrib.period_label} • {new Date(contrib.created_at).toLocaleDateString("sw-TZ")}
                  </p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => setViewingContrib(contrib)}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium text-foreground/80 hover:bg-muted"
                  >
                    <Eye className="h-4 w-4" />
                    Angalia
                  </button>
                  {canConfirm(contrib) && (
                    <>
                      <button
                        onClick={() => {
                          if (confirm("Thibitisha mchango huu?")) {
                            confirmMutation.mutate(contrib.id);
                          }
                        }}
                        disabled={confirmMutation.isPending}
                        className="inline-flex items-center gap-1.5 rounded-lg bg-green-600 px-3 py-2 text-sm font-semibold text-white hover:bg-green-700 disabled:opacity-50"
                      >
                        <CheckCircle className="h-4 w-4" />
                        Thibitisha
                      </button>
                      <button
                        onClick={() => {
                          const reason = prompt("Sababu ya kukataa:");
                          if (reason) {
                            rejectMutation.mutate({ id: contrib.id, reason });
                          }
                        }}
                        disabled={rejectMutation.isPending}
                        className="inline-flex items-center gap-1.5 rounded-lg bg-destructive px-3 py-2 text-sm font-semibold text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
                      >
                        <XCircle className="h-4 w-4" />
                        Kataa
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* View Details Modal */}
      {viewingContrib && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setViewingContrib(null)}>
          <div className="card-surface max-w-2xl w-full p-6" onClick={(e) => e.stopPropagation()}>
            <h3 className="font-display text-xl font-semibold mb-4">Maelezo ya Mchango</h3>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Mwanachama</p>
                  <p className="font-semibold">{viewingContrib.member?.full_name}</p>
                  <p className="text-sm text-muted-foreground">{viewingContrib.member?.member_no}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Kiasi</p>
                  <p className="font-display text-2xl font-bold">TZS {viewingContrib.amount.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Aina</p>
                  <p className="font-semibold">{viewingContrib.contribution_type}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Kipindi</p>
                  <p className="font-semibold">{viewingContrib.period_label}</p>
                </div>
              </div>
              {viewingContrib.proof_image_url && (
                <div>
                  <p className="text-sm text-muted-foreground mb-2">Picha ya Uthibitisho</p>
                  <img src={viewingContrib.proof_image_url} alt="Proof" className="max-w-full rounded-lg border" />
                </div>
              )}
              {viewingContrib.proof_message && (
                <div>
                  <p className="text-sm text-muted-foreground mb-2">Ujumbe wa Muamala</p>
                  <div className="p-3 bg-muted/50 rounded-lg">
                    <p className="text-sm">{viewingContrib.proof_message}</p>
                  </div>
                </div>
              )}
              <div className="flex gap-2 pt-4">
                {canConfirm(viewingContrib) && (
                  <>
                    <button
                      onClick={() => {
                        if (confirm("Thibitisha mchango huu?")) {
                          confirmMutation.mutate(viewingContrib.id);
                        }
                      }}
                      disabled={confirmMutation.isPending}
                      className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2 font-semibold text-white hover:bg-green-700 disabled:opacity-50"
                    >
                      <CheckCircle className="h-4 w-4" />
                      Thibitisha
                    </button>
                    <button
                      onClick={() => {
                        const reason = prompt("Sababu ya kukataa:");
                        if (reason) {
                          rejectMutation.mutate({ id: viewingContrib.id, reason });
                        }
                      }}
                      disabled={rejectMutation.isPending}
                      className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-destructive px-4 py-2 font-semibold text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
                    >
                      <XCircle className="h-4 w-4" />
                      Kataa
                    </button>
                  </>
                )}
                <button
                  onClick={() => setViewingContrib(null)}
                  className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg border border-border px-4 py-2 font-medium text-foreground/80 hover:bg-muted"
                >
                  Funga
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
