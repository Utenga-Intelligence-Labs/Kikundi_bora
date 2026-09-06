package models

import (
	"gorm.io/gorm"
	"time"

	"github.com/shopspring/decimal"
)

// ContributionInterval defines how often members contribute.
type ContributionInterval string

const (
	IntervalWeekly     ContributionInterval = "weekly"
	IntervalMonthly    ContributionInterval = "monthly"
	IntervalSemiAnnual ContributionInterval = "semi_annual"
	IntervalYearly     ContributionInterval = "yearly"
)

// ValidContributionIntervals lists all supported intervals.
var ValidContributionIntervals = []ContributionInterval{
	IntervalWeekly, IntervalMonthly, IntervalSemiAnnual, IntervalYearly,
}

// IsValidContributionInterval checks whether the interval is supported.
func IsValidContributionInterval(i string) bool {
	for _, v := range ValidContributionIntervals {
		if string(v) == i {
			return true
		}
	}
	return false
}

// Group holds group-wide settings. This deployment serves a single group;
// the row is created idempotently on startup (database.EnsureGroupSetup).
type Group struct {
	ID   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name string `gorm:"type:varchar(150);not null" json:"name"`

	// Contribution interval settings — changed ONLY via approved proposals.
	// contribution_due_date is the day inside the interval when contributions
	// are expected:
	//   weekly:      "1".."7"  (1=Monday .. 7=Sunday)
	//   monthly:     "1".."31" (day of month)
	//   semi_annual: "MM-DD"   (applies every 6 months)
	//   yearly:      "MM-DD"
	ContributionInterval    ContributionInterval `gorm:"type:varchar(20);not null;default:'monthly'" json:"contribution_interval"`
	ContributionDueDate     *string              `gorm:"type:varchar(10)" json:"contribution_due_date,omitempty"`
	FixedContributionAmount *decimal.Decimal     `gorm:"type:decimal(15,2)" json:"fixed_contribution_amount,omitempty"`

	// Idempotency for due-date notifications (per cycle):
	LastReminderNotifiedFor *time.Time `json:"last_reminder_notified_for,omitempty"`
	LastDueNotifiedFor      *time.Time `json:"last_due_notified_for,omitempty"`

	// SMS channel master switch for the group (Part 1). Effective sending
	// additionally requires a configured provider + per-type opt-in.
	SMSNotificationsEnabled bool `gorm:"not null;default:false" json:"sms_notifications_enabled"`

	Status string `gorm:"type:varchar(20);not null;default:'active'" json:"status"` // active | dissolved

	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	DeletedBy *string `gorm:"type:uuid" json:"deleted_by,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
