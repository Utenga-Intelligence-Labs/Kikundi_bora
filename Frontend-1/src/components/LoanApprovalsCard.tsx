import { Link } from "@tanstack/react-router";
import { ShieldCheck, ChevronRight, Clock } from "lucide-react";
import { useAuth } from "@/lib/auth-provider";
import { usePendingApprovalLoans } from "@/hooks/use-loans";
import { useIsCommitteeMember } from "@/hooks/use-loan-committee";

const STAGE_LABEL: Record<string, string> = {
  hazina: "Anasubiri Hazina",
  katibu: "Anasubiri Katibu",
  bodi: "Anasubiri Bodi",
  mwenyekiti: "Anasubiri Mwenyekiti",
};

/**
 * BUG-2 fix: "Mikopo Inayosubiri Idhini" — role-scoped loan queue section for
 * the dashboard. Shows the loans currently awaiting the viewer's stage in the
 * sequential chain (Hazina → Katibu → Bodi → Mwenyekiti). Visible to all three
 * leadership roles AND appointed bodi members; hidden for everyone else.
 */
export function LoanApprovalsCard() {
  const { user } = useAuth();
  const { data: committeeCheck } = useIsCommitteeMember();
  const isBodi =
    user?.role === "member" && !!committeeCheck?.is_committee_member;
  const inChain =
    (!!user && ["chair", "secretary", "treasurer"].includes(user.role)) ||
    isBodi;

  const { data, isLoading } = usePendingApprovalLoans(true, !!inChain);

  if (!inChain) return null;

  const mine = data?.data ?? [];

  return (
    <div
      className="card-surface p-5 space-y-3"
      data-testid="loan-approvals-card"
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-primary" />
          <div>
            <h3 className="font-display text-base font-semibold">
              Mikopo Inayosubiri Idhini
            </h3>
            <p className="text-xs text-muted-foreground">
              Ombi zilizofika hatua yako kwenye mfuatano: Hazina → Katibu → Bodi
              → Mwenyekiti
            </p>
          </div>
        </div>
        <span
          className={`chip rounded-full px-2.5 py-1 text-xs font-bold ${
            mine.length > 0
              ? "bg-amber-100 text-amber-800"
              : "bg-muted text-muted-foreground"
          }`}
          data-testid="loan-approvals-count"
        >
          {isLoading ? "…" : mine.length}
        </span>
      </div>

      {!isLoading && mine.length === 0 ? (
        <p className="rounded-lg border border-dashed p-3 text-center text-sm text-muted-foreground">
          Hakuna mkopo unasubiri hatua yako kwa sasa.
        </p>
      ) : (
        <div className="space-y-2">
          {mine.slice(0, 3).map((loan) => (
            <Link
              key={loan.id}
              to="/ukaguzi-mkopo/$loanId"
              params={{ loanId: loan.id }}
              className="flex items-center justify-between gap-3 rounded-lg border border-amber-300 bg-amber-50 dark:bg-amber-950/30 p-3 hover:bg-amber-100 dark:hover:bg-amber-950/60"
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">
                  {loan.member?.full_name ?? "Mwanachama"} ·{" "}
                  {Number(loan.amount).toLocaleString()} TZS
                </p>
                <p className="flex items-center gap-1 text-xs text-muted-foreground">
                  <Clock className="h-3 w-3" />{" "}
                  {STAGE_LABEL[loan.awaiting_role] ?? "Inasubiri"}
                </p>
              </div>
              <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
            </Link>
          ))}
        </div>
      )}

      <Link
        to="/uongozi/mikopo"
        className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline"
      >
        Fungua mgawanyo wote <ChevronRight className="h-3.5 w-3.5" />
      </Link>
    </div>
  );
}
