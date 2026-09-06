package models

import (
	"crypto/rand"
	"math/big"
	"time"
)

type Role string

const (
	RoleChair     Role = "chair"
	RoleTreasurer Role = "treasurer"
	RoleSecretary Role = "secretary"
	RoleMember    Role = "member"
	RoleAdmin     Role = "admin"
)

// User status constants
const (
	UserStatusPending   = "PENDING"
	UserStatusActive    = "ACTIVE"
	UserStatusRejected  = "REJECTED"
	UserStatusSuspended = "SUSPENDED"
)

// DefaultTempPassword generates a random 12-character temp password.
// Never use a hardcoded value for production.
func DefaultTempPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 12)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

type User struct {
	ID                 string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name               string     `gorm:"type:varchar(100);not null" json:"name"`
	Email              *string    `gorm:"type:varchar(150);uniqueIndex" json:"email,omitempty"`
	Phone              string     `gorm:"type:varchar(15);not null" json:"phone"`
	Password           string     `gorm:"type:varchar(255)" json:"-"`
	Role               Role       `gorm:"type:varchar(20);not null" json:"role"`
	Status             string     `gorm:"type:varchar(20);default:'PENDING';not null" json:"status"`
	MustChangePassword bool       `gorm:"default:false" json:"must_change_password"`
	AvatarURL          string     `gorm:"type:text" json:"avatar_url"`
	Bio                string     `gorm:"type:text" json:"bio"`
	IsActive           bool       `gorm:"default:true" json:"is_active"`
	CreatedBy          *string    `gorm:"type:uuid" json:"created_by,omitempty"`
	ApprovedBy         *string    `gorm:"type:uuid" json:"approved_by,omitempty"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	DeletedAt          *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type UserSession struct {
	ID           string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID       string     `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash    string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	IPAddress    string     `gorm:"type:varchar(45);not null" json:"ip_address"`
	UserAgent    string     `gorm:"type:text" json:"user_agent,omitempty"`
	LastActiveAt time.Time  `gorm:"not null" json:"last_active_at"`
	ExpiresAt    time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

type FailedLogin struct {
	ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	EmailAttempted string    `gorm:"type:varchar(150);not null;index" json:"email_attempted"`
	IPAddress      string    `gorm:"type:varchar(45);not null;index" json:"ip_address"`
	UserAgent      string    `gorm:"type:text" json:"user_agent,omitempty"`
	AttemptedAt    time.Time `gorm:"not null;index" json:"attempted_at"`
}
