import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-provider";
import { groupsApi } from "@/api/groups";
import { onboardingApi, ONBOARDING_PROGRESS_KEY } from "@/api/onboarding";
import {
  OnboardingWizard,
  OPEN_SETUP_EVENT,
} from "@/components/OnboardingWizard";
import { OnboardingTour } from "@/components/OnboardingTour";

/**
 * Mounts both onboarding surfaces, decided purely from auth state:
 *  - Mwenyekiti (chair) of a group with onboarding_completed=false → setup
 *    wizard. Re-openable via the "Setup ya Kikundi" link (OPEN_SETUP_EVENT).
 *  - Approved member with onboarding_seen=false → short app tour, exactly once.
 * Neither blocks the app: both render as dismissable overlays.
 */
export function OnboardingGate() {
  const { user } = useAuth();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [tourOpen, setTourOpen] = useState(false);
  const [tourShown, setTourShown] = useState(false);

  // "Setup ya Kikundi" link (group settings) re-opens the wizard on demand —
  // including after completion, so skipped steps can be revisited.
  useEffect(() => {
    const handler = () => setWizardOpen(true);
    window.addEventListener(OPEN_SETUP_EVENT, handler);
    return () => window.removeEventListener(OPEN_SETUP_EVENT, handler);
  }, []);

  const isChair = user?.role === "chair" && user.status === "ACTIVE";

  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
    enabled: isChair,
    staleTime: 5 * 60 * 1000,
  });
  const groupId = gs?.data.id;

  const { data: statusResp } = useQuery({
    queryKey: ["onboarding", "status", groupId],
    queryFn: () => onboardingApi.status(groupId!),
    enabled: !!groupId && isChair,
    staleTime: 30 * 1000,
  });
  const status = statusResp?.data;

  // Auto-open the wizard only for a chair whose group hasn't completed setup.
  useEffect(() => {
    if (!user || !isChair || !status) return;
    if (!status.onboarding_completed) setWizardOpen(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status?.onboarding_completed, isChair, user?.id]);

  // Auto-show the tour exactly once for an approved member.
  useEffect(() => {
    if (!user || tourShown) return;
    if (
      user.role === "member" &&
      user.status === "ACTIVE" &&
      user.onboarding_seen === false &&
      user.member_id
    ) {
      setTourOpen(true);
      setTourShown(true);
    }
  }, [user, tourShown]);

  // Leftover wizard progress from a previous session is meaningless once
  // setup completed — keep storage tidy.
  useEffect(() => {
    if (status?.onboarding_completed) {
      try {
        localStorage.removeItem(ONBOARDING_PROGRESS_KEY);
      } catch {
        /* ignore */
      }
    }
  }, [status?.onboarding_completed]);

  if (!user) return null;

  return (
    <>
      {isChair && (
        <OnboardingWizard
          open={wizardOpen}
          onClose={() => setWizardOpen(false)}
        />
      )}
      <OnboardingTour
        open={tourOpen}
        memberId={user.member_id ?? ""}
        onClose={() => setTourOpen(false)}
      />
    </>
  );
}
