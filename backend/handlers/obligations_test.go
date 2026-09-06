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
	"github.com/shopspring/decimal"
)

func obligationsTestApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()

	app := fiber.New(fiber.Config{AppName: "Kikundi Obligations Test"})
	authHandler := NewAuthHandler()
	api := app.Group("/api/v1")
	api.Post("/auth/login", authHandler.Login)
	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	leadership3 := middleware.RequireRoles(models.RoleChair, models.RoleSecretary, models.RoleTreasurer)
	protected.Get("/members/:id/obligations/summary",
		middleware.RequireSelfOrLeadership(func(c *fiber.Ctx) (string, string) {
			return c.Params("id"), ""
		}), ObligationsMemberSummary)
	protected.Get("/groups/:id/obligations/summary", leadership3, ObligationsGroupSummary)

	offences := protected.Group("/groups/:id/fine-offence-types")
	offences.Post("/", middleware.RequireRoles(models.RoleChair), CreateOffenceType)
	offences.Post("/:typeId/approve", middleware.RequireRoles(models.RoleSecretary), ApproveOffenceType)

	fines := protected.Group("/fines")
	fines.Post("/:id/collect", middleware.RequireRoles(models.RoleTreasurer), CollectFine)
	fines.Post("/:id/waive-propose", middleware.RequireRoles(models.RoleChair), ProposeFineWaiver)
	fines.Post("/:id/waive-approve", middleware.RequireRoles(models.RoleSecretary), ApproveFineWaiver)

	mtg := protected.Group("/meetings/:id")
	mtg.Post("/trigger-fines", middleware.RequireRoles(models.RoleSecretary), TriggerMeetingFines)
	return app
}

func hReq(t *testing.T, app *fiber.App, method, path string, body interface{}, token string) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, _ := app.Test(req)
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, data
}

func cleanObligationTables(t *testing.T) {
	t.Helper()
	requireTestDB(t)
	for _, stmt := range []string{
		"DELETE FROM meeting_attendances",
		"DELETE FROM meetings",
		"DELETE FROM fines",
		"DELETE FROM contribution_cycles",
		"DELETE FROM fine_offence_types",
	} {
		database.DB.Exec(stmt)
	}
}

func seedObligationGroup(t *testing.T) models.Group {
	t.Helper()
	var g models.Group
	database.DB.First(&g)
	amt := decimal.NewFromInt(10000)
	due := "05"
	g.FixedContributionAmount = &amt
	g.ContributionDueDate = &due
	g.ContributionInterval = models.IntervalMonthly
	database.DB.Save(&g)
	return g
}

func makeOffence(t *testing.T, groupID, kind, name string, amount decimal.Decimal) models.FineOffenceType {
	t.Helper()
	ot := models.FineOffenceType{
		GroupID: groupID, Kind: kind, Name: name, FineType: models.FineTypeFixed,
		FineAmount: &amount, Status: models.OffenceActive, CreatedBy: "00000000-0000-0000-0000-000000000000",
	}
	if err := database.DB.Create(&ot).Error; err != nil {
		t.Fatalf("offence create: %v", err)
	}
	return ot
}

func TestFineCollectRBAC(t *testing.T) {
	app := obligationsTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	secretary := hLogin(t, app, "rashidi@kikundi.tz", "demo123")
	treasurer := hLogin(t, app, "fatuma@kikundi.tz", "demo123")

	ot := makeOffence(t, g.ID, models.OffenceLateContribution, "Kuchelewa", decimal.NewFromInt(2000))
	var member models.Member
	database.DB.Where("deleted_at IS NULL").First(&member)
	f, err := services.CreateCycleFine(g.ID, member.ID, ot.ID, "2026-05", time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), "test")
	if err != nil {
		t.Fatalf("cycle fine: %v", err)
	}

	if code, _ := hReq(t, app, "POST", "/api/v1/fines/"+f.ID+"/collect", nil, chair); code != 403 {
		t.Errorf("chair collect = %d, want 403", code)
	}
	if code, _ := hReq(t, app, "POST", "/api/v1/fines/"+f.ID+"/collect", nil, secretary); code != 403 {
		t.Errorf("secretary collect = %d, want 403", code)
	}
	if code, d := hReq(t, app, "POST", "/api/v1/fines/"+f.ID+"/collect", nil, treasurer); code != 200 {
		t.Errorf("treasurer collect = %d %s, want 200", code, d)
	}
	var after models.Fine
	database.DB.First(&after, "id = ?", f.ID)
	if after.Status != models.FinePaid || after.CollectedBy == nil {
		t.Errorf("fine not marked paid: %+v", after)
	}
}

