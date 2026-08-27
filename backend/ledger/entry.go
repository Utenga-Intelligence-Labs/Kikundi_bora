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

// Signed returns +amount for a debit-positive account type on debit,
// etc., per the normal-balance-side weighting used for reconciliation.
func (d Direction) Signed(t AccountType, amount int64) int64 {
	positive := amount
	if d == Credit {
		positive = -amount
	}
	if t.DebitPositive() {
		return positive
	}
	return -positive
}

// Entry is one leg of a transaction in the domain model.
type Entry struct {
	AccountName string    // chart-of-accounts name, e.g. akiba_ya_mwanachama:42
	Direction   Direction // debit | credit
	Amount      Money     // exact integer minor units; never float64
}
