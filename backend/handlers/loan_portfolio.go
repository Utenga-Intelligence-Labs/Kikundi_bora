package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LoanPortfolioHandler struct{}

func NewLoanPortfolioHandler() *LoanPortfolioHandler {
	return &LoanPortfolioHandler{}
}

// Portfolio returns every disbursed loan in the group with aggregate stats
// (total disbursed / repaid / outstanding / overdue, counts by status) and
// the individual loan rows. Filters: status, member_id, from/to (disbursed
// date range). Access: mwenyekiti, katibu, mweka hazina (+ admin) via route
// guard.
// GET /api/v1/loans/portfolio
func (h *LoanPortfolioHandler) Portfolio(c *fiber.Ctx) error {
	q := database.DB.Model(&models.Loan{}).
		Where("disbursed_at IS NOT NULL").
		Where("status IN ?", []models.LoanStatus{models.LoanOutstanding, models.LoanClosed})

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if memberID := c.Query("member_id"); memberID != "" {
		q = q.Where("member_id = ?", memberID)
	}
	if from := c.Query("from"); from != "" {
		q = q.Where("disbursed_at >= ?", from)
	}
	if to := c.Query("to"); to != "" {
		q = q.Where("disbursed_at <= ?", to+" 23:59:59")
	}

	var loans []models.Loan
	if err := q.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name")
		}).
		Order("disbursed_at DESC").
		Find(&loans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata portfolio"})
	}

	sum := services.BuildLoanPortfolio(loans, time.Now())
	return c.JSON(fiber.Map{"data": sum})
}
