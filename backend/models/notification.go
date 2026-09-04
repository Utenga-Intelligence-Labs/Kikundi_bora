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
	ReadAt    *time.Time       `gorm:"index:idx_user_read" json:"read_at,omitempty"`
	CreatedAt time.Time        `gorm:"autoCreateTime" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
