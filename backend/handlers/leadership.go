package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	// Get contributions this month from NEW MemberContribution table (Phase 4)
	var contributionsThisMonth float64
	startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	database.DB.Model(&models.MemberContribution{}).
		Where("status = ? AND created_at >= ?", models.ContributionConfirmed, startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&contributionsThisMonth)

	// Also check OLD Contribution table for legacy data
	var legacyContributionsThisMonth float64
	database.DB.Model(&models.Contribution{}).
		Where("status = 'PAID' AND paid_at >= ?", startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&legacyContributionsThisMonth)
	contributionsThisMonth += legacyContributionsThisMonth

	// Get pending contributions count
	var pendingContributions int64
	database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionPending).
		Count(&pendingContributions)

	// Get outstanding loans
	var outstandingLoans int64
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).
		Count(&outstandingLoans)

	// Get pending loans
	var pendingLoans int64
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanPending).
		Count(&pendingLoans)

	// Get treasury balance
	treasury, err := treasurySvc.CalculateHazinaBalance()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kupata hesabu ya hazina",
		})
	}

	return c.JSON(fiber.Map{
		"total_members":          totalMembers,
		"contributions_month":    contributionsThisMonth,
		"pending_contributions":  pendingContributions,
		"outstanding_loans":      outstandingLoans,
		"pending_loans":          pendingLoans,
		"treasury_balance":       treasury.AvailableBalance,
		"total_contributions":    treasury.TotalContributions,
		"total_repayments":       treasury.TotalRepayments,
		"total_disbursed":        treasury.TotalDisbursed,
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

// ApproveLoan handles sequential loan approval: Hazina → Katibu → Mwenyekiti.
// POST /api/v1/uongozi/mikopo/:id/approve
func (h *LeadershipHandler) ApproveLoan(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	var req struct {
		ApprovedAmount float64 `json:"approved_amount"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	tx := database.DB.Begin()

	var loan models.Loan
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	if loan.Status != models.LoanPending {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mkopo huu hauwezi kuidhinishwa. Hali yake: " + string(loan.Status)})
	}

	now := time.Now()

	// Enforce sequential order: Hazina → Katibu → Bodi Member → Mwenyekiti
	switch role {
	case models.RoleTreasurer:
		if loan.HazinaApprovedAt != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hazina tayari ameidhinisha mkopo huu"})
		}
		loan.HazinaApprovedBy = &userID
		loan.HazinaApprovedAt = &now

	case models.RoleSecretary:
		if loan.HazinaApprovedAt == nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hazina lazima aidhinishe kwanza kabla ya Katibu"})
		}
		if loan.KatibuApprovedAt != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Katibu tayari ameidhinisha mkopo huu"})
		}
		loan.KatibuApprovedBy = &userID
		loan.KatibuApprovedAt = &now

	case models.RoleChair:
		if loan.HazinaApprovedAt == nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hazina lazima aidhinishe kwanza kabla ya Mwenyekiti"})
		}
		if loan.KatibuApprovedAt == nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Katibu lazima aidhinishe kwanza kabla ya Mwenyekiti"})
		}
		if loan.BodiApprovedAt == nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Bodi ya mikopo lazima iidhinishe kwanza kabla ya Mwenyekiti"})
		}
		if loan.MwenyekitiApprovedAt != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mwenyekiti tayari ameidhinisha mkopo huu"})
		}
		loan.MwenyekitiApprovedBy = &userID
		loan.MwenyekitiApprovedAt = &now

		// Final approval: all four have approved
		loan.Status = models.LoanApproved
		loan.ApprovedAmount = &loan.Amount
		if req.ApprovedAmount > 0 {
			loan.ApprovedAmount = &req.ApprovedAmount
		}
		loan.ReviewedBy = &userID
		loan.ReviewedAt = &now

	default:
		// Bodi member (appointed committee member) — must come after Katibu and before Mwenyekiti
		var isCommittee bool
		database.DB.Model(&models.LoanCommitteeMember{}).
			Where("user_id = ? AND is_active = TRUE", userID).
			Select("1").Scan(&isCommittee)
		if !isCommittee {
			tx.Rollback()
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Huna ruhusa ya kuidhinisha mkopo"})
		}
		if loan.HazinaApprovedAt == nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hazina lazima aidhinishe kwanza"})
		}
		if loan.KatibuApprovedAt == nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Katibu lazima aidhinishe kwanza"})
		}
		if loan.BodiApprovedAt != nil {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Bodi tayari imeidhinisha mkopo huu"})
		}
		loan.BodiApprovedBy = &userID
		loan.BodiApprovedAt = &now
	}

	if err := tx.Save(&loan).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	tx.Commit()

	services.LogAudit(c, &userID, models.AuditLoanReview, "loans", &loan.ID, nil, map[string]interface{}{
		"status": string(loan.Status), "role": string(role),
	})

	return c.JSON(fiber.Map{"message": "Mkopo umeidhinishwa", "data": loan})
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
