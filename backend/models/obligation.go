package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ── Offence kinds ────────────────────────────────────────────────────────────
const (
	OffenceLateContribution = "late_contribution" // scheduler-created after grace
	OffenceMeetingAbsence   = "meeting_absence"   // katibu-triggered from attendance
	OffenceMeetingLate      = "meeting_late"      // katibu-triggered from attendance
	OffenceOther            = "other"             // manual/misc, via collect flow only
)

func IsValidOffenceKind(k string) bool {
	return k == OffenceLateContribution || k == OffenceMeetingAbsence ||
		k == OffenceMeetingLate || k == OffenceOther
}

// ── Offence-type lifecycle ───────────────────────────────────────────────────
const (
	OffencePending  = "pending"  // proposed by mwenyekiti, awaiting katibu
	OffenceActive   = "active"   // approved — creates fines
	OffenceInactive = "inactive" // deactivated — no new fines, old ones stand
)

// FineOffenceType is a mwenyekiti-defined offence (per group). Only ACTIVE
// types create fines; deactivation never touches already-issued fines.
type FineOffenceType struct {
	ID               string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID          string    `gorm:"type:uuid;not null;index" json:"group_id"`
	Kind             string    `gorm:"type:varchar(30);not null;index" json:"kind"`
	Name             string    `gorm:"type:varchar(150);not null" json:"name"`
	FineType         string    `gorm:"type:varchar(20);not null;default:'fixed'" json:"fine_type"`
	FineAmount       *decimal.Decimal `gorm:"type:decimal(15,2)" json:"fine_amount,omitempty"`
	FinePercentage   *decimal.Decimal `gorm:"type:decimal(7,2)" json:"fine_percentage,omitempty"`
	GracePeriodDays  int       `gorm:"not null;default:0" json:"grace_period_days"`
	Status           string    `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	CreatedBy        string    `gorm:"type:uuid;not null" json:"created_by"`
	ApprovedBy       *string   `gorm:"type:uuid" json:"approved_by,omitempty"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Group *Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// ── Fine waiver lifecycle ────────────────────────────────────────────────────
const (
	WaiverNone     = "none"
	WaiverPending  = "pending"  // proposed by mwenyekiti, awaiting katibu
	WaiverApproved = "approved" // approved by katibu → fine.status = waived
	WaiverRejected = "rejected"
)

// ── Contribution cycles ──────────────────────────────────────────────────────
const (
	CyclePaid    = "paid"
	CyclePartial = "partial"
	CycleUnpaid  = "unpaid"
)

// ContributionCycle tracks per member, per contribution cycle whether the
// expected fixed_contribution_amount was paid. ExpectedAmount is a SNAPSHOT
// taken at refresh time — later settings changes don't rewrite history.
type ContributionCycle struct {
	ID             string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID        string          `gorm:"type:uuid;not null;index:idx_cycles_group_member_label,priority:1" json:"group_id"`
	MemberID       string          `gorm:"type:uuid;not null;index:idx_cycles_group_member_label,priority:2" json:"member_id"`
	CycleLabel     string          `gorm:"type:varchar(32);not null;index:idx_cycles_group_member_label,priority:3" json:"cycle_label"`
	DueDate        time.Time       `gorm:"type:date;not null;index" json:"due_date"`
	ExpectedAmount decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"expected_amount"`
	PaidAmount     decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"paid_amount"`
	Status         string          `gorm:"type:varchar(20);not null;default:'unpaid';index" json:"status"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	Member *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
}

func IsValidCycleStatus(s string) bool {
	return s == CyclePaid || s == CyclePartial || s == CycleUnpaid
}

// ── Meetings & attendance ────────────────────────────────────────────────────
const (
	AttendancePresent = "present"
	AttendanceAbsent  = "absent"
	AttendanceLate    = "late"
)

func IsValidAttendanceStatus(s string) bool {
	return s == AttendancePresent || s == AttendanceAbsent || s == AttendanceLate
}

// Meeting is a group meeting whose attendance (marked by katibu) can trigger
// meeting-kind fines for absent/late members.
type Meeting struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	GroupID     string    `gorm:"type:uuid;not null;index" json:"group_id"`
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	MeetingDate time.Time `gorm:"type:date;not null;index" json:"meeting_date"`
	Notes       *string   `gorm:"type:text" json:"notes,omitempty"`
	CreatedBy   string    `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Group *Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// MeetingAttendance marks one member's presence. Fined=true once a fine (if
// any applied) was created, so re-triggering stays idempotent.
type MeetingAttendance struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MeetingID string    `gorm:"type:uuid;not null;index:idx_attend_meeting_member,priority:1" json:"meeting_id"`
	MemberID  string    `gorm:"type:uuid;not null;index:idx_attend_meeting_member,priority:2" json:"member_id"`
	Status    string    `gorm:"type:varchar(20);not null;default:'present'" json:"status"`
	Fined     bool      `gorm:"not null;default:false" json:"fined"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Member *Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
}
