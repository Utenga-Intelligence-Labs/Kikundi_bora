package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

// scopedTestApp wires up the role-scoped dashboard endpoints plus the
// contribution flows needed to drive them (mirrors the main.go routing).
func scopedTestApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()

	app := fiber.New(fiber.Config{AppName: "Kikundi API Test"})
	api := app.Group("/api/v1")

	authHandler := NewAuthHandler()
	dashHandler := NewDashboardHandler()
	memberContribHandler := NewMemberContributionHandler()
	contribHandler := NewContributionHandler()

	api.Post("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)
	protected.Get("/me", authHandler.Me)

	members := protected.Group("/members")
	members.Get("/:id/dashboard-summary", dashHandler.MemberSummary)

	groups := protected.Group("/groups")
	groups.Get("/:id/dashboard-summary",
		middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer, models.LeadershipSecretary),
		dashHandler.GroupSummary)
	groups.Get("/:id/dashboard-summary/katibu",
		middleware.RequireLeadership(models.LeadershipSecretary),
		dashHandler.GroupSummaryKatibu)
	groups.Get("/:id/dashboard-summary/mweka-hazina",
		middleware.RequireLeadership(models.LeadershipTreasurer),
		dashHandler.GroupSummaryMwekaHazina)

	users := protected.Group("/users")
	users.Get("/:id/roles", dashHandler.UserRoles)

	michango := protected.Group("/michango")
	michango.Post("/", memberContribHandler.Submit)
	michango.Post("/:id/confirm",
		middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer, models.LeadershipSecretary),
		memberContribHandler.Confirm)

	contribs := protected.Group("/contributions")
	contribs.Post("/", middleware.RequirePosition(models.PositionTreasurer), contribHandler.Create)

	return app
}

// scopedCleanAndSeed resets all tables the scoped dashboards read from and
// re-seeds demo data (users, members, leadership, group).
func scopedCleanAndSeed(t *testing.T) {
	t.Helper()
	for _, stmt := range []string{
		"DELETE FROM loan_reviews",
		"DELETE FROM repayments",
		"DELETE FROM loans",
		"DELETE FROM contribution_edits",
		"DELETE FROM contributions",
		"DELETE FROM member_contributions",
		"DELETE FROM user_approvals",
		"DELETE FROM user_sessions",
		"DELETE FROM failed_logins",
		"DELETE FROM loan_committee_members",
		"DELETE FROM leadership_positions",
		"DELETE FROM user_positions",
		"DELETE FROM notifications",
		"DELETE FROM audit_logs",
		"DELETE FROM admin_logs",
		"DELETE FROM pending_actions",
		"DELETE FROM welfare_contributions",
		"DELETE FROM welfare_events",
		"DELETE FROM meeting_attendances",
		"DELETE FROM meetings",
		"DELETE FROM fines",
		"DELETE FROM contribution_cycles",
		"DELETE FROM fine_offence_types",
		"DELETE FROM fine_settings",
		"DELETE FROM group_setting_proposals",
		"DELETE FROM members",
		"DELETE FROM users",
		"DELETE FROM groups",
	} {
		if err := database.DB.Exec(stmt).Error; err != nil {
			t.Fatalf("cleanup %s: %v", stmt, err)
		}
	}
	database.Seed()             // re-creates users, members, positions
	database.EnsureGroupSetup() // single group row
}

// scopedHelpers fetches frequently needed fixtures straight from the DB.
func scopedFixtures(t *testing.T) (group models.Group, asha, juma models.User, ashaMember models.Member, otherMember models.Member) {
	t.Helper()
	if err := database.DB.First(&group).Error; err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := database.DB.Where("email = ?", "asha@kikundi.tz").First(&asha).Error; err != nil {
		t.Fatalf("asha user: %v", err)
	}
	if err := database.DB.Where("email = ?", "juma@kikundi.tz").First(&juma).Error; err != nil {
		t.Fatalf("juma user: %v", err)
	}
	// Asha's own linked member row (created by BackfillMembersFromUsers)
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", asha.ID).First(&ashaMember).Error; err != nil {
		t.Fatalf("asha member: %v", err)
	}
	// A different member (one of the phone-book seeded rows, not linked to asha)
	if err := database.DB.Where("user_id IS NULL AND deleted_at IS NULL").Order("member_no").First(&otherMember).Error; err != nil {
		// fall back: any member that is not asha's
		if err2 := database.DB.Where("id <> ? AND deleted_at IS NULL", ashaMember.ID).Order("member_no").First(&otherMember).Error; err2 != nil {
			t.Fatalf("other member: %v", err2)
		}
	}
	return
}

