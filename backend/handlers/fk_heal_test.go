package handlers

import (
	"encoding/json"
	"testing"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"
)

// Regression test for the broken fk_members_approver: with the constraint
// wrongly pointing at members(id), katibu approval of any member fails with
// an FK violation (a user UUID never exists in members.id). AutoMigrate must
// heal it to users(id), after which approval succeeds end-to-end.
func TestApproverFKHealsAndApprovalWorks(t *testing.T) {
	config.AppConfig = testConfig()
	database.Connect()
	database.AutoMigrate()
	cleanAndSeed(t)

	// Sabotage: drop whatever is there and recreate the reported-broken
	// definition (approved_by → members(id)).
	database.DB.Exec(`ALTER TABLE "members" DROP CONSTRAINT IF EXISTS "fk_members_approver"`)
	if err := database.DB.Exec(`ALTER TABLE "members" ADD CONSTRAINT "fk_members_approver" FOREIGN KEY ("approved_by") REFERENCES "members" ("id")`).Error; err != nil {
		t.Fatalf("sabotage setup: %v", err)
	}

	// Pending member fixture (katibu-approved user flow needs a user row).
	u, m := makePendingFixture(t, "0700000999", "KKK-T99", models.MemberApprovalPending)
	_ = u

	var katibu models.User
	if err := database.DB.Where("role = ? AND deleted_at IS NULL", models.RoleSecretary).First(&katibu).Error; err != nil {
		t.Fatalf("katibu: %v", err)
	}

	// Reproduce: approval write with a USER uuid must violate the wrong FK.
	broken := database.DB.Model(&models.Member{}).Where("id = ?", m.ID).Update("approved_by", katibu.ID).Error
	if broken == nil {
		t.Fatalf("expected FK violation with mis-pointed fk_members_approver, got nil")
	}

	// Heal via the real boot path.
	database.AutoMigrate()

	def, found := database.ExistingFKDef("members", "fk_members_approver")
	if !found {
		t.Fatalf("fk_members_approver missing after heal")
	}
	if def.RefTable != "users" || def.Column != "approved_by" || def.RefColumn != "id" {
		t.Fatalf("fk_members_approver still wrong after heal: %s", def.Describe())
	}

	// Approval write now succeeds.
	if err := database.DB.Model(&models.Member{}).Where("id = ?", m.ID).Update("approved_by", katibu.ID).Error; err != nil {
		t.Fatalf("approval write after heal: %v", err)
	}

	// Full HTTP approve flow as katibu (secretary) succeeds end-to-end.
	app := gatingTestApp()
	code, body := gatingLogin(t, app, "rashidi@kikundi.tz", "demo123")
	if code != 200 {
		t.Fatalf("katibu login: %d %s", code, body)
	}
	var lr struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &lr); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	code, body = hReq(t, app, "PATCH", "/api/v1/members/"+m.ID+"/approve", nil, lr.Token)
	if code != 200 {
		t.Fatalf("approve member: want 200, got %d (%s)", code, body)
	}
	var after models.Member
	database.DB.First(&after, "id = ?", m.ID)
	if after.ApprovalStatus != models.MemberApprovalApproved {
		t.Fatalf("member status = %q, want approved", after.ApprovalStatus)
	}
}
