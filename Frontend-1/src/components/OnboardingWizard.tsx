import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  X,
  Users,
  Wallet,
  PiggyBank,
  PartyPopper,
  CheckCircle2,
  Clock,
  CircleDashed,
  Loader2,
  ArrowRight,
  UserPlus,
} from "lucide-react";
import {
  onboardingApi,
  loadOnboardingProgress,
  saveOnboardingProgress,
  clearOnboardingProgress,
} from "@/api/onboarding";
import { groupsApi } from "@/api/groups";
import { ContributionSettingsCard } from "@/components/ContributionSettingsCard";
import { PaymentMethodsCard } from "@/components/PaymentMethodsCard";

export const OPEN_SETUP_EVENT = "kikundi:open-setup";

/** Ask the app to (re)open the group setup wizard — used by "Setup ya Kikundi". */
export function openGroupSetup() {
  window.dispatchEvent(new CustomEvent(OPEN_SETUP_EVENT));
}

const STEPS = [
  { title: "Karibu kwenye Kikundi chako", icon: PartyPopper },
  { title: "Mipangilio ya Michango", icon: PiggyBank },
  { title: "Njia za Malipo", icon: Wallet },
  { title: "Ongeza Wanachama", icon: Users },
  { title: "Tayari!", icon: CheckCircle2 },
] as const;

/**
 * Mwenyekiti first-time setup wizard — a guided sequence over the EXISTING
 * settings / payment / member-creation functionality. Never blocks the app:
 * every step has "Ruka kwa sasa", and the whole wizard can be skipped.
 * Progress survives refresh: client-side (localStorage) + confirmed against
 * GET /groups/:id/onboarding-status so resume lands on the right step.
 */
