package ledger

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Golden-file / audit regression guard (spec §7 requirement 6):
// a fixed event-log fixture must always export the same trial balance and
// member statement. Regenerate intentionally ONLY via LEDGER_UPDATE_GOLDEN=1,
// review the diff like code.
const goldenPath = "testdata/golden_audit.txt"

func TestGoldenAuditExport(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	gid := testGroup(t, ctx, pool)
	lg, _ := New(pool)
	actor, _ := uuid.Parse("00000000-0000-0000-0000-000000000001")

	base := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	bank := GroupBankName("nmb-audit")
	sav := MemberSavingsName("m-5")
	fines := NameFinesIncome
	for _, oa := range []struct {
		name string
		typ  AccountType
	}{
		{bank, Asset}, {sav, Liability}, {fines, Income}, {NameGroupCash, Asset},
	} {
		if _, err := lg.OpenAccount(ctx, gid, actor, base, oa.name, oa.typ, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	post := func(at time.Time, memo string, es []Entry) uuid.UUID {
		id, err := lg.RecordTransaction(ctx, gid, actor, at, memo, es)
		if err != nil {
			t.Fatalf("post %s: %v", memo, err)
		}
		return id
	}
	t2 := post(base.Add(24*time.Hour), "amana ya mwanachama", []Entry{
		{bank, Debit, NewTZS(30_000)},
		{sav, Credit, NewTZS(30_000)},
	})
	post(base.Add(48*time.Hour), "faini ya uchelewaji", []Entry{
		{NameGroupCash, Debit, NewTZS(1_500)},
		{fines, Credit, NewTZS(1_500)},
	})
	post(base.Add(72*time.Hour), "kutuma hazina benki", []Entry{
		{bank, Debit, NewTZS(800)},
		{NameGroupCash, Credit, NewTZS(800)},
	})
	if _, err := lg.ReverseTransaction(ctx, gid, actor, t2, base.Add(96*time.Hour), "kosa la kurekodi"); err != nil {
		t.Fatal(err)
	}
	post(base.Add(120*time.Hour), "amana sahihi", []Entry{
		{bank, Debit, NewTZS(28_000)},
		{sav, Credit, NewTZS(28_000)},
	})

	// Deterministic export — every rendered value comes from committed state.
	var b strings.Builder
	fmt.Fprintf(&b, "# Kikundibora ledger audit export\n")
	fmt.Fprintf(&b, "# currency: TZS (minor units == shillings)\n")
	fmt.Fprintf(&b, "# fixture: open %s/%s/%s; deposit 30000; fine 1500; transfer 800;\n", bank, sav, fines)
	fmt.Fprintf(&b, "#          reverse deposit; correct redeposit 28000 at fixed UTC times\n\n")

	tb, err := lg.GetTrialBalance(ctx, gid, nil)
	if err != nil {
		t.Fatalf("trial balance: %v", err)
	}
	b.WriteString("== TRIAL BALANCE ==\n")
	for _, ln := range tb.Lines {
		fmt.Fprintf(&b, "%-28s %-9s debit=%-8d credit=%d\n", ln.AccountName, ln.Type, ln.DebitMinor, ln.CreditMinor)
	}
	fmt.Fprintf(&b, "totals debit=%d credit=%d balanced=%v\n\n",
		tb.TotalDebitMinor, tb.TotalCreditMinor, tb.Balanced)

	from := base.Add(-time.Hour)
	to := base.Add(200 * time.Hour)
	lines, err := lg.GetLedgerEntries(ctx, gid, sav, from, to)
	if err != nil {
		t.Fatal(err)
	}
	b.WriteString(fmt.Sprintf("== STATEMENT %s [%s .. %s] ==\n", sav, from.Format(time.RFC3339), to.Format(time.RFC3339)))
	for _, ln := range lines {
		fmt.Fprintf(&b, "%s %-21s %-7s %-7d memo=%q\n",
			ln.OccurredAt.UTC().Format(time.RFC3339), ln.EventType, ln.Direction, ln.AmountMinor, ln.Memo)
	}
	finalBalance, err := lg.GetBalance(ctx, gid, sav, nil)
	if err != nil || finalBalance.AmountMinor != 28_000 {
		t.Fatalf("final savings balance=%v err=%v want 28000", finalBalance, err)
	}

	got := b.String()

	if os.Getenv("LEDGER_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file unreadable (%v); regenerate with LEDGER_UPDATE_GOLDEN=1 go test ./ledger/ -run TestGoldenAuditExport", err)
	}
	if got != string(wantBytes) {
		diffIdx := 0
		wb := string(wantBytes)
		for diffIdx < len(wb) && diffIdx < len(got) && wb[diffIdx] == got[diffIdx] {
			diffIdx++
		}
		t.Fatalf("golden export mismatch at byte %d:\n--- got ---\n%s\n--- want ---\n%s", diffIdx, got, wb)
	}
}
