package models

import "time"

type AdminLog struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AdminID      string    `gorm:"type:uuid;not null;index" json:"admin_id"`
	Action       string    `gorm:"type:varchar(100);not null" json:"action"`
	TargetUserID *string   `gorm:"type:uuid;index" json:"target_user_id,omitempty"`
	Metadata     string    `gorm:"type:jsonb" json:"metadata,omitempty"`
	IPAddress    string    `gorm:"type:varchar(45)" json:"ip_address"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	Admin      User  `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	TargetUser *User `gorm:"foreignKey:TargetUserID" json:"target_user,omitempty"`
}
