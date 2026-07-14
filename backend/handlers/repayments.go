package handlers

import (
	"fmt"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RepaymentHandler struct{}

func NewRepaymentHandler() *RepaymentHandler {
	return &RepaymentHandler{}
}

func (h *RepaymentHandler) List(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	loanID := c.Query("loan_id")
	memberID := c.Query("member_id")
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	query := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Recorder", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		})

	// Members only see their own repayments
	if role == models.RoleMember {
		var ownMember models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&ownMember).Error; err != nil {
			return c.JSON(fiber.Map{"data": []models.Repayment{}, "total": 0, "page": pq.Page, "limit": pq.Limit})
		}
		query = query.Where("member_id = ?", ownMember.ID)
	} else if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if loanID != "" {
		query = query.Where("loan_id = ?", loanID)
	}

	var total int64
	query.Model(&models.Repayment{}).Count(&total)

	var repayments []models.Repayment
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("paid_at DESC").Find(&repayments)

	return c.JSON(fiber.Map{
		"data":  repayments,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

func (h *RepaymentHandler) Record(c *fiber.Ctx) error {
	var req models.RecordRepaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi cha malipo lazima kiwe zaidi ya sifuri"})
	}

	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tarehe ya malipo si sahihi"})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var loan models.Loan
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, "id = ?", req.LoanID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	if loan.Status != models.LoanOutstanding {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": fmt.Sprintf("Mkopo huu hauhitaji malipo. Hali yake ni: %s", loan.Status)})
	}

	if loan.BalanceRemaining == nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Salio la mkopo halijapatikana"})
	}

	balance := *loan.BalanceRemaining
	if req.Amount > balance {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": fmt.Sprintf("Kiasi kimezidi salio. Salio ni TZS %.2f", balance)})
	}

	newBalance := balance - req.Amount
	newStatus := models.LoanOutstanding
	const epsilon = 0.001
	if newBalance < epsilon {
		newStatus = models.LoanClosed
	}

	repayment := models.Repayment{
		LoanID:          req.LoanID,
		MemberID:        loan.MemberID,
		RecordedBy:      userID,
		Amount:          req.Amount,
		BalanceAfter:    newBalance,
		PaidAt:          paidAt,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		ReceiptURL:      req.ReceiptURL,
		Notes:           req.Notes,
	}

	if err := tx.Create(&repayment).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kurekodi malipo"})
	}

	loan.BalanceRemaining = &newBalance
	loan.Status = newStatus
	if err := tx.Save(&loan).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusasisha mkopo"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kurekodi malipo"})
	}

	// Audit and notifications only after successful commit
	services.LogAudit(c, &userID, models.AuditUpdate, "repayments", &repayment.ID,
		map[string]interface{}{"balance_before": balance},
		map[string]interface{}{"amount_paid": req.Amount, "balance_after": newBalance, "loan_status": string(newStatus), "payment_method": req.PaymentMethod},
	)

	if newStatus == models.LoanClosed {
		services.NotifyRole(models.RoleChair, models.NotifRepayment, "Mkopo Umefungwa", "Hongera! Mkopo umelipwa kikamilifu.", "")
		services.NotifyRole(models.RoleTreasurer, models.NotifRepayment, "Mkopo Umefungwa", "Hongera! Mkopo umelipwa kikamilifu.", "")
	}

	loanClosed := newStatus == models.LoanClosed
	msg := fmt.Sprintf("Malipo yamerekodiwa. Salio lililobaki: TZS %.2f", newBalance)
	if loanClosed {
		msg = "Malipo yamerekodiwa. Mkopo umefungwa kikamilifu!"
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": msg,
		"data": models.RepaymentResponse{
			RepaymentID:  repayment.ID,
			BalanceAfter: newBalance,
			LoanClosed:   loanClosed,
		},
	})
}
