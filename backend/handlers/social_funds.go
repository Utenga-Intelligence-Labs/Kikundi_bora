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

type SocialFundHandler struct{}

func NewSocialFundHandler() *SocialFundHandler {
	return &SocialFundHandler{}
}

// ---------- Mwenyekiti: create (goes to Katibu for approval) ----------

type createSocialFundRequest struct {
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	TargetAmount *decimal.Decimal `json:"target_amount"`
}

// Create declares a new social fund. Mwenyekiti only (route guard). The fund
// stays PENDING_APPROVAL until the Katibu approves it.
// POST /api/v1/groups/:id/social-funds
func (h *SocialFundHandler) Create(c *fiber.Ctx) error {
	var req createSocialFundRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Jina la mfuko linahitajika"})
	}
	if req.TargetAmount != nil && req.TargetAmount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi lengwa lazima kiwe zaidi ya sifuri"})
	}

	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	userID := middleware.GetUserID(c)
	fund := models.SocialFund{
		GroupID:      g.ID,
		Name:         req.Name,
		Description:  req.Description,
		TargetAmount: req.TargetAmount,
		Status:       models.SocialFundPendingApproval,
		CreatedBy:    userID,
	}
	if err := database.DB.Create(&fund).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutunda mfuko"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "social_funds", &fund.ID, nil, map[string]interface{}{
		"name": fund.Name, "target": fund.TargetAmount,
	})

	services.NotifyRole(models.RoleSecretary, models.NotifSystem,
		"Mfuko Mpya wa Kijamii",
		"Mwenyekiti ameunda mfuko \""+fund.Name+"\". Unahitaji kuuidhinisha.",
		"",
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Mfuko umetundwa. Unasubiri idhini ya Katibu.",
		"data":    fund,
	})
}

// ---------- Katibu: approve / reject fund creation ----------

// ApproveFund activates the fund. Katibu only (route guard).
// POST /api/v1/groups/:id/social-funds/:fundId/approve
func (h *SocialFundHandler) ApproveFund(c *fiber.Ctx) error {
	fund, err := h.loadFund(c)
	if fund == nil {
		return err
	}
	if fund.Status != models.SocialFundPendingApproval {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mfuko huu haunasubiri idhini. Hali yake: " + string(fund.Status)})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	fund.Status = models.SocialFundActive
	fund.ApprovedBy = &userID
	fund.ApprovedAt = &now
	if err := database.DB.Save(fund).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	services.LogAudit(c, &userID, models.AuditApprove, "social_funds", &fund.ID,
		map[string]interface{}{"status": models.SocialFundPendingApproval},
		map[string]interface{}{"status": models.SocialFundActive},
	)

	services.NotifyUser(fund.CreatedBy, models.NotifSystem,
		"Mfuko Umeidhinishwa",
		"Mfuko \""+fund.Name+"\" umeidhinishwa na Katibu. Wanachama wanaweza kuanza kuchanga.",
	)

	return c.JSON(fiber.Map{"message": "Mfuko umeidhinishwa na sasa unatumika.", "data": fund})
}

// RejectFund declines a pending fund. Katibu only (route guard).
// POST /api/v1/groups/:id/social-funds/:fundId/reject
func (h *SocialFundHandler) RejectFund(c *fiber.Ctx) error {
	fund, err := h.loadFund(c)
	if fund == nil {
		return err
	}
	if fund.Status != models.SocialFundPendingApproval {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mfuko huu haunasubiri idhini"})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Sababu ya kukataa inahitajika"})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	fund.Status = models.SocialFundRejected
	fund.RejectionReason = &req.Reason
	fund.ApprovedBy = &userID
	fund.ApprovedAt = &now
	if err := database.DB.Save(fund).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	services.LogAudit(c, &userID, models.AuditReject, "social_funds", &fund.ID,
		map[string]interface{}{"status": models.SocialFundPendingApproval},
		map[string]interface{}{"status": models.SocialFundRejected, "reason": req.Reason},
	)

	services.NotifyUser(fund.CreatedBy, models.NotifSystem,
		"Mfuko Umekataliwa",
		"Mfuko \""+fund.Name+"\" umekataliwa na Katibu. Sababu: "+req.Reason,
	)

	return c.JSON(fiber.Map{"message": "Mfuko umekataliwa.", "data": fund})
}

// ---------- Mwenyekiti: close a fund ----------

