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
)

type MemberContributionHandler struct{}

func NewMemberContributionHandler() *MemberContributionHandler {
	return &MemberContributionHandler{}
}

// Submit handles member contribution submission
// POST /api/v1/michango
func (h *MemberContributionHandler) Submit(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Find member linked to user
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Lazima uwe mwanachama wa kikundi kuweka mchango",
		})
	}

	// Parse request
	var req struct {
		ContributionType string          `json:"contribution_type" validate:"required,oneof=AKIBA MFUKO_WA_KIJAMII"`
		PeriodLabel      string          `json:"period_label" validate:"required,max=30"`
		Amount           decimal.Decimal `json:"amount" validate:"required,gt=0"`
		ProofImageURL    string          `json:"proof_image_url" validate:"omitempty,url,max=500"`
		ProofMessage     string          `json:"proof_message" validate:"omitempty,max=1000"`
		WelfareEventID   string          `json:"welfare_event_id" validate:"omitempty,uuid"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	// Validate
	if req.ContributionType == "" || req.PeriodLabel == "" || req.Amount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Aina ya mchango, kipindi, na kiasi vinahitajika",
		})
	}

	if req.ProofImageURL == "" && req.ProofMessage == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Lazima uweke picha ya uthibitisho au ujumbe wa muamala",
		})
	}

	if req.ContributionType != "AKIBA" && req.ContributionType != "MFUKO_WA_KIJAMII" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Aina ya mchango si sahihi. Chagua AKIBA au MFUKO_WA_KIJAMII",
		})
	}

	// Fixed contribution amount enforcement (group settings) — AKIBA is the
	// periodic contribution governed by the interval setting. MFUKO_WA_KIJAMII
	// amounts follow the welfare event, so they are not fixed.
	if req.ContributionType == "AKIBA" {
		var group models.Group
		if err := database.DB.First(&group).Error; err == nil {
			if err := services.CheckFixedContributionAmount(group.FixedContributionAmount, req.Amount); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
			}
		}
	}

	// MFUKO_WA_KIJAMII must reference an approved, member-funded welfare event
	var welfareEventID *string
	if req.ContributionType == "MFUKO_WA_KIJAMII" {
		if req.WelfareEventID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Lazima uchague mfuko wa kijamii unaochangia",
			})
		}
		var event models.WelfareEvent
		if err := database.DB.
			Where("id = ? AND status = ? AND funding_source IN ?",
				req.WelfareEventID, models.WelfareApproved,
				[]models.WelfareFundingSource{models.FundMemberContribution, models.FundBoth}).
			First(&event).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Mfuko wa kijamii haujapatikana au haukuidhinishwa kuchangia",
			})
		}
		welfareEventID = &req.WelfareEventID
	} else if req.WelfareEventID != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Kuchagua mfuko kunatumika kwa MFUKO_WA_KIJAMII tu",
		})
	}

	// Create contribution
	contribution := models.MemberContribution{
		MemberID:         member.ID,
		ContributionType: models.ContributionType(req.ContributionType),
		PeriodLabel:      req.PeriodLabel,
		Amount:           req.Amount,
		ProofImageURL:    req.ProofImageURL,
		ProofMessage:     req.ProofMessage,
		WelfareEventID:   welfareEventID,
		Status:           models.ContributionPending,
	}

	if err := database.DB.Create(&contribution).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kuwasilisha mchango",
		})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "member_contributions", &contribution.ID, nil, map[string]interface{}{
		"member_id": member.ID, "amount": req.Amount, "type": req.ContributionType, "status": "PENDING_VERIFICATION",
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Mchango umewasilishwa. Inasubiri uthibitisho.",
		"data":    contribution,
	})
}

// MyContributions returns member's own contributions
// GET /api/v1/michango/mine
func (h *MemberContributionHandler) MyContributions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Lazima uwe mwanachama wa kikundi",
		})
	}

	var contributions []models.MemberContribution
	database.DB.Preload("WelfareEvent", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, event_type, description, status")
	}).
		Where("member_id = ?", member.ID).
		Order("created_at DESC").
		Find(&contributions)

	return c.JSON(fiber.Map{
		"data":  contributions,
		"total": len(contributions),
	})
}

// AllContributions returns all contributions (leadership only)
// GET /api/v1/michango
func (h *MemberContributionHandler) AllContributions(c *fiber.Ctx) error {
	var contributions []models.MemberContribution
	database.DB.Preload("Member", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, member_no, full_name, phone")
	}).
		Preload("WelfareEvent", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, event_type, description, status")
		}).
		Order("created_at DESC").
		Find(&contributions)

	return c.JSON(fiber.Map{
		"data":  contributions,
		"total": len(contributions),
	})
}

// PendingContributions returns contributions awaiting verification (leadership only)
// GET /api/v1/michango/pending
func (h *MemberContributionHandler) PendingContributions(c *fiber.Ctx) error {
	var contributions []models.MemberContribution
	database.DB.Where("status = ?", models.ContributionPending).
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Order("created_at ASC").
		Find(&contributions)

	return c.JSON(fiber.Map{
		"data":  contributions,
		"total": len(contributions),
	})
}

// Confirm verifies a contribution
// POST /api/v1/michango/:id/confirm
func (h *MemberContributionHandler) Confirm(c *fiber.Ctx) error {
	contribID := c.Params("id")
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	// Find reviewer's member ID
	var reviewerMember models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&reviewerMember).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Lazima uwe mwanachama wa kikundi kuthibitisha michango",
		})
	}

	// Find contribution
	var contribution models.MemberContribution
	if err := database.DB.Preload("Member").First(&contribution, "id = ?", contribID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Mchango haujapatikana",
		})
	}

	if contribution.Status != models.ContributionPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Mchango huu tayari umethibitishwa au kukataliwa",
		})
	}

	// Check if reviewer has authority based on contribution type
	if contribution.ContributionType == models.ContributionAkiba && role != models.RoleTreasurer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Mweka Hazina pekee anaweza kuthibitisha AKIBA",
		})
	}

	if contribution.ContributionType == models.ContributionMfuko && role != models.RoleChair {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Mwenyekiti pekee anaweza kuthibitisha MFUKO_WA_KIJAMII",
		})
	}

	// Update contribution
	now := time.Now()
	reviewedBy := reviewerMember.ID
	contribution.Status = models.ContributionConfirmed
	contribution.ReviewedByMemberID = &reviewedBy
	contribution.ReviewedAt = now

	if err := database.DB.Save(&contribution).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kuthibitisha mchango",
		})
	}

	services.LogAudit(c, &userID, models.AuditApprove, "member_contributions", &contribution.ID, nil, map[string]interface{}{
		"status": "CONFIRMED", "reviewed_by": reviewerMember.ID,
	})

	// Auto-post into the double-entry ledger (best-effort — a ledger
	// failure is logged but never fails the confirmation). Only
	// MFUKO_WA_KIJAMII is skipped: it is earmarked welfare money, not
	// savings. Empty/legacy types post as savings like AKIBA.
	if contribution.ContributionType != models.ContributionMfuko && contribution.Member != nil {
		if err := services.PostContribution(contribution.Member.MemberNo, contribution.Amount, now, userID,
			fmt.Sprintf("Mchango AKIBA %s %s", contribution.Member.MemberNo, contribution.PeriodLabel)); err != nil {
			log.Printf("WARN: ledger auto-post michango %s: %v", contribution.ID, err)
		}
	}

	// Notify member
	if contribution.Member != nil && contribution.Member.UserID != nil {
		services.NotifyUser(*contribution.Member.UserID, models.NotifSystem,
			"Mchango Wako Umeithibitishwa",
			fmt.Sprintf("Mchango wako wa TZS %s umethibitishwa.", contribution.Amount.StringFixed(0)),
		)
	}

	return c.JSON(fiber.Map{
		"message": "Mchango umethibitishwa",
		"data":    contribution,
	})
}

// Reject rejects a contribution
// POST /api/v1/michango/:id/reject
func (h *MemberContributionHandler) Reject(c *fiber.Ctx) error {
	contribID := c.Params("id")
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Sababu ya kukataa inahitajika",
		})
	}

	// Find reviewer's member ID
	var reviewerMember models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&reviewerMember).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Lazima uwe mwanachama wa kikundi kukataa mchango",
		})
	}

	// Find contribution
	var contribution models.MemberContribution
	if err := database.DB.Preload("Member").First(&contribution, "id = ?", contribID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Mchango haujapatikana",
		})
	}

	if contribution.Status != models.ContributionPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Mchango huu tayari umethibitishwa au kukataliwa",
		})
	}

	// Check if reviewer has authority based on contribution type
	if contribution.ContributionType == models.ContributionAkiba && role != models.RoleTreasurer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Mweka Hazina pekee anaweza kukataa AKIBA",
		})
	}

	if contribution.ContributionType == models.ContributionMfuko && role != models.RoleChair {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Mwenyekiti pekee anaweza kukataa MFUKO_WA_KIJAMII",
		})
	}

	// Update contribution
	now := time.Now()
	reviewedBy := reviewerMember.ID
	contribution.Status = models.ContributionRejected
	contribution.ReviewedByMemberID = &reviewedBy
	contribution.ReviewReason = req.Reason
	contribution.ReviewedAt = now

	if err := database.DB.Save(&contribution).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kukataa mchango",
		})
	}

	services.LogAudit(c, &userID, models.AuditReject, "member_contributions", &contribution.ID, nil, map[string]interface{}{
		"status": "REJECTED", "reason": req.Reason, "reviewed_by": reviewerMember.ID,
	})

	// Notify member
	if contribution.Member != nil && contribution.Member.UserID != nil {
		services.NotifyUser(*contribution.Member.UserID, models.NotifSystem,
			"Mchango Wako Umekataliwa",
			fmt.Sprintf("Mchango wako wa TZS %s umekataliwa. Sababu: %s", contribution.Amount.StringFixed(0), req.Reason),
		)
	}

	return c.JSON(fiber.Map{
		"message": "Mchango umekataliwa",
		"data":    contribution,
	})
}

// MembersSummary returns all active members with their latest contribution status.
// Used by Mwenyekiti/Katibu (read-only) and Hazina (with approve/decline actions).
// GET /api/v1/michango/members-summary
func (h *MemberContributionHandler) MembersSummary(c *fiber.Ctx) error {
	type MemberRow struct {
		MemberID    string  `json:"member_id"`
		FullName    string  `json:"full_name"`
		MemberNo    string  `json:"member_no"`
		Phone       string  `json:"phone"`
		Status      string  `json:"status"` // PENDING_VERIFICATION / CONFIRMED / REJECTED / HAJACHANGIA
		Amount      float64 `json:"amount"`
		PeriodLabel string  `json:"period_label"`
		ContributionType string `json:"contribution_type"`
		ContributionID   string `json:"contribution_id"`
		ProofImageURL    string `json:"proof_image_url"`
		ProofMessage     string `json:"proof_message"`
		SubmittedAt      string `json:"submitted_at"`
	}

	var rows []MemberRow

	// Left join: all active members with their latest contribution.
	// FIX: the lateral subquery previously filtered on member_contributions.deleted_at,
	// a column that does not exist on that table (no soft-delete on member_contributions)
	// — the whole query failed with a 500. The filter is removed.
	database.DB.Raw(`
		SELECT
			m.id AS member_id,
			m.full_name,
			m.member_no,
			m.phone,
			COALESCE(mc.status, 'HAJACHANGIA') AS status,
			COALESCE(mc.amount, 0) AS amount,
			COALESCE(mc.period_label, '') AS period_label,
			COALESCE(mc.contribution_type, '') AS contribution_type,
			COALESCE(mc.id::text, '') AS contribution_id,
			COALESCE(mc.proof_image_url, '') AS proof_image_url,
			COALESCE(mc.proof_message, '') AS proof_message,
			COALESCE(to_char(mc.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS submitted_at
		FROM members m
		LEFT JOIN LATERAL (
			SELECT * FROM member_contributions
			WHERE member_id = m.id
			ORDER BY created_at DESC LIMIT 1
		) mc ON TRUE
		WHERE m.deleted_at IS NULL AND m.is_active = TRUE
		ORDER BY
			CASE COALESCE(mc.status, 'HAJACHANGIA')
				WHEN 'PENDING_VERIFICATION' THEN 1
				WHEN 'HAJACHANGIA' THEN 2
				WHEN 'CONFIRMED' THEN 3
				WHEN 'REJECTED' THEN 4
				ELSE 5
			END,
			m.full_name ASC
	`).Scan(&rows)

	return c.JSON(fiber.Map{
		"data":  rows,
		"total": len(rows),
	})
}