func decodeData(t *testing.T, body []byte, out interface{}) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v — %s", err, body)
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		t.Fatalf("decode data: %v — %s", err, envelope.Data)
	}
}

// ============================================================================
// REGRESSION (priority-1 bug): a member-submitted contribution ("Weka
// Mchango" flow → member_contributions, CONFIRMED) must be visible
// IMMEDIATELY in the member's dashboard summary. Previously the member view
// read only the treasurer-recorded `contributions` table, so confirmed
// self-submissions showed "Akiba Yangu: 0 TZS" (real case: Asha, KKK-0009).
// ============================================================================
func TestMemberDashboardSummaryShowsSelfSubmittedContribution(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	group, asha, _, ashaMember, _ := scopedFixtures(t)
	_ = group
	_ = asha

	ashaToken := hLogin(t, app, "asha@kikundi.tz", "demo123")
	treasurerToken := hLogin(t, app, "fatuma@kikundi.tz", "demo123")

	path := "/api/v1/members/" + ashaMember.ID + "/dashboard-summary"

	// Baseline: empty summary, member sees own data
	code, d := hGet(t, app, path, ashaToken)
	if code != 200 {
		t.Fatalf("own summary: %d %s", code, d)
	}
	var sum MemberDashboardSummary
	decodeData(t, d, &sum)
	if !sum.TotalContributions.Equal(decimal.Zero) || sum.ContributionsCount != 0 {
		t.Fatalf("baseline should be empty, got %s / %d", sum.TotalContributions, sum.ContributionsCount)
	}

	// Step 1: member self-submits an AKIBA contribution ("Weka Mchango")
	period := time.Now().Format("2006-01")
	code, d = hPost(t, app, "/api/v1/michango", map[string]interface{}{
		"contribution_type": "AKIBA",
		"period_label":      period,
		"amount":            40000.0,
		"proof_message":     "Malipo ya mchango kwa simu",
	}, ashaToken)
	if code != 201 {
		t.Fatalf("submit contribution: %d %s", code, d)
	}
	contribID := hExtract(t, d, "data")

	// While PENDING_VERIFICATION it must NOT count toward akiba, but must
	// appear as pending — visible immediately, no stale cache.
	code, d = hGet(t, app, path, ashaToken)
	if code != 200 {
		t.Fatalf("summary after submit: %d", code)
	}
	decodeData(t, d, &sum)
	if !sum.TotalContributions.Equal(decimal.Zero) {
		t.Errorf("pending contribution must not count toward akiba yet, got %s", sum.TotalContributions)
	}
	if sum.PendingContributionsCount != 1 {
		t.Errorf("pending count = %d, want 1", sum.PendingContributionsCount)
	}

	// Step 2: treasurer confirms
	code, d = hPost(t, app, "/api/v1/michango/"+contribID+"/confirm", nil, treasurerToken)
	if code != 200 {
		t.Fatalf("confirm contribution: %d %s", code, d)
	}

	// Step 3 (regression assertion): the confirmed contribution is visible in
	// the summary IMMEDIATELY after confirmation.
	code, d = hGet(t, app, path, ashaToken)
	if code != 200 {
		t.Fatalf("summary after confirm: %d", code)
	}
	decodeData(t, d, &sum)
	if !sum.TotalContributions.Equal(decimal.NewFromInt(40000)) {
		t.Errorf("total_contributions = %s, want 40000 (the Asha KKK-0009 bug)", sum.TotalContributions)
	}
	if sum.ContributionsCount != 1 {
		t.Errorf("contributions_count = %d, want 1", sum.ContributionsCount)
	}
	if sum.PendingContributionsCount != 0 {
		t.Errorf("pending count after confirm = %d, want 0", sum.PendingContributionsCount)
	}
	if len(sum.RecentContributions) != 1 {
		t.Fatalf("recent_contributions len = %d, want 1", len(sum.RecentContributions))
	}
	rc := sum.RecentContributions[0]
	if rc.Source != "member_contribution" || rc.Status != "CONFIRMED" || rc.PeriodLabel != period {
		t.Errorf("recent contribution wrong: %+v", rc)
	}
	if !rc.Amount.Equal(decimal.NewFromInt(40000)) {
		t.Errorf("recent amount = %s, want 40000", rc.Amount)
	}
}

