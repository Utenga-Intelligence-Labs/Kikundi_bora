package services

import (
	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

// EnsureWelfareObligations creates missing per-member contribution rows for
// already-approved, member-funded events — e.g. when a member is approved
// AFTER the event was approved (late joiners otherwise have no fixed amount
// to pay). The newcomer gets the SAME per-member amount as existing rows so
// nobody else's obligation changes; only if the event has no rows at all is
// the split recomputed. Returns rows created. Idempotent.
func EnsureWelfareObligations(memberID string) (int, error) {
	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", memberID).First(&member).Error; err != nil {
		return 0, err
	}

	var events []models.WelfareEvent
	if err := database.DB.
		Where("status = ? AND funding_source IN ?",
			models.WelfareApproved,
			[]models.WelfareFundingSource{models.FundMemberContribution, models.FundBoth}).
		Find(&events).Error; err != nil {
		return 0, err
	}

	created := 0
	for _, ev := range events {
		var n int64
		database.DB.Model(&models.WelfareContribution{}).
			Where("event_id = ? AND member_id = ?", ev.ID, memberID).Count(&n)
		if n > 0 {
			continue
		}
		amount := decimal.Zero
		var sample models.WelfareContribution
		if err := database.DB.Where("event_id = ?", ev.ID).First(&sample).Error; err == nil {
			amount = sample.Amount
		} else {
			// No rows yet (edge): split across currently eligible members.
			var eligible int64
			database.DB.Model(&models.Member{}).
				Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").
				Count(&eligible)
			if eligible == 0 {
				continue
			}
			total := ev.MemberAmount
			if ev.FundingSource == models.FundMemberContribution && ev.AmountApproved != nil {
				total = *ev.AmountApproved
			}
			if total.IsZero() {
				continue
			}
			amount = total.Div(decimal.NewFromInt(eligible))
		}
		if amount.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if err := database.DB.Create(&models.WelfareContribution{
			EventID: ev.ID, MemberID: memberID, Amount: amount,
			Status: models.WelfareContribPending,
		}).Error; err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
