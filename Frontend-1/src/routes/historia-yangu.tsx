import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useQuery } from "@tanstack/react-query";
import { useState, useMemo } from "react";
import { Loader2, PiggyBank, Banknote, CheckCircle, XCircle, Clock } from "lucide-react";

export const Route = createFileRoute("/historia-yangu")({
  beforeLoad: () => {
    requireAuth();
  },
  component: HistoriaYanguPage,
});

interface Activity {
  id: string;
  type: "CONTRIBUTION" | "LOAN" | "REPAYMENT";
  date: string;
  amount: number;
  status: string;
  description: string;
  icon: any;
  color: string;
}

function HistoriaYanguPage() {
  const { user } = useAuth();
  const [filter, setFilter] = useState<"ALL" | "CONTRIBUTION" | "LOAN">("ALL");

  // Fetch contributions
  const { data: contribData, isLoading: contribLoading } = useQuery<{ data: any[] }>({
    queryKey: ["michango", "mine"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/michango/mine", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana");
      return res.json();
    },
  });

  // Fetch loans
  const { data: loanData, isLoading: loanLoading } = useQuery<{ data: any[] }>({
    queryKey: ["mikopo", "mine"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/mikopo?member_id=self", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana");
      return res.json();
    },
  });

  if (!user) return null;

  const isLoading = contribLoading || loanLoading;

  // Merge and sort activities
  const activities = useMemo(() => {
    const all: Activity[] = [];

    // Add contributions
    contribData?.data?.forEach((c: any) => {
      all.push({
        id: c.id,
        type: "CONTRIBUTION",
        date: c.created_at,
        amount: c.amount,
        status: c.status,
        description: `${c.contribution_type} - ${c.period_label}`,
        icon: PiggyBank,
        color: c.status === "CONFIRMED" ? "text-green-600 bg-green-100" : 
               c.status === "REJECTED" ? "text-red-600 bg-red-100" : 
               "text-amber-600 bg-amber-100",
      });
    });

    // Add loans
    loanData?.data?.forEach((l: any) => {
      all.push({
        id: l.id,
        type: "LOAN",
        date: l.applied_at,
        amount: l.amount,
        status: l.status,
        description: l.purpose || "Mkopo",
        icon: Banknote,
        color: l.status === "APPROVED" || l.status === "OUTSTANDING" ? "text-blue-600 bg-blue-100" : 
               l.status === "REJECTED" ? "text-red-600 bg-red-100" : 
               "text-amber-600 bg-amber-100",
      });
    });

    // Sort by date descending
    return all.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
  }, [contribData, loanData]);

  const filteredActivities = filter === "ALL" ? activities : activities.filter((a) => a.type === filter);

  return (
    <AppShell title="Historia Yangu" subtitle="Michango na mikopo yako">
      {/* Filter Tabs */}
      <div className="flex gap-2 mb-6">
        {[
          { value: "ALL", label: "Zote" },
          { value: "CONTRIBUTION", label: "Michango" },
          { value: "LOAN", label: "Mikopo" },
        ].map((tab) => (
          <button
            key={tab.value}
            onClick={() => setFilter(tab.value as any)}
            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
              filter === tab.value
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-foreground/80 hover:bg-muted/80"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : filteredActivities.length === 0 ? (
        <div className="card-surface p-12 text-center">
          <p className="text-muted-foreground">Hakuna historia bado</p>
        </div>
      ) : (
        <div className="space-y-3">
          {filteredActivities.map((activity) => {
            const Icon = activity.icon;
            return (
              <div key={`${activity.type}-${activity.id}`} className="card-surface p-5">
                <div className="flex items-start gap-4">
                  <div className={`grid h-12 w-12 place-items-center rounded-xl ${activity.color}`}>
                    <Icon className="h-6 w-6" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <p className="font-semibold">{activity.description}</p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {new Date(activity.date).toLocaleDateString("sw-TZ", {
                            year: "numeric",
                            month: "long",
                            day: "numeric",
                          })}
                        </p>
                      </div>
                      <p className="font-display text-lg font-bold">TZS {activity.amount.toLocaleString()}</p>
                    </div>
                    <div className="mt-2">
                      <span className={`chip ${activity.color} text-[10px] font-semibold px-2 py-0.5 rounded`}>
                        {activity.status}
                      </span>
                    </div>
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
