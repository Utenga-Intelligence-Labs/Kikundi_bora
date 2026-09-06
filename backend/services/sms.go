package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"

	"gorm.io/gorm/clause"
)

// ── Provider abstraction (Part 1.1) ──────────────────────────────────────────
// Business logic NEVER calls a vendor SDK directly — it calls SMSProvider.
// Swapping in a real vendor later = one new implementation + SMS_PROVIDER
// env value. Nothing else changes.

// SMSProvider sends one text message.
type SMSProvider interface {
	// Name identifies the provider in logs/settings ("noop", "twilio", ...).
	Name() string
	SendSMS(ctx context.Context, phoneE164 string, message string) error
}

// LoggingSMSProvider is the placeholder: it logs instead of sending so the
// whole channel can be built, tested and demoed before a real vendor exists.
type LoggingSMSProvider struct{}

func (LoggingSMSProvider) Name() string { return "noop" }
func (LoggingSMSProvider) SendSMS(ctx context.Context, phoneE164 string, message string) error {
	log.Printf("SMS[noop] to=%s msg=%q", phoneE164, message)
	return nil
}

// smsProvider is the process-wide instance, set by InitSMS (default noop).
var smsProvider SMSProvider = LoggingSMSProvider{}

// InitSMS selects the provider from config. Unknown names fall back to noop
// with a loud log so a typo can never silently black-hole messages.
func InitSMS() {
	switch strings.ToLower(strings.TrimSpace(config.AppConfig.SMSProvider)) {
	case "", "noop", "logging":
		smsProvider = LoggingSMSProvider{}
	default:
		log.Printf("WARN: unknown SMS_PROVIDER %q — falling back to noop logger",
			config.AppConfig.SMSProvider)
		smsProvider = LoggingSMSProvider{}
	}
}

// SetSMSProvider swaps the implementation (tests use this to prove the
// interface boundary with a mock "real" provider).
func SetSMSProvider(p SMSProvider) { smsProvider = p }

// ── Phone normalization (Part 1.2) ───────────────────────────────────────────

var tzMobileRe = regexp.MustCompile(`^\+255[67]\d{8}$`)

// NormalizeTanzanianPhone accepts the formats Tanzanians actually type and
// returns strict E.164 (+255...). Rejects landlines, short codes and garbage
// instead of letting them fail silently at send time.
func NormalizeTanzanianPhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	if s == "" {
		return "", errors.New("namba ya simu haipo")
	}
	switch {
	case strings.HasPrefix(s, "+"):
		// already international — validate as-is
	case strings.HasPrefix(s, "00255"):
		s = "+" + s[2:]
	case strings.HasPrefix(s, "255"):
		s = "+" + s
	case strings.HasPrefix(s, "0"):
		s = "+255" + s[1:]
	default:
		// bare mobile without trunk, e.g. 710123456
		s = "+255" + s
	}
	if !tzMobileRe.MatchString(s) {
		return "", errors.New("namba ya simu si sahihi ya Tanzania (+255...)")
	}
	return s, nil
}

// ── Channel policy (Part 1.3–1.4) ────────────────────────────────────────────

// defaultSMSTypes are SMS-enabled when no explicit pref row exists. Kept
// deliberately narrow: SMS costs money per message.
var defaultSMSTypes = map[models.NotificationType]bool{
	models.NotifContributionDue: true,
	models.NotifFineIssued:      true,
}

// SMSProviderName reports the configured provider (for the settings UI note).
func SMSProviderName() string { return smsProvider.Name() }

// SMSProviderReal reports whether a real vendor (not the noop logger) is wired.
func SMSProviderReal() bool { return smsProvider.Name() != "noop" }

// SMDPrefOrDefault is the exported per-type opt-in lookup.
func SMDPrefOrDefault(groupID string, t models.NotificationType) bool {
	return smsEnabledForType(groupID, t)
}
func smsEnabledForType(groupID string, t models.NotificationType) bool {
	var pref models.NotificationSMSPref
	if err := database.DB.Where("group_id = ? AND notif_type = ?", groupID, string(t)).
		First(&pref).Error; err == nil {
		return pref.Enabled
	}
	return defaultSMSTypes[t]
}

// groupSMSOn reads the group master switch (default off).
func groupSMSOn(groupID string) bool {
	var g models.Group
	if err := database.DB.Select("sms_notifications_enabled").First(&g, "id = ?", groupID).Error; err != nil {
		return false
	}
	return g.SMSNotificationsEnabled
}

