package ledger

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOpenAccountAndRecordTransactionEndToEnd(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, err := New(pool)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	actor := uuid.New()

	cashID, err := lg.OpenAccount(ctx, gid, actor, now, NameGroupCash, Asset, "", 0)
	if err != nil {
		t.Fatalf("open cash: %v", err)
	}
	savingsName := MemberSavingsName("m-42")
	if _, err := lg.OpenAccount(ctx, gid, actor, now, savingsName, Liability, "42", 0); err != nil {
		t.Fatalf("open savings: %v", err)
	}

	// Duplicate name must be refused via OCC guard on the deterministic stream.
	if _, err := lg.OpenAccount(ctx, gid, actor, now, NameGroupCash, Asset, "", 0); !IsConcurrencyConflict(err) {
		t.Fatalf("duplicate open must conflict, got %v", err)
	}

	txID, err := lg.RecordTransaction(ctx, gid, actor, now.Add(time.Minute),
		"amanisha ya mwanachama 42", []Entry{
			{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(25_000)},
			{AccountName: savingsName, Direction: Credit, Amount: NewTZS(25_000)},
		})
	if err != nil {
		t.Fatalf("record deposit: %v", err)
	}
	if txID == uuid.Nil {
		t.Fatal("transaction id must not be nil")
	}

	// Projections: hazina (asset) +25000; akiba (liability) +25000 normalized.
	var cashBal, savBal int64
	if err := pool.QueryRow(ctx,
		`SELECT balance_minor FROM ledger_account_balances WHERE account_name=$1`, NameGroupCash,
	).Scan(&cashBal); err != nil {
		t.Fatal(err)
	}
	if cashBal != 25_000 {
		t.Fatalf("cash balance=%d want 25000", cashBal)
	}
	if err := pool.QueryRow(ctx,
		`SELECT balance_minor FROM ledger_account_balances WHERE account_name=$1`, savingsName,
	).Scan(&savBal); err != nil {
		t.Fatal(err)
	}
	if savBal != 25_000 { // liability grows positive on credit
		t.Fatalf("savings balance=%d want 25000", savBal)
	}

	// Trial balance checkpoint for this group must net to zero.
	var net int64
	if err := pool.QueryRow(ctx,
		`SELECT net_minor FROM ledger_trial_balance WHERE group_id=$1 ORDER BY as_of_global_seq DESC LIMIT 1`, gid,
	).Scan(&net); err != nil {
		t.Fatal(err)
	}
	if net != 0 {
		t.Fatalf("trial balance net=%d want 0", net)
	}

	// Reconciliation invariant across weighted projected balances.
	var wsum int64
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(b.balance_minor * CASE WHEN a.type IN ('asset','expense') THEN 1 ELSE -1 END),0)
		  FROM ledger_account_balances b JOIN ledger_accounts a USING (account_name)
		 WHERE a.group_id=$1`, gid).Scan(&wsum); err != nil {
		t.Fatal(err)
	}
	if wsum != 0 {
		t.Fatalf("weighted balances sum=%d want 0", wsum)
	}
	_ = cashID
}

func TestRecordTransactionValidationFailures(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	now := time.Now().UTC()
	actor := uuid.New()

	countEvents := func() int64 {
		var n int64
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM ledger_events WHERE group_id=$1`, gid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	openTxs := func() int64 { return countEvents() }

	// Unbalanced -> rejected with NOTHING appended.
	before := openTxs()
	_, err := lg.RecordTransaction(ctx, gid, actor, now, "mbaya", []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(1000)},
		{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(999)},
	})
	if !errors.Is(err, ErrUnbalancedTransaction) {
		t.Fatalf("want ErrUnbalancedTransaction, got %v", err)
	}
	if after := openTxs(); after != before {
		t.Fatalf("rejected transaction wrote events: %d -> %d", before, after)
	}

	// Nonexistent account -> ErrAccountNotFound.
	_, err = lg.RecordTransaction(ctx, gid, actor, now, "haitakai", []Entry{
		{AccountName: LoanReceivableName("ghost"), Direction: Debit, Amount: NewTZS(10)},
		{AccountName: NameGroupCash, Direction: Credit, Amount: NewTZS(10)},
	})
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("want ErrAccountNotFound, got %v", err)
	}

	// Single entry -> structurally invalid.
	_, err = lg.RecordTransaction(ctx, gid, actor, now, "moja", []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(5)},
	})
	if !errors.Is(err, ErrTooFewEntries) {
		t.Fatalf("want ErrTooFewEntries, got %v", err)
	}
	if after := openTxs(); after != before {
		t.Fatalf("failed validations must append nothing: %d -> %d", before, after)
	}
}

func TestClosedAccountRejectsEntries(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	now := time.Now().UTC()

	acc, err := lg.OpenAccount(ctx, gid, uuid.New(), now, GroupBankName("tmp"), Asset, "", 0)
	if err != nil {
		t.Fatal(err)
	}

	// Closing is not yet a public command (stage 5) — simulate by direct event
	// application of an AccountClosed envelope through the projector path.
	payload := fmt.Sprintf(`{"account_name":%q,"reason":"test"}`, GroupBankName("tmp"))
	env := Envelope{
		EventID: uuid.New(), GroupID: gid, StreamID: acc, SequenceNo: 2,
		GlobalSeq: 9_999, EventType: EventAccountClosed, EventVersion: 1,
		Payload: []byte(payload), ActorID: uuid.New(),
		OccurredAt: now, RecordedAt: now,
	}
	if err := applyToProjections(ctx, pool, env); err != nil {
		t.Fatalf("project close: %v", err)
	}

	lg2, _ := New(pool)
	_, err = lg2.RecordTransaction(ctx, gid, uuid.New(), now, "kwa akaunti iliyofungwa", []Entry{
		{AccountName: GroupBankName("tmp"), Direction: Debit, Amount: NewTZS(50)},
		{AccountName: NameGroupCash, Direction: Credit, Amount: NewTZS(50)},
	})
	if !errors.Is(err, ErrAccountClosed) {
		t.Fatalf("entries against closed account must fail, got %v", err)
	}
}

func TestCrossGroupAccountsRejected(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gidA := testGroup(t, ctx, pool)
	gidB := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	now := time.Now().UTC()

	if _, err := lg.OpenAccount(ctx, gidA, uuid.New(), now, NameGroupCash, Asset, "", 0); err != nil {
		t.Fatal(err)
	}
	// gidB tries to use gidA's account.
	_, err := lg.RecordTransaction(ctx, gidB, uuid.New(), now, "mipaka ya kikundi", []Entry{
		{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(70)},
		{AccountName: NameInterestIncome, Direction: Credit, Amount: NewTZS(70)},
	})
	if !errors.Is(err, ErrAccountNotFound) && !errors.Is(err, ErrGroupMismatch) {
		t.Fatalf("cross-group entry must fail, got %v", err)
	}
}
