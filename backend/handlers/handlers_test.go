package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	config.AppConfig = &config.Config{
		DBHost:      "127.0.0.1",
		DBPort:      "5432",
		DBUser:      "dahdio",
		DBPassword:  "devpass123",
		DBName:      "kikundi_db",
		DBSSLMode:   "disable",
		JWTSecret:   "test-secret-key-at-least-32-characters!!",
		Port:        "0",
		CORSOrigins: "http://localhost:3000",
		Environment: "test",
	}
	database.Connect()
	services.InitEmail()

	app := fiber.New(fiber.Config{AppName: "Kikundi API Test"})
	app.Use(middleware.SetupCORS())

	api := app.Group("/api/v1")

	auth := NewAuthHandler()
	api.Post("/auth/login", auth.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	protected.Get("/me", auth.Me)

	memberHandler := NewMemberHandler()
	protected.Get("/members", memberHandler.List)
	protected.Get("/members/:id", memberHandler.Get)

	return app
}

func TestLoginInvalidCredentials(t *testing.T) {
	app := setupTestApp()

	body := map[string]string{"email": "nonexistent@test.com", "password": "wrong"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for invalid credentials, got %d", resp.StatusCode)
	}
}

func TestLoginEmptyBody(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for empty body, got %d", resp.StatusCode)
	}
}

func TestProtectedRouteNoToken(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for protected route without token, got %d", resp.StatusCode)
	}
}

func TestInvalidTokenFormat(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for invalid token, got %d", resp.StatusCode)
	}
}

func TestMemberListRequiresAuth(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/members", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for member list without auth, got %d", resp.StatusCode)
	}
}
