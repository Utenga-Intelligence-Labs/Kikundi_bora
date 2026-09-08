import { api } from "./client";

// --- Types ---

export interface OnboardingSteps {
  /** A contribution-settings proposal is awaiting Katibu approval. */
  contribution_proposed: boolean;
  /** Approved settings exist (due date and/or fixed amount set). */
  contribution_approved: boolean;
  payment_method_added: boolean;
  members_added: boolean;
}

export interface OnboardingStatus {
  onboarding_completed: boolean;
  steps: OnboardingSteps;
}

// --- API ---

export const onboardingApi = {
  /** Which setup steps are done — lets the wizard resume mid-way. */
  status: (groupId: string) =>
    api.get<{ data: OnboardingStatus }>(`/groups/${groupId}/onboarding-status`),

  /** Mwenyekiti only — marks the wizard finished/skipped (stops auto-open). */
  complete: (groupId: string) =>
    api.patch<{ message: string }>(`/groups/${groupId}/onboarding-complete`),

  /** Member marks the first-login app tour as seen (self only). */
  markTourSeen: (memberId: string) =>
    api.patch<{ message: string }>(`/members/${memberId}/onboarding-seen`),
};

/** localStorage key for the wizard's client-side progress (step position). */
export const ONBOARDING_PROGRESS_KEY = "kikundi-setup-progress";

export interface OnboardingProgress {
  step: number;
  /** Steps the mwenyekiti completed during this setup run. */
  done: number[];
}

export function loadOnboardingProgress(): OnboardingProgress | null {
  try {
    const raw = localStorage.getItem(ONBOARDING_PROGRESS_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as OnboardingProgress;
    if (typeof parsed.step !== "number") return null;
    return {
      step: parsed.step,
      done: Array.isArray(parsed.done) ? parsed.done : [],
    };
  } catch {
    return null;
  }
}

export function saveOnboardingProgress(progress: OnboardingProgress) {
  try {
    localStorage.setItem(ONBOARDING_PROGRESS_KEY, JSON.stringify(progress));
  } catch {
    // storage unavailable (private mode) — progress just won't persist
  }
}

export function clearOnboardingProgress() {
  try {
    localStorage.removeItem(ONBOARDING_PROGRESS_KEY);
  } catch {
    // ignore
  }
}
