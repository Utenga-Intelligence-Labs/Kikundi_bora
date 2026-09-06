package models

import (
	"time"

	"gorm.io/gorm"
)

type Member struct {
	ID             string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID         *string        `gorm:"type:uuid;index" json:"user_id,omitempty"`
	MemberNo       string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"member_no"`
	FullName       string         `gorm:"type:varchar(150);not null" json:"full_name"`
	Phone          string         `gorm:"type:varchar(15);uniqueIndex;not null" json:"phone"`
	Address        *string        `gorm:"type:text" json:"address,omitempty"`
	Gender         *string        `gorm:"type:varchar(10)" json:"gender,omitempty"` // MME | MKE
	Occupation     *string        `gorm:"type:varchar(100)" json:"occupation,omitempty"`
	Email          *string        `gorm:"type:varchar(150)" json:"email,omitempty"`
	NextOfKinName  *string        `gorm:"type:varchar(150)" json:"next_of_kin_name,omitempty"`
	NextOfKinPhone *string        `gorm:"type:varchar(15)" json:"next_of_kin_phone,omitempty"`
	PhotoURL       *string        `gorm:"type:text" json:"photo_url,omitempty"`
	JoinedAt       time.Time      `gorm:"type:date;not null" json:"joined_at"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	RegisteredBy   string         `gorm:"type:uuid;not null" json:"registered_by"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy      *string        `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Approval workflow: mwenyekiti submits (pending) → katibu approves/rejects.
	// A member only counts toward totals and gains dashboard access once
	// approval_status = 'approved' (default 'approved' keeps pre-existing
	// rows active after migration).
	ApprovalStatus  string     `gorm:"type:varchar(20);not null;default:'approved';index" json:"approval_status"`
	ApprovedBy      *string    `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectionReason *string    `gorm:"type:text" json:"rejection_reason,omitempty"`

	Registrar *User `gorm:"foreignKey:RegisteredBy" json:"registrar,omitempty"`
	Approver  *User `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
	User      *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Member approval status constants
const (
	MemberApprovalPending  = "pending"
	MemberApprovalApproved = "approved"
	MemberApprovalRejected = "rejected"
)
