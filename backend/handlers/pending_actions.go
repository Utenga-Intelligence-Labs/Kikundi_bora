package handlers

import (
	"encoding/json"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type PendingActionHandler struct{}

func NewPendingActionHandler() *PendingActionHandler {
	return &PendingActionHandler{}
}

// List returns pending actions, filtered by status.
func (h *PendingActionHandler) List(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	status := c.Query("status", models.ActionStatusPending)
	actionType := c.Query("type")

	query := database.DB.
		Preload("Requester", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Preload("Approver", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}

	var total int64
	query.Model(&models.PendingAction{}).Count(&total)

	var actions []models.PendingAction
	query.Order("created_at DESC").
		Offset(pq.GetOffset()).
		Limit(pq.Limit).
		Find(&actions)

	return c.JSON(fiber.Map{
		"data":  actions,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// Get returns a single pending action by ID.
func (h *PendingActionHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")

	var action models.PendingAction
	if err := database.DB.
		Preload("Requester", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Preload("Approver", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		First(&action, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kitendo hakijapatikana"})
	}

	return c.JSON(action)
}

// Approve approves a pending action and executes it.
func (h *PendingActionHandler) Approve(c *fiber.Ctx) error {
	id := c.Params("id")
	approverID := middleware.GetUserID(c)

	var req struct {
		Remarks string `json:"remarks"`
	}
	c.BodyParser(&req)

	var action models.PendingAction
	if err := database.DB.First(&action, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kitendo hakijapatikana"})
	}

	if action.Status != models.ActionStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kitendo tayari kimeshughulikiwa"})
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      models.ActionStatusApproved,
		"approved_by": approverID,
		"remarks":     req.Remarks,
		"resolved_at": now,
	}
	if err := database.DB.Model(&action).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	// Execute the action based on type
	if err := h.executeAction(action); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutekeleza: " + err.Error()})
	}

	services.LogAudit(c, &approverID, models.AuditApprove, "pending_actions", &action.ID, nil, map[string]interface{}{
		"action_type": action.ActionType, "status": "APPROVED",
	})

	// Notify the requester
	services.NotifyUser(action.RequestedBy, models.NotifSystem,
		"Kitendo Kimedhinishwa",
		"Kitendo chako cha "+action.ActionType+" kimeidhinishwa na msimamizi.",
	)

	return c.JSON(fiber.Map{"message": "Kitendo kimeidhinishwa na kutekelezwa"})
}

// Reject rejects a pending action.
func (h *PendingActionHandler) Reject(c *fiber.Ctx) error {
	id := c.Params("id")
	approverID := middleware.GetUserID(c)

	var req struct {
		Remarks string `json:"remarks"`
	}
	c.BodyParser(&req)

	var action models.PendingAction
	if err := database.DB.First(&action, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kitendo hakijapatikana"})
	}

	if action.Status != models.ActionStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kitendo tayari kimeshughulikiwa"})
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      models.ActionStatusRejected,
		"approved_by": approverID,
		"remarks":     req.Remarks,
		"resolved_at": now,
	}
	if err := database.DB.Model(&action).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	services.LogAudit(c, &approverID, models.AuditReject, "pending_actions", &action.ID, nil, map[string]interface{}{
		"action_type": action.ActionType, "status": "REJECTED", "reason": req.Remarks,
	})

	// Notify the requester
	services.NotifyUser(action.RequestedBy, models.NotifSystem,
		"Kitendo Kimetenguliwa",
		"Kitendo chako cha "+action.ActionType+" kimetenguliwa. Sababu: "+req.Remarks,
	)

	return c.JSON(fiber.Map{"message": "Kitendo kimetenguliwa"})
}

// executeAction runs the business logic for an approved action.
func (h *PendingActionHandler) executeAction(action models.PendingAction) error {
	switch action.ActionType {
	case models.ActionContributionEdit:
		return h.executeContributionEdit(action.Payload)
	case models.ActionWelfareCreate:
		return h.executeWelfareCreate(action.Payload)
	default:
		return nil
	}
}

func (h *PendingActionHandler) executeContributionEdit(payload json.RawMessage) error {
	var data struct {
		ContributionID string  `json:"contribution_id"`
		NewAmount      float64 `json:"new_amount"`
		Reason         string  `json:"reason"`
		EditedBy       string  `json:"edited_by"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}

	var contrib models.Contribution
	if err := database.DB.First(&contrib, "id = ?", data.ContributionID).Error; err != nil {
		return err
	}

	oldAmount := contrib.Amount
	tx := database.DB.Begin()

	contrib.Amount = data.NewAmount
	if err := tx.Save(&contrib).Error; err != nil {
		tx.Rollback()
		return err
	}

	edit := models.ContributionEdit{
		ContributionID: contrib.ID,
		EditedBy:       data.EditedBy,
		OldAmount:      oldAmount,
		NewAmount:      data.NewAmount,
		Reason:         data.Reason,
	}
	if err := tx.Create(&edit).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (h *PendingActionHandler) executeWelfareCreate(payload json.RawMessage) error {
	// Welfare creation execution — the welfare handler already handles this
	// This is a placeholder for the approval flow integration
	return nil
}
