package handlers

import (
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

// OnboardingHandler serves the lightweight resume/complete endpoints behind
// the mwenyekiti setup wizard and the new-member app tour. The wizard steps
// themselves reuse the existing contribution-settings / payment-methods /
// member-creation endpoints — no business logic is duplicated here.
type OnboardingHandler struct{}

func NewOnboardingHandler() *OnboardingHandler { return &OnboardingHandler{} }

// onboardingStatus is the wizard resume payload: which setup steps are done
// so a mwenyekiti who left partway through resumes at the right place.
type onboardingStatus struct {
	OnboardingCompleted bool            `json:"onboarding_completed"`
	Steps               onboardingSteps `json:"steps"`
}

type onboardingSteps struct {
	// ContributionProposed: a settings proposal is awaiting katibu approval.
	ContributionProposed bool `json:"contribution_proposed"`
	// ContributionApproved: approved settings exist (due date or fixed amount set).
	ContributionApproved bool `json:"contribution_approved"`
	PaymentMethodAdded   bool `json:"payment_method_added"`
	MembersAdded         bool `json:"members_added"`
}

// currentGroupOr404 loads the current-group row for the :id param; ok=false
// means the caller should answer 404 (foreign id or missing row).
func currentGroupOr404(c *fiber.Ctx) (*models.Group, bool) {
	if ok, err := database.IsCurrentGroup(c.Params("id")); err != nil || !ok {
		return nil, false
	}
	var g models.Group
	if err := database.DB.First(&g, "id = ?", c.Params("id")).Error; err != nil {
		return nil, false
	}
	return &g, true
}

// GET /api/v1/groups/:id/onboarding-status — any authenticated user.
func (h *OnboardingHandler) Status(c *fiber.Ctx) error {
	g, found := currentGroupOr404(c)
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	st := onboardingStatus{OnboardingCompleted: g.OnboardingCompleted}
	st.Steps.ContributionProposed = pendingProposalOfKind(g.ID, models.ProposalKindContribution) != nil
	st.Steps.ContributionApproved = g.ContributionDueDate != nil || g.FixedContributionAmount != nil

	var pmCount int64
	database.DB.Model(&models.PaymentMethod{}).
		Where("group_id = ? AND deleted_at IS NULL", g.ID).
		Count(&pmCount)
	st.Steps.PaymentMethodAdded = pmCount > 0

	var memberCount int64
	database.DB.Model(&models.Member{}).
		Where("deleted_at IS NULL").
		Count(&memberCount)
	st.Steps.MembersAdded = memberCount > 0

	return c.JSON(fiber.Map{"data": st})
}

// PATCH /api/v1/groups/:id/onboarding-complete — Mwenyekiti only (route
// guard). Marks the wizard finished or skipped so it stops auto-opening.
func (h *OnboardingHandler) Complete(c *fiber.Ctx) error {
	g, found := currentGroupOr404(c)
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	if !g.OnboardingCompleted {
		if err := database.DB.Model(&models.Group{}).
			Where("id = ?", g.ID).
			Update("onboarding_completed", true).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi hali ya setup"})
		}
		userID := middleware.GetUserID(c)
		services.LogAudit(c, &userID, models.AuditUpdate, "groups", &g.ID, nil, map[string]interface{}{
			"onboarding_completed": true,
		})
	}

	return c.JSON(fiber.Map{
		"message": "Setup ya kikundi imekamilika.",
		"data":    fiber.Map{"onboarding_completed": true},
	})
}

// PATCH /api/v1/members/:id/onboarding-seen — the member themself (or admin):
// marks the first-login app tour as seen for the member's linked user so it
// never auto-shows again.
func (h *OnboardingHandler) MarkOnboardingSeen(c *fiber.Ctx) error {
	var m models.Member
	if err := database.DB.
		Where("id = ? AND deleted_at IS NULL", c.Params("id")).
		First(&m).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hakupatikana"})
	}

	callerID := middleware.GetUserID(c)
	if m.UserID == nil || *m.UserID != callerID {
		if middleware.GetUserRole(c) != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Hunaruhusiwa kubadilisha hali ya mwanachama mwingine"})
		}
	}

	if err := database.DB.Model(&models.User{}).
		Where("id = ?", *m.UserID).
		Update("onboarding_seen", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi hali ya onboarding"})
	}

	return c.JSON(fiber.Map{
		"message": "Onboarding imekamilika.",
		"data":    fiber.Map{"onboarding_seen": true},
	})
}
