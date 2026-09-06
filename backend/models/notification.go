package models

import (
	"encoding/json"
	"time"
)

type NotificationType string

const (
	NotifLoanRequest       NotificationType = "LOAN_REQUEST"
	NotifLoanApproved      NotificationType = "LOAN_APPROVED"
	NotifLoanRejected      NotificationType = "LOAN_REJECTED"
	NotifLoanDisbursed     NotificationType = "LOAN_DISBURSED"
	NotifLoanUnderReview   NotificationType = "LOAN_UNDER_REVIEW"
	NotifCommitteeAppoint  NotificationType = "COMMITTEE_APPOINTED"
	NotifCommitteeRemove   NotificationType = "COMMITTEE_REMOVED"
	NotifRepayment         NotificationType = "REPAYMENT"
	NotifContribution      NotificationType = "CONTRIBUTION"
	NotifContributionDue   NotificationType = "CONTRIBUTION_DUE"
	NotifFineIssued          NotificationType = "FINE_ISSUED"
	NotifSystem            NotificationType = "SYSTEM"
	NotifWelfareCreated    NotificationType = "WELFARE_CREATED"
	NotifWelfareApproved   NotificationType = "WELFARE_APPROVED"
	NotifWelfareRejected   NotificationType = "WELFARE_REJECTED"
	NotifWelfarePayment    NotificationType = "WELFARE_PAYMENT"
	NotifWelfareCompleted  NotificationType = "WELFARE_COMPLETED"
	NotifUserCreated       NotificationType = "USER_CREATED"
	NotifUserApproved      NotificationType = "USER_APPROVED"
	NotifUserRejected      NotificationType = "USER_REJECTED"
	NotifPasswordSetup     NotificationType = "PASSWORD_SETUP"
)

type Notification struct {
	ID        string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string           `gorm:"type:uuid;not null;index:idx_user_read" json:"user_id"`
	Type      NotificationType `gorm:"type:varchar(30);not null" json:"type"`
	Title     string           `gorm:"type:varchar(200);not null" json:"title"`
	Message   string           `gorm:"type:text;not null" json:"message"`
	Data      json.RawMessage  `gorm:"type:json" json:"data,omitempty"`
	// DedupKey is the single idempotency guard shared by BOTH channels:
	// rows with the same non-null key are created once, so neither the
	// in-app record nor its SMS can ever be duplicated for one event.
	// NULL (recurring/ad-hoc notices) is never deduplicated.
	DedupKey *string `gorm:"type:varchar(200);uniqueIndex" json:"dedup_key,omitempty"`
	// SMS channel state for this notification: none (not attempted —
	// type not SMS-enabled or toggle off), sent, failed, skipped
	// (missing/invalid number, provider off).
	SMSStatus string  `gorm:"type:varchar(20);not null;default:'none';index" json:"sms_status"`
	SMSError  *string `gorm:"type:text" json:"sms_error,omitempty"`
	ReadAt    *time.Time       `gorm:"index:idx_user_read" json:"read_at,omitempty"`
	CreatedAt time.Time        `gorm:"autoCreateTime" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// SMS channel states.
const (
	SMSNone    = "none"
	SMSSent    = "sent"
	SMSFailed  = "failed"
	SMSSkipped = "skipped"
)

// NotificationSMSPref is the per-type SMS opt-in for a group. Absent rows
// fall back to built-in defaults (due reminders + fine notices on).
type NotificationSMSPref struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID   string    `gorm:"type:uuid;not null;uniqueIndex:idx_sms_pref_group_type,priority:1" json:"group_id"`
	NotifType string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_sms_pref_group_type,priority:2" json:"notif_type"`
	Enabled   bool      `gorm:"not null;default:false" json:"enabled"`
	UpdatedBy *string   `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
