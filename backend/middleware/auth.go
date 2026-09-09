package middleware

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	tokenStr := ""
	if strings.HasPrefix(header, "Bearer ") {
		tokenStr = strings.TrimPrefix(header, "Bearer ")
	} else if qt := c.Query("token"); qt != "" && strings.HasPrefix(c.Path(), "/uploads/") {
		// <img>/<a> tags cannot send an Authorization header — allow the
		// token via query parameter ONLY on the uploads static route
		// (AUTH-02: previously accepted on every route, leaking JWTs into
		// access logs / Referer for arbitrary endpoints).
		tokenStr = qt
	}
	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token ya ukaguzi haijapatikana",
		})
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token si sahihi au imeisha muda",
		})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token haiwezi kusomwa",
		})
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token haina kitambulisho cha mtumiaji",
		})
	}
	_, ok = claims["role"].(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token haina jukumu la mtumiaji",
		})
	}

	// AUTH-03: single session lookup replaces the two separate Count
	// queries — enforces revocation AND server-side expiry (previously
	// only the JWT exp was trusted), and slides LastActiveAt forward.
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenStr)))
	var session models.UserSession
	if err := database.DB.
		Where("user_id = ? AND token_hash = ?", userID, tokenHash).
		First(&session).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Kipindi kimeisha. Tafadhali ingia tena.",
		})
	}
	if session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Kipindi kimeisha. Tafadhali ingia tena.",
		})
	}
	database.DB.Model(&session).Update("last_active_at", time.Now())

	// Verify role from database to prevent stale role from JWT
	var user models.User
	if err := database.DB.Select("role").Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Mtumiaji hajapatikana",
		})
	}

	c.Locals("user_id", userID)
	c.Locals("role", user.Role)

	return c.Next()
}

func RequireRoles(roles ...models.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(models.Role)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Huna ruhusa ya kufanya hili",
			})
		}

		// Admin is a distinct system-level role — NOT a superset of
		// leadership/member roles. It passes ONLY when explicitly listed
		// (e.g. RequireRoles(RoleChair, RoleAdmin) for audit-logs /
		// notification-settings, RequireRoles(RoleAdmin) for /admin/*).
		// Group-operational endpoints (contribution-settings propose/
		// approve, payment-methods manage, michango, etc.) list only
		// leadership roles, so admin correctly gets 403 there.

		for _, r := range roles {
			if role == r {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Jukumu lako haliruhusiwi. Lazima uwe " + string(roles[0]),
		})
	}
}

// RequireLoanCommitteeMember allows access if the user is an eligible committee voter:
// - Leadership role (chair / secretary / treasurer)
// - Active leadership position (CHAIRPERSON, SECRETARY, TREASURER)
// - Active appointed committee member in loan_committee_members
// Admin is a system-level role, NOT a committee voter — it must use
// explicit admin endpoints, so it is NOT auto-allowed here.
// Matches handlers.LoanCommitteeHandler.isEligibleCommitteeVoter.
func RequireLoanCommitteeMember() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(models.Role)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Huna ruhusa ya kufanya hili",
			})
		}

		// Leadership roles may access (same as review eligibility).
		// NOTE: admin is NOT included — it is a system-level role, not a
		// group member or committee voter.
		if role == models.RoleChair || role == models.RoleSecretary || role == models.RoleTreasurer {
			return c.Next()
		}

		userID := GetUserID(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Huna ruhusa ya kufanya hili",
			})
		}

		// Check if user has a leadership position
		var posCount int64
		database.DB.Model(&models.UserPosition{}).
			Where("user_id = ? AND position_type IN ? AND is_active = TRUE",
				userID, []models.PositionType{models.PositionChairperson, models.PositionSecretary, models.PositionTreasurer}).
			Count(&posCount)
		if posCount > 0 {
			return c.Next()
		}

		// Check if user is an active appointed committee member
		var count int64
		if err := database.DB.Model(&models.LoanCommitteeMember{}).
			Where("user_id = ? AND is_active = TRUE", userID).
			Count(&count).Error; err != nil || count == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Wanachama wa kamati ya mikopo pekee ndio wanaweza kufikia rasilimali hii",
			})
		}

		return c.Next()
	}
}

// GetUserID returns the authenticated user ID, or "" if missing/invalid.
func GetUserID(c *fiber.Ctx) string {
	userID, ok := c.Locals("user_id").(string)
	if !ok {
		return ""
	}
	return userID
}

// GetUserRole returns the authenticated role, or empty Role if missing/invalid.
func GetUserRole(c *fiber.Ctx) models.Role {
	role, ok := c.Locals("role").(models.Role)
	if !ok {
		return ""
	}
	return role
}
