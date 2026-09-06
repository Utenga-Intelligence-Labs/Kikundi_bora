package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Loan offset (savings applied to overdue debt) approval lifecycle.
// Three-role check: mwenyekiti proposes → katibu approves/rejects →
// mweka-hazina executes. This is the documented assumption for this
// significant financial action — flag in PR if a simpler single-approver
// flow is preferred instead.
const (
	LoanOffsetProposed  = "PROPOSED"
	LoanOffsetApproved  = "APPROVED"
	LoanOffsetExecuted  = "EXECUTED"
	LoanOffsetRejected  = "REJECTED"
)

// LoanOffsetTransaction records savings forcibly applied to an overdue loan.
// Deliberately separate from Repayment and Contribution rows so the member's
// history shows WHY savings dropped — never confused with a cash payment.
type LoanOffsetTransaction struct {
	ID       string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	LoanID   string `gorm:"type:uuid;not null;index" json:"loan_id"`
	MemberID string `gorm:"type:uuid;not null;index" json:"member_id"`

	// ProposedAmount is the capped snapshot at propose time;
	// Amount is the final capped value recomputed at execution.
	ProposedAmount decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"proposed_amount"`
	Amount         decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"amount"`
	// Snapshots for itemization (what the cap was computed from).
	OutstandingBefore decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"outstanding_before"`
	SavingsBefore     decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"savings_before"`

	Status      string     `gorm:"type:varchar(20);not null;default:'PROPOSED';index" json:"status"`
	ProposedBy  string     `gorm:"type:uuid;not null" json:"proposed_by"`
	ApprovedBy  *string    `gorm:"type:uuid" json:"approved_by,omitempty"`
	ExecutedBy  *string    `gorm:"type:uuid" json:"executed_by,omitempty"`
	ProposedAt  time.Time  `gorm:"autoCreateTime" json:"proposed_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
	Reason      *string    `gorm:"type:text" json:"reason,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy   *string    `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Loan     *Loan   `gorm:"foreignKey:LoanID" json:"loan,omitempty"`
	Member   *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Proposer *User   `gorm:"foreignKey:ProposedBy" json:"proposer,omitempty"`
	Approver *User   `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
	Executor *User   `gorm:"foreignKey:ExecutedBy" json:"executor,omitempty"`
}
