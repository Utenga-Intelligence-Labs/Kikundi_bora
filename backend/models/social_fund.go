package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// SocialFundStatus lifecycle: Mwenyekiti creates (PENDING_APPROVAL), Katibu
// approves (ACTIVE), chair can close later (CLOSED). Rejected funds are kept
// for the record.
type SocialFundStatus string

const (
	SocialFundPendingApproval SocialFundStatus = "PENDING_APPROVAL"
	SocialFundActive          SocialFundStatus = "ACTIVE"
	SocialFundRejected        SocialFundStatus = "REJECTED"
	SocialFundClosed          SocialFundStatus = "CLOSED"
)

// SocialFund is a standing money pool inside the group (e.g. "Mfuko wa
// Msiba", "Mfuko wa Harusi") — SEPARATE from main savings/AKIBA and never
// included in the "Salio la Kikundi" dashboard calculation.
type SocialFund struct {
	ID              string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID         string           `gorm:"type:uuid;not null;index" json:"group_id"`
	Name            string           `gorm:"type:varchar(100);not null" json:"name"`
	Description     string           `gorm:"type:text" json:"description"`
	TargetAmount    *decimal.Decimal `gorm:"type:decimal(15,2)" json:"target_amount,omitempty"`
	CurrentBalance  decimal.Decimal  `gorm:"type:decimal(15,2);not null;default:0" json:"current_balance"`
	Status          SocialFundStatus `gorm:"type:varchar(30);not null;default:'PENDING_APPROVAL';index" json:"status"`
	CreatedBy       string           `gorm:"type:uuid;not null" json:"created_by"`
	ApprovedBy      *string          `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt      *time.Time       `json:"approved_at,omitempty"`
	RejectionReason *string          `gorm:"type:text" json:"rejection_reason,omitempty"`
	ClosedAt        *time.Time       `json:"closed_at,omitempty"`
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	Creator  *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Approver *User `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

// SocialFundContributionStatus mirrors the michango verification workflow:
// a member declares a contribution (PENDING), the Mweka Hazina confirms it
// (CONFIRMED — fund balance grows) or rejects it.
type SocialFundContributionStatus string

const (
	SFCPending   SocialFundContributionStatus = "PENDING"
	SFCConfirmed SocialFundContributionStatus = "CONFIRMED"
	SFCRejected  SocialFundContributionStatus = "REJECTED"
)

// SocialFundContribution is a member's payment into a social fund — a
// separate ledger from regular michango.
type SocialFundContribution struct {
	ID              string                       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	FundID          string                       `gorm:"type:uuid;not null;index" json:"fund_id"`
	MemberID        string                       `gorm:"type:uuid;not null;index" json:"member_id"`
	Amount          decimal.Decimal              `gorm:"type:decimal(15,2);not null" json:"amount"`
	Status          SocialFundContributionStatus `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	ContributedAt   *time.Time                   `json:"contributed_at,omitempty"`
	VerifiedBy      *string                      `gorm:"type:uuid" json:"verified_by,omitempty"`
	VerifiedAt      *time.Time                   `json:"verified_at,omitempty"`
	RejectionReason *string                      `gorm:"type:text" json:"rejection_reason,omitempty"`
	CreatedAt       time.Time                    `gorm:"autoCreateTime" json:"created_at"`

	Fund     *SocialFund `gorm:"foreignKey:FundID" json:"fund,omitempty"`
	Member   *Member     `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Verifier *User       `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
}
