package database

import (
	"log"

	"kikundibora/models"
)

// EnsureGroupSetup guarantees exactly one Group row exists (this deployment
// serves a single group). Safe defaults: interval=monthly, due date and
// fixed amount unset until a proposal is approved. Idempotent — runs on
// every startup.
func EnsureGroupSetup() {
	var count int64
	DB.Model(&models.Group{}).Count(&count)
	if count == 0 {
		g := models.Group{
			Name:                  "Kikundi Bora",
			ContributionInterval:  models.IntervalMonthly,
		}
		if err := DB.Create(&g).Error; err != nil {
			log.Printf("ERROR: Failed to create default group: %v", err)
			return
		}
		log.Printf("Created default group %s (%s)", g.Name, g.ID)
	}
}

// GetCurrentGroup returns the single group row, creating it if missing.
func GetCurrentGroup() (*models.Group, error) {
	EnsureGroupSetup()
	var g models.Group
	if err := DB.First(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// IsCurrentGroup reports whether :id refers to this deployment's group.
// RBAC-M01: group-scoped endpoints must reject foreign IDs with 404 even
// if a second group ever appears in the table.
func IsCurrentGroup(id string) (bool, error) {
	g, err := GetCurrentGroup()
	if err != nil {
		return false, err
	}
	return g.ID == id, nil
}
