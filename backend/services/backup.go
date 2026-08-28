package services

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"
)

const backupDir = "./backups"

func GenerateBackup(backupType, userID string) (*models.BackupHistory, error) {
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006_01_02_150405")
	filename := fmt.Sprintf("backup_%s.zip", timestamp)
	zipPath := filepath.Join(backupDir, filename)

	record := models.BackupHistory{
		Filename:   filename,
		BackupType: backupType,
		Status:     models.BackupStatusPending,
		CreatedBy:  userID,
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}

	var err error
	switch backupType {
	case models.BackupTypeDatabase:
		err = generateDBBackup(zipPath)
	case models.BackupTypeWithFiles:
		err = generateDBWithFilesBackup(zipPath)
	case models.BackupTypeFull:
		err = generateFullBackup(zipPath)
	default:
		err = generateDBBackup(zipPath)
	}

	if err != nil {
		record.Status = models.BackupStatusFailed
		record.ErrorMessage = err.Error()
		database.DB.Save(&record)
		return &record, err
	}

	info, err := os.Stat(zipPath)
	if err != nil {
		record.Status = models.BackupStatusFailed
		record.ErrorMessage = err.Error()
		database.DB.Save(&record)
		return &record, err
	}
	record.SizeBytes = info.Size()
	record.Status = models.BackupStatusCompleted
	database.DB.Save(&record)

	LogAuditBackup(&userID, models.AuditAction("BACKUP_GENERATE"), "backups", &record.ID, nil, map[string]interface{}{
		"filename":    filename,
		"backup_type": backupType,
		"size_bytes":  info.Size(),
	})

	return &record, nil
}

func generateDBBackup(zipPath string) error {
	cfg := config.AppConfig
	dumpFile := filepath.Join(backupDir, "dump.sql")

	cmd := exec.Command("pg_dump",
		"-h", cfg.DBHost,
		"-p", cfg.DBPort,
		"-U", cfg.DBUser,
		"-d", cfg.DBName,
		"--no-owner",
		"--no-privileges",
		"-f", dumpFile,
	)
	// Use PGPASSFILE to avoid password in process listing
	pgpass, _ := os.CreateTemp("", "pgpass")
	if pgpass != nil {
		pgpass.WriteString(fmt.Sprintf("%s:%s:%s:%s:%s\n",
			cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword))
		pgpass.Close()
		cmd.Env = append(os.Environ(), "PGPASSFILE="+pgpass.Name())
		defer os.Remove(pgpass.Name())
	} else {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.DBPassword)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}
	defer os.Remove(dumpFile)

	return createZip(zipPath, map[string]string{
		"dump.sql": dumpFile,
	})
}

func generateDBWithFilesBackup(zipPath string) error {
	dumpFile := filepath.Join(backupDir, "dump.sql")

	cfg := config.AppConfig
	cmd := exec.Command("pg_dump",
		"-h", cfg.DBHost,
		"-p", cfg.DBPort,
		"-U", cfg.DBUser,
		"-d", cfg.DBName,
		"--no-owner",
		"--no-privileges",
		"-f", dumpFile,
	)
	pgpass, _ := os.CreateTemp("", "pgpass")
	if pgpass != nil {
		pgpass.WriteString(fmt.Sprintf("%s:%s:%s:%s:%s\n",
			cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword))
		pgpass.Close()
		cmd.Env = append(os.Environ(), "PGPASSFILE="+pgpass.Name())
		defer os.Remove(pgpass.Name())
	} else {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.DBPassword)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}
	defer os.Remove(dumpFile)

	files := map[string]string{
		"dump.sql": dumpFile,
	}

	walkErr := filepath.Walk("uploads", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		files[path] = path
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		log.Printf("Warning: walking uploads dir: %v", walkErr)
	}

	return createZip(zipPath, files)
}

func generateFullBackup(zipPath string) error {
	return generateDBWithFilesBackup(zipPath)
}

func createZip(zipPath string, files map[string]string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	for name, path := range files {
		if err := addFileToZip(w, name, path); err != nil {
			log.Printf("Warning: skipping file %s: %v", path, err)
		}
	}

	return nil
}

func addFileToZip(w *zip.Writer, name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, f)
	return err
}

func SendBackupEmail(record *models.BackupHistory, to string) error {
	zipPath := filepath.Join(backupDir, record.Filename)

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", record.Filename)
	}

	subject := "Nakala ya Mfumo — Money Seeking"
	body := fmt.Sprintf(`
		<h2>Nakala ya Mfumo</h2>
		<p>Backup ya mfumo imeambatanishwa.</p>
		<table>
			<tr><td><strong>Tarehe:</strong></td><td>%s</td></tr>
			<tr><td><strong>Aina:</strong></td><td>%s</td></tr>
			<tr><td><strong>Ukubwa:</strong></td><td>%s</td></tr>
		</table>
	`, record.CreatedAt.Format("2006-01-02 15:04:05"), record.BackupType, formatBytes(record.SizeBytes))

	if err := SendEmail(to, subject, body, []string{zipPath}); err != nil {
		record.Status = models.BackupStatusFailed
		record.ErrorMessage = err.Error()
		database.DB.Save(record)
		return err
	}

	record.EmailSentTo = to
	record.Status = models.BackupStatusCompleted
	database.DB.Save(record)

	os.Remove(zipPath)

	return nil
}

func GetBackupHistory(page, limit int) ([]models.BackupHistory, int64, error) {
	var records []models.BackupHistory
	var total int64

	database.DB.Model(&models.BackupHistory{}).Count(&total)
	database.DB.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&records)

	for i := range records {
		var creator models.User
		if err := database.DB.Select("id, name").First(&creator, records[i].CreatedBy).Error; err == nil {
			records[i].Creator = &creator
		}
	}

	return records, total, nil
}

func GetBackupSettings() (*models.BackupSettings, error) {
	var settings models.BackupSettings
	if err := database.DB.First(&settings).Error; err != nil {
		return &models.BackupSettings{
			BackupType: models.BackupTypeDatabase,
			Frequency:  models.FrequencyManual,
		}, nil
	}
	return &settings, nil
}

func SaveBackupSettings(req models.SaveBackupSettingsRequest, userID string) (*models.BackupSettings, error) {
	var settings models.BackupSettings
	result := database.DB.Where("1 = 1").Assign(models.BackupSettings{
		Email:      req.Email,
		BackupType: req.BackupType,
		Frequency:  req.Frequency,
		UpdatedBy:  userID,
	}).FirstOrCreate(&settings)

	if result.Error != nil {
		return nil, result.Error
	}
	return &settings, nil
}

func CleanupOldBackups() {
	cutoff := time.Now().AddDate(0, 0, -7)
	var old []models.BackupHistory
	database.DB.Where("created_at < ? AND status = ?", cutoff, models.BackupStatusCompleted).Find(&old)
	for _, r := range old {
		os.Remove(filepath.Join(backupDir, r.Filename))
		database.DB.Delete(&r)
	}
	if len(old) > 0 {
		log.Printf("Cleaned up %d old backups", len(old))
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
