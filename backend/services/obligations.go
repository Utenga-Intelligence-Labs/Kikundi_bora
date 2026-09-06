package services

import (
	"errors"
	"log"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

// ── Payment allocation (CONFIRMED RULE) ──────────────────────────────────────
// Order is fixed and must not change: ARREARS (oldest first) → CURRENT CYCLE
// → FINES (oldest first). Partial payments apply in the same order and leave
// the rest outstanding; overpayments return a remainder (change).

const (
	AllocArrears = "arrears"
	AllocCurrent = "current"
	AllocFine    = "fine"
)

// OwedItem is one outstanding obligation, pre-sorted by the caller:
// arrears cycles oldest-due first, then the current cycle, then fines
// oldest-occurrence first.
type OwedItem struct {
	Kind  string // AllocArrears | AllocCurrent | AllocFine
	Ref   string // cycle label (cycles) or fine ID (fines)
	Label string // human label for receipts
	Owed  decimal.Decimal
}

// AllocatedLine is one slice of a payment applied to an OwedItem.
type AllocatedLine struct {
	Kind   string
	Ref    string
	Label  string
	Amount decimal.Decimal
}

// AllocatePayment splits amount across items in order. Pure function —
// unit-tested, no DB. Returns applied lines plus any remainder.
func AllocatePayment(items []OwedItem, amount decimal.Decimal) (lines []AllocatedLine, remainder decimal.Decimal) {
	lines = []AllocatedLine{}
	rest := amount
	if rest.LessThanOrEqual(decimal.Zero) {
		return lines, rest
	}
	for _, it := range items {
		if rest.LessThanOrEqual(decimal.Zero) {
			break
		}
		if it.Owed.LessThanOrEqual(decimal.Zero) {
			continue
		}
		take := it.Owed
		if take.GreaterThan(rest) {
			take = rest
		}
		lines = append(lines, AllocatedLine{Kind: it.Kind, Ref: it.Ref, Label: it.Label, Amount: take})
		rest = rest.Sub(take)
	}
	return lines, rest
}

// ── Cycle tracking ───────────────────────────────────────────────────────────

// cycleGraceDays returns the grace applied before a missed cycle counts as
// arrears: the active late-contribution offence's grace if one exists, else
// the legacy FineSettings grace, else 0.
func cycleGraceDays(groupID string) int {
	var ots []models.FineOffenceType
	database.DB.Where("group_id = ? AND kind = ? AND status = ?",
		groupID, models.OffenceLateContribution, models.OffenceActive).Find(&ots)
	if len(ots) > 0 {
		return ots[0].GracePeriodDays
	}
	var s models.FineSettings
	if err := database.DB.Where("group_id = ?", groupID).First(&s).Error; err == nil {
		return s.GracePeriodDays
	}
	return 0
}

// paidInCycleWindow sums reconciled (treasurer-recorded PAID) contributions
// for a member inside (start, end]. CONFIRMED member self-submissions flow
// into Contribution rows at confirm time, so they are counted here exactly
// once — never double-counted.
func paidInCycleWindow(memberID string, start, end time.Time) decimal.Decimal {
	type sumRow struct {
		Total string
	}
	var r sumRow
	database.DB.Raw(`SELECT COALESCE(SUM(amount),0)::text AS total FROM contributions
		WHERE member_id = ? AND status = 'PAID' AND paid_at > ? AND paid_at <= ?`,
		memberID, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&r)
	d, _ := decimal.NewFromString(r.Total)
	return d
}

// RefreshMemberCycles rebuilds a member's cycle rows from settings +
// reconciled payments. Idempotent: expected amounts are snapshots set once
// at row creation; paid amounts are recomputed (never incremented), so
// re-running can never double-count.
func RefreshMemberCycles(groupID, memberID string, now time.Time) error {
	var g models.Group
	if err := database.DB.First(&g, "id = ?", groupID).Error; err != nil {
		return err
	}
	if g.ContributionDueDate == nil || g.ContributionInterval == "" {
		return errors.New("group has no contribution schedule")
	}
	var m models.Member
	if err := database.DB.First(&m, "id = ?", memberID).Error; err != nil {
		return err
	}
	today := dateOf(now)
	joined := dateOf(m.JoinedAt)

	// Walk due dates from joining to the current open cycle (inclusive).
	// Past cycles become arrears rows; the open cycle carries current due.
	due, ok := NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, joined)
	if !ok {
		return errors.New("invalid contribution schedule")
	}
	for i := 0; i < 240 && ok; i++ {
		label := ContributionCycleLabel(g.ContributionInterval, due)
		if err := ensureCycleRow(&g, &m, label, dateOf(due)); err != nil {
			return err
		}
		if dateOf(due).After(today) {
			break // current open cycle recorded; stop (never future cycles)
		}
		due, ok = NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, dateOf(due).AddDate(0, 0, 1))
	}
	return nil
}

