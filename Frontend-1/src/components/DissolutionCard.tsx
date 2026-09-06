import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { dissolutionApi } from "@/api/dissolution";
import { groupsApi } from "@/api/groups";
import { useAuth } from "@/lib/auth-provider";
import { Loader2 } from "lucide-react";

export function DissolutionProposeCard() {
  const { user } = useAuth();
  const canPropose = user?.role === "chair" || user?.role === "secretary";
  const { data: grp } = useQuery({ queryKey: ["groups","current"], queryFn: groupsApi.current });
  const groupId = grp?.data.id;
  const qc = useQueryClient();
  const [span, setSpan] = useState(1);
  const [deadline, setDeadline] = useState("");
  const [msg, setMsg] = useState<string|null>(null);

  const propose = useMutation({
    mutationFn: () => dissolutionApi.propose(groupId!, { cycle_span_years: span, voting_deadline: deadline }),
    onSuccess: () => { setMsg("Pendekezo limetumwa"); qc.invalidateQueries({queryKey:["dissolution"]}); },
    onError: (e:any) => setMsg(e.message),
  });

  if (!canPropose) return null;
  if (grp?.data && (grp.data as any).status === "dissolved") return <div className="card-surface p-4 text-sm text-muted-foreground">Kikundi kimevunjwa — hakuna pendekezo jipya.</div>;

  return (
    <section className="card-surface overflow-hidden">
      <header className="px-4 py-3 border-b border-border"><h3 className="font-semibold text-sm">Pendekeza Uvunjaji wa Kikundi</h3><p className="text-xs text-muted-foreground">Baada ya miaka 1 au 2, kura ya wanachama</p></header>
      <div className="p-4 space-y-3">
        <label className="block text-xs">Muda (miaka) <select value={span} onChange={e=>setSpan(Number(e.target.value))} className="ml-2 border rounded px-2 py-1"><option value={1}>1 mwaka</option><option value={2}>2 miaka</option></select></label>
        <label className="block text-xs">Mwisho wa kura <input type="date" value={deadline} onChange={e=>setDeadline(e.target.value)} className="ml-2 border rounded px-2 py-1" /></label>
        <button onClick={()=>propose.mutate()} disabled={!deadline || propose.isPending || !groupId} className="rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground disabled:opacity-50">{propose.isPending?<Loader2 className="h-4 w-4 animate-spin"/>:"Pendekeza"}</button>
        {msg && <p className="text-xs text-muted-foreground">{msg}</p>}
        <p className="text-[11px] text-muted-foreground">Threshold: simple majority ya kura zilizopigwa (&gt;50% yes). Principal-only share-out; interest-sharing ni follow-up.</p>
      </div>
    </section>
  );
}

export function DissolutionVotingCard({ proposalId }: { proposalId: string }) {
  const { data, refetch } = useQuery({ queryKey: ["dissolution", proposalId], queryFn: ()=>dissolutionApi.get(proposalId), refetchInterval: 10000 });
  const qc = useQueryClient();
  const vote = useMutation({
    mutationFn: (v:string)=>dissolutionApi.vote(proposalId, v),
    onSuccess: ()=>{ qc.invalidateQueries({queryKey:["dissolution",proposalId]}); refetch(); },
  });
  if (!data) return <Loader2 className="h-5 w-5 animate-spin"/>;
  const tally = data.tally;
  const status = data.data.status;
  return (
    <section className="card-surface p-4">
      <h3 className="font-semibold text-sm mb-2">Kura ya Uvunjaji — {status}</h3>
      <p className="text-xs text-muted-foreground mb-2">Mwisho: {new Date(data.data.voting_deadline).toLocaleString("sw-TZ")}</p>
      <div className="flex gap-4 text-sm mb-3"><span className="text-success">Ndio: {tally.yes}</span><span className="text-destructive">Hapana: {tally.no}</span><span>Jumla: {tally.total}</span></div>
      {data.my_vote && <p className="text-xs mb-2">Kura yako: <b>{data.my_vote}</b> (kutuma tena kunasasisha)</p>}
      {status==="voting_open" && <div className="flex gap-2"><button onClick={()=>vote.mutate("yes")} className="flex-1 rounded-xl bg-success px-3 py-2 text-sm font-semibold text-white">Ndio</button><button onClick={()=>vote.mutate("no")} className="flex-1 rounded-xl bg-destructive px-3 py-2 text-sm font-semibold text-white">Hapana</button></div>}
      {tally.approved && <p className="text-xs text-success mt-2">Imepitishwa — simple majority</p>}
    </section>
  );
}

export function DissolutionPayoutTable({ proposalId }: { proposalId: string }) {
  const { data } = useQuery({ queryKey: ["dissolution-payouts", proposalId], queryFn: ()=>dissolutionApi.payouts(proposalId) });
  const qc = useQueryClient();
  const mark = useMutation({ mutationFn: (id:string)=>dissolutionApi.markPaid(id), onSuccess:()=>qc.invalidateQueries({queryKey:["dissolution-payouts",proposalId]}) });
  const payouts = data?.data ?? [];
  if (!payouts.length) return <p className="text-sm text-muted-foreground p-4">Hakuna payouts bado — tekeleza pendekezo baada ya kura.</p>;
  return (
    <div className="card-surface overflow-hidden">
      <div className="overflow-auto">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-xs"><tr><th className="px-3 py-2 text-left">Mwanachama</th><th className="px-3 py-2 text-right">Michango</th><th className="px-3 py-2 text-right">Deni lililokatwa</th><th className="px-3 py-2 text-right">Anachopata</th><th className="px-3 py-2">Hali</th><th className="px-3 py-2"></th></tr></thead>
          <tbody>{payouts.map(p=>(
            <tr key={p.id} className="border-t border-border"><td className="px-3 py-2">{p.member?.full_name ?? p.member_id} <span className="text-xs text-muted-foreground">{p.member?.member_no}</span></td><td className="px-3 py-2 text-right">{p.total_contributed}</td><td className="px-3 py-2 text-right text-destructive">{p.total_owed}</td><td className="px-3 py-2 text-right font-semibold">{p.amount_owed}</td><td className="px-3 py-2">{p.status}</td><td className="px-3 py-2">{p.status==="pending" && <button onClick={()=>mark.mutate(p.id)} className="text-xs text-primary underline">Thibitisha kulipwa</button>}</td></tr>
          ))}</tbody>
        </table>
      </div>
      <p className="text-[11px] text-muted-foreground p-3">Netting inayoonekana — michango jumla minus madeni (mikopo+faini) kutoka obligations, floor 0.</p>
    </div>
  );
}

export function DissolvedBanner() {
  const { data } = useQuery({ queryKey: ["groups","current"], queryFn: groupsApi.current });
  if ((data?.data as any)?.status !== "dissolved") return null;
  return <div className="rounded-xl bg-amber-100 border border-amber-300 px-4 py-3 text-sm font-semibold text-amber-900">Kikundi kimevunjwa — hakuna michango/mikopo/wanachama wapya. Tazama historia na malipo tu.</div>;
}
