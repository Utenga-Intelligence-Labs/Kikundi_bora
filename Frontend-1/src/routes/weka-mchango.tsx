import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Upload, MessageSquare, Loader2 } from "lucide-react";

export const Route = createFileRoute("/weka-mchango")({
  beforeLoad: () => {
    requireAuth();
  },
  component: WekaMchangoPage,
});

function WekaMchangoPage() {
  const { user } = useAuth();
  const qc = useQueryClient();

  const [formData, setFormData] = useState({
    contribution_type: "AKIBA",
    period_label: new Date().toISOString().slice(0, 7), // Current month YYYY-MM
    amount: "",
    proof_image_url: "",
    proof_message: "",
  });

  const submitMutation = useMutation({
    mutationFn: async (data: typeof formData) => {
      const token = localStorage.getItem("auth_token");
      const res = await fetch("/api/v1/michango", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.message || "Imeshindikana kuwasilisha");
      }
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      alert("Mchango umewasilishwa!");
      setFormData({
        contribution_type: "AKIBA",
        period_label: new Date().toISOString().slice(0, 7),
        amount: "",
        proof_image_url: "",
        proof_message: "",
      });
    },
    onError: (err: Error) => {
      alert(err.message);
    },
  });

  if (!user) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.amount || parseFloat(formData.amount) <= 0) {
      alert("Kiasi kinahitajika");
      return;
    }
    if (!formData.proof_image_url && !formData.proof_message) {
      alert("Lazima uweke picha ya uthibitisho au ujumbe wa muamala");
      return;
    }
    submitMutation.mutate(formData);
  };

  return (
    <AppShell title="Weka Mchango" subtitle="Wasilisha mchango wako kwa uthibitisho">
      <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
        <div className="card-surface p-6">
          <h3 className="font-display text-lg font-semibold mb-4">Aina ya Mchango</h3>
          <div className="space-y-3">
            <label className="flex items-start gap-3 rounded-lg border p-4 cursor-pointer transition-colors hover:bg-muted/50">
              <input
                type="radio"
                name="contribution_type"
                value="AKIBA"
                checked={formData.contribution_type === "AKIBA"}
                onChange={(e) => setFormData({ ...formData, contribution_type: e.target.value })}
                className="mt-0.5"
              />
              <div>
                <p className="font-medium">Akiba</p>
                <p className="text-sm text-muted-foreground">Mchango wa akiba ya kikundi (unathibitishwa na Hazina)</p>
              </div>
            </label>
            <label className="flex items-start gap-3 rounded-lg border p-4 cursor-pointer transition-colors hover:bg-muted/50">
              <input
                type="radio"
                name="contribution_type"
                value="MFUKO_WA_KIJAMII"
                checked={formData.contribution_type === "MFUKO_WA_KIJAMII"}
                onChange={(e) => setFormData({ ...formData, contribution_type: e.target.value })}
                className="mt-0.5"
              />
              <div>
                <p className="font-medium">Mfuko wa Kijamii</p>
                <p className="text-sm text-muted-foreground">Mchango wa mfuko wa kijamii (unathibitishwa na Mwenyekiti)</p>
              </div>
            </label>
          </div>
        </div>

        <div className="card-surface p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Kipindi (Mwezi)</label>
            <input
              type="month"
              value={formData.period_label}
              onChange={(e) => setFormData({ ...formData, period_label: e.target.value })}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Kiasi (TZS)</label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={formData.amount}
              onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
              placeholder="0.00"
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">
              Uthibitisho (Lazima uweke moja)
            </label>
            <div className="space-y-3">
              <div className="flex items-start gap-2">
                <Upload className="h-5 w-5 text-muted-foreground mt-2" />
                <div className="flex-1">
                  <label className="text-xs text-muted-foreground">URL ya Picha ya Uthibitisho</label>
                  <input
                    type="url"
                    value={formData.proof_image_url}
                    onChange={(e) => setFormData({ ...formData, proof_image_url: e.target.value })}
                    placeholder="https://example.com/receipt.jpg"
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
              <div className="flex items-start gap-2">
                <MessageSquare className="h-5 w-5 text-muted-foreground mt-2" />
                <div className="flex-1">
                  <label className="text-xs text-muted-foreground">Ujumbe wa Muamala</label>
                  <textarea
                    value={formData.proof_message}
                    onChange={(e) => setFormData({ ...formData, proof_message: e.target.value })}
                    placeholder="Namba ya muamala au maelezo..."
                    rows={3}
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <button
          type="submit"
          disabled={submitMutation.isPending}
          className="w-full inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-3 font-semibold text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
          {submitMutation.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Inatuma...
            </>
          ) : (
            <>
              <Plus className="h-4 w-4" />
              Wasilisha Mchango
            </>
          )}
        </button>
      </form>
    </AppShell>
  );
}