// ensureCycleRow creates the cycle row if missing (snapshotting the current
// fixed amount), then recomputes paid/status from reconciled rows.
func ensureCycleRow(g *models.Group, m *models.Member, label string, due time.Time) error {
	var c models.ContributionCycle
	err := database.DB.Where("group_id = ? AND member_id = ? AND cycle_label = ?",
		g.ID, m.ID, label).First(&c).Error
	if err != nil {
		expected := decimal.Zero
		if g.FixedContributionAmount != nil {
			expected = *g.FixedContributionAmount
		}
		c = models.ContributionCycle{
			GroupID: g.ID, MemberID: m.ID, CycleLabel: label,
			DueDate: dateOf(due), ExpectedAmount: expected,
		}
		if err := database.DB.Create(&c).Error; err != nil {
			return err
		}
	}
	prev, ok := PreviousContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, dateOf(due))
	if !ok {
		return nil
	}
	// Allocated payments are already persisted on the cycle row. Only import
	// legacy reconciled contributions while the row has not been allocated yet;
	// otherwise a payment made today against an older cycle would be erased on
	// the next refresh because it falls outside that cycle's date window.
	paid := c.PaidAmount
	legacyPaid := paidInCycleWindow(m.ID, prev, dateOf(due))
	if paid.IsZero() || legacyPaid.GreaterThan(paid) {
		paid = legacyPaid
	}
	status := models.CycleUnpaid
	switch {
	case paid.GreaterThanOrEqual(c.ExpectedAmount) && c.ExpectedAmount.GreaterThan(decimal.Zero):
		status = models.CyclePaid
	case paid.GreaterThan(decimal.Zero):
		status = models.CyclePartial
	case c.ExpectedAmount.IsZero():
		status = models.CyclePaid // nothing assessed → nothing owed
	}
	return database.DB.Model(&c).Updates(map[string]interface{}{
		"paid_amount": paid, "status": status,
	}).Error
}

// BackdateMemberArrears generates MAIN-contribution cycle rows for a member
// from `from` (inclusive) up to the current open cycle, using the group's
// fixed amount snapshot per cycle via ensureCycleRow. It touches ONLY the
// contribution_cycles table — social-fund (welfare) obligations are
// voluntary by nature and are NEVER generated here, regardless of settings.
// Returns the cycle labels created/found, for audit logging. Idempotent:
// re-running creates no duplicates (ensureCycleRow is upsert-by-label).
func BackdateMemberArrears(groupID, memberID string, from, now time.Time) ([]string, error) {
	var g models.Group
	if err := database.DB.First(&g, "id = ?", groupID).Error; err != nil {
		return nil, err
	}
	if g.ContributionDueDate == nil || g.ContributionInterval == "" {
		return nil, errors.New("group has no contribution schedule")
	}
	var m models.Member
	if err := database.DB.First(&m, "id = ?", memberID).Error; err != nil {
		return nil, err
	}
	today := dateOf(now)
	start := dateOf(from)
	if start.After(today) {
		return nil, errors.New("backdate start must not be in the future")
	}
	due, ok := NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, start)
	if !ok {
		return nil, errors.New("invalid contribution schedule")
	}
	labels := []string{}
	for i := 0; i < 240 && ok; i++ {
		label := ContributionCycleLabel(g.ContributionInterval, due)
		if err := ensureCycleRow(&g, &m, label, dateOf(due)); err != nil {
			return labels, err
		}
		labels = append(labels, label)
		if dateOf(due).After(today) {
			break // current open cycle recorded; stop (never future cycles)
		}
		due, ok = NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, dateOf(due).AddDate(0, 0, 1))
	}
	return labels, nil
}

