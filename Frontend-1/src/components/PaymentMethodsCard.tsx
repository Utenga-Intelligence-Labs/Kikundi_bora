import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Smartphone, Landmark, Copy, Check, Plus, Pencil, Power, Trash2, Loader2, X,
} from "lucide-react";
import { useAuth } from "@/lib/auth-provider";
import { groupsApi } from "@/api/groups";
import {
  paymentMethodsApi,
  type PaymentMethod,
  type PaymentMethodType,
} from "@/api/payment-methods";
import { useAppModal } from "@/components/AppModal";

/**
 * Payment info card for Weka Mchango: shows the group's LipaNamba numbers
 * and bank accounts (read-only for members, with copy buttons). Mwenyekiti /
 * Mweka Hazina additionally get the add/edit/deactivate management form.
 */
export function PaymentMethodsCard() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const canManage = user?.role === "chair" || user?.role === "treasurer";
  const canApprove = user?.role === "chair" || user?.role === "admin";
  const isPending = (pm: PaymentMethod) => pm.status === "pending";

  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const groupId = gs?.data.id;

  const { data, isLoading } = useQuery({
    queryKey: ["payment-methods", groupId],
    queryFn: () => paymentMethodsApi.list(groupId!),
    enabled: !!groupId,
  });

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<PaymentMethod | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [form, setForm] = useState({
    type: "lipa_namba" as PaymentMethodType,
    provider_name: "",
    account_number: "",
    account_name: "",
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["payment-methods"] });

  const saveMutation = useMutation({
    mutationFn: () => {
      const payload = {
        type: form.type,
        provider_name: form.provider_name,
        account_number: form.account_number,
        account_name: form.account_name,
      };
      return editing
        ? paymentMethodsApi.update(groupId!, editing.id, payload)
        : paymentMethodsApi.create(groupId!, payload);
    },
    onSuccess: (res) => {
      showModal({
        title: "Imefanikiwa",
        message: res.message || (editing ? "Mabadiliko yamehifadhiwa." : "Njia ya malipo imeongezwa."),
        variant: "success",
        primaryLabel: "Sawa",
      });
      setFormOpen(false);
      setEditing(null);
      invalidate();
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const toggleMutation = useMutation({
    mutationFn: (pm: PaymentMethod) =>
      paymentMethodsApi.update(groupId!, pm.id, { is_active: !pm.is_active }),
    onSuccess: invalidate,
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const deleteMutation = useMutation({
    mutationFn: (pm: PaymentMethod) => paymentMethodsApi.remove(groupId!, pm.id),
    onSuccess: () => {
      showModal({ title: "Imefanikiwa", message: "Njia ya malipo imefutwa.", variant: "success", primaryLabel: "Sawa" });
      invalidate();
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const approveMutation = useMutation({
    mutationFn: (pm: PaymentMethod) => paymentMethodsApi.approve(groupId!, pm.id),
    onSuccess: (res) => {
      showModal({ title: "Imefanikiwa", message: res.message || "Njia ya malipo imeidhinishwa.", variant: "success", primaryLabel: "Sawa" });
      invalidate();
    },
    onError: (e: Error) =>
      showModal({ title: "Hitilafu", message: e.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const copyNumber = async (pm: PaymentMethod) => {
    try {
      await navigator.clipboard.writeText(pm.account_number);
    } catch {
      // clipboard API unavailable (e.g. insecure context) — fallback
      const ta = document.createElement("textarea");
      ta.value = pm.account_number;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      ta.remove();
    }
    setCopiedId(pm.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const startEdit = (pm: PaymentMethod) => {
    setEditing(pm);
    setForm({
      type: pm.type,
      provider_name: pm.provider_name,
      account_number: pm.account_number,
      account_name: pm.account_name,
    });
    setFormOpen(true);
  };

  const startAdd = () => {
    setEditing(null);
    setForm({ type: "lipa_namba", provider_name: "", account_number: "", account_name: "" });
    setFormOpen(true);
  };

  const methods = data?.data ?? [];
  const lipa = methods.filter((m) => m.type === "lipa_namba");
  const bank = methods.filter((m) => m.type === "bank");

  const renderRow = (pm: PaymentMethod) => (
    <div
      key={pm.id}
      className={`flex items-center justify-between gap-3 px-4 py-3 ${!pm.is_active ? "opacity-50" : ""}`}
    >
      <div className="min-w-0">
        <p className="text-sm font-semibold">
          {pm.provider_name} · <span className="font-mono">{pm.account_number}</span>
        </p>
        <p className="text-xs text-muted-foreground truncate">{pm.account_name}</p>
        {isPending(pm) && (
          <span className="chip bg-amber-100 text-amber-800 text-[10px] mt-1">Inasubiri kuidhinishwa</span>
        )}
        {!pm.is_active && (
          <span className="chip bg-muted text-muted-foreground text-[10px] mt-1">Imezimwa</span>
        )}
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        {pm.is_active && (
          <button
            data-testid={`copy-${pm.id}`}
            onClick={() => copyNumber(pm)}
            className="inline-flex items-center gap-1 rounded-lg border border-border px-2 py-1 text-xs font-medium hover:bg-muted"
          >
            {copiedId === pm.id ? (
              <>
                <Check className="h-3 w-3 text-success" /> Imenakiliwa
              </>
            ) : (
              <>
                <Copy className="h-3 w-3" /> Nakili
              </>
            )}
          </button>
        )}
        {canManage && (
          <>
            {canApprove && isPending(pm) && (
              <button
                onClick={() => approveMutation.mutate(pm)}
                disabled={approveMutation.isPending}
                aria-label="Idhinisha"
                title="Idhinisha — ionekane kwa wanachama"
                className="grid h-7 w-7 place-items-center rounded-lg border border-emerald-500/50 text-emerald-600 hover:bg-emerald-500/10 disabled:opacity-50"
              >
                <Check className="h-3.5 w-3.5" />
              </button>
            )}
            <button
              onClick={() => startEdit(pm)}
              aria-label="Badilisha"
              className="grid h-7 w-7 place-items-center rounded-lg border border-border hover:bg-muted"
            >
              <Pencil className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => toggleMutation.mutate(pm)}
              disabled={toggleMutation.isPending}
              aria-label={pm.is_active ? "Zima" : "Washa"}
              title={pm.is_active ? "Zima" : "Washa"}
              className="grid h-7 w-7 place-items-center rounded-lg border border-border hover:bg-muted disabled:opacity-50"
            >
              <Power className={`h-3.5 w-3.5 ${pm.is_active ? "text-success" : "text-muted-foreground"}`} />
            </button>
            <button
              onClick={() =>
                showModal({
                  title: "Futa njia ya malipo?",
                  message: `${pm.provider_name} · ${pm.account_number} itafutwa kabisa.`,
                  variant: "warning",
                  primaryLabel: "Futa",
                  onPrimary: () => deleteMutation.mutate(pm),
                })
              }
              disabled={deleteMutation.isPending}
              aria-label="Futa"
              className="grid h-7 w-7 place-items-center rounded-lg border border-destructive/40 text-destructive hover:bg-destructive/10 disabled:opacity-50"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </>
        )}
      </div>
    </div>
  );

  const renderSection = (label: string, icon: React.ElementType, items: PaymentMethod[]) => {
    if (items.length === 0) return null;
    const SectionIcon = icon;
    return (
      <div>
        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-1 flex items-center gap-1.5">
          <SectionIcon className="h-3.5 w-3.5" /> {label}
        </p>
        <div className="rounded-lg border divide-y divide-border">{items.map(renderRow)}</div>
      </div>
    );
  };

  return (
    <div className="card-surface p-6 space-y-4" data-testid="payment-methods-card">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h3 className="font-display text-lg font-semibold">Malipo ya Mchango</h3>
          <p className="text-xs text-muted-foreground">
            Tuma mchango kwa LipaNamba ama benki hapa chini, kisha wasilisha uthibitisho.
          </p>
        </div>
        {canManage && !formOpen && (
          <button
            onClick={startAdd}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-3.5 w-3.5" /> Ongeza
          </button>
        )}
      </div>

      {isLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
          <Loader2 className="h-4 w-4 animate-spin" /> Inapakia njia za malipo...
        </div>
      ) : methods.length === 0 && !formOpen ? (
        <p className="text-sm text-muted-foreground rounded-lg border border-dashed p-4 text-center">
          Hakuna njia za malipo zilizowekwa bado.
          {canManage ? " Bofya \"Ongeza\" kuweka LipaNamba ama akaunti ya benki." : " Wasiliana na Mwenyekiti au Mweka Hazina."}
        </p>
      ) : (
        <div className="space-y-4">
          {renderSection("LipaNamba", Smartphone, lipa)}
          {renderSection("Benki", Landmark, bank)}
        </div>
      )}

      {/* Management form — mwenyekiti / mweka hazina only */}
      {canManage && formOpen && (
        <div className="rounded-lg border p-4 space-y-3 bg-muted/20">
          <div className="flex items-center justify-between">
            <p className="text-sm font-semibold">
              {editing ? "Badilisha njia ya malipo" : "Ongeza njia ya malipo"}
            </p>
            <button onClick={() => { setFormOpen(false); setEditing(null); }} className="rounded-lg p-1 hover:bg-muted" aria-label="Funga">
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="grid sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Aina</label>
              <select
                value={form.type}
                onChange={(e) => setForm({ ...form, type: e.target.value as PaymentMethodType })}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              >
                <option value="lipa_namba">LipaNamba (Simu)</option>
                <option value="bank">Benki</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1">
                {form.type === "lipa_namba" ? "Mtandao (M-Pesa, Tigo Pesa...)" : "Benki (CRDB, NMB...)"}
              </label>
              <input
                value={form.provider_name}
                onChange={(e) => setForm({ ...form, provider_name: e.target.value })}
                placeholder={form.type === "lipa_namba" ? "M-Pesa" : "CRDB"}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1">
                {form.type === "lipa_namba" ? "Namba ya LipaNamba" : "Namba ya akaunti"}
              </label>
              <input
                value={form.account_number}
                onChange={(e) => setForm({ ...form, account_number: e.target.value })}
                placeholder={form.type === "lipa_namba" ? "255700000000" : "0150000000000"}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm font-mono"
              />
            </div>
            <div>
              <label className="block text-xs text-muted-foreground mb-1">Jina lililosajiliwa</label>
              <input
                value={form.account_name}
                onChange={(e) => setForm({ ...form, account_name: e.target.value })}
                placeholder="Kikundi cha Money Seeking"
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              />
            </div>
          </div>
          <button
            onClick={() => saveMutation.mutate()}
            disabled={
              saveMutation.isPending ||
              !form.provider_name.trim() ||
              !form.account_number.trim() ||
              !form.account_name.trim()
            }
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {saveMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
            Hifadhi
          </button>
        </div>
      )}
    </div>
  );
}
