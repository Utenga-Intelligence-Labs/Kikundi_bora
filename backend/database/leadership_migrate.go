package database

import (
	"log"

	"kikundibora/models"
)

// MigrateLeadershipPositions creates the partial unique index and backfills
// leadership_positions from existing user_positions + members linkage.
func MigrateLeadershipPositions() {
	if DB == nil {
		return
	}

	// Create partial unique index: one active role per member
	DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_role_per_member
		ON leadership_positions(member_id, role)
		WHERE is_current = true
	`)

	// Backfill from existing user_positions + members
	// Map: CHAIRPERSON → MWENYEKITI, TREASURER → HAZINA, SECRETARY → KATIBU
	type posMapping struct {
		PositionType string
		LeadershipRole string
	}
	mappings := []posMapping{
		{"CHAIRPERSON", "MWENYEKITI"},
		{"TREASURER", "HAZINA"},
		{"SECRETARY", "KATIBU"},
	}

	for _, m := range mappings {
		// Find users with this position who have a linked member
		var userPositions []models.UserPosition
		if err := DB.Where("position_type = ? AND is_active = TRUE", m.PositionType).
			Find(&userPositions).Error; err != nil {
			log.Printf("Backfill leadership: query positions failed: %v", err)
			continue
		}

		for _, up := range userPositions {
			// Find member linked to this user
			var member models.Member
			if err := DB.Where("user_id = ? AND deleted_at IS NULL", up.UserID).
				First(&member).Error; err != nil {
				continue // No member linked to this user
			}

			// Check if leadership position already exists
			var count int64
			DB.Model(&models.LeadershipPosition{}).
				Where("member_id = ? AND role = ? AND is_current = TRUE",
					member.ID, m.LeadershipRole).
				Count(&count)
			if count > 0 {
				continue // Already has this leadership role
			}

			// Create leadership position
			lp := models.LeadershipPosition{
				MemberID:  member.ID,
				Role:      models.LeadershipRole(m.LeadershipRole),
				IsCurrent: true,
			}
			if err := DB.Create(&lp).Error; err != nil {
				log.Printf("Backfill leadership: create for member %s failed: %v",
					member.ID, err)
			}
		}
	}

	log.Println("Leadership positions migration + backfill complete.")
}
