package models

import (
	"time"
)

type Contribution struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MemberID        string    `gorm:"type:uuid;not null;uniqueIndex:idx_member_month" json:"member_id"`
	RecordedBy      string    `gorm:"type:uuid;not null" json:"recorded_by"`
	Amount          float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	Month           time.Time `gorm:"type:date;not null;uniqueIndex:idx_member_month" json:"month"`
	PaidAt          time.Time `gorm:"type:date;not null" json:"paid_at"`
	PaymentMethod   string    `gorm:"type:varchar(20);not null;default:'CASH'" json:"payment_method"`
	ReferenceNumber string    `gorm:"type:varchar(100)" json:"reference_number,omitempty"`
	ReceiptURL      string    `gorm:"type:varchar(500)" json:"receipt_url,omitempty"`
	Status          string    `gorm:"type:varchar(20);default:'PAID';not null" json:"status"`
	ConfirmedBy     *string   `gorm:"type:uuid" json:"confirmed_by,omitempty"`
	Notes           *string   `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`

	Member    *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Recorder  *User   `gorm:"foreignKey:RecordedBy" json:"recorder,omitempty"`
	Confirmer *User   `gorm:"foreignKey:ConfirmedBy" json:"confirmer,omitempty"`
}

type ContributionEdit struct {
	ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ContributionID string    `gorm:"type:uuid;not null;index" json:"contribution_id"`
	EditedBy       string    `gorm:"type:uuid;not null" json:"edited_by"`
	OldAmount      float64   `gorm:"type:decimal(15,2);not null" json:"old_amount"`
	NewAmount      float64   `gorm:"type:decimal(15,2);not null" json:"new_amount"`
	Reason         string    `gorm:"type:text;not null" json:"reason"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	Contribution *Contribution `gorm:"foreignKey:ContributionID" json:"contribution,omitempty"`
	Editor       *User         `gorm:"foreignKey:EditedBy" json:"editor,omitempty"`
}
