package database

import (
	"log"

	"kikundibora/models"
)

func AutoMigrate() {
	// Enable pgcrypto extension for gen_random_uuid()
	log.Println("Migration: enabling pgcrypto extension...")
	DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto")

	// First pass: create all tables without foreign key constraints (safe — won't drop existing).
	log.Println("Migration: creating/migrating tables (pass 1 — structure only)...")
	DB.Config.DisableForeignKeyConstraintWhenMigrating = true
	err := DB.AutoMigrate(
		// Core auth / user tables
		&models.User{},
		&models.UserSession{},
		&models.FailedLogin{},
		&models.UserApproval{},
		&models.AdminLog{},
		&models.UserPosition{},
		&models.LeadershipPosition{},
		&models.MemberContribution{},
		&models.PendingAction{},
		&models.Group{},
		&models.GroupSettingProposal{},
		&models.FineSettings{},
		&models.Fine{},
		&models.PaymentMethod{},

		// Member & financial tables
		&models.Member{},
		&models.Contribution{},
		&models.ContributionEdit{},
		&models.Loan{},
		&models.Repayment{},

		// Loan committee tables
		&models.LoanCommitteeMember{},
		&models.LoanReview{},

		// Welfare (Mfuko wa Kijamii) tables
		&models.WelfareEvent{},
		&models.WelfareContribution{},

		// System tables
		&models.AuditLog{},
		&models.Notification{},

		// Backup tables
		&models.BackupHistory{},
		&models.BackupSettings{},
	)
	if err != nil {
		panic("Failed to auto-migrate (pass 1): " + err.Error())
	}
	DB.Config.DisableForeignKeyConstraintWhenMigrating = false

	// Second pass: add foreign key constraints.
	log.Println("Migration: adding foreign key constraints (pass 2)...")
	addFKConstraints()

	// Backfill: payment methods created before the approval workflow
	// carry an empty status — treat them as approved (they were live).
	DB.Model(&models.PaymentMethod{}).
		Where("status = '' OR status IS NULL").
		Update("status", models.PaymentMethodApproved)

	log.Println("Migration complete.")
}

