package services

import (
	"encoding/json"
	"fmt"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
)

// LogAudit creates an audit log entry. Shared across all handlers.
func LogAudit(c *fiber.Ctx, userID *string, action models.AuditAction, table string, recordID *string, oldVals, newVals interface{}) {
	var oldJSON, newJSON json.RawMessage
	if oldVals != nil {
		b, _ := json.Marshal(oldVals)
		oldJSON = b
	}
	if newVals != nil {
		b, _ := json.Marshal(newVals)
		newJSON = b
	}
	var ip, ua string
	if c != nil {
		ip = c.IP()
		ua = c.Get("User-Agent")
	}
	log := models.AuditLog{
		UserID:    userID,
		Action:    action,
		TableName: table,
		RecordID:  recordID,
		OldValues: oldJSON,
		NewValues: newJSON,
		IPAddress: &ip,
		UserAgent: &ua,
	}
	if err := database.DB.Create(&log).Error; err != nil {
		fmt.Printf("AUDIT FAILED [%s.%s id=%v]: %v\n", table, action, recordID, err)
	}
}

// LogAuditBackup creates an audit log entry without a fiber context (used by backup service).
func LogAuditBackup(userID *string, action models.AuditAction, table string, recordID *string, oldVals, newVals interface{}) {
	var oldJSON, newJSON json.RawMessage
	if oldVals != nil {
		b, _ := json.Marshal(oldVals)
		oldJSON = b
	}
	if newVals != nil {
		b, _ := json.Marshal(newVals)
		newJSON = b
	}
	ip := "127.0.0.1"
	ua := "system/backup"
	log := models.AuditLog{
		UserID:    userID,
		Action:    action,
		TableName: table,
		RecordID:  recordID,
		OldValues: oldJSON,
		NewValues: newJSON,
		IPAddress: &ip,
		UserAgent: &ua,
	}
	if err := database.DB.Create(&log).Error; err != nil {
		fmt.Printf("AUDIT FAILED [%s.%s id=%v]: %v\n", table, action, recordID, err)
	}
}
