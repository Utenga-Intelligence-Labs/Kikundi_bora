package middleware

import (
	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

// RequireMember ensures the authenticated user has a linked member row.
// Admin bypasses. Returns 403 if no member found.
func RequireMember() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := GetUserRole(c)
		if role == models.RoleAdmin {
			return c.Next()
		}

		userID := GetUserID(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Huna ruhusa ya kufikia rasilimali hii",
			})
		}

		var member models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
			First(&member).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Lazima uwe mwanachama wa kikundi kufikia hii",
			})
		}

		c.Locals("member_id", member.ID)
		return c.Next()
	}
}

// RequireLeadership ensures the authenticated user holds at least one of the specified leadership roles.
// Admin bypasses. Leadership is checked via leadership_positions table (not user.role).
func RequireLeadership(roles ...models.LeadershipRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := GetUserRole(c)
		if role == models.RoleAdmin {
			return c.Next()
		}

		userID := GetUserID(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Huna ruhusa ya kufikia rasilimali hii",
			})
		}

		// Find member linked to user
		var member models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
			First(&member).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Lazima uwe mwanachama wa kikundi kufikia hii",
			})
		}

		// Check leadership positions
		var count int64
		if err := database.DB.Model(&models.LeadershipPosition{}).
			Where("member_id = ? AND role IN ? AND is_current = TRUE",
				member.ID, roles).
			Count(&count).Error; err != nil || count == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Huna nafasi ya uongozi inayohitajika",
			})
		}

		c.Locals("member_id", member.ID)
		return c.Next()
	}
}

// GetMemberID returns the member_id from Fiber locals (set by RequireMember/RequireLeadership).
func GetMemberID(c *fiber.Ctx) string {
	memberID, ok := c.Locals("member_id").(string)
	if !ok {
		return ""
	}
	return memberID
}
