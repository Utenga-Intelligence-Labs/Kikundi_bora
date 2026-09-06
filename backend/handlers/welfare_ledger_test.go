package handlers

import (
	"context"
	"testing"
	"time"

	"kikundibora/database"
	"kikundibora/ledger"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

// wireTestLedger connects the auto-post ledger to the test database so
// welfare flows can assert real ledger entries (not just the WARN path).
func wireTestLedger(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.ConnectPgx(ctx)
	if err != nil {
		t.Fatalf("pgx connect: %v", err)
	}
	if err := ledger.Migrate(ctx, pool); err != nil {
		t.Fatalf("ledger migrate: %v", err)
	}
	gid, err := ledger.CreateGroup(ctx, pool, "kikundi-test", ledger.CurrencyTZS)
	if err != nil {
		t.Fatalf("ledger group: %v", err)
	}
	lg, err := ledger.New(pool)
	if err != nil {
		t.Fatalf("ledger new: %v", err)
	}
	services.SetAutoLedger(lg, gid)
}

func welfareLedgerApp() *fiber.App {
	app := fullApp()
	welfareHandler := NewWelfareHandler()
	g := app.Group("/api/v1/welfare", middleware.AuthRequired)
	g.Post("/contributions/:id/approve",
		middleware.RequireRoles(models.RoleTreasurer), welfareHandler.ApproveContribution)
	g.Post("/events/:id/disburse",
		middleware.RequireRoles(models.RoleTreasurer), welfareHandler.DisburseEvent)
	g.Post("/events/:id/confirm-receipt", welfareHandler.ConfirmReceipt)
	return app
}

func seedWelfarePaidFlow(t *testing.T, app *fiber.App) (treasurerTok, chairTok, eventID string, contribIDs []string) {
	t.Helper()
	cleanupTestData()
	reseedIfEmpty()
	wireTestLedger(t)

	treasurerTok = doLogin(t, app, "fatuma@kikundi.tz", "demo123")
	chairTok = doLogin(t, app, "juma@kikundi.tz", "demo123")

	var members []models.Member
	database.DB.Where("deleted_at IS NULL").Order("member_no").Limit(2).Find(&members)
	if len(members) < 2 {
		t.Fatalf("need 2 members")
	}

	var treasurer models.User
	database.DB.Where("role = ?", models.RoleTreasurer).First(&treasurer)
	event := models.WelfareEvent{
		MemberID:        members[0].ID,
		EventType:       models.WelfareMedical,
		Description:     "Ledger test event",
		AmountRequested: decimal.NewFromInt(60000),
		AmountApproved:  decimalPtrWelfare(decimal.NewFromInt(60000)),
		FundingSource:   models.FundMemberContribution,
		Status:          models.WelfareApproved,
		CreatedBy:       treasurer.ID,
	}
	if err := database.DB.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}
	eventID = event.ID
	for _, m := range members {
		database.DB.Create(&models.WelfareContribution{
			EventID: event.ID, MemberID: m.ID,
			Amount: decimal.NewFromInt(30000), Status: models.WelfareContribPending,
		})
	}
	var rows []models.WelfareContribution
	database.DB.Where("event_id = ?", event.ID).Order("created_at").Find(&rows)
	for _, r := range rows {
		contribIDs = append(contribIDs, r.ID)
	}
	return treasurerTok, chairTok, eventID, contribIDs
}

func decimalPtrWelfare(d decimal.Decimal) *decimal.Decimal { return &d }

func welfareFundBalance(t *testing.T) decimal.Decimal {
	t.Helper()
	// mfuko_wa_kijamii is a LIABILITY: credit-positive.
	var bal decimal.Decimal
	database.DB.Raw(`SELECT COALESCE(SUM(CASE WHEN direction='credit' THEN amount_minor ELSE -amount_minor END),0)
		FROM ledger_statement_lines WHERE account_name = ?`, "mfuko_wa_kijamii").Scan(&bal)
	return bal
}

func TestWelfareApprovePostsLedger(t *testing.T) {
	app := welfareLedgerApp()
	treasurerTok, _, _, contribIDs := seedWelfarePaidFlow(t, app)
	contribID := contribIDs[0]

	before := welfareFundBalance(t)
	code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+contribID+"/approve", nil, treasurerTok)
	if code != 200 {
		t.Fatalf("approve = %d, want 200", code)
	}
	after := welfareFundBalance(t)
	// 30000 TZS = 3000000 minor units into the fund (liability grows on credit).
	if after.Sub(before).Cmp(decimal.NewFromInt(3000000)) != 0 {
		t.Fatalf("fund delta = %s, want 3000000 minor", after.Sub(before))
	}
}

func TestWelfareDisbursePostsLedgerAndReceipt(t *testing.T) {
	app := welfareLedgerApp()
	treasurerTok, chairTok, eventID, contribIDs := seedWelfarePaidFlow(t, app)

	// Approve both (funds the welfare account; event completes on the last).
	for _, cid := range contribIDs {
		if code, _ := doRequest(t, app, "POST", "/api/v1/welfare/contributions/"+cid+"/approve", nil, treasurerTok); code != 200 {
			t.Fatalf("approve = %d", code)
		}
	}

	before := welfareFundBalance(t)
	code, _ := doRequest(t, app, "POST", "/api/v1/welfare/events/"+eventID+"/disburse", nil, treasurerTok)
	if code != 200 {
		t.Fatalf("disburse = %d, want 200", code)
	}
	after := welfareFundBalance(t)
	if before.Sub(after).Cmp(decimal.NewFromInt(6000000)) != 0 {
		t.Fatalf("disburse delta = %s, want 6000000 minor", before.Sub(after))
	}

	var ev models.WelfareEvent
	database.DB.First(&ev, "id = ?", eventID)
	if ev.Status != models.WelfareCompleted {
		t.Fatalf("status = %s, want COMPLETED", ev.Status)
	}
	if ev.DisbursedBy == nil || ev.DisbursedAt == nil {
		t.Fatalf("disburse attribution missing")
	}

	// Unauthenticated confirm must fail.
	code, _ = doRequest(t, app, "POST", "/api/v1/welfare/events/"+eventID+"/confirm-receipt", nil, "")
	if code != 401 {
		t.Fatalf("unauthenticated confirm = %d, want 401", code)
	}

	// Leadership confirm succeeds (beneficiary path covered by role branch).
	code, _ = doRequest(t, app, "POST", "/api/v1/welfare/events/"+eventID+"/confirm-receipt", nil, chairTok)
	if code != 200 {
		t.Fatalf("confirm receipt = %d, want 200", code)
	}
	database.DB.First(&ev, "id = ?", eventID)
	if ev.ReceivedAt == nil || ev.ReceivedBy == nil {
		t.Fatalf("receipt not recorded")
	}

	// Double confirm must fail.
	if code, _ := doRequest(t, app, "POST", "/api/v1/welfare/events/"+eventID+"/confirm-receipt", nil, chairTok); code != 409 {
		t.Fatalf("double confirm = %d, want 409", code)
	}
}
