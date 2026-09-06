package handlers

import (
	"encoding/json"
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

func offsetTestApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	services.InitEmail()

	app := fiber.New(fiber.Config{AppName: "Kikundi Offset Test"})
	api := app.Group("/api/v1")

	authHandler := NewAuthHandler()
	dashHandler := NewDashboardHandler()
	offsetHandler := NewLoanOffsetHandler()

	api.Post("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	leadership3 := middleware.RequireRoles(models.RoleChair, models.RoleSecretary, models.RoleTreasurer)
	loans := protected.Group("/loans")
	loans.Get("/:id/offset-preview", leadership3, offsetHandler.Preview)
	loans.Post("/:id/offset-propose", middleware.RequireRoles(models.RoleChair), offsetHandler.Propose)
	offsets := protected.Group("/loan-offsets")
	offsets.Get("/", leadership3, offsetHandler.List)
	offsets.Post("/:id/approve", middleware.RequireRoles(models.RoleSecretary), offsetHandler.Approve)
	offsets.Post("/:id/reject", middleware.RequireRoles(models.RoleSecretary), offsetHandler.Reject)
	offsets.Post("/:id/execute", middleware.RequireRoles(models.RoleTreasurer), offsetHandler.Execute)

	protected.Get("/members/:id/dashboard-summary", dashHandler.MemberSummary)

	return app
}

// offsetFixture creates savings (PAID contributions) + one overdue
// OUTSTANDING loan for asha's linked member. Returns member + loan.
func offsetFixture(t *testing.T, savings, outstanding int64) (models.Member, models.Loan) {
	t.Helper()
	var asha models.User
	if err := database.DB.Where("email = ?", "asha@kikundi.tz").First(&asha).Error; err != nil {
		t.Fatalf("asha: %v", err)
	}
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", asha.ID).First(&member).Error; err != nil {
		t.Fatalf("asha member: %v", err)
	}
	now := time.Now()
	monthFirst := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if savings > 0 {
		c := models.Contribution{
			MemberID: member.ID, RecordedBy: asha.ID,
			Amount: decimal.NewFromInt(savings), Month: monthFirst,
			PaidAt: now, PaymentMethod: "CASH", Status: "PAID",
		}
		if err := database.DB.Create(&c).Error; err != nil {
			t.Fatalf("savings: %v", err)
		}
	}
	approved := decimal.NewFromInt(outstanding)
	remaining := decimal.NewFromInt(outstanding)
	past := now.AddDate(0, 0, -30)
	disbursed := now.AddDate(0, 0, -60)
	loan := models.Loan{
		MemberID: member.ID, Amount: approved, ApprovedAmount: &approved,
		BalanceRemaining: &remaining, DueDate: past,
		Status: models.LoanOutstanding, DisbursedAt: &disbursed, DisbursedBy: &asha.ID,
	}
	if err := database.DB.Create(&loan).Error; err != nil {
		t.Fatalf("loan: %v", err)
	}
	return member, loan
}

func offsetTokens(t *testing.T, app *fiber.App) (chair, secretary, treasurer, member string) {
	t.Helper()
	tok := func(email string) string {
		code, body := gatingLogin(t, app, email, "demo123")
		if code != 200 {
			t.Fatalf("login %s: %d %s", email, code, body)
		}
		var r struct {
			Token string `json:"token"`
		}
		json.Unmarshal(body, &r)
		return r.Token
	}
	return tok("juma@kikundi.tz"), tok("rashidi@kikundi.tz"), tok("fatuma@kikundi.tz"), tok("asha@kikundi.tz")
}

