package services

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/ledger"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Auto-posting: every real-money business event is mirrored into the
// double-entry ledger automatically, so the Kitabu never drifts from the
// operational tables. Posting is best-effort — the business write has
// already committed, so a ledger failure is logged loudly but never fails
// the request. The manual Kitabu screen remains for adjustments.

var (
	autoLedger      *ledger.Ledger
	autoLedgerGroup uuid.UUID
	autoLedgerOn    bool
)

// SetAutoLedger wires the ledger core + group scope for auto-posting.
// Called once from main after ledgerInit. Nil lg disables auto-posting.
func SetAutoLedger(lg *ledger.Ledger, groupID uuid.UUID) {
	autoLedger = lg
	autoLedgerGroup = groupID
	autoLedgerOn = lg != nil
}

func autoActor(raw string) uuid.UUID {
	if id, err := uuid.Parse(raw); err == nil {
		return id
	}
	if raw == "" {
		return uuid.NameSpaceOID
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("actor|"+raw))
}

// tzsMinor converts decimal TZS to integer minor units (cents).
// Pure — covered by unit test.
func tzsMinor(d decimal.Decimal) int64 {
	return d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

// savingsAccount returns the member savings account name, refusing blank
// refs so a data bug can never create a garbage "akiba_ya_mwanachama:" account.
func savingsAccount(memberNo string) (string, error) {
	if strings.TrimSpace(memberNo) == "" {
		return "", errors.New("empty member ref")
	}
	return ledger.MemberSavingsName(strings.TrimSpace(memberNo)), nil
}

// receivableAccount is the blank-ref guard for loan receivable accounts.
func receivableAccount(memberNo string) (string, error) {
	if strings.TrimSpace(memberNo) == "" {
		return "", errors.New("empty member ref")
	}
	return ledger.LoanReceivableName(strings.TrimSpace(memberNo)), nil
}

func ensureLedgerAccount(ctx context.Context, actor uuid.UUID, name string, typ ledger.AccountType, owner string) error {
	if !autoLedgerOn {
		return errors.New("ledger auto-post not wired")
	}
	_, err := autoLedger.OpenAccount(ctx, autoLedgerGroup, actor, time.Now().UTC(), name, typ, owner, 0)
	if err != nil && errors.Is(err, ledger.ErrConcurrencyConflict) {
		return nil // already exists — the expected steady state
	}
	return err
}

func postBalanced(ctx context.Context, actor uuid.UUID, occurredAt time.Time, memo string, debit, credit string, amountMinor int64) error {
	if !autoLedgerOn {
		return errors.New("ledger auto-post not wired")
	}
	if amountMinor <= 0 {
		return errors.New("non-positive amount")
	}
	_, err := autoLedger.RecordTransaction(ctx, autoLedgerGroup, actor, occurredAt, memo, []ledger.Entry{
		{AccountName: debit, Direction: ledger.Debit, Amount: ledger.NewTZS(amountMinor)},
		{AccountName: credit, Direction: ledger.Credit, Amount: ledger.NewTZS(amountMinor)},
	})
	return err
}

// PostContribution mirrors a confirmed/paid AKIBA contribution:
// debit cash on hand, credit the member's savings account.
func PostContribution(memberNo string, amount decimal.Decimal, occurredAt time.Time, actorUserID, memo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	actor := autoActor(actorUserID)
	savings, err := savingsAccount(memberNo)
	if err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameGroupCash, ledger.Asset, ""); err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, savings, ledger.Liability, memberNo); err != nil {
		return err
	}
	return postBalanced(ctx, actor, occurredAt, memo, ledger.NameGroupCash, savings, tzsMinor(amount))
}

// PostRepayment mirrors a recorded loan repayment:
// debit cash on hand, credit the member's loan receivable.
func PostRepayment(memberNo string, amount decimal.Decimal, occurredAt time.Time, actorUserID, memo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	actor := autoActor(actorUserID)
	receivable, err := receivableAccount(memberNo)
	if err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameGroupCash, ledger.Asset, ""); err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, receivable, ledger.Asset, memberNo); err != nil {
		return err
	}
	return postBalanced(ctx, actor, occurredAt, memo, ledger.NameGroupCash, receivable, tzsMinor(amount))
}

