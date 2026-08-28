import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { api } from "@/api/client";
import { AppShell } from "@/components/AppShell";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle, XCircle, Clock, Loader2 } from "lucide-react";

export const Route = createFileRoute("/michango-yangu")({
  beforeLoad: () => {
    requireAuth();
  },
  component: MichangoYanguPage,
});

interface MemberContribution {
  id: string;
  contribution_type: "AKIBA" | "MFUKO_WA_KIJAMII";
  period_label: string;
  amount: number;
  proof_image_url?: string;
  proof_message?: string;
  status: "PENDING_VERIFICATION" | "CONFIRMED" | "REJECTED";
  review_reason?: string;
  created_at: string;
  reviewed_at?: string;
}

function MichangoYanguPage() {
  const { user } = useAuth();

  const { data, isLoading } = useQuery<{ data: MemberContribution[]; total: number }>({
    queryKey: ["michango", "mine"],
    queryFn: async () => {
      return api.get("/michango/mine");
    },
  });

  if (!user) return null;

  const contributions = data?.data ?? [];

  const statusConfig = {
    PENDING_VERIFICATION: { icon: Clock, label: "Inasubiri", color: "bg-warning/25 text-foreground" },
    CONFIRMED: { icon: CheckCircle, label: "Imethibitishwa", color: "bg-success/25 text-success" },
    REJECTED: { icon: XCircle, label: "Imekataliwa", color: "bg-destructive/10 text-destructive" },
  };

  return (
    <AppShell title="Michango Yangu" subtitle="Historia ya michango yako">
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : contributions.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <p className="text-muted-foreground">Huna michango bado</p>
          <p className="text-sm text-muted-foreground mt-2">Bofya "Weka Mchango" kuanza</p>
        </div>
      ) : (
        <div className="space-y-3">
          {contributions.map((contrib) => {
            const status = statusConfig[contrib.status];
            const StatusIcon = status.icon;
            return (
              <div key={contrib.id} className="card-surface p-5">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <span className={`chip ${contrib.contribution_type === "AKIBA" ? "bg-blue-100 text-blue-700" : "bg-purple-100 text-purple-700"} text-[10px] font-semibold px-2 py-0.5 rounded`}>
                        {contrib.contribution_type === "AKIBA" ? "Akiba" : "Mfuko"}
                      </span>
                      <span className={`chip ${status.color} text-[10px] font-semibold px-2 py-0.5 rounded inline-flex items-center gap-1`}>
                        <StatusIcon className="h-3 w-3" />
                        {status.label}
                      </span>
                    </div>
                    <p className="font-display text-lg font-bold">TZS {contrib.amount.toLocaleString()}</p>
                    <p className="text-sm text-muted-foreground mt-1">Kipindi: {contrib.period_label}</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Tarehe: {new Date(contrib.created_at).toLocaleDateString("sw-TZ")}
                    </p>
                    {contrib.proof_image_url && (
                      <div className="mt-3">
                        <p className="text-xs text-muted-foreground mb-1">Picha ya Uthibitisho:</p>
                        <img
                          src={contrib.proof_image_url}
                          alt="Uthibitisho"
                          className="max-h-48 rounded-lg border object-contain"
                        />
                      </div>
                    )}
                    {contrib.proof_message && (
                      <div className="mt-3 p-3 bg-muted/50 rounded-lg">
                        <p className="text-xs text-muted-foreground mb-1">Ujumbe wa Muamala:</p>
                        <p className="text-sm">{contrib.proof_message}</p>
                      </div>
                    )}
                    {contrib.review_reason && (
                      <div className="mt-3 p-3 bg-destructive/10 rounded-lg">
                        <p className="text-xs text-destructive mb-1">Sababu ya Kukataliwa:</p>
                        <p className="text-sm text-destructive">{contrib.review_reason}</p>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </AppShell>
  );
}
