package handlers

import (
	"errors"
	"fmt"
	"strings"

	"kikundibora/models"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// roleToPosition maps JWT roles to UserPosition types for dual-authz sync.
func roleToPosition(role models.Role) (models.PositionType, bool) {
	switch role {
	case models.RoleChair:
		return models.PositionChairperson, true
	case models.RoleTreasurer:
		return models.PositionTreasurer, true
	case models.RoleSecretary:
		return models.PositionSecretary, true
	default:
		return "", false
	}
}

// upsertUserPosition ensures the user has an active leadership position matching role.
// Deactivates other leadership positions for the same user when role maps to a position.
func upsertUserPosition(db *gorm.DB, userID string, role models.Role) error {
	posType, ok := roleToPosition(role)
	if !ok {
		// Non-leadership role: deactivate any leadership positions
		return db.Model(&models.UserPosition{}).
			Where("user_id = ? AND is_active = TRUE AND position_type IN ?",
				userID, []models.PositionType{models.PositionChairperson, models.PositionSecretary, models.PositionTreasurer}).
			Update("is_active", false).Error
	}

	// Deactivate other leadership positions for this user
	if err := db.Model(&models.UserPosition{}).
		Where("user_id = ? AND is_active = TRUE AND position_type <> ?", userID, posType).
		Update("is_active", false).Error; err != nil {
		return err
	}

	var existing models.UserPosition
	err := db.Where("user_id = ? AND position_type = ?", userID, posType).First(&existing).Error
	if err == nil {
		if !existing.IsActive {
			return db.Model(&existing).Update("is_active", true).Error
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.Create(&models.UserPosition{
		UserID:       userID,
		PositionType: posType,
		IsActive:     true,
	}).Error
}

// escapeLike escapes SQL LIKE special characters to prevent wildcard injection.
// Use this before constructing LIKE search patterns.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// likePattern creates a safe LIKE search pattern with escaped wildcards.
func likePattern(search string) string {
	escaped := escapeLike(strings.ToLower(search))
	return "%" + escaped + "%"
}

// formatValidationErrors converts validator errors to user-friendly messages
// without leaking internal struct field names.
func formatValidationErrors(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var msgs []string
		for _, e := range validationErrors {
			switch e.Tag() {
			case "required":
				msgs = append(msgs, fmt.Sprintf("'%s' is required", friendlyFieldName(e.Field())))
			case "min":
				msgs = append(msgs, fmt.Sprintf("'%s' must be at least %s characters", friendlyFieldName(e.Field()), e.Param()))
			case "max":
				msgs = append(msgs, fmt.Sprintf("'%s' must be at most %s characters", friendlyFieldName(e.Field()), e.Param()))
			case "email":
				msgs = append(msgs, fmt.Sprintf("'%s' must be a valid email", friendlyFieldName(e.Field())))
			case "oneof":
				msgs = append(msgs, fmt.Sprintf("'%s' must be one of: %s", friendlyFieldName(e.Field()), e.Param()))
			case "gt":
				msgs = append(msgs, fmt.Sprintf("'%s' must be greater than %s", friendlyFieldName(e.Field()), e.Param()))
			default:
				msgs = append(msgs, fmt.Sprintf("'%s' is invalid", friendlyFieldName(e.Field())))
			}
		}
		return strings.Join(msgs, "; ")
	}
	return "Data si sahihi"
}

// friendlyFieldName converts Go struct field names to user-friendly names.
func friendlyFieldName(field string) string {
	nameMap := map[string]string{
		"Email":           "Barua pepe",
		"Password":        "Nenosiri",
		"Name":            "Jina",
		"Phone":           "Nambari ya simu",
		"Role":            "Jukumu",
		"FullName":        "Jina kamili",
		"NewPassword":     "Nenosiri jipya",
		"ConfirmPassword": "Uthibitisho wa nenosiri",
		"OldPassword":     "Nenosiri la zamani",
		"Amount":          "Kiasi",
		"MemberID":        "Mwanachama",
		"Month":           "Mwezi",
		"PaidAt":          "Tarehe ya malipo",
		"PaymentMethod":   "Njia ya malipo",
		"DueDate":         "Tarehe ya mwisho",
		"Action":          "Kitendo",
		"Reason":          "Sababu",
		"Decision":        "Uamuzi",
		"EventType":       "Aina ya tukio",
		"Description":     "Maelezo",
		"FundingSource":   "Chanzo cha fedha",
	}
	if friendly, ok := nameMap[field]; ok {
		return friendly
	}
	return field
}
