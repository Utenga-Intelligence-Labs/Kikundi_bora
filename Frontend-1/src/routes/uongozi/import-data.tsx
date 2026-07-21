import { createFileRoute } from "@tanstack/react-router";
import { useState, useRef } from "react";
import { useAuth } from "@/lib/auth-provider";
import { requireRole } from "@/lib/role-guards";
import { AppShell } from "@/components/AppShell";
import { Upload, FileSpreadsheet, CheckCircle, XCircle, Loader2, AlertTriangle, Download } from "lucide-react";

export const Route = createFileRoute("/uongozi/import-data")({
  beforeLoad: () => {
    requireRole("chair", "treasurer");
  },
  component: ImportDataPage,
});

interface ImportResult {
  total_rows: number;
  imported: number;
  skipped: number;
  errors?: string[];
}

function ImportDataPage() {
  const { user } = useAuth();
  const [importType, setImportType] = useState<"contributions" | "loans">("contributions");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!user) return null;

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.name.endsWith(".csv")) {
      setError("Chagua faili ya CSV tu");
      return;
    }

    setSelectedFile(file);
    setError(null);
    setResult(null);
  };

  const handleImport = async () => {
    if (!selectedFile) return;

    setIsUploading(true);
    setError(null);
    setResult(null);

    try {
      const token = localStorage.getItem("auth_token");
      const formData = new FormData();
      formData.append("file", selectedFile);

      const endpoint = importType === "contributions"
        ? "/api/v1/import/contributions"
        : "/api/v1/import/loans";

      const res = await fetch(endpoint, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: formData,
      });

      const data = await res.json();

      if (!res.ok) {
        throw new Error(data.message || "Import imeshindikana");
      }

      setResult(data.data);
      setSelectedFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    } catch (err: any) {
      setError(err.message || "Import imeshindikana");
    } finally {
      setIsUploading(false);
    }
  };

  const downloadTemplate = () => {
    const template = importType === "contributions"
      ? "member_code,amount,type,date,status\nKKK-0001,50000,AKIBA,2024-01-15,CONFIRMED\nKKK-0002,30000,MFUKO_WA_KIJAMII,2024-01-15,CONFIRMED"
      : "member_code,amount,purpose,due_date,status,approved_amount\nKKK-0001,200000,Biashara,2024-12-31,CLOSED,200000";

    const blob = new Blob([template], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `template-${importType}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <AppShell title="Ingiza Data ya Zamani" subtitle="Pakia data kutoka vitabu vya zamani (CSV)">
      <div className="space-y-6 max-w-2xl">
        {/* Import Type Selector */}
        <div className="card-surface p-6">
          <h3 className="font-display text-lg font-semibold mb-4">Aina ya Data</h3>
          <div className="space-y-3">
            <label className="flex items-start gap-3 rounded-lg border p-4 cursor-pointer transition-colors hover:bg-muted/50">
              <input
                type="radio"
                name="importType"
                value="contributions"
                checked={importType === "contributions"}
                onChange={() => { setImportType("contributions"); setSelectedFile(null); setError(null); setResult(null); }}
                className="mt-0.5"
              />
              <div>
                <p className="font-medium">Michango</p>
                <p className="text-sm text-muted-foreground">Ingiza michango ya zamani (AKIBA / Mfuko wa Kijamii)</p>
              </div>
            </label>
            <label className="flex items-start gap-3 rounded-lg border p-4 cursor-pointer transition-colors hover:bg-muted/50">
              <input
                type="radio"
                name="importType"
                value="loans"
                checked={importType === "loans"}
                onChange={() => { setImportType("loans"); setSelectedFile(null); setError(null); setResult(null); }}
                className="mt-0.5"
              />
              <div>
                <p className="font-medium">Mikopo</p>
                <p className="text-sm text-muted-foreground">Ingiza mikopo ya zamani</p>
              </div>
            </label>
          </div>
        </div>

        {/* Template Download */}
        <div className="card-surface p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Mfano wa CSV</p>
              <p className="text-xs text-muted-foreground">Pakua mfano wa jinsi ya kujaza data</p>
            </div>
            <button
              onClick={downloadTemplate}
              className="inline-flex items-center gap-1.5 rounded-lg bg-muted px-3 py-2 text-sm font-medium hover:bg-muted/80"
            >
              <Download className="h-4 w-4" />
              Pakua Mfano
            </button>
          </div>
        </div>

        {/* File Upload */}
        <div className="card-surface p-6">
          <h3 className="font-display text-lg font-semibold mb-4">Pakia Faili ya CSV</h3>

          {importType === "contributions" && (
            <div className="mb-4 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs font-medium mb-1">Muundo wa CSV (Michango):</p>
              <code className="text-xs text-muted-foreground">
                member_code, amount, type, date, status
              </code>
              <p className="text-xs text-muted-foreground mt-1">
                Mfano: KKK-0001, 50000, AKIBA, 2024-01-15, CONFIRMED
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                Aina: AKIBA au MFUKO_WA_KIJAMII | Status: CONFIRMED, PENDING_VERIFICATION, REJECTED
              </p>
            </div>
          )}

          {importType === "loans" && (
            <div className="mb-4 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs font-medium mb-1">Muundo wa CSV (Mikopo):</p>
              <code className="text-xs text-muted-foreground">
                member_code, amount, purpose, due_date, status, approved_amount
              </code>
              <p className="text-xs text-muted-foreground mt-1">
                Mfano: KKK-0001, 200000, Biashara, 2024-12-31, CLOSED, 200000
              </p>
            </div>
          )}

          {/* Drop zone */}
          {!selectedFile ? (
            <div
              onClick={() => fileInputRef.current?.click()}
              className="flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed border-border p-8 cursor-pointer transition-colors hover:border-primary/50 hover:bg-muted/30"
            >
              <div className="grid h-12 w-12 place-items-center rounded-full bg-muted">
                <Upload className="h-5 w-5 text-muted-foreground" />
              </div>
              <div className="text-center">
                <p className="text-sm font-medium">Bofya kupakia faili ya CSV</p>
                <p className="text-xs text-muted-foreground mt-1">.csv files tu</p>
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-3 rounded-lg border p-4">
              <FileSpreadsheet className="h-8 w-8 text-green-600" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{selectedFile.name}</p>
                <p className="text-xs text-muted-foreground">
                  {(selectedFile.size / 1024).toFixed(1)} KB
                </p>
              </div>
              <button
                onClick={() => {
                  setSelectedFile(null);
                  if (fileInputRef.current) fileInputRef.current.value = "";
                }}
                className="text-sm text-destructive hover:underline"
              >
                Ondoa
              </button>
            </div>
          )}

          <input
            ref={fileInputRef}
            type="file"
            accept=".csv"
            onChange={handleFileSelect}
            className="hidden"
          />

          {/* Import Button */}
          <button
            onClick={handleImport}
            disabled={!selectedFile || isUploading}
            className="mt-4 w-full inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-3 font-semibold text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {isUploading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Inaingiza data...
              </>
            ) : (
              <>
                <Upload className="h-4 w-4" />
                Anza Import
              </>
            )}
          </button>
        </div>

        {/* Error */}
        {error && (
          <div className="card-surface p-4 border-destructive/50 bg-destructive/5">
            <div className="flex items-start gap-3">
              <XCircle className="h-5 w-5 text-destructive shrink-0 mt-0.5" />
              <div>
                <p className="font-medium text-destructive">Import imeshindikana</p>
                <p className="text-sm text-destructive/80 mt-1">{error}</p>
              </div>
            </div>
          </div>
        )}

        {/* Result */}
        {result && (
          <div className="card-surface p-6 border-green-200 bg-green-50/50">
            <div className="flex items-start gap-3 mb-4">
              <CheckCircle className="h-6 w-6 text-green-600 shrink-0" />
              <div>
                <p className="font-display text-lg font-semibold text-green-800">Import Imekamilika!</p>
                <p className="text-sm text-green-700">Data ya zamani imeingizwa kwenye mfumo</p>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-3 mb-4">
              <div className="text-center p-3 bg-white rounded-lg">
                <p className="text-2xl font-bold">{result.total_rows}</p>
                <p className="text-xs text-muted-foreground">Jumla</p>
              </div>
              <div className="text-center p-3 bg-white rounded-lg">
                <p className="text-2xl font-bold text-green-600">{result.imported}</p>
                <p className="text-xs text-muted-foreground">Ziliingizwa</p>
              </div>
              <div className="text-center p-3 bg-white rounded-lg">
                <p className="text-2xl font-bold text-amber-600">{result.skipped}</p>
                <p className="text-xs text-muted-foreground">Zilikataliwa</p>
              </div>
            </div>

            {result.errors && result.errors.length > 0 && (
              <div className="p-3 bg-white rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <AlertTriangle className="h-4 w-4 text-amber-600" />
                  <p className="text-sm font-medium">Makosa ({result.errors.length})</p>
                </div>
                <div className="max-h-40 overflow-y-auto space-y-1">
                  {result.errors.slice(0, 20).map((err, i) => (
                    <p key={i} className="text-xs text-muted-foreground">• {err}</p>
                  ))}
                  {result.errors.length > 20 && (
                    <p className="text-xs text-muted-foreground">... na mengine {result.errors.length - 20}</p>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </AppShell>
  );
}
