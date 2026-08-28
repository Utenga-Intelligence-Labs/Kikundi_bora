package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ContributionType represents the type of contribution
type ContributionType string

const (
	ContributionAkiba ContributionType = "AKIBA"
	ContributionMfuko ContributionType = "MFUKO_WA_KIJAMII"
)

// ContributionStatus represents the verification status
type ContributionStatus string

const (
	ContributionPending   ContributionStatus = "PENDING_VERIFICATION"
	ContributionConfirmed ContributionStatus = "CONFIRMED"
	ContributionRejected  ContributionStatus = "REJECTED"
)

// MemberContribution represents a member-submitted contribution with verification workflow
type MemberContribution struct {
	ID                    string             `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MemberID              string             `gorm:"type:uuid;not null;index" json:"member_id"`
	ContributionType      ContributionType   `gorm:"type:varchar(20);not null" json:"contribution_type"`
	PeriodLabel           string             `gorm:"type:varchar(30);not null" json:"period_label"`
	Amount                decimal.Decimal    `gorm:"type:decimal(12,2);not null" json:"amount"`
	ProofImageURL         string             `gorm:"type:text" json:"proof_image_url,omitempty"`
	ProofMessage          string             `gorm:"type:text" json:"proof_message,omitempty"`
	Status                ContributionStatus `gorm:"type:varchar(20);not null;default:'PENDING_VERIFICATION'" json:"status"`
	ReviewedByMemberID    *string            `gorm:"type:uuid" json:"reviewed_by_member_id,omitempty"`
	ReviewReason          string             `gorm:"type:text" json:"review_reason,omitempty"`
	IsHistoricalImport    bool               `gorm:"not null;default:false" json:"is_historical_import"`
	CreatedAt             time.Time          `gorm:"autoCreateTime" json:"created_at"`
	ReviewedAt            time.Time          `json:"reviewed_at,omitempty"`

	Member         *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	ReviewedBy     *Member `gorm:"foreignKey:ReviewedByMemberID" json:"reviewed_by,omitempty"`
}
