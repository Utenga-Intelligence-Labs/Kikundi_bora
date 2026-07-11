package models

import "time"

type BackupHistory struct {
	ID           string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Filename     string     `gorm:"type:varchar(255);not null" json:"filename"`
	SizeBytes    int64      `gorm:"not null" json:"size_bytes"`
	BackupType   string     `gorm:"type:varchar(50);not null" json:"backup_type"`
	EmailSentTo  string     `gorm:"type:varchar(255)" json:"email_sent_to"`
	Status       string     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedBy    string     `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	Creator      *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

type BackupSettings struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Email       string    `gorm:"type:varchar(255)" json:"email"`
	BackupType  string    `gorm:"type:varchar(50);default:'database_only'" json:"backup_type"`
	Frequency   string    `gorm:"type:varchar(20);default:'manual'" json:"frequency"`
	UpdatedBy   string    `gorm:"type:uuid" json:"updated_by"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// Backup type constants
const (
	BackupTypeDatabase = "database_only"
	BackupTypeWithFiles = "database_files"
	BackupTypeFull     = "full_system"
)

// Backup frequency constants
const (
	FrequencyManual  = "manual"
	FrequencyDaily   = "daily"
	FrequencyWeekly  = "weekly"
	FrequencyMonthly = "monthly"
)

// Backup status constants
const (
	BackupStatusPending   = "pending"
	BackupStatusCompleted = "completed"
	BackupStatusFailed    = "failed"
)

// Request/Response types
type GenerateBackupRequest struct {
	BackupType string `json:"backup_type"`
}

type SaveBackupSettingsRequest struct {
	Email      string `json:"email" validate:"required,email"`
	BackupType string `json:"backup_type" validate:"required,oneof=database_only database_files full_system"`
	Frequency  string `json:"frequency" validate:"required,oneof=manual daily weekly monthly"`
}
