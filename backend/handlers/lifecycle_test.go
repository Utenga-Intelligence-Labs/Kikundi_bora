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
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

func fullApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	services.InitEmail()

	app := fiber.New(fiber.Config{AppName: "Kikundi API Test"})
	app.Use(middleware.SetupCORS())

	api := app.Group("/api/v1")

	authHandler := NewAuthHandler()
	memberHandler := NewMemberHandler()
	welfareHandler := NewWelfareHandler()
	userMgmtHandler := NewUserManagementHandler()

	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	protected.Get("/me", authHandler.Me)
	protected.Post("/auth/logout", authHandler.Logout)

	members := protected.Group("/members")
	members.Get("/", memberHandler.List)

	users := protected.Group("/users")
	users.Post("/create", middleware.RequireRoles(models.RoleChair), userMgmtHandler.CreateUser)
	users.Post("/:id/approve", middleware.RequireRoles(models.RoleSecretary), userMgmtHandler.ApproveUser)

	welfare := protected.Group("/welfare")
	welfare.Post("/events", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.CreateEvent)
	welfare.Post("/events/:id/approve", middleware.RequireRoles(models.RoleChair), welfareHandler.ApproveEvent)

	return app
}

func doLogin(t *testing.T, app *fiber.App, email, password string) string {
	t.Helper()
	body := map[string]string{"email": email, "password": password}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var lr struct {
		Token string `json:"token"`
	}
	json.Unmarshal(data, &lr)
	if resp.StatusCode != 200 {
		t.Fatalf("login returned %d: %s", resp.StatusCode, string(data))
	}
	return lr.Token
}

func doRequest(t *testing.T, app *fiber.App, method, path string, body interface{}, token string) (int, []byte) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, path, err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, data
}

func getNestedID(t *testing.T, data []byte, key string) string {
	t.Helper()
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if nested, ok := m[key].(map[string]interface{}); ok {
		if id, ok := nested["id"].(string); ok {
			return id
		}
	}
	t.Fatalf("could not extract %s.id from: %s", key, string(data))
	return ""
}

func cleanupTestData() {
	database.DB.Exec("DELETE FROM repayments")
	database.DB.Exec("DELETE FROM loan_reviews")
	database.DB.Exec("DELETE FROM loans")
	database.DB.Exec("DELETE FROM welfare_contributions")
	database.DB.Exec("DELETE FROM welfare_events")
	database.DB.Exec("DELETE FROM contribution_edits")
	database.DB.Exec("DELETE FROM contributions")
	database.DB.Exec("DELETE FROM user_approvals")
	database.DB.Exec("DELETE FROM user_positions")
	database.DB.Exec("DELETE FROM audit_logs")
	database.DB.Exec("DELETE FROM notifications")
	database.DB.Exec("DELETE FROM admin_logs")
	database.DB.Exec("DELETE FROM failed_logins")
	database.DB.Exec("DELETE FROM user_sessions")
	database.DB.Exec("DELETE FROM pending_actions")
	database.DB.Exec("DELETE FROM loan_committee_members")
	database.DB.Exec("DELETE FROM members")
	database.DB.Exec("DELETE FROM users")
}

func reseedIfEmpty() {
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		database.Seed()
	}
}

// ---- AUTH LIFECYCLE ----

func TestAuthLifecycle(t *testing.T) {
	app := fullApp()
	cleanupTestData()
	reseedIfEmpty()

	// Login
	token := doLogin(t, app, "juma@kikundi.tz", "demo123")

	// Access protected endpoint
	code, data := doRequest(t, app, "GET", "/api/v1/me", nil, token)
	if code != 200 {
		t.Fatalf("GET /me failed: %d — %s", code, string(data))
	}

	// Parse user info
	var resp struct {
		Name string `json:"name"`
	}
	json.Unmarshal(data, &resp)
	if resp.Name != "Mwenyekiti Juma" {
		t.Errorf("expected Mwenyekiti Juma, got %s", resp.Name)
	}

	// Logout
	code, _ = doRequest(t, app, "POST", "/api/v1/auth/logout", nil, token)
	if code != 200 {
		t.Fatalf("logout failed: %d", code)
	}

	// Token now invalid
	code, _ = doRequest(t, app, "GET", "/api/v1/me", nil, token)
	if code != 401 {
		t.Errorf("expected 401 after logout, got %d", code)
	}
}

// ---- LOAN LIFECYCLE (database level) ----

