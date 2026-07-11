package models

import (
	"encoding/json"
	"time"
)

type AuditAction string

const (
	AuditCreate             AuditAction = "CREATE"
	AuditUpdate             AuditAction = "UPDATE"
	AuditDelete             AuditAction = "DELETE"
	AuditLogin              AuditAction = "LOGIN"
	AuditLogout             AuditAction = "LOGOUT"
	AuditApprove            AuditAction = "APPROVE"
	AuditReject             AuditAction = "REJECT"
	AuditCommitteeAppoint   AuditAction = "COMMITTEE_APPOINT"
	AuditCommitteeRemove    AuditAction = "COMMITTEE_REMOVE"
	AuditLoanReview            AuditAction = "LOAN_REVIEW"
	AuditLoanSubmitReview      AuditAction = "LOAN_SUBMIT_REVIEW"
	AuditCreateWelfareEvent    AuditAction = "CREATE_WELFARE_EVENT"
	AuditApproveWelfareEvent   AuditAction = "APPROVE_WELFARE_EVENT"
	AuditRejectWelfareEvent    AuditAction = "REJECT_WELFARE_EVENT"
	AuditRecordWelfarePayment  AuditAction = "RECORD_WELFARE_PAYMENT"
	AuditCompleteWelfareEvent  AuditAction = "COMPLETE_WELFARE_EVENT"
	AuditUserCreated           AuditAction = "USER_CREATED"
	AuditUserApproved          AuditAction = "USER_APPROVED"
	AuditUserRejected          AuditAction = "USER_REJECTED"
	AuditPasswordSet           AuditAction = "PASSWORD_SET"
	AuditAdminOverride         AuditAction = "ADMIN_OVERRIDE"
)

type AuditLog struct {
	ID        string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    *string          `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Action    AuditAction      `gorm:"type:varchar(20);not null" json:"action"`
	TableName string           `gorm:"type:varchar(50);not null;index:idx_table_record" json:"table_name"`
	RecordID  *string          `gorm:"type:uuid;index:idx_table_record" json:"record_id,omitempty"`
	OldValues json.RawMessage  `gorm:"type:json" json:"old_values,omitempty"`
	NewValues json.RawMessage  `gorm:"type:json" json:"new_values,omitempty"`
	IPAddress *string          `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent *string          `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt time.Time        `gorm:"autoCreateTime;index" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