// A treasurer-recorded contribution (legacy `contributions` table) must also
// appear in the member summary — both stores are aggregated.
func TestMemberDashboardSummaryShowsTreasurerRecordedContribution(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	_, _, _, ashaMember, otherMember := scopedFixtures(t)

	treasurerToken := hLogin(t, app, "fatuma@kikundi.tz", "demo123")
	ashaToken := hLogin(t, app, "asha@kikundi.tz", "demo123")

	month := time.Now().Format("2006-01")
	code, d := hPost(t, app, "/api/v1/contributions", map[string]interface{}{
		"member_id":      ashaMember.ID,
		"amount":         25000.0,
		"month":          month,
		"paid_at":        time.Now().Format("2006-01-02"),
		"payment_method": "CASH",
	}, treasurerToken)
	if code != 201 {
		t.Fatalf("record contribution: %d %s", code, d)
	}

	code, d = hGet(t, app, "/api/v1/members/"+ashaMember.ID+"/dashboard-summary", ashaToken)
	if code != 200 {
		t.Fatalf("summary: %d", code)
	}
	var sum MemberDashboardSummary
	decodeData(t, d, &sum)
	if !sum.TotalContributions.Equal(decimal.NewFromInt(25000)) {
		t.Errorf("total_contributions = %s, want 25000", sum.TotalContributions)
	}
	if len(sum.RecentContributions) != 1 || sum.RecentContributions[0].Source != "contribution" {
		t.Errorf("legacy contribution missing from recent list: %+v", sum.RecentContributions)
	}

	// Loans: outstanding balance + closed count
	approved := decimal.NewFromInt(100000)
	balance := decimal.NewFromInt(70000)
	database.DB.Create(&models.Loan{
		MemberID:         ashaMember.ID,
		Amount:           approved,
		ApprovedAmount:   &approved,
		BalanceRemaining: &balance,
		Status:           models.LoanOutstanding,
		DueDate:          time.Now().AddDate(1, 0, 0),
	})
	database.DB.Create(&models.Loan{
		MemberID: ashaMember.ID,
		Amount:   decimal.NewFromInt(50000),
		Status:   models.LoanClosed,
		DueDate:  time.Now().AddDate(-1, 0, 0),
	})

	code, d = hGet(t, app, "/api/v1/members/"+ashaMember.ID+"/dashboard-summary", ashaToken)
	decodeData(t, d, &sum)
	if sum.OutstandingLoansCount != 1 || !sum.OutstandingLoansBalance.Equal(balance) {
		t.Errorf("outstanding loans = %d / %s, want 1 / 70000", sum.OutstandingLoansCount, sum.OutstandingLoansBalance)
	}
	if sum.ClosedLoansCount != 1 {
		t.Errorf("closed loans = %d, want 1", sum.ClosedLoansCount)
	}

	// Welfare (MFUKO_WA_KIJAMII) confirmed contributions are reported
	// separately and do not pollute the akiba total.
	database.DB.Create(&models.MemberContribution{
		MemberID:         otherMember.ID,
		ContributionType: models.ContributionMfuko,
		PeriodLabel:      time.Now().Format("2006-01"),
		Amount:           decimal.NewFromInt(5000),
		Status:           models.ContributionConfirmed,
	})
	code, d = hGet(t, app, "/api/v1/members/"+otherMember.ID+"/dashboard-summary", treasurerToken)
	decodeData(t, d, &sum)
	if !sum.WelfareContributionsTotal.Equal(decimal.NewFromInt(5000)) || sum.WelfareContributionsCount != 1 {
		t.Errorf("welfare totals = %s / %d, want 5000 / 1", sum.WelfareContributionsTotal, sum.WelfareContributionsCount)
	}
	if !sum.TotalContributions.Equal(decimal.Zero) {
		t.Errorf("welfare contribution must not count toward akiba, got %s", sum.TotalContributions)
	}
}

