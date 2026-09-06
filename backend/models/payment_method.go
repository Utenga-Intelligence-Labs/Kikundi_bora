package models

import (
	"gorm.io/gorm"
	"time"
)

// PaymentMethodType distinguishes mobile-money paybill numbers from bank accounts.
type PaymentMethodType string

const (
	PaymentLipaNamba PaymentMethodType = "lipa_namba"
	PaymentBank      PaymentMethodType = "bank"
)

// IsValidPaymentMethodType checks whether the type is supported.
func IsValidPaymentMethodType(t string) bool {
	return t == string(PaymentLipaNamba) || t == string(PaymentBank)
}

// PaymentMethod approval status: treasurer submissions stay pending until
// the Mwenyekiti approves; chair-created methods are approved immediately.
const (
	PaymentMethodPending  = "pending"
	PaymentMethodApproved = "approved"
)

// PaymentMethod holds group-level payment details (LipaNamba paybill numbers
// and bank accounts) shown to members on the Weka Mchango page. Managed by
// the Mwenyekiti / Mweka Hazina.
type PaymentMethod struct {
	ID            string            `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID       string            `gorm:"type:uuid;not null;index" json:"group_id"`
	Type          PaymentMethodType `gorm:"type:varchar(20);not null" json:"type"`           // lipa_namba | bank
	ProviderName  string            `gorm:"type:varchar(50);not null" json:"provider_name"`  // M-Pesa, Tigo Pesa, CRDB, NMB...
	AccountNumber string            `gorm:"type:varchar(50);not null" json:"account_number"` // lipa namba number or bank account number
	AccountName   string            `gorm:"type:varchar(100);not null" json:"account_name"`  // registered / business name
	IsActive      bool              `gorm:"not null;default:true" json:"is_active"`
	Status        string            `gorm:"type:varchar(20);not null;default:'approved';index" json:"status"` // pending | approved
	CreatedBy     string            `gorm:"type:uuid;not null" json:"created_by"`
	ApprovedBy    *string           `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt    *time.Time        `json:"approved_at,omitempty"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"deleted_at,omitempty"`
	DeletedBy     *string           `gorm:"type:uuid" json:"deleted_by,omitempty"`
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}
