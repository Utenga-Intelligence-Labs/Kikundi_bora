import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useQuery } from "@tanstack/react-query";
import { useState, useMemo } from "react";
import { Loader2, PiggyBank, Banknote, Receipt, CheckCircle, XCircle, Clock, ArrowDownLeft } from "lucide-react";

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
  detail?: string;
  icon: any;
  color: string;
  bgColor: string;
}

function HistoriaYanguPage() {
  const { user } = useAuth();
  const [filter, setFilter] = useState<"ALL" | "CONTRIBUTION" | "LOAN" | "REPAYMENT">("ALL");

  // Fetch contributions
  const { data: contribData, isLoading: contribLoading } = useQuery<{ data: any[] }>({
    queryKey: ["michango", "mine"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/michango/mine", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata michango");
      return res.json();
    },
  });

  // Fetch loans (backend auto-filters for members)
  const { data: loanData, isLoading: loanLoading } = useQuery<{ data: any[] }>({
    queryKey: ["mikopo", "mine"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/loans", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata mikopo");
      return res.json();
    },
  });

  // Fetch repayments (backend auto-filters for members)
  const { data: repaymentData, isLoading: repaymentLoading } = useQuery<{ data: any[] }>({
    queryKey: ["marejesho", "mine"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/repayments", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata marejesho");
      return res.json();
    },
  });

  if (!user) return null;

  const isLoading = contribLoading || loanLoading || repaymentLoading;

  // Merge and sort activities
  const activities = useMemo(() => {
    const all: Activity[] = [];

    // Add contributions (AKIBA + MFUKO_WA_KIJAMII)
    contribData?.data?.forEach((c: any) => {
      const isAkiba = c.contribution_type === "AKIBA";
      all.push({
        id: c.id,
        type: "CONTRIBUTION",
        date: c.created_at,
        amount: c.amount,
        status: c.status,
        description: isAkiba ? "Mchango wa Akiba" : "Mchango wa Mfuko wa Kijamii",
        detail: c.period_label,
        icon: PiggyBank,
        color: c.status === "CONFIRMED" ? "text-green-600" :
               c.status === "REJECTED" ? "text-red-600" :
               "text-amber-600",
        bgColor: c.status === "CONFIRMED" ? "bg-green-100" :
                 c.status === "REJECTED" ? "bg-red-100" :
                 "bg-amber-100",
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
        description: "Ombi la Mkopo",
        detail: l.purpose || undefined,
        icon: Banknote,
        color: l.status === "APPROVED" || l.status === "OUTSTANDING" ? "text-blue-600" :
               l.status === "CLOSED" ? "text-green-600" :
               l.status === "REJECTED" ? "text-red-600" :
               "text-amber-600",
        bgColor: l.status === "APPROVED" || l.status === "OUTSTANDING" ? "bg-blue-100" :
                 l.status === "CLOSED" ? "bg-green-100" :
                 l.status === "REJECTED" ? "bg-red-100" :
                 "bg-amber-100",
      });
    });

    // Add repayments
    repaymentData?.data?.forEach((r: any) => {
      all.push({
        id: r.id,
        type: "REPAYMENT",
        date: r.paid_at || r.created_at,
        amount: r.amount,
        status: "PAID",
        description: "Rejesho la Mkopo",
        detail: r.payment_method,
        icon: ArrowDownLeft,
        color: "text-green-600",
        bgColor: "bg-green-100",
      });
    });

    // Sort by date descending (newest first)
    return all.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
  }, [contribData, loanData, repaymentData]);

  const filteredActivities = filter === "ALL" ? activities : activities.filter((a) => a.type === filter);

  const statusLabel = (status: string) => {
    switch (status) {
      case "CONFIRMED": return "Imethibitishwa";
      case "REJECTED": return "Imekataliwa";
      case "PENDING_VERIFICATION": return "Inasubiri";
      case "PENDING": return "Inasubiri";
      case "UNDER_REVIEW": return "Inapitiwa";
      case "APPROVED": return "Imeidhinishwa";
      case "OUTSTANDING": return "Wazi";
      case "CLOSED": return "Imefungwa";
      case "PAID": return "Imelipwa";
      default: return status;
    }
  };

  const typeLabel = (type: string) => {
    switch (type) {
      case "CONTRIBUTION": return "Mchango";
      case "LOAN": return "Mkopo";
      case "REPAYMENT": return "Rejesho";
      default: return type;
    }
  };

  return (
    <AppShell title="Historia Yangu" subtitle="Michango, mikopo na marejesho yako yote">
      {/* Filter Tabs */}
      <div className="flex gap-2 mb-6 overflow-x-auto pb-1">
        {[
          { value: "ALL", label: "Zote", count: activities.length },
          { value: "CONTRIBUTION", label: "Michango", count: activities.filter(a => a.type === "CONTRIBUTION").length },
          { value: "LOAN", label: "Mikopo", count: activities.filter(a => a.type === "LOAN").length },
          { value: "REPAYMENT", label: "Marejesho", count: activities.filter(a => a.type === "REPAYMENT").length },
        ].map((tab) => (
          <button
            key={tab.value}
            onClick={() => setFilter(tab.value as any)}
            className={`shrink-0 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              filter === tab.value
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-foreground/80 hover:bg-muted/80"
            }`}
          >
            {tab.label} ({tab.count})
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
              <div key={`${activity.type}-${activity.id}`} className="card-surface p-4">
                <div className="flex items-start gap-3">
                  <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-xl ${activity.bgColor}`}>
                    <Icon className={`h-5 w-5 ${activity.color}`} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="font-semibold text-sm">{activity.description}</p>
                        {activity.detail && (
                          <p className="text-xs text-muted-foreground mt-0.5">{activity.detail}</p>
                        )}
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {new Date(activity.date).toLocaleDateString("sw-TZ", {
                            year: "numeric",
                            month: "long",
                            day: "numeric",
                          })}
                        </p>
                      </div>
                      <div className="text-right shrink-0">
                        <p className={`font-display text-base font-bold ${activity.type === "REPAYMENT" ? "text-green-600" : ""}`}>
                          {activity.type === "REPAYMENT" ? "+" : ""}TZS {activity.amount.toLocaleString()}
                        </p>
                        <span className={`chip ${activity.bgColor} ${activity.color} text-[10px] font-semibold px-2 py-0.5 rounded mt-1 inline-block`}>
                          {statusLabel(activity.status)}
                        </span>
                      </div>
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