// Access control on the member summary: self OK, leadership OK, another
// member NO (multi-tenant/privacy guard).
func TestMemberSummaryAccessControl(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	_, asha, _, ashaMember, otherMember := scopedFixtures(t)
	_ = asha

	ashaToken := hLogin(t, app, "asha@kikundi.tz", "demo123")
	treasurerToken := hLogin(t, app, "fatuma@kikundi.tz", "demo123")
	chairToken := hLogin(t, app, "juma@kikundi.tz", "demo123")

	// Another member's data: forbidden for a plain member
	code, _ := hGet(t, app, "/api/v1/members/"+otherMember.ID+"/dashboard-summary", ashaToken)
	if code != 403 {
		t.Errorf("member viewing another member: got %d, want 403", code)
	}
	// Leadership may view any member
	code, _ = hGet(t, app, "/api/v1/members/"+otherMember.ID+"/dashboard-summary", treasurerToken)
	if code != 200 {
		t.Errorf("treasurer viewing member: got %d, want 200", code)
	}
	code, _ = hGet(t, app, "/api/v1/members/"+ashaMember.ID+"/dashboard-summary", chairToken)
	if code != 200 {
		t.Errorf("chair viewing member: got %d, want 200", code)
	}
	// Unknown member: 404
	code, _ = hGet(t, app, "/api/v1/members/00000000-0000-0000-0000-000000000000/dashboard-summary", ashaToken)
	if code != 404 {
		t.Errorf("unknown member: got %d, want 404", code)
	}
}

// Group summary ("Uongozi" view): leadership sees it, members do not, and the
// totals aggregate BOTH contribution stores.
func TestGroupDashboardSummary(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	group, asha, _, _, otherMember := scopedFixtures(t)
	_ = asha

	chairToken := hLogin(t, app, "juma@kikundi.tz", "demo123")
	ashaToken := hLogin(t, app, "asha@kikundi.tz", "demo123")

	path := "/api/v1/groups/" + group.ID + "/dashboard-summary"

	// Plain members are blocked by the leadership guard
	code, _ := hGet(t, app, path, ashaToken)
	if code != 403 {
		t.Errorf("member accessing group summary: got %d, want 403", code)
	}

	// Nonexistent group: 404
	code, _ = hGet(t, app, "/api/v1/groups/00000000-0000-0000-0000-000000000000/dashboard-summary", chairToken)
	if code != 404 {
		t.Errorf("unknown group: got %d, want 404", code)
	}

	// Seed money from BOTH stores:
	// 1. member self-submits, treasurer confirms (40,000)
	period := time.Now().Format("2006-01")
	code, d := hPost(t, app, "/api/v1/michango", map[string]interface{}{
		"contribution_type": "AKIBA",
		"period_label":      period,
		"amount":            40000.0,
		"proof_message":     "malipo simu",
	}, ashaToken)
	if code != 201 {
		t.Fatalf("submit: %d %s", code, d)
	}
	contribID := hExtract(t, d, "data")
	treasurerToken := hLogin(t, app, "fatuma@kikundi.tz", "demo123")
	if code, d = hPost(t, app, "/api/v1/michango/"+contribID+"/confirm", nil, treasurerToken); code != 200 {
		t.Fatalf("confirm: %d %s", code, d)
	}
	// 2. treasurer records a legacy contribution (25,000)
	code, d = hPost(t, app, "/api/v1/contributions", map[string]interface{}{
		"member_id":      otherMember.ID,
		"amount":         25000.0,
		"month":          period,
		"paid_at":        time.Now().Format("2006-01-02"),
		"payment_method": "CASH",
	}, treasurerToken)
	if code != 201 {
		t.Fatalf("record: %d %s", code, d)
	}
	// 3. a disbursed loan (100,000) with a repayment (30,000)
	approved := decimal.NewFromInt(100000)
	balance := decimal.NewFromInt(70000)
	now := time.Now()
	loan := models.Loan{
		MemberID:         otherMember.ID,
		Amount:           approved,
		ApprovedAmount:   &approved,
		BalanceRemaining: &balance,
		Status:           models.LoanOutstanding,
		DueDate:          now.AddDate(1, 0, 0),
		DisbursedAt:      &now,
	}
	database.DB.Create(&loan)
	database.DB.Create(&models.Repayment{
		LoanID:       loan.ID,
		MemberID:     otherMember.ID,
		Amount:       decimal.NewFromInt(30000),
		BalanceAfter: decimal.NewFromInt(0),
		RecordedBy:   asha.ID,
		PaidAt:       now,
	})

	code, d = hGet(t, app, path, chairToken)
	if code != 200 {
		t.Fatalf("group summary: %d %s", code, d)
	}
	var sum GroupDashboardSummary
	decodeData(t, d, &sum)

	if !sum.TotalContributions.Equal(decimal.NewFromInt(65000)) {
		t.Errorf("total_contributions = %s, want 65000 (both stores)", sum.TotalContributions)
	}
	if !sum.ContributionsThisPeriod.Equal(decimal.NewFromInt(65000)) {
		t.Errorf("contributions_this_period = %s, want 65000", sum.ContributionsThisPeriod)
	}
	if !sum.TotalRepayments.Equal(decimal.NewFromInt(30000)) {
		t.Errorf("total_repayments = %s, want 30000", sum.TotalRepayments)
	}
	if !sum.TotalDisbursed.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("total_disbursed = %s, want 100000", sum.TotalDisbursed)
	}
	// available = 65000 + 30000 − 100000 = −5000
	if !sum.AvailableBalance.Equal(decimal.NewFromInt(-5000)) {
		t.Errorf("available_balance = %s, want -5000", sum.AvailableBalance)
	}
	if sum.OutstandingLoansCount != 1 || !sum.OutstandingLoansBalance.Equal(decimal.NewFromInt(70000)) {
		t.Errorf("outstanding = %d / %s, want 1 / 70000", sum.OutstandingLoansCount, sum.OutstandingLoansBalance)
	}
	if sum.TotalActiveMembers == 0 {
		t.Error("total_active_members should be > 0")
	}
	if sum.GroupID != group.ID || sum.GroupName != group.Name {
		t.Errorf("group identity wrong: %s / %s", sum.GroupID, sum.GroupName)
	}
}

