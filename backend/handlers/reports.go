package handlers

import (
	"os"
	"path/filepath"
	"strings"

	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

type ReportHandler struct{}

func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

func (h *ReportHandler) MembersReport(c *fiber.Ctx) error {
	report, err := services.GenerateMembersReport()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda ripoti"})
	}
	defer os.Remove(report.Path)
	return c.Download(report.Path, report.Filename)
}

func (h *ReportHandler) ContributionsReport(c *fiber.Ctx) error {
	month := c.Query("month")
	report, err := services.GenerateContributionsReport(month)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda ripoti"})
	}
	defer os.Remove(report.Path)
	return c.Download(report.Path, report.Filename)
}

func (h *ReportHandler) LoansReport(c *fiber.Ctx) error {
	status := c.Query("status")
	report, err := services.GenerateLoansReport(status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda ripoti"})
	}
	defer os.Remove(report.Path)
	return c.Download(report.Path, report.Filename)
}

func (h *ReportHandler) IncomeExpenseReport(c *fiber.Ctx) error {
	report, err := services.GenerateIncomeExpenseReport()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda ripoti"})
	}
	defer os.Remove(report.Path)
	return c.Download(report.Path, report.Filename)
}

func (h *ReportHandler) SummaryReport(c *fiber.Ctx) error {
	report, err := services.GenerateSummaryReport()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda ripoti"})
	}
	defer os.Remove(report.Path)
	return c.Download(report.Path, report.Filename)
}

// DownloadReport serves a report file with path traversal protection
func (h *ReportHandler) DownloadReport(c *fiber.Ctx) error {
	filename := c.Params("filename")

	// Sanitize: strip any path components
	cleaned := filepath.Base(filename)
	if cleaned == "." || cleaned == ".." || strings.Contains(cleaned, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Jina la faili si sahihi"})
	}

	path := filepath.Join("uploads", "reports", cleaned)

	// Verify resolved path is within expected directory
	absPath, _ := filepath.Abs(path)
	absBase, _ := filepath.Abs("uploads/reports")
	if !strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Ruhusa imekataliwa"})
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Ripoti haipatikana"})
	}

	return c.Download(path, cleaned)
}
