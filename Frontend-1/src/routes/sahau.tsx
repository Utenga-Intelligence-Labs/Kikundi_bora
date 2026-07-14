import { createFileRoute, Link } from "@tanstack/react-router";
import { AuthLayout } from "@/components/AuthLayout";
import { KeyRound, Phone } from "lucide-react";

export const Route = createFileRoute("/sahau")({
  head: () => ({
    meta: [
      { title: "Umesahau Nenosiri — Kikundi" },
      { name: "description", content: "Omba msaada wa kurejesha nenosiri lako." },
    ],
  }),
  component: SahauPage,
});

/**
 * Backend has no public self-service reset (only admin POST /admin/auth/reset-password
 * and admin user reset). Do not present an email+new_password form that claims to reset.
 */
function SahauPage() {
  return (
    <AuthLayout
      title="Umesahau nenosiri?"
      subtitle="Hakuna urejesho wa nenosiri wa umma. Wasiliana na uongozi."
      footer={
        <p className="text-sm text-muted-foreground">
          <Link to="/ingia" className="font-semibold text-primary">
            Rudi kwenye Ingia
          </Link>
        </p>
      }
    >
      <div className="space-y-4">
        <div className="rounded-xl border border-border bg-muted/40 p-4">
          <div className="flex items-start gap-3">
            <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-primary/10">
              <KeyRound className="h-5 w-5 text-primary" />
            </div>
            <div className="min-w-0 space-y-2 text-sm">
              <p className="font-medium text-foreground">
                Omba msimamizi au Katibu/Mwenyekiti akuweke nenosiri upya.
              </p>
              <p className="text-muted-foreground">
                Baada ya kuwekewa nenosiri la muda, utaingia na utakazwa kuweka nenosiri jipya mwenyewe.
              </p>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-dashed border-border p-4 text-sm text-muted-foreground">
          <div className="mb-2 flex items-center gap-2 font-medium text-foreground">
            <Phone className="h-4 w-4" />
            Nani wa kuwasiliana naye
          </div>
          <ul className="list-inside list-disc space-y-1">
            <li>Mwenyekiti (anayeweza kuunda akaunti / kuomba msaada)</li>
            <li>Msimamizi wa mfumo (anayeweza kuweka nenosiri upya)</li>
          </ul>
        </div>

        <Link
          to="/ingia"
          className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground"
        >
          Rudi kwenye Ingia
        </Link>
      </div>
    </AuthLayout>
  );
}
