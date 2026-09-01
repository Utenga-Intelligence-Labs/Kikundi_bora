package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	unreadOnly := c.Query("unread")

	query := database.DB.Where("user_id = ?", userID)
	if unreadOnly == "true" {
		query = query.Where("read_at IS NULL")
	}

	var total int64
	query.Model(&models.Notification{}).Count(&total)

	var notifs []models.Notification
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&notifs)

	var unreadCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND read_at IS NULL", userID).Count(&unreadCount)

	return c.JSON(fiber.Map{
		"data":        notifs,
		"total":       total,
		"unread":      unreadCount,
		"page":        pq.Page,
		"limit":       pq.Limit,
	})
}

func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.NotificationReadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	now := time.Now()
	if len(req.IDs) == 0 {
		database.DB.Model(&models.Notification{}).
			Where("user_id = ? AND read_at IS NULL", userID).
			Update("read_at", now)
		return c.JSON(fiber.Map{"message": "Arifa zote zimewekwa zimesomwa"})
	}

	database.DB.Model(&models.Notification{}).
		Where("id IN ? AND user_id = ?", req.IDs, userID).
		Update("read_at", now)

	return c.JSON(fiber.Map{"message": "Arifa zimewekwa zimesomwa"})
}

// MarkAllRead marks every unread notification of the user as read.
// POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	now := time.Now()
	res := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now)

	return c.JSON(fiber.Map{"message": "Arifa zote zimewekwa zimesomwa", "updated": res.RowsAffected})
}