// Katibu summary: membership movement + late payments for the current period
// (union of both contribution stores).
func TestKatibuDashboardSummary(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	group, asha, _, ashaMember, otherMember := scopedFixtures(t)
	_ = asha
	_ = otherMember

	secretaryToken := hLogin(t, app, "rashidi@kikundi.tz", "demo123")
	chairToken := hLogin(t, app, "juma@kikundi.tz", "demo123")

	path := "/api/v1/groups/" + group.ID + "/dashboard-summary/katibu"

	// Chair (not secretary) is blocked — katibu endpoint is secretary-scoped
	code, _ := hGet(t, app, path, chairToken)
	if code != 403 {
		t.Errorf("chair accessing katibu summary: got %d, want 403", code)
	}

	code, d := hGet(t, app, path, secretaryToken)
	if code != 200 {
		t.Fatalf("katibu summary: %d %s", code, d)
	}
	var sum KatibuDashboardSummary
	decodeData(t, d, &sum)

	// Nobody has paid this period → every active member is late
	if sum.LatePaymentsCount != sum.TotalActiveMembers {
		t.Errorf("late_payments_count = %d, want %d (all active members)", sum.LatePaymentsCount, sum.TotalActiveMembers)
	}
	if len(sum.LatePayments) != int(sum.LatePaymentsCount) {
		t.Errorf("late payments list len %d != count %d", len(sum.LatePayments), sum.LatePaymentsCount)
	}
	if sum.CurrentPeriodLabel != time.Now().Format("2006-01") {
		t.Errorf("current_period_label = %s", sum.CurrentPeriodLabel)
	}

	// Asha pays via the self-submission flow (confirm by treasurer)…
	period := time.Now().Format("2006-01")
	code, d = hPost(t, app, "/api/v1/michango", map[string]interface{}{
		"contribution_type": "AKIBA",
		"period_label":      period,
		"amount":            40000.0,
		"proof_message":     "malipo simu",
	}, hLogin(t, app, "asha@kikundi.tz", "demo123"))
	if code != 201 {
		t.Fatalf("submit: %d %s", code, d)
	}
	contribID := hExtract(t, d, "data")
	if code, d = hPost(t, app, "/api/v1/michango/"+contribID+"/confirm", nil,
		hLogin(t, app, "fatuma@kikundi.tz", "demo123")); code != 200 {
		t.Fatalf("confirm: %d %s", code, d)
	}
	// …another member pays via the treasurer-recorded flow
	if code, d = hPost(t, app, "/api/v1/contributions", map[string]interface{}{
		"member_id":      otherMember.ID,
		"amount":         25000.0,
		"month":          period,
		"paid_at":        time.Now().Format("2006-01-02"),
		"payment_method": "CASH",
	}, hLogin(t, app, "fatuma@kikundi.tz", "demo123")); code != 201 {
		t.Fatalf("record: %d %s", code, d)
	}

	code, d = hGet(t, app, path, secretaryToken)
	decodeData(t, d, &sum)
	if sum.LatePaymentsCount != sum.TotalActiveMembers-2 {
		t.Errorf("late_payments_count after 2 payments = %d, want %d", sum.LatePaymentsCount, sum.TotalActiveMembers-2)
	}
	for _, lp := range sum.LatePayments {
		if lp.MemberID == ashaMember.ID {
			t.Error("asha paid this period (self-submitted, confirmed) but is still listed as late")
		}
		if lp.MemberID == otherMember.ID {
			t.Error("member paid this period (treasurer-recorded) but is still listed as late")
		}
	}

	// A PENDING (unconfirmed) self-submission must NOT clear the late flag
	if code, d = hPost(t, app, "/api/v1/michango", map[string]interface{}{
		"contribution_type": "AKIBA",
		"period_label":      period,
		"amount":            40000.0,
		"proof_message":     "pending",
	}, hLogin(t, app, "asha@kikundi.tz", "demo123")); code != 201 {
		t.Fatalf("submit pending: %d %s", code, d)
	}
	// (asha now has a pending duplicate — ignore; use a fresh member)
	var fresh models.Member
	if err := database.DB.Where("id NOT IN ? AND deleted_at IS NULL AND is_active = TRUE",
		[]string{ashaMember.ID, otherMember.ID}).Order("member_no").First(&fresh).Error; err != nil {
		t.Fatalf("fresh member: %v", err)
	}
	database.DB.Create(&models.MemberContribution{
		MemberID:         fresh.ID,
		ContributionType: models.ContributionAkiba,
		PeriodLabel:      period,
		Amount:           decimal.NewFromInt(10000),
		Status:           models.ContributionPending,
	})
	code, d = hGet(t, app, path, secretaryToken)
	decodeData(t, d, &sum)
	stillLate := false
	for _, lp := range sum.LatePayments {
		if lp.MemberID == fresh.ID {
			stillLate = true
		}
	}
	if !stillLate {
		t.Error("member with only a PENDING contribution must still be listed as late")
	}
	if sum.PendingContributionsCount != 2 {
		t.Errorf("pending_contributions_count = %d, want 2", sum.PendingContributionsCount)
	}
}

