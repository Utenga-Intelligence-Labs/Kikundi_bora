package models

import (
	"time"
)

type LoanStatus string

const (
	LoanPending     LoanStatus = "PENDING"
	LoanUnderReview LoanStatus = "UNDER_REVIEW"
	LoanApproved    LoanStatus = "APPROVED"
	LoanOutstanding LoanStatus = "OUTSTANDING"
	LoanRejected    LoanStatus = "REJECTED"
	LoanClosed      LoanStatus = "CLOSED"
)

type Loan struct {
	ID               string      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MemberID         string      `gorm:"type:uuid;not null;index" json:"member_id"`
	ReviewedBy       *string     `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	Amount           float64     `gorm:"type:decimal(15,2);not null" json:"amount"`
	ApprovedAmount   *float64    `gorm:"type:decimal(15,2)" json:"approved_amount,omitempty"`
	BalanceRemaining *float64    `gorm:"type:decimal(15,2)" json:"balance_remaining,omitempty"`
	Purpose          *string     `gorm:"type:text" json:"purpose,omitempty"`
	DueDate          time.Time   `gorm:"type:date;not null;index" json:"due_date"`
	Status           LoanStatus  `gorm:"type:varchar(20);not null;default:'PENDING';index" json:"status"`
	RejectionReason  *string     `gorm:"type:text" json:"rejection_reason,omitempty"`
	AppliedAt        time.Time   `gorm:"autoCreateTime" json:"applied_at"`
	ReviewedAt       *time.Time  `json:"reviewed_at,omitempty"`
	DisbursedBy      *string     `gorm:"type:uuid" json:"disbursed_by,omitempty"`
	DisbursedAt      *time.Time  `json:"disbursed_at,omitempty"`
	UpdatedAt        time.Time   `gorm:"autoUpdateTime" json:"updated_at"`

	// Sequential approval — Hazina → Katibu → Bodi Member → Mwenyekiti (in order)
	HazinaApprovedBy     *string    `gorm:"type:uuid" json:"hazina_approved_by,omitempty"`
	HazinaApprovedAt     *time.Time `json:"hazina_approved_at,omitempty"`
	KatibuApprovedBy     *string    `gorm:"type:uuid" json:"katibu_approved_by,omitempty"`
	KatibuApprovedAt     *time.Time `json:"katibu_approved_at,omitempty"`
	BodiApprovedBy       *string    `gorm:"type:uuid" json:"bodi_approved_by,omitempty"`
	BodiApprovedAt       *time.Time `json:"bodi_approved_at,omitempty"`
	MwenyekitiApprovedBy *string    `gorm:"type:uuid" json:"mwenyekiti_approved_by,omitempty"`
	MwenyekitiApprovedAt *time.Time `json:"mwenyekiti_approved_at,omitempty"`

	Member    *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Reviewer  *User   `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
	Disburser *User   `gorm:"foreignKey:DisbursedBy" json:"disburser,omitempty"`
}

type Repayment struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	LoanID          string    `gorm:"type:uuid;not null;index" json:"loan_id"`
	MemberID        string    `gorm:"type:uuid;not null;index" json:"member_id"`
	RecordedBy      string    `gorm:"type:uuid;not null" json:"recorded_by"`
	Amount          float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	BalanceAfter    float64   `gorm:"type:decimal(15,2);not null" json:"balance_after"`
	PaidAt          time.Time `gorm:"type:date;not null" json:"paid_at"`
	PaymentMethod   string    `gorm:"type:varchar(20);not null;default:'CASH'" json:"payment_method"`
	ReferenceNumber string    `gorm:"type:varchar(100)" json:"reference_number,omitempty"`
	ReceiptURL      string    `gorm:"type:varchar(500)" json:"receipt_url,omitempty"`
	Notes           *string   `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`

	Loan     *Loan   `gorm:"foreignKey:LoanID" json:"loan,omitempty"`
	Member   *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Recorder *User   `gorm:"foreignKey:RecordedBy" json:"recorder,omitempty"`
}