// CloseFund stops contributions to an active fund. Mwenyekiti only (route guard).
// POST /api/v1/groups/:id/social-funds/:fundId/close
func (h *SocialFundHandler) CloseFund(c *fiber.Ctx) error {
	fund, err := h.loadFund(c)
	if fund == nil {
		return err
	}
	if fund.Status != models.SocialFundActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mfuko haiwezi kufungwa. Hali yake: " + string(fund.Status)})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	fund.Status = models.SocialFundClosed
	fund.ClosedAt = &now
	if err := database.DB.Save(fund).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kufunga mfuko"})
	}

	services.LogAudit(c, &userID, models.AuditUpdate, "social_funds", &fund.ID,
		map[string]interface{}{"status": models.SocialFundActive},
		map[string]interface{}{"status": models.SocialFundClosed},
	)

	return c.JSON(fiber.Map{"message": "Mfuko umefungwa.", "data": fund})
}

// ---------- Lists & detail ----------

// List returns the group's funds. Members see ACTIVE (and CLOSED for
// history); leadership sees everything including pending/rejected.
// GET /api/v1/groups/:id/social-funds
func (h *SocialFundHandler) List(c *fiber.Ctx) error {
	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	role := middleware.GetUserRole(c)
	query := database.DB.Where("group_id = ?", g.ID)
	if role != models.RoleChair && role != models.RoleSecretary && role != models.RoleTreasurer && role != models.RoleAdmin {
		query = query.Where("status IN ?", []models.SocialFundStatus{models.SocialFundActive, models.SocialFundClosed})
	}

	var funds []models.SocialFund
	query.Order("created_at DESC").Find(&funds)

	return c.JSON(fiber.Map{"data": funds, "total": len(funds)})
}

// GetFund returns one fund with its contribution history and running balance.
// GET /api/v1/groups/:id/social-funds/:fundId
func (h *SocialFundHandler) GetFund(c *fiber.Ctx) error {
	fund, err := h.loadFund(c)
	if fund == nil {
		return err
	}

	// Members can only open active/closed funds
	role := middleware.GetUserRole(c)
	if role != models.RoleChair && role != models.RoleSecretary && role != models.RoleTreasurer && role != models.RoleAdmin {
		if fund.Status != models.SocialFundActive && fund.Status != models.SocialFundClosed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Mfuko huu bado haujaidhinishwa"})
		}
	}

	var contribs []models.SocialFundContribution
	database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name")
		}).
		Where("fund_id = ?", fund.ID).
		Order("created_at DESC").
		Find(&contribs)

	var confirmed, pending, rejected int64
	for _, cn := range contribs {
		switch cn.Status {
		case models.SFCConfirmed:
			confirmed++
		case models.SFCPending:
			pending++
		case models.SFCRejected:
			rejected++
		}
	}

	return c.JSON(fiber.Map{
		"data": fund,
		"contributions": contribs,
		"stats": fiber.Map{
			"confirmed_count": confirmed,
			"pending_count":   pending,
			"rejected_count":  rejected,
		},
	})
}

// ---------- Wanachama: contribute ----------

