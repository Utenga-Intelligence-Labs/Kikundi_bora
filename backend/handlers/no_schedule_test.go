package handlers

import (
	"testing"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"
	"kikundibora/services"
)

// Regression test for the scheduler WARN spam ("cycle refresh KKK-XXXX:
// group has no contribution schedule" every 30 min per member). With no
// approved contribution schedule, group refresh must create zero cycle rows
// and member summaries must return a fines-only result without error —
// instead of failing once per member.
func TestNoScheduleSkipsCycleRefresh(t *testing.T) {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	cleanAndSeed(t)
	cleanObligationTables(t)

	var g models.Group
	if err := database.DB.First(&g).Error; err != nil {
		t.Fatalf("group: %v", err)
	}
	// Mimic the live pre-configuration state: interval set, but no due
	// date and no fixed amount (nothing ever proposed/approved).
	database.DB.Model(&g).Updates(map[string]interface{}{
		"contribution_due_date": nil, "fixed_contribution_amount": nil,
	})

	services.RefreshGroupCycles(g.ID, time.Now())

	var n int64
	database.DB.Model(&models.ContributionCycle{}).Count(&n)
	if n != 0 {
		t.Errorf("cycle rows with no schedule = %d, want 0", n)
	}

	var member models.Member
	if err := database.DB.Where("deleted_at IS NULL").First(&member).Error; err != nil {
		t.Fatalf("member: %v", err)
	}
	out, err := services.GetMemberObligations(g.ID, member.ID, time.Now())
	if err != nil {
		t.Fatalf("GetMemberObligations with no schedule: %v", err)
	}
	if out.HasSchedule {
		t.Errorf("HasSchedule = true, want false")
	}
	if !out.GrandTotal.IsZero() {
		t.Errorf("GrandTotal = %s, want 0 (fines-only)", out.GrandTotal)
	}

	// Sanity: once a schedule IS approved, refresh tracks cycles again.
	g2 := seedObligationGroup(t)
	services.RefreshGroupCycles(g2.ID, time.Now())
	database.DB.Model(&models.ContributionCycle{}).Count(&n)
	if n == 0 {
		t.Errorf("cycle rows with schedule = 0, want > 0")
	}
}
