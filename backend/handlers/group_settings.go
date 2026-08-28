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

type GroupSettingsHandler struct{}

func NewGroupSettingsHandler() *GroupSettingsHandler {
	return &GroupSettingsHandler{}
}

func loadPendingProposal(groupID string) *models.GroupSettingProposal {
	var p models.GroupSettingProposal
	if err := database.DB.
		Preload("Proposer", func(db *gorm.DB) *gorm.DB { return db.Select("id, name, role") }).
		Where("group_id = ? AND status = ?", groupID, models.ProposalPending).
		Order("created_at DESC").
		First(&p).Error; err != nil {
		return nil
	}
	return &p
}

// GET /api/v1/groups/current — convenience for the frontend (single-group
// deployment): resolves the group id, then returns the settings payload.
func (h *GroupSettingsHandler) GetCurrent(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	return h.settingsResponse(c, g)
}

// GET /api/v1/groups/:id/contribution-settings — any authenticated user
// (members need read-only settings for the submission form and banners).
func (h *GroupSettingsHandler) GetSettings(c *fiber.Ctx) error {
	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}
	return h.settingsResponse(c, &g)
}

func (h *GroupSettingsHandler) settingsResponse(c *fiber.Ctx, g *models.Group) error {
	nextDue := ""
	if g.ContributionDueDate != nil && g.ContributionInterval != "" {
		if due, ok := services.NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, time.Now()); ok {
			nextDue = due.Format("2006-01-02")
		}
	}
	return c.JSON(fiber.Map{
		"data":              g,
		"pending_proposal":  loadPendingProposal(g.ID),
		"next_due_date":     nextDue,
	})
}

type proposeRequest struct {
	ContributionInterval    string           `json:"contribution_interval"`
	ContributionDueDate     string           `json:"contribution_due_date"`
	FixedContributionAmount *decimal.Decimal `json:"fixed_contribution_amount"`
}

// POST /api/v1/groups/:id/contribution-settings/propose — Mwenyekiti only.
func (h *GroupSettingsHandler) Propose(c *fiber.Ctx) error {
	var req proposeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	if err := services.ValidateProposalSpec(req.ContributionInterval, req.ContributionDueDate, req.FixedContributionAmount); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	// Only one PENDING proposal at a time
	if existing := loadPendingProposal(g.ID); existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Kuna pendekezo lililopo bado halijajibiwa. Katibu lazima alipe Wakati wa kwanza kabla ya pendekezo jipya.",
			"data":    existing,
		})
	}

	userID := middleware.GetUserID(c)
	proposal := models.GroupSettingProposal{
		GroupID:                 g.ID,
		ContributionInterval:    models.ContributionInterval(req.ContributionInterval),
		ContributionDueDate:     &req.ContributionDueDate,
		FixedContributionAmount: req.FixedContributionAmount,
		Status:                  models.ProposalPending,
		ProposedBy:              userID,
	}
	if err := database.DB.Create(&proposal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutuma pendekezo"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "group_setting_proposals", &proposal.ID, nil, map[string]interface{}{
		"group_id": g.ID, "interval": req.ContributionInterval, "due_date": req.ContributionDueDate, "amount": req.FixedContributionAmount,
	})

	services.NotifyRole(models.RoleSecretary, models.NotifSystem,
		"Pendekezo la Mipangilio ya Michango",
		"Mwenyekiti amependekeza kipindi cha michango ("+req.ContributionInterval+"). Subiri uthibitisho wako.",
		"",
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pendekezo limetumwa kwa Katibu kwa idhini.",
		"data":    proposal,
	})
}

// POST /api/v1/groups/:id/contribution-settings/approve — Katibu only.
// Applies the pending proposal to the group settings.
func (h *GroupSettingsHandler) Approve(c *fiber.Ctx) error {
	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	proposal := loadPendingProposal(g.ID)
	if proposal == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Hakuna pendekezo lililosubiri"})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()

	// Apply to group — the ONLY path that mutates group contribution settings
	g.ContributionInterval = proposal.ContributionInterval
	g.ContributionDueDate = proposal.ContributionDueDate
	g.FixedContributionAmount = proposal.FixedContributionAmount
	if err := database.DB.Save(&g).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi mipangilio"})
	}

	proposal.Status = models.ProposalApproved
	proposal.ApprovedBy = &userID
	proposal.ReviewedAt = &now
	if err := database.DB.Save(proposal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi pendekezo"})
	}

	services.LogAudit(c, &userID, models.AuditApprove, "group_setting_proposals", &proposal.ID,
		map[string]interface{}{"status": models.ProposalPending},
		map[string]interface{}{"status": models.ProposalApproved, "interval": g.ContributionInterval, "due_date": g.ContributionDueDate, "amount": g.FixedContributionAmount},
	)

	services.NotifyRole(models.RoleChair, models.NotifSystem,
		"Mipangilio ya Michango Imeidhinishwa",
		"Katibu ameidhinisha mipangilio mpya ya michango. Sasa inatumika.",
		"",
	)

	return c.JSON(fiber.Map{"message": "Mipangilio imeidhinishwa na sasa inatumika.", "data": g})
}

// POST /api/v1/groups/:id/contribution-settings/reject — Katibu only.
func (h *GroupSettingsHandler) Reject(c *fiber.Ctx) error {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Sababu ya kukataa inahitajika"})
	}

	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	proposal := loadPendingProposal(g.ID)
	if proposal == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Hakuna pendekezo lililosubiri"})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()

	proposal.Status = models.ProposalRejected
	proposal.ApprovedBy = &userID
	proposal.RejectionReason = &req.Reason
	proposal.ReviewedAt = &now
	if err := database.DB.Save(proposal).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi pendekezo"})
	}

	services.LogAudit(c, &userID, models.AuditReject, "group_setting_proposals", &proposal.ID,
		map[string]interface{}{"status": models.ProposalPending},
		map[string]interface{}{"status": models.ProposalRejected, "reason": req.Reason},
	)

	services.NotifyRole(models.RoleChair, models.NotifSystem,
		"Pendekezo la Mipangilio Limekataliwa",
		"Katibu amekatalia pendekezo la mipangilio ya michango. Sababu: "+req.Reason,
		"",
	)

	return c.JSON(fiber.Map{"message": "Pendekezo limekataliwa.", "data": proposal})
}
