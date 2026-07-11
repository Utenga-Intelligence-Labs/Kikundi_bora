package handlers

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

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
