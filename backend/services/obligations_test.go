package services

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal {
	v, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Spec: member with 2 unpaid past cycles + current cycle + 1 unpaid fine →
// allocation order arrears → current → fines.
func obligationFixture() []OwedItem {
	return []OwedItem{
		{Kind: AllocArrears, Ref: "2026-06", Label: "Mchango 2026-06", Owed: d("10000")},
		{Kind: AllocArrears, Ref: "2026-07", Label: "Mchango 2026-07", Owed: d("10000")},
		{Kind: AllocCurrent, Ref: "2026-08", Label: "Mchango 2026-08", Owed: d("10000")},
		{Kind: AllocFine, Ref: "fine-1", Label: "Faini: Kuchelewa", Owed: d("5000")},
	}
}

func TestAllocatePaymentFull(t *testing.T) {
	lines, rem := AllocatePayment(obligationFixture(), d("35000"))
	if !rem.IsZero() {
		t.Fatalf("remainder = %s, want 0", rem)
	}
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4", len(lines))
	}
	wantKinds := []string{AllocArrears, AllocArrears, AllocCurrent, AllocFine}
	for i, w := range wantKinds {
		if lines[i].Kind != w {
			t.Errorf("line %d kind = %s, want %s", i, lines[i].Kind, w)
		}
	}
}

func TestAllocatePaymentPartialCoversArrearsOnly(t *testing.T) {
	// 15000 covers first arrears fully + half of second; current + fine untouched.
	lines, rem := AllocatePayment(obligationFixture(), d("15000"))
	if !rem.IsZero() {
		t.Fatalf("remainder = %s, want 0", rem)
	}
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if !lines[0].Amount.Equal(d("10000")) || lines[0].Ref != "2026-06" {
		t.Errorf("line0 = %+v, want full 2026-06", lines[0])
	}
	if !lines[1].Amount.Equal(d("5000")) || lines[1].Ref != "2026-07" {
		t.Errorf("line1 = %+v, want partial 2026-07", lines[1])
	}
}

func TestAllocatePaymentSkipsToFines(t *testing.T) {
	// Exact arrears+current (30000) then 2000 of the 5000 fine.
	lines, rem := AllocatePayment(obligationFixture(), d("32000"))
	if !rem.IsZero() {
		t.Fatalf("remainder = %s", rem)
	}
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4", len(lines))
	}
	if lines[3].Kind != AllocFine || !lines[3].Amount.Equal(d("2000")) {
		t.Errorf("fine line = %+v, want partial 2000", lines[3])
	}
}

func TestAllocatePaymentOverpayReturnsRemainder(t *testing.T) {
	lines, rem := AllocatePayment(obligationFixture(), d("50000"))
	if !rem.Equal(d("15000")) {
		t.Errorf("remainder = %s, want 15000", rem)
	}
	if len(lines) != 4 {
		t.Errorf("lines = %d, want 4", len(lines))
	}
}

func TestAllocatePaymentZeroAndSkipsZeroOwed(t *testing.T) {
	items := []OwedItem{
		{Kind: AllocArrears, Ref: "x", Owed: d("0")},
		{Kind: AllocCurrent, Ref: "y", Owed: d("100")},
	}
	lines, rem := AllocatePayment(items, d("0"))
	if len(lines) != 0 || !rem.IsZero() {
		t.Errorf("zero payment should apply nothing, got %+v rem %s", lines, rem)
	}
	lines, _ = AllocatePayment(items, d("50"))
	if len(lines) != 1 || lines[0].Ref != "y" {
		t.Errorf("should skip zero-owed item, got %+v", lines)
	}
}
