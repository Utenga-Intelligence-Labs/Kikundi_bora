package handlers

import (
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

type AnnouncementHandler struct{}

func NewAnnouncementHandler() *AnnouncementHandler {
	return &AnnouncementHandler{}
}

// Broadcast sends an announcement to all members (leadership only)
// POST /api/v1/uongozi/announcements
func (h *AnnouncementHandler) Broadcast(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if req.Title == "" || req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Kichwa na ujumbe vinahitajika",
		})
	}

	// Find active members only (L-6: deactivated members must not receive
	// announcements; GORM auto-filters soft-deleted)
	var members []models.Member
	database.DB.Where("is_active = TRUE").Find(&members)

	// Send notification to each member's linked user
	count := 0
	for _, member := range members {
		if member.UserID != nil && *member.UserID != "" {
			services.NotifyUser(*member.UserID, models.NotifSystem, req.Title, req.Message)
			count++
		}
	}

	services.LogAudit(c, &userID, models.AuditCreate, "announcements", nil, nil, map[string]interface{}{
		"title": req.Title, "message": req.Message, "recipients": count,
	})

	return c.JSON(fiber.Map{
		"message":    "Tangazo limetumwa",
		"recipients": count,
	})
}