// Mweka-hazina summary: cash flow in (confirmed / pending / expected),
// disbursements out, repayments, available balance.
func TestHazinaDashboardSummary(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	group, asha, _, _, otherMember := scopedFixtures(t)
	_ = asha

	treasurerToken := hLogin(t, app, "fatuma@kikundi.tz", "demo123")
	chairToken := hLogin(t, app, "juma@kikundi.tz", "demo123")

	path := "/api/v1/groups/" + group.ID + "/dashboard-summary/mweka-hazina"

	// Chair is blocked — hazina endpoint is treasurer-scoped
	code, _ := hGet(t, app, path, chairToken)
	if code != 403 {
		t.Errorf("chair accessing hazina summary: got %d, want 403", code)
	}

	// Money in: confirmed self-submission (40k) + pending (10k) + legacy (25k)
	period := time.Now().Format("2006-01")
	ashaToken := hLogin(t, app, "asha@kikundi.tz", "demo123")
	code, d := hPost(t, app, "/api/v1/michango", map[string]interface{}{
		"contribution_type": "AKIBA", "period_label": period,
		"amount": 40000.0, "proof_message": "malipo",
	}, ashaToken)
	if code != 201 {
		t.Fatalf("submit: %d %s", code, d)
	}
	if code, d = hPost(t, app, "/api/v1/michango/"+hExtract(t, d, "data")+"/confirm", nil, treasurerToken); code != 200 {
		t.Fatalf("confirm: %d %s", code, d)
	}
	if code, d = hPost(t, app, "/api/v1/michango", map[string]interface{}{
		"contribution_type": "AKIBA", "period_label": period,
		"amount": 10000.0, "proof_message": "pending",
	}, ashaToken); code != 201 {
		t.Fatalf("submit pending: %d %s", code, d)
	}
	if code, d = hPost(t, app, "/api/v1/contributions", map[string]interface{}{
		"member_id": otherMember.ID, "amount": 25000.0, "month": period,
		"paid_at": time.Now().Format("2006-01-02"), "payment_method": "CASH",
	}, treasurerToken); code != 201 {
		t.Fatalf("record: %d %s", code, d)
	}

	// Money out: a disbursed loan (100k) + a repayment (30k)
	approved := decimal.NewFromInt(100000)
	balance := decimal.NewFromInt(70000)
	now := time.Now()
	database.DB.Create(&models.Loan{
		MemberID:         otherMember.ID,
		Amount:           approved,
		ApprovedAmount:   &approved,
		BalanceRemaining: &balance,
		Status:           models.LoanOutstanding,
		DueDate:          now.AddDate(1, 0, 0),
		DisbursedAt:      &now,
	})
	var loan models.Loan
	database.DB.Where("member_id = ?", otherMember.ID).First(&loan)
	database.DB.Create(&models.Repayment{
		LoanID:       loan.ID,
		MemberID:     otherMember.ID,
		Amount:       decimal.NewFromInt(30000),
		BalanceAfter: decimal.NewFromInt(70000),
		RecordedBy:   asha.ID,
		PaidAt:       now,
	})

	code, d = hGet(t, app, path, treasurerToken)
	if code != 200 {
		t.Fatalf("hazina summary: %d %s", code, d)
	}
	var sum HazinaDashboardSummary
	decodeData(t, d, &sum)

	if !sum.CashInConfirmed.Equal(decimal.NewFromInt(65000)) {
		t.Errorf("cash_in_confirmed = %s, want 65000", sum.CashInConfirmed)
	}
	if !sum.CashInPending.Equal(decimal.NewFromInt(10000)) || sum.CashInPendingCount != 1 {
		t.Errorf("cash_in_pending = %s / %d, want 10000 / 1", sum.CashInPending, sum.CashInPendingCount)
	}
	if !sum.CashInThisPeriod.Equal(decimal.NewFromInt(65000)) {
		t.Errorf("cash_in_this_period = %s, want 65000", sum.CashInThisPeriod)
	}
	if !sum.RepaymentsTotal.Equal(decimal.NewFromInt(30000)) {
		t.Errorf("repayments_total = %s, want 30000", sum.RepaymentsTotal)
	}
	if !sum.DisbursementsTotal.Equal(decimal.NewFromInt(100000)) || sum.DisbursementsCount != 1 {
		t.Errorf("disbursements = %s / %d, want 100000 / 1", sum.DisbursementsTotal, sum.DisbursementsCount)
	}
	// available = 65000 + 30000 − 100000 = −5000
	if !sum.AvailableBalance.Equal(decimal.NewFromInt(-5000)) {
		t.Errorf("available_balance = %s, want -5000", sum.AvailableBalance)
	}
	if len(sum.RecentDisbursements) != 1 || sum.RecentDisbursements[0].MemberNo != otherMember.MemberNo {
		t.Errorf("recent disbursements wrong: %+v", sum.RecentDisbursements)
	}
	if sum.ExpectedThisPeriod != nil {
		t.Errorf("expected_this_period should be null when no fixed amount is set, got %s", *sum.ExpectedThisPeriod)
	}

	// With a fixed contribution amount configured, expected = fixed × active
	fixed := decimal.NewFromInt(10000)
	if err := database.DB.Model(&models.Group{}).Where("id = ?", group.ID).
		Update("fixed_contribution_amount", fixed).Error; err != nil {
		t.Fatalf("set fixed amount: %v", err)
	}
	code, d = hGet(t, app, path, treasurerToken)
	decodeData(t, d, &sum)
	if sum.ExpectedThisPeriod == nil {
		t.Fatal("expected_this_period should be set with a fixed amount")
	}
	want := fixed.Mul(decimal.NewFromInt(sum2active(t)))
	if !sum.ExpectedThisPeriod.Equal(want) {
		t.Errorf("expected_this_period = %s, want %s", *sum.ExpectedThisPeriod, want)
	}
}

