package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

func historyTestApp() *fiber.App {
	app := fullApp()
	memberContribHandler := NewMemberContributionHandler()
	g := app.Group("/api/v1/michango", middleware.AuthRequired)
	g.Get("/mine", memberContribHandler.MyContributions)
	return app
}

// Regression: treasurer-recorded contributions MUST appear in the member's
// own history fetch. MyContributions used to read only the self-submitted
// member_contributions table, so /michango-yangu and /historia-yangu showed
// "empty" for members whose payments were all receipted by the treasurer.
func TestMyContributionsIncludesTreasurerRows(t *testing.T) {
	app := historyTestApp()
	cleanupTestData()
	reseedIfEmpty()

	memberTok := doLogin(t, app, "asha@kikundi.tz", "demo123")

	var member models.Member
	database.DB.Where("user_id = (SELECT id FROM users WHERE email = ?)", "asha@kikundi.tz").First(&member)
	if member.ID == "" {
		t.Fatalf("no member linked to asha")
	}

	// Treasurer-recorded row (the store MyContributions used to ignore).
	month := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	database.DB.Create(&models.Contribution{
		MemberID: member.ID, RecordedBy: member.RegisteredBy,
		Amount: decimal.NewFromInt(10000), Month: month,
		PaidAt: month, PaymentMethod: "CASH", Status: "PAID",
	})

	code, body := doRequest(t, app, "GET", "/api/v1/michango/mine", nil, memberTok)
	if code != 200 {
		t.Fatalf("mine = %d %s, want 200", code, body)
	}
	var env struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Total == 0 {
		t.Fatalf("member history empty despite treasurer-recorded row: %s", body)
	}
	var rows []struct {
		PeriodLabel string `json:"period_label"`
		Amount      string `json:"amount"`
		Status      string `json:"status"`
		Source      string `json:"source"`
	}
	decodeData(t, body, &rows)
	found := false
	for _, r := range rows {
		if r.PeriodLabel == "2026-08" && r.Source == "treasurer" && r.Status == "CONFIRMED" {
			found = true
		}
	}
	if !found {
		t.Fatalf("treasurer row missing from history: %+v", rows)
	}
}
