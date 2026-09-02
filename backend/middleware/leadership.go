package middleware

import (
	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

// RequireSelfOrLeadership ensures the authenticated user is the target
// themselves (member row or user id), or holds an elevated role
// (chair/secretary/treasurer/admin), or holds any current leadership
// position (dual plane). RBAC-M02: promotes the fragile handler-level
// checks to middleware so future refactors cannot drop them silently.
func RequireSelfOrLeadership(resolveTarget func(c *fiber.Ctx) (targetMemberID, targetUserID string)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		role := GetUserRole(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Huna ruhusa ya kufikia rasilimali hii",
			})
		}
		if role == models.RoleAdmin {
			return c.Next()
		}
		if role == models.RoleChair || role == models.RoleSecretary || role == models.RoleTreasurer {
			return c.Next()
		}

		targetMemberID, targetUserID := resolveTarget(c)

		var own models.Member
		hasOwn := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
			First(&own).Error == nil
		if hasOwn && own.ID != "" && own.ID == targetMemberID {
			return c.Next()
		}
		if targetUserID != "" && userID == targetUserID {
			return c.Next()
		}
		if hasOwn {
			var n int64
			database.DB.Model(&models.LeadershipPosition{}).
				Where("member_id = ? AND is_current = TRUE", own.ID).
				Count(&n)
			if n > 0 {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Huna ruhusa ya kufikia rasilimali hii",
		})
	}
}
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
