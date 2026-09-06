package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// DissolutionProposalStatus lifecycle.
const (
	DissolutionVotingOpen string = "voting_open"
	DissolutionApproved   string = "approved"
	DissolutionRejected   string = "rejected"
	DissolutionExecuted   string = "executed"
)

// GroupDissolutionProposal is the vote to dissolve a group after N years.
// Threshold assumption (documented): simple majority of votes cast (>50% yes).
// Flagged as open decision — confirm quorum/majority rule if different.
// Share-out is principal-only (sum of MAIN contributions within period); interest-sharing is a follow-up.
type GroupDissolutionProposal struct {
	ID              string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID         string         `gorm:"type:uuid;not null;index" json:"group_id"`
	ProposedBy      string         `gorm:"type:uuid;not null" json:"proposed_by"`
	CycleSpanYears  int            `gorm:"not null" json:"cycle_span_years"` // 1 or 2
	PeriodStart     time.Time      `gorm:"type:date;not null" json:"period_start"`
	PeriodEnd       time.Time      `gorm:"type:date;not null" json:"period_end"`
	Status          string         `gorm:"type:varchar(20);not null;default:'voting_open';index" json:"status"`
	VotingDeadline  time.Time      `gorm:"not null" json:"voting_deadline"`
	ExecutedAt      *time.Time     `json:"executed_at,omitempty"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy       *string        `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Group    *Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Proposer *User  `gorm:"foreignKey:ProposedBy" json:"proposer,omitempty"`
}

type DissolutionVote struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ProposalID string         `gorm:"type:uuid;not null;index" json:"proposal_id"`
	MemberID   string         `gorm:"type:uuid;not null" json:"member_id"`
	Vote       string         `gorm:"type:varchar(10);not null" json:"vote"` // yes | no
	VotedAt    time.Time      `gorm:"autoCreateTime" json:"voted_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy  *string        `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Proposal *GroupDissolutionProposal `gorm:"foreignKey:ProposalID" json:"proposal,omitempty"`
	Member   *Member                   `gorm:"foreignKey:MemberID" json:"member,omitempty"`
}

type DissolutionPayout struct {
	ID               string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ProposalID       string          `gorm:"type:uuid;not null;index" json:"proposal_id"`
	MemberID         string          `gorm:"type:uuid;not null;index" json:"member_id"`
	TotalContributed decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"total_contributed"`
	TotalOwed        decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"total_owed"`
	AmountOwed       decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount_owed"` // net = max(0, contributed - owed)
	Status           string          `gorm:"type:varchar(20);not null;default:'pending'" json:"status"` // pending | paid
	CalculatedAt     time.Time       `gorm:"autoCreateTime" json:"calculated_at"`
	PaidAt           *time.Time      `json:"paid_at,omitempty"`
	PaidBy           *string         `gorm:"type:uuid" json:"paid_by,omitempty"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy        *string         `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt        time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	Member   *Member                   `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Proposal *GroupDissolutionProposal `gorm:"foreignKey:ProposalID" json:"proposal,omitempty"`
}

type CreateDissolutionProposalRequest struct {
	CycleSpanYears int    `json:"cycle_span_years" validate:"required,oneof=1 2"`
	VotingDeadline string `json:"voting_deadline" validate:"required"` // ISO8601
}

type VoteRequest struct {
	Vote string `json:"vote" validate:"required,oneof=yes no YES NO Yes No"`
}
