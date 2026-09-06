import { useRef, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Upload, X, HeartHandshake } from "lucide-react";
import { welfareApi, type WelfareEvent } from "@/api/welfare";
import { PaymentMethodsCard } from "@/components/PaymentMethodsCard";
import { uploadApi } from "@/api/upload";
import { api } from "@/api/client";
import { useAppModal } from "@/components/AppModal";

const EVENT_LABELS: Record<string, string> = {
  MSIBA: "Msiba",
  HARUSI: "Harusi",
  DHARURA: "Dharura",
  MATIBABU: "Matibabu",
  KUZALIWA: "Kuzaliwa",
  ELIMU: "Elimu",
};

/**
 * Form shown on /weka-mchango when the member selects MFUKO_WA_KIJAMII.
 * Member picks the welfare event (mfuko) they are contributing to, then
 * enters the amount and proof (photo or transaction message) — the same
 * verification workflow as AKIBA contributions (chair confirms).
 */
export function MfukoContributionForm() {
  const qc = useQueryClient();
  const { showModal } = useAppModal();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [eventId, setEventId] = useState("");
  const [amount, setAmount] = useState("");
  const [proofMessage, setProofMessage] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [uploadedUrl, setUploadedUrl] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const { data: eventsData, isLoading: eventsLoading } = useQuery({
    queryKey: ["welfare", "contribute-events"],
    queryFn: () => welfareApi.contributeEvents(),
  });
  const events: WelfareEvent[] = eventsData?.data ?? [];
  const selectedEvent = events.find((e) => e.id === eventId);

  // Fixed per-member obligation for the selected mfuko — fetched from the
  // server, never typed by hand.
  const { data: obligationData, isLoading: obligationLoading, error: obligationError } = useQuery({
    queryKey: ["welfare", "my-obligation", eventId],
    queryFn: () => welfareApi.myObligation(eventId),
    enabled: !!eventId,
    retry: false,
  });
  const obligation = obligationData?.data ?? null;
  const obligationPaid = obligation != null && obligation.status !== "PENDING";
  const obligationMsg = obligationError instanceof Error ? obligationError.message : null;

  const handleSelectEvent = (id: string) => {
    setEventId(id);
    setAmount("");
    setProofMessage("");
    removeFile();
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      setUploadError("Chagua picha tu (JPG, PNG, GIF, WebP)");
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      setUploadError("Picha ni kubwa mno. Kiwango cha juu ni 5MB");
      return;
    }
    setSelectedFile(file);
    setUploadError(null);
    setUploadedUrl(null);
    const reader = new FileReader();
    reader.onload = (ev) => setPreviewUrl(ev.target?.result as string);
    reader.readAsDataURL(file);
  };

  const removeFile = () => {
    setSelectedFile(null);
    setPreviewUrl(null);
    setUploadedUrl(null);
    setUploadError(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const uploadFile = async (): Promise<string | null> => {
    if (!selectedFile) return null;
    setIsUploading(true);
    setUploadError(null);
    try {
      const data = await uploadApi.doc(selectedFile, "contributions");
      setUploadedUrl(data.url);
      return data.url;
    } catch (err: unknown) {
      setUploadError(err instanceof Error ? err.message : "Imeshindikana kupakia picha");
      return null;
    } finally {
      setIsUploading(false);
    }
  };

  const submitMutation = useMutation({
    mutationFn: async (data: {
      period_label: string;
      amount: number;
      welfare_event_id: string;
      proof_image_url?: string;
      proof_message?: string;
    }) => api.post("/michango", { contribution_type: "MFUKO_WA_KIJAMII", ...data }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["michango"] });
      showModal({ title: "Imefanikiwa", message: "Mchango wa mfuko umewasilishwa! Unasubiri idhini ya Mwenyekiti.", variant: "success", primaryLabel: "Sawa" });
      setAmount("");
      setProofMessage("");
      setEventId("");
      removeFile();
    },
    onError: (err: Error) => showModal({ title: "Hitilafu", message: err.message, variant: "error", primaryLabel: "Sawa" }),
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!eventId) {
      showModal({ title: "Hitilafu", message: "Lazima uchague mfuko wa kijamii", variant: "warning", primaryLabel: "Sawa" });
      return;
    }
    if (obligationLoading || obligation == null) {
      showModal({ title: "Hitilafu", message: "Subiri kiwango chako kipakiwe kwanza", variant: "warning", primaryLabel: "Sawa" });
      return;
    }
    if (obligationPaid) {
      showModal({ title: "Hitilafu", message: "Tayari umelipa kiwango cha mfuko huu", variant: "warning", primaryLabel: "Sawa" });
      return;
    }
    if (!selectedFile && !proofMessage) {
      showModal({ title: "Hitilafu", message: "Lazima uweke picha ya uthibitisho au ujumbe wa muamala", variant: "warning", primaryLabel: "Sawa" });
      return;
    }

    let proofImageUrl = uploadedUrl;
    if (selectedFile && !uploadedUrl) {
      proofImageUrl = await uploadFile();
      if (!proofImageUrl) return; // upload failed
    }

    submitMutation.mutate({
      period_label: new Date().toISOString().slice(0, 7),
      amount: Number(obligation.amount),
      welfare_event_id: eventId,
      proof_image_url: proofImageUrl || undefined,
      proof_message: proofMessage || undefined,
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
      {/* Event selector */}
      <div className="card-surface p-6">
        <h3 className="font-display text-lg font-semibold mb-4 flex items-center gap-2">
          <HeartHandshake className="h-5 w-5 text-primary" />
          Chagua Mfuko wa Kijamii
        </h3>
        {eventsLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
            <Loader2 className="h-4 w-4 animate-spin" /> Inapakia mifuko...
          </div>
        ) : events.length === 0 ? (
          <p className="text-sm text-muted-foreground py-2">
            Hakuna mfuko wa kijamii ulioidhinishwa kwa sasa. Mfuko hufunguliwa na Mweka Hazina na kuidhinishwa na Mwenyekiti.
          </p>
        ) : (
          <select
            value={eventId}
            onChange={(e) => handleSelectEvent(e.target.value)}
            className="w-full rounded-lg border border-border bg-background px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
            required
          >
            <option value="">— Chagua mfuko —</option>
            {events.map((ev) => (
              <option key={ev.id} value={ev.id}>
                {EVENT_LABELS[ev.event_type] ?? ev.event_type} — {ev.member?.full_name ?? ""} (
                {Number(ev.amount_approved ?? ev.amount_requested).toLocaleString()} TZS)
              </option>
            ))}
          </select>
        )}
        {selectedEvent && (
          <div className="mt-3 rounded-lg border bg-muted/30 p-3 text-sm">
            <p className="font-medium">
              {EVENT_LABELS[selectedEvent.event_type] ?? selectedEvent.event_type} ·{" "}
              <span className="text-muted-foreground">
                kwa ajili ya {selectedEvent.member?.full_name ?? "mwanachama"}
              </span>
            </p>
            <p className="text-muted-foreground mt-1">{selectedEvent.description}</p>
            <p className="mt-2 text-sm">
              {obligationLoading ? (
                <span className="inline-flex items-center gap-1 text-muted-foreground">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" /> Inapakia kiwango chako...
                </span>
              ) : obligation != null ? (
                <>Kiwango kilichowekwa kwako: <strong>{Number(obligation.amount).toLocaleString()} TZS</strong>
                  {obligationPaid && <span className="ml-2 text-xs text-success">✓ Tayari umelipa</span>}
                </>
              ) : (
                <span className={obligationMsg != null && /haijaidhinishwa|subiri/i.test(obligationMsg) ? "font-medium text-amber-700" : "text-destructive"}>{obligationMsg ?? "Huna kiwango kilichowekwa kwa mfuko huu."}</span>
              )}
            </p>
          </div>
        )}
      </div>

      {selectedEvent && (
        <div className="card-surface p-6">
          <h3 className="font-display text-lg font-semibold mb-1">Namba ya Malipo</h3>
          <p className="text-xs text-muted-foreground mb-3">Tuma kiwango hapo juu kwenda mojawapo ya njia hizi, kisha weka uthibitisho hapa chini.</p>
          <PaymentMethodsCard />
        </div>
      )}

      {/* Amount (locked to the fixed per-member obligation) + proof card */}
      <div className="card-surface p-6 space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Kiasi (TZS) — kilichowekwa</label>
          <input
            type="number"
            value={obligation != null ? String(obligation.amount) : amount}
            readOnly
            placeholder="Chagua mfuko kwanza"
            className="w-full rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm focus:outline-none"
          />
          <p className="mt-1 text-xs text-muted-foreground">
            Kiasi hakibadilishwi — ni kiwango kilichowekwa kwa kila mwanachama kwenye mfuko huu.
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium mb-2">
            Uthibitisho (Lazima uweke moja)
          </label>

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
                <div className="absolute top-2 right-2">
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
                  {uploadedUrl && <p className="text-xs text-success mt-1">Imepakiwa</p>}
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
                <Loader2 className="h-4 w-4 animate-spin" /> Inapakia picha...
              </div>
            )}

            {uploadError && <p className="text-sm text-destructive">{uploadError}</p>}

            <div className="flex items-center gap-3">
              <div className="h-px flex-1 bg-border" />
              <span className="text-xs text-muted-foreground">AU</span>
              <div className="h-px flex-1 bg-border" />
            </div>

            <div>
              <label className="text-xs text-muted-foreground">
                Ujumbe wa Muamala (badala ya picha)
              </label>
              <textarea
                value={proofMessage}
                onChange={(e) => setProofMessage(e.target.value)}
                placeholder="Namba ya muamala au maelezo..."
                rows={3}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              />
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
            <Loader2 className="h-4 w-4 animate-spin" /> Inatuma...
          </>
        ) : (
          <>Wasilisha Mchango wa Mfuko</>
        )}
      </button>
    </form>
  );
}
