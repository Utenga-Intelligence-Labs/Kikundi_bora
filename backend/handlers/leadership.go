package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LeadershipHandler struct{}

var treasurySvc = services.NewTreasuryService()

func NewLeadershipHandler() *LeadershipHandler {
	return &LeadershipHandler{}
}

// QuickStats returns quick statistics for the "Takwimu za Haraka" dashboard section.
// GET /api/v1/uongozi/quick-stats
func (h *LeadershipHandler) QuickStats(c *fiber.Ctx) error {
	// Get total members count
	var totalMembers int64
	database.DB.Model(&models.Member{}).
		Where("deleted_at IS NULL").
		Count(&totalMembers)

	// Get contributions this month
	var contributionsThisMonth float64
	startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	database.DB.Model(&models.Contribution{}).
		Where("status = 'PAID' AND paid_at >= ?", startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&contributionsThisMonth)

	// Get outstanding loans
	var outstandingLoans int64
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).
		Count(&outstandingLoans)

	// Get treasury balance
	treasury, err := treasurySvc.CalculateHazinaBalance()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kupata hesabu ya hazina",
		})
	}

	return c.JSON(fiber.Map{
		"total_members":         totalMembers,
		"contributions_month":   contributionsThisMonth,
		"outstanding_loans":     outstandingLoans,
		"treasury_balance":      treasury.AvailableBalance,
		"total_contributions":   treasury.TotalContributions,
		"total_repayments":      treasury.TotalRepayments,
		"total_disbursed":       treasury.TotalDisbursed,
	})
}

// PendingLoans returns loans with status PENDING (leadership view).
// GET /api/v1/uongozi/mikopo/pending
func (h *LeadershipHandler) PendingLoans(c *fiber.Ctx) error {
	var loans []models.Loan
	if err := database.DB.
		Where("status = ?", models.LoanPending).
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Order("created_at DESC").
		Find(&loans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kupata mikopo",
		})
	}

	return c.JSON(fiber.Map{
		"data":  loans,
		"total": len(loans),
	})
}

// ApproveLoan delegates to loan approval logic (same as POST /loans/:id/approve).
// POST /api/v1/uongozi/mikopo/:id/approve
func (h *LeadershipHandler) ApproveLoan(c *fiber.Ctx) error {
	// Reuse existing loan handler
	loanHandler := NewLoanHandler()
	return loanHandler.Approve(c)
}

// Reports returns leadership reports (delegates to report handler).
// GET /api/v1/uongozi/ripoti
func (h *LeadershipHandler) Reports(c *fiber.Ctx) error {
	reportType := c.Query("type", "summary")

	reportHandler := NewReportHandler()

	switch reportType {
	case "wanachama":
		return reportHandler.MembersReport(c)
	case "michango":
		return reportHandler.ContributionsReport(c)
	case "mikopo":
		return reportHandler.LoansReport(c)
	case "mapato":
		return reportHandler.IncomeExpenseReport(c)
	case "summary":
		return reportHandler.SummaryReport(c)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Aina ya ripoti si sahihi",
		})
	}
}

// Dashboard returns leadership-specific dashboard data.
// GET /api/v1/uongozi/dashboard
func (h *LeadershipHandler) Dashboard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Find member linked to user
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Lazima uwe mwanachama wa kikundi",
		})
	}

	// Get leadership roles
	var positions []models.LeadershipPosition
	database.DB.Where("member_id = ? AND is_current = TRUE", member.ID).
		Find(&positions)

	var roles []string
	for _, p := range positions {
		roles = append(roles, string(p.Role))
	}

	// Get pending loans count
	var pendingLoans int64
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanPending).
		Count(&pendingLoans)

	// Get total members count
	var totalMembers int64
	database.DB.Model(&models.Member{}).
		Where("deleted_at IS NULL").
		Count(&totalMembers)

	// Get outstanding loans count
	var outstandingLoans int64
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).
		Count(&outstandingLoans)

	return c.JSON(fiber.Map{
		"member_id":       member.ID,
		"member_code":     member.MemberNo,
		"leadership":      roles,
		"pending_loans":   pendingLoans,
		"total_members":   totalMembers,
		"outstanding_loans": outstandingLoans,
	})
}