// ── Dispatch (Part 1.3, 1.5) ─────────────────────────────────────────────────
//
// The SAME dedup_key guard protects both channels: the notification row is
// inserted with ON CONFLICT DO NOTHING; only the inserter (RowsAffected==1)
// may attempt SMS. A duplicate event therefore produces neither a second
// in-app row nor a second SMS — the guards cannot drift apart because there
// is only one.

// NotifyUserDedup creates the in-app notification guarded by dedupKey.
// Empty dedupKey = no guard (legacy ad-hoc behavior, unchanged).
func NotifyUserDedup(userID string, notifType models.NotificationType, title, message, dedupKey string) *models.Notification {
	return createNotificationRow(userID, notifType, title, message, dedupKey)
}

// NotifyUserSMS creates the in-app notification under the shared guard and,
// only if this call actually created the row, attempts the SMS leg on the
// same row. Failures (bad number, provider error) are recorded on the row
// and logged — the caller's action is never failed or rolled back.
func NotifyUserSMS(groupID, userID string, notifType models.NotificationType, title, message, dedupKey string) *models.Notification {
	n := createNotificationRow(userID, notifType, title, message, dedupKey)
	if n == nil {
		return nil // duplicate event — same guard already consumed it
	}
	dispatchSMS(groupID, n)
	return n
}

func createNotificationRow(userID string, notifType models.NotificationType, title, message, dedupKey string) *models.Notification {
	n := models.Notification{
		UserID: userID, Type: notifType, Title: title, Message: message,
		SMSStatus: models.SMSNone,
	}
	if dedupKey != "" {
		n.DedupKey = &dedupKey
		res := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&n)
		if res.Error != nil {
			log.Printf("ERROR: notification create for user %s: %v", userID, res.Error)
			return nil
		}
		if res.RowsAffected == 0 {
			return nil // duplicate — guard consumed by an earlier call
		}
		return &n
	}
	if err := database.DB.Create(&n).Error; err != nil {
		log.Printf("ERROR: notification create for user %s: %v", userID, err)
		return nil
	}
	return &n
}

// dispatchSMS evaluates policy and sends exactly once for the row.
func dispatchSMS(groupID string, n *models.Notification) {
	mark := func(status, errMsg string) {
		updates := map[string]interface{}{"sms_status": status}
		if errMsg != "" {
			updates["sms_error"] = errMsg
		}
		database.DB.Model(n).Updates(updates)
	}
	if !groupSMSOn(groupID) {
		mark(models.SMSSkipped, "sms off for group")
		return
	}
	if !smsEnabledForType(groupID, n.Type) {
		mark(models.SMSSkipped, "type not sms-enabled")
		return
	}
	var user models.User
	if err := database.DB.Select("phone").First(&user, "id = ?", n.UserID).Error; err != nil {
		mark(models.SMSSkipped, "user not found")
		log.Printf("WARN: SMS skip notif %s: user %s not found", n.ID, n.UserID)
		return
	}
	phone, err := NormalizeTanzanianPhone(user.Phone)
	if err != nil {
		mark(models.SMSSkipped, "invalid phone: "+user.Phone)
		log.Printf("WARN: SMS skip notif %s user %s: %v", n.ID, n.UserID, err)
		return
	}
	text := n.Title + ": " + n.Message
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := smsProvider.SendSMS(ctx, phone, text); err == nil {
		mark(models.SMSSent, "")
		return
	} else {
		log.Printf("WARN: SMS send failed notif %s to %s (%s), retrying once: %v",
			n.ID, phone, smsProvider.Name(), err)
	}
	// Retry once after a short backoff for transient failures.
	time.Sleep(2 * time.Second)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	if err := smsProvider.SendSMS(ctx2, phone, text); err != nil {
		mark(models.SMSFailed, err.Error())
		log.Printf("ERROR: SMS failed notif %s to %s (%s): %v", n.ID, phone, smsProvider.Name(), err)
		return
	}
	mark(models.SMSSent, "")
}

// SendOTPSMS delivers a verification code through the configured provider.
func SendOTPSMS(phoneE164, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return smsProvider.SendSMS(ctx, phoneE164,
		fmt.Sprintf("Msimbo wako wa uthibitisho: %s. Hauhamishiki kwa mtu mwingine.", code))
}

// NotifyUsersSMS is the multi-recipient variant (same per-row guard each).
func NotifyUsersSMS(groupID string, userIDs []string, notifType models.NotificationType, title, message string, dedupKeyFor func(userID string) string) {
	for _, uid := range userIDs {
		NotifyUserSMS(groupID, uid, notifType, title, message, dedupKeyFor(uid))
	}
}
