package handlers

import (
	"errors"
	"sort"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ============================================================================
// Role-scoped dashboard summary endpoints.
//
// BUG BACKGROUND (data binding): contributions live in TWO tables:
//   1. `contributions`         — treasurer-recorded ("Pokea Mchango"), status PAID
//   2. `member_contributions`  — member self-submitted ("Weka Mchango") with a
//                                verification workflow (PENDING_VERIFICATION →
//                                CONFIRMED / REJECTED)
// The member dashboard previously read ONLY `contributions`, so a member whose
// contribution went through the self-submission flow (e.g. Asha, KKK-0009)
// saw "Akiba Yangu: 0 TZS" even after the contribution was CONFIRMED.
// Every summary below aggregates BOTH stores so a newly confirmed
// contribution is visible immediately.
// ============================================================================

// ---- shared helpers --------------------------------------------------------

// swahiliRole maps canonical role codes to the human-facing Swahili names
// used by the frontend role-switch toggle (per product spec).
var swahiliRole = map[string]string{
	"chair":     "mwenyekiti",
	"secretary": "katibu",
	"treasurer": "mweka-hazina",
	"member":    "mwanachama",
	"admin":     "msimamizi",
}

var swahiliLeadershipRole = map[models.LeadershipRole]string{
	models.LeadershipChair:     "mwenyekiti",
	models.LeadershipSecretary: "katibu",
	models.LeadershipTreasurer: "mweka-hazina",
}

var errGroupNotFound = errors.New("kikundi not found")

// loadGroupOr404 parses :id and returns the group, or a sentinel error the
// caller turns into a 404 response. Multi-tenant guard: this deployment
// serves a single group — a request for any id that is not an existing group
// row is rejected.
func loadGroupOr404(c *fiber.Ctx) (*models.Group, error) {
	id := c.Params("id")
	// RBAC-M01 tenant check: reject foreign group IDs with 404.
	if ok, err := database.IsCurrentGroup(id); err != nil || !ok {
		return nil, errGroupNotFound
	}
	var group models.Group
	if err := database.DB.Where("id = ?", id).First(&group).Error; err != nil {
		return nil, errGroupNotFound
	}
	return &group, nil
}

// currentPeriodInfo returns the current contribution period for a group:
// monthFirst (legacy `contributions.month` value), the YYYY-MM period label
// used by `member_contributions.period_label`, and the next due date.
func currentPeriodInfo(g *models.Group) (monthFirst time.Time, label string, nextDue string) {
	now := time.Now()
	monthFirst = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	label = monthFirst.Format("2006-01")
	if g != nil && g.ContributionDueDate != nil {
		if due, ok := services.NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, now); ok {
			nextDue = due.Format("2006-01-02")
		}
	}
	return monthFirst, label, nextDue
}

// sumContributionsBothStores returns the total confirmed contributions for an
// optional member scope (nil = whole group): PAID rows in `contributions` plus
// CONFIRMED rows in `member_contributions`.
func sumContributionsBothStores(memberID string, onlyAkiba bool) (decimal.Decimal, int64, error) {
	var legacyTotal, mcTotal decimal.Decimal
	var legacyCount, mcCount int64

	lq := database.DB.Model(&models.Contribution{}).
		Where("status = ?", "PAID")
	mq := database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionConfirmed)
	if onlyAkiba {
		mq = mq.Where("contribution_type = ?", models.ContributionAkiba)
	}
	if memberID != "" {
		lq = lq.Where("member_id = ?", memberID)
		mq = mq.Where("member_id = ?", memberID)
	}

	if err := lq.Select("COALESCE(SUM(amount), 0)").Scan(&legacyTotal).Error; err != nil {
		return decimal.Zero, 0, err
	}
	lq.Count(&legacyCount)
	if err := mq.Select("COALESCE(SUM(amount), 0)").Scan(&mcTotal).Error; err != nil {
		return decimal.Zero, 0, err
	}
	mq.Count(&mcCount)

	return legacyTotal.Add(mcTotal), legacyCount + mcCount, nil
}

