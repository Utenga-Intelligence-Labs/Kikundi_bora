package models

import (
	"time"

	"gorm.io/gorm"
)

type Member struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID       *string        `gorm:"type:uuid;index" json:"user_id,omitempty"`
	MemberNo     string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"member_no"`
	FullName     string         `gorm:"type:varchar(150);not null" json:"full_name"`
	Phone        string         `gorm:"type:varchar(15);uniqueIndex;not null" json:"phone"`
	Address      *string        `gorm:"type:text" json:"address,omitempty"`
	JoinedAt     time.Time      `gorm:"type:date;not null" json:"joined_at"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	RegisteredBy string         `gorm:"type:uuid;not null" json:"registered_by"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Registrar *User `gorm:"foreignKey:RegisteredBy" json:"registrar,omitempty"`
	User      *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
