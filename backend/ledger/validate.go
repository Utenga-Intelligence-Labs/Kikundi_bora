package ledger

import (
	"fmt"
)

// ValidateEntries enforces the double-entry invariants at the domain layer,
// BEFORE anything is appended (spec §2 invariant 2: reject, don't record):
//   - at least 2 entries
//   - every entry structurally sound (valid direction, positive amount,
//     non-empty account name)
//   - single currency across all entries (v1: TZS only)
//   - sum(debits) == sum(credits) exactly
//
// It returns ErrUnbalancedTransaction, ErrCurrencyMismatch,
// ErrUnsupportedCurrency, ErrInvalidEntry or ErrTooFewEntries respectively.
func ValidateEntries(entries []Entry) error {
	if len(entries) < 2 {
		return fmt.Errorf("%w: got %d", ErrTooFewEntries, len(entries))
	}
	currency := ""
	var debits, credits int64
	for i, e := range entries {
		if e.AccountName == "" {
			return fmt.Errorf("%w: entry %d has empty account name", ErrInvalidEntry, i)
		}
		if !e.Direction.Valid() {
			return fmt.Errorf("%w: entry %d direction %q", ErrInvalidEntry, i, e.Direction)
		}
		if err := checkCurrency(e.Amount.Currency); err != nil {
			return fmt.Errorf("entry %d: %w", i, err)
		}
		if currency == "" {
			currency = e.Amount.Currency
		} else if currency != e.Amount.Currency {
			return fmt.Errorf("%w: entry %d mixes %s with %s",
				ErrCurrencyMismatch, i, currency, e.Amount.Currency)
		}
		if e.Amount.AmountMinor <= 0 {
			return fmt.Errorf("%w: entry %d amount %d must be positive",
				ErrInvalidEntry, i, e.Amount.AmountMinor)
		}
		if e.Direction == Debit {
			debits += e.Amount.AmountMinor
		} else {
			credits += e.Amount.AmountMinor
		}
	}
	if debits != credits {
		return fmt.Errorf("%w: debits=%d credits=%d (%s)",
			ErrUnbalancedTransaction, debits, credits, currency)
	}
	return nil
}

func checkCurrency(c string) error {
	switch c {
	case CurrencyTZS:
		return nil
	case "":
		return fmt.Errorf("%w: missing currency", ErrUnsupportedCurrency)
	default:
		return fmt.Errorf("%w: %q (v1 supports only %s)", ErrUnsupportedCurrency, c, CurrencyTZS)
	}
}

// ReconciliationDelta is the signed contribution of one leg to the per-event
// reconciliation check: +amount for debits, −amount for credits (see
// Direction.Signed). Weighting by normal balance side (spec §7.2) collapses
// to exactly this, so Σ over any transaction's entries is provably zero.
func ReconciliationDelta(_ AccountType, d Direction, amountMinor int64) int64 {
	return d.Signed(amountMinor)
}
