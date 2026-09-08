package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

func onboardingTestApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()

	app := fiber.New(fiber.Config{AppName: "Kikundi Onboarding Test"})
	api := app.Group("/api/v1")

	authHandler := NewAuthHandler()
	onboardingHandler := NewOnboardingHandler()

	api.Post("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	groups := protected.Group("/groups")
	groups.Get("/:id/onboarding-status", onboardingHandler.Status)
	groups.Patch("/:id/onboarding-complete", middleware.RequireRoles(models.RoleChair), onboardingHandler.Complete)

	members := protected.Group("/members")
	members.Patch("/:id/onboarding-seen", onboardingHandler.MarkOnboardingSeen)

	return app
}

func onboardingLogin(t *testing.T, app *fiber.App, loginID, password string) string {
	t.Helper()
	body := map[string]string{"email": loginID, "password": password}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("login %s: want 200, got %d (%s)", loginID, resp.StatusCode, data)
	}
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	token, _ := m["token"].(string)
	if token == "" {
		t.Fatalf("no token in login response: %s", data)
	}
	return token
}

func currentGroupID(t *testing.T) string {
	t.Helper()
	g, err := database.GetCurrentGroup()
	if err != nil {
		t.Fatalf("current group: %v", err)
	}
	return g.ID
}

// TestOnboardingStatusAndComplete covers the wizard resume payload and the
// completion flag: steps start false on a wiped DB, complete is chair-only,
// and completing flips the flag so the wizard stops auto-opening.
func TestOnboardingStatusAndComplete(t *testing.T) {
	app := onboardingTestApp()
	cleanAndSeed(t)
	gid := currentGroupID(t)

	// cleanAndSeed preserves the group row — reset it to a fresh-group state.
	database.DB.Exec(`UPDATE groups SET onboarding_completed = FALSE, contribution_due_date = NULL, fixed_contribution_amount = NULL`)
	database.DB.Exec(`DELETE FROM payment_methods`)
	database.DB.Exec(`DELETE FROM group_setting_proposals`)

	chairToken := onboardingLogin(t, app, "juma@kikundi.tz", "demo123")
	memberToken := onboardingLogin(t, app, "asha@kikundi.tz", "demo123")

	// Member (not chair) can READ the status.
	code, body := hGet(t, app, "/api/v1/groups/"+gid+"/onboarding-status", memberToken)
	if code != 200 {
		t.Fatalf("status as member: want 200, got %d (%s)", code, body)
	}
	var statusResp struct {
		Data struct {
			OnboardingCompleted bool `json:"onboarding_completed"`
			Steps               struct {
				ContributionProposed bool `json:"contribution_proposed"`
				ContributionApproved bool `json:"contribution_approved"`
				PaymentMethodAdded   bool `json:"payment_method_added"`
				MembersAdded         bool `json:"members_added"`
			} `json:"steps"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &statusResp); err != nil {
		t.Fatalf("status decode: %v (%s)", err, body)
	}
	if statusResp.Data.OnboardingCompleted {
		t.Errorf("fresh group should have onboarding_completed=false")
	}
	if statusResp.Data.Steps.PaymentMethodAdded {
		t.Errorf("no payment methods seeded — payment_method_added should be false")
	}
	if statusResp.Data.Steps.ContributionProposed || statusResp.Data.Steps.ContributionApproved {
		t.Errorf("no proposal/settings seeded — contribution steps should be false")
	}

	// Member can't complete onboarding (chair-only).
	code, _ = hPatch(t, app, "/api/v1/groups/"+gid+"/onboarding-complete", memberToken, nil)
	if code != 403 {
		t.Fatalf("complete as member: want 403, got %d", code)
	}

	// Chair completes → flag flips.
	code, body = hPatch(t, app, "/api/v1/groups/"+gid+"/onboarding-complete", chairToken, nil)
	if code != 200 {
		t.Fatalf("complete as chair: want 200, got %d (%s)", code, body)
	}

	code, body = hGet(t, app, "/api/v1/groups/"+gid+"/onboarding-status", chairToken)
	if code != 200 {
		t.Fatalf("status re-read: want 200, got %d", code)
	}
	if err := json.Unmarshal(body, &statusResp); err != nil {
		t.Fatalf("status decode: %v", err)
	}
	if !statusResp.Data.OnboardingCompleted {
		t.Errorf("after complete, onboarding_completed should be true")
	}

	// Foreign group id → 404.
	code, _ = hGet(t, app, "/api/v1/groups/00000000-0000-0000-0000-000000000000/onboarding-status", chairToken)
	if code != 404 {
		t.Errorf("foreign group status: want 404, got %d", code)
	}
}

// TestOnboardingMarkSeen covers the member tour flag: self can set it,
// another member cannot, and it persists on the linked user.
func TestOnboardingMarkSeen(t *testing.T) {
	app := onboardingTestApp()
	cleanAndSeed(t)

	u, m := makePendingFixture(t, "0700000201", "KKK-T02", models.MemberApprovalApproved)
	database.DB.Model(&m).Update("is_active", true)

	otherToken := onboardingLogin(t, app, "asha@kikundi.tz", "demo123")
	ownToken := onboardingLogin(t, app, u.Phone, "testpass123")

	// Another member can't flip someone else's flag.
	code, _ := hPatch(t, app, "/api/v1/members/"+m.ID+"/onboarding-seen", otherToken, nil)
	if code != 403 {
		t.Fatalf("seen as other member: want 403, got %d", code)
	}

	// Self flips it.
	code, body := hPatch(t, app, "/api/v1/members/"+m.ID+"/onboarding-seen", ownToken, nil)
	if code != 200 {
		t.Fatalf("seen as self: want 200, got %d (%s)", code, body)
	}

	var u2 models.User
	if err := database.DB.First(&u2, "id = ?", u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !u2.OnboardingSeen {
		t.Errorf("onboarding_seen should be true after PATCH")
	}

	// Unknown member id → 404.
	code, _ = hPatch(t, app, "/api/v1/members/00000000-0000-0000-0000-000000000000/onboarding-seen", ownToken, nil)
	if code != 404 {
		t.Errorf("unknown member: want 404, got %d", code)
	}
}

// hPatch is a tiny JSON PATCH helper (integration_test.go only ships GET/POST).
func hPatch(t *testing.T, app *fiber.App, path, token string, body map[string]string) (int, []byte) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest("PATCH", path, rd)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", path, err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, data
}
