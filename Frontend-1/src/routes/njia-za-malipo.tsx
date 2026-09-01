import { createFileRoute } from "@tanstack/react-router";
import { AppShell } from "@/components/AppShell";
import { PaymentMethodsCard } from "@/components/PaymentMethodsCard";
import { requireRole } from "@/lib/role-guards";

export const Route = createFileRoute("/njia-za-malipo")({
  head: () => ({
    meta: [
      { title: "Njia za Malipo — Money Seeking" },
      {
        name: "description",
        content: "Simamia LipaNamba na akaunti za benki za kikundi.",
      },
    ],
  }),
  beforeLoad: () => {
    // Mwenyekiti na Mweka Hazina tu — wanachama wanaona kwenye Weka Mchango
    requireRole("chair", "treasurer");
  },
  component: NjiaZaMalipoPage,
});

function NjiaZaMalipoPage() {
  return (
    <AppShell
      title="Njia za Malipo"
      subtitle="Simamia LipaNamba na akaunti za benki za kikundi — wanachama wanaona hii taarifa wakiwa wanachanga"
    >
      <div className="max-w-3xl">
        <PaymentMethodsCard />
        <p className="mt-4 text-xs text-muted-foreground">
          Mabadiliko yanaonekana kwa wanachama mara moja kwenye ukurasa wa "Weka Mchango".
          Kuzima (deactivate) kunaficha njia kwa wanachama bila kuifuta kwenye kumbukumbu.
        </p>
      </div>
    </AppShell>
  );
}