func TestLoanLifecycleDB(t *testing.T) {
	fullApp()
	cleanupTestData()
	reseedIfEmpty()

	// Get chair and member
	var chair models.User
	database.DB.Where("role = ?", models.RoleChair).First(&chair)

	var member models.Member
	database.DB.Where("deleted_at IS NULL").First(&member)

	// Apply
	loan := models.Loan{
		MemberID: member.ID,
		Amount:   decimal.NewFromInt(100000),
		Status:   models.LoanPending,
	}
	if err := database.DB.Create(&loan).Error; err != nil {
		t.Fatalf("create loan: %v", err)
	}
	if loan.ID == "" {
		t.Fatal("loan ID not populated")
	}
	t.Logf("Loan created: %s", loan.ID)

	// Approve
	database.DB.Model(&loan).Updates(map[string]interface{}{
		"status":          models.LoanApproved,
		"approved_amount": 100000.0,
		"reviewed_by":     chair.ID,
	})

	// Reload and verify
	var reloaded models.Loan
	database.DB.First(&reloaded, "id = ?", loan.ID)
	if reloaded.Status != models.LoanApproved {
		t.Fatalf("loan not approved: %s", reloaded.Status)
	}

	// Disburse
	bal := 100000.0
	database.DB.Model(&loan).Updates(map[string]interface{}{
		"status":            models.LoanOutstanding,
		"balance_remaining": bal,
		"disbursed_by":      chair.ID,
	})

	// Reload
	database.DB.First(&reloaded, "id = ?", loan.ID)
	if reloaded.Status != models.LoanOutstanding {
		t.Fatalf("loan not outstanding: %s", reloaded.Status)
	}
	if reloaded.BalanceRemaining == nil || !reloaded.BalanceRemaining.Equal(decimal.NewFromInt(100000)) {
		t.Fatalf("balance incorrect: %v", reloaded.BalanceRemaining)
	}

	// Repay 40k
	database.DB.Create(&models.Repayment{
		LoanID:       loan.ID,
		MemberID:     member.ID,
		Amount:       decimal.NewFromInt(40000),
		BalanceAfter: decimal.NewFromInt(60000),
		RecordedBy:   chair.ID,
	})
	database.DB.Model(&loan).Update("balance_remaining", 60000.0)

	// Reload
	database.DB.First(&reloaded, "id = ?", loan.ID)
	if !reloaded.BalanceRemaining.Equal(decimal.NewFromInt(60000)) {
		t.Fatalf("balance after repayment: %v", *reloaded.BalanceRemaining)
	}

	// Final repayment — close loan
	database.DB.Create(&models.Repayment{
		LoanID:       loan.ID,
		MemberID:     member.ID,
		Amount:       decimal.NewFromInt(60000),
		BalanceAfter: decimal.NewFromInt(0),
		RecordedBy:   chair.ID,
	})
	database.DB.Model(&loan).Updates(map[string]interface{}{
		"balance_remaining": 0.0,
		"status":            models.LoanClosed,
	})

	database.DB.First(&reloaded, "id = ?", loan.ID)
	if reloaded.Status != models.LoanClosed {
		t.Fatalf("loan not closed: %s", reloaded.Status)
	}
	t.Log("Loan lifecycle complete — closed")
}

// ---- USER APPROVAL FLOW ----

func TestUserApprovalFlow(t *testing.T) {
	app := fullApp()
	cleanupTestData()
	reseedIfEmpty()

	chairToken := doLogin(t, app, "juma@kikundi.tz", "demo123")
	secretaryToken := doLogin(t, app, "rashidi@kikundi.tz", "demo123")

	createBody := map[string]interface{}{
		"full_name": "Mtumiaji Mpya",
		"phone":     "0711111111",
		"role":      "member",
	}
	code, data := doRequest(t, app, "POST", "/api/v1/users/create", createBody, chairToken)
	if code != 201 {
		t.Fatalf("create user failed: %d — %s", code, string(data))
	}
	newUserID := getNestedID(t, data, "data")
	t.Logf("New user created: %s", newUserID)

	// Verify user exists in DB
	var user models.User
	if err := database.DB.First(&user, "id = ?", newUserID).Error; err != nil {
		t.Fatalf("user not found in DB after create: %v", err)
	}
	if user.Status != models.UserStatusPending {
		t.Errorf("expected PENDING, got %s", user.Status)
	}

	// Try approving via HTTP — verify the user ID is correct
	code, data = doRequest(t, app, "POST", "/api/v1/users/"+newUserID+"/approve", nil, secretaryToken)
	if code != 200 {
		t.Fatalf("approve user failed: %d — %s", code, string(data))
	}

	// Verify user is ACTIVE
	database.DB.First(&user, "id = ?", newUserID)
	if user.Status != models.UserStatusActive {
		t.Errorf("expected ACTIVE, got %s", user.Status)
	}
	t.Logf("User approval flow complete: %s is now ACTIVE", user.Name)
}

// ---- WELFARE EVENT LIFECYCLE (database level) ----

func TestWelfareEventLifecycleDB(t *testing.T) {
	fullApp()
	cleanupTestData()
	reseedIfEmpty()

	var member models.Member
	database.DB.Where("deleted_at IS NULL").First(&member)

	var treasurer, chair models.User
	database.DB.Where("role = ?", models.RoleTreasurer).First(&treasurer)
	database.DB.Where("role = ?", models.RoleChair).First(&chair)

	// Create event
	event := models.WelfareEvent{
		MemberID:        member.ID,
		EventType:       models.WelfareMedical,
		Description:     "Matibabu ya dharura",
		AmountRequested: decimal.NewFromInt(50000),
		FundingSource:   models.FundTreasury,
		TreasuryAmount:  decimal.NewFromInt(50000),
		Status:          models.WelfarePending,
		CreatedBy:       treasurer.ID,
	}
	if err := database.DB.Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	// Verify
	var reloaded models.WelfareEvent
	database.DB.First(&reloaded, "id = ?", event.ID)
	if reloaded.Status != models.WelfarePending {
		t.Fatalf("event not pending: %s", reloaded.Status)
	}

	// Approve (treasury-funded auto-completes)
	amount := 50000.0
	database.DB.Model(&event).Updates(map[string]interface{}{
		"status":          models.WelfareCompleted,
		"amount_approved": amount,
		"approved_by":     chair.ID,
	})

	database.DB.First(&reloaded, "id = ?", event.ID)
	if reloaded.Status != models.WelfareCompleted {
		t.Errorf("expected COMPLETED, got %s", reloaded.Status)
	}
	t.Logf("Welfare event lifecycle: %s -> %s", event.ID, reloaded.Status)
}
