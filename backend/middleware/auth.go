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

	userID := claims["user_id"].(string)
	role := models.Role(claims["role"].(string))

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

	// Update last active timestamp
	database.DB.Model(&models.UserSession{}).
		Where("user_id = ? AND token_hash = ? AND revoked_at IS NULL", userID, tokenHash).
		Updates(map[string]interface{}{"last_active_at": time.Now()})

	c.Locals("user_id", userID)
	c.Locals("role", role)

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

// RequireLoanCommitteeMember allows access if the user is:
// - Admin
// - Has a leadership position (CHAIRPERSON, SECRETARY, TREASURER)
// - An active appointed committee member in loan_committee_members
func RequireLoanCommitteeMember() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(models.Role)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Huna ruhusa ya kufanya hili",
			})
		}

		// Admin bypasses
		if role == models.RoleAdmin {
			return c.Next()
		}

		userID := GetUserID(c)

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

func GetUserID(c *fiber.Ctx) string {
	return c.Locals("user_id").(string)
}

func GetUserRole(c *fiber.Ctx) models.Role {
	return c.Locals("role").(models.Role)
}
