package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"

	"golang.org/x/crypto/bcrypt"
)

// ── OTP verification (Part 2) ────────────────────────────────────────────────
// STATUS: DISABLED by default (OTPVerificationEnabled=false). Nothing in the
// auth handlers calls this yet — the model, logic and endpoint below exist
// so re-enabling later is a flag flip + wiring, not a rewrite. DO NOT DELETE.
//
// Re-enable runbook (for the PR description):
//  1. Set OTP_VERIFICATION_ENABLED=true in the backend .env (and document it
//     in .env.example) and restart the API.
//  2. Wire the login response: after a correct password, if the flag is on,
//     call IssueOTPChallenge(user) and return 202 {otp_required:true,
//     challenge_id} INSTEAD of a session token; the client then posts the
//     code to POST /api/v1/auth/verify-otp.
//  3. Point an SMSProvider at a real vendor (SMS_PROVIDER + SMS_API_KEY +
//     SMS_SENDER_ID) so codes actually arrive; until then codes are only
//     logged server-side.
//  4. Frontend: register the preserved OtpForm component on a route and
//     branch the login page on otp_required.

// OTPEnabled reports whether verification is switched on. Read live (not
// cached) so tests can flip config.AppConfig.OTPVerificationEnabled.
func OTPEnabled() bool {
	return config.AppConfig != nil && config.AppConfig.OTPVerificationEnabled
}

// IssueOTPChallenge creates a fresh 6-digit challenge for a user, voiding
// any previous unconsumed ones. Returns the challenge and the PLAINTEXT
// code (deliver it once via SMS; only the bcrypt hash is stored).
func IssueOTPChallenge(userID, purpose string) (*models.OTPChallenge, string, error) {
	digits := make([]byte, 6)
	if _, err := rand.Read(digits); err != nil {
		return nil, "", err
	}
	code := ""
	for _, b := range digits {
		code += fmt.Sprintf("%d", int(b)%10)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	database.DB.Model(&models.OTPChallenge{}).
		Where("user_id = ? AND consumed = FALSE", userID).
		Update("consumed", true)
	ch := models.OTPChallenge{
		UserID: userID, Purpose: purpose, CodeHash: string(hash),
		ExpiresAt: time.Now().Add(models.OTPCodeTTL),
	}
	if err := database.DB.Create(&ch).Error; err != nil {
		return nil, "", err
	}
	return &ch, code, nil
}

var (
	ErrOTPNotFound  = errors.New("changamoto haijapatikana")
	ErrOTPExpired   = errors.New("msimbo umeisha muda")
	ErrOTPConsumed  = errors.New("msimbo umeshatumika")
	ErrOTPBlocked   = errors.New("majaribio mengi — omba msimbo mpya")
	ErrOTPInvalid   = errors.New("msimbo si sahihi")
	ErrOTPDisabled  = errors.New("uthibitisho wa OTP umezimwa")
)

// VerifyOTPChallenge checks a code. Wrong codes increment attempts (voiding
// the challenge past MaxOTPAttempts); a correct code consumes it. The
// caller must refuse to proceed unless err == nil — and while the feature
// flag is off, callers must not reach this function at all.
func VerifyOTPChallenge(challengeID, code string) (*models.OTPChallenge, error) {
	var ch models.OTPChallenge
	if err := database.DB.First(&ch, "id = ?", challengeID).Error; err != nil {
		return nil, ErrOTPNotFound
	}
	if ch.Consumed {
		return nil, ErrOTPConsumed
	}
	if time.Now().After(ch.ExpiresAt) {
		return nil, ErrOTPExpired
	}
	if ch.Attempts >= models.MaxOTPAttempts {
		database.DB.Model(&ch).Update("consumed", true)
		return nil, ErrOTPBlocked
	}
	if err := bcrypt.CompareHashAndPassword([]byte(ch.CodeHash), []byte(code)); err != nil {
		database.DB.Model(&ch).Update("attempts", ch.Attempts+1)
		return nil, ErrOTPInvalid
	}
	database.DB.Model(&ch).Updates(map[string]interface{}{"consumed": true})
	return &ch, nil
}
