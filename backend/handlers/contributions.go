package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ContributionHandler struct{}

func NewContributionHandler() *ContributionHandler {
	return &ContributionHandler{}
}

func (h *ContributionHandler) List(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	memberID := c.Query("member_id")
	month := c.Query("month")
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	query := database.DB.Preload("Member", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, member_no, full_name, phone")
	}).Preload("Recorder", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, name, role")
	})

	// Members only see their own contributions
	if role == models.RoleMember {
		var ownMember models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&ownMember).Error; err != nil {
			return c.JSON(fiber.Map{"data": []models.Contribution{}, "total": 0, "page": pq.Page, "limit": pq.Limit})
		}
		query = query.Where("member_id = ?", ownMember.ID)
	} else if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if month != "" {
		query = query.Where("to_char(month, 'YYYY-MM') = ?", month)
	}

	var total int64
	query.Model(&models.Contribution{}).Count(&total)

	var contributions []models.Contribution
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&contributions)

	return c.JSON(fiber.Map{
		"data":  contributions,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

func (h *ContributionHandler) Create(c *fiber.Ctx) error {
	if IsGroupDissolved() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Kikundi kimevunjwa — michango imefungwa"})
	}
	var req models.CreateContributionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL AND is_active = TRUE", req.MemberID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana au si hai"})
	}

	monthDate, err := time.Parse("2006-01", req.Month)
	if err != nil {
		monthDate, err = time.Parse("2006-01-02", req.Month)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mwezi si sahihi"})
		}
	}
	monthFirst := time.Date(monthDate.Year(), monthDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tarehe ya malipo si sahihi"})
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi lazima kiwe zaidi ya sifuri"})
	}

	// NOTE: no exact-amount or already-paid-month rejection here. Payments
	// are allocated across arrears → current → fines, so any positive
	// amount is accepted (over/under-payments flow by rule).

	userID := middleware.GetUserID(c)

	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	// Keep legacy groups usable until contribution-interval settings are
	// configured. Scheduled groups always use the obligations allocator below.
	if g.ContributionDueDate == nil || g.ContributionInterval == "" {
		row := models.Contribution{
			MemberID: member.ID, RecordedBy: userID, Amount: req.Amount,
			Month: monthFirst, PaidAt: paidAt, PaymentMethod: req.PaymentMethod,
			ReferenceNumber: req.ReferenceNumber, Status: "PAID", ConfirmedBy: &userID,
		}
		if err := database.DB.Create(&row).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mchango wa mwezi huu tayari umerekodiwa"})
		}
		line := services.AllocatedLine{Kind: services.AllocCurrent, Ref: req.Month, Label: "Mchango " + req.Month, Amount: req.Amount}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Malipo yamerekodiwa",
			"data":    services.AllocationReceipt{Lines: []services.AllocatedLine{line}, Applied: req.Amount, Remainder: decimal.Zero},
		})
	}

	// Allocate across arrears → current → fines (confirmed order). The
	// treasurer may record any positive amount — catching up on arrears
	// legitimately exceeds one cycle's fixed amount. The old "already paid
	// this month" conflict is gone: extra money flows to arrears/fines.
	receipt, err := services.ApplyAllocation(g.ID, req.MemberID, req.Amount, userID,
		req.PaymentMethod, req.ReferenceNumber, paidAt, time.Now())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "contributions", nil, nil, map[string]interface{}{
		"member_id": req.MemberID, "amount": req.Amount, "month": monthFirst.Format("2006-01-02"),
		"payment_method": req.PaymentMethod, "reference": req.ReferenceNumber,
		"allocated": receipt.Lines, "remainder": receipt.Remainder,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Malipo yamegawanywa na kurekodiwa",
		"data":    receipt,
	})
}

func (h *ContributionHandler) Edit(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.EditContributionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	var contrib models.Contribution
	if err := database.DB.First(&contrib, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mchango haujapatikana"})
	}

	if req.NewAmount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi lazima kiwe zaidi ya sifuri"})
	}

	oldAmount := contrib.Amount
	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	contrib.Amount = req.NewAmount
	if err := tx.Save(&contrib).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha"})
	}

	edit := models.ContributionEdit{
		ContributionID: contrib.ID,
		EditedBy:       userID,
		OldAmount:      oldAmount,
		NewAmount:      req.NewAmount,
		Reason:         req.Reason,
	}
	if err := tx.Create(&edit).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kurekodi mabadiliko"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kurekodi mabadiliko"})
	}

	services.LogAudit(c, &userID, models.AuditUpdate, "contributions", &contrib.ID, map[string]interface{}{
		"amount": oldAmount,
	}, map[string]interface{}{
		"amount": req.NewAmount, "reason": req.Reason,
	})

	return c.JSON(fiber.Map{"message": "Mchango umebadilishwa", "data": contrib})
}

func (h *ContributionHandler) MonthlyReport(c *fiber.Ctx) error {
	month := c.Query("month", time.Now().Format("2006-01"))
	monthFirst, err := time.Parse("2006-01", month)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mwezi si sahihi. Tumia YYYY-MM"})
	}
	monthDate := time.Date(monthFirst.Year(), monthFirst.Month(), 1, 0, 0, 0, 0, time.UTC)

	type Row struct {
		MemberNo   string           `json:"member_no"`
		FullName   string           `json:"full_name"`
		Phone      string           `json:"phone"`
		AmountPaid *decimal.Decimal `json:"amount_paid"`
		PaidAt     *string          `json:"paid_at"`
		ContribID  *string          `json:"contrib_id"`
		Notes      *string          `json:"notes"`
	}

	var rows []Row
	database.DB.Raw(`
		SELECT m.member_no, m.full_name, m.phone,
		       c.amount AS amount_paid,
		       c.paid_at,
		       c.id AS contrib_id,
		       c.notes
		FROM members m
		LEFT JOIN contributions c ON c.member_id = m.id AND c.month = ?
		WHERE m.is_active = TRUE AND m.deleted_at IS NULL
		ORDER BY m.member_no
	`, monthDate).Scan(&rows)

	result := make([]models.MonthlyContributionRow, len(rows))
	for i, r := range rows {
		status := "HAJALIPA"
		paidAt := ""
		if r.ContribID != nil {
			status = "AMELIPA"
			if r.PaidAt != nil {
				paidAt = *r.PaidAt
			}
		}
		amt := decimal.Zero
		if r.AmountPaid != nil {
			amt = *r.AmountPaid
		}
		result[i] = models.MonthlyContributionRow{
			MemberNo:   r.MemberNo,
			FullName:   r.FullName,
			Phone:      r.Phone,
			AmountPaid: amt,
			Status:     status,
			Notes:      r.Notes,
		}
		if r.ContribID != nil && r.PaidAt != nil {
			result[i].PaidAt = &paidAt
		}
	}

	return c.JSON(fiber.Map{"data": result, "month": month})
}
