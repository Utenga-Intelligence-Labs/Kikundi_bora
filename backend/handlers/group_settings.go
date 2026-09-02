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
	if ok, err := database.IsCurrentGroup(c.Params("id")); err != nil || !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}
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

	// Per-member current-cycle contribution status: has the requesting user
	// (if they hold a member row) already submitted their AKIBA contribution
	// for the currently open cycle?
	var myContribution *myCycleStatus
	if userID := middleware.GetUserID(c); userID != "" {
		var member models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
			First(&member).Error; err == nil {
			myContribution = h.cycleStatusForMember(g, member.ID, time.Now())
		}
	}

	return c.JSON(fiber.Map{
		"data":              g,
		"pending_proposal":  loadPendingProposal(g.ID),
		"next_due_date":     nextDue,
		"my_contribution":   myContribution,
	})
}

type myCycleStatus struct {
	Status        string `json:"status"` // "none" | "pending" | "confirmed"
	PeriodDueDate string `json:"period_due_date,omitempty"`
}

// cycleStatusForMember checks BOTH contribution stores inside the current
// cycle window (prevDue, due]:
//   - member_contributions: self-submitted (Weka Mchango), created_at
//     CONFIRMED → "confirmed"; only PENDING_VERIFICATION → "pending"
//   - contributions: treasurer-recorded (Pokea Mchango), PAID paid_at → "confirmed"
func (h *GroupSettingsHandler) cycleStatusForMember(g *models.Group, memberID string, now time.Time) *myCycleStatus {
	start, end, ok := services.ContributionCycleWindow(g, now)
	if !ok {
		return nil
	}
	dueStr := end.Format("2006-01-02")

	var mcConfirmed, legacyConfirmed int64
	database.DB.Model(&models.MemberContribution{}).
		Where("member_id = ? AND contribution_type = ? AND status = ? AND created_at > ? AND created_at <= ?",
			memberID, models.ContributionAkiba, models.ContributionConfirmed, start, end).
		Count(&mcConfirmed)
	database.DB.Model(&models.Contribution{}).
		Where("member_id = ? AND status = ? AND paid_at > ? AND paid_at <= ?",
			memberID, "PAID", start, end).
		Count(&legacyConfirmed)
	confirmed := mcConfirmed + legacyConfirmed

	if confirmed > 0 {
		return &myCycleStatus{Status: "confirmed", PeriodDueDate: dueStr}
	}

	var pending int64
	database.DB.Model(&models.MemberContribution{}).
		Where("member_id = ? AND contribution_type = ? AND status = ? AND created_at > ? AND created_at <= ?",
			memberID, models.ContributionAkiba, models.ContributionPending, start, end).
		Count(&pending)
	if pending > 0 {
		return &myCycleStatus{Status: "pending", PeriodDueDate: dueStr}
	}

	return &myCycleStatus{Status: "none", PeriodDueDate: dueStr}
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

	if ok, err := database.IsCurrentGroup(c.Params("id")); err != nil || !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
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
	if ok, err := database.IsCurrentGroup(c.Params("id")); err != nil || !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
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

	if ok, err := database.IsCurrentGroup(c.Params("id")); err != nil || !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
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
