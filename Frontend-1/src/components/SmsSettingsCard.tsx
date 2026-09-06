import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { groupsApi } from "@/api/groups";
import {
  notificationSettingsApi,
  type NotificationSettings,
} from "@/api/notification-settings";
import { MessageSquare, Loader2 } from "lucide-react";

const TYPE_LABELS: Record<string, string> = {
  CONTRIBUTION_DUE: "Ukumbusho wa mchango",
  FINE_ISSUED: "Taarifa ya faini",
  LOAN_REQUEST: "Maombi ya mkopo",
  LOAN_APPROVED: "Mkopo umeidhinishwa",
  LOAN_DISBURSED: "Mkopo umetolewa",
  REPAYMENT: "Marejesho ya mkopo",
  CONTRIBUTION: "Mchango",
  WELFARE_PAYMENT: "Malipo ya ustawi",
  USER_CREATED: "Mtumiaji mpya",
  SYSTEM: "Mfumo",
};

/**
 * SMS channel settings (mwenyekiti/admin). Group master toggle plus
 * per-type checkboxes. Shows a notice while no real provider is wired so
 * testers are not confused about missing real SMS.
 */
export function SmsSettingsCard() {
  const qc = useQueryClient();
  const { data: gs } = useQuery({
    queryKey: ["groups", "current"],
    queryFn: () => groupsApi.current(),
  });
  const groupId = (gs?.data as { id?: string } | undefined)?.id ?? null;

  const { data, isLoading } = useQuery({
    queryKey: ["notification-settings", groupId],
    queryFn: () => notificationSettingsApi.get(groupId as string),
    enabled: !!groupId,
  });
  const settings: NotificationSettings | undefined = data?.data;

  const [pending, setPending] = useState<Record<string, boolean>>({});
  const mutation = useMutation({
    mutationFn: (update: { sms_enabled?: boolean; types?: Record<string, boolean> }) =>
      notificationSettingsApi.update(groupId as string, update),
    onSuccess: () => {
      setPending({});
      qc.invalidateQueries({ queryKey: ["notification-settings"] });
    },
  });

  if (isLoading || !settings) {
    return (
      <section className="card-surface p-4">
        <Loader2 className="h-5 w-5 animate-spin text-primary" />
      </section>
    );
  }

  const effective = (key: string, fallback: boolean) =>
    pending[key] ?? (settings.types[key] ?? fallback);
  const smsOn = pending["__master"] ?? settings.sms_enabled;
  const dirty =
    Object.keys(pending).length > 0 || mutation.isPending;

  const ordered = Object.keys(settings.types).sort();

  return (
    <section className="card-surface overflow-hidden" data-testid="sms-settings">
      <header className="flex items-center gap-2.5 border-b border-border px-4 py-3">
        <span className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
          <MessageSquare className="h-4 w-4" />
        </span>
        <div>
          <h3 className="font-display text-sm font-semibold">Arifa za SMS</h3>
          <p className="text-[11px] text-muted-foreground">
            Kichwa kikuu + aina zinazotuma SMS
          </p>
        </div>
      </header>
      <div className="space-y-3 p-4">
        <label className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5">
          <span className="text-sm font-medium">SMS zimewashwa</span>
          <input
            type="checkbox"
            aria-label="SMS zimewashwa"
            checked={smsOn}
            onChange={(e) =>
              setPending((p) => ({ ...p, __master: e.target.checked }))
            }
            className="h-4 w-4"
          />
        </label>

        {!settings.provider_real && (
          <p className="rounded-lg bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
            Mtoa huduma wa SMS bado haujaunganishwa — ujumbe utaandikwa kwenye
            kumbukumbu tu hadi mtoa huduma awekwe (SMS_PROVIDER).
          </p>
        )}

        <div className={smsOn ? "" : "opacity-50 pointer-events-none"}>
          <p className="mb-1 text-xs font-medium text-muted-foreground">
            Aina zinazotuma SMS pia
          </p>
          <div className="space-y-1.5">
            {ordered.map((key) => (
              <label
                key={key}
                className="flex items-center justify-between gap-3 rounded-lg border border-border/60 px-3 py-2"
              >
                <span className="text-sm">{TYPE_LABELS[key] ?? key}</span>
                <input
                  type="checkbox"
                  aria-label={TYPE_LABELS[key] ?? key}
                  checked={!!effective(key, false)}
                  onChange={(e) =>
                    setPending((p) => ({ ...p, [key]: e.target.checked }))
                  }
                  className="h-4 w-4"
                />
              </label>
            ))}
          </div>
        </div>

        <button
          onClick={() => {
            const types: Record<string, boolean> = {};
            for (const [k, v] of Object.entries(pending)) {
              if (k !== "__master") types[k] = v;
            }
            mutation.mutate({
              ...(pending.__master !== undefined
                ? { sms_enabled: pending.__master }
                : {}),
              ...(Object.keys(types).length > 0 ? { types } : {}),
            });
          }}
          disabled={!dirty}
          className="rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-50"
        >
          {mutation.isPending ? "Inahifadhi…" : "Hifadhi"}
        </button>
        {mutation.isError && (
          <p className="text-xs text-destructive">
            {(mutation.error as Error)?.message ?? "Imeshindikana kuhifadhi"}
          </p>
        )}
      </div>
    </section>
  );
}
