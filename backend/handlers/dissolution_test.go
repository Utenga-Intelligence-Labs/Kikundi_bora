package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

func dissApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	services.InitEmail()
	app := fiber.New()
	app.Use(middleware.SetupCORS())
	api := app.Group("/api/v1")
	ah := NewAuthHandler()
	api.Post("/auth/login", ah.Login)
	protected := api.Group("")
	protected.Use(middleware.AuthRequired)
	dh := NewDissolutionHandler()
	protected.Post("/groups/:id/dissolution-proposals", middleware.RequireRoles(models.RoleChair, models.RoleSecretary), dh.Propose)
	protected.Get("/groups/:id/dissolution-proposals", dh.ListByGroup)
	protected.Post("/dissolution-proposals/:id/vote", dh.Vote)
	protected.Get("/dissolution-proposals/:id", dh.Get)
	protected.Post("/dissolution-proposals/:id/execute", middleware.RequireRoles(models.RoleChair, models.RoleSecretary), dh.Execute)
	protected.Get("/dissolution-proposals/:id/payouts", dh.ListPayouts)
	protected.Patch("/dissolution-payouts/:id/mark-paid", middleware.RequireRoles(models.RoleTreasurer), dh.MarkPaid)
	// blocked routes
	mh := NewMemberHandler()
	ch := NewContributionHandler()
	lh := NewLoanHandler()
	protected.Post("/members", mh.Create)
	protected.Post("/contributions", middleware.RequirePosition(models.PositionTreasurer), ch.Create)
	protected.Post("/loans/apply", lh.Apply)
	return app
}

func dissLogin(t *testing.T, app *fiber.App, email string) string {
	t.Helper()
	body := map[string]string{"email": email, "password": "demo123"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("login %s: %d %s", email, resp.StatusCode, data)
	}
	var r struct{ Token string }
	json.Unmarshal(data, &r)
	return r.Token
}
func dissPost(t *testing.T, app *fiber.App, path string, body interface{}, token string) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest("POST", path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test POST %s: %v", path, err)
	}
	if resp == nil {
		t.Fatalf("nil resp POST %s", path)
	}
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}
func dissGet(t *testing.T, app *fiber.App, path, token string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test GET %s: %v", path, err)
	}
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}
func dissPatch(t *testing.T, app *fiber.App, path string, body interface{}, token string) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest("PATCH", path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test PATCH %s: %v", path, err)
	}
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func TestDissolutionVotingAndThreshold(t *testing.T) {
	app := dissApp()
	cleanAndSeed(t)
	chair := dissLogin(t, app, "juma@kikundi.tz")
	memberTok := dissLogin(t, app, "asha@kikundi.tz")
	secretary := dissLogin(t, app, "rashidi@kikundi.tz")

	var g models.Group
	database.DB.First(&g)

	deadline := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	code, body := dissPost(t, app, "/api/v1/groups/"+g.ID+"/dissolution-proposals", map[string]interface{}{
		"cycle_span_years": 1, "voting_deadline": deadline,
	}, chair)
	if code != 201 {
		t.Fatalf("propose %d %s", code, body)
	}
	var pr struct{ Data models.GroupDissolutionProposal `json:"data"` }
	json.Unmarshal(body, &pr)
	pid := pr.Data.ID

	// vote yes as member
	code, _ = dissPost(t, app, "/api/v1/dissolution-proposals/"+pid+"/vote", map[string]string{"vote": "yes"}, memberTok)
	if code != 201 && code != 200 {
		t.Fatalf("vote %d", code)
	}
	// second vote updates not duplicates
	code, body = dissPost(t, app, "/api/v1/dissolution-proposals/"+pid+"/vote", map[string]string{"vote": "no"}, memberTok)
	if code != 200 {
		t.Fatalf("second vote should update 200 got %d %s", code, body)
	}
	var cnt int64
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ?", pid).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("expected 1 vote row, got %d", cnt)
	}
	// chair votes yes to make majority
	dissPost(t, app, "/api/v1/dissolution-proposals/"+pid+"/vote", map[string]string{"vote": "yes"}, chair)
	dissPost(t, app, "/api/v1/dissolution-proposals/"+pid+"/vote", map[string]string{"vote": "yes"}, secretary)

	// execution blocked before deadline
	code, _ = dissPost(t, app, "/api/v1/dissolution-proposals/"+pid+"/execute", nil, chair)
	if code == 200 {
		t.Fatalf("should block before deadline")
	}
	// fast-forward deadline
	database.DB.Model(&models.GroupDissolutionProposal{}).Where("id = ?", pid).Update("voting_deadline", time.Now().Add(-time.Hour))
	code, body = dissPost(t, app, "/api/v1/dissolution-proposals/"+pid+"/execute", nil, chair)
	if code != 200 {
		t.Fatalf("execute after deadline %d %s", code, body)
	}
	// (contributions already seeded; payout netting uses existing PAID rows within period)
	// payouts already created on execute; check amount_owed = contributed - grandTotal (maybe 0 if no owed)
	code, body = dissGet(t, app, "/api/v1/dissolution-proposals/"+pid+"/payouts", chair)
	if code != 200 {
		t.Fatalf("payouts %d", code)
	}
	var prs struct{ Data []models.DissolutionPayout `json:"data"` }
	json.Unmarshal(body, &prs)
	if len(prs.Data) == 0 {
		t.Fatalf("no payouts")
	}
	// group dissolved blocks new contributions
	treasurer := dissLogin(t, app, "fatuma@kikundi.tz")
	var anyMember models.Member
	database.DB.Where("deleted_at IS NULL").First(&anyMember)
	code, _ = dissPost(t, app, "/api/v1/contributions", map[string]interface{}{"member_id": anyMember.ID, "month": "2026-09", "amount": 1000, "paid_at": "2026-09-01"}, treasurer)
	if code != 403 {
		t.Fatalf("contributions should be blocked after dissolved, got %d", code)
	}
	code, _ = dissPost(t, app, "/api/v1/loans/apply", map[string]interface{}{"member_id": anyMember.ID, "amount": 1000, "purpose": "x", "due_date": "2026-12-31"}, chair)
	if code != 403 {
		t.Fatalf("loans should be blocked after dissolved, got %d", code)
	}
	code, _ = dissPost(t, app, "/api/v1/members", map[string]interface{}{"full_name": "Test", "phone": "0712345678", "joined_at": "2026-01-01"}, chair)
	if code != 403 {
		t.Fatalf("members should be blocked after dissolved, got %d", code)
	}
	// mark-paid by treasurer
	pid2 := prs.Data[0].ID
	code, _ = dissPatch(t, app, "/api/v1/dissolution-payouts/"+pid2+"/mark-paid", nil, treasurer)
	if code != 200 {
		t.Fatalf("mark-paid %d", code)
	}
	// cleanup: reset group to active for next tests
	database.DB.Exec("DELETE FROM dissolution_payouts")
	database.DB.Exec("DELETE FROM dissolution_votes")
	database.DB.Exec("DELETE FROM group_dissolution_proposals")
	database.DB.Model(&models.Group{}).Where("1=1").Update("status", "active")
}
