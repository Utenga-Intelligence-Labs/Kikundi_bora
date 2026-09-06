package models

import (
	"gorm.io/gorm"
	"time"
)

type PositionType string

const (
	PositionChairperson PositionType = "CHAIRPERSON"
	PositionSecretary   PositionType = "SECRETARY"
	PositionTreasurer   PositionType = "TREASURER"
)

type UserPosition struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID       string         `gorm:"type:uuid;not null;index" json:"user_id"`
	PositionType PositionType   `gorm:"type:varchar(30);not null" json:"position_type"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy    *string        `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
