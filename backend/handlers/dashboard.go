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
		Select("COALESCE(SUM(amount), 0)").Scan(&totalContributions)

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

	database.DB.Model(&models.Contribution{}).
		Where("month = ? AND member_id IN (SELECT id FROM members WHERE is_active = TRUE AND deleted_at IS NULL)", monthFirst).
		Distinct("member_id").
		Count(&membersPaid)

	database.DB.Model(&models.Contribution{}).
		Where("month = ?", monthFirst).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalContributionsMonth)

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
