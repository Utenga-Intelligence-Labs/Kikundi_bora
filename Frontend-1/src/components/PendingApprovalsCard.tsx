import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { hasRole } from "@/lib/role-guards";
import {
  useOffenceTypes,
  useDecideOffenceType,
  useFines,
  useDecideWaiver,
} from "@/hooks/use-obligations";
import { groupsApi } from "@/api/groups";
import { Stamp } from "lucide-react";

/**
 * PendingApprovalsCard — extends the leadership settings page with the
 * katibu approvals queue for offence-type changes and fine waivers.
 * Only katibu (and admin) see the decide buttons; others see nothing.
 */
export function PendingApprovalsCard() {
  const { user } = useAuth();
  const isSecretary = hasRole(user, "secretary", "admin");
  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const groupId = (gs?.data.id ?? null) as string | null;
  const { data: offences } = useOffenceTypes(groupId);
  const { data: fines } = useFines({});
  const decideOffence = useDecideOffenceType();
  const decideWaiver = useDecideWaiver();

  const pendingOffences = (offences?.data ?? []).filter((o) => o.status === "pending");
  const pendingWaivers = (fines?.data ?? []).filter((f) => f.waiver_status === "pending");
  if (!isSecretary || (pendingOffences.length === 0 && pendingWaivers.length === 0)) {
    return null;
  }

  return (
    <section className="card-surface overflow-hidden" data-testid="pending-approvals">
      <header className="flex items-center gap-2.5 border-b border-border px-4 py-3">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
          <Stamp className="h-4 w-4" />
        </span>
        <div>
          <h3 className="font-display text-sm font-semibold">Vibali Vinasubiri (Katibu)</h3>
          <p className="text-[11px] text-muted-foreground">Aina za makosa na misamaha ya faini</p>
        </div>
      </header>
      <div className="space-y-2 p-4">
        {pendingOffences.map((o) => (
          <div key={o.id} className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2 text-sm">
            <span>Aina ya kosa: <strong>{o.name}</strong></span>
            <span className="flex gap-1.5">
              <button
                onClick={() => groupId && decideOffence.mutate({ groupId, id: o.id, approve: true })}
                className="rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground"
              >Idhinisha</button>
              <button
                onClick={() => groupId && decideOffence.mutate({ groupId, id: o.id, approve: false })}
                className="rounded-lg border border-destructive/40 px-2 py-1 text-xs text-destructive"
              >Zima</button>
            </span>
          </div>
        ))}
        {pendingWaivers.map((f) => (
          <div key={f.id} className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2 text-sm">
            <span>Msamaha: <strong>{f.offence_type?.name ?? f.reason}</strong> · {f.member?.full_name}</span>
            <span className="flex gap-1.5">
              <button
                onClick={() => decideWaiver.mutate({ id: f.id, approve: true })}
                className="rounded-lg bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground"
              >Kubali</button>
              <button
                onClick={() => decideWaiver.mutate({ id: f.id, approve: false })}
                className="rounded-lg border border-destructive/40 px-2 py-1 text-xs text-destructive"
              >Kataa</button>
            </span>
          </div>
        ))}
        <Link to="/mikutano" className="text-xs font-medium text-primary">Maelezo zaidi →</Link>
      </div>
    </section>
  );
}
