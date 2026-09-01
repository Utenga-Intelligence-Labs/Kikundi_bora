package services

import (
	"testing"
	"time"

	"kikundibora/models"

	"github.com/shopspring/decimal"
)

func loanForTest(status models.LoanStatus, principal string, balance *string, dueDate time.Time) models.Loan {
	amt, _ := decimal.NewFromString(principal)
 disbursed := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	l := models.Loan{
		ID:        "loan-" + string(status) + "-" + principal,
		Status:    status,
		Amount:    amt,
		DueDate:   dueDate,
		DisbursedAt: &disbursed,
	}
	if balance != nil {
		b, _ := decimal.NewFromString(*balance)
		l.BalanceRemaining = &b
		l.ApprovedAmount = &amt
	}
	return l
}

func TestBuildLoanPortfolioAggregation(t *testing.T) {
	dec := func(s string) *string { return &s }
	today := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)

	loans := []models.Loan{
		// Outstanding, NOT overdue (due tomorrow), 100k disbursed, 40k remaining
		loanForTest(models.LoanOutstanding, "100000", dec("40000"), time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)),
		// Outstanding, OVERDUE (due last month), 50k disbursed, 50k remaining
		loanForTest(models.LoanOutstanding, "50000", dec("50000"), time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)),
		// Closed (fully paid): 200k disbursed, 0 remaining
		loanForTest(models.LoanClosed, "200000", dec("0"), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)),
		// NOT disbursed (approved, awaiting) — must be excluded entirely
		{ID: "approved-loan", Status: models.LoanApproved, Amount: decimal.NewFromInt(999)},
		// Pending — excluded
		{ID: "pending-loan", Status: models.LoanPending, Amount: decimal.NewFromInt(888)},
	}

	sum := BuildLoanPortfolio(loans, today)

	// Totals: disbursed = 100k + 50k + 200k = 350k
	if sum.TotalDisbursed.String() != "350000" {
		t.Errorf("total_disbursed = %s, want 350000", sum.TotalDisbursed)
	}
	// Outstanding = 40k + 50k = 90k
	if sum.TotalOutstanding.String() != "90000" {
		t.Errorf("total_outstanding = %s, want 90000", sum.TotalOutstanding)
	}
	// Repaid = 60k + 0 + 200k = 260k
	if sum.TotalRepaid.String() != "260000" {
		t.Errorf("total_repaid = %s, want 260000", sum.TotalRepaid)
	}
	// Overdue = only the 50k past-due outstanding loan
	if sum.TotalOverdue.String() != "50000" || sum.CountOverdue != 1 {
		t.Errorf("overdue = %s/%d, want 50000/1", sum.TotalOverdue, sum.CountOverdue)
	}
	// Counts
	if sum.CountOutstanding != 2 || sum.CountClosed != 1 {
		t.Errorf("counts outstanding=%d closed=%d, want 2/1", sum.CountOutstanding, sum.CountClosed)
	}
	if sum.StatusCounts["OUTSTANDING"] != 2 || sum.StatusCounts["CLOSED"] != 1 {
		t.Errorf("status_counts = %v", sum.StatusCounts)
	}
	// Only disbursed loans in rows
	if len(sum.Loans) != 3 {
		t.Errorf("loan rows = %d, want 3 (non-disbursed excluded)", len(sum.Loans))
	}
	// Per-loan repaid math: 100k − 40k = 60k
	if sum.Loans[0].AmountRepaid.String() != "60000" {
		t.Errorf("first loan repaid = %s, want 60000", sum.Loans[0].AmountRepaid)
	}
	// Overdue flag set on the right loan
	overdueFlags := map[string]bool{}
	for _, l := range sum.Loans {
		overdueFlags[l.ID] = l.IsOverdue
	}
	if !overdueFlags[loans[1].ID] || overdueFlags[loans[0].ID] {
		t.Error("overdue flag should be set only on the past-due outstanding loan")
	}
}

func TestBuildLoanPortfolioEmpty(t *testing.T) {
	sum := BuildLoanPortfolio(nil, time.Now())
	if !sum.TotalDisbursed.Equal(decimal.Zero) || len(sum.Loans) != 0 {
		t.Error("empty input should produce zeroed summary")
	}
}

func TestBuildLoanPortfolioOverdueBoundary(t *testing.T) {
	// Due exactly today → NOT overdue yet
	l := loanForTest(models.LoanOutstanding, "10000", strPtr("10000"), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	sum := BuildLoanPortfolio([]models.Loan{l}, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC))
	if sum.CountOverdue != 0 {
		t.Error("loan due today must not be overdue")
	}
	// Due yesterday → overdue
	l.DueDate = time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	sum = BuildLoanPortfolio([]models.Loan{l}, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC))
	if sum.CountOverdue != 1 {
		t.Error("loan due yesterday must be overdue")
	}
}
