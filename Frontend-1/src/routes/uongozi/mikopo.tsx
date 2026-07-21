import { createFileRoute, Link } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { ShieldCheck, Check, X, Clock, Banknote } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

export const Route = createFileRoute("/uongozi/mikopo")({
  beforeLoad: () => {
    requireAuth();
  },
  component: MikopoPage,
});

interface Loan {
  id: string;
  member_id: string;
  amount: number;
  approved_amount?: number;
  balance_remaining?: number;
  purpose?: string;
  due_date: string;
  status: string;
  rejection_reason?: string;
  applied_at: string;
  member?: {
    id: string;
    member_no: string;
    full_name: string;
    phone: string;
  };
}

function MikopoPage() {
  const { user, isLeadership } = useAuth();
  const qc = useQueryClient();
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  if (!user || !isLeadership) {
    return (
      <AppShell title="Idhinisha Mikopo">
        <div className="card-surface p-12 text-center">
          <p className="text-muted-foreground">Huna ruhusa ya kufikia ukurasa huu</p>
        </div>
      </AppShell>
    );
  }

  const { data: loans, isLoading } = useQuery({
    queryKey: ["uongozi", "mikopo", "pending"],
    queryFn: async () => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/uongozi/mikopo/pending", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error("Imeshindikana kupata mikopo");
      const data = await res.json();
      return data.data as Loan[];
    },
  });

  const approveMutation = useMutation({
    mutationFn: async ({ loanId, amount }: { loanId: string; amount: number }) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch(`/api/v1/uongozi/mikopo/${loanId}/approve`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ approved_amount: amount }),
      });
      if (!res.ok) throw new Error("Imeshindikana kuidhinisha");
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["uongozi", "mikopo"] });
    },
  });

  const handleApprove = async (loan: Loan) => {
    const confirmed = window.confirm(
      `Idhinisha mkopo wa TZS ${loan.amount.toLocaleString()} kwa ${loan.member?.full_name}?`
    );
    if (!confirmed) return;

    setActionLoading(loan.id);
    try {
      await approveMutation.mutateAsync({ loanId: loan.id, amount: loan.amount });
      alert("Mkopo umeidhinishwa!");
    } catch (err) {
      alert("Imeshindikana kuidhinisha mkopo");
    } finally {
      setActionLoading(null);
    }
  };

  const pendingLoans = loans ?? [];

  return (
    <AppShell title="Idhinisha Mikopo" subtitle="Orodha ya mikopo inayosubiri idhini">
      <div className="space-y-4">
        {isLoading ? (
          <div className="card-surface animate-pulse p-6">
            <div className="h-4 w-1/3 rounded bg-muted" />
          </div>
        ) : pendingLoans.length === 0 ? (
          <div className="card-surface p-12 text-center">
            <ShieldCheck className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <p className="mt-4 text-muted-foreground">Hakuna mikopo inayosubiri idhini</p>
          </div>
        ) : (
          pendingLoans.map((loan) => (
            <div key={loan.id} className="card-surface p-5">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="chip bg-amber-100 text-amber-700 text-[10px] font-semibold px-2 py-0.5 rounded">
                      {loan.member?.member_no}
                    </span>
                    <span className="chip bg-blue-100 text-blue-700 text-[10px] font-semibold px-2 py-0.5 rounded">
                      <Clock className="inline h-3 w-3 mr-0.5" />
                      Inasubiri
                    </span>
                  </div>
                  <h3 className="font-display text-lg font-semibold">{loan.member?.full_name}</h3>
                  <p className="text-sm text-muted-foreground">{loan.member?.phone}</p>
                  <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <p className="text-muted-foreground">Kiasi</p>
                      <p className="font-semibold">TZS {loan.amount.toLocaleString()}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Tarehe ya Kulipa</p>
                      <p className="font-semibold">{new Date(loan.due_date).toLocaleDateString("sw-TZ")}</p>
                    </div>
                    {loan.purpose && (
                      <div className="col-span-2">
                        <p className="text-muted-foreground">Sababu</p>
                        <p className="font-semibold">{loan.purpose}</p>
                      </div>
                    )}
                    <div className="col-span-2">
                      <p className="text-muted-foreground">Tarehe ya Kuomba</p>
                      <p className="font-semibold">{new Date(loan.applied_at).toLocaleDateString("sw-TZ")}</p>
                    </div>
                  </div>
                </div>
                <div className="flex flex-col gap-2">
                  <button
                    onClick={() => handleApprove(loan)}
                    disabled={actionLoading === loan.id}
                    className="inline-flex items-center gap-1.5 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-green-700 disabled:opacity-50"
                  >
                    <Check className="h-4 w-4" />
                    {actionLoading === loan.id ? "..." : "Idhinisha"}
                  </button>
                  <Link
                    to={`/ukaguzi-mkopo/${loan.id}`}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground/80 hover:bg-muted"
                  >
                    Angalia
                  </Link>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </AppShell>
  );
}
