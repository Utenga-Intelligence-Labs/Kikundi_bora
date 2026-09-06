package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

func gatingTestApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	services.InitEmail()

	app := fiber.New(fiber.Config{AppName: "Kikundi Approval Gating Test"})
	api := app.Group("/api/v1")

	authHandler := NewAuthHandler()
	memberHandler := NewMemberHandler()
	memberContribHandler := NewMemberContributionHandler()
	welfareHandler := NewWelfareHandler()
	leadershipHandler := NewLeadershipHandler()

	api.Post("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	members := protected.Group("/members")
	members.Post("/", middleware.RequireRoles(models.RoleChair, models.RoleSecretary, models.RoleTreasurer), memberHandler.Create)

	michango := protected.Group("/michango")
	michango.Post("/", memberContribHandler.Submit)
	michango.Get("/members-summary", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer, models.LeadershipSecretary), memberContribHandler.MembersSummary)

	welfare := protected.Group("/welfare")
	welfare.Post("/events/:id/contributions/:memberId/pay", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.RecordPayment)

	uongozi := protected.Group("/uongozi")
	uongozi.Get("/dashboard", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer, models.LeadershipSecretary), leadershipHandler.Dashboard)
	uongozi.Get("/quick-stats", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer, models.LeadershipSecretary), leadershipHandler.QuickStats)

	return app
}

func gatingLogin(t *testing.T, app *fiber.App, loginID, password string) (int, []byte) {
	t.Helper()
	body := map[string]string{"email": loginID, "password": password}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, data
}

// makePendingFixture creates an ACTIVE user + linked member with the given
// approval status, returning both. Password is "testpass123".
func makePendingFixture(t *testing.T, phone, memberNo, approval string) (models.User, models.Member) {
	t.Helper()
	var chair models.User
	if err := database.DB.Where("role = ?", models.RoleChair).First(&chair).Error; err != nil {
		t.Fatalf("chair: %v", err)
	}
	hashed, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	u := models.User{
		Name: "Mtihani " + memberNo, Phone: phone, Password: string(hashed),
		Role: models.RoleMember, Status: models.UserStatusActive, IsActive: true,
		CreatedBy: &chair.ID,
	}
	if err := database.DB.Create(&u).Error; err != nil {
		t.Fatalf("user create: %v", err)
	}
	m := models.Member{
		MemberNo: memberNo, UserID: &u.ID, FullName: "Mtihani " + memberNo,
		Phone: phone, JoinedAt: time.Now(), IsActive: false,
		RegisteredBy: chair.ID, ApprovalStatus: approval,
	}
	if err := database.DB.Create(&m).Error; err != nil {
		t.Fatalf("member create: %v", err)
	}
	database.DB.Model(&m).Update("is_active", false)
	return u, m
}

func TestPendingMemberLoginRejected(t *testing.T) {
	app := gatingTestApp()
	cleanAndSeed(t)

	_, _ = makePendingFixture(t, "0700000101", "KKK-T01", models.MemberApprovalPending)

	// Correct credentials, pending member → 403 with clear message, no token
	code, body := gatingLogin(t, app, "0700000101", "testpass123")
	if code != 403 {
		t.Fatalf("pending login: want 403, got %d (%s)", code, body)
	}
	if !strings.Contains(string(body), "subiri idhini") {
		t.Errorf("pending login message should mention katibu approval, got: %s", body)
	}
	if strings.Contains(string(body), "\"token\"") {
		t.Errorf("pending login must not issue a token: %s", body)
	}

	// Wrong password still 401 (distinguishable from pending-approval 403)
	code, _ = gatingLogin(t, app, "0700000101", "wrongpass")
	if code != 401 {
		t.Errorf("wrong password: want 401, got %d", code)
	}

	// Rejected member → 403 with rejection message
	database.DB.Exec("UPDATE members SET approval_status = 'rejected' WHERE member_no = 'KKK-T01'")
	code, body = gatingLogin(t, app, "0700000101", "testpass123")
	if code != 403 {
		t.Fatalf("rejected login: want 403, got %d (%s)", code, body)
	}
	if !strings.Contains(string(body), "haikuidhinishwa") {
		t.Errorf("rejected login message wrong: %s", body)
	}
}

