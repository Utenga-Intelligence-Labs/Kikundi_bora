package models

import (
	"time"
)

// OTP purposes.
const (
	OTPPurposeLogin    = "login"
	OTPPurposeRegister = "register"
)

// OTPChallenge is a single-use verification code challenge.
//
// PART 2 STATUS: the whole OTP flow is DISABLED by default
// (OTPVerificationEnabled=false) and nothing in the auth handlers invokes
// it yet — this model, the service in services/otp.go and the verify
// endpoint exist so re-enabling later is a flag flip + wiring the login
// response, not a rewrite. DO NOT DELETE.
type OTPChallenge struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Purpose   string    `gorm:"type:varchar(20);not null" json:"purpose"`
	CodeHash  string    `gorm:"type:varchar(255);not null" json:"-"`
	Attempts  int       `gorm:"not null;default:0" json:"attempts"`
	Consumed  bool      `gorm:"not null;default:false;index" json:"consumed"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// MaxOTPAttempts bounds guessing before the challenge is voided.
const MaxOTPAttempts = 5

// OTPCodeTTL is how long an issued code stays valid.
const OTPCodeTTL = 10 * time.Minute
