import { createFileRoute, Link } from "@tanstack/react-router";
import { AuthLayout } from "@/components/AuthLayout";
import { UserPlus, ShieldCheck, Phone } from "lucide-react";

export const Route = createFileRoute("/sajili")({
  head: () => ({
    meta: [
      { title: "Sajili — Kikundi" },
      { name: "description", content: "Jinsi ya kujiunga na Kikundi." },
    ],
  }),
  component: SajiliPage,
});

function SajiliPage() {
  return (
    <AuthLayout
      title="Jiunge na Kikundi"
      subtitle="Usajili unafanywa na Mwenyekiti wa kikundi chako"
      footer={
        <p className="text-sm text-muted-foreground">
          Tayari una akaunti? <Link to="/ingia" className="font-semibold text-primary">Ingia</Link>
        </p>
      }
    >
      <div className="space-y-4">
        <div className="rounded-xl border border-border bg-muted/30 p-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10">
              <UserPlus className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium">Hatua ya 1: Mwenyekiti anaunda akaunti</p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                Mwenyekiti wako ataunda akaunti yako kwa kutumia jina na nambari ya simu.
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-muted/30 p-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10">
              <ShieldCheck className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium">Hatua ya 2: Katibu anaongeza</p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                Katibu anakagua na kuidhinisha akaunti yako. Utapata taarifa ukishaidhinishwa.
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-muted/30 p-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10">
              <Phone className="h-4 w-4 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium">Hatua ya 3: Weka nenosiri lako</p>
              <p className="mt-0.5 text-xs text-muted-foreground">
                Baada ya kuidhinishwa, ingia kwa nenosiri la mfumo &quot;1-9&quot; na utakazwa kuweka nenosiri jipya.
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl bg-primary/5 border border-primary/20 p-4 text-center">
          <p className="text-sm text-muted-foreground">
            Wasiliana na <strong className="text-foreground">Mwenyekiti</strong> wa kikundi chako ili akujiunge.
          </p>
        </div>
      </div>
    </AuthLayout>
  );
}