func TestOffenceApproveRBAC(t *testing.T) {
	app := obligationsTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	secretary := hLogin(t, app, "rashidi@kikundi.tz", "demo123")
	treasurer := hLogin(t, app, "fatuma@kikundi.tz", "demo123")

	code, d := hPost(t, app, "/api/v1/groups/"+g.ID+"/fine-offence-types", map[string]interface{}{
		"kind": "late_contribution", "name": "Kuchelewa", "fine_type": "fixed",
		"fine_amount": 3000, "grace_period_days": 3,
	}, chair)
	if code != 201 {
		t.Fatalf("chair propose = %d %s, want 201", code, d)
	}
	var created struct {
		Data models.FineOffenceType `json:"data"`
	}
	json.Unmarshal(d, &created)
	if created.Data.Status != models.OffencePending {
		t.Errorf("new offence status = %s, want pending", created.Data.Status)
	}

	if code, _ := hReq(t, app, "POST", "/api/v1/groups/"+g.ID+"/fine-offence-types/"+created.Data.ID+"/approve", nil, treasurer); code != 403 {
		t.Errorf("treasurer approve = %d, want 403", code)
	}
	if code, _ := hReq(t, app, "POST", "/api/v1/groups/"+g.ID+"/fine-offence-types/"+created.Data.ID+"/approve", nil, secretary); code != 200 {
		t.Errorf("secretary approve = %d, want 200", code)
	}
}

func TestWaiverProposeApproveRBAC(t *testing.T) {
	app := obligationsTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	secretary := hLogin(t, app, "rashidi@kikundi.tz", "demo123")
	treasurer := hLogin(t, app, "fatuma@kikundi.tz", "demo123")

	ot := makeOffence(t, g.ID, models.OffenceLateContribution, "Kuchelewa", decimal.NewFromInt(2000))
	var member models.Member
	database.DB.Where("deleted_at IS NULL").First(&member)
	f, _ := services.CreateCycleFine(g.ID, member.ID, ot.ID, "2026-05", time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), "test")

	if code, _ := hReq(t, app, "POST", "/api/v1/fines/"+f.ID+"/waive-approve", nil, treasurer); code != 403 {
		t.Errorf("treasurer waive-approve = %d, want 403", code)
	}
	code, d := hPost(t, app, "/api/v1/fines/"+f.ID+"/waive-propose", map[string]string{"reason": "Mgongo"}, chair)
	if code != 200 {
		t.Fatalf("chair waive-propose = %d %s, want 200", code, d)
	}
	if code, _ := hReq(t, app, "POST", "/api/v1/fines/"+f.ID+"/waive-approve", nil, secretary); code != 200 {
		t.Errorf("secretary waive-approve = %d, want 200", code)
	}
	var after models.Fine
	database.DB.First(&after, "id = ?", f.ID)
	if after.Status != models.FineWaived || after.WaiverStatus != models.WaiverApproved {
		t.Errorf("waiver not applied: %+v", after)
	}
}

func TestMeetingTriggerRBAC(t *testing.T) {
	app := obligationsTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	secretary := hLogin(t, app, "rashidi@kikundi.tz", "demo123")

	if code, _ := hReq(t, app, "POST", "/api/v1/meetings/some-id/trigger-fines", nil, chair); code != 403 {
		t.Errorf("chair trigger = %d, want 403", code)
	}

	// Secretary success path: absent member + active absence offence → fine.
	ot := makeOffence(t, g.ID, models.OffenceMeetingAbsence, "Kutohudhuria", decimal.NewFromInt(1000))
	var member models.Member
	database.DB.Where("deleted_at IS NULL").First(&member)
	var creator models.User
	database.DB.Where("email = ?", "rashidi@kikundi.tz").First(&creator)
	mtg := models.Meeting{GroupID: g.ID, Title: "Mkutano Mkuu",
		MeetingDate: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), CreatedBy: creator.ID}
	if err := database.DB.Create(&mtg).Error; err != nil {
		t.Fatalf("meeting create: %v", err)
	}
	database.DB.Create(&models.MeetingAttendance{
		MeetingID: mtg.ID, MemberID: member.ID, Status: models.AttendanceAbsent,
	})
	code, d := hReq(t, app, "POST", "/api/v1/meetings/"+mtg.ID+"/trigger-fines", nil, secretary)
	if code != 200 {
		t.Fatalf("secretary trigger = %d %s, want 200", code, d)
	}
	var nf int64
	database.DB.Model(&models.Fine{}).Where("group_id = ? AND member_id = ? AND offence_type_id = ?",
		g.ID, member.ID, ot.ID).Count(&nf)
	if nf != 1 {
		t.Errorf("meeting fines = %d, want 1", nf)
	}
	// Re-trigger creates nothing new.
	if code, _ := hReq(t, app, "POST", "/api/v1/meetings/"+mtg.ID+"/trigger-fines", nil, secretary); code != 200 {
		t.Errorf("re-trigger = %d, want 200", code)
	}
	database.DB.Model(&models.Fine{}).Where("group_id = ? AND member_id = ? AND offence_type_id = ?",
		g.ID, member.ID, ot.ID).Count(&nf)
	if nf != 1 {
		t.Errorf("re-trigger duplicated: count = %d", nf)
	}
}

