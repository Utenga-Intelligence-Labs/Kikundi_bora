package handlers

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

type BackupHandler struct{}

func NewBackupHandler() *BackupHandler {
	return &BackupHandler{}
}

func (h *BackupHandler) GenerateBackup(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)

	var recentCount int64
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	database.DB.Model(&models.BackupHistory{}).
		Where("created_by = ? AND created_at > ?", adminID, oneHourAgo).
		Count(&recentCount)
	if recentCount >= 5 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"message": "Umefikia kiwango cha juu cha backup. Jaribu tena baada ya saa 1.",
		})
	}

	var req models.GenerateBackupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if req.BackupType == "" {
		req.BackupType = models.BackupTypeDatabase
	}

	record, err := services.GenerateBackup(req.BackupType, adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kuunda backup",
		})
	}

	settings, _ := services.GetBackupSettings()
	if settings.Email != "" && services.IsEmailConfigured() {
		go services.SendBackupEmail(record, settings.Email)
	}

	return c.JSON(fiber.Map{
		"message": "Backup imeundwa",
		"data":    record,
	})
}

func (h *BackupHandler) GetBackupHistory(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	records, total, err := services.GetBackupHistory(page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	return c.JSON(fiber.Map{
		"data":  records,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *BackupHandler) GetBackupSettings(c *fiber.Ctx) error {
	settings, err := services.GetBackupSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}
	return c.JSON(settings)
}

func (h *BackupHandler) SaveBackupSettings(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)

	var req models.SaveBackupSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	settings, err := services.SaveBackupSettings(req, adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi"})
	}

	return c.JSON(fiber.Map{"message": "Mipangilio imehifadhiwa", "data": settings})
}

func (h *BackupHandler) DownloadBackup(c *fiber.Ctx) error {
	id := c.Params("id")

	var record models.BackupHistory
	if err := database.DB.First(&record, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Backup haipatikana"})
	}

	// Sanitize filename from DB
	cleaned := filepath.Base(record.Filename)
	if cleaned == "." || cleaned == ".." || strings.Contains(cleaned, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Jina la faili si sahihi"})
	}

	path := filepath.Join("backups", cleaned)

	absPath, _ := filepath.Abs(path)
	absBase, _ := filepath.Abs("backups")
	if !strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Ruhusa imekataliwa"})
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Faili ya backup haipatikana"})
	}

	return c.Download(path, cleaned)
}
