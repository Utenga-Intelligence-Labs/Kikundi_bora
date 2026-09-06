package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

// mockSMSProvider is the test stand-in for a "real" vendor: it proves the
// SMSProvider interface boundary without changing any calling code.
type mockSMSProvider struct {
	mu        sync.Mutex
	sends     []mockSMSSend
	failTimes int
}

type mockSMSSend struct {
	Phone   string
	Message string
}

func (m *mockSMSProvider) Name() string { return "mock" }

func (m *mockSMSProvider) SendSMS(_ context.Context, phone, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sends = append(m.sends, mockSMSSend{Phone: phone, Message: message})
	if m.failTimes > 0 {
		m.failTimes--
		return errors.New("mock transient failure")
	}
	return nil
}

func (m *mockSMSProvider) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sends)
}

func smsOTPTestApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()

	app := fiber.New(fiber.Config{AppName: "Kikundi SMS OTP Test"})
	authHandler := NewAuthHandler()
	api := app.Group("/api/v1")
	api.Post("/auth/login", authHandler.Login)
	api.Post("/auth/verify-otp", authHandler.VerifyOTP)
	protected := api.Group("")
	protected.Use(middleware.AuthRequired)
	groups := protected.Group("/groups")
	chairAdmin := middleware.RequireRoles(models.RoleChair, models.RoleAdmin)
	groups.Get("/:id/notification-settings", chairAdmin, GetNotificationSettings)
	groups.Put("/:id/notification-settings", chairAdmin, UpdateNotificationSettings)
	return app
}

func smsTestUser(t *testing.T, phone string) models.User {
	t.Helper()
	u := models.User{
		Name: "SMS Test User", Phone: phone, Role: models.RoleMember,
		Status: models.UserStatusActive, IsActive: true,
	}
	// Unique phone per test to avoid cross-test collisions.
	u.Phone = phone
	if err := database.DB.Create(&u).Error; err != nil {
		t.Fatalf("create sms user: %v", err)
	}
	return u
}

func smsTestGroup(t *testing.T, smsOn bool) models.Group {
	t.Helper()
	var g models.Group
	database.DB.First(&g)
	g.SMSNotificationsEnabled = smsOn
	database.DB.Save(&g)
	return g
}

func TestSMSDispatchExactlyOnce(t *testing.T) {
	app := smsOTPTestApp()
	cleanAndSeed(t)
	mock := &mockSMSProvider{}
	services.SetSMSProvider(mock)
	defer services.SetSMSProvider(services.LoggingSMSProvider{})

	g := smsTestGroup(t, true)
	u := smsTestUser(t, "0710123456")
	database.DB.Create(&models.NotificationSMSPref{
		GroupID: g.ID, NotifType: string(models.NotifContributionDue), Enabled: true,
	})

	key := "test-dedup-once"
	n1 := services.NotifyUserSMS(g.ID, u.ID, models.NotifContributionDue, "T", "M", key)
	n2 := services.NotifyUserSMS(g.ID, u.ID, models.NotifContributionDue, "T", "M", key)
	if n1 == nil {
		t.Fatalf("first dispatch returned nil")
	}
	if n2 != nil {
		t.Errorf("duplicate event created a second row — guard failed")
	}
	var count int64
	database.DB.Model(&models.Notification{}).Where("dedup_key = ?", key).Count(&count)
	if count != 1 {
		t.Errorf("dedup_key rows = %d, want 1", count)
	}
	if mock.count() != 1 {
		t.Errorf("provider sends = %d, want exactly 1", mock.count())
	}
	var row models.Notification
	database.DB.First(&row, "dedup_key = ?", key)
	if row.SMSStatus != models.SMSSent {
		t.Errorf("sms_status = %q, want sent", row.SMSStatus)
	}
	_ = app
}

func TestSMSToggleOffStopsSMSOnly(t *testing.T) {
	smsOTPTestApp()
	cleanAndSeed(t)
	mock := &mockSMSProvider{}
	services.SetSMSProvider(mock)
	defer services.SetSMSProvider(services.LoggingSMSProvider{})

	g := smsTestGroup(t, false) // master switch OFF
	u := smsTestUser(t, "0710123457")
	database.DB.Create(&models.NotificationSMSPref{
		GroupID: g.ID, NotifType: string(models.NotifContributionDue), Enabled: true,
	})

	n := services.NotifyUserSMS(g.ID, u.ID, models.NotifContributionDue, "T", "M", "test-toggle-off")
	if n == nil {
		t.Fatalf("in-app row must still be created when SMS is off")
	}
	if mock.count() != 0 {
		t.Errorf("provider sends = %d, want 0", mock.count())
	}
	var row models.Notification
	database.DB.First(&row, "id = ?", n.ID)
	if row.SMSStatus != models.SMSSkipped {
		t.Errorf("sms_status = %q, want skipped", row.SMSStatus)
	}
}

