package handlers

import (
	"fmt"
	"log"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var treasuryService = services.NewTreasuryService()

type LoanHandler struct{}

func NewLoanHandler() *LoanHandler {
	return &LoanHandler{}
}

func (h *LoanHandler) List(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	status := c.Query("status")
	memberID := c.Query("member_id")
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	query := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		})

	// Members only see loans for their linked member record
	if role == models.RoleMember {
		var ownMember models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&ownMember).Error; err != nil {
			return c.JSON(fiber.Map{"data": []models.Loan{}, "total": 0, "page": pq.Page, "limit": pq.Limit})
		}
		query = query.Where("member_id = ?", ownMember.ID)
	} else if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Loan{}).Count(&total)

	var loans []models.Loan
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("applied_at DESC").Find(&loans)

	return c.JSON(fiber.Map{
		"data":  loans,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

func (h *LoanHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	var loan models.Loan
	if err := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone, user_id")
		}).
		Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		First(&loan, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	// Members may only read their own linked loan
	if role == models.RoleMember {
		var ownMember models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&ownMember).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Huna ruhusa ya kuona mkopo huu"})
		}
		if loan.MemberID != ownMember.ID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Huna ruhusa ya kuona mkopo huu"})
		}
	}

	var repayments []models.Repayment
	database.DB.Where("loan_id = ?", loan.ID).Order("paid_at DESC").Find(&repayments)

	return c.JSON(fiber.Map{
		"data":       loan,
		"repayments": repayments,
	})
}

func (h *LoanHandler) Apply(c *fiber.Ctx) error {
	if IsGroupDissolved() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Kikundi kimevunjwa — mikopo imefungwa"})
	}
	var req models.ApplyLoanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL AND is_active = TRUE", req.MemberID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama huyu hayupo au si hai"})
	}

	// Members may only apply for their own linked member record; staff may apply on behalf
	canApplyOnBehalf := role == models.RoleChair || role == models.RoleSecretary || role == models.RoleTreasurer || role == models.RoleAdmin
	if !canApplyOnBehalf {
		if member.UserID == nil || *member.UserID != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Unaweza kuomba mkopo kwa akaunti yako tu. Hakikisha akaunti yako imeunganishwa na rekodi ya mwanachama.",
			})
		}
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi cha mkopo lazima kiwe zaidi ya sifuri"})
	}

	// PHASE 2: Check if treasury can afford this loan
	canAfford, availableBalance, err := treasuryService.CanAffordLoan(req.Amount)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuangalia hali ya hazina"})
	}
	if !canAfford {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": fmt.Sprintf("Kiasi kinazidi hazina ya kikundi (TZS %s iliyopo). Kiasi ulichoomba: TZS %s", availableBalance.StringFixed(2), req.Amount.StringFixed(2)),
		})
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tarehe ya mwisho si sahihi"})
	}

	if !dueDate.After(time.Now()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tarehe ya mwisho lazima iwe baadaye ya leo"})
	}

	var activeCount int64
	database.DB.Model(&models.Loan{}).
		Where("member_id = ? AND status IN ?", req.MemberID, []models.LoanStatus{models.LoanPending, models.LoanApproved, models.LoanOutstanding}).
		Count(&activeCount)
	if activeCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Mwanachama ana mkopo unaoendelea. Lazima ulipwe kwanza"})
	}

	loan := models.Loan{
		MemberID: req.MemberID,
		Amount:   req.Amount,
		Purpose:  req.Purpose,
		DueDate:  dueDate,
		Status:   models.LoanPending,
	}

	if err := database.DB.Create(&loan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutuma ombi"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "loans", &loan.ID, nil, map[string]interface{}{
		"member_id": req.MemberID, "amount": req.Amount, "due_date": req.DueDate, "status": "PENDING",
	})

	// Notify all committee members (leaders + appointed)
	msg := "Kuna ombi jipya la mkopo la TZS " + formatMoney(req.Amount) + " linahitaji ukaguzi wa kamati."
	services.NotifyRole(models.RoleChair, models.NotifLoanRequest, "Ombi Jipya la Mkopo", msg, "")
	services.NotifyRole(models.RoleSecretary, models.NotifLoanRequest, "Ombi Jipya la Mkopo", msg, "")
	services.NotifyRole(models.RoleTreasurer, models.NotifLoanRequest, "Ombi Jipya la Mkopo", msg, "")

	var appointed []models.LoanCommitteeMember
	database.DB.Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").Find(&appointed)
	for _, m := range appointed {
		services.NotifyUser(m.UserID, models.NotifLoanRequest, "Ombi Jipya la Mkopo", msg)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Ombi la mkopo limetumwa. Kamati ya mikopo itapitia ombi lako.",
		"data":    loan,
	})
}

