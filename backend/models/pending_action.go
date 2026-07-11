package models

import (
	"encoding/json"
	"time"
)

type PendingAction struct {
	ID          string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ActionType  string          `gorm:"type:varchar(50);not null;index" json:"action_type"`
	Payload     json.RawMessage `gorm:"type:jsonb;not null" json:"payload"`
	RequestedBy string          `gorm:"type:uuid;not null;index" json:"requested_by"`
	Status      string          `gorm:"type:varchar(20);default:'PENDING';not null;index" json:"status"` // PENDING, APPROVED, REJECTED
	ApprovedBy  *string         `gorm:"type:uuid" json:"approved_by,omitempty"`
	Remarks     string          `gorm:"type:text" json:"remarks,omitempty"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
	ResolvedAt  *time.Time      `json:"resolved_at,omitempty"`

	Requester User  `gorm:"foreignKey:RequestedBy" json:"requester,omitempty"`
	Approver  *User `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

// Action type constants
const (
	ActionContributionEdit = "CONTRIBUTION_EDIT"
	ActionWelfareCreate    = "WELFARE_CREATE"
	ActionLoanDisburse     = "LOAN_DISBURSE"
)

// Status constants
const (
	ActionStatusPending  = "PENDING"
	ActionStatusApproved = "APPROVED"
	ActionStatusRejected = "REJECTED"
)
