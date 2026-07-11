package models

import (
	"time"
)

// LoanCommitteeMember represents an ordinary member appointed to the loan committee.
// Chair, Secretary, and Treasurer are automatically committee members by role
// and are NOT stored in this table.
type LoanCommitteeMember struct {
	ID          string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID      string     `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	AppointedBy string     `gorm:"type:uuid;not null" json:"appointed_by"`
	AppointedAt time.Time  `gorm:"autoCreateTime" json:"appointed_at"`
	RemovedAt   *time.Time `json:"removed_at,omitempty"`
	IsActive    bool       `gorm:"default:true;not null" json:"is_active"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	User      *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Appointer *User `gorm:"foreignKey:AppointedBy" json:"appointer,omitempty"`
}

// LoanReviewDecision represents the decision a committee member makes on a loan.
type LoanReviewDecision string

const (
	ReviewPending LoanReviewDecision = "PENDING"
	ReviewApprove LoanReviewDecision = "APPROVE"
	ReviewReject  LoanReviewDecision = "REJECT"
)

// LoanReview records each committee member's review of a specific loan.
// A committee member can review a specific loan only once.
type LoanReview struct {
	ID         string             `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	LoanID     string             `gorm:"type:uuid;not null;uniqueIndex:idx_loan_reviewer" json:"loan_id"`
	ReviewerID string             `gorm:"type:uuid;not null;uniqueIndex:idx_loan_reviewer" json:"reviewer_id"`
	Decision   LoanReviewDecision `gorm:"type:varchar(20);not null;default:'PENDING'" json:"decision"`
	Comments   *string            `gorm:"type:text" json:"comments,omitempty"`
	ReviewedAt *time.Time         `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time          `gorm:"autoUpdateTime" json:"updated_at"`

	Loan     *Loan `gorm:"foreignKey:LoanID" json:"loan,omitempty"`
	Reviewer *User `gorm:"foreignKey:ReviewerID" json:"reviewer,omitempty"`
}