func (h *LoanHandler) Approve(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.ApproveLoanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var loan models.Loan
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	if loan.Status != models.LoanPending {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mkopo huu hauwezi kuidhinishwa. Hali yake ni: " + string(loan.Status)})
	}

	if req.ApprovedAmount.LessThanOrEqual(decimal.Zero) || req.ApprovedAmount.GreaterThan(loan.Amount) {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi kilichoidhinishwa sio sahihi"})
	}

	now := time.Now()
	loan.Status = models.LoanApproved
	loan.ApprovedAmount = &req.ApprovedAmount
	loan.ReviewedBy = &userID
	loan.ReviewedAt = &now

	if err := tx.Save(&loan).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	services.LogAudit(c, &userID, models.AuditApprove, "loans", &loan.ID, map[string]interface{}{
		"status": "PENDING",
	}, map[string]interface{}{
		"status": "APPROVED", "approved_amount": req.ApprovedAmount,
	})

	// Notify member
	var member models.Member
	if err := database.DB.First(&member, "id = ?", loan.MemberID).Error; err == nil {
		var notifUserID string
		if member.UserID != nil {
			notifUserID = *member.UserID
		}
		if notifUserID == "" {
			notifUserID = member.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifLoanApproved, "Mkopo Umeidhinishwa",
				"Mkopo wa TZS "+formatMoney(req.ApprovedAmount)+" umeidhinishwa.")
		}
	}

	return c.JSON(fiber.Map{
		"message": "Mkopo umeidhinishwa. Mweka Hazina atautolea fedha.",
		"data":    loan,
	})
}

func (h *LoanHandler) Reject(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.RejectLoanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var loan models.Loan
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	if loan.Status != models.LoanPending {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mkopo huu hauwezi kukataliwa. Hali yake ni: " + string(loan.Status)})
	}

	now := time.Now()
	reason := req.Reason
	loan.Status = models.LoanRejected
	loan.RejectionReason = &reason
	loan.ReviewedBy = &userID
	loan.ReviewedAt = &now

	if err := tx.Save(&loan).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	services.LogAudit(c, &userID, models.AuditReject, "loans", &loan.ID, map[string]interface{}{
		"status": "PENDING",
	}, map[string]interface{}{
		"status": "REJECTED", "reason": reason,
	})

	return c.JSON(fiber.Map{"message": "Mkopo umekataliwa", "data": loan})
}

func (h *LoanHandler) Disburse(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var loan models.Loan
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	if loan.Status != models.LoanApproved {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Mkopo hauwezi kutolewa. Hali yake ni: " + string(loan.Status),
		})
	}

	if loan.ApprovedAmount == nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Mkopo hauna kiasi kilichoidhinishwa"})
	}

	now := time.Now()
	bal := *loan.ApprovedAmount
	loan.Status = models.LoanOutstanding
	loan.DisbursedBy = &userID
	loan.DisbursedAt = &now
	loan.BalanceRemaining = &bal

	if err := tx.Save(&loan).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutolea mkopo"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutolea mkopo"})
	}

	services.LogAudit(c, &userID, models.AuditUpdate, "loans", &loan.ID,
		map[string]interface{}{"status": "APPROVED"},
		map[string]interface{}{"status": "OUTSTANDING"},
	)

	// Notify member
	var member models.Member
	if err := database.DB.First(&member, "id = ?", loan.MemberID).Error; err == nil {
		var notifUserID string
		if member.UserID != nil {
			notifUserID = *member.UserID
		}
		if notifUserID == "" {
			notifUserID = member.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifLoanDisbursed, "Mkopo Umetolewa",
				"Mkopo wako wa TZS "+formatMoney(*loan.ApprovedAmount)+" umetolewa.")
		}
		// Auto-post into the double-entry ledger (best-effort).
		if err := services.PostDisbursement(member.MemberNo, *loan.ApprovedAmount, now, userID,
			fmt.Sprintf("Mkopo uliotolewa %s", member.MemberNo)); err != nil {
			log.Printf("WARN: ledger auto-post disburse %s: %v", loan.ID, err)
		}
	}

	return c.JSON(fiber.Map{"message": "Mkopo umetolewa", "data": loan})
}

func (h *LoanHandler) OutstandingReport(c *fiber.Ctx) error {
	type Row struct {
		MemberNo         string  `json:"member_no"`
		FullName         string  `json:"full_name"`
		Phone            string  `json:"phone"`
		LoanID           string  `json:"loan_id"`
		ApprovedAmount   decimal.Decimal `json:"approved_amount"`
		BalanceRemaining decimal.Decimal `json:"balance_remaining"`
		DueDate          string          `json:"due_date"`
	}

	var rows []Row
	database.DB.Raw(`
		SELECT m.member_no, m.full_name, m.phone,
		       l.id AS loan_id,
		       COALESCE(l.approved_amount, 0) AS approved_amount,
		       COALESCE(l.balance_remaining, 0) AS balance_remaining,
		       l.due_date
		FROM loans l
		JOIN members m ON m.id = l.member_id
		WHERE l.status = 'OUTSTANDING'
		ORDER BY l.due_date ASC
	`).Scan(&rows)

	now := time.Now()
	result := make([]models.OutstandingLoanRow, len(rows))
	for i, r := range rows {
		due, _ := time.Parse("2006-01-02", r.DueDate)
		daysRemaining := int(due.Sub(now).Hours() / 24)
		urgency := "KAWAIDA"
		if daysRemaining < 0 {
			urgency = "IMEPITA MUDA"
		} else if daysRemaining <= 7 {
			urgency = "INAKARIBIA MUDA"
		}

		result[i] = models.OutstandingLoanRow{
			MemberNo:         r.MemberNo,
			FullName:         r.FullName,
			Phone:            r.Phone,
			LoanID:           r.LoanID,
			ApprovedAmount:   r.ApprovedAmount,
			BalanceRemaining: r.BalanceRemaining,
			AmountPaidSoFar:  r.ApprovedAmount.Sub(r.BalanceRemaining),
			DueDate:          r.DueDate,
			Urgency:          urgency,
			DaysRemaining:    daysRemaining,
		}
	}

	return c.JSON(fiber.Map{"data": result})
}

func formatMoney(n decimal.Decimal) string {
	return n.StringFixed(2)
}