func TestObligationsSummaryMath(t *testing.T) {
	app := obligationsTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t)
	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")

	var member models.Member
	database.DB.Where("deleted_at IS NULL").Order("member_no").First(&member)

	// Two unpaid past cycles + current: seed cycle rows directly (fixed 10000).
	now := time.Now()
	labels := []string{"2026-01", "2026-02", "2026-03"}
	for i, lb := range labels {
		due := time.Date(2026, time.Month(i+1), 5, 0, 0, 0, 0, time.UTC)
		database.DB.Create(&models.ContributionCycle{
			GroupID: g.ID, MemberID: member.ID, CycleLabel: lb, DueDate: due,
			ExpectedAmount: decimal.NewFromInt(10000), Status: models.CycleUnpaid,
		})
	}
	_ = now

	ot := makeOffence(t, g.ID, models.OffenceLateContribution, "Kuchelewa", decimal.NewFromInt(5000))
	f, err := services.CreateCycleFine(g.ID, member.ID, ot.ID, "2026-02",
		time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), "test")
	if err != nil {
		t.Fatalf("fine: %v", err)
	}

	code, d := hGet(t, app, "/api/v1/members/"+member.ID+"/obligations/summary", chair)
	if code != 200 {
		t.Fatalf("summary = %d %s", code, d)
	}
	var res struct {
		Data services.MemberObligations `json:"data"`
	}
	json.Unmarshal(d, &res)
	// Fine snapshot: change offence amount — summary must not move.
	newAmt := decimal.NewFromInt(99999)
	database.DB.Model(&models.FineOffenceType{}).Where("id = ?", ot.ID).Update("fine_amount", newAmt)
	code, d = hGet(t, app, "/api/v1/members/"+member.ID+"/obligations/summary", chair)
	json.Unmarshal(d, &res)
	if res.Data.TotalFinesUnpaid.String() != "5000" {
		t.Errorf("fine snapshot moved: total_fines = %s, want 5000", res.Data.TotalFinesUnpaid)
	}
	sum := res.Data.TotalArrears.Add(res.Data.CurrentCycleDue).Add(res.Data.TotalFinesUnpaid)
	if !sum.Equal(res.Data.GrandTotal) {
		t.Errorf("grand_total %s != parts sum %s", res.Data.GrandTotal, sum)
	}
	if len(res.Data.ItemizedFines) != 1 || res.Data.ItemizedFines[0].ID != f.ID {
		t.Errorf("itemized fines wrong: %+v", res.Data.ItemizedFines)
	}
	t.Logf("arrears=%s current=%s fines=%s grand=%s",
		res.Data.TotalArrears, res.Data.CurrentCycleDue, res.Data.TotalFinesUnpaid, res.Data.GrandTotal)
}

func TestFineIdempotencyAndDeactivation(t *testing.T) {
	_ = obligationsTestApp()
	cleanAndSeed(t)
	cleanObligationTables(t)
	g := seedObligationGroup(t)

	ot := makeOffence(t, g.ID, models.OffenceLateContribution, "Kuchelewa", decimal.NewFromInt(2000))
	var member models.Member
	database.DB.Where("deleted_at IS NULL").First(&member)
	due := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)

	f1, err := services.CreateCycleFine(g.ID, member.ID, ot.ID, "2026-05", due, "test")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	f2, err := services.CreateCycleFine(g.ID, member.ID, ot.ID, "2026-05", due, "test")
	if err != nil || f1.ID != f2.ID {
		t.Fatalf("cycle dedup failed: %v %v", f1.ID, f2.ID)
	}
	var n int64
	database.DB.Model(&models.Fine{}).Where("group_id = ? AND member_id = ? AND offence_type_id = ? AND contribution_cycle_label = ?",
		g.ID, member.ID, ot.ID, "2026-05").Count(&n)
	if n != 1 {
		t.Errorf("cycle fines = %d, want 1", n)
	}

	e1, err := services.CreateEventFine(g.ID, member.ID, ot.ID, time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), "test", "")
	if err != nil {
		t.Fatalf("event first: %v", err)
	}
	e2, err := services.CreateEventFine(g.ID, member.ID, ot.ID, time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), "test", "")
	if err != nil || e1.ID != e2.ID {
		t.Fatalf("event dedup failed")
	}

	// Deactivation stops new fines.
	database.DB.Model(&models.FineOffenceType{}).Where("id = ?", ot.ID).Update("status", models.OffenceInactive)
	if _, err := services.CreateCycleFine(g.ID, member.ID, ot.ID, "2026-06", time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), "test"); err == nil {
		t.Errorf("inactive offence should refuse new fines")
	}
	var total int64
	database.DB.Model(&models.Fine{}).Where("group_id = ? AND member_id = ?", g.ID, member.ID).Count(&total)
	if total != 2 {
		t.Errorf("old fines affected by deactivation: count = %d, want 2", total)
	}
}