func TestSMSInvalidPhoneSkipped(t *testing.T) {
	smsOTPTestApp()
	cleanAndSeed(t)
	mock := &mockSMSProvider{}
	services.SetSMSProvider(mock)
	defer services.SetSMSProvider(services.LoggingSMSProvider{})

	g := smsTestGroup(t, true)
	u := smsTestUser(t, "not-a-number")
	database.DB.Create(&models.NotificationSMSPref{
		GroupID: g.ID, NotifType: string(models.NotifFineIssued), Enabled: true,
	})

	n := services.NotifyUserSMS(g.ID, u.ID, models.NotifFineIssued, "T", "M", "test-bad-phone")
	if n == nil {
		t.Fatalf("in-app row must still be created on bad phone")
	}
	if mock.count() != 0 {
		t.Errorf("provider sends = %d, want 0", mock.count())
	}
	var row models.Notification
	database.DB.First(&row, "id = ?", n.ID)
	if row.SMSStatus != models.SMSSkipped {
		t.Errorf("sms_status = %q, want skipped", row.SMSStatus)
	}
}

func TestSMSRetryOnceThenFail(t *testing.T) {
	smsOTPTestApp()
	cleanAndSeed(t)
	mock := &mockSMSProvider{failTimes: 100} // always fails
	services.SetSMSProvider(mock)
	defer services.SetSMSProvider(services.LoggingSMSProvider{})

	g := smsTestGroup(t, true)
	u := smsTestUser(t, "0710123458")
	database.DB.Create(&models.NotificationSMSPref{
		GroupID: g.ID, NotifType: string(models.NotifContributionDue), Enabled: true,
	})

	n := services.NotifyUserSMS(g.ID, u.ID, models.NotifContributionDue, "T", "M", "test-retry")
	if n == nil {
		t.Fatalf("row must exist even when provider fails")
	}
	if mock.count() != 2 {
		t.Errorf("provider sends = %d, want 2 (initial + one retry)", mock.count())
	}
	var row models.Notification
	database.DB.First(&row, "id = ?", n.ID)
	if row.SMSStatus != models.SMSFailed {
		t.Errorf("sms_status = %q, want failed", row.SMSStatus)
	}
	if row.SMSError == nil || *row.SMSError == "" {
		t.Errorf("sms_error should record the failure reason")
	}
}

func TestOTPDisabledByDefault(t *testing.T) {
	app := smsOTPTestApp()
	cleanAndSeed(t)
	if config.AppConfig.OTPVerificationEnabled {
		t.Fatalf("OTP flag must default to false in tests")
	}

	// Verify endpoint is effectively absent.
	code, _ := hReq(t, app, "POST", "/api/v1/auth/verify-otp",
		map[string]string{"challenge_id": "00000000-0000-0000-0000-000000000000", "code": "123456"}, "")
	if code != 404 {
		t.Errorf("verify-otp with flag off = %d, want 404", code)
	}

	// Normal login succeeds with no OTP step and creates no challenge.
	code, body := hPost(t, app, "/api/v1/auth/login",
		map[string]string{"email": "fatuma@kikundi.tz", "password": "demo123"}, "")
	if code != 200 {
		t.Fatalf("login with flag off = %d %s, want 200", code, body)
	}
	var r struct {
		Token string `json:"token"`
	}
	json.Unmarshal(body, &r)
	if r.Token == "" {
		t.Errorf("login must return a session token when OTP is off")
	}
	var challenges int64
	database.DB.Model(&models.OTPChallenge{}).Count(&challenges)
	if challenges != 0 {
		t.Errorf("challenges = %d, want 0 when OTP is off", challenges)
	}
}

func TestOTPEnabledFlow(t *testing.T) {
	app := smsOTPTestApp()
	cleanAndSeed(t)
	config.AppConfig.OTPVerificationEnabled = true
	defer func() { config.AppConfig.OTPVerificationEnabled = false }()

	// Password-correct login now yields a challenge, not a session.
	code, body := hPost(t, app, "/api/v1/auth/login",
		map[string]string{"email": "fatuma@kikundi.tz", "password": "demo123"}, "")
	if code != 202 {
		t.Fatalf("login with flag on = %d %s, want 202", code, body)
	}
	var pending struct {
		OTPRequired bool   `json:"otp_required"`
		ChallengeID string `json:"challenge_id"`
	}
	json.Unmarshal(body, &pending)
	if !pending.OTPRequired || pending.ChallengeID == "" {
		t.Fatalf("202 must carry otp_required + challenge_id: %s", body)
	}

	// Wrong code is rejected.
	code, _ = hPost(t, app, "/api/v1/auth/verify-otp",
		map[string]string{"challenge_id": pending.ChallengeID, "code": "000000"}, "")
	if code != 401 {
		t.Errorf("wrong code = %d, want 401", code)
	}

	// A fresh service-issued challenge verifies end-to-end (proves the
	// endpoint consumes real codes, not fixtures).
	var user models.User
	database.DB.Where("email = ?", "fatuma@kikundi.tz").First(&user)
	ch, plaintext, err := services.IssueOTPChallenge(user.ID, models.OTPPurposeLogin)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	code, body = hPost(t, app, "/api/v1/auth/verify-otp",
		map[string]string{"challenge_id": ch.ID, "code": plaintext}, "")
	if code != 200 {
		t.Fatalf("correct code = %d %s, want 200", code, body)
	}
	var r struct {
		Token string `json:"token"`
	}
	json.Unmarshal(body, &r)
	if r.Token == "" {
		t.Errorf("verify must return a session token")
	}

	// Reuse of a consumed challenge is rejected.
	code, _ = hPost(t, app, "/api/v1/auth/verify-otp",
		map[string]string{"challenge_id": ch.ID, "code": plaintext}, "")
	if code != 401 {
		t.Errorf("reused code = %d, want 401", code)
	}
}
