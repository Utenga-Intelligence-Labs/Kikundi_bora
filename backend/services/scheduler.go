package services

import (
	"log"
	"time"
)

// StartScheduler launches background periodic jobs. Currently:
//   - contribution due-date reminders/checks (every 30 minutes; the
//     notification logic itself is idempotent per cycle per kind, so
//     frequent runs never duplicate notifications).
//
// Deliberately dependency-free (time.Ticker instead of robfig/cron) — the
// only schedule needed is "check periodically, decide by date".
func StartScheduler() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ERROR: Scheduler panicked, restarting: %v", r)
				StartScheduler()
			}
		}()

		log.Println("Scheduler started (contribution due-date checks every 30m)")
		// Run once shortly after startup so a due date on the deployment day
		// is not missed, then keep ticking.
		time.Sleep(10 * time.Second)
		RunContributionDueCheck()

		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			RunContributionDueCheck()
		}
	}()
}