// groupHasSchedule reports whether the group has an approved contribution
// schedule (interval + due date). Without it there are no cycles to track —
// callers must skip quietly instead of warning once per member.
func groupHasSchedule(g *models.Group) bool {
	return g != nil && g.ContributionDueDate != nil && g.ContributionInterval != ""
}

// RefreshGroupCycles refreshes every active approved member. Called from the
// scheduler tick and on-demand before summaries. When the group has no
// approved contribution schedule yet, it returns after a single concise log
// line (previously it warned once PER MEMBER on every tick).
func RefreshGroupCycles(groupID string, now time.Time) {
	var g models.Group
	if err := database.DB.First(&g, "id = ?", groupID).Error; err != nil {
		log.Printf("WARN: cycle refresh: group %s not found: %v", groupID, err)
		return
	}
	if !groupHasSchedule(&g) {
		log.Printf("Scheduler: group %s has no approved contribution schedule — skipping cycle refresh (mwenyekiti proposes settings in Mipangilio, katibu approves)", groupID)
		return
	}
	var members []models.Member
	database.DB.Where("deleted_at IS NULL AND is_active = TRUE AND approval_status = 'approved'").
		Find(&members)
	for i := range members {
		if err := RefreshMemberCycles(groupID, members[i].ID, now); err != nil {
			log.Printf("WARN: cycle refresh %s: %v", members[i].MemberNo, err)
		}
	}
}

// ── Obligation summaries ─────────────────────────────────────────────────────

type ArrearsItem struct {
	CycleLabel string          `json:"cycle_label"`
	DueDate    time.Time       `json:"due_date"`
	Expected   decimal.Decimal `json:"expected_amount"`
	Paid       decimal.Decimal `json:"paid_amount"`
	Owed       decimal.Decimal `json:"owed"`
}

type FineItem struct {
	ID             string          `json:"id"`
	OffenceName    string          `json:"offence_name"`
	OffenceKind    string          `json:"offence_kind"`
	Amount         decimal.Decimal `json:"amount"`
	OccurrenceDate time.Time       `json:"occurrence_date"`
	CycleLabel     string          `json:"cycle_label"`
	Status         string          `json:"status"`
	WaiverStatus   string          `json:"waiver_status"`
}

type MemberObligations struct {
	MemberID          string          `json:"member_id"`
	MemberNo          string          `json:"member_no"`
	FullName          string          `json:"full_name"`
	TotalArrears      decimal.Decimal `json:"total_arrears"`
	CurrentCycleDue   decimal.Decimal `json:"current_cycle_due"`
	CurrentCycleLabel string          `json:"current_cycle_label"`
	TotalFinesUnpaid  decimal.Decimal `json:"total_fines_unpaid"`
	GrandTotal        decimal.Decimal `json:"grand_total_owed"`
	HasSchedule       bool             `json:"has_schedule"`
	ItemizedArrears   []ArrearsItem   `json:"itemized_arrears"`
	ItemizedFines     []FineItem      `json:"itemized_fines"`
}

