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
	// Loan portfolio: leadership read-only view
	app.Get("/loans/portfolio", RequireRoles(models.RoleChair, models.RoleSecretary, models.RoleTreasurer), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	// "Pokea Michango" (all-contributions listing): treasurer + secretary
	// only — mwenyekiti removed from receipting.
	app.Get("/michango", RequireRoles(models.RoleTreasurer, models.RoleSecretary), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	// Member approval workflow: katibu (secretary) only
	app.Patch("/members/:id/approve", RequireRoles(models.RoleSecretary), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})
	app.Patch("/members/:id/reject", RequireRoles(models.RoleSecretary), func(c *fiber.Ctx) error {
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

func TestRequireRolesSocialFundPermissions(t *testing.T) {
	t.Skip("social funds feature removed — welfare events (/mfuko-kijamii) is the single Mfuko wa Kijamii feature")
}

// Member approval workflow: katibu (secretary) approves/rejects —
// mwenyekiti, mweka hazina na mwanachama hawana ruhusa.
func TestRequireRolesMemberApprovalPermissions(t *testing.T) {
	cases := []struct {
		role    models.Role
		approve int
		reject  int
	}{
		{models.RoleChair, 403, 403},
		{models.RoleSecretary, 200, 200},
		{models.RoleTreasurer, 403, 403},
		{models.RoleMember, 403, 403},
		{models.RoleAdmin, 200, 200}, // admin bypasses
	}
	for _, tc := range cases {
		app := newRoleTestApp(tc.role)
		if got := testRequest(t, app, "PATCH", "/members/m-1/approve"); got != tc.approve {
			t.Errorf("role=%s member approve: got %d, want %d", tc.role, got, tc.approve)
		}
		if got := testRequest(t, app, "PATCH", "/members/m-1/reject"); got != tc.reject {
			t.Errorf("role=%s member reject: got %d, want %d", tc.role, got, tc.reject)
		}
	}
}

// Loan portfolio: mwenyekiti/katibu/mweka hazina can view; member cannot.
func TestRequireRolesLoanPortfolioPermissions(t *testing.T) {
	cases := []struct {
		role models.Role
		code int
	}{
		{models.RoleChair, 200},
		{models.RoleSecretary, 200},
		{models.RoleTreasurer, 200},
		{models.RoleMember, 403},
		{models.RoleAdmin, 200},
	}
	for _, tc := range cases {
		app := newRoleTestApp(tc.role)
		if got := testRequest(t, app, "GET", "/loans/portfolio"); got != tc.code {
			t.Errorf("role=%s portfolio: got %d, want %d", tc.role, got, tc.code)
		}
	}
}

// "Pokea Michango" (all-contributions listing): mweka hazina + katibu only.
// KEY ASSERTION: mwenyekiti gets 403 hitting the endpoint directly.
func TestRequireRolesPokeaMichangoPermissions(t *testing.T) {
	cases := []struct {
		role models.Role
		code int
	}{
		{models.RoleChair, 403}, // removed from receipting
		{models.RoleTreasurer, 200},
		{models.RoleSecretary, 200}, // katibu keeps records view
		{models.RoleMember, 403},
		{models.RoleAdmin, 200},
	}
	for _, tc := range cases {
		app := newRoleTestApp(tc.role)
		if got := testRequest(t, app, "GET", "/michango"); got != tc.code {
			t.Errorf("role=%s pokea michango: got %d, want %d", tc.role, got, tc.code)
		}
	}
}
