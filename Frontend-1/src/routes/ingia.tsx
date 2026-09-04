import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useAuth } from "@/lib/auth-provider";
import { AuthLayout } from "@/components/AuthLayout";
import { Field } from "@/components/Field";
import { Mail, KeyRound, ArrowRight, Phone, Eye, EyeOff } from "lucide-react";

export const Route = createFileRoute("/ingia")({
  head: () => ({
    meta: [
      { title: "Ingia — Kikundi" },
      { name: "description", content: "Ingia kwenye akaunti yako ya Kikundi." },
    ],
  }),
  component: IngiaPage,
});

function IngiaPage() {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [loginId, setLoginId] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await login({ email: loginId.trim(), password });
      if (res.first_login_required) {
        navigate({ to: "/weka-nenosiri" });
      } else {
        navigate({ to: "/dashibodi" });
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Imeshindikana kuingia.";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const isPhone = !loginId.includes("@") && loginId.length > 0;

  return (
    <AuthLayout
      title="Karibu tena"
      subtitle="Ingia kwenye akaunti yako ili kuendelea"
      footer={
        <p className="text-sm text-muted-foreground">
          Huna akaunti? Wasiliana na Mwenyekiti kujisajili.
        </p>
      }
    >
      <form onSubmit={onSubmit} className="space-y-3">
        <Field
          icon={isPhone ? Phone : Mail}
          label="Nambari ya simu au Barua pepe"
          type="text"
          value={loginId}
          onChange={setLoginId}
          placeholder="0712345678 au jina@mfano.com"
          autoComplete="username"
        />
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-muted-foreground">
            Nenosiri
          </span>
          <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-ring/20">
            <KeyRound className="h-4 w-4 text-muted-foreground" />
            <input
              type={showPassword ? "text" : "password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="current-password"
              className="w-full bg-transparent text-sm outline-none"
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              aria-label={showPassword ? "Ficha nenosiri" : "Onesha nenosiri"}
              title={showPassword ? "Ficha nenosiri" : "Onesha nenosiri"}
              className="shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground"
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
        </label>
        <div className="flex items-center justify-between text-xs">
          <span />
          <Link to="/sahau" className="font-medium text-primary">Umesahau nenosiri?</Link>
        </div>
        {error && <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-60"
        >
          {loading ? "Inaingia..." : "Ingia"} <ArrowRight className="h-4 w-4" />
        </button>
      </form>
    </AuthLayout>
  );
}