// GetMemberObligations refreshes then builds the combined summary.
// total_arrears covers closed cycles past grace; current_cycle_due is the
// open cycle's remainder; grand_total = arrears + current + fines.
func GetMemberObligations(groupID, memberID string, now time.Time) (*MemberObligations, error) {
	var m models.Member
	if err := database.DB.First(&m, "id = ?", memberID).Error; err != nil {
		return nil, err
	}
	// No contribution schedule configured yet (no due date): there are no
	// cycles to assess, but fines still apply. Return a zeros summary
	// rather than 404 so the page explains itself. This is an expected
	// pre-configuration state — stay quiet here (the scheduler and group
	// refresh already log a single actionable line per tick).
	var g models.Group
	if dbErr := database.DB.First(&g, "id = ?", groupID).Error; dbErr != nil || !groupHasSchedule(&g) {
		out := &MemberObligations{
			MemberID: m.ID, MemberNo: m.MemberNo, FullName: m.FullName,
			TotalArrears: decimal.Zero, CurrentCycleDue: decimal.Zero,
			TotalFinesUnpaid: decimal.Zero, GrandTotal: decimal.Zero,
			HasSchedule: false,
			ItemizedArrears: []ArrearsItem{}, ItemizedFines: []FineItem{},
		}
		return out, appendFinesOnly(groupID, memberID, out)
	}
	if err := RefreshMemberCycles(groupID, memberID, now); err != nil {
		log.Printf("WARN: obligations refresh %s: %v", memberID, err)
	}
	today := dateOf(now)
	grace := cycleGraceDays(groupID)

	var cycles []models.ContributionCycle
	database.DB.Where("group_id = ? AND member_id = ?", groupID, memberID).
		Order("due_date ASC").Find(&cycles)

	out := &MemberObligations{
		MemberID: m.ID, MemberNo: m.MemberNo, FullName: m.FullName,
		TotalArrears: decimal.Zero, CurrentCycleDue: decimal.Zero,
		TotalFinesUnpaid: decimal.Zero, GrandTotal: decimal.Zero,
		HasSchedule: true,
		ItemizedArrears: []ArrearsItem{}, ItemizedFines: []FineItem{},
	}
	for _, c := range cycles {
		owed := c.ExpectedAmount.Sub(c.PaidAmount)
		if owed.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if !c.DueDate.After(today) && today.After(c.DueDate.AddDate(0, 0, grace)) {
			// closed cycle past grace → arrears
			out.TotalArrears = out.TotalArrears.Add(owed)
			out.ItemizedArrears = append(out.ItemizedArrears, ArrearsItem{
				CycleLabel: c.CycleLabel, DueDate: c.DueDate,
				Expected: c.ExpectedAmount, Paid: c.PaidAmount, Owed: owed,
			})
		} else if c.DueDate.After(today) || c.DueDate.Equal(today) {
			// current open cycle
			out.CurrentCycleDue = out.CurrentCycleDue.Add(owed)
			out.CurrentCycleLabel = c.CycleLabel
		} else {
			// closed but still inside grace → counts toward current due
			out.CurrentCycleDue = out.CurrentCycleDue.Add(owed)
			out.CurrentCycleLabel = c.CycleLabel
		}
	}

	var frows []fineRow
	database.DB.Raw(`
		SELECT f.id, o.name AS offence_name, o.kind AS offence_kind,
		       f.amount::text AS amount, f.occurrence_date AS occurrence,
		       f.contribution_cycle_label AS cycle_label, f.status, f.waiver_status AS waiver
		  FROM fines f JOIN fine_offence_types o ON o.id = f.offence_type_id
		 WHERE f.group_id = ? AND f.member_id = ? AND f.status = 'unpaid'
		 ORDER BY f.occurrence_date ASC`, groupID, memberID).Scan(&frows)
	appendFineItems(out, frows)

	out.GrandTotal = out.TotalArrears.Add(out.CurrentCycleDue).Add(out.TotalFinesUnpaid)
	return out, nil
}

type fineRow struct {
	ID          string
	OffenceName string
	OffenceKind string
	Amount      string
	Occurrence  time.Time
	CycleLabel  string
	Status      string
	Waiver      string
}

