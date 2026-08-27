package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReverseTransactionFlow(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	actor := uuid.New()
	t0 := time.Now().UTC().Truncate(time.Second)
	bank := GroupBankName("crdb")
	savings := MemberSavingsName("m-9")

	if _, err := lg.OpenAccount(ctx, gid, actor, t0, bank, Asset, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := lg.OpenAccount(ctx, gid, actor, t0, savings, Liability, "9", 0); err != nil {
		t.Fatal(err)
	}

	baselineBank, _ := lg.GetBalance(ctx, gid, bank, nil)
	txID, err := lg.RecordTransaction(ctx, gid, actor, t0.Add(time.Minute), "amana kubwa", []Entry{
		{AccountName: bank, Direction: Debit, Amount: NewTZS(12_000)},
		{AccountName: savings, Direction: Credit, Amount: NewTZS(12_000)},
	})
	if err != nil {
		t.Fatal(err)
	}

	beforeBank, _ := lg.GetBalance(ctx, gid, bank, nil)
	beforeSav, _ := lg.GetBalance(ctx, gid, savings, nil)
	if beforeBank.AmountMinor != 12_000 {
		t.Fatalf("pre-reversal bank=%d want 12000", beforeBank.AmountMinor)
	}

	// Snapshot the original event row to prove immutability end-to-end.
	var evBefore []byte
	var recBefore time.Time
	if err := pool.QueryRow(ctx,
		`SELECT payload, recorded_at FROM ledger_events WHERE stream_id=$1 AND event_type='TransactionRecorded'`,
		txID).Scan(&evBefore, &recBefore); err != nil {
		t.Fatal(err)
	}

	revEventID, err := lg.ReverseTransaction(ctx, gid, actor, txID, t0.Add(5*time.Minute), "mwanachama alitangaza kuondoka")
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if revEventID == uuid.Nil {
		t.Fatal("reversal event id missing")
	}

	// Original untouched.
	var evAfter []byte
	var recAfter time.Time
	if err := pool.QueryRow(ctx,
		`SELECT payload, recorded_at FROM ledger_events WHERE stream_id=$1 AND event_type='TransactionRecorded'`,
		txID).Scan(&evAfter, &recAfter); err != nil {
		t.Fatal(err)
	}
	if string(evBefore) != string(evAfter) || !recBefore.Equal(recAfter) {
		t.Fatal("original transaction event must be byte-identical after reversal")
	}

	// Net effect zero: final balances equal the PRE-DEPOSIT baseline
	// (spec §7.5: net effect of original + reversal is zero).
	afterBank, _ := lg.GetBalance(ctx, gid, bank, nil)
	afterSav, _ := lg.GetBalance(ctx, gid, savings, nil)
	if afterBank.AmountMinor != baselineBank.AmountMinor || afterSav.AmountMinor != 0 {
		t.Fatalf("net effect must be zero: bank %d->%d sav %d->%d",
			beforeBank.AmountMinor, afterBank.AmountMinor,
			beforeSav.AmountMinor, afterSav.AmountMinor)
	}

	// Statement lines show both directions per account.
	lines, _ := lg.GetLedgerEntries(ctx, gid, savings, t0, t0.Add(10*time.Minute))
	kinds := map[EventType]bool{}
	for _, ln := range lines {
		kinds[ln.EventType] = true
		if ln.EventType == EventTransactionReversed && ln.Direction != Debit {
			t.Fatalf("reversal leg on savings must be debit (mirror), got %s", ln.Direction)
		}
	}
	if !kinds[EventTransactionRecorded] || !kinds[EventTransactionReversed] {
		t.Fatalf("statement must contain both kinds, got %v", kinds)
	}

	// Trial balance stays balanced; reconciliation invariant holds.
	if _, err := lg.GetTrialBalance(ctx, gid, nil); err != nil {
		t.Fatalf("trial balance after reversal: %v", err)
	}

	// Double reversal refused.
	if _, err := lg.ReverseTransaction(ctx, gid, actor, txID, t0.Add(time.Hour), "pili"); !errors.Is(err, ErrAlreadyReversed) {
		t.Fatalf("double reversal must fail with ErrAlreadyReversed, got %v", err)
	}

	// Unknown transaction.
	if _, err := lg.ReverseTransaction(ctx, gid, actor, uuid.New(), t0, "?"); !errors.Is(err, ErrTransactionNotFound) {
		t.Fatalf("want ErrTransactionNotFound, got %v", err)
	}

	// Replay determinism must reproduce post-reversal state identically.
	pre := snapshotGroup(t, ctx, gid)
	if err := lg.RebuildProjections(ctx, &gid); err != nil {
		t.Fatal(err)
	}
	post := snapshotGroup(t, ctx, gid)
	for i := range pre {
		if i >= len(post) || pre[i] != post[i] {
			t.Fatalf("replay changed reversal outcome at %d: %s -> %v", i, pre[i], post)
		}
	}
}
