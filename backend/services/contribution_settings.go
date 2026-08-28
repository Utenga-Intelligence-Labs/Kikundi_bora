package services

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

// ---------- Pure validation / computation helpers (unit-testable) ----------

// ValidateProposalSpec validates an interval + due-date pair for a settings
// proposal. Amount (if given) must be positive.
func ValidateProposalSpec(interval, dueDate string, amount *decimal.Decimal) error {
	if !models.IsValidContributionInterval(interval) {
		return fmt.Errorf("kipindi si sahihi. Chagua: weekly, monthly, semi_annual, au yearly")
	}
	if err := validateDueDateSpec(models.ContributionInterval(interval), dueDate); err != nil {
		return err
	}
	if amount != nil && amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("kiasi lazima kiwe zaidi ya sifuri")
	}
	return nil
}

// validateDueDateSpec checks the due-date format for the interval:
//   weekly: "1".."7" (Mon..Sun), monthly: "1".."31", semi_annual/yearly: "MM-DD"
func validateDueDateSpec(interval models.ContributionInterval, spec string) error {
	if spec == "" {
		return fmt.Errorf("tarehe ya mchango inahitajika")
	}
	switch interval {
	case models.IntervalWeekly:
		n, err := strconv.Atoi(spec)
		if err != nil || n < 1 || n > 7 {
			return fmt.Errorf("kwa weekly, due date lazima iwe siku ya wiki 1-7 (1=Jumatatu, 7=Jumapili)")
		}
	case models.IntervalMonthly:
		n, err := strconv.Atoi(spec)
		if err != nil || n < 1 || n > 31 {
			return fmt.Errorf("kwa monthly, due date lazima iwe siku ya mwezi 1-31")
		}
	case models.IntervalSemiAnnual, models.IntervalYearly:
		parts := strings.Split(spec, "-")
		if len(parts) != 2 {
			return fmt.Errorf("due date lazima iwe MM-DD (mfano 03-15)")
		}
		mm, err1 := strconv.Atoi(parts[0])
		dd, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || mm < 1 || mm > 12 || dd < 1 || dd > 31 {
			return fmt.Errorf("due date lazima iwe MM-DD sahihi (mfano 03-15)")
		}
	default:
		return fmt.Errorf("kipindi si sahihi")
	}
	return nil
}

