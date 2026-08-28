package middleware

import (
	"net/http/httptest"
	"testing"

	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

// newRoleTestApp builds a fiber app that injects a role into locals (as
// AuthRequired would after verifying the JWT against the DB), then guards
// the test route with RequireRoles.
func newRoleTestApp(role models.Role) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", role)
		return c.Next()
	})
	app.Post("/propose", RequireRoles(models.RoleChair), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	app.Post("/approve", RequireRoles(models.RoleSecretary), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	return app
}

func testRequest(t *testing.T, app *fiber.App, method, path string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return res.StatusCode
}

// Contribution-settings permission matrix:
//   - propose: mwenyekiti (chair) only
//   - approve/reject: katibu (secretary) only
//   - member: neither
func TestRequireRolesContributionSettingsPermissions(t *testing.T) {
	cases := []struct {
		role        models.Role
		proposeCode int
		approveCode int
	}{
		{models.RoleChair, 200, 403},
		{models.RoleSecretary, 403, 200},
		{models.RoleMember, 403, 403},
		{models.RoleTreasurer, 403, 403},
		{models.RoleAdmin, 200, 200}, // admin bypasses
	}
	for _, tc := range cases {
		app := newRoleTestApp(tc.role)
		if got := testRequest(t, app, "POST", "/propose"); got != tc.proposeCode {
			t.Errorf("role=%s propose: got %d, want %d", tc.role, got, tc.proposeCode)
		}
		if got := testRequest(t, app, "POST", "/approve"); got != tc.approveCode {
			t.Errorf("role=%s approve: got %d, want %d", tc.role, got, tc.approveCode)
		}
	}
}
