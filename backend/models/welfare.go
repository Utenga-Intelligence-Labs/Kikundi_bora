package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// WelfareEventType represents the type of welfare event.
type WelfareEventType string

const (
	WelfareMisiba   WelfareEventType = "MSIBA"    // Death/Funeral
	WelfareHarusi   WelfareEventType = "HARUSI"   // Wedding
	WelfareDharura  WelfareEventType = "DHARURA"  // Emergency
	WelfareMedical  WelfareEventType = "MATIBABU" // Medical
	WelfareKuzaliwa WelfareEventType = "KUZALIWA" // Birth
	WelfareElimu    WelfareEventType = "ELIMU"    // Education
)

// WelfareFundingSource represents how the welfare event is funded.
type WelfareFundingSource string

const (
	FundTreasury            WelfareFundingSource = "TREASURY"
	FundMemberContribution  WelfareFundingSource = "MEMBER_CONTRIBUTION"
	FundBoth                WelfareFundingSource = "BOTH"
)

// WelfareEventStatus represents the lifecycle status of a welfare event.
type WelfareEventStatus string

const (
	WelfarePending   WelfareEventStatus = "PENDING"
	WelfareApproved  WelfareEventStatus = "APPROVED"
	WelfareRejected  WelfareEventStatus = "REJECTED"
	WelfareCompleted WelfareEventStatus = "COMPLETED"
)

// WelfareEvent represents a welfare event (misiba, harusi, dharura, etc.).
type WelfareEvent struct {
	ID              string               `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MemberID        string               `gorm:"type:uuid;not null;index" json:"member_id"`
	EventType       WelfareEventType     `gorm:"type:varchar(20);not null" json:"event_type"`
	Description     string               `gorm:"type:text;not null" json:"description"`
	AmountRequested decimal.Decimal      `gorm:"type:decimal(15,2);not null" json:"amount_requested"`
	AmountApproved  *decimal.Decimal     `gorm:"type:decimal(15,2)" json:"amount_approved,omitempty"`
	FundingSource   WelfareFundingSource `gorm:"type:varchar(30);not null" json:"funding_source"`
	TreasuryAmount  decimal.Decimal      `gorm:"type:decimal(15,2);not null;default:0" json:"treasury_amount"`
	MemberAmount    decimal.Decimal      `gorm:"type:decimal(15,2);not null;default:0" json:"member_amount"`
	Status          WelfareEventStatus   `gorm:"type:varchar(20);not null;default:'PENDING';index" json:"status"`
	CreatedBy       string               `gorm:"type:uuid;not null" json:"created_by"`
	ApprovedBy      *string              `gorm:"type:uuid" json:"approved_by,omitempty"`
	RejectedBy      *string              `gorm:"type:uuid" json:"rejected_by,omitempty"`
	RejectionReason *string              `gorm:"type:text" json:"rejection_reason,omitempty"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	DisbursedBy     *string              `gorm:"type:uuid" json:"disbursed_by,omitempty"`
	DisbursedAt     *time.Time           `json:"disbursed_at,omitempty"`
	ReceivedAt      *time.Time           `json:"received_at,omitempty"`
	ReceivedBy      *string              `gorm:"type:uuid" json:"received_by,omitempty"`
	CreatedAt       time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time            `gorm:"autoUpdateTime" json:"updated_at"`

	Member   *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Creator  *User   `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Approver *User   `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
	Rejector *User   `gorm:"foreignKey:RejectedBy" json:"rejector,omitempty"`
}

// WelfareContributionStatus represents the payment status of a member's welfare contribution.
type WelfareContributionStatus string

const (
	WelfareContribPending WelfareContributionStatus = "PENDING"
	WelfareContribPaid    WelfareContributionStatus = "PAID"
	WelfareContribWaived  WelfareContributionStatus = "WAIVED"
)

// WelfareContribution represents a member's obligation to contribute to a welfare event.
type WelfareContribution struct {
	ID         string                    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	EventID    string                    `gorm:"type:uuid;not null;index:idx_event_member,unique" json:"event_id"`
	MemberID   string                    `gorm:"type:uuid;not null;index:idx_event_member,unique" json:"member_id"`
	Amount     decimal.Decimal           `gorm:"type:decimal(15,2);not null" json:"amount"`
	Status     WelfareContributionStatus `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	PaidAt     *time.Time                `json:"paid_at,omitempty"`
	RecordedBy *string                   `gorm:"type:uuid" json:"recorded_by,omitempty"`
	CreatedAt  time.Time                 `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time                 `gorm:"autoUpdateTime" json:"updated_at"`

	Event    *WelfareEvent `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Member   *Member       `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Recorder *User         `gorm:"foreignKey:RecordedBy" json:"recorder,omitempty"`
}