func TestSocialFundPendingGates(t *testing.T) {
	app := gatingTestApp()
	cleanAndSeed(t)

	// Approved fixture user logs in, then gets flipped to pending — the
	// session stays valid so the handler-level gates are what block.
	_, pm := makePendingFixture(t, "0700000102", "KKK-T02", models.MemberApprovalApproved)
	code, body := gatingLogin(t, app, "0700000102", "testpass123")
	if code != 200 {
		t.Fatalf("approved login: %d %s", code, body)
	}
	var lr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(body, &lr)
	database.DB.Model(&models.Member{}).Where("id = ?", pm.ID).Update("approval_status", models.MemberApprovalPending)

	// Direct MFUKO self-contribution as pending → 403 (not UI-hidden only)
	mfukoBody := map[string]interface{}{
		"contribution_type": "MFUKO_WA_KIJAMII", "period_label": "2026-01",
		"amount": 5000, "proof_message": "halisi", "welfare_event_id": "00000000-0000-0000-0000-000000000000",
	}
	code, body = hReq(t, app, "POST", "/api/v1/michango", mfukoBody, lr.Token)
	if code != 403 {
		t.Errorf("pending MFUKO submit: want 403, got %d (%s)", code, body)
	}

	// Treasurer attributing a social-fund payment to a pending member → 403
	treasCode, treasBody := gatingLogin(t, app, "fatuma@kikundi.tz", "demo123")
	if treasCode != 200 {
		t.Fatalf("treasurer login: %d %s", treasCode, treasBody)
	}
	var tr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(treasBody, &tr)
	payBody := map[string]interface{}{"amount": 5000}
	code, body = hReq(t, app, "POST", "/api/v1/welfare/events/00000000-0000-0000-0000-000000000000/contributions/"+pm.ID+"/pay", payBody, tr.Token)
	if code != 403 {
		t.Errorf("welfare pay to pending: want 403, got %d (%s)", code, body)
	}

	// Leadership member-selection list excludes the pending member
	chairCode, chairBody := gatingLogin(t, app, "juma@kikundi.tz", "demo123")
	if chairCode != 200 {
		t.Fatalf("chair login: %d %s", chairCode, chairBody)
	}
	var cr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(chairBody, &cr)
	code, body = hReq(t, app, "GET", "/api/v1/michango/members-summary", nil, cr.Token)
	if code != 200 {
		t.Fatalf("members-summary: %d %s", code, body)
	}
	if strings.Contains(string(body), pm.ID) {
		t.Errorf("members-summary must not list pending member %s", pm.ID)
	}
}

func TestCountsExcludePending(t *testing.T) {
	app := gatingTestApp()
	cleanAndSeed(t)

	chairCode, chairBody := gatingLogin(t, app, "juma@kikundi.tz", "demo123")
	if chairCode != 200 {
		t.Fatalf("chair login: %d %s", chairCode, chairBody)
	}
	var cr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(chairBody, &cr)

	getTotal := func() int64 {
		code, body := hReq(t, app, "GET", "/api/v1/uongozi/quick-stats", nil, cr.Token)
		if code != 200 {
			t.Fatalf("quick-stats: %d %s", code, body)
		}
		var r struct {
			TotalMembers int64 `json:"total_members"`
		}
		json.Unmarshal(body, &r)
		return r.TotalMembers
	}
	before := getTotal()

	_, _ = makePendingFixture(t, "0700000103", "KKK-T03", models.MemberApprovalPending)

	after := getTotal()
	if after != before {
		t.Errorf("quick-stats total_members inflated by pending: before=%d after=%d", before, after)
	}

	code, body := hReq(t, app, "GET", "/api/v1/uongozi/dashboard", nil, cr.Token)
	if code != 200 {
		t.Fatalf("uongozi dashboard: %d %s", code, body)
	}
	var dash struct {
		TotalMembers int64 `json:"total_members"`
	}
	json.Unmarshal(body, &dash)
	if dash.TotalMembers != before {
		t.Errorf("uongozi dashboard total_members=%d, want %d (pending excluded)", dash.TotalMembers, before)
	}
}