func TestOffsetCapsPartialAndSummary(t *testing.T) {
	app := offsetTestApp()
	cleanAndSeed(t)
	member, loan := offsetFixture(t, 30000, 100000)
	chair, secretary, treasurer, memberTok := offsetTokens(t, app)

	// Regular mwanachama cannot propose → 403
	code, _ := hReq(t, app, "POST", "/api/v1/loans/"+loan.ID+"/offset-propose", map[string]string{"reason": "x"}, memberTok)
	if code != 403 {
		t.Fatalf("member propose: want 403, got %d", code)
	}

	// Preview shows capped amount = savings (30k < 100k outstanding)
	code, body := hReq(t, app, "GET", "/api/v1/loans/"+loan.ID+"/offset-preview", nil, chair)
	if code != 200 {
		t.Fatalf("preview: %d %s", code, body)
	}
	var pv struct {
		Data struct {
			Eligible         bool   `json:"eligible"`
			Outstanding      string `json:"outstanding"`
			AvailableSavings string `json:"available_savings"`
			OffsetAmount     string `json:"offset_amount"`
		} `json:"data"`
	}
	json.Unmarshal(body, &pv)
	if !pv.Data.Eligible || pv.Data.OffsetAmount != "30000" {
		t.Fatalf("preview wrong: %+v", pv.Data)
	}

	// Chair proposes
	code, body = hReq(t, app, "POST", "/api/v1/loans/"+loan.ID+"/offset-propose", map[string]string{"reason": "defaulter"}, chair)
	if code != 201 {
		t.Fatalf("propose: %d %s", code, body)
	}
	var pr struct {
		Data models.LoanOffsetTransaction `json:"data"`
	}
	json.Unmarshal(body, &pr)
	offID := pr.Data.ID
	if pr.Data.ProposedAmount.String() != "30000" {
		t.Fatalf("proposed amount=%s, want 30000", pr.Data.ProposedAmount)
	}

	// Member cannot approve/execute either
	code, _ = hReq(t, app, "POST", "/api/v1/loan-offsets/"+offID+"/approve", nil, memberTok)
	if code != 403 {
		t.Fatalf("member approve: want 403, got %d", code)
	}
	code, _ = hReq(t, app, "POST", "/api/v1/loan-offsets/"+offID+"/execute", nil, memberTok)
	if code != 403 {
		t.Fatalf("member execute: want 403, got %d", code)
	}

	// Secretary approves (client-supplied approved_by must be ignored)
	code, body = hReq(t, app, "POST", "/api/v1/loan-offsets/"+offID+"/approve",
		map[string]string{"approved_by": "00000000-0000-0000-0000-000000000000"}, secretary)
	if code != 200 {
		t.Fatalf("approve: %d %s", code, body)
	}
	var secUser models.User
	database.DB.Where("email = ?", "rashidi@kikundi.tz").First(&secUser)
	var stored models.LoanOffsetTransaction
	database.DB.First(&stored, "id = ?", offID)
	if stored.ApprovedBy == nil || *stored.ApprovedBy != secUser.ID {
		t.Fatalf("approved_by not from session: %+v", stored.ApprovedBy)
	}

	// Treasurer executes: 30k applied, 70k stays outstanding (not forgiven)
	code, body = hReq(t, app, "POST", "/api/v1/loan-offsets/"+offID+"/execute", nil, treasurer)
	if code != 200 {
		t.Fatalf("execute: %d %s", code, body)
	}
	var loanAfter models.Loan
	database.DB.First(&loanAfter, "id = ?", loan.ID)
	if loanAfter.Status != models.LoanOutstanding {
		t.Fatalf("partial offset must leave loan OUTSTANDING, got %s", loanAfter.Status)
	}
	if loanAfter.BalanceRemaining.String() != "70000" {
		t.Fatalf("balance=%s, want 70000", loanAfter.BalanceRemaining)
	}
	var tresUser models.User
	database.DB.Where("email = ?", "fatuma@kikundi.tz").First(&tresUser)
	database.DB.First(&stored, "id = ?", offID)
	if stored.Status != models.LoanOffsetExecuted || stored.Amount.String() != "30000" {
		t.Fatalf("offset record wrong: %+v", stored)
	}
	if stored.ExecutedBy == nil || *stored.ExecutedBy != tresUser.ID {
		t.Fatalf("executed_by not from session")
	}

	// Re-execute → 400
	code, _ = hReq(t, app, "POST", "/api/v1/loan-offsets/"+offID+"/execute", nil, treasurer)
	if code != 400 {
		t.Fatalf("re-execute: want 400, got %d", code)
	}

	// Member summary: net available 0, offsets itemized distinctly
	code, body = hReq(t, app, "GET", "/api/v1/members/"+member.ID+"/dashboard-summary", nil, memberTok)
	if code != 200 {
		t.Fatalf("summary: %d %s", code, body)
	}
	var sum struct {
		Data struct {
			TotalContributions  string `json:"total_contributions"`
			TotalOffsetsApplied string `json:"total_offsets_applied"`
			AvailableSavings    string `json:"available_savings"`
			RecentContributions []struct {
				Source string `json:"source"`
				Status string `json:"status"`
				Amount string `json:"amount"`
			} `json:"recent_contributions"`
		} `json:"data"`
	}
	json.Unmarshal(body, &sum)
	if sum.Data.TotalOffsetsApplied != "30000" || sum.Data.AvailableSavings != "0" {
		t.Fatalf("summary offsets wrong: %+v", sum.Data)
	}
	found := false
	for _, r := range sum.Data.RecentContributions {
		if r.Source == "loan_offset" && r.Status == "OFFSET_APPLIED" && r.Amount == "30000" {
			found = true
		}
		if r.Source == "loan_offset" && r.Status != "OFFSET_APPLIED" {
			t.Fatalf("offset entry mislabeled as normal payment: %+v", r)
		}
	}
	if !found {
		t.Fatalf("offset entry missing from history: %+v", sum.Data.RecentContributions)
	}
}

