package handlers

import (
	"strconv"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// List returns audit logs with comprehensive filters (Admin and Chair).
func (h *AuditHandler) List(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	tableName := c.Query("table")
	action := c.Query("action")
	userID := c.Query("user_id")
	search := c.Query("q")
	dateFrom := c.Query("from")
	dateTo := c.Query("to")

	query := database.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, name, email, role")
	})

	if tableName != "" {
		query = query.Where("table_name = ?", tableName)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if search != "" {
		searchPattern := likePattern(search)
		query = query.Where("ip_address LIKE ? ESCAPE '\\' OR user_agent LIKE ? ESCAPE '\\'", searchPattern, searchPattern)
	}
	if dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	query.Model(&models.AuditLog{}).Count(&total)

	var logs []models.AuditLog
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&logs)

	return c.JSON(fiber.Map{
		"data":  logs,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// LoginActivity returns login-specific audit events (logins, logouts, failed attempts).
// Accessible by Admin and Chair.
func (h *AuditHandler) LoginActivity(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	userID := c.Query("user_id")
	ipAddress := c.Query("ip")
	dateFrom := c.Query("from")
	dateTo := c.Query("to")
	actionFilter := c.Query("action") // LOGIN, LOGOUT, PASSWORD_SET

	// Base query for login-related actions
	query := database.DB.
		Where("action IN ?", []models.AuditAction{
			models.AuditLogin,
			models.AuditLogout,
			models.AuditPasswordSet,
		}).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, email, role, phone")
		})

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}
	if actionFilter != "" {
		query = query.Where("action = ?", actionFilter)
	}
	if dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	query.Model(&models.AuditLog{}).Count(&total)

	var logs []models.AuditLog
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&logs)

	// Also get failed login attempts from the failed_logins table
	var failedTotal int64
	failedQuery := database.DB.Model(&models.FailedLogin{})
	if ipAddress != "" {
		failedQuery = failedQuery.Where("ip_address = ?", ipAddress)
	}
	if dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			failedQuery = failedQuery.Where("attempted_at >= ?", t)
		}
	}
	if dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			failedQuery = failedQuery.Where("attempted_at < ?", t.AddDate(0, 0, 1))
		}
	}
	failedQuery.Count(&failedTotal)

	return c.JSON(fiber.Map{
		"data":            logs,
		"total":           total,
		"failed_attempts": failedTotal,
		"page":            pq.Page,
		"limit":           pq.Limit,
	})
}

// FailedLogins returns failed login attempts (Admin and Chair).
func (h *AuditHandler) FailedLogins(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	ipAddress := c.Query("ip")
	email := c.Query("email")
	dateFrom := c.Query("from")
	dateTo := c.Query("to")

	query := database.DB.Model(&models.FailedLogin{})

	if ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}
	if email != "" {
		emailPattern := likePattern(email)
		query = query.Where("email_attempted LIKE ? ESCAPE '\\'", emailPattern)
	}
	if dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("attempted_at >= ?", t)
		}
	}
	if dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("attempted_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	query.Count(&total)

	var logs []models.FailedLogin
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("attempted_at DESC").Find(&logs)

	return c.JSON(fiber.Map{
		"data":  logs,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// AuditSummary returns aggregate statistics about audit events (Admin and Chair).
func (h *AuditHandler) AuditSummary(c *fiber.Ctx) error {
	days := 30 // default lookback
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	var stats struct {
		TotalEvents     int64 `json:"total_events"`
		LoginCount      int64 `json:"login_count"`
		LogoutCount     int64 `json:"logout_count"`
		FailedLogins    int64 `json:"failed_logins"`
		CreateCount     int64 `json:"create_count"`
		UpdateCount     int64 `json:"update_count"`
		DeleteCount     int64 `json:"delete_count"`
		PasswordChanges int64 `json:"password_changes"`
		UniqueUsers     int64 `json:"unique_users"`
		UniqueIPs       int64 `json:"unique_ips"`
	}

	database.DB.Model(&models.AuditLog{}).Where("created_at >= ?", cutoff).Count(&stats.TotalEvents)
	database.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", models.AuditLogin, cutoff).Count(&stats.LoginCount)
	database.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", models.AuditLogout, cutoff).Count(&stats.LogoutCount)
	database.DB.Model(&models.AuditLog{}).Where("action IN ? AND created_at >= ?", []models.AuditAction{models.AuditCreate, models.AuditUserCreated}, cutoff).Count(&stats.CreateCount)
	database.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", models.AuditUpdate, cutoff).Count(&stats.UpdateCount)
	database.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", models.AuditDelete, cutoff).Count(&stats.DeleteCount)
	database.DB.Model(&models.AuditLog{}).Where("action = ? AND created_at >= ?", models.AuditPasswordSet, cutoff).Count(&stats.PasswordChanges)

	database.DB.Model(&models.FailedLogin{}).Where("attempted_at >= ?", cutoff).Count(&stats.FailedLogins)

	database.DB.Model(&models.AuditLog{}).Where("created_at >= ?", cutoff).Distinct("user_id").Count(&stats.UniqueUsers)
	database.DB.Model(&models.AuditLog{}).Where("created_at >= ? AND ip_address IS NOT NULL", cutoff).Distinct("ip_address").Count(&stats.UniqueIPs)

	return c.JSON(fiber.Map{
		"period_days": days,
		"stats":       stats,
	})
}
