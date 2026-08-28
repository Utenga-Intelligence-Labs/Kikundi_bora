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
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token ya ukaguzi haijapatikana",
		})
	}

	tokenStr := strings.TrimPrefix(header, "Bearer ")

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
	roleStr, ok := claims["role"].(string)
	if !ok || roleStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Token haina jukumu la mtumiaji",
		})
	}
	role := models.Role(roleStr)

	// Check if token session has been revoked (logout)
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenStr)))
	var revoked int64
	database.DB.Model(&models.UserSession{}).
		Where("user_id = ? AND token_hash = ? AND revoked_at IS NOT NULL", userID, tokenHash).
		Count(&revoked)
	if revoked > 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Kipindi kimeisha. Tafadhali ingia tena.",
		})
	}

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

		// Admin bypasses all role checks
		if role == models.RoleAdmin {
			return c.Next()
		}

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
// - Admin or leadership role (chair / secretary / treasurer)
// - Active leadership position (CHAIRPERSON, SECRETARY, TREASURER)
// - Active appointed committee member in loan_committee_members
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

		// Admin and leadership roles may access (same as review eligibility)
		if role == models.RoleAdmin || role == models.RoleChair || role == models.RoleSecretary || role == models.RoleTreasurer {
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
