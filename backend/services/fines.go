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

// ValidateFineSettingsSpec checks a proposed fines policy.
func ValidateFineSettingsSpec(enabled bool, fineType string, amount, pct *decimal.Decimal, graceDays int) error {
	if !models.IsValidFineType(fineType) {
		return fmt.Errorf("aina ya faini si sahihi. Chagua: fixed au percentage")
	}
	if graceDays < 0 {
		return fmt.Errorf("siku za neema haziwezi kuwa hasi")
	}
	if !enabled {
		return nil
	}
	switch fineType {
	case models.FineTypeFixed:
		if amount == nil || amount.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("kiasi cha faini kinahitajika na lazima kiwe zaidi ya sifuri")
		}
	case models.FineTypePercentage:
		if pct == nil || pct.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("asilimia ya faini inahitajika na lazima iwe zaidi ya sifuri")
		}
		if pct.GreaterThan(decimal.NewFromInt(100)) {
			return fmt.Errorf("asilimia ya faini haiwezi kuzidi 100")
		}
	}
	return nil
}

// ComputeFineAmount returns the TZS amount for a missed cycle given approved
// settings and the group's fixed contribution (used for percentage type).
func ComputeFineAmount(settings *models.FineSettings, fixedContribution *decimal.Decimal) (decimal.Decimal, bool) {
	if settings == nil {
		return decimal.Zero, false
	}
	switch settings.FineType {
	case models.FineTypeFixed:
		if settings.FineAmount == nil || settings.FineAmount.LessThanOrEqual(decimal.Zero) {
			return decimal.Zero, false
		}
		return *settings.FineAmount, true
	case models.FineTypePercentage:
		if settings.FinePercentage == nil || fixedContribution == nil {
			return decimal.Zero, false
		}
		if settings.FinePercentage.LessThanOrEqual(decimal.Zero) || fixedContribution.LessThanOrEqual(decimal.Zero) {
			return decimal.Zero, false
		}
		amt := fixedContribution.Mul(*settings.FinePercentage).Div(decimal.NewFromInt(100)).Round(2)
		if amt.LessThanOrEqual(decimal.Zero) {
			return decimal.Zero, false
		}
		return amt, true
	}
	return decimal.Zero, false
}

// ApplyFinesForGroup creates unpaid fines for members who missed the last
// closed cycle after the grace period. Idempotent per (group, member, cycle)
// via a pre-insert existence check and the unique constraint.
func ApplyFinesForGroup(g *models.Group, now time.Time) (created int, err error) {
	if g == nil {
		return 0, nil
	}
	settings, err := database.GetOrCreateFineSettings(g.ID)
	if err != nil || settings == nil || !settings.Enabled {
		return 0, err
	}

	start, due, ok := LastClosedContributionCycle(g, now)
	if !ok {
		return 0, nil
	}
	today := dateOf(now)
	graceEnd := dateOf(due).AddDate(0, 0, settings.GracePeriodDays)
	if !today.After(graceEnd) {
		return 0, nil
	}

	amount, ok := ComputeFineAmount(settings, g.FixedContributionAmount)
	if !ok {
		return 0, nil
	}

	label := ContributionCycleLabel(g.ContributionInterval, due)
	reason := fmt.Sprintf(
		"Mchango wa kipindi %s haukuwasilishwa baada ya tarehe %s (siku %d za neema).",
		label, due.Format("02 Jan 2006"), settings.GracePeriodDays,
	)

	var members []models.Member
	if err := database.DB.
		Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved' AND joined_at <= ?", due).
		Find(&members).Error; err != nil {
		return 0, err
	}

	for _, m := range members {
		if MemberContributedInWindow(m.ID, start, due) {
			continue
		}
		var existing int64
		database.DB.Model(&models.Fine{}).
			Where("group_id = ? AND member_id = ? AND contribution_cycle_label = ?", g.ID, m.ID, label).
			Count(&existing)
		if existing > 0 {
			continue
		}

		// Legacy policy rows ride on the hidden legacy offence type so the
		// NOT NULL offence reference (and the new idempotency keys) hold.
		legacyOT, lotErr := ensureLegacyOffenceType(g.ID)
		if lotErr != nil {
			log.Printf("ERROR: legacy offence type for group %s: %v", g.ID, lotErr)
			continue
		}
		fine := models.Fine{
			GroupID:                g.ID,
			MemberID:               m.ID,
			OffenceTypeID:          legacyOT.ID,
			ContributionCycleLabel: label,
			OccurrenceDate:         dateOf(due),
			DueDate:                dateOf(due),
			Amount:                 amount,
			Reason:                 reason,
			Status:                 models.FineUnpaid,
		}
		if err := database.DB.Create(&fine).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				continue
			}
			log.Printf("ERROR: Fine create for member %s cycle %s: %v", m.ID, label, err)
			continue
		}
		created++
		if m.UserID != nil && *m.UserID != "" {
			NotifyUserSMS(g.ID, *m.UserID, models.NotifSystem, "Faini ya mchango",
				fmt.Sprintf("Umepata faini ya TZS %s kwa kuchelewa kuchangia kipindi %s.", amount.StringFixed(2), label),
				"fine:"+fine.ID)
		}
	}
	if created > 0 {
		log.Printf("Scheduler: created %d fine(s) for group %s cycle %s", created, g.ID, label)
	}
	return created, nil
}

// RunFineCheck iterates groups and applies late-contribution fines. Called
// from the same scheduler tick as due-date notifications.
func RunFineCheck() {
	var groups []models.Group
	if err := database.DB.Find(&groups).Error; err != nil {
		log.Printf("ERROR: Scheduler could not load groups for fines: %v", err)
		return
	}
	now := time.Now()
	for i := range groups {
		if _, err := ApplyFinesForGroup(&groups[i], now); err != nil {
			log.Printf("ERROR: Scheduler fines for group %s failed: %v", groups[i].ID, err)
		}
	}
}
