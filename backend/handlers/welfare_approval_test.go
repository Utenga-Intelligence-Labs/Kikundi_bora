package handlers

import (
	"encoding/json"
	"testing"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

// Approval-hub routes for the consolidated /michango page + welfare
// contribution approve/reject. Registered on the shared fullApp harness.
func registerApprovalHubRoutes(app *fiber.App) {
	welfareHandler := NewWelfareHandler()
	memberContribHandler := NewMemberContributionHandler()
	g := app.Group("/api/v1", middleware.AuthRequired)
	g.Post("/welfare/contributions/:id/approve",
		middleware.RequireRoles(models.RoleTreasurer), welfareHandler.ApproveContribution)
	g.Post("/welfare/contributions/:id/reject",
		middleware.RequireRoles(models.RoleTreasurer), welfareHandler.RejectContribution)
	g.Get("/welfare/contributions",
		middleware.RequireRoles(models.RoleTreasurer, models.RoleChair, models.RoleSecretary),
		welfareHandler.ListContributions)
	g.Post("/michango/:id/confirm",
		middleware.RequireRoles(models.RoleChair, models.RoleTreasurer), memberContribHandler.Confirm)
	g.Post("/michango/:id/reject",
		middleware.RequireRoles(models.RoleChair, models.RoleTreasurer), memberContribHandler.Reject)
}

func seedApprovalHub(t *testing.T, app *fiber.App) (treasurerTok, chairTok, secretaryTok, memberID, eventID, contribID string) {
	t.Helper()
	cleanupTestData()
	reseedIfEmpty()

	treasurerTok = doLogin(t, app, "fatuma@kikundi.tz", "demo123")
	chairTok = doLogin(t, app, "juma@kikundi.tz", "demo123")
	secretaryTok = doLogin(t, app, "rashidi@kikundi.tz", "demo123")

	var member models.Member
	if err := database.DB.Where("deleted_at IS NULL").First(&member).Error; err != nil {
		t.Fatalf("no member: %v", err)
	}
	memberID = member.ID

	var treasurer models.User
	database.DB.Where("role = ?", models.RoleTreasurer).First(&treasurer)
	event := models.WelfareEvent{
		MemberID:        member.ID,
		EventType:       models.WelfareMedical,
		Description:     "Test event",
		AmountRequested: decimal.NewFromInt(90000),
		AmountApproved:  decimalPtr(decimal.NewFromInt(90000)),
		FundingSource:   models.FundMemberContribution,
		Status:          models.WelfareApproved,
		CreatedBy:       treasurer.ID,
	}
	if err := database.DB.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}
	eventID = event.ID

	// Three member obligations, all PENDING
	var members []models.Member
	database.DB.Where("deleted_at IS NULL").Limit(3).Find(&members)
	if len(members) == 0 {
		t.Fatalf("no members")
	}
	for _, m := range members {
		database.DB.Create(&models.WelfareContribution{
			EventID: event.ID, MemberID: m.ID,
			Amount: decimal.NewFromInt(30000), Status: models.WelfareContribPending,
		})
	}
	var first models.WelfareContribution
	database.DB.Where("event_id = ?", event.ID).First(&first)
	return treasurerTok, chairTok, secretaryTok, memberID, eventID, first.ID
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal { return &d }

func welfarePaidTotal(t *testing.T, eventID string) decimal.Decimal {
	t.Helper()
	var total decimal.Decimal
	database.DB.Model(&models.WelfareContribution{}).
		Where("event_id = ? AND status = ?", eventID, models.WelfareContribPaid).
		Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total
}

func TestWelfareApproveIncreasesBalance(t *testing.T) {
	app := fullApp()
	registerApprovalHubRoutes(app)
	treasurerTok, _, _, _, eventID, contribID := seedApprovalHub(t, app)

	if got := welfarePaidTotal(t, eventID); !got.IsZero() {
		t.Fatalf("paid total before approve = %s, want 0", got)
	}

	code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/approve", nil, treasurerTok)
	if code != 200 {
		t.Fatalf("approve = %d, want 200", code)
	}
	if got := welfarePaidTotal(t, eventID); !got.Equal(decimal.NewFromInt(30000)) {
		t.Fatalf("paid total after approve = %s, want 30000", got)
	}

	// Double-approve must fail, not double-count.
	code, _ = doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/approve", nil, treasurerTok)
	if code == 200 {
		t.Fatalf("double approve should fail")
	}
	if got := welfarePaidTotal(t, eventID); !got.Equal(decimal.NewFromInt(30000)) {
		t.Fatalf("paid total after double approve = %s, want 30000", got)
	}
}

func TestWelfarePendingNotCountedAndReject(t *testing.T) {
	app := fullApp()
	registerApprovalHubRoutes(app)
	treasurerTok, _, _, _, eventID, contribID := seedApprovalHub(t, app)

	// Reject requires a reason.
	code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/reject",
		map[string]string{}, treasurerTok)
	if code != 400 {
		t.Fatalf("reject without reason = %d, want 400", code)
	}
	code, _ = doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/reject",
		map[string]string{"reason": "Uchunguzi"}, treasurerTok)
	if code != 200 {
		t.Fatalf("reject = %d, want 200", code)
	}
	var c models.WelfareContribution
	database.DB.First(&c, "id = ?", contribID)
	if c.Status != models.WelfareContribWaived {
		t.Fatalf("status = %s, want WAIVED", c.Status)
	}
	if got := welfarePaidTotal(t, eventID); !got.IsZero() {
		t.Fatalf("rejected must not count: paid total = %s", got)
	}
}

func TestWelfareApproveRejectRBAC(t *testing.T) {
	app := fullApp()
	registerApprovalHubRoutes(app)
	treasurerTok, chairTok, secretaryTok, _, _, contribID := seedApprovalHub(t, app)

	for _, tok := range []string{chairTok, secretaryTok} {
		if code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/approve", nil, tok); code != 403 {
			t.Errorf("non-treasurer approve = %d, want 403", code)
		}
		if code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/reject", map[string]string{"reason": "x"}, tok); code != 403 {
			t.Errorf("non-treasurer reject = %d, want 403", code)
		}
	}
	// Treasurer succeeds (also proves the treasurer path post-RBAC checks).
	if code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/approve", nil, treasurerTok); code != 200 {
		t.Errorf("treasurer approve = %d, want 200", code)
	}
}

func TestWelfarePendingListFilter(t *testing.T) {
	app := fullApp()
	registerApprovalHubRoutes(app)
	treasurerTok, _, _, _, eventID, contribID := seedApprovalHub(t, app)

	code, body := doRequest(t, app, "GET", "/api/v1/welfare/contributions?status=PENDING", nil, treasurerTok)
	if code != 200 {
		t.Fatalf("pending list = %d", code)
	}
	var list struct {
		Data  []models.WelfareContribution `json:"data"`
		Total int                          `json:"total"`
	}
	json.Unmarshal(body, &list)
	if list.Total != 3 {
		t.Fatalf("pending total = %d, want 3", list.Total)
	}

	// Approve one → pending drops to 2.
	doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/approve", nil, treasurerTok)
	_, body = doRequest(t, app, "GET", "/api/v1/welfare/contributions?status=PENDING", nil, treasurerTok)
	json.Unmarshal(body, &list)
	if list.Total != 2 {
		t.Fatalf("pending total after approve = %d, want 2", list.Total)
	}
	_ = eventID
}
