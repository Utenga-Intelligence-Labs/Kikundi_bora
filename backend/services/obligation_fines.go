package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ── Allocation persistence ───────────────────────────────────────────────────

// AllocationReceipt is returned to the payer: what each shilling covered.
type AllocationReceipt struct {
	Lines     []AllocatedLine `json:"lines"`
	Applied   decimal.Decimal `json:"applied"`
	Remainder decimal.Decimal `json:"remainder"`
}

// cycleMonthForLabel maps a cycle label back to a month date for the
// treasurer Contribution rows. Monthly labels are exact; other intervals
// fall back to the cycle's due month.
func cycleMonthForLabel(label string, due time.Time) time.Time {
	if t, err := time.Parse("2006-01", label); err == nil {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(due.Year(), due.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// ApplyAllocation splits a payment across a member's obligations in the
// confirmed order (arrears → current → fines) and persists everything in one
// transaction: cycle rows updated, Contribution rows created/merged per
// cycle portion, fines marked paid. Returns a receipt. Remainder (overpay)
// is reported, never silently kept as credit.
func ApplyAllocation(groupID, memberID string, amount decimal.Decimal, recordedByUserID, paymentMethod, reference string, paidAt time.Time, now time.Time) (*AllocationReceipt, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("kiasi lazima kiwe zaidi ya sifuri")
	}
	var m models.Member
	if err := database.DB.First(&m, "id = ?", memberID).Error; err != nil {
		return nil, errors.New("mwanachama hajapatikana")
	}
	if err := RefreshMemberCycles(groupID, memberID, now); err != nil {
		return nil, err
	}

	// Build the ordered obligation list: unpaid cycles oldest-due first
	// (arrears and current alike — allocation order is positional), then
	// unpaid fines oldest-occurrence first.
	type cycleRow struct {
		Label    string
		Due      time.Time
		Expected string
		Paid     string
	}
	var crows []cycleRow
	database.DB.Raw(`SELECT cycle_label AS label, due_date AS due,
		expected_amount::text AS expected, paid_amount::text AS paid
		FROM contribution_cycles WHERE group_id = ? AND member_id = ?
		ORDER BY due_date ASC`, groupID, memberID).Scan(&crows)

	today := dateOf(now)
	grace := cycleGraceDays(groupID)
	var items []OwedItem
	cycleDue := map[string]time.Time{}
	for _, cr := range crows {
		exp, _ := decimal.NewFromString(cr.Expected)
		paid, _ := decimal.NewFromString(cr.Paid)
		owed := exp.Sub(paid)
		if owed.LessThanOrEqual(decimal.Zero) {
			continue
		}
		kind := AllocArrears
		dueDate := dateOf(cr.Due)
		if dueDate.After(today) || dueDate.Equal(today) || !today.After(dueDate.AddDate(0, 0, grace)) {
			kind = AllocCurrent
		}
		items = append(items, OwedItem{Kind: kind, Ref: cr.Label, Label: "Mchango " + cr.Label, Owed: owed})
		cycleDue[cr.Label] = dueDate
	}
	type fineRow struct {
		ID     string
		Amount string
		Name   string
	}
	var frows []fineRow
	database.DB.Raw(`SELECT f.id, f.amount::text AS amount, o.name AS name
		FROM fines f JOIN fine_offence_types o ON o.id = f.offence_type_id
		WHERE f.group_id = ? AND f.member_id = ? AND f.status = 'unpaid'
		ORDER BY f.occurrence_date ASC`, groupID, memberID).Scan(&frows)
	for _, fr := range frows {
		amt, _ := decimal.NewFromString(fr.Amount)
		if amt.LessThanOrEqual(decimal.Zero) {
			continue
		}
		items = append(items, OwedItem{Kind: AllocFine, Ref: fr.ID, Label: "Faini: " + fr.Name, Owed: amt})
	}

	lines, remainder := AllocatePayment(items, amount)
	receipt := &AllocationReceipt{Lines: lines, Remainder: remainder,
		Applied: amount.Sub(remainder)}
	if len(lines) == 0 {
		return receipt, nil
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, ln := range lines {
			switch ln.Kind {
			case AllocArrears, AllocCurrent:
				var c models.ContributionCycle
				if err := tx.Where("group_id = ? AND member_id = ? AND cycle_label = ?",
					groupID, memberID, ln.Ref).First(&c).Error; err != nil {
					return err
				}
				newPaid := c.PaidAmount.Add(ln.Amount)
				status := models.CyclePartial
				if newPaid.GreaterThanOrEqual(c.ExpectedAmount) {
					status = models.CyclePaid
				}
				if err := tx.Model(&c).Updates(map[string]interface{}{
					"paid_amount": newPaid, "status": status,
				}).Error; err != nil {
					return err
				}
				// Reconciled Contribution row per cycle portion (merge into
				// the month row so monthly reports stay consistent).
				month := cycleMonthForLabel(ln.Ref, c.DueDate)
				var existing models.Contribution
				err := tx.Where("member_id = ? AND month = ?", memberID,
					month.Format("2006-01-02")).First(&existing).Error
				if err == nil {
					existing.Amount = existing.Amount.Add(ln.Amount)
					if err := tx.Save(&existing).Error; err != nil {
						return err
					}
				} else {
					row := models.Contribution{
						MemberID:      memberID,
						RecordedBy:    recordedByUserID,
						Amount:        ln.Amount,
						Month:         month,
						PaidAt:        dateOf(paidAt),
						PaymentMethod: paymentMethod,
						Status:        "PAID",
						ConfirmedBy:   &recordedByUserID,
						Notes:         &[]string{"Ugawaji malipo: " + ln.Label}[0],
					}
					if reference != "" {
						row.ReferenceNumber = reference
					}
					if err := tx.Create(&row).Error; err != nil {
						return err
					}
				}
			case AllocFine:
				var f models.Fine
				if err := tx.Where("id = ? AND status = 'unpaid'", ln.Ref).First(&f).Error; err != nil {
					return fmt.Errorf("faini %s haipatikani: %w", ln.Ref, err)
				}
				nowT := now
				if err := tx.Model(&f).Updates(map[string]interface{}{
					"status": "paid", "collected_by": recordedByUserID,
					"collected_at": &nowT,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Mirror the contribution portion into the double-entry ledger (single
	// aggregated entry; best-effort).
	var contribTotal decimal.Decimal
	for _, ln := range lines {
		if ln.Kind == AllocArrears || ln.Kind == AllocCurrent {
			contribTotal = contribTotal.Add(ln.Amount)
		}
	}
	if contribTotal.GreaterThan(decimal.Zero) {
		if err := PostContribution(m.MemberNo, contribTotal, dateOf(paidAt), recordedByUserID,
			fmt.Sprintf("Malipo %s (%s)", m.MemberNo, reference)); err != nil {
			log.Printf("WARN: ledger auto-post allocation %s: %v", memberID, err)
		}
	}
	return receipt, nil
}

// ── Offence fine creation ────────────────────────────────────────────────────

// ComputeOffenceAmount snapshots the fine for an offence type: fixed amount,
// or percentage of the group's fixed contribution. Returns false when the
// amount cannot be determined (no fine is created then).
func ComputeOffenceAmount(ot *models.FineOffenceType, fixedContribution *decimal.Decimal) (decimal.Decimal, bool) {
	switch ot.FineType {
	case models.FineTypeFixed:
		if ot.FineAmount != nil && ot.FineAmount.GreaterThan(decimal.Zero) {
			return *ot.FineAmount, true
		}
	case models.FineTypePercentage:
		if ot.FinePercentage != nil && ot.FinePercentage.GreaterThan(decimal.Zero) &&
			fixedContribution != nil && fixedContribution.GreaterThan(decimal.Zero) {
			return fixedContribution.Mul(*ot.FinePercentage).Div(decimal.NewFromInt(100)).Round(2), true
		}
	}
	return decimal.Zero, false
}

// CreateCycleFine creates one lateness fine idempotently: if a fine already
// exists for (group, member, offence, cycle) it is returned as-is.
func CreateCycleFine(groupID, memberID, offenceID, cycleLabel string, due time.Time, reason string) (*models.Fine, error) {
	var existing models.Fine
	if err := database.DB.Where("group_id = ? AND member_id = ? AND offence_type_id = ? AND contribution_cycle_label = ?",
		groupID, memberID, offenceID, cycleLabel).First(&existing).Error; err == nil {
		return &existing, nil
	}
	var ot models.FineOffenceType
	if err := database.DB.First(&ot, "id = ?", offenceID).Error; err != nil {
		return nil, err
	}
	if ot.Status != models.OffenceActive {
		return nil, errors.New("offence type is not active")
	}
	var g models.Group
	if err := database.DB.First(&g, "id = ?", groupID).Error; err != nil {
		return nil, err
	}
	amt, ok := ComputeOffenceAmount(&ot, g.FixedContributionAmount)
	if !ok {
		return nil, errors.New("fine amount cannot be determined")
	}
	f := models.Fine{
		GroupID: groupID, MemberID: memberID, OffenceTypeID: offenceID,
		ContributionCycleLabel: cycleLabel, OccurrenceDate: dateOf(due), DueDate: dateOf(due),
		Amount: amt, Reason: reason, Status: models.FineUnpaid,
	}
	if err := database.DB.Create(&f).Error; err != nil {
		// Lost a race — fetch the winner.
		if err2 := database.DB.Where("group_id = ? AND member_id = ? AND offence_type_id = ? AND contribution_cycle_label = ?",
			groupID, memberID, offenceID, cycleLabel).First(&existing).Error; err2 == nil {
			return &existing, nil
		}
		return nil, err
	}
	return &f, nil
}

// CreateEventFine creates one event-based fine idempotently on
// (group, member, offence, occurrence date).
func CreateEventFine(groupID, memberID, offenceID string, occurrence time.Time, reason, reasonNote string) (*models.Fine, error) {
	occ := dateOf(occurrence)
	var existing models.Fine
	if err := database.DB.Where("group_id = ? AND member_id = ? AND offence_type_id = ? AND occurrence_date = ? AND contribution_cycle_label = ''",
		groupID, memberID, offenceID, occ.Format("2006-01-02")).First(&existing).Error; err == nil {
		return &existing, nil
	}
	var ot models.FineOffenceType
	if err := database.DB.First(&ot, "id = ?", offenceID).Error; err != nil {
		return nil, err
	}
	if ot.Status != models.OffenceActive {
		return nil, errors.New("offence type is not active")
	}
	var g models.Group
	if err := database.DB.First(&g, "id = ?", groupID).Error; err != nil {
		return nil, err
	}
	amt, ok := ComputeOffenceAmount(&ot, g.FixedContributionAmount)
	if !ok {
		return nil, errors.New("fine amount cannot be determined")
	}
	f := models.Fine{
		GroupID: groupID, MemberID: memberID, OffenceTypeID: offenceID,
		ContributionCycleLabel: "", OccurrenceDate: occ, DueDate: occ,
		Amount: amt, Reason: reason, Status: models.FineUnpaid,
	}
	if reasonNote != "" {
		f.ReasonNote = &reasonNote
	}
	if err := database.DB.Create(&f).Error; err != nil {
		if err2 := database.DB.Where("group_id = ? AND member_id = ? AND offence_type_id = ? AND occurrence_date = ? AND contribution_cycle_label = ''",
			groupID, memberID, offenceID, occ.Format("2006-01-02")).First(&existing).Error; err2 == nil {
			return &existing, nil
		}
		return nil, err
	}
	return &f, nil
}

// activeOffenceTypes returns ACTIVE offence types of a kind for a group.
func activeOffenceTypes(groupID, kind string) []models.FineOffenceType {
	var ots []models.FineOffenceType
	database.DB.Where("group_id = ? AND kind = ? AND status = ?",
		groupID, kind, models.OffenceActive).Find(&ots)
	return ots
}

// ApplyLateOffenceFines creates lateness fines for the last closed cycle
// past each active offence's grace. Members with any submitted contribution
// in the window are skipped (proof may still be under verification).
func ApplyLateOffenceFines(g *models.Group, now time.Time) (int, error) {
	ots := activeOffenceTypes(g.ID, models.OffenceLateContribution)
	if len(ots) == 0 {
		return 0, nil
	}
	start, due, ok := LastClosedContributionCycle(g, now)
	if !ok {
		return 0, nil
	}
	today := dateOf(now)
	label := ContributionCycleLabel(g.ContributionInterval, due)
	var members []models.Member
	database.DB.Where("deleted_at IS NULL AND is_active = TRUE AND approval_status = 'approved' AND joined_at <= ?",
		due.Format("2006-01-02")).Find(&members)
	created := 0
	for _, ot := range ots {
		if !today.After(dateOf(due).AddDate(0, 0, ot.GracePeriodDays)) {
			continue
		}
		for i := range members {
			if MemberContributedInWindow(members[i].ID, start, due) {
				continue
			}
			f, err := CreateCycleFine(g.ID, members[i].ID, ot.ID, label, due,
				fmt.Sprintf("%s — mzunguko %s", ot.Name, label))
			if err != nil {
				log.Printf("WARN: late fine %s/%s: %v", members[i].MemberNo, label, err)
				continue
			}
			_ = f
			created++
			notifyFineIssued(g.ID, &members[i], &ot, "fine:"+f.ID, label)
		}
	}
	return created, nil
}

func notifyFineIssued(groupID string, m *models.Member, ot *models.FineOffenceType, dedupKey, label string) {
	if m.UserID == nil {
		return
	}
	// In-app + SMS share one guard keyed by the fine itself.
	NotifyUserSMS(groupID, *m.UserID, models.NotifFineIssued,
		"Faini mpya",
		fmt.Sprintf("%s: %s", ot.Name, label),
		dedupKey)
}

// ensureLegacyOffenceType provides the offence row behind the pre-existing
// FineSettings policy so legacy scheduler fines satisfy the NOT NULL
// offence reference. Created once, active, clearly labeled.
func ensureLegacyOffenceType(groupID string) (*models.FineOffenceType, error) {
	var ot models.FineOffenceType
	if err := database.DB.Where("group_id = ? AND kind = ? AND name = ?",
		groupID, models.OffenceLateContribution, "Faini ya kuchelewa (mpangilio wa zamani)").First(&ot).Error; err == nil {
		return &ot, nil
	}
	ot = models.FineOffenceType{
		GroupID: groupID, Kind: models.OffenceLateContribution,
		Name:     "Faini ya kuchelewa (mpangilio wa zamani)",
		FineType: models.FineTypeFixed, Status: models.OffenceActive,
		CreatedBy: "00000000-0000-0000-0000-000000000000",
	}
	var s models.FineSettings
	if err := database.DB.Where("group_id = ?", groupID).First(&s).Error; err == nil {
		ot.FineType = s.FineType
		ot.FineAmount = s.FineAmount
		ot.FinePercentage = s.FinePercentage
		ot.GracePeriodDays = s.GracePeriodDays
	}
	if err := database.DB.Create(&ot).Error; err != nil {
		return nil, err
	}
	return &ot, nil
}

// RunObligationChecks is the scheduler entry: refresh cycles, then create
// lateness fines — via active offence types when any exist, else via the
// legacy FineSettings policy (preserved for backward compatibility).
func RunObligationChecks(g *models.Group, now time.Time) {
	RefreshGroupCycles(g.ID, now)
	if len(activeOffenceTypes(g.ID, models.OffenceLateContribution)) > 0 {
		if n, err := ApplyLateOffenceFines(g, now); err != nil {
			log.Printf("ERROR: Scheduler late fines for group %s failed: %v", g.ID, err)
		} else if n > 0 {
			log.Printf("Scheduler: created %d lateness fines for group %s", n, g.ID)
		}
		return
	}
	if _, err := ApplyFinesForGroup(g, now); err != nil {
		log.Printf("ERROR: Scheduler fines for group %s failed: %v", g.ID, err)
	}
}

// ── Meeting-triggered fines (katibu) ─────────────────────────────────────────

// TriggerMeetingFines creates fines for absent/late members against the
// active meeting-kind offence types. Idempotent per (member, type, date):
// re-running creates nothing new.
func TriggerMeetingFines(groupID, meetingID, triggeredBy string) (int, error) {
	var mtg models.Meeting
	if err := database.DB.First(&mtg, "id = ? AND group_id = ?", meetingID, groupID).Error; err != nil {
		return 0, errors.New("mkutano haujapatikana")
	}
	var rows []models.MeetingAttendance
	database.DB.Where("meeting_id = ?", meetingID).Find(&rows)
	created := 0
	for i := range rows {
		var kinds []string
		switch rows[i].Status {
		case models.AttendanceAbsent:
			kinds = []string{models.OffenceMeetingAbsence}
		case models.AttendanceLate:
			kinds = []string{models.OffenceMeetingLate}
		default:
			continue
		}
		for _, kind := range kinds {
			for _, ot := range activeOffenceTypes(groupID, kind) {
				var m models.Member
				if err := database.DB.First(&m, "id = ?", rows[i].MemberID).Error; err != nil {
					continue
				}
				var existing models.Fine
				if database.DB.Where("group_id = ? AND member_id = ? AND offence_type_id = ? AND occurrence_date = ? AND contribution_cycle_label = ''",
					groupID, rows[i].MemberID, ot.ID, dateOf(mtg.MeetingDate)).First(&existing).Error == nil {
					continue
				}
				ef, err := CreateEventFine(groupID, rows[i].MemberID, ot.ID, mtg.MeetingDate,
					fmt.Sprintf("%s — %s (%s)", ot.Name, mtg.Title, mtg.MeetingDate.Format("2006-01-02")), "")
				if err != nil {
					log.Printf("WARN: meeting fine %s: %v", m.MemberNo, err)
					continue
				}
				created++
				notifyFineIssued(groupID, &m, &ot, "fine:"+ef.ID, mtg.Title)
			}
		}
		database.DB.Model(&rows[i]).Update("fined", true)
	}
	_ = triggeredBy
	return created, nil
}