// PostDisbursement mirrors a disbursed loan:
// debit the member's loan receivable, credit cash on hand.
func PostDisbursement(memberNo string, amount decimal.Decimal, occurredAt time.Time, actorUserID, memo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	actor := autoActor(actorUserID)
	receivable, err := receivableAccount(memberNo)
	if err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, receivable, ledger.Asset, memberNo); err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameGroupCash, ledger.Asset, ""); err != nil {
		return err
	}
	return postBalanced(ctx, actor, occurredAt, memo, receivable, ledger.NameGroupCash, tzsMinor(amount))
}

// PostWelfareIn mirrors a member's welfare-fund payment received by the group:
// debit cash on hand, credit the welfare fund holdings account.
func PostWelfareIn(memberNo string, amount decimal.Decimal, occurredAt time.Time, actorUserID, memo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	actor := autoActor(actorUserID)
	if strings.TrimSpace(memberNo) == "" {
		return errors.New("empty member ref")
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameGroupCash, ledger.Asset, ""); err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameWelfareFund, ledger.Liability, ""); err != nil {
		return err
	}
	return postBalanced(ctx, actor, occurredAt, memo, ledger.NameGroupCash, ledger.NameWelfareFund, tzsMinor(amount))
}

// PostWelfareOut mirrors a welfare payout to the beneficiary member:
// debit the welfare fund holdings, credit cash on hand.
func PostWelfareOut(memberNo string, amount decimal.Decimal, occurredAt time.Time, actorUserID, memo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	actor := autoActor(actorUserID)
	if strings.TrimSpace(memberNo) == "" {
		return errors.New("empty member ref")
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameWelfareFund, ledger.Liability, ""); err != nil {
		return err
	}
	if err := ensureLedgerAccount(ctx, actor, ledger.NameGroupCash, ledger.Asset, ""); err != nil {
		return err
	}
	return postBalanced(ctx, actor, occurredAt, memo, ledger.NameWelfareFund, ledger.NameGroupCash, tzsMinor(amount))
}

// BackfillLedgerFromHistory posts one opening-balance transaction per member
// for all PAID contributions that predate auto-posting. Guard: runs only on
// a fresh ledger (no trial-balance lines yet) so it can never double-post.
func BackfillLedgerFromHistory(actorUserID string) (int, error) {
	if !autoLedgerOn {
		return 0, errors.New("ledger auto-post not wired")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tb, err := autoLedger.GetTrialBalance(ctx, autoLedgerGroup, nil)
	if err == nil && len(tb.Lines) > 0 {
		return 0, errors.New("ledger already has activity — backfill refused")
	}

	type agg struct {
		MemberID string
		MemberNo string
		Total    string
		MaxPaid  time.Time
	}
	var rows []agg
	if err := database.DB.Raw(`
		SELECT c.member_id, m.member_no, SUM(c.amount)::text AS total, MAX(c.paid_at) AS max_paid
		  FROM contributions c JOIN members m ON m.id = c.member_id
		 WHERE c.status = 'PAID' AND m.deleted_at IS NULL
		 GROUP BY c.member_id, m.member_no`).Scan(&rows).Error; err != nil {
		return 0, err
	}

	posted := 0
	for _, r := range rows {
		total, err := decimal.NewFromString(r.Total)
		if err != nil || total.LessThanOrEqual(decimal.Zero) {
			continue
		}
		memo := "Salio la ufunguzi — michango kabla ya auto-post (" + r.MemberNo + ")"
		if err := PostContribution(r.MemberNo, total, r.MaxPaid, actorUserID, memo); err != nil {
			log.Printf("WARN: ledger backfill %s: %v", r.MemberNo, err)
			continue
		}
		posted++
	}
	return posted, nil
}
