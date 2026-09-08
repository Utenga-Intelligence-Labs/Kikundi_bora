import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  LayoutDashboard,
  HandCoins,
  History,
  Bell,
  HeartHandshake,
  ArrowRight,
  Loader2,
} from "lucide-react";
import { onboardingApi } from "@/api/onboarding";

const TOUR_STEPS = [
  {
    icon: LayoutDashboard,
    title: "Dashibodi",
    body: (
      <p>
        Hapa ndipo kila kitu kinapoanza: angalia <strong>"Akiba Yangu"</strong>{" "}
        kwenye dashibodi, na kwenye <strong>"Deni Langu"</strong> utaona{" "}
        <strong>"Jumla Unayodaiwa"</strong> — jumla ya michango na adhaba
        zinazokukabili.
      </p>
    ),
    to: "/dashibodi",
  },
  {
    icon: HandCoins,
    title: "Weka Mchango",
    body: (
      <p>
        Wasilisha mchango wako wa AKIBA hapa. Ukurasa unaonyesha pia njia za
        malipo za kikundi (LipaNamba / benki) — tuma kisha wasilisha uthibitisho
        wa malipo yako.
      </p>
    ),
    to: "/weka-mchango",
  },
  {
    icon: History,
    title: "Historia Yangu",
    body: (
      <p>
        Rekodi kamili ya michango yote uliyoweka na mikopo yote — kila kitu
        kikiwa na tarehe na kiasi, wakati wowote ukihitaji kuthibitisha.
      </p>
    ),
    to: "/historia-yangu",
  },
  {
    icon: Bell,
    title: "Arifa",
    body: (
      <p>
        Kengele ya arifa (juu kushoto) itakujulisha: vikumbusho vya tarehe za
        michango, idhini za mikopo na michango yako, na taarifa muhimu za
        kikundi.
      </p>
    ),
    to: "/arifa",
  },
  {
    icon: HeartHandshake,
    title: "Mfuko wa Kijamii",
    body: (
      <p>
        Mfuko wa msaada wa kijamii (mfano: msiba, harusi, ajali). Wakati tukio
        litakapofunguliwa, kila mwanachama anachangia sehemu yake hapa.
      </p>
    ),
    to: "/mfuko-kijamii",
  },
] as const;

/**
 * New-member first-login app tour — a short 5-slide orientation, NOT a wizard:
 * no required actions, "Ruka" available at every step. Marks onboarding_seen
 * on completion or skip (PATCH /members/:id/onboarding-seen) so it never
 * auto-shows again.
 */
export function OnboardingTour({
  open,
  memberId,
  onClose,
}: {
  open: boolean;
  memberId: string;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [step, setStep] = useState(0);
  const [closed, setClosed] = useState(false);

  const seenMutation = useMutation({
    mutationFn: () => onboardingApi.markTourSeen(memberId),
    onSettled: () => {
      // Hide immediately, even if the request failed — never trap the user.
      setClosed(true);
      qc.invalidateQueries({ queryKey: ["auth", "me"] });
      onClose();
    },
  });

  if (!open || closed) return null;

  const finish = () => seenMutation.mutate();

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      data-testid="onboarding-tour"
    >
      <div className="card-surface w-full max-w-md p-6 space-y-4">
        <div className="flex items-center gap-2">
          {(() => {
            const Icon = TOUR_STEPS[step].icon;
            return <Icon className="h-6 w-6 text-primary" />;
          })()}
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Safari ya kwanza — {step + 1} kati ya {TOUR_STEPS.length}
          </p>
        </div>
        <h2 className="font-display text-lg font-bold">
          {TOUR_STEPS[step].title}
        </h2>
        <div className="text-sm text-foreground/90 space-y-2">
          {TOUR_STEPS[step].body}
        </div>
        <p className="text-xs text-muted-foreground">
          Ukurasa: <span className="font-mono">{TOUR_STEPS[step].to}</span>
        </p>

        <div className="flex gap-1.5" aria-hidden>
          {TOUR_STEPS.map((_, i) => (
            <div
              key={i}
              className={`h-1.5 flex-1 rounded-full ${i <= step ? "bg-primary" : "bg-muted"}`}
            />
          ))}
        </div>

        <div className="flex items-center justify-between border-t pt-4">
          <button
            onClick={finish}
            disabled={seenMutation.isPending}
            className="text-sm text-muted-foreground hover:text-foreground disabled:opacity-50"
            data-testid="tour-skip"
          >
            Ruka
          </button>
          {step < TOUR_STEPS.length - 1 ? (
            <button
              onClick={() => setStep((s) => s + 1)}
              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
            >
              Endelea <ArrowRight className="h-4 w-4" />
            </button>
          ) : (
            <button
              onClick={finish}
              disabled={seenMutation.isPending}
              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              data-testid="tour-finish"
            >
              {seenMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : null}
              Sawa, nimeelewa
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
