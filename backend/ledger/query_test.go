package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// queryFixture opens hazina + akiba for a member and posts two deposits at
// distinct business times. Returns group id, member savings account name,
// and the time between the two transactions.
func queryFixture(t *testing.T, ctx context.Context) (uuid.UUID, string, time.Time) {
	t.Helper()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	if _, err := lg.OpenAccount(ctx, gid, uuid.New(), now, NameGroupCash, Asset, "", 0); err != nil {
		t.Fatal(err)
	}
	savings := MemberSavingsName("m-77")
	if _, err := lg.OpenAccount(ctx, gid, uuid.New(), now.Add(time.Second), savings, Liability, "77", 0); err != nil {
		t.Fatal(err)
	}
	t1 := now.Add(2 * time.Minute)
	if _, err := lg.RecordTransaction(ctx, gid, uuid.New(), t1, "amana ya kwanza", []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(10_000)},
		{AccountName: savings, Direction: Credit, Amount: NewTZS(10_000)},
	}); err != nil {
		t.Fatal(err)
	}
	t2 := t1.Add(5 * time.Minute)
	if _, err := lg.RecordTransaction(ctx, gid, uuid.New(), t2, "amana ya pili", []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(5_500)},
		{AccountName: savings, Direction: Credit, Amount: NewTZS(5_500)},
	}); err != nil {
		t.Fatal(err)
	}
	return gid, savings, t1
}

func TestGetBalanceCurrentAndAsOf(t *testing.T) {
	ctx := context.Background()
	gid, savings, t1 := queryFixture(t, ctx)
	lg, _ := New(testPool(t))

	m, err := lg.GetBalance(ctx, gid, savings, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.AmountMinor != 15_500 || m.Currency != CurrencyTZS {
		t.Fatalf("current balance=%d want 15500", m.AmountMinor)
	}
	if m.String() != "15500" {
		t.Fatalf("String()=%q", m.String())
	}

	before := t1.Add(-time.Minute)
	m, err = lg.GetBalance(ctx, gid, savings, &before)
	if err != nil {
		t.Fatal(err)
	}
	if m.AmountMinor != 0 {
		t.Fatalf("asOf-before balance=%d want 0", m.AmountMinor)
	}
	at1 := t1.Add(time.Second)
	m, _ = lg.GetBalance(ctx, gid, savings, &at1)
	if m.AmountMinor != 10_000 {
		t.Fatalf("asOf-t1 balance=%d want 10000", m.AmountMinor)
	}

	// Cash (asset): current projection matches derived state.
	cashNow, _ := lg.GetBalance(ctx, gid, NameGroupCash, nil)
	if cashNow.AmountMinor != 15_500 {
		t.Fatalf("cash=%d want 15500", cashNow.AmountMinor)
	}

	// Cache provenance tag must be set (>0).
	seq, err := lg.GetAsOfGlobalSeq(ctx, gid, NameGroupCash)
	if err != nil || seq <= 0 {
		t.Fatalf("as_of_global_seq=%d err=%v", seq, err)
	}
}

func TestGetLedgerEntriesWindow(t *testing.T) {
	ctx := context.Background()
	gid, savings, t1 := queryFixture(t, ctx)
	lg, _ := New(testPool(t))

	lines, err := lg.GetLedgerEntries(ctx, gid, savings, t1.Add(-time.Minute), t1.Add(9*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("lines=%d want 2", len(lines))
	}
	for i, ln := range lines {
		if ln.Direction != Credit || ln.AmountMinor == 0 {
			t.Fatalf("line %d wrong leg %+v", i, ln)
		}
		wantMemo := [2]string{"amana ya kwanza", "amana ya pili"}
		if ln.Memo != wantMemo[i] {
			t.Fatalf("line %d memo=%q want %q", i, ln.Memo, wantMemo[i])
		}
		if lines[0].OccurredAt.After(lines[1].OccurredAt) {
			t.Fatal("statement lines must be chronological")
		}
	}

	// Empty window.
	none, err := lg.GetLedgerEntries(ctx, gid, savings, t1.Add(-3*time.Hour), t1.Add(-2*time.Hour))
	if err != nil || len(none) != 0 {
		t.Fatalf("empty window: n=%d err=%v", len(none), err)
	}
}

func TestGetTrialBalanceNetsZero(t *testing.T) {
	ctx := context.Background()
	gid, _, _ := queryFixture(t, ctx)
	lg, _ := New(testPool(t))

	tb, err := lg.GetTrialBalance(ctx, gid, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !tb.Balanced || tb.TotalDebitMinor != tb.TotalCreditMinor || tb.TotalDebitMinor != 15_500 {
		t.Fatalf("trial balance off: d=%d c=%d bal=%v", tb.TotalDebitMinor, tb.TotalCreditMinor, tb.Balanced)
	}
	if len(tb.Lines) < 2 {
		t.Fatalf("expected >=2 accounts in trial balance, got %d", len(tb.Lines))
	}

	past := time.Now().Add(-24 * time.Hour)
	tb, err = lg.GetTrialBalance(ctx, gid, &past)
	if err == nil {
		t.Fatal("empty history trial balance should not claim balanced")
	}
}