// requesterIsSelfOrLeadership reports whether the authenticated user may view
// the given target member's / user's data: themself, an admin, a leadership
// user role (chair/secretary/treasurer), or a holder of a current leadership
// position (dual plane).
func requesterIsSelfOrLeadership(c *fiber.Ctx, targetMemberID, targetUserID string) bool {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)
	if userID == "" {
		return false
	}
	if role == models.RoleAdmin {
		return true
	}

	var own models.Member
	hasOwn := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&own).Error == nil
	if hasOwn && own.ID != "" && own.ID == targetMemberID {
		return true
	}
	if userID == targetUserID {
		return true
	}

	// Leadership user role
	if role == models.RoleChair || role == models.RoleSecretary || role == models.RoleTreasurer {
		return true
	}
	// Current leadership position (dual plane — a member can hold a role)
	if hasOwn {
		var n int64
		database.DB.Model(&models.LeadershipPosition{}).
			Where("member_id = ? AND is_current = TRUE", own.ID).
			Count(&n)
		if n > 0 {
			return true
		}
	}
	return false
}

// ---- 1. member dashboard summary -------------------------------------------

type RecentContribution struct {
	ID               string          `json:"id"`
	Source           string          `json:"source"` // "contribution" (treasurer-recorded) | "member_contribution" (self-submitted)
	ContributionType string          `json:"contribution_type"`
	PeriodLabel      string          `json:"period_label"`
	Amount           decimal.Decimal `json:"amount"`
	Status           string          `json:"status"`
	PaidAt           string          `json:"paid_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

type MemberDashboardSummary struct {
	MemberID                  string              `json:"member_id"`
	MemberNo                  string              `json:"member_no"`
	FullName                  string              `json:"full_name"`
	TotalContributions        decimal.Decimal     `json:"total_contributions"`
	ContributionsCount        int64               `json:"contributions_count"`
	WelfareContributionsTotal decimal.Decimal     `json:"welfare_contributions_total"`
	WelfareContributionsCount int64               `json:"welfare_contributions_count"`
	PendingContributionsCount int64               `json:"pending_contributions_count"`
	RejectedContributionsCount int64             `json:"rejected_contributions_count"`
	OutstandingLoansCount     int64               `json:"outstanding_loans_count"`
	OutstandingLoansBalance   decimal.Decimal     `json:"outstanding_loans_balance"`
	ClosedLoansCount          int64               `json:"closed_loans_count"`
	RecentContributions       []RecentContribution `json:"recent_contributions"`
}

// MemberSummary returns the PERSONAL dashboard data of one member. Used by the
// "Mwanachama" view — including when a mwenyekiti/katibu/mweka-hazina views
// their own member view (role switch).
// GET /api/v1/members/:id/dashboard-summary
func (h *DashboardHandler) MemberSummary(c *fiber.Ctx) error {
	memberID := c.Params("id")

	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", memberID).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Mwanachama hajapatikana",
		})
	}

	// Multi-tenant / privacy guard: self, admin or leadership only.
	if !requesterIsSelfOrLeadership(c, member.ID, "") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Huna ruhusa ya kuona data ya mwanachama huu",
		})
	}

	sum := MemberDashboardSummary{
		MemberID: member.ID,
		MemberNo: member.MemberNo,
		FullName: member.FullName,
		RecentContributions: []RecentContribution{},
	}

	// Confirmed AKIBA savings: legacy PAID + CONFIRMED AKIBA self-submissions
	total, count, err := sumContributionsBothStores(member.ID, true)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata hesabu za michango"})
	}
	sum.TotalContributions = total
	sum.ContributionsCount = count

	// Welfare (MFUKO_WA_KIJAMII) confirmed contributions
	database.DB.Model(&models.MemberContribution{}).
		Where("member_id = ? AND status = ? AND contribution_type = ?",
			member.ID, models.ContributionConfirmed, models.ContributionMfuko).
		Select("COALESCE(SUM(amount), 0)").Scan(&sum.WelfareContributionsTotal)
	database.DB.Model(&models.MemberContribution{}).
		Where("member_id = ? AND status = ? AND contribution_type = ?",
			member.ID, models.ContributionConfirmed, models.ContributionMfuko).
		Count(&sum.WelfareContributionsCount)

	database.DB.Model(&models.MemberContribution{}).
		Where("member_id = ? AND status = ?", member.ID, models.ContributionPending).
		Count(&sum.PendingContributionsCount)
	database.DB.Model(&models.MemberContribution{}).
		Where("member_id = ? AND status = ?", member.ID, models.ContributionRejected).
		Count(&sum.RejectedContributionsCount)

	database.DB.Model(&models.Loan{}).
		Where("member_id = ? AND status = ?", member.ID, models.LoanOutstanding).
		Select("COALESCE(SUM(COALESCE(balance_remaining, 0)), 0)").
		Scan(&sum.OutstandingLoansBalance)
	database.DB.Model(&models.Loan{}).
		Where("member_id = ? AND status = ?", member.ID, models.LoanOutstanding).
		Count(&sum.OutstandingLoansCount)
	database.DB.Model(&models.Loan{}).
		Where("member_id = ? AND status = ?", member.ID, models.LoanClosed).
		Count(&sum.ClosedLoansCount)

	// Recent contributions merged from BOTH stores (newest first).
	var legacy []models.Contribution
	database.DB.Where("member_id = ?", member.ID).
		Order("created_at DESC").Limit(10).Find(&legacy)
	for _, l := range legacy {
		sum.RecentContributions = append(sum.RecentContributions, RecentContribution{
			ID:               l.ID,
			Source:           "contribution",
			ContributionType: string(models.ContributionAkiba),
			PeriodLabel:      l.Month.Format("2006-01"),
			Amount:           l.Amount,
			Status:           l.Status,
			PaidAt:           l.PaidAt.Format("2006-01-02"),
			CreatedAt:        l.CreatedAt,
		})
	}
	var mcs []models.MemberContribution
	database.DB.Where("member_id = ?", member.ID).
		Order("created_at DESC").Limit(10).Find(&mcs)
	for _, mc := range mcs {
		sum.RecentContributions = append(sum.RecentContributions, RecentContribution{
			ID:               mc.ID,
			Source:           "member_contribution",
			ContributionType: string(mc.ContributionType),
			PeriodLabel:      mc.PeriodLabel,
			Amount:           mc.Amount,
			Status:           string(mc.Status),
			CreatedAt:        mc.CreatedAt,
		})
	}
	sort.Slice(sum.RecentContributions, func(i, j int) bool {
		return sum.RecentContributions[i].CreatedAt.After(sum.RecentContributions[j].CreatedAt)
	})
	if len(sum.RecentContributions) > 10 {
		sum.RecentContributions = sum.RecentContributions[:10]
	}

	return c.JSON(fiber.Map{"data": sum})
}

// ---- 2. group dashboard summary (Uongozi view) ------------------------------

type GroupDashboardSummary struct {
	GroupID                  string          `json:"group_id"`
	GroupName                string          `json:"group_name"`
	TotalActiveMembers       int64           `json:"total_active_members"`
	TotalContributions       decimal.Decimal `json:"total_contributions"`
	TotalRepayments          decimal.Decimal `json:"total_repayments"`
	TotalDisbursed           decimal.Decimal `json:"total_disbursed"`
	AvailableBalance         decimal.Decimal `json:"available_balance"`
	OutstandingLoansCount    int64           `json:"outstanding_loans_count"`
	OutstandingLoansBalance  decimal.Decimal `json:"outstanding_loans_balance"`
	PendingLoansCount        int64           `json:"pending_loans_count"`
	PendingContributionsCount int64          `json:"pending_contributions_count"`
	ContributionsThisPeriod  decimal.Decimal `json:"contributions_this_period"`
	ContributionInterval     string          `json:"contribution_interval"`
	NextDueDate              string          `json:"next_due_date,omitempty"`
}

// GroupSummary returns group-wide dashboard data (the "Uongozi" view).
// Route access is restricted to leadership positions + admin (see main.go).
// GET /api/v1/groups/:id/dashboard-summary
func (h *DashboardHandler) GroupSummary(c *fiber.Ctx) error {
	group, err := loadGroupOr404(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	sum := GroupDashboardSummary{
		GroupID:              group.ID,
		GroupName:            group.Name,
		ContributionInterval: string(group.ContributionInterval),
	}

	database.DB.Model(&models.Member{}).
		Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").
		Count(&sum.TotalActiveMembers)

	// All confirmed contributions (both stores, AKIBA + welfare)
	total, _, err := sumContributionsBothStores("", false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata hesabu za michango"})
	}
	sum.TotalContributions = total

	database.DB.Model(&models.Repayment{}).
		Select("COALESCE(SUM(amount), 0)").Scan(&sum.TotalRepayments)

	database.DB.Model(&models.Loan{}).
		Where("status IN ?", []models.LoanStatus{models.LoanOutstanding, models.LoanClosed}).
		Select("COALESCE(SUM(COALESCE(approved_amount, 0)), 0)").Scan(&sum.TotalDisbursed)

	// Available balance = confirmed contributions + repayments − disbursed
	// (same formula as services.TreasuryService).
	sum.AvailableBalance = sum.TotalContributions.Add(sum.TotalRepayments).Sub(sum.TotalDisbursed)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).
		Select("COALESCE(SUM(COALESCE(balance_remaining, 0)), 0)").
		Scan(&sum.OutstandingLoansBalance)
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanOutstanding).
		Count(&sum.OutstandingLoansCount)
	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanPending).
		Count(&sum.PendingLoansCount)
	database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionPending).
		Count(&sum.PendingContributionsCount)

	// Contributions for the current period (both stores)
	monthFirst, label, nextDue := currentPeriodInfo(group)
	sum.NextDueDate = nextDue
	var legacyPeriod decimal.Decimal
	database.DB.Model(&models.Contribution{}).
		Where("month = ? AND status = ?", monthFirst, "PAID").
		Select("COALESCE(SUM(amount), 0)").Scan(&legacyPeriod)
	var mcPeriod decimal.Decimal
	database.DB.Model(&models.MemberContribution{}).
		Where("period_label = ? AND status = ?", label, models.ContributionConfirmed).
		Select("COALESCE(SUM(amount), 0)").Scan(&mcPeriod)
	sum.ContributionsThisPeriod = legacyPeriod.Add(mcPeriod)

	return c.JSON(fiber.Map{"data": sum})
}

// ---- 3. katibu (secretary) summary ------------------------------------------

type LatePaymentRow struct {
	MemberID       string           `json:"member_id"`
	MemberNo       string           `json:"member_no"`
	FullName       string           `json:"full_name"`
	Phone          string           `json:"phone"`
	PeriodLabel    string           `json:"period_label"`
	ExpectedAmount *decimal.Decimal `json:"expected_amount"`
}

type KatibuDashboardSummary struct {
	GroupID                 string           `json:"group_id"`
	TotalActiveMembers      int64            `json:"total_active_members"`
	MembersJoinedThisMonth  int64            `json:"members_joined_this_month"`
	MembersLeftThisMonth    int64            `json:"members_left_this_month"`
	PendingUserApprovals    int64            `json:"pending_user_approvals"`
	AnnouncementsThisMonth  int64            `json:"announcements_this_month"`
	PendingContributionsCount int64          `json:"pending_contributions_count"`
	CurrentPeriodLabel      string           `json:"current_period_label"`
	NextDueDate             string           `json:"next_due_date,omitempty"`
	LatePaymentsCount       int64            `json:"late_payments_count"`
	LatePayments            []LatePaymentRow `json:"late_payments"`
}

// GroupSummaryKatibu returns secretary-specific data: membership movement,
// records/announcements activity, and late payments for the current period.
// GET /api/v1/groups/:id/dashboard-summary/katibu
func (h *DashboardHandler) GroupSummaryKatibu(c *fiber.Ctx) error {
	group, err := loadGroupOr404(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	now := time.Now()
	monthFirst := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	sum := KatibuDashboardSummary{
		GroupID:            group.ID,
		CurrentPeriodLabel: monthFirst.Format("2006-01"),
		LatePayments:       []LatePaymentRow{},
	}

	database.DB.Model(&models.Member{}).
		Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").
		Count(&sum.TotalActiveMembers)
	database.DB.Model(&models.Member{}).
		Where("deleted_at IS NULL AND approval_status = 'approved' AND joined_at >= ?", monthFirst).
		Count(&sum.MembersJoinedThisMonth)
	// "Left" = soft-deleted this month OR deactivated (inactive, not deleted)
	database.DB.Model(&models.Member{}).
		Where("deleted_at IS NOT NULL AND deleted_at >= ?", monthFirst).
		Count(&sum.MembersLeftThisMonth)
	var deactivated int64
	database.DB.Model(&models.Member{}).
		Where("deleted_at IS NULL AND is_active = FALSE AND updated_at >= ?", monthFirst).
		Count(&deactivated)
	sum.MembersLeftThisMonth += deactivated

	database.DB.Model(&models.User{}).
		Where("status = ? AND deleted_at IS NULL", models.UserStatusPending).
		Count(&sum.PendingUserApprovals)

	// Records: announcements are persisted in the audit log (table "announcements")
	database.DB.Model(&models.AuditLog{}).
		Where("table_name = ? AND action = ? AND created_at >= ?",
			"announcements", models.AuditCreate, monthFirst).
		Count(&sum.AnnouncementsThisMonth)

	database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionPending).
		Count(&sum.PendingContributionsCount)

	// Late payments: active members with NO confirmed AKIBA contribution for
	// the current period in EITHER store.
	monthFirst, label, nextDue := currentPeriodInfo(group)
	sum.NextDueDate = nextDue

	var late []models.Member
	database.DB.
		Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").
		Where("id NOT IN (SELECT member_id FROM contributions WHERE month = ? AND status = 'PAID')", monthFirst).
		Where("id NOT IN (SELECT member_id FROM member_contributions WHERE period_label = ? AND status = ? AND contribution_type = ?)",
			label, models.ContributionConfirmed, models.ContributionAkiba).
		Order("member_no ASC").
		Find(&late)

	sum.LatePaymentsCount = int64(len(late))
	for _, m := range late {
		sum.LatePayments = append(sum.LatePayments, LatePaymentRow{
			MemberID:       m.ID,
			MemberNo:       m.MemberNo,
			FullName:       m.FullName,
			Phone:          m.Phone,
			PeriodLabel:    label,
			ExpectedAmount: group.FixedContributionAmount,
		})
	}

	return c.JSON(fiber.Map{"data": sum})
}

// ---- 4. mweka-hazina (treasurer) summary ------------------------------------

type DisbursementRow struct {
	LoanID      string          `json:"loan_id"`
	MemberNo    string          `json:"member_no"`
	FullName    string          `json:"full_name"`
	Amount      decimal.Decimal `json:"amount"`
	Status      string          `json:"status"`
	DisbursedAt string          `json:"disbursed_at"`
}

type HazinaDashboardSummary struct {
	GroupID              string            `json:"group_id"`
	CashInConfirmed      decimal.Decimal   `json:"cash_in_confirmed"`
	CashInPending        decimal.Decimal   `json:"cash_in_pending"`
	CashInPendingCount   int64             `json:"cash_in_pending_count"`
	CashInThisPeriod     decimal.Decimal   `json:"cash_in_this_period"`
	ExpectedThisPeriod   *decimal.Decimal  `json:"expected_this_period"`
	RepaymentsTotal      decimal.Decimal   `json:"repayments_total"`
	RepaymentsThisMonth  decimal.Decimal   `json:"repayments_this_month"`
	DisbursementsTotal   decimal.Decimal   `json:"disbursements_total"`
	DisbursementsCount   int64             `json:"disbursements_count"`
	RecentDisbursements  []DisbursementRow `json:"recent_disbursements"`
	AvailableBalance     decimal.Decimal   `json:"available_balance"`
}

// GroupSummaryMwekaHazina returns treasurer-specific data: cash flow in
// (received vs pending verification vs expected), loan disbursements out,
// and the actual available balance.
// GET /api/v1/groups/:id/dashboard-summary/mweka-hazina
func (h *DashboardHandler) GroupSummaryMwekaHazina(c *fiber.Ctx) error {
	group, err := loadGroupOr404(c)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}

	now := time.Now()
	monthFirst := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	sum := HazinaDashboardSummary{
		GroupID:             group.ID,
		RecentDisbursements: []DisbursementRow{},
	}

	// Cash in — confirmed (both stores)
	confirmed, _, err := sumContributionsBothStores("", false)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata hesabu za michango"})
	}
	sum.CashInConfirmed = confirmed

	// Cash in — awaiting verification
	database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionPending).
		Select("COALESCE(SUM(amount), 0)").Scan(&sum.CashInPending)
	database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionPending).
		Count(&sum.CashInPendingCount)

	// Cash in — current period (both stores)
	monthFirst, label, _ := currentPeriodInfo(group)
	var legacyPeriod decimal.Decimal
	database.DB.Model(&models.Contribution{}).
		Where("month = ? AND status = ?", monthFirst, "PAID").
		Select("COALESCE(SUM(amount), 0)").Scan(&legacyPeriod)
	var mcPeriod decimal.Decimal
	database.DB.Model(&models.MemberContribution{}).
		Where("period_label = ? AND status = ?", label, models.ContributionConfirmed).
		Select("COALESCE(SUM(amount), 0)").Scan(&mcPeriod)
	sum.CashInThisPeriod = legacyPeriod.Add(mcPeriod)

	// Expected this period = fixed amount × active members (null if unset)
	var activeMembers int64
	database.DB.Model(&models.Member{}).
		Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").
		Count(&activeMembers)
	if group.FixedContributionAmount != nil {
		expected := group.FixedContributionAmount.Mul(decimal.NewFromInt(activeMembers))
		sum.ExpectedThisPeriod = &expected
	}

	database.DB.Model(&models.Repayment{}).
		Select("COALESCE(SUM(amount), 0)").Scan(&sum.RepaymentsTotal)
	database.DB.Model(&models.Repayment{}).
		Where("paid_at >= ? AND paid_at < ?", monthFirst, monthFirst.AddDate(0, 1, 0)).
		Select("COALESCE(SUM(amount), 0)").Scan(&sum.RepaymentsThisMonth)

	// Disbursements: loans actually disbursed (OUTSTANDING or CLOSED)
	disbursedQ := database.DB.Model(&models.Loan{}).
		Where("status IN ? AND disbursed_at IS NOT NULL",
			[]models.LoanStatus{models.LoanOutstanding, models.LoanClosed})
	disbursedQ.Select("COALESCE(SUM(COALESCE(approved_amount, 0)), 0)").Scan(&sum.DisbursementsTotal)
	disbursedQ.Count(&sum.DisbursementsCount)

	var loans []models.Loan
	database.DB.
		Where("status IN ? AND disbursed_at IS NOT NULL",
			[]models.LoanStatus{models.LoanOutstanding, models.LoanClosed}).
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name")
		}).
		Order("disbursed_at DESC").Limit(10).Find(&loans)
	for _, l := range loans {
		amount := l.Amount
		if l.ApprovedAmount != nil {
			amount = *l.ApprovedAmount
		}
		row := DisbursementRow{
			LoanID:      l.ID,
			Amount:      amount,
			Status:      string(l.Status),
			DisbursedAt: l.DisbursedAt.Format("2006-01-02"),
		}
		if l.Member != nil {
			row.MemberNo = l.Member.MemberNo
			row.FullName = l.Member.FullName
		}
		sum.RecentDisbursements = append(sum.RecentDisbursements, row)
	}

	// Available balance = confirmed cash in + repayments − disbursed
	sum.AvailableBalance = sum.CashInConfirmed.Add(sum.RepaymentsTotal).Sub(sum.DisbursementsTotal)

	return c.JSON(fiber.Map{"data": sum})
}

// ---- 5. user roles -----------------------------------------------------------

type UserRolesResponse struct {
	UserID              string   `json:"user_id"`
	MemberID            *string  `json:"member_id"`
	PrimaryRole         string   `json:"primary_role"`
	LeadershipPositions []string `json:"leadership_positions"`
	Roles               []string `json:"roles"`
}

// UserRoles returns every role a user holds in the group — including the
// implicit "mwanachama" role every linked member has, so a person can be e.g.
// ["mwenyekiti", "mwanachama"]. The frontend uses this to decide whether to
// show the role-switch toggle.
// GET /api/v1/users/:id/roles
func (h *DashboardHandler) UserRoles(c *fiber.Ctx) error {
	targetUserID := c.Params("id")

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", targetUserID).
		First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Mtumiaji hajapatikana",
		})
	}

	// Privacy guard: self, admin or leadership only.
	if !requesterIsSelfOrLeadership(c, "", targetUserID) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Huna ruhusa ya kuona majukumu ya mtumiaji huyu",
		})
	}

	resp := UserRolesResponse{
		UserID:              user.ID,
		PrimaryRole:         string(user.Role),
		LeadershipPositions: []string{},
		Roles:               []string{},
	}

	seen := map[string]bool{}
	addRole := func(r string) {
		if !seen[r] {
			seen[r] = true
			resp.Roles = append(resp.Roles, r)
		}
	}

	// Admins have no member row and hold only the admin role.
	if user.Role == models.RoleAdmin {
		addRole(swahiliRole["admin"])
		return c.JSON(fiber.Map{"data": resp})
	}

	// Linked member → implicit "mwanachama" role
	var member models.Member
	hasMember := database.DB.Where("user_id = ? AND deleted_at IS NULL", user.ID).
		First(&member).Error == nil
	if hasMember {
		mid := member.ID
		resp.MemberID = &mid

		// Leadership positions held by this member (a member can hold several)
		var positions []models.LeadershipPosition
		database.DB.Where("member_id = ? AND is_current = TRUE", member.ID).
			Find(&positions)
		for _, p := range positions {
			resp.LeadershipPositions = append(resp.LeadershipPositions, string(p.Role))
		}
	}

	// Order: leadership positions first (they trump the account role), then
	// the account role, then the implicit member role.
	for _, p := range resp.LeadershipPositions {
		if r, ok := swahiliLeadershipRole[models.LeadershipRole(p)]; ok {
			addRole(r)
		}
	}
	if r, ok := swahiliRole[string(user.Role)]; ok {
		addRole(r)
	}
	if hasMember {
		addRole(swahiliRole["member"])
	}

	return c.JSON(fiber.Map{"data": resp})
}
