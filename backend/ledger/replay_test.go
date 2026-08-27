package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// snapshotGroup dumps the full projected state of a group for comparison.
func snapshotGroup(t *testing.T, ctx context.Context, gid uuid.UUID) []string {
	t.Helper()
	rows, err := poolQuerier(testPool(t)).Query(ctx, `
		SELECT b.account_name, b.balance_minor
		  FROM ledger_account_balances b WHERE b.group_id=$1 ORDER BY b.account_name`, gid)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		var bal int64
		if err := rows.Scan(&name, &bal); err != nil {
			t.Fatal(err)
		}
		out = append(out, name+"="+itoa(bal))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [21]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TestReplayDeterminism records a random-ish history of accounts and
// transactions, snapshots balances, wipes + replays projections, and asserts
// identical state — spec §7 requirement 3. Also re-checks the trial balance.
func TestReplayDeterminism(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	now := time.Now().UTC().Truncate(time.Second)
	actor := uuid.New()

	open := func(name string, typ AccountType) {
		if _, err := lg.OpenAccount(ctx, gid, actor, now, name, typ, "", 0); err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
	}
	bank := GroupBankName("nmb")
	sav1, sav2 := MemberSavingsName("m-1"), MemberSavingsName("m-2")
	fines := NameFinesIncome
	open(NameGroupCash, Asset)
	open(bank, Asset)
	open(sav1, Liability)
	open(sav2, Liability)
	open(fines, Income)

	post := func(at time.Time, memo string, entries []Entry) {
		if _, err := lg.RecordTransaction(ctx, gid, actor, at, memo, entries); err != nil {
			t.Fatalf("post %s: %v", memo, err)
		}
	}
	for i := 0; i < 7; i++ {
		at := now.Add(time.Duration(i) * time.Minute)
		switch i % 3 {
		case 0: // member deposits 10k/4k
			post(at, "amana wiki", []Entry{
				{AccountName: bank, Direction: Debit, Amount: NewTZS(14_000)},
				{AccountName: sav1, Direction: Credit, Amount: NewTZS(10_000)},
				{AccountName: sav2, Direction: Credit, Amount: NewTZS(4_000)},
			})
		case 1: // fines collected in cash
			post(at, "faini uchelewaji", []Entry{
				{AccountName: NameGroupCash, Direction: Debit, Amount: NewTZS(2_500)},
				{AccountName: fines, Direction: Credit, Amount: NewTZS(2_500)},
			})
		case 2: // cash moved to bank
			post(at, "kutuma hazina benki", []Entry{
				{AccountName: bank, Direction: Debit, Amount: NewTZS(1_800)},
				{AccountName: NameGroupCash, Direction: Credit, Amount: NewTZS(1_800)},
			})
		}
	}

	before := snapshotGroup(t, ctx, gid)
	if len(before) == 0 {
		t.Fatal("fixture produced no balances")
	}

	if err := lg.RebuildProjections(ctx, &gid); err != nil {
		t.Fatalf("replay group: %v", err)
	}
	after := snapshotGroup(t, ctx, gid)
	if len(before) != len(after) {
		t.Fatalf("replay changed row count: %d -> %d\nbefore=%v\nafter=%v",
			len(before), len(after), before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("replay mismatch at %d: %s -> %s", i, before[i], after[i])
		}
	}

	if _, err := lg.GetTrialBalance(ctx, gid, nil); err != nil {
		t.Fatalf("trial balance after replay: %v", err)
	}

	// Full-store rebuild must work too and leave this group untouched.
	before = snapshotGroup(t, ctx, gid)
	if err := lg.RebuildProjections(ctx, nil); err != nil {
		t.Fatalf("full replay: %v", err)
	}
	after = snapshotGroup(t, ctx, gid)
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("full-replay mismatch at %d: %s -> %s", i, before[i], after[i])
		}
	}
}
