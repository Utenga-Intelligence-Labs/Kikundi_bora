package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// ListAllUsers returns all users with full details (Admin only).
func (h *AdminHandler) ListAllUsers(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	offset := pq.GetOffset()

	status := c.Query("status")
	role := c.Query("role")
	search := c.Query("q")

	query := database.DB.Model(&models.User{}).Where("deleted_at IS NULL")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if search != "" {
		searchPattern := likePattern(search)
		query = query.Where("LOWER(name) LIKE ? ESCAPE '\\' OR phone LIKE ? ESCAPE '\\' OR LOWER(email) LIKE ? ESCAPE '\\'", searchPattern, searchPattern, searchPattern)
	}

	var total int64
	query.Count(&total)

	var users []models.User
	query.Order("created_at DESC").
		Offset(offset).
		Limit(pq.Limit).
		Find(&users)

	return c.JSON(fiber.Map{
		"data":  users,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// GetAdminLogs returns admin action logs (Admin only).
func (h *AdminHandler) GetAdminLogs(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	offset := pq.GetOffset()

	var total int64
	database.DB.Model(&models.AdminLog{}).Count(&total)

	var logs []models.AdminLog
	database.DB.
		Preload("Admin", func(db *gorm.DB) *gorm.DB { return db.Select("id, name, role") }).
		Preload("TargetUser", func(db *gorm.DB) *gorm.DB { return db.Select("id, name, phone, role") }).
		Order("created_at DESC").
		Offset(offset).
		Limit(pq.Limit).
		Find(&logs)

	return c.JSON(fiber.Map{
		"data":  logs,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// OverrideUser allows admin to activate/deactivate/suspend any user.
func (h *AdminHandler) OverrideUser(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)
	targetID := c.Params("id")

	var req models.AdminOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", targetID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	oldStatus := user.Status

	var newStatus string
	var isActive bool
	switch req.Action {
	case "activate":
		newStatus = models.UserStatusActive
		isActive = true
	case "deactivate":
		newStatus = models.UserStatusRejected
		isActive = false
	case "suspend":
		newStatus = models.UserStatusSuspended
		isActive = false
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kitendo si sahihi"})
	}

	updates := map[string]interface{}{
		"status":    newStatus,
		"is_active": isActive,
	}
	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha"})
	}

	// Log to admin_logs
	metadata, _ := json.Marshal(map[string]interface{}{
		"action":      req.Action,
		"reason":      req.Reason,
		"old_status":  oldStatus,
		"new_status":  newStatus,
		"target_name": user.Name,
	})
	adminLog := models.AdminLog{
		AdminID:      adminID,
		Action:       "USER_OVERRIDE",
		TargetUserID: &user.ID,
		Metadata:     string(metadata),
		IPAddress:    c.IP(),
	}
	database.DB.Create(&adminLog)

	// Log to audit
	services.LogAudit(c, &adminID, models.AuditAdminOverride, "users", &user.ID, nil, map[string]interface{}{
		"action": req.Action, "reason": req.Reason, "old_status": oldStatus, "new_status": newStatus,
	})

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Mtumiaji %s. Hali mpya: %s", req.Action, newStatus),
	})
}

// ResetUserPassword resets a user's password to default (Admin only).
func (h *AdminHandler) ResetUserPassword(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)
	targetID := c.Params("id")

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", targetID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	// Generate once (or use provided password); return plaintext once to admin
	newPwd := models.DefaultTempPassword()
	providedPwd := false
	var req models.AdminResetPasswordRequest
	if err := c.BodyParser(&req); err == nil && req.NewPassword != "" {
		if len(req.NewPassword) < 8 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nenosiri lazima liwe na angalau herufi 8"})
		}
		newPwd = req.NewPassword
		providedPwd = true
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	updates := map[string]interface{}{
		"password":             string(hashedPwd),
		"must_change_password": true,
	}
	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha nenosiri"})
	}

	// Log to admin_logs
	metadata, _ := json.Marshal(map[string]interface{}{
		"action":       "reset_password",
		"target_name":  user.Name,
		"target_phone": user.Phone,
	})
	adminLog := models.AdminLog{
		AdminID:      adminID,
		Action:       "PASSWORD_RESET",
		TargetUserID: &user.ID,
		Metadata:     string(metadata),
		IPAddress:    c.IP(),
	}
	database.DB.Create(&adminLog)

	// Log to audit
	services.LogAudit(c, &adminID, models.AuditAdminOverride, "users", &user.ID, nil, map[string]interface{}{
		"action": "reset_password",
	})

	// Notify user — do not include plaintext password in notifications
	services.NotifyUser(user.ID, models.NotifSystem,
		"Nenosiri Limewekwa Upya",
		"Nenosiri lako limewekwa upya na msimamizi. Tumia nenosiri la muda ulilopewa na msimamizi kuingia; utakazwa kuweka nenosiri jipya.",
	)

	resp := fiber.Map{
		"message": "Nenosiri limewekwa upya. Mpe mtumiaji nenosiri la muda moja kwa moja (halitaonekana tena).",
	}
	// Only return generated temp password (not admin-supplied password) once
	if !providedPwd {
		resp["temp_password"] = newPwd
	}
	return c.JSON(resp)
}

// GetSystemHealth returns system statistics (Admin only).
func (h *AdminHandler) GetSystemHealth(c *fiber.Ctx) error {
	var stats struct {
		TotalUsers      int64 `json:"total_users"`
		PendingUsers    int64 `json:"pending_users"`
		ActiveUsers     int64 `json:"active_users"`
		RejectedUsers   int64 `json:"rejected_users"`
		SuspendedUsers  int64 `json:"suspended_users"`
		UsersByRole     []struct {
			Role  string `json:"role"`
			Count int64  `json:"count"`
		} `json:"users_by_role"`
		RecentLogins int64 `json:"recent_logins_24h"`
		TotalMembers int64 `json:"total_members"`
		TotalLoans   int64 `json:"total_loans"`
	}

	database.DB.Model(&models.User{}).Where("deleted_at IS NULL").Count(&stats.TotalUsers)
	database.DB.Model(&models.User{}).Where("status = ? AND deleted_at IS NULL", models.UserStatusPending).Count(&stats.PendingUsers)
	database.DB.Model(&models.User{}).Where("status = ? AND deleted_at IS NULL", models.UserStatusActive).Count(&stats.ActiveUsers)
	database.DB.Model(&models.User{}).Where("status = ? AND deleted_at IS NULL", models.UserStatusRejected).Count(&stats.RejectedUsers)
	database.DB.Model(&models.User{}).Where("status = ? AND deleted_at IS NULL", models.UserStatusSuspended).Count(&stats.SuspendedUsers)

	database.DB.Model(&models.User{}).
		Select("role, count(*) as count").
		Where("deleted_at IS NULL").
		Group("role").
		Scan(&stats.UsersByRole)

	oneDayAgo := time.Now().Add(-24 * time.Hour)
	database.DB.Model(&models.User{}).Where("last_login_at > ? AND deleted_at IS NULL", oneDayAgo).Count(&stats.RecentLogins)
	database.DB.Model(&models.Member{}).Where("is_active = TRUE").Count(&stats.TotalMembers)
	database.DB.Model(&models.Loan{}).Count(&stats.TotalLoans)

	return c.JSON(stats)
}
