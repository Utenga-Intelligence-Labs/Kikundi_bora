import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { AppShell } from "@/components/AppShell";
import { Field } from "@/components/Field";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth, requireRole, hasRole } from "@/lib/role-guards";
import {
  useCollectionQueue,
  useCollectFine,
  obligationKeys,
} from "@/hooks/use-obligations";
import { groupsApi } from "@/api/groups";
import { contributionsApi } from "@/api/contributions";
import { tzs } from "@/lib/format";
import { HandCoins, Loader2, Check, Banknote } from "lucide-react";
import { useAppModal } from "@/components/AppModal";

export const Route = createFileRoute("/ukusanyaji")({
  head: () => ({
    meta: [
      { title: "Ukusanyaji — Money Seeking" },
      { name: "description", content: "Foleni ya ukusanyaji: malimbikizo na faini zinazosubiri (mweka hazina)." },
    ],
  }),
  beforeLoad: () => {
    requireAuth();
    requireRole("treasurer");
  },
  component: UkusanyajiPage,
});

function UkusanyajiPage() {
  const { user } = useAuth();
  const { showModal } = useAppModal();
  const qc = useQueryClient();
  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const groupId = gs?.data.id ?? null;
  const { data, isLoading } = useCollectionQueue(groupId);
  const collect = useCollectFine();
  const [payFor, setPayFor] = useState<string | null>(null);

  if (!hasRole(user, "treasurer", "admin")) {
    return (
      <AppShell title="Ukusanyaji" subtitle="Huna ruhusa">
        <p className="text-sm text-muted-foreground">Ukurasa huu ni kwa Mweka Hazina tu.</p>
      </AppShell>
    );
  }

  const onCollect = (fineId: string, label: string) => {
    showModal({
      title: "Mark as Collected?",
      message: `Thibitisha umepokea ${label}.`,
      variant: "warning",
      primaryLabel: "Nimepokea",
      onPrimary: () =>
        collect.mutate(fineId, {
          onSuccess: (res: any) => {
            showModal({ title: "Imefanikiwa", message: res.message, variant: "success", primaryLabel: "Sawa" });
            qc.invalidateQueries({ queryKey: ["obligations"] });
          },
          onError: (e: Error) =>
            showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
        }),
    });
  };

  return (
    <AppShell title="Ukusanyaji" subtitle="Malimbikizo na faini — kubwa kwanza">
      {isLoading ? (
        <div className="mt-8 flex justify-center"><Loader2 className="h-6 w-6 animate-spin text-primary" /></div>
      ) : (data?.data.length ?? 0) === 0 ? (
        <p className="text-sm text-muted-foreground">Hakuna madeni — kila mwanachama yuko sawa.</p>
      ) : (
        <div className="max-w-3xl space-y-3">
          {(data?.data ?? []).map((entry) => (
            <section key={entry.member.member_id} className="card-surface overflow-hidden" data-testid={`queue-${entry.member.member_no}`}>
              <header className="flex items-center justify-between border-b border-border px-4 py-3">
                <div>
                  <h3 className="font-display text-sm font-semibold">{entry.member.full_name}</h3>
                  <p className="text-xs text-muted-foreground">{entry.member.member_no}</p>
                </div>
                <span className="text-sm font-bold">{tzs(Number(entry.member.grand_total_owed))}</span>
              </header>
              <div>
                {entry.arrears.map((a) => (
                  <div key={a.cycle_label} className="flex items-center justify-between border-t border-border px-4 py-2.5 first:border-t-0">
                    <span className="text-sm">Malimbikizo {a.cycle_label} <span className="text-xs text-muted-foreground">({new Date(a.due_date).toLocaleDateString()})</span></span>
                    <span className="text-sm font-semibold">{tzs(Number(a.owed))}</span>
                  </div>
                ))}
                {entry.fines.map((f) => (
                  <div key={f.id} className="flex items-center justify-between gap-2 border-t border-border px-4 py-2.5">
                    <span className="text-sm">Faini: {f.offence_name} <span className="text-xs text-muted-foreground">({new Date(f.occurrence_date).toLocaleDateString()})</span></span>
                    <span className="flex items-center gap-2">
                      <span className="text-sm font-semibold">{tzs(Number(f.amount))}</span>
                      <button
                        onClick={() => onCollect(f.id, `${f.offence_name} — ${tzs(Number(f.amount))}`)}
                        disabled={collect.isPending}
                        aria-label={`Mark as Collected ${f.offence_name}`}
                        className="inline-flex items-center gap-1 rounded-lg bg-primary px-2.5 py-1.5 text-xs font-semibold text-primary-foreground disabled:opacity-50"
                      >
                        <Check className="h-3.5 w-3.5" /> Pokea
                      </button>
                    </span>
                  </div>
                ))}
                {payFor === entry.member.member_id ? (
                  <RecordPaymentForm
                    memberId={entry.member.member_id}
                    memberName={entry.member.full_name}
                    suggested={entry.member.grand_total_owed}
                    onDone={() => {
                      setPayFor(null);
                      qc.invalidateQueries({ queryKey: obligationKeys.queue(groupId ?? "") });
                      qc.invalidateQueries({ queryKey: ["obligations"] });
                    }}
                    onCancel={() => setPayFor(null)}
                  />
                ) : (
                  <div className="border-t border-border px-4 py-2.5">
                    <button
                      onClick={() => setPayFor(entry.member.member_id)}
                      className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-semibold hover:bg-muted"
                    >
                      <Banknote className="h-3.5 w-3.5" /> Rekodi Malipo
                    </button>
                  </div>
                )}
              </div>
            </section>
          ))}
        </div>
      )}
    </AppShell>
  );
}

