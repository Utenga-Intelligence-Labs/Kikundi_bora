package services

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTzsMinor(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"10000", 1000000},
		{"10000.00", 1000000},
		{"2020000", 202000000},
		{"5000.50", 500050},
		{"0.01", 1},
		{"0", 0},
	}
	for _, tc := range cases {
		d, err := decimal.NewFromString(tc.in)
		if err != nil {
			t.Fatalf("bad fixture %q: %v", tc.in, err)
		}
		if got := tzsMinor(d); got != tc.want {
			t.Errorf("tzsMinor(%s) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestLedgerAccountGuards(t *testing.T) {
	if _, err := savingsAccount(""); err == nil {
		t.Error("savingsAccount(\"\") should fail")
	}
	if _, err := savingsAccount("   "); err == nil {
		t.Error("savingsAccount(whitespace) should fail")
	}
	if _, err := receivableAccount(""); err == nil {
		t.Error("receivableAccount(\"\") should fail")
	}
	name, err := savingsAccount("KKK-0001")
	if err != nil || name != "akiba_ya_mwanachama:KKK-0001" {
		t.Errorf("savingsAccount(KKK-0001) = %q, %v", name, err)
	}
	name, err = receivableAccount("KKK-0001")
	if err != nil || name != "dai_la_mkopo:KKK-0001" {
		t.Errorf("receivableAccount(KKK-0001) = %q, %v", name, err)
	}
}
