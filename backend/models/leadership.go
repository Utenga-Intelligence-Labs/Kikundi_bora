package models

import "time"

// LeadershipRole represents a leadership role within the kikundi.
type LeadershipRole string

const (
	LeadershipChair     LeadershipRole = "MWENYEKITI"
	LeadershipTreasurer LeadershipRole = "HAZINA"
	LeadershipSecretary LeadershipRole = "KATIBU"
)

// LeadershipPosition links a member (not user) to a leadership role for a term.
// A single member can hold multiple roles concurrently (e.g. MWENYEKITI + HAZINA).
// The partial unique index (member_id, role) WHERE is_current=true enforces
// one active slot per role per member.
type LeadershipPosition struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MemberID  string          `gorm:"type:uuid;not null;index" json:"member_id"`
	Role      LeadershipRole  `gorm:"type:varchar(20);not null" json:"role"`
	TermStart time.Time       `gorm:"type:date;not null;default:current_date" json:"term_start"`
	TermEnd   *time.Time      `gorm:"type:date" json:"term_end,omitempty"`
	IsCurrent bool            `gorm:"not null;default:true;index" json:"is_current"`
	CreatedAt time.Time       `gorm:"autoCreateTime" json:"created_at"`

	Member Member `gorm:"foreignKey:MemberID" json:"member,omitempty"`
}

// AllLeadershipRoles returns the valid leadership roles.
func AllLeadershipRoles() []LeadershipRole {
	return []LeadershipRole{LeadershipChair, LeadershipTreasurer, LeadershipSecretary}
}

// ValidLeadershipRole checks if a string is a valid leadership role.
func ValidLeadershipRole(r string) bool {
	switch LeadershipRole(r) {
	case LeadershipChair, LeadershipTreasurer, LeadershipSecretary:
		return true
	}
	return false
}