function RecordPaymentForm({ memberId, memberName, suggested, onDone, onCancel }: {
  memberId: string;
  memberName: string;
  suggested: string;
  onDone: () => void;
  onCancel: () => void;
}) {
  const { showModal } = useAppModal();
  const [amount, setAmount] = useState(suggested);
  const [paidAt, setPaidAt] = useState(new Date().toISOString().slice(0, 10));
  const mutation = useMutation({
    mutationFn: () =>
      contributionsApi.create({
        member_id: memberId,
        amount: parseFloat(amount),
        month: new Date().toISOString().slice(0, 7),
        paid_at: paidAt,
        payment_method: "CASH",
      }),
    onSuccess: (res: any) => {
      const lines = (res.receipt?.lines ?? res.data?.lines ?? [])
        .map((l: any) => `${l.label}: ${tzs(Number(l.amount))}`)
        .join("\n");
      showModal({
        title: "Malipo yamegawanywa",
        message: `Malipo ya ${memberName} yamegawanywa:\n${lines}`,
        variant: "success",
        primaryLabel: "Sawa",
      });
      onDone();
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  return (
    <div className="space-y-2 border-t border-border bg-muted/20 px-4 py-3">
      <p className="text-xs font-semibold">Rekodi malipo — {memberName} (deni lote: {tzs(Number(suggested))})</p>
      <div className="grid grid-cols-2 gap-2">
        <Field label="Kiasi (TZS)" type="number" value={amount} onChange={setAmount} placeholder="0" />
        <Field label="Tarehe" type="date" value={paidAt} onChange={setPaidAt} />
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending || !(parseFloat(amount) > 0)}
          className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground disabled:opacity-50"
        >
          {mutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <HandCoins className="h-3.5 w-3.5" />}
          Gawa &amp; Hifadhi
        </button>
        <button onClick={onCancel} className="rounded-lg border border-border px-3 py-2 text-xs">Ghairi</button>
      </div>
      <p className="text-[11px] text-muted-foreground">Kiasi kitagawanywa: malimbikizo ya zamani → mchango wa sasa → faini.</p>
    </div>
  );
}
