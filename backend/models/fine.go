package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Fine type: fixed TZS amount or a percentage of the group's contribution.
const (
	FineTypeFixed      = "fixed"
	FineTypePercentage = "percentage"
)

// Fine status
const (
	FineUnpaid = "unpaid"
	FinePaid   = "paid"
	FineWaived = "waived"
)

// FineSettings is the approved, group-scoped fines policy. Mutated only when
// a GroupSettingProposal of kind "fines" is approved by the Katibu.
type FineSettings struct {
	ID              string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID         string           `gorm:"type:uuid;not null;uniqueIndex" json:"group_id"`
	Enabled         bool             `gorm:"not null;default:false" json:"enabled"`
	FineType        string           `gorm:"type:varchar(20);not null;default:'fixed'" json:"fine_type"`
	FineAmount      *decimal.Decimal `gorm:"type:decimal(15,2)" json:"fine_amount,omitempty"`
	FinePercentage  *decimal.Decimal `gorm:"type:decimal(7,2)" json:"fine_percentage,omitempty"`
	GracePeriodDays int              `gorm:"not null;default:0" json:"grace_period_days"`
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	Group *Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// Fine is one collectible charge for a member who missed a contribution cycle.
// Unique (group_id, member_id, contribution_cycle_label) is the idempotency
// guard so the scheduler can never create two fines for the same cycle.
type Fine struct {
	ID                     string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID                string          `gorm:"type:uuid;not null;uniqueIndex:idx_fines_group_member_cycle" json:"group_id"`
	MemberID               string          `gorm:"type:uuid;not null;uniqueIndex:idx_fines_group_member_cycle;index" json:"member_id"`
	ContributionCycleLabel string          `gorm:"type:varchar(32);not null;uniqueIndex:idx_fines_group_member_cycle" json:"contribution_cycle_label"`
	DueDate                time.Time       `gorm:"type:date;not null;index" json:"due_date"`
	Amount                 decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Reason                 string          `gorm:"type:text;not null" json:"reason"`
	Status                 string          `gorm:"type:varchar(20);not null;default:'unpaid';index" json:"status"`
	CollectedBy            *string         `gorm:"type:uuid" json:"collected_by,omitempty"`
	CollectedAt            *time.Time      `json:"collected_at,omitempty"`
	WaivedBy               *string         `gorm:"type:uuid" json:"waived_by,omitempty"`
	WaivedReason           *string         `gorm:"type:text" json:"waived_reason,omitempty"`
	CreatedAt              time.Time       `gorm:"autoCreateTime" json:"created_at"`

	Member    *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Collector *User   `gorm:"foreignKey:CollectedBy" json:"collector,omitempty"`
	Waiver    *User   `gorm:"foreignKey:WaivedBy" json:"waiver,omitempty"`
	Group     *Group  `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

func IsValidFineType(t string) bool {
	return t == FineTypeFixed || t == FineTypePercentage
}

func IsValidFineStatus(s string) bool {
	return s == FineUnpaid || s == FinePaid || s == FineWaived
}
