package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

func minimalApp() *fiber.App {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	services.InitEmail()

	app := fiber.New(fiber.Config{AppName: "Kikundi API Test"})
	app.Use(middleware.SetupCORS())

	api := app.Group("/api/v1")
	authHandler := NewAuthHandler()

	api.Post("/auth/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	members := protected.Group("/members")
	memberHandler := NewMemberHandler()
	members.Get("/", memberHandler.List)

	welfareGroup := protected.Group("/welfare")
	welfareHandler := NewWelfareHandler()
	welfareGroup.Post("/events", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.CreateEvent)
	welfareGroup.Post("/events/:id/approve", middleware.RequireRoles(models.RoleChair), welfareHandler.ApproveEvent)

	return app
}

func decodeID(data []byte, key string) string {
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	nested := m[key].(map[string]interface{})
	return nested["id"].(string)
}

func login(app *fiber.App, email, pass string) string {
	b, _ := json.Marshal(map[string]string{"email": email, "password": pass})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var r struct{ Token string }
	json.Unmarshal(data, &r)
	return r.Token
}

func post(app *fiber.App, path string, body interface{}, token string) (int, []byte) {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest("POST", path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, _ := app.Test(req)
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, data
}

func get(app *fiber.App, path string, token string) (int, []byte) {
	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, _ := app.Test(req)
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, data
}

func TestFiberUUIDBug(t *testing.T) {
	app := minimalApp()

	// Clean state
	database.DB.Exec("DELETE FROM welfare_contributions")
	database.DB.Exec("DELETE FROM welfare_events")
	database.DB.Exec("DELETE FROM user_sessions")
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		database.Seed()
	}

	treasurerToken := login(app, "fatuma@kikundi.tz", "demo123")
	chairToken := login(app, "juma@kikundi.tz", "demo123")

	// Get member ID
	_, memberData := get(app, "/api/v1/members", chairToken)
	var mList struct{ Data []struct{ ID string `json:"id"` } `json:"data"` }
	json.Unmarshal(memberData, &mList)
	if len(mList.Data) == 0 {
		t.Fatal("no members")
	}
	memberID := mList.Data[0].ID

	// Create welfare event
	eventBody := map[string]interface{}{
		"member_id":        memberID,
		"event_type":       "MATIBABU",
		"description":      "Test",
		"amount_requested": 10000,
		"funding_source":   "TREASURY",
		"treasury_amount":  10000,
		"member_amount":    0,
	}
	code, data := post(app, "/api/v1/welfare/events", eventBody, treasurerToken)
	if code != 201 {
		t.Fatalf("create event: %d %s", code, data)
	}
	eventID := decodeID(data, "data")
	t.Logf("Created event: %s", eventID)

	// Verify via direct GORM query
	var ev models.WelfareEvent
	err := database.DB.First(&ev, "id = ?", eventID).Error
	if err != nil {
		t.Fatalf("GORM First by column: %v", err)
	}
	t.Logf("GORM by column: found id=%s status=%s", ev.ID, ev.Status)

	// Test tx.First by primary key
	tx := database.DB.Begin()
	var ev2 models.WelfareEvent
	err = tx.First(&ev2, eventID).Error
	t.Logf("tx.First(&ev2, %q): err=%v", eventID, err)
	if err != nil {
		t.Logf("  DETAIL: tx.First with pk form FAILED")
	} else {
		t.Logf("  DETAIL: tx.First with pk form: found id=%s", ev2.ID)
	}
	tx.Rollback()

	// Test tx.First by column
	tx2 := database.DB.Begin()
	var ev3 models.WelfareEvent
	err = tx2.First(&ev3, "id = ?", eventID).Error
	t.Logf("tx.First(&ev3, \"id = ?\", %q): err=%v", eventID, err)
	tx2.Rollback()

	// Now try the HTTP approve
	approveBody := map[string]interface{}{"approved_amount": 10000}
	code, data = post(app, "/api/v1/welfare/events/"+eventID+"/approve", approveBody, chairToken)
	t.Logf("HTTP approve: %d %s", code, data)

	// Try direct GORM update equivalent
	var ev4 models.WelfareEvent
	err = database.DB.First(&ev4, eventID).Error
	t.Logf("db.First(&ev4, %q): err=%v", eventID, err)
}
