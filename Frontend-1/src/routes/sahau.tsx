import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { authApi } from "@/api/auth";
import { AuthLayout } from "@/components/AuthLayout";
import { Field } from "@/components/Field";
import { Mail, KeyRound, ArrowRight } from "lucide-react";

export const Route = createFileRoute("/sahau")({
  head: () => ({
    meta: [
      { title: "Umesahau Nenosiri — Money Seeking" },
      { name: "description", content: "Rejesha nenosiri lako la Money Seeking." },
    ],
  }),
  component: SahauPage,
});

function SahauPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [pwd, setPwd] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [ok, setOk] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (pwd !== confirm) return setError("Nenosiri haziendani.");
    setLoading(true);
    try {
      await authApi.resetPassword({ email: email.trim().toLowerCase(), new_password: pwd });
      setOk(true);
      setTimeout(() => navigate({ to: "/ingia" }), 1500);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Imeshindikana kubadilisha nenosiri.";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout
      title="Rejesha nenosiri"
      subtitle="Weka barua pepe yako na nenosiri jipya"
      footer={<p className="text-sm text-muted-foreground"><Link to="/ingia" className="font-semibold text-primary">Rudi kwenye Ingia</Link></p>}
    >
      {ok ? (
        <div className="rounded-xl bg-success/10 p-4 text-center text-sm text-success">
          Nenosiri limebadilishwa. Inakupeleka kwenye Ingia...
        </div>
      ) : (
        <form onSubmit={onSubmit} className="space-y-3">
          <Field icon={Mail} label="Barua pepe" type="email" value={email} onChange={setEmail} placeholder="jina@mfano.com" autoComplete="email" />
          <Field icon={KeyRound} label="Nenosiri jipya" type="password" value={pwd} onChange={setPwd} placeholder="••••••••" autoComplete="new-password" />
          <Field icon={KeyRound} label="Thibitisha" type="password" value={confirm} onChange={setConfirm} placeholder="••••••••" autoComplete="new-password" />
          {error && <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>}
          <button type="submit" disabled={loading} className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-60">
            {loading ? "Inatumwa..." : "Badilisha Nenosiri"} <ArrowRight className="h-4 w-4" />
          </button>
        </form>
      )}
    </AuthLayout>
  );
}
