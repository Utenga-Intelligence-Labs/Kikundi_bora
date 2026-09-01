import { createFileRoute } from "@tanstack/react-router";
import { useState, useRef, useEffect } from "react";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { api } from "@/api/client";
import { uploadApi } from "@/api/upload";
import { AppShell } from "@/components/AppShell";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Upload, MessageSquare, Loader2, X, ImageIcon, CalendarDays } from "lucide-react";
import { MfukoContributionForm } from "@/components/MfukoContributionForm";
import { PaymentMethodsCard } from "@/components/PaymentMethodsCard";
import { groupsApi, INTERVAL_LABELS } from "@/api/groups";
import { useAppModal } from "@/components/AppModal";

export const Route = createFileRoute("/weka-mchango")({
  beforeLoad: () => {
    requireAuth();
  },
  component: WekaMchangoPage,
});

function WekaMchangoPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Group contribution settings (approved) — banner + fixed amount enforcement
  const { data: settingsData } = useQuery({
    queryKey: ["groups", "settings"],
    queryFn: () => groupsApi.current(),
    staleTime: 5 * 60 * 1000,
  });
  const fixedAmount =
    settingsData?.data?.fixed_contribution_amount != null
      ? Number(settingsData.data.fixed_contribution_amount)
      : null;
  const nextDue = settingsData?.next_due_date ?? null;

  // Prefill the fixed amount for AKIBA when configured
  useEffect(() => {
    if (fixedAmount != null) {
      setFormData((f) => ({ ...f, amount: String(fixedAmount) }));
    }
  }, [fixedAmount]);

  const [formData, setFormData] = useState({
    contribution_type: "AKIBA",
    period_label: new Date().toISOString().slice(0, 7),
    amount: "",
    proof_message: "",
  });

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [uploadedUrl, setUploadedUrl] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith("image/")) {
      setUploadError("Chagua picha tu (JPG, PNG, GIF, WebP)");
      return;
    }

    // Validate file size (max 5MB)
    if (file.size > 5 * 1024 * 1024) {
      setUploadError("Picha ni kubwa mno. Kiwango cha juu ni 5MB");
      return;
    }

    setSelectedFile(file);
    setUploadError(null);
    setUploadedUrl(null);

    // Create preview
    const reader = new FileReader();
    reader.onload = (ev) => {
      setPreviewUrl(ev.target?.result as string);
    };
    reader.readAsDataURL(file);
  };

  const removeFile = () => {
    setSelectedFile(null);
    setPreviewUrl(null);
    setUploadedUrl(null);
    setUploadError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const uploadFile = async (): Promise<string | null> => {
    if (!selectedFile) return null;

    setIsUploading(true);
    setUploadError(null);

    try {
      const data = await uploadApi.doc(selectedFile, "contributions");
      setUploadedUrl(data.url);
      return data.url;
    } catch (err: any) {
      setUploadError(err.message || "Imeshindikana kupakia picha");
      return null;
    } finally {
      setIsUploading(false);
    }
  };

  const submitMutation = useMutation({
    mutationFn: async (data: { contribution_type: string; period_label: string; amount: number; proof_image_url?: string; proof_message?: string }) => {
      return api.post("/michango", data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      // Refresh cycle status so the dashboard "Mchango ujao" banner updates
      qc.invalidateQueries({ queryKey: ["groups", "settings"] });
      showModal({ title: "Imefanikiwa", message: "Mchango umewasilishwa!", variant: "success", primaryLabel: "Sawa" });
      setFormData({
        contribution_type: "AKIBA",
        period_label: new Date().toISOString().slice(0, 7),
        amount: "",
        proof_message: "",
      });
      removeFile();
    },
    onError: (err: Error) => {
      showModal({ title: "Hitilafu", message: err.message, variant: "error", primaryLabel: "Sawa" });
    },
  });

  if (!user) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.amount || parseFloat(formData.amount) <= 0) {
      showModal({ title: "Hitilafu", message: "Kiasi kinahitajika", variant: "warning", primaryLabel: "Sawa" });
      return;
    }

    // Must have either a file or a message
    if (!selectedFile && !formData.proof_message) {
      showModal({ title: "Hitilafu", message: "Lazima uweke picha ya uthibitisho au ujumbe wa muamala", variant: "warning", primaryLabel: "Sawa" });
      return;
    }

    let proofImageUrl = uploadedUrl;

    // Upload file if selected and not yet uploaded
    if (selectedFile && !uploadedUrl) {
      proofImageUrl = await uploadFile();
      if (!proofImageUrl) return; // Upload failed
    }

    submitMutation.mutate({
      contribution_type: formData.contribution_type,
      period_label: formData.period_label,
      amount: parseFloat(formData.amount),
      proof_image_url: proofImageUrl || undefined,
      proof_message: formData.proof_message || undefined,
    });
  };

  return (
    <AppShell title="Weka Mchango" subtitle="Wasilisha mchango wako kwa uthibitisho">
      <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
        {(fixedAmount != null || nextDue) && (
          <div className="card-surface p-4 border-l-4 border-l-primary">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <CalendarDays className="h-3 w-3" /> Mipangilio ya michango ya kikundi
            </p>
            <p className="text-sm font-semibold mt-0.5">
              {fixedAmount != null && <>TZS {fixedAmount.toLocaleString()}</>}
              {fixedAmount != null && settingsData && <> · {INTERVAL_LABELS[settingsData.data.contribution_interval]}</>}
              {nextDue && <> · mchango ujao: {nextDue}</>}
            </p>
          </div>
        )}
        <PaymentMethodsCard />
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

        {formData.contribution_type === "MFUKO_WA_KIJAMII" ? (
          <MfukoContributionForm />
        ) : (
          <>
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
            <label className="block text-sm font-medium mb-1">
              Kiasi (TZS){fixedAmount != null ? " — lililopangwa na kikundi" : ""}
            </label>
            <input
              type="number"
              step="0.01"
              min="0"
              value={formData.amount}
              onChange={(e) => setFormData({ ...formData, amount: e.target.value })}
              placeholder="0.00"
              readOnly={fixedAmount != null}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-70 read-only:bg-muted/40"
              required
            />
            {fixedAmount != null && (
              <p className="text-xs text-muted-foreground mt-1">
                Kikundi kimepanga mchango wa TZS {fixedAmount.toLocaleString()} — kiasi hiki hakiwezi kubadilishwa.
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              Uthibitisho (Lazima uweke moja)
            </label>

            {/* File Upload Section */}
            <div className="space-y-3">
              {!previewUrl ? (
                <div
                  onClick={() => fileInputRef.current?.click()}
                  className="flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed border-border p-8 cursor-pointer transition-colors hover:border-primary/50 hover:bg-muted/30"
                >
                  <div className="grid h-12 w-12 place-items-center rounded-full bg-muted">
                    <Upload className="h-5 w-5 text-muted-foreground" />
                  </div>
                  <div className="text-center">
                    <p className="text-sm font-medium">Bofya kupakia picha</p>
                    <p className="text-xs text-muted-foreground mt-1">JPG, PNG, GIF, WebP • Max 5MB</p>
                  </div>
                </div>
              ) : (
                <div className="relative rounded-lg border overflow-hidden">
                  <img
                    src={previewUrl}
                    alt="Preview ya uthibitisho"
                    className="max-h-64 w-full object-contain bg-muted/30"
                  />
                  <div className="absolute top-2 right-2 flex gap-2">
                    <button
                      type="button"
                      onClick={removeFile}
                      className="grid h-8 w-8 place-items-center rounded-full bg-destructive text-destructive-foreground shadow-lg hover:bg-destructive/90"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                  <div className="p-3 bg-card border-t">
                    <p className="text-xs text-muted-foreground truncate">{selectedFile?.name}</p>
                    {uploadedUrl && (
                      <p className="text-xs text-success mt-1 flex items-center gap-1">
                        <ImageIcon className="h-3 w-3" /> Imepakiwa
                      </p>
                    )}
                  </div>
                </div>
              )}

              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                onChange={handleFileSelect}
                className="hidden"
              />

              {isUploading && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Inapakia picha...
                </div>
              )}

              {uploadError && (
                <p className="text-sm text-destructive">{uploadError}</p>
              )}

              {/* Divider */}
              <div className="flex items-center gap-3">
                <div className="h-px flex-1 bg-border" />
                <span className="text-xs text-muted-foreground">AU</span>
                <div className="h-px flex-1 bg-border" />
              </div>

              {/* Message alternative */}
              <div className="flex items-start gap-2">
                <MessageSquare className="h-5 w-5 text-muted-foreground mt-2" />
                <div className="flex-1">
                  <label className="text-xs text-muted-foreground">Ujumbe wa Muamala (badala ya picha)</label>
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
          disabled={submitMutation.isPending || isUploading}
          className="w-full inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-3 font-semibold text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
          {submitMutation.isPending || isUploading ? (
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
          </>
        )}
      </form>
    </AppShell>
  );
}
