package services

import (
	"testing"
	"time"

	"kikundibora/models"

	"github.com/shopspring/decimal"
)

func fixedPtr(s string) *decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return &d
}

// ---------- Fixed contribution amount enforcement ----------

func TestCheckFixedContributionAmount(t *testing.T) {
	fixed := fixedPtr("10000")

	// No fixed amount configured → any amount accepted
	if err := CheckFixedContributionAmount(nil, decimal.NewFromInt(123)); err != nil {
		t.Errorf("nil fixed should not enforce, got %v", err)
	}

	// Exact match accepted
	if err := CheckFixedContributionAmount(fixed, decimal.RequireFromString("10000")); err != nil {
		t.Errorf("exact amount should pass, got %v", err)
	}
	// 10000.00 == 10000 (decimal equality, not string equality)
	if err := CheckFixedContributionAmount(fixed, decimal.RequireFromString("10000.00")); err != nil {
		t.Errorf("10000.00 should equal 10000, got %v", err)
	}

	// Over → rejected with the expected amount in the message
	err := CheckFixedContributionAmount(fixed, decimal.RequireFromString("15000"))
	if err == nil {
		t.Fatal("over amount should be rejected")
	}
	if want := "TZS 10000.00"; !contains(err.Error(), want) {
		t.Errorf("error should mention expected amount %q, got %q", want, err.Error())
	}

	// Under → rejected
	if err := CheckFixedContributionAmount(fixed, decimal.RequireFromString("5000")); err == nil {
		t.Error("under amount should be rejected")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ---------- Proposal spec validation ----------

func TestValidateProposalSpec(t *testing.T) {
	cases := []struct {
		interval, due string
		amount        *decimal.Decimal
		wantErr       bool
	}{
		{"monthly", "5", nil, false},
		{"weekly", "3", nil, false},
		{"yearly", "03-15", nil, false},
		{"semi_annual", "12-01", fixedPtr("5000"), false},
		{"monthly", "32", nil, true},          // bad day
		{"weekly", "8", nil, true},            // bad weekday
		{"yearly", "13-01", nil, true},        // bad month
		{"yearly", "15-01", nil, true},        // bad format semantics (month>12)
		{"daily", "5", nil, true},             // unsupported interval
		{"monthly", "", nil, true},            // missing due date
		{"monthly", "5", fixedPtr("0"), true}, // non-positive amount
		{"monthly", "5", fixedPtr("-1"), true},
	}
	for _, tc := range cases {
		err := ValidateProposalSpec(tc.interval, tc.due, tc.amount)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateProposalSpec(%q,%q,%v) error=%v, wantErr=%v", tc.interval, tc.due, tc.amount, err, tc.wantErr)
		}
	}
}

// ---------- Next due date computation ----------

func TestNextContributionDueDate(t *testing.T) {
	loc := time.UTC

	// Monthly: from Aug 28 2026, day 5 → Sep 5 2026
	d, ok := NextContributionDueDate(models.IntervalMonthly, "5", time.Date(2026, 8, 28, 10, 0, 0, 0, loc))
	if !ok || d.Format("2006-01-02") != "2026-09-05" {
		t.Errorf("monthly next due = %v (%v), want 2026-09-05", d, ok)
	}

	// Monthly: from Aug 3, day 5 → Aug 5 (same month)
	d, _ = NextContributionDueDate(models.IntervalMonthly, "5", time.Date(2026, 8, 3, 10, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2026-08-05" {
		t.Errorf("monthly same-month due = %v, want 2026-08-05", d)
	}

	// Monthly: day 31 clamps in short months (from Sep 1 → Sep 30)
	d, _ = NextContributionDueDate(models.IntervalMonthly, "31", time.Date(2026, 9, 1, 10, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2026-09-30" {
		t.Errorf("monthly clamped due = %v, want 2026-09-30", d)
	}

	// Weekly: from Saturday (2026-08-29), due weekday Monday (1) → Aug 31
	d, _ = NextContributionDueDate(models.IntervalWeekly, "1", time.Date(2026, 8, 29, 10, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2026-08-31" {
		t.Errorf("weekly next due = %v, want 2026-08-31", d)
	}

	// Weekly: due weekday = today → today
	d, _ = NextContributionDueDate(models.IntervalWeekly, "7", time.Date(2026, 8, 30, 10, 0, 0, 0, loc)) // Sunday
	if d.Format("2006-01-02") != "2026-08-30" {
		t.Errorf("weekly same-day due = %v, want 2026-08-30", d)
	}

	// Yearly: MM-DD later this year
	d, _ = NextContributionDueDate(models.IntervalYearly, "12-25", time.Date(2026, 8, 28, 10, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2026-12-25" {
		t.Errorf("yearly due = %v, want 2026-12-25", d)
	}

	// Yearly: MM-DD already passed → next year
	d, _ = NextContributionDueDate(models.IntervalYearly, "01-15", time.Date(2026, 8, 28, 10, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2027-01-15" {
		t.Errorf("yearly next-year due = %v, want 2027-01-15", d)
	}

	// Semi-annual: passed date → +6 months
	d, _ = NextContributionDueDate(models.IntervalSemiAnnual, "01-15", time.Date(2026, 8, 28, 10, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2027-01-15" {
		t.Errorf("semi-annual due = %v, want 2027-01-15", d)
	}

	// Invalid spec → not ok
	if _, ok := NextContributionDueDate(models.IntervalMonthly, "", time.Now()); ok {
		t.Error("empty spec should not be ok")
	}
}

// ---------- Notification idempotency (no DB required) ----------

func TestContributionDueStatusIdempotency(t *testing.T) {
	// Monthly due on day 5. Reminder = Sep 3, due = Sep 5 2026.
	g := &models.Group{
		ContributionInterval: models.IntervalMonthly,
		ContributionDueDate:  strPtr("5"),
	}

	sep3 := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)
	sep4 := time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC)
	sep5 := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)

	// Sep 3 = reminder day → fires once, then suppressed
	kind, due, ok := ContributionDueStatus(g, sep3)
	if !ok || kind != "reminder" || due.Format("2006-01-02") != "2026-09-05" {
		t.Fatalf("Sep 3 should be reminder for due 2026-09-05, got kind=%q ok=%v due=%v", kind, ok, due)
	}
	g.LastReminderNotifiedFor = &[]time.Time{sep3}[0]
	if _, _, ok := ContributionDueStatus(g, sep3); ok {
		t.Error("reminder must not fire twice on the same day (idempotency)")
	}

	// Sep 4 = neither reminder (1 day before is out of window) nor due
	if _, _, ok := ContributionDueStatus(g, sep4); ok {
		t.Error("Sep 4 should not trigger any notification")
	}

	// Sep 5 = due day → fires once, then suppressed
	kind, _, ok = ContributionDueStatus(g, sep5)
	if !ok || kind != "due" {
		t.Fatalf("Sep 5 should be due day, got kind=%q ok=%v", kind, ok)
	}
	g.LastDueNotifiedFor = &[]time.Time{sep5}[0]
	if _, _, ok := ContributionDueStatus(g, sep5); ok {
		t.Error("due notification must not fire twice (idempotency)")
	}

	// Next cycle (Oct) → reminder fires again (Oct 3) even after previous cycle
	oct3 := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	kind, due, ok = ContributionDueStatus(g, oct3)
	if !ok || kind != "reminder" || due.Format("2006-01-02") != "2026-10-05" {
		t.Errorf("next cycle should remind again, got kind=%q ok=%v due=%v", kind, ok, due)
	}
}

func TestContributionDueStatusDisabled(t *testing.T) {
	// No due date configured → never notifies
	g := &models.Group{ContributionInterval: models.IntervalMonthly}
	if _, _, ok := ContributionDueStatus(g, time.Now()); ok {
		t.Error("group without due date should never notify")
	}
	// nil group safety
	if _, _, ok := ContributionDueStatus(nil, time.Now()); ok {
		t.Error("nil group should never notify")
	}
}

// ---------- Cycle window / previous due date ----------

func TestPreviousContributionDueDate(t *testing.T) {
	loc := time.UTC

	// Monthly day 5: previous due before Sep 5 = Aug 5
	d, ok := PreviousContributionDueDate(models.IntervalMonthly, "5",
		time.Date(2026, 9, 5, 9, 0, 0, 0, loc))
	if !ok || d.Format("2006-01-02") != "2026-08-05" {
		t.Errorf("monthly prev due = %v (%v), want 2026-08-05", d, ok)
	}

	// Monthly day 31: previous before Feb 28 (clamped) = Jan 31
	d, _ = PreviousContributionDueDate(models.IntervalMonthly, "31",
		time.Date(2026, 2, 28, 9, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2026-01-31" {
		t.Errorf("monthly clamped prev due = %v, want 2026-01-31", d)
	}

	// Weekly: previous Monday before Mon Aug 31 = Mon Aug 24
	d, _ = PreviousContributionDueDate(models.IntervalWeekly, "1",
		time.Date(2026, 8, 31, 9, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2026-08-24" {
		t.Errorf("weekly prev due = %v, want 2026-08-24", d)
	}

	// Yearly 03-15: previous before Mar 15 2026 = Mar 15 2025
	d, _ = PreviousContributionDueDate(models.IntervalYearly, "03-15",
		time.Date(2026, 3, 15, 9, 0, 0, 0, loc))
	if d.Format("2006-01-02") != "2025-03-15" {
		t.Errorf("yearly prev due = %v, want 2025-03-15", d)
	}
}

func TestContributionCycleWindow(t *testing.T) {
	// Monthly day 5, today Sep 3 2026 → open cycle window (Aug 5, Sep 5]
	g := &models.Group{
		ContributionInterval: models.IntervalMonthly,
		ContributionDueDate:  strPtr("5"),
	}
	start, end, ok := ContributionCycleWindow(g, time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC))
	if !ok || start.Format("2006-01-02") != "2026-08-05" || end.Format("2006-01-02") != "2026-09-05" {
		t.Errorf("cycle window = (%v, %v], ok=%v; want (2026-08-05, 2026-09-05]", start, end, ok)
	}

	// No due date → no window
	g2 := &models.Group{ContributionInterval: models.IntervalMonthly}
	if _, _, ok := ContributionCycleWindow(g2, time.Now()); ok {
		t.Error("group without due date should have no cycle window")
	}
}

func strPtr(s string) *string { return &s }
