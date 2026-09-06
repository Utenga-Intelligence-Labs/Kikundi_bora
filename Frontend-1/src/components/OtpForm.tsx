import { useState } from "react";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp";

/**
 * OTP verification form — PRESERVED but NOT ROUTED anywhere while backend
 * OTP is disabled (OTP_VERIFICATION_ENABLED=false).
 *
 * Re-enable runbook:
 *  1. Set OTP_VERIFICATION_ENABLED=true on the backend and restart it.
 *  2. Register this component on a route (e.g. /thibitisha-otp).
 *  3. In the login page, when the login response contains
 *     `otp_required: true`, navigate there with the `challengeId` and call
 *     POST /api/v1/auth/verify-otp via onVerified below.
 * Do NOT delete this file — it is the kept implementation, not dead code.
 */
export function OtpForm({
  challengeId,
  onVerified,
  onError,
}: {
  challengeId: string;
  onVerified: (token: string) => void;
  onError?: (message: string) => void;
}) {
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/verify-otp", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ challenge_id: challengeId, code }),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(body.message ?? "Msimbo si sahihi");
      }
      onVerified(body.token as string);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Imeshindikana kuthibitisha";
      setError(msg);
      onError?.(msg);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4" data-testid="otp-form">
      <p className="text-sm text-muted-foreground">
        Weka msimbo wa tarakimu 6 uliotumwa kwenye simu yako.
      </p>
      <InputOTP maxLength={6} value={code} onChange={setCode}>
        <InputOTPGroup>
          <InputOTPSlot index={0} />
          <InputOTPSlot index={1} />
          <InputOTPSlot index={2} />
          <InputOTPSlot index={3} />
          <InputOTPSlot index={4} />
          <InputOTPSlot index={5} />
        </InputOTPGroup>
      </InputOTP>
      {error && (
        <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      )}
      <button
        onClick={submit}
        disabled={busy || code.length !== 6}
        className="inline-flex w-full items-center justify-center rounded-xl bg-primary py-3 text-sm font-semibold text-primary-foreground disabled:opacity-60"
      >
        {busy ? "Inathibitisha..." : "Thibitisha"}
      </button>
    </div>
  );
}
