package ledger

import (
	"math/rand"
	"strings"
	"testing"
)

func TestValidateEntriesRejectsUnbalanced(t *testing.T) {
	entries := []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(5000)},
		{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(4999)},
	}
	err := ValidateEntries(entries)
	if err == nil || !strings.Contains(err.Error(), "not balanced") {
		t.Fatalf("unbalanced transaction must be rejected, got %v", err)
	}
}

func TestValidateEntriesRejectsWrongCurrencyMix(t *testing.T) {
	usd := Money{AmountMinor: 100, Currency: "USD"}
	entries := []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: usd},
		{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(100)},
	}
	err := ValidateEntries(entries)
	if err == nil || !(strings.Contains(err.Error(), "currency mismatch") ||
		strings.Contains(err.Error(), "unsupported currency")) {
		t.Fatalf("mixed-currency transaction must be rejected, got %v", err)
	}
}

func TestValidateEntriesRejectsUnsupportedCurrency(t *testing.T) {
	entries := []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: Money{10, "KES"}},
		{AccountName: NameInterestIncome, Direction: Credit, Amount: Money{10, "KES"}},
	}
	err := ValidateEntries(entries)
	if err == nil || !strings.Contains(err.Error(), "unsupported currency") {
		t.Fatalf("non-TZS transaction must be rejected, got %v", err)
	}
}

// TestValidateEntriesStructuralRules covers per-entry structural rejection.
func TestValidateEntriesStructuralRules(t *testing.T) {
	cases := []struct {
		name    string
		entries []Entry
	}{
		{"single entry", []Entry{
			{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(1)},
		}},
		{"zero amount", []Entry{
			{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(0)},
			{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(0)},
		}},
		{"negative amount", []Entry{
			{AccountName: NameGroupCash, Direction: Debit, Amount: Money{-5, CurrencyTZS}},
			{AccountName: NameInterestIncome, Direction: Credit, Amount: Money{-5, CurrencyTZS}},
		}},
		{"empty account name", []Entry{
			{AccountName: "", Direction: Debit, Amount: NewTZS(5)},
			{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(5)},
		}},
		{"invalid direction", []Entry{
			{AccountName: NameGroupCash, Direction: "sideways", Amount: NewTZS(5)},
			{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(5)},
		}},
		{"missing currency", []Entry{
			{AccountName: NameGroupCash, Direction: Debit, Amount: Money{5, ""}},
			{AccountName: NameInterestIncome, Direction: Credit, Amount: Money{5, ""}},
		}},
		{"empty entries", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateEntries(tc.entries); err == nil {
				t.Fatalf("%s must be rejected", tc.name)
			}
		})
	}
}

// TestNormalizedDeltaSides pins the stored-balance (normal-side) deltas.
func TestNormalizedDeltaSides(t *testing.T) {
	cases := []struct {
		typ  AccountType
		dir  Direction
		want int64
	}{
		{Asset, Debit, +1}, {Asset, Credit, -1},
		{Expense, Debit, +1}, {Expense, Credit, -1},
		{Liability, Credit, +1}, {Liability, Debit, -1},
		{Income, Credit, +1}, {Income, Debit, -1},
		{Equity, Credit, +1}, {Equity, Debit, -1},
	}
	for _, c := range cases {
		got := NormalizedDelta(c.typ, c.dir, 100)
		if got != c.want*100 {
			t.Errorf("%s/%s = %d want %d", c.typ, c.dir, got, c.want*100)
		}
	}
}

// TestPropertyRandomBalancedMultiLegTransactions generates random balanced
// multi-leg transactions and verifies three things per iteration:
//  1. Σ ReconciliationDelta over entries == exactly zero (invariant §7.2),
//  2. the validator accepts the transaction,
//  3. cumulative weighted account balances stay reconciled across a whole
//     random history (simulating event application without a DB).
func TestPropertyRandomBalancedMultiLegTransactions(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	type acct struct {
		name string
		typ  AccountType
	}
	pool := []acct{
		{GroupBankName("crdb"), Asset}, {NameGroupCash, Asset}, {LoanReceivableName("m1"), Asset},
		{MemberSavingsName("m2"), Liability}, {NameFinesIncome, Income},
		{NameRetainedEarnings, Equity}, {NameLoanLossProvision, Expense},
	}
	types := make(map[string]AccountType, len(pool))
	balances := make(map[string]int64, len(pool)) // normalized stored balances
	for _, a := range pool {
		types[a.name] = a.typ
	}

	const iterations = 2000
	for i := 0; i < iterations; i++ {
		var total int64
		nD := rnd.Intn(3) + 1
		debits := make([]int64, nD)
		for d := range debits {
			debits[d] = rnd.Int63n(9_000_00) + 1
			total += debits[d]
		}
		cSplit := rnd.Int63n(total)
		pick := func() acct { return pool[rnd.Intn(len(pool))] }

		var recon int64 // invariant-1 accumulator: must end at exactly 0
		entries := make([]Entry, 0, nD+2)
		add := func(a acct, d Direction, amt int64) bool {
			if amt <= 0 {
				return false
			}
			entries = append(entries, Entry{a.name, d, NewTZS(amt)})
			recon += ReconciliationDelta(a.typ, d, amt)
			return true
		}
		for _, amt := range debits {
			add(pick(), Debit, amt)
		}
		ok1 := add(pick(), Credit, cSplit)
		ok2 := add(pick(), Credit, total-cSplit)
		if !ok1 && !ok2 {
			continue // degenerate split; skip iteration
		}

		// Invariant 1: per-event reconciliation nets to exactly zero.
		if recon != 0 {
			t.Fatalf("iter %d: reconciliation sum=%d want 0", i, recon)
		}
		// Invariant 2: validator accepts a well-formed transaction.
		if err := ValidateEntries(entries); err != nil {
			t.Fatalf("iter %d: valid transaction rejected: %v", i, err)
		}
		// Invariant 3: apply like an event would; global weighted sum stays 0.
		for _, e := range entries {
			balances[e.AccountName] += NormalizedDelta(types[e.AccountName], e.Direction, e.Amount.AmountMinor)
		}
		var weightedSum int64
		for name, bal := range balances {
			w := int64(1)
			if !types[name].DebitPositive() {
				w = -1
			}
			weightedSum += w * bal
		}
		if weightedSum != 0 {
			t.Fatalf("iter %d: cumulative weighted balances = %d want 0", i, weightedSum)
		}
	}
}
