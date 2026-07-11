package middleware

import (
	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

// RequirePosition checks if the authenticated user holds at least one of the specified positions.
// Admin role bypasses this check.
func RequirePosition(positions ...models.PositionType) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(models.Role)
		if ok && role == models.RoleAdmin {
			return c.Next()
		}

		userID := GetUserID(c)

		var count int64
		database.DB.Model(&models.UserPosition{}).
			Where("user_id = ? AND position_type IN ? AND is_active = TRUE", userID, positions).
			Count(&count)

		if count == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Huna nafasi inayohitajika kufanya hili",
			})
		}

		return c.Next()
	}
}

// HasPosition checks if a user holds a specific position (non-middleware helper).
func HasPosition(userID string, position models.PositionType) bool {
	var count int64
	database.DB.Model(&models.UserPosition{}).
		Where("user_id = ? AND position_type = ? AND is_active = TRUE", userID, position).
		Count(&count)
	return count > 0
}

// GetUserPositions returns all active positions for a user.
func GetUserPositions(userID string) []models.PositionType {
	var positions []models.UserPosition
	database.DB.Where("user_id = ? AND is_active = TRUE", userID).Find(&positions)
	result := make([]models.PositionType, len(positions))
	for i, p := range positions {
		result[i] = p.PositionType
	}
	return result
}