func TestOffsetFullCoverClosesLoan(t *testing.T) {
	app := offsetTestApp()
	cleanAndSeed(t)
	_, loan := offsetFixture(t, 150000, 100000)
	chair, secretary, treasurer, _ := offsetTokens(t, app)

	code, body := hReq(t, app, "POST", "/api/v1/loans/"+loan.ID+"/offset-propose", nil, chair)
	if code != 201 {
		t.Fatalf("propose: %d %s", code, body)
	}
	var pr struct {
		Data models.LoanOffsetTransaction `json:"data"`
	}
	json.Unmarshal(body, &pr)
	// Cap = outstanding (100k), never the full 150k savings
	if pr.Data.ProposedAmount.String() != "100000" {
		t.Fatalf("proposed=%s, want 100000 (capped at outstanding)", pr.Data.ProposedAmount)
	}
	hReq(t, app, "POST", "/api/v1/loan-offsets/"+pr.Data.ID+"/approve", nil, secretary)
	code, body = hReq(t, app, "POST", "/api/v1/loan-offsets/"+pr.Data.ID+"/execute", nil, treasurer)
	if code != 200 {
		t.Fatalf("execute: %d %s", code, body)
	}
	var loanAfter models.Loan
	database.DB.First(&loanAfter, "id = ?", loan.ID)
	if loanAfter.Status != models.LoanClosed || loanAfter.BalanceRemaining.String() != "0" {
		t.Fatalf("loan should be CLOSED at 0, got %s %s", loanAfter.Status, loanAfter.BalanceRemaining)
	}
}

func TestOffsetNonOverdueRejected(t *testing.T) {
	app := offsetTestApp()
	cleanAndSeed(t)
	member, loan := offsetFixture(t, 50000, 40000)
	// Move due date to the future → not overdue
	future := time.Now().AddDate(0, 0, 30)
	database.DB.Model(&models.Loan{}).Where("id = ?", loan.ID).Update("due_date", future)
	chair, _, _, _ := offsetTokens(t, app)

	code, body := hReq(t, app, "GET", "/api/v1/loans/"+loan.ID+"/offset-preview", nil, chair)
	if code != 200 {
		t.Fatalf("preview: %d %s", code, body)
	}
	var pv struct {
		Data struct {
			Eligible bool `json:"eligible"`
		} `json:"data"`
	}
	json.Unmarshal(body, &pv)
	if pv.Data.Eligible {
		t.Fatalf("future-due loan must not be eligible")
	}
	code, _ = hReq(t, app, "POST", "/api/v1/loans/"+loan.ID+"/offset-propose", nil, chair)
	if code != 400 {
		t.Fatalf("propose on non-overdue: want 400, got %d", code)
	}
	_ = member
}
