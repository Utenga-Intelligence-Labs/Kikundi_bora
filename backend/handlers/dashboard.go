package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func (h *DashboardHandler) Summary(c *fiber.Ctx) error {
	var (
		activeMembers              int64
		totalContributions         decimal.Decimal
		totalLoansIssued           decimal.Decimal
		totalRepayments            decimal.Decimal
		totalOutstanding           decimal.Decimal
		countOutstanding           int64
		countPending               int64
		membersPaid                int64
		totalContributionsMonth    decimal.Decimal
		totalRepaymentsMonth       decimal.Decimal
	)

	now := time.Now()
	monthFirst := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	database.DB.Model(&models.Member{}).
		Where("is_active = TRUE AND deleted_at IS NULL").Count(&activeMembers)

	database.DB.Model(&models.Contribution{}).
		Where("status = ?", "PAID").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalContributions)
	// FIX (data binding): also include CONFIRMED member-submitted contributions
	// ("Weka Mchango" flow) — previously only treasurer-recorded rows were
	// counted, so confirmed self-submissions never appeared in any total.
	var confirmedMemberContributions decimal.Decimal
	database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionConfirmed).
		Select("COALESCE(SUM(amount), 0)").Scan(&confirmedMemberContributions)
	totalContributions = totalContributions.Add(confirmedMemberContributions)

	database.DB.Model(&models.Loan{}).
		Where("status != ?", models.LoanRejected).
		Select("COALESCE(SUM(COALESCE(approved_amount, 0)), 0)").Scan(&totalLoansIssued)

	database.DB.Model(&models.Repayment{}).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRepayments)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).
		Select("COALESCE(SUM(COALESCE(balance_remaining, 0)), 0)").Scan(&totalOutstanding)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).Count(&countOutstanding)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanPending).Count(&countPending)

	// FIX (data binding): members who paid this period — union of BOTH stores
	// (treasurer-recorded legacy rows AND confirmed self-submissions),
	// counted via one query so a member in both stores is not double-counted.
	database.DB.Raw(`
		SELECT COUNT(DISTINCT member_id) FROM (
			SELECT member_id FROM contributions WHERE month = ? AND status = 'PAID'
			UNION
			SELECT member_id FROM member_contributions WHERE period_label = ? AND status = ?
		) paid
	`, monthFirst, monthFirst.Format("2006-01"), models.ContributionConfirmed).Scan(&membersPaid)

	database.DB.Model(&models.Contribution{}).
		Where("month = ? AND status = ?", monthFirst, "PAID").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalContributionsMonth)
	// FIX (data binding): current-month confirmed member-submissions too
	// (period_label is the "YYYY-MM" string used by the "Weka Mchango" flow).
	var confirmedMemberContributionsMonth decimal.Decimal
	database.DB.Model(&models.MemberContribution{}).
		Where("period_label = ? AND status = ?", monthFirst.Format("2006-01"), models.ContributionConfirmed).
		Select("COALESCE(SUM(amount), 0)").Scan(&confirmedMemberContributionsMonth)
	totalContributionsMonth = totalContributionsMonth.Add(confirmedMemberContributionsMonth)

	database.DB.Model(&models.Repayment{}).
		Where("paid_at >= ? AND paid_at < ?", monthFirst, monthFirst.AddDate(0, 1, 0)).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRepaymentsMonth)

	membersDefaulted := activeMembers - membersPaid

	return c.JSON(models.DashboardSummary{
		TotalActiveMembers:          activeMembers,
		TotalContributions:          totalContributions,
		TotalLoansIssued:            totalLoansIssued,
		TotalRepayments:             totalRepayments,
		TotalOutstanding:            totalOutstanding,
		CountOutstandingLoans:       countOutstanding,
		CountPendingLoans:           countPending,
		MembersPaidThisMonth:        membersPaid,
		MembersDefaulted:            membersDefaulted,
		TotalContributionsThisMonth: totalContributionsMonth,
		TotalRepaymentsThisMonth:    totalRepaymentsMonth,
	})
}