func sum2active(t *testing.T) int64 {
	t.Helper()
	var n int64
	database.DB.Model(&models.Member{}).Where("is_active = TRUE AND deleted_at IS NULL").Count(&n)
	return n
}

// User roles endpoint — including the multi-role edge case: one person can be
// e.g. katibu AND mweka-hazina AND still a mwanachama.
func TestUserRolesMultiRole(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	_, asha, juma, _, _ := scopedFixtures(t)

	ashaToken := hLogin(t, app, "asha@kikundi.tz", "demo123")
	jumaToken := hLogin(t, app, "juma@kikundi.tz", "demo123")

	// Plain member: just ["mwanachama"]
	code, d := hGet(t, app, "/api/v1/users/"+asha.ID+"/roles", ashaToken)
	if code != 200 {
		t.Fatalf("asha roles: %d %s", code, d)
	}
	var roles UserRolesResponse
	decodeData(t, d, &roles)
	if len(roles.Roles) != 1 || roles.Roles[0] != "mwanachama" {
		t.Errorf("asha roles = %v, want [mwanachama]", roles.Roles)
	}
	if roles.MemberID == nil || *roles.MemberID == "" {
		t.Error("asha should have a linked member_id")
	}

	// Chair with leadership position: ["mwenyekiti", "mwanachama"]
	code, d = hGet(t, app, "/api/v1/users/"+juma.ID+"/roles", jumaToken)
	if code != 200 {
		t.Fatalf("juma roles: %d %s", code, d)
	}
	decodeData(t, d, &roles)
	if len(roles.Roles) != 2 || roles.Roles[0] != "mwenyekiti" || roles.Roles[1] != "mwanachama" {
		t.Errorf("juma roles = %v, want [mwenyekiti mwanachama]", roles.Roles)
	}
	if len(roles.LeadershipPositions) != 1 || roles.LeadershipPositions[0] != "MWENYEKITI" {
		t.Errorf("juma leadership positions = %v, want [MWENYEKITI]", roles.LeadershipPositions)
	}
	if roles.PrimaryRole != "chair" {
		t.Errorf("primary_role = %s, want chair", roles.PrimaryRole)
	}

	// MULTI-ROLE EDGE CASE: give the chair a second, concurrent position
	// (HAZINA) — one person, several hats, still a member.
	var jumaMember models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", juma.ID).First(&jumaMember).Error; err != nil {
		t.Fatalf("juma member: %v", err)
	}
	database.DB.Create(&models.LeadershipPosition{
		MemberID:  jumaMember.ID,
		Role:      models.LeadershipTreasurer,
		IsCurrent: true,
	})
	code, d = hGet(t, app, "/api/v1/users/"+juma.ID+"/roles", jumaToken)
	decodeData(t, d, &roles)
	want := []string{"mwenyekiti", "mweka-hazina", "mwanachama"}
	if len(roles.Roles) != len(want) {
		t.Fatalf("multi-role: got %v, want %v", roles.Roles, want)
	}
	for i := range want {
		if roles.Roles[i] != want[i] {
			t.Errorf("multi-role: got %v, want %v", roles.Roles, want)
		}
	}
	if len(roles.LeadershipPositions) != 2 {
		t.Errorf("leadership positions = %v, want 2 entries", roles.LeadershipPositions)
	}

	// Access control: a plain member cannot inspect another user's roles
	code, _ = hGet(t, app, "/api/v1/users/"+juma.ID+"/roles", ashaToken)
	if code != 403 {
		t.Errorf("member viewing another user's roles: got %d, want 403", code)
	}
	// Unknown user: 404
	code, _ = hGet(t, app, "/api/v1/users/00000000-0000-0000-0000-000000000000/roles", jumaToken)
	if code != 404 {
		t.Errorf("unknown user: got %d, want 404", code)
	}
}

