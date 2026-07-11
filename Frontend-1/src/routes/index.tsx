import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/lib/auth-provider";
import { DEMO_ACCOUNTS } from "@/lib/role-guards";
import type { Jukumu } from "@/api/types";
import { PiggyBank, Banknote, Users, FileBarChart2, Check } from "lucide-react";
import heroImage from "@/assets/hero-app.png";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Kikundi — Mfumo wa Ushirika wa Akiba na Mikopo" },
      { name: "description", content: "Mfumo wako wa kidijitali wa kusimamia wanachama, michango, mikopo na marejesho ya kikundi chako — kwa Kiswahili, kwenye simu yako." },
      { property: "og:title", content: "Kikundi — Mfumo wa Ushirika wa Akiba na Mikopo" },
      { property: "og:description", content: "Simamia kikundi chako cha akiba na mikopo kwa urahisi — kwa Kiswahili, kwenye simu na desktop." },
      { property: "og:image", content: "/icon.png" },
    ],
  }),
  component: HomePage,
});

function HomePage() {
  const { user, isLoading, login } = useAuth();
  const navigate = useNavigate();

  const loginDemo = async (jukumu: Jukumu) => {
    const account = DEMO_ACCOUNTS[jukumu];
    if (!account) return;
    try {
      await login({ email: account.email, password: account.password });
      navigate({ to: "/dashibodi" });
    } catch {
      // silent fail on landing page
    }
  };

  return (
    <div className="min-h-dvh bg-background">
      {/* Top nav */}
      <header className="sticky top-0 z-30 border-b border-border bg-background/90 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-3 lg:px-8">
          <Link to="/" className="flex items-center gap-2">
            <span className="grid h-10 w-10 place-items-center rounded-xl bg-primary text-primary-foreground font-display text-lg font-bold">K</span>
            <span className="font-display text-xl font-bold tracking-tight">Kikundi</span>
          </Link>
          <div className="flex items-center gap-2">
            {user ? (
              <Link to="/dashibodi" className="rounded-full bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground">
                Dashibodi
              </Link>
            ) : (
              <Link to="/ingia" className="rounded-full bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground">
                Ingia
              </Link>
            )}
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto max-w-6xl px-4 pt-10 pb-12 lg:px-8 lg:pt-20 lg:pb-20">
        <div className="grid items-center gap-10 lg:grid-cols-2">
          <div>
            <h1 className="animate-slide-in-left font-display text-4xl font-extrabold leading-tight tracking-tight lg:text-6xl">
              Simamia kikundi chako cha <span className="text-primary">akiba na mikopo</span> kwa urahisi.
            </h1>
            <p className="mt-4 max-w-xl text-base text-muted-foreground lg:text-lg">
              Mfumo wako wa kidijitali wa kusimamia wanachama, michango, mikopo na marejesho.
              Rahisi, salama, kwa lugha ya Kiswahili — kwenye simu yako.
            </p>
            <div className="mt-6 flex flex-row flex-nowrap items-center gap-2 sm:gap-3">
              <Link to={user ? "/dashibodi" : "/ingia"} className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-full bg-primary px-4 py-2.5 text-xs font-semibold text-primary-foreground shadow-lg shadow-primary/20 sm:px-6 sm:py-3 sm:text-sm">
                {user ? "Dashibodi" : "Ingia"}
              </Link>
            </div>
            <ul className="mt-6 grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
              {["Lugha ya Kiswahili 100%", "Inafanya kazi simu na desktop", "Hifadhi salama kwenye kifaa", "Rahisi kutumia"].map((t) => (
                <li key={t} className="flex items-center gap-2"><Check className="h-4 w-4 text-primary" /> {t}</li>
              ))}
            </ul>
          </div>
          <div className="relative">
            <img
              src={heroImage}
              alt="Mfumo wa Kikundi kwenye simu"
              width={1024}
              height={1024}
              className="w-full"
            />
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="border-y border-border bg-muted/40 py-14 lg:py-20">
        <div className="mx-auto max-w-6xl px-4 lg:px-8">
          <h2 className="font-display text-3xl font-bold lg:text-4xl">Kila kitu kikundi chako kinachohitaji</h2>
          <p className="mt-2 max-w-2xl text-muted-foreground">Vipengele vinavyofanya kazi ya hazina iwe rahisi — bila daftari.</p>
          <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[
              { i: Users, t: "Wanachama", d: "Sajili na hariri taarifa za wanachama wote." },
              { i: PiggyBank, t: "Michango", d: "Kumbuka michango ya kila mwezi kwa kila mwanachama." },
              { i: Banknote, t: "Mikopo", d: "Toa mikopo na fuatilia salio na muda wa kurejesha." },
              { i: FileBarChart2, t: "Ripoti", d: "Ripoti za uwazi kwa ajili ya vikao vya kikundi." },
            ].map(({ i: Icon, t, d }) => (
              <div key={t} className="card-surface p-5">
                <span className="grid h-11 w-11 place-items-center rounded-xl bg-accent text-primary">
                  <Icon className="h-5 w-5" />
                </span>
                <h3 className="mt-4 font-display text-lg font-semibold">{t}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{d}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How it works */}
      <section id="namna" className="mx-auto max-w-6xl px-4 py-14 lg:px-8 lg:py-20">
        <h2 className="font-display text-3xl font-bold lg:text-4xl">Inafanya nini</h2>
        <div className="mt-8 grid gap-5 lg:grid-cols-3">
          {[
            { n: "1", t: "Wanachama", d: "Sajili na fuatilia taarifa za wanachama wote wa kikundi." },
            { n: "2", t: "Michango na Mikopo", d: "Rekodi michango ya mwezi, toa mikopo na fuatilia marejesho." },
            { n: "3", t: "Ripoti", d: "Angalia ripoti za uwazi za hali ya kikundi chako — wakati wowote." },
          ].map((s) => (
            <div key={s.n} className="card-surface p-6">
              <span className="grid h-10 w-10 place-items-center rounded-full bg-primary text-primary-foreground font-display font-bold">{s.n}</span>
              <h3 className="mt-4 font-display text-lg font-semibold">{s.t}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{s.d}</p>
            </div>
          ))}
        </div>
      </section>

      <footer className="border-t border-border py-8">
        <div className="mx-auto flex max-w-6xl flex-col gap-5 px-4 lg:px-8">
          <div className="flex flex-col items-center justify-between gap-3 border-t border-border pt-5 text-xs text-muted-foreground sm:flex-row">
            <p>© 2026 Kikundi · BIG LITE CODE — Iringa, Tanzania</p>
            <div className="flex gap-4">
              <Link to="/ingia" className="hover:text-foreground">Ingia</Link>
              <Link to="/sahau" className="hover:text-foreground">Umesahau nenosiri?</Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
