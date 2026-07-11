import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useAuth } from "@/lib/auth-provider";
import { authApi } from "@/api/auth";
import { tokenStorage } from "@/lib/auth-storage";
import { useQueryClient } from "@tanstack/react-query";
import { KeyRound, ArrowRight, ShieldCheck, Eye, EyeOff } from "lucide-react";

export const Route = createFileRoute("/weka-nenosiri")({
  head: () => ({
    meta: [
      { title: "Weka Nenosiri — Kikundi" },
      { name: "description", content: "Weka nenosiri jipya la akaunti yako." },
    ],
  }),
  component: WekaNenosiriPage,
});

function WekaNenosiriPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const qc = useQueryClient();
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showNew, setShowNew] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (newPassword.length < 6) {
      setError("Nenosiri lazima liwe na angalau herufi 6");
      return;
    }
    if (newPassword === "1-9") {
      setError("Huwezi kutumia nenosiri la mfumo. Chagua nenosiri jipya.");
      return;
    }
    if (newPassword !== confirmPassword) {
      setError("Nenosiri hazifanani");
      return;
    }

    setLoading(true);
    try {
      const res = await authApi.firstLoginSetup({
        new_password: newPassword,
        confirm_password: confirmPassword,
      });
      // Update token and user in cache
      tokenStorage.set(res.token);
      qc.setQueryData(["auth", "me"], res.user);
      navigate({ to: "/dashibodi" });
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Imeshindikana kuweka nenosiri.";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-dvh bg-muted/30">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3 lg:px-8">
          <div className="flex items-center gap-2">
            <span className="grid h-9 w-9 place-items-center rounded-xl bg-primary text-primary-foreground font-display font-bold">K</span>
            <span className="font-display text-lg font-bold">Kikundi</span>
          </div>
        </div>
      </header>
      <main className="mx-auto flex max-w-md flex-col px-4 py-8 lg:py-16">
        <div className="card-surface p-6 lg:p-8">
          <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10">
            <ShieldCheck className="h-6 w-6 text-primary" />
          </div>
          <h1 className="font-display text-2xl font-bold">Weka Nenosiri Jipya</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Karibu{user?.name ? `, ${user.name}` : ""}! Lazima uweke nenosiri jipya kabla ya kuendelea.
          </p>

          <div className="mt-4 rounded-xl bg-amber-50 border border-amber-200 px-4 py-3">
            <p className="text-xs text-amber-800">
              <strong>Muhimu:</strong> Nenosiri la mfumo &quot;1-9&quot; halitaweza kutumika tena baada ya kuweka jipya.
            </p>
          </div>

          <form onSubmit={onSubmit} className="mt-6 space-y-4">
            <div>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-muted-foreground">Nenosiri Jipya</span>
                <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-ring/20">
                  <KeyRound className="h-4 w-4 text-muted-foreground" />
                  <input
                    type={showNew ? "text" : "password"}
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="Angalau herufi 6"
                    autoComplete="new-password"
                    className="w-full bg-transparent text-sm outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => setShowNew(!showNew)}
                    className="text-muted-foreground hover:text-foreground"
                  >
                    {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </label>
            </div>

            <div>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-muted-foreground">Thibitisha Nenosiri</span>
                <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-ring/20">
                  <KeyRound className="h-4 w-4 text-muted-foreground" />
                  <input
                    type={showConfirm ? "text" : "password"}
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="Rudisha nenosiri"
                    autoComplete="new-password"
                    className="w-full bg-transparent text-sm outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => setShowConfirm(!showConfirm)}
                    className="text-muted-foreground hover:text-foreground"
                  >
                    {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </label>
            </div>

            {error && (
              <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-60"
            >
              {loading ? "Inahifadhi..." : "Weka Nenosiri"} <ArrowRight className="h-4 w-4" />
            </button>
          </form>
        </div>
      </main>
    </div>
  );
}