// Contribute declares a payment into a social fund (status PENDING until the
// Mweka Hazina confirms). Any authenticated user with a member row can
// contribute for themselves.
// POST /api/v1/groups/:id/social-funds/:fundId/contribute
func (h *SocialFundHandler) Contribute(c *fiber.Ctx) error {
	var req struct {
		Amount decimal.Decimal `json:"amount"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi lazima kiwe zaidi ya sifuri"})
	}

	fund, err := h.loadFund(c)
	if fund == nil {
		return err
	}
	if fund.Status != models.SocialFundActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mfuko huu haukubali michango kwa sasa. Hali yake: " + string(fund.Status)})
	}

	userID := middleware.GetUserID(c)
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Lazima uwe mwanachama wa kikundi kuchanga"})
	}

	contrib := models.SocialFundContribution{
		FundID:   fund.ID,
		MemberID: member.ID,
		Amount:   req.Amount,
		Status:   models.SFCPending,
	}
	if err := database.DB.Create(&contrib).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutuma mchango"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "social_fund_contributions", &contrib.ID, nil, map[string]interface{}{
		"fund_id": fund.ID, "amount": req.Amount,
	})

	services.NotifyRole(models.RoleTreasurer, models.NotifContribution,
		"Mchango Mpya wa Mfuko",
		"Kuna mchango wa TZS "+req.Amount.StringFixed(2)+" kwenye mfuko \""+fund.Name+"\" unasubiri uthibitisho wako.",
		"",
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Mchango umetumwa. Unasubiri uthibitisho wa Mweka Hazina.",
		"data":    contrib,
	})
}

// ---------- Mweka Hazina: confirm / reject contributions ----------

// ConfirmContribution marks a pending contribution CONFIRMED and credits the
// fund balance. Mweka Hazina only (route guard).
// POST /api/v1/groups/:id/social-funds/:fundId/contributions/:cid/confirm
func (h *SocialFundHandler) ConfirmContribution(c *fiber.Ctx) error {
	contrib, fund, err := h.loadContribution(c)
	if contrib == nil {
		return err
	}
	if fund.Status != models.SocialFundActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mfuko huu haukubali michango. Hali yake: " + string(fund.Status)})
	}
	if contrib.Status != models.SFCPending {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Mchango huu umeshashughulikiwa. Hali yake: " + string(contrib.Status)})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()

	tx := database.DB.Begin()
	contrib.Status = models.SFCConfirmed
	contrib.ContributedAt = &now
	contrib.VerifiedBy = &userID
	contrib.VerifiedAt = &now
	if err := tx.Save(contrib).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuthibitisha"})
	}

	// Fund balance grows ONLY on confirmation — kept in sync with the
	// CONFIRMED ledger rows.
	fund.CurrentBalance = fund.CurrentBalance.Add(contrib.Amount)
	if err := tx.Save(fund).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusasisha salio la mfuko"})
	}
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuthibitisha"})
	}

	services.LogAudit(c, &userID, models.AuditApprove, "social_fund_contributions", &contrib.ID,
		map[string]interface{}{"status": models.SFCPending},
		map[string]interface{}{"status": models.SFCConfirmed, "amount": contrib.Amount},
	)

	// Notify the contributing member
	var cm models.Member
	if err := database.DB.First(&cm, "id = ?", contrib.MemberID).Error; err == nil {
		notified := ""
		if cm.UserID != nil {
			notified = *cm.UserID
		} else {
			notified = cm.RegisteredBy
		}
		if notified != "" {
			services.NotifyUser(notified, models.NotifContribution,
				"Mchango wa Mfuko Umethibitishwa",
				"Mchango wako wa TZS "+contrib.Amount.StringFixed(2)+" kwenye \""+fund.Name+"\" umethibitishwa.",
			)
		}
	}

	return c.JSON(fiber.Map{"message": "Mchango umethibitishwa.", "data": contrib})
}

// RejectContribution declines a pending contribution (balance untouched).
// Mweka Hazina only (route guard).
// POST /api/v1/groups/:id/social-funds/:fundId/contributions/:cid/reject
func (h *SocialFundHandler) RejectContribution(c *fiber.Ctx) error {
	contrib, _, err := h.loadContribution(c)
	if contrib == nil {
		return err
	}
	if contrib.Status != models.SFCPending {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Mchango huu umeshashughulikiwa"})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Sababu ya kukataa inahitajika"})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	contrib.Status = models.SFCRejected
	contrib.RejectionReason = &req.Reason
	contrib.VerifiedBy = &userID
	contrib.VerifiedAt = &now
	if err := database.DB.Save(contrib).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	services.LogAudit(c, &userID, models.AuditReject, "social_fund_contributions", &contrib.ID,
		map[string]interface{}{"status": models.SFCPending},
		map[string]interface{}{"status": models.SFCRejected, "reason": req.Reason},
	)

	return c.JSON(fiber.Map{"message": "Mchango umekataliwa.", "data": contrib})
}

// ---------- helpers ----------

func (h *SocialFundHandler) loadFund(c *fiber.Ctx) (*models.SocialFund, error) {
	var fund models.SocialFund
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("fundId"), c.Params("id")).
		First(&fund).Error; err != nil {
		return nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mfuko haujapatikana"})
	}
	return &fund, nil
}

func (h *SocialFundHandler) loadContribution(c *fiber.Ctx) (*models.SocialFundContribution, *models.SocialFund, error) {
	fund, err := h.loadFund(c)
	if err != nil {
		return nil, nil, err
	}
	var contrib models.SocialFundContribution
	if err := database.DB.Where("id = ? AND fund_id = ?", c.Params("cid"), fund.ID).
		First(&contrib).Error; err != nil {
		return nil, nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mchango haujapatikana"})
	}
	return &contrib, fund, nil
}
