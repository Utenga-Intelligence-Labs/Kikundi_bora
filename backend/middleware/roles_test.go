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
	// Payment methods: mwenyekiti + mweka hazina manage, members read-only
	app.Post("/payment-methods", RequireRoles(models.RoleChair, models.RoleTreasurer), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	app.Patch("/payment-methods/:pmId", RequireRoles(models.RoleChair, models.RoleTreasurer), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	app.Delete("/payment-methods/:pmId", RequireRoles(models.RoleChair, models.RoleTreasurer), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	// Social funds: mwenyekiti creates/closes, katibu approves, hazina confirms
	app.Post("/social-funds", RequireRoles(models.RoleChair), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	app.Post("/social-funds/:id/approve", RequireRoles(models.RoleSecretary), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	app.Post("/social-funds/:id/confirm", RequireRoles(models.RoleTreasurer), func(c *fiber.Ctx) error {
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
//
// Payment-methods permission matrix:
//   - create/edit/delete: mwenyekiti (chair) + mweka hazina (treasurer) only
//   - member: read-only (all writes 403)
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

func TestRequireRolesPaymentMethodPermissions(t *testing.T) {
	cases := []struct {
		role   models.Role
		create int
		update int
		delete int
	}{
		{models.RoleChair, 200, 200, 200},
		{models.RoleTreasurer, 200, 200, 200},
		{models.RoleMember, 403, 403, 403},
		{models.RoleSecretary, 403, 403, 403},
		{models.RoleAdmin, 200, 200, 200}, // admin bypasses
	}
	for _, tc := range cases {
		app := newRoleTestApp(tc.role)
		if got := testRequest(t, app, "POST", "/payment-methods"); got != tc.create {
			t.Errorf("role=%s create payment method: got %d, want %d", tc.role, got, tc.create)
		}
		if got := testRequest(t, app, "PATCH", "/payment-methods/pm-1"); got != tc.update {
			t.Errorf("role=%s update payment method: got %d, want %d", tc.role, got, tc.update)
		}
		if got := testRequest(t, app, "DELETE", "/payment-methods/pm-1"); got != tc.delete {
			t.Errorf("role=%s delete payment method: got %d, want %d", tc.role, got, tc.delete)
		}
	}
}

// Social-fund permission matrix:
//   - create/close fund: mwenyekiti (chair) only
//   - approve fund: katibu (secretary) only
//   - confirm contribution: mweka hazina (treasurer) only
//   - member: none of the above (contribute endpoint has no role guard —
//     any authenticated member contributes; enforced via member row check)
func TestRequireRolesSocialFundPermissions(t *testing.T) {
	cases := []struct {
		role    models.Role
		create  int
		approve int
		confirm int
	}{
		{models.RoleChair, 200, 403, 403},
		{models.RoleSecretary, 403, 200, 403},
		{models.RoleTreasurer, 403, 403, 200},
		{models.RoleMember, 403, 403, 403},
		{models.RoleAdmin, 200, 200, 200}, // admin bypasses
	}
	for _, tc := range cases {
		app := newRoleTestApp(tc.role)
		if got := testRequest(t, app, "POST", "/social-funds"); got != tc.create {
			t.Errorf("role=%s create social fund: got %d, want %d", tc.role, got, tc.create)
		}
		if got := testRequest(t, app, "POST", "/social-funds/f-1/approve"); got != tc.approve {
			t.Errorf("role=%s approve social fund: got %d, want %d", tc.role, got, tc.approve)
		}
		if got := testRequest(t, app, "POST", "/social-funds/f-1/confirm"); got != tc.confirm {
			t.Errorf("role=%s confirm contribution: got %d, want %d", tc.role, got, tc.confirm)
		}
	}
}