// appendFineItems folds raw fine rows into the summary (shared by the full
// and the fines-only/no-schedule paths).
func appendFineItems(out *MemberObligations, frows []fineRow) {
	for _, fr := range frows {
		amt, _ := decimal.NewFromString(fr.Amount)
		out.TotalFinesUnpaid = out.TotalFinesUnpaid.Add(amt)
		out.ItemizedFines = append(out.ItemizedFines, FineItem{
			ID: fr.ID, OffenceName: fr.OffenceName, OffenceKind: fr.OffenceKind,
			Amount: amt, OccurrenceDate: fr.Occurrence, CycleLabel: fr.CycleLabel,
			Status: fr.Status, WaiverStatus: fr.Waiver,
		})
	}
}

// appendFinesOnly loads just the unpaid fines for members whose group has no
// contribution schedule (no cycles exist to assess).
func appendFinesOnly(groupID, memberID string, out *MemberObligations) error {
	var frows []fineRow
	if err := database.DB.Raw(`
		SELECT f.id, o.name AS offence_name, o.kind AS offence_kind,
		       f.amount::text AS amount, f.occurrence_date AS occurrence,
		       f.contribution_cycle_label AS cycle_label, f.status, f.waiver_status AS waiver
		  FROM fines f JOIN fine_offence_types o ON o.id = f.offence_type_id
		 WHERE f.group_id = ? AND f.member_id = ? AND f.status = 'unpaid'
		 ORDER BY f.occurrence_date ASC`, groupID, memberID).Scan(&frows).Error; err != nil {
		return err
	}
	appendFineItems(out, frows)
	out.GrandTotal = out.TotalArrears.Add(out.CurrentCycleDue).Add(out.TotalFinesUnpaid)
	return nil
}

type MemberObligationRollup struct {
	MemberID     string          `json:"member_id"`
	MemberNo     string          `json:"member_no"`
	FullName     string          `json:"full_name"`
	TotalArrears decimal.Decimal `json:"total_arrears"`
	CurrentDue   decimal.Decimal `json:"current_cycle_due"`
	TotalFines   decimal.Decimal `json:"total_fines_unpaid"`
	GrandTotal   decimal.Decimal `json:"grand_total_owed"`
}

type GroupObligations struct {
	TotalArrearsOutstanding decimal.Decimal          `json:"total_arrears_outstanding"`
	TotalFinesOutstanding   decimal.Decimal          `json:"total_fines_outstanding"`
	MemberCountOwing        int                      `json:"member_count_owing"`
	Members                 []MemberObligationRollup `json:"members"`
}

// GetGroupObligations aggregates outstanding arrears + fines group-wide.
func GetGroupObligations(groupID string, now time.Time) (*GroupObligations, error) {
	RefreshGroupCycles(groupID, now)
	var members []models.Member
	database.DB.Where("deleted_at IS NULL AND is_active = TRUE AND approval_status = 'approved'").
		Order("member_no ASC").Find(&members)
	out := &GroupObligations{
		TotalArrearsOutstanding: decimal.Zero,
		TotalFinesOutstanding:   decimal.Zero,
		Members:                 []MemberObligationRollup{},
	}
	for i := range members {
		mo, err := GetMemberObligations(groupID, members[i].ID, now)
		if err != nil {
			continue
		}
		out.TotalArrearsOutstanding = out.TotalArrearsOutstanding.
			Add(mo.TotalArrears).Add(mo.CurrentCycleDue)
		out.TotalFinesOutstanding = out.TotalFinesOutstanding.Add(mo.TotalFinesUnpaid)
		if mo.GrandTotal.GreaterThan(decimal.Zero) {
			out.MemberCountOwing++
		}
		out.Members = append(out.Members, MemberObligationRollup{
			MemberID: members[i].ID, MemberNo: mo.MemberNo, FullName: mo.FullName,
			TotalArrears: mo.TotalArrears, CurrentDue: mo.CurrentCycleDue,
			TotalFines: mo.TotalFinesUnpaid, GrandTotal: mo.GrandTotal,
		})
	}
	return out, nil
}
