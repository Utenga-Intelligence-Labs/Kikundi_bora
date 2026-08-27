package ledger

// Direction is one side of a double-entry leg.
type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

// Valid reports whether d is debit or credit.
func (d Direction) Valid() bool { return d == Debit || d == Credit }

// Opposite returns the other side of the entry (used by reversals).
func (d Direction) Opposite() Direction {
	if d == Debit {
		return Credit
	}
	return Debit
}

// Signed returns the contribution of a leg to the GLOBAL reconciliation
// invariant: every debit leg counts +amount and every credit leg −amount,
// INDEPENDENT of account type. Summing these over any valid transaction's
// entries yields exactly zero, which is what the property test asserts.
//
// (Proof sketch: with normal-side weights w=+1 for {asset,expense} and w=−1
// otherwise, and normalized deltas Ñ, the product w·ΔÑ collapses to +amt on
// debits and −amt on credits for every type — the signs cancel.)
func (d Direction) Signed(amountMinor int64) int64 {
	if d == Debit {
		return amountMinor
	}
	return -amountMinor
}

// NormalizedDelta returns a leg's contribution to an account's STORED
// balance, which is expressed positive-on-normal-side: debit-positive types
// (asset, expense) grow on debit; credit-positive types (liability, income,
// equity) grow on credit.
func NormalizedDelta(t AccountType, d Direction, amountMinor int64) int64 {
	if (d == Debit) == t.DebitPositive() {
		return amountMinor
	}
	return -amountMinor
}

// Entry is one leg of a transaction in the domain model.
type Entry struct {
	AccountName string    // chart-of-accounts name, e.g. akiba_ya_mwanachama:42
	Direction   Direction // debit | credit
	Amount      Money     // exact integer minor units; never float64
}