// The fixed-amount contribution guard also affects the member submit flow;
// this test pins the interaction between group settings and summaries.
func TestGroupSummaryReflectsFixedAmountSetting(t *testing.T) {
	app := scopedTestApp()
	scopedCleanAndSeed(t)
	group, _, _, _, _ := scopedFixtures(t)

	fixed := decimal.NewFromInt(5000)
	if err := database.DB.Model(&models.Group{}).Where("id = ?", group.ID).
		Updates(map[string]interface{}{
			"fixed_contribution_amount": fixed,
			"contribution_due_date":     fmt.Sprintf("%d", 28),
		}).Error; err != nil {
		t.Fatalf("update group: %v", err)
	}

	chairToken := hLogin(t, app, "juma@kikundi.tz", "demo123")
	code, d := hGet(t, app, "/api/v1/groups/"+group.ID+"/dashboard-summary", chairToken)
	if code != 200 {
		t.Fatalf("group summary: %d %s", code, d)
	}
	var sum GroupDashboardSummary
	decodeData(t, d, &sum)
	if sum.ContributionInterval != "monthly" {
		t.Errorf("contribution_interval = %s, want monthly", sum.ContributionInterval)
	}
	if sum.NextDueDate == "" {
		t.Error("next_due_date should be set when a due-date spec exists")
	}
}