// dateOf truncates a time to midnight in its own location.
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// monthDayClamped builds a date in year/month with day clamped to the
// month's length (e.g. day 31 in April → April 30).
func monthDayClamped(year int, month time.Month, day int, loc *time.Location) time.Time {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// NextContributionDueDate computes the next due date on or after `from`
// for the given interval + due-date spec.
func NextContributionDueDate(interval models.ContributionInterval, spec string, from time.Time) (time.Time, bool) {
	if spec == "" {
		return time.Time{}, false
	}
	today := dateOf(from)
	switch interval {
	case models.IntervalWeekly:
		n, err := strconv.Atoi(spec)
		if err != nil || n < 1 || n > 7 {
			return time.Time{}, false
		}
		target := time.Weekday(n % 7) // 1=Mon .. 6=Sat, 7=Sun(0)
		offset := (int(target) - int(today.Weekday()) + 7) % 7
		return today.AddDate(0, 0, offset), true
	case models.IntervalMonthly:
		day, err := strconv.Atoi(spec)
		if err != nil || day < 1 || day > 31 {
			return time.Time{}, false
		}
		y, m, _ := today.Date()
		cand := monthDayClamped(y, m, day, today.Location())
		if cand.Before(today) {
			cand = monthDayClamped(y, m+1, day, today.Location())
		}
		return cand, true
	case models.IntervalSemiAnnual, models.IntervalYearly:
		parts := strings.Split(spec, "-")
		if len(parts) != 2 {
			return time.Time{}, false
		}
		mm, err1 := strconv.Atoi(parts[0])
		dd, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || mm < 1 || mm > 12 || dd < 1 || dd > 31 {
			return time.Time{}, false
		}
		y, _, _ := today.Date()
		cand := monthDayClamped(y, time.Month(mm), dd, today.Location())
		for cand.Before(today) {
			if interval == models.IntervalSemiAnnual {
				cand = cand.AddDate(0, 6, 0)
			} else {
				cand = cand.AddDate(1, 0, 0)
			}
		}
		return cand, true
	}
	return time.Time{}, false
}

// CheckFixedContributionAmount enforces the group's fixed contribution
// amount. nil fixed = no enforcement. Returns a user-facing error when the
// amount does not match exactly.
func CheckFixedContributionAmount(fixed *decimal.Decimal, amount decimal.Decimal) error {
	if fixed == nil {
		return nil
	}
	if !amount.Equal(*fixed) {
		return fmt.Errorf(
			"Kiasi si sahihi. Mchango wa kikundi ni TZS %s kwa mujibu wa mipangilio iliyoidhinishwa",
			fixed.StringFixed(2),
		)
	}
	return nil
}

// ---------- Due-date notifications ----------

// ContributionDueStatus decides whether a due-date notification should go
// out today for the group. kind is "reminder" (2 days before) or "due".
// Idempotency: a given (kind, day) is only reported once per group — the
// group records the last notified day per kind.
func ContributionDueStatus(g *models.Group, today time.Time) (kind string, dueDate time.Time, ok bool) {
	if g == nil || g.ContributionDueDate == nil || g.ContributionInterval == "" {
		return "", time.Time{}, false
	}
	due, valid := NextContributionDueDate(g.ContributionInterval, *g.ContributionDueDate, today)
	if !valid {
		return "", time.Time{}, false
	}
	todayD := dateOf(today)

	if due.Equal(todayD) {
		if g.LastDueNotifiedFor != nil && dateOf(*g.LastDueNotifiedFor).Equal(todayD) {
			return "", time.Time{}, false // already notified for this cycle's due date
		}
		return "due", due, true
	}
	reminder := due.AddDate(0, 0, -2)
	if reminder.Equal(todayD) {
		if g.LastReminderNotifiedFor != nil && dateOf(*g.LastReminderNotifiedFor).Equal(todayD) {
			return "", time.Time{}, false // already reminded for this cycle
		}
		return "reminder", due, true
	}
	return "", time.Time{}, false
}

// SendContributionDueNotifications sends the due-date (or reminder)
// notification to ALL active members of the group, once per cycle per kind.
// Returns whether a notification was sent.
func SendContributionDueNotifications(g *models.Group, today time.Time) (sent bool, kind string, err error) {
	k, due, ok := ContributionDueStatus(g, today)
	if !ok {
		return false, "", nil
	}

	var members []models.Member
	if err := database.DB.Where("is_active = TRUE AND deleted_at IS NULL").Find(&members).Error; err != nil {
		return false, "", err
	}

	userIDs := make([]string, 0, len(members))
	for _, m := range members {
		uid := ""
		if m.UserID != nil {
			uid = *m.UserID
		}
		if uid == "" {
			uid = m.RegisteredBy
		}
		if uid != "" {
			userIDs = append(userIDs, uid)
		}
	}

	amountPart := ""
	if g.FixedContributionAmount != nil {
		amountPart = " wa TZS " + g.FixedContributionAmount.StringFixed(2)
	}
	when := due.Format("02 Jan 2006")
	var title, msg string
	if k == "reminder" {
		title = "Kumbusho la Mchango"
		msg = fmt.Sprintf("Mchango%s unatarajiwa ifikapo %s. Tafadhali andaa malipo yako.", amountPart, when)
	} else {
		title = "Siku ya Mchango"
		msg = fmt.Sprintf("Leo %s ni siku ya mchango%s. Tafadhali wasilisha mchango wako.", when, amountPart)
	}
	NotifyUsers(userIDs, models.NotifContributionDue, title, msg)

	// Record idempotency marker
	now := dateOf(today)
	if k == "due" {
		g.LastDueNotifiedFor = &now
	} else {
		g.LastReminderNotifiedFor = &now
	}
	if err := database.DB.Save(g).Error; err != nil {
		return false, "", err
	}
	log.Printf("Scheduler: %s notification sent to %d members of group %s (due %s)", k, len(userIDs), g.ID, when)
	return true, k, nil
}

// RunContributionDueCheck iterates all groups and sends due-date
// notifications where today qualifies. Called by the scheduler.
func RunContributionDueCheck() {
	var groups []models.Group
	if err := database.DB.Find(&groups).Error; err != nil {
		log.Printf("ERROR: Scheduler could not load groups: %v", err)
		return
	}
	now := time.Now()
	for i := range groups {
		if _, _, ok := ContributionDueStatus(&groups[i], now); ok {
			if _, _, err := SendContributionDueNotifications(&groups[i], now); err != nil {
				log.Printf("ERROR: Scheduler notification for group %s failed: %v", groups[i].ID, err)
			}
		}
	}
}