func TestBackdateArrearsMainOnly(t *testing.T) {
	app := gatingTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t) // monthly, 10000, due 05

	chairCode, chairBody := gatingLogin(t, app, "juma@kikundi.tz", "demo123")
	if chairCode != 200 {
		t.Fatalf("chair login: %d %s", chairCode, chairBody)
	}
	var cr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(chairBody, &cr)

	// Non-chair (secretary) may not backdate
	secCode, secBody := gatingLogin(t, app, "rashidi@kikundi.tz", "demo123")
	if secCode != 200 {
		t.Fatalf("secretary login: %d %s", secCode, secBody)
	}
	var sr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(secBody, &sr)
	from := time.Now().AddDate(0, -3, 0).Format("2006-01-02")
	secReq := map[string]interface{}{
		"full_name": "Jaribio Katibu", "phone": "0700000201", "joined_at": time.Now().Format("2006-01-02"),
		"backdate_arrears": true, "backdate_from_cycle": from,
	}
	code, _ := hReq(t, app, "POST", "/api/v1/members", secReq, sr.Token)
	if code != 403 {
		t.Errorf("secretary backdate: want 403, got %d", code)
	}

	// Chair backdates 3 months of main cycles
	req := map[string]interface{}{
		"full_name": "Jaribio Mzee", "phone": "0700000202", "joined_at": time.Now().Format("2006-01-02"),
		"backdate_arrears": true, "backdate_from_cycle": from,
	}
	code, body := hReq(t, app, "POST", "/api/v1/members", req, cr.Token)
	if code != 201 {
		t.Fatalf("chair create+backdate: %d %s", code, body)
	}
	var created struct {
		Data            models.Member `json:"data"`
		BackdatedCycles []string      `json:"backdated_cycles"`
	}
	json.Unmarshal(body, &created)
	if len(created.BackdatedCycles) == 0 {
		t.Fatalf("expected backdated cycles, got none (%s)", body)
	}

	// Main-cycle rows exist…
	var cycles int64
	database.DB.Model(&models.ContributionCycle{}).Where("member_id = ?", created.Data.ID).Count(&cycles)
	if cycles == 0 {
		t.Errorf("expected contribution_cycles rows for backdated member")
	}
	// …amounts match the group fixed amount
	var sumRow struct {
		Total string
	}
	database.DB.Raw(`SELECT COALESCE(SUM(expected_amount),0)::text AS total FROM contribution_cycles WHERE member_id = ?`, created.Data.ID).Scan(&sumRow)
	sum, _ := decimal.NewFromString(sumRow.Total)
	want := decimal.NewFromInt(10000).Mul(decimal.NewFromInt(cycles))
	if !sum.Equal(want) {
		t.Errorf("backdated expected total=%s, want %s (fixed 10000 x %d cycles)", sum, want, cycles)
	}

	// …and ZERO social-fund records were created regardless of backdate
	var welfareRows int64
	database.DB.Model(&models.WelfareContribution{}).Where("member_id = ?", created.Data.ID).Count(&welfareRows)
	if welfareRows != 0 {
		t.Errorf("backdate must never create welfare rows, got %d", welfareRows)
	}

	// Audit trail records who/when/which cycles
	var audits int64
	database.DB.Model(&models.AuditLog{}).
		Where("table_name = ? AND record_id = ?", "contribution_cycles", created.Data.ID).
		Count(&audits)
	if audits == 0 {
		t.Errorf("backdate action missing from audit trail")
	}

	// No backdate → fresh obligations only (no historical rows at creation)
	req2 := map[string]interface{}{
		"full_name": "Jaribio Freshi", "phone": "0700000203", "joined_at": time.Now().Format("2006-01-02"),
	}
	code, body = hReq(t, app, "POST", "/api/v1/members", req2, cr.Token)
	if code != 201 {
		t.Fatalf("plain create: %d %s", code, body)
	}
	var created2 struct {
		Data models.Member `json:"data"`
	}
	json.Unmarshal(body, &created2)
	var fresh int64
	database.DB.Model(&models.ContributionCycle{}).Where("member_id = ?", created2.Data.ID).Count(&fresh)
	if fresh != 0 {
		t.Errorf("member without backdate should start with zero cycles, got %d", fresh)
	}

	_ = g
}
