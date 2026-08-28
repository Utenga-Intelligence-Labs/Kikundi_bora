import { createFileRoute } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { tokenStorage } from "@/lib/auth-storage";
import { AppShell } from "@/components/AppShell";
import { FileBarChart2, Download, Users, PiggyBank, Banknote, TrendingUp, Loader2, Clock } from "lucide-react";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

export const Route = createFileRoute("/uongozi/ripoti")({
  beforeLoad: () => {
    requireAuth();
  },
  component: RipotiPage,
});

interface QuickStats {
  total_members: number;
  contributions_month: number;
  pending_contributions: number;
  outstanding_loans: number;
  pending_loans: number;
  treasury_balance: number;
  total_contributions: number;
  total_repayments: number;
  total_disbursed: number;
}

function RipotiPage() {
  const { user, isLeadership } = useAuth();
  const [reportType, setReportType] = useState("summary");
  const [loading, setLoading] = useState(false);

  // Fetch quick stats from backend
  const { data: stats, isLoading: statsLoading } = useQuery<QuickStats>({
    queryKey: ["uongozi", "quick-stats"],
    queryFn: async () => {
      const token = tokenStorage.get();
      const res = await fetch("/api/v1/uongozi/quick-stats", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata takwimu");
      return res.json();
    },
    enabled: isLeadership,
  });

  if (!user || !isLeadership) {
    return (
      <AppShell title="Ripoti za Kikundi">
        <div className="card-surface p-12 text-center">
          <p className="text-muted-foreground">Huna ruhusa ya kufikia ukurasa huu</p>
        </div>
      </AppShell>
    );
  }

  const handleDownload = async () => {
    setLoading(true);
    try {
      const token = tokenStorage.get();
      const res = await fetch(`/api/v1/uongozi/ripoti?type=${reportType}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata ripoti");
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `ripoti-${reportType}-${Date.now()}.csv`;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error(err);
      alert("Imeshindikana kupakua ripoti");
    } finally {
      setLoading(false);
    }
  };

  const reports = [
    { value: "summary", label: "Muhtasari", desc: "Takwimu za jumla za kikundi" },
    { value: "wanachama", label: "Wanachama", desc: "Orodha ya wanachama wote" },
    { value: "michango", label: "Michango", desc: "Ripoti ya michango ya mwezi" },
    { value: "mikopo", label: "Mikopo", desc: "Ripoti ya mikopo yote" },
    { value: "mapato", label: "Mapato na Matumizi", desc: "Ripoti ya kifedha" },
  ];

  return (
    <AppShell title="Ripoti za Kikundi" subtitle="Pakua ripoti za kikundi chako">
      <div className="space-y-6">
        <div className="card-surface p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="grid h-10 w-10 place-items-center rounded-xl bg-amber-100 text-amber-700">
              <FileBarChart2 className="h-5 w-5" />
            </div>
            <div>
              <h2 className="font-display text-lg font-semibold">Aina ya Ripoti</h2>
              <p className="text-sm text-muted-foreground">Chagua ripoti unayotaka kupakua</p>
            </div>
          </div>

          <div className="space-y-2">
            {reports.map((r) => (
              <label
                key={r.value}
                className={`flex items-start gap-3 rounded-lg border p-4 cursor-pointer transition-colors ${
                  reportType === r.value
                    ? "border-amber-500 bg-amber-50"
                    : "border-border hover:bg-muted/50"
                }`}
              >
                <input
                  type="radio"
                  name="reportType"
                  value={r.value}
                  checked={reportType === r.value}
                  onChange={(e) => setReportType(e.target.value)}
                  className="mt-0.5"
                />
                <div>
                  <p className="font-medium">{r.label}</p>
                  <p className="text-sm text-muted-foreground">{r.desc}</p>
                </div>
              </label>
            ))}
          </div>

          <button
            onClick={handleDownload}
            disabled={loading}
            className="mt-6 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-amber-600 px-4 py-3 font-semibold text-white transition-colors hover:bg-amber-700 disabled:opacity-50"
          >
            <Download className="h-4 w-4" />
            {loading ? "Inapakua..." : "Pakua Ripoti"}
          </button>
        </div>

        <div className="card-surface p-6">
          <h3 className="font-display text-lg font-semibold mb-4">Takwimu za Haraka</h3>
          {statsLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : stats ? (
            <div className="grid gap-4 sm:grid-cols-2">
              <StatCard icon={Users} label="Wanachama" value={stats.total_members.toString()} color="blue" />
              <StatCard
                icon={PiggyBank}
                label="Michango Mwezi Huu"
                value={`TZS ${stats.contributions_month.toLocaleString()}`}
                color="green"
              />
              <StatCard
                icon={Banknote}
                label="Mikopo Wazi"
                value={stats.outstanding_loans.toString()}
                color="red"
              />
              <StatCard
                icon={TrendingUp}
                label="Hazina"
                value={`TZS ${stats.treasury_balance.toLocaleString()}`}
                color="purple"
              />
              {stats.pending_contributions > 0 && (
                <StatCard
                  icon={Clock}
                  label="Michango Inayosubiri"
                  value={stats.pending_contributions.toString()}
                  color="amber"
                />
              )}
              {stats.pending_loans > 0 && (
                <StatCard
                  icon={Clock}
                  label="Mikopo Inayosubiri"
                  value={stats.pending_loans.toString()}
                  color="amber"
                />
              )}
            </div>
          ) : (
            <p className="text-muted-foreground text-center py-4">Imeshindikana kupata takwimu</p>
          )}
        </div>
      </div>
    </AppShell>
  );
}

function StatCard({ icon: Icon, label, value, color }: { icon: any; label: string; value: string; color: string }) {
  const colorClasses: Record<string, string> = {
    blue: "bg-blue-100 text-blue-700",
    green: "bg-green-100 text-green-700",
    red: "bg-red-100 text-red-700",
    purple: "bg-purple-100 text-purple-700",
    amber: "bg-amber-100 text-amber-700",
  };

  return (
    <div className="flex items-center gap-3 rounded-lg border p-4">
      <div className={`grid h-10 w-10 place-items-center rounded-lg ${colorClasses[color]}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="text-sm text-muted-foreground">{label}</p>
        <p className="font-display text-xl font-bold">{value}</p>
      </div>
    </div>
  );
}
