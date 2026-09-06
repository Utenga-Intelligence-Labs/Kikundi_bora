package models

import (
	"gorm.io/gorm"
	"time"
)

type UserApproval struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID     string         `gorm:"type:uuid;not null;index" json:"user_id"`
	ApprovedBy string         `gorm:"type:uuid;not null" json:"approved_by"`
	Status     string         `gorm:"type:varchar(20);not null" json:"status"` // APPROVED, REJECTED
	Remarks    string         `gorm:"type:text" json:"remarks,omitempty"`
	ApprovedAt time.Time      `gorm:"not null" json:"approved_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy  *string        `gorm:"type:uuid" json:"deleted_by,omitempty"`

	User     User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Approver User `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}
