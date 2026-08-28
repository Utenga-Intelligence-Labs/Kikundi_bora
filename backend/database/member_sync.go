package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	"kikundibora/models"

	"gorm.io/gorm"
)

// EnsureMemberForUser links or creates a members row for a non-admin user.
// Idempotent: safe to call on create, approve, and backfill.
// registeredBy must be a valid users.id (FK); falls back to user.ID if empty.
func EnsureMemberForUser(db *gorm.DB, user models.User, registeredBy string) error {
	if db == nil {
		db = DB
	}
	if user.Role == models.RoleAdmin {
		return nil
	}
	if registeredBy == "" {
		if user.CreatedBy != nil && *user.CreatedBy != "" {
			registeredBy = *user.CreatedBy
		} else {
			registeredBy = user.ID
		}
	}

	var byUser models.Member
	err := db.Where("user_id = ? AND deleted_at IS NULL", user.ID).First(&byUser).Error
	if err == nil {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var byPhone models.Member
	err = db.Where("phone = ? AND deleted_at IS NULL", user.Phone).First(&byPhone).Error
	if err == nil {
		if byPhone.UserID == nil || *byPhone.UserID == "" {
			return db.Model(&byPhone).Updates(map[string]interface{}{
				"user_id":   user.ID,
				"full_name": user.Name,
			}).Error
		}
		if *byPhone.UserID == user.ID {
			return nil
		}
		return fmt.Errorf("simu %s tayari imesajiliwa kwa mwanachama mwingine", user.Phone)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	memberNo, err := NextMemberNo(db)
	if err != nil {
		return err
	}

	uid := user.ID
	member := models.Member{
		UserID:       &uid,
		MemberNo:     memberNo,
		FullName:     user.Name,
		Phone:        user.Phone,
		JoinedAt:     time.Now(),
		IsActive:     true,
		RegisteredBy: registeredBy,
	}
	return db.Create(&member).Error
}

// NextMemberNo returns the next KKK-NNNN membership number.
// Retries up to 5 times if a duplicate is detected (race condition).
func NextMemberNo(db *gorm.DB) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		var maxNum int
		if err := db.Raw(`
			SELECT COALESCE(MAX(CAST(SUBSTRING(member_no FROM 5) AS INTEGER)), 0)
			FROM members
			WHERE deleted_at IS NULL
			  AND member_no ~ '^KKK-[0-9]+$'
		`).Scan(&maxNum).Error; err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("KKK-%04d", maxNum+1+attempt)
		// Verify uniqueness
		var count int64
		db.Model(&models.Member{}).Where("member_no = ? AND deleted_at IS NULL", candidate).Count(&count)
		if count == 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique member number after 5 attempts")
}

// BackfillMembersFromUsers creates/links member rows for existing non-admin users
// who do not yet have a members record (e.g. approved before auto-member fix).
func BackfillMembersFromUsers() {
	if DB == nil {
		return
	}
	var users []models.User
	if err := DB.Where("deleted_at IS NULL AND role <> ?", models.RoleAdmin).
		Order("created_at ASC").
		Find(&users).Error; err != nil {
		log.Printf("Backfill members: list users failed: %v", err)
		return
	}

	created := 0
	for _, u := range users {
		var count int64
		DB.Model(&models.Member{}).Where("user_id = ? AND deleted_at IS NULL", u.ID).Count(&count)
		if count > 0 {
			continue
		}
		if err := EnsureMemberForUser(DB, u, ""); err != nil {
			log.Printf("Backfill members: skip user %s (%s): %v", u.Name, u.Phone, err)
			continue
		}
		created++
	}
	if created > 0 {
		log.Printf("Backfill members: created/linked %d member row(s)", created)
	}
}