// addFKConstraints creates FK constraints that GORM may skip when
// DisableForeignKeyConstraintWhenMigrating is used.
func addFKConstraints() {
	type fkDef struct {
		table      string
		constraint string
		column     string
		refTable   string
		refColumn  string
		onDelete   string
	}

	fks := []fkDef{
		// UserSession → User
		{"user_sessions", "fk_user_sessions_user", "user_id", "users", "id", "CASCADE"},
		// UserApproval → User (two FKs)
		{"user_approvals", "fk_user_approvals_user", "user_id", "users", "id", "RESTRICT"},
		{"user_approvals", "fk_user_approvals_approver", "approved_by", "users", "id", "RESTRICT"},
		// AdminLog → User (two FKs)
		{"admin_logs", "fk_admin_logs_admin", "admin_id", "users", "id", "RESTRICT"},
		{"admin_logs", "fk_admin_logs_target_user", "target_user_id", "users", "id", "SET NULL"},
		// UserPosition → User
		{"user_positions", "fk_user_positions_user", "user_id", "users", "id", "RESTRICT"},
		// PendingAction → User (two FKs)
		{"pending_actions", "fk_pending_actions_requester", "requested_by", "users", "id", "RESTRICT"},
		{"pending_actions", "fk_pending_actions_approver", "approved_by", "users", "id", "SET NULL"},
		// Member → User (two FKs)
		{"members", "fk_members_registered_by", "registered_by", "users", "id", "RESTRICT"},
		{"members", "fk_members_user", "user_id", "users", "id", "SET NULL"},
		// Contribution → Member, User
		{"contributions", "fk_contributions_member", "member_id", "members", "id", "RESTRICT"},
		{"contributions", "fk_contributions_recorder", "recorded_by", "users", "id", "RESTRICT"},
		{"contributions", "fk_contributions_confirmer", "confirmed_by", "users", "id", "SET NULL"},
		// ContributionEdit → Contribution, User
		{"contribution_edits", "fk_contribution_edits_contribution", "contribution_id", "contributions", "id", "RESTRICT"},
		{"contribution_edits", "fk_contribution_edits_editor", "edited_by", "users", "id", "RESTRICT"},
		// Loan → Member, User
		{"loans", "fk_loans_member", "member_id", "members", "id", "RESTRICT"},
		{"loans", "fk_loans_reviewer", "reviewed_by", "users", "id", "SET NULL"},
		{"loans", "fk_loans_disburser", "disbursed_by", "users", "id", "SET NULL"},
		// Repayment → Loan, Member, User
		{"repayments", "fk_repayments_loan", "loan_id", "loans", "id", "RESTRICT"},
		{"repayments", "fk_repayments_member", "member_id", "members", "id", "RESTRICT"},
		{"repayments", "fk_repayments_recorder", "recorded_by", "users", "id", "RESTRICT"},
		// LoanCommitteeMember → User (two FKs)
		{"loan_committee_members", "fk_loan_committee_members_user", "user_id", "users", "id", "RESTRICT"},
		{"loan_committee_members", "fk_loan_committee_members_appointer", "appointed_by", "users", "id", "RESTRICT"},
		// LoanReview → Loan, User
		{"loan_reviews", "fk_loan_reviews_loan", "loan_id", "loans", "id", "RESTRICT"},
		{"loan_reviews", "fk_loan_reviews_reviewer", "reviewer_id", "users", "id", "RESTRICT"},
		// WelfareEvent → Member, User
		{"welfare_events", "fk_welfare_events_member", "member_id", "members", "id", "RESTRICT"},
		{"welfare_events", "fk_welfare_events_creator", "created_by", "users", "id", "RESTRICT"},
		{"welfare_events", "fk_welfare_events_approver", "approved_by", "users", "id", "SET NULL"},
		{"welfare_events", "fk_welfare_events_rejector", "rejected_by", "users", "id", "SET NULL"},
		// WelfareContribution → WelfareEvent, Member, User
		{"welfare_contributions", "fk_welfare_contributions_event", "event_id", "welfare_events", "id", "RESTRICT"},
		{"welfare_contributions", "fk_welfare_contributions_member", "member_id", "members", "id", "RESTRICT"},
		{"welfare_contributions", "fk_welfare_contributions_recorder", "recorded_by", "users", "id", "SET NULL"},
		// AuditLog → User
		{"audit_logs", "fk_audit_logs_user", "user_id", "users", "id", "SET NULL"},
		// Notification → User
		{"notifications", "fk_notifications_user", "user_id", "users", "id", "CASCADE"},
		// LeadershipPosition → Member
		{"leadership_positions", "fk_leadership_positions_member", "member_id", "members", "id", "CASCADE"},
		// FineSettings → Group
		{"fine_settings", "fk_fine_settings_group", "group_id", "groups", "id", "CASCADE"},
		// Fine → Group, Member, User
		{"fines", "fk_fines_group", "group_id", "groups", "id", "CASCADE"},
		{"fines", "fk_fines_member", "member_id", "members", "id", "RESTRICT"},
		{"fines", "fk_fines_collector", "collected_by", "users", "id", "SET NULL"},
		{"fines", "fk_fines_waiver", "waived_by", "users", "id", "SET NULL"},
	}

	for _, fk := range fks {
		// Skip if constraint already exists
		var exists int64
		DB.Raw(`SELECT 1 FROM pg_constraint WHERE conname = ?`, fk.constraint).Scan(&exists)
		if exists == 1 {
			continue
		}

		sql := `ALTER TABLE "` + fk.table + `" ADD CONSTRAINT "` + fk.constraint + `"` +
			` FOREIGN KEY ("` + fk.column + `")` +
			` REFERENCES "` + fk.refTable + `" ("` + fk.refColumn + `")` +
			` ON DELETE ` + fk.onDelete

		if err := DB.Exec(sql).Error; err != nil {
			log.Printf("FK %s.%s: %v", fk.table, fk.constraint, err)
		}
	}
}