export function OnboardingWizard({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [step, setStep] = useState(0);
  const [resumed, setResumed] = useState(false);
  const [finishing, setFinishing] = useState(false);

  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
  });
  const groupId = gs?.data.id;

  const { data: statusResp } = useQuery({
    queryKey: ["onboarding", "status", groupId],
    queryFn: () => onboardingApi.status(groupId!),
    enabled: !!groupId,
  });
  const status = statusResp?.data;

  // Resume: prefer the saved wizard position; if none, jump to the first
  // incomplete step reported by the server. Runs once per open.
  useEffect(() => {
    if (!open || resumed || !status) return;
    const s = status.steps;
    const saved = loadOnboardingProgress();
    const anyDone =
      s.contribution_proposed ||
      s.contribution_approved ||
      s.payment_method_added ||
      s.members_added;
    const firstIncomplete = !anyDone
      ? 0 // fresh group → start at Karibu
      : !s.contribution_proposed && !s.contribution_approved
        ? 1
        : !s.payment_method_added
          ? 2
          : !s.members_added
            ? 3
            : 4;
    setStep(
      saved && saved.step >= 0 && saved.step <= 4
        ? Math.max(saved.step, 1)
        : firstIncomplete,
    );
    setResumed(true);
  }, [open, resumed, status]);

  useEffect(() => {
    if (open && resumed) saveOnboardingProgress({ step, done: [] });
  }, [step, open, resumed]);

  if (!open) return null;

  const finish = async () => {
    if (!groupId) return onClose();
    setFinishing(true);
    try {
      await onboardingApi.complete(groupId);
      qc.invalidateQueries({ queryKey: ["onboarding"] });
    } finally {
      clearOnboardingProgress();
      setFinishing(false);
      onClose();
    }
  };

  const next = () => setStep((s) => Math.min(s + 1, 4));
  const skip = () => {
    if (step >= 4) void finish();
    else next();
  };

  const st = status?.steps;
  const pendingProposal = !!st?.contribution_proposed;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      data-testid="onboarding-wizard"
    >
      <div className="card-surface w-full max-w-2xl max-h-[90dvh] overflow-y-auto p-6 space-y-5">
        {/* Header: step indicator + escape hatch */}
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Setup ya Kikundi — Hatua {step + 1} kati ya 5
            </p>
            <h2 className="font-display text-xl font-bold flex items-center gap-2">
              {(() => {
                const Icon = STEPS[step].icon;
                return <Icon className="h-5 w-5 text-primary" />;
              })()}
              {STEPS[step].title}
            </h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Funga"
            className="rounded-lg p-1.5 hover:bg-muted"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="flex gap-1.5" aria-hidden>
          {STEPS.map((_, i) => (
            <div
              key={i}
              className={`h-1.5 flex-1 rounded-full ${i <= step ? "bg-primary" : "bg-muted"}`}
            />
          ))}
        </div>

        {/* Step 1 — Karibu */}
        {step === 0 && (
          <div className="space-y-3">
            <p className="text-sm">
              Karibu! Hii ni safari fupi ya kuandaa kikundi chako kabla ya
              kuanza kutumia mfumo. Tutapitia:
            </p>
            <ul className="space-y-2 text-sm">
              <li className="flex items-center gap-2">
                <PiggyBank className="h-4 w-4 text-primary" /> Mipangilio ya
                michango — kipindi, tarehe na kiasi
              </li>
              <li className="flex items-center gap-2">
                <Wallet className="h-4 w-4 text-primary" /> Njia za malipo —
                LipaNamba ama akaunti ya benki
              </li>
              <li className="flex items-center gap-2">
                <Users className="h-4 w-4 text-primary" /> Wanachama — wasajili
                wanaoanza
              </li>
            </ul>
            <p className="text-xs text-muted-foreground">
              Unaweza kuruka hatua yoyote na kuikamilisha baadaye kupitia
              "Mipangilio" → "Setup ya Kikundi".
            </p>
          </div>
        )}

        {/* Step 2 — Contribution settings (reuses the existing propose form) */}
        {step === 1 && (
          <div className="space-y-3">
            <div className="rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-950/30 p-3 text-xs">
              <p className="font-semibold flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5 text-amber-600" /> Hii ni
                PENDEKEZO, si mipangilio ya mwisho
              </p>
              <p className="mt-1">
                Katibu atalazimika kuidhinisha pendekezo hili kabla ya kuanza
                kutumika. Kikundi kinaweza kuendelea na hatua nyingine za setup
                wakati pendekezo bado linasubiri.
              </p>
            </div>
            <ContributionSettingsCard />
          </div>
        )}

        {/* Step 3 — Payment methods (reuses the existing management card) */}
        {step === 2 && (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Ongeza angalau njia moja ya malipo — LipaNamba ama akaunti ya
              benki — ili wanachama wajue watalipa wapi. Njia unayoongeza kama
              Mwenyekiti zinakubaliwa moja kwa moja.
            </p>
            <PaymentMethodsCard />
          </div>
        )}

        {/* Step 4 — Members (shortcut into the existing member-creation flow) */}
        {step === 3 && (
          <div className="space-y-3">
            <p className="text-sm">
              Sasa ongeza wanachama wa kwanza. Utapelekwa kwenye fomu ya usajili
              — unaweza kuongeza wanachama kadhaa mfululizo, au kuruka na
              kuwaongeza baadaye.
            </p>
            <p className="text-xs text-muted-foreground">
              Kila mwanachama mpya ataomba idhini ya Katibu kabla ya kuanza
              kutumia akaunti yake.
            </p>
            <button
              onClick={() => {
                void finish();
                navigate({ to: "/wanachama" });
              }}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
            >
              <UserPlus className="h-4 w-4" /> Ongeza wanachama sasa
            </button>
          </div>
        )}

        {/* Step 5 — Done: summary of configured vs pending */}
        {step === 4 && (
          <div className="space-y-3">
            <p className="text-sm">Hiki kile kilichoandaliwa:</p>
            <ul className="space-y-2 text-sm">
              <SummaryRow
                done={!!st?.contribution_approved}
                pending={pendingProposal}
                doneLabel="Mipangilio ya michango: imeidhinishwa na inatumika"
                pendingLabel="Mipangilio ya michango: inasubiri idhini ya Katibu"
                todoLabel="Mipangilio ya michango: bado haijawekwa"
              />
              <SummaryRow
                done={!!st?.payment_method_added}
                doneLabel="Njia za malipo: zimeongezwa"
                todoLabel="Njia za malipo: hazijaongezwa — ongeza baadaye kupitia Mipangilio"
              />
              <SummaryRow
                done={!!st?.members_added}
                doneLabel="Wanachama: wameshajulishwa"
                todoLabel="Wanachama: hakuna bado — ongeza kupitia ukurasa wa Wanachama"
              />
            </ul>
            <p className="text-xs text-muted-foreground">
              Unaweza kukamilisha hatua zilizorukwa wakati wowote: Mipangilio →
              "Setup ya Kikundi".
            </p>
          </div>
        )}

        {/* Footer navigation — skip available at every step */}
        <div className="flex items-center justify-between border-t pt-4">
          <button
            onClick={() => {
              clearOnboardingProgress();
              void finish();
            }}
            disabled={finishing}
            className="text-sm text-muted-foreground hover:text-foreground disabled:opacity-50"
            data-testid="wizard-skip-all"
          >
            Ruka setup yote
          </button>
          <div className="flex items-center gap-2">
            {step < 4 && (
              <button
                onClick={skip}
                className="rounded-lg border border-border px-4 py-2 text-sm font-medium hover:bg-muted"
              >
                Ruka kwa sasa
              </button>
            )}
            {step < 4 ? (
              <button
                onClick={next}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
              >
                Endelea <ArrowRight className="h-4 w-4" />
              </button>
            ) : (
              <button
                onClick={() => void finish()}
                disabled={finishing}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
                data-testid="wizard-finish"
              >
                {finishing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <CheckCircle2 className="h-4 w-4" />
                )}
                Nenda Dashibodi
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function SummaryRow({
  done,
  pending,
  doneLabel,
  pendingLabel,
  todoLabel,
}: {
  done: boolean;
  pending?: boolean;
  doneLabel: string;
  pendingLabel?: string;
  todoLabel: string;
}) {
  return (
    <li className="flex items-center gap-2">
      {done ? (
        <CheckCircle2 className="h-4 w-4 text-success shrink-0" />
      ) : pending ? (
        <Clock className="h-4 w-4 text-amber-600 shrink-0" />
      ) : (
        <CircleDashed className="h-4 w-4 text-muted-foreground shrink-0" />
      )}
      <span>
        {done ? doneLabel : pending ? (pendingLabel ?? todoLabel) : todoLabel}
      </span>
    </li>
  );
}
