package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/shopspring/decimal"

	"github.com/gofiber/fiber/v2"
)

type ImportHandler struct{}

func NewImportHandler() *ImportHandler {
	return &ImportHandler{}
}

// ImportResult represents the result of a CSV import
type ImportResult struct {
	TotalRows int `json:"total_rows"`
	Imported  int `json:"imported"`
	Skipped   int `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
}

// ImportContributions handles CSV file upload for historical contributions
// POST /api/v1/import/contributions
// CSV format: member_code, amount, type, date, status
// Example: KKK-0001, 50000, AKIBA, 2024-01-15, CONFIRMED
func (h *ImportHandler) ImportContributions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Faili ya CSV haijapatikana",
		})
	}

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Faili lazima iwe ya aina ya CSV",
		})
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kufungua faili",
		})
	}
	defer src.Close()

	// Limit CSV file size to 10MB to prevent memory exhaustion
	const maxCSVSize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(src, maxCSVSize+1)

	// Read CSV
	reader := csv.NewReader(limitedReader)
	records, err := reader.ReadAll()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Imeshindikana kusoma faili ya CSV: " + err.Error(),
		})
	}

	if len(records) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Faili ya CSV haina data (inahitaji angalau header na safu moja)",
		})
	}

	result := ImportResult{
		TotalRows: len(records) - 1, // Exclude header
	}

	// Process rows (skip header)
	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}

		if len(record) < 5 {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: columns hazitoshi (inahitaji 5)", i+1))
			result.Skipped++
			continue
		}

		memberCode := strings.TrimSpace(record[0])
		amountStr := strings.TrimSpace(record[1])
		contribType := strings.TrimSpace(record[2])
		dateStr := strings.TrimSpace(record[3])
		status := strings.TrimSpace(record[4])

		// Find member by member_no
		var member models.Member
		if err := database.DB.Where("member_no = ? AND deleted_at IS NULL", memberCode).First(&member).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Mwanachama %s hajapatikana", i+1, memberCode))
			result.Skipped++
			continue
		}

		// Parse amount
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Kiasi si sahihi: %s", i+1, amountStr))
			result.Skipped++
			continue
		}

		// Validate type
		if contribType != "AKIBA" && contribType != "MFUKO_WA_KIJAMII" {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Aina si sahihi: %s (inahitaji AKIBA au MFUKO_WA_KIJAMII)", i+1, contribType))
			result.Skipped++
			continue
		}

		// Parse date
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Tarehe si sahihi: %s (format: YYYY-MM-DD)", i+1, dateStr))
			result.Skipped++
			continue
		}

		// Validate status
		if status != "CONFIRMED" && status != "PENDING_VERIFICATION" && status != "REJECTED" {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Status si sahihi: %s (inahitaji CONFIRMED, PENDING_VERIFICATION, au REJECTED)", i+1, status))
			result.Skipped++
			continue
		}

		// Create contribution
		contribution := models.MemberContribution{
			MemberID:           member.ID,
			ContributionType:   models.ContributionType(contribType),
			PeriodLabel:        parsedDate.Format("2006-01"),
			Amount:             decimal.NewFromFloat(amount),
			Status:             models.ContributionStatus(status),
			IsHistoricalImport: true,
			CreatedAt:          parsedDate,
		}

		if status == "CONFIRMED" || status == "REJECTED" {
			contribution.ReviewedByMemberID = member.ID
			contribution.ReviewedAt = time.Now()
		}

		if err := database.DB.Create(&contribution).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Imeshindikana kuunda: %v", i+1, err))
			result.Skipped++
			continue
		}

		result.Imported++
	}

	// Log audit
	services.LogAudit(c, &userID, models.AuditCreate, "member_contributions_import", nil, nil, map[string]interface{}{
		"total_rows": result.TotalRows,
		"imported":   result.Imported,
		"skipped":    result.Skipped,
	})

	log.Printf("CSV Import by user %s: %d imported, %d skipped out of %d total",
		userID, result.Imported, result.Skipped, result.TotalRows)

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Import imekamilika: %d ziliingizwa, %d zilikataliwa", result.Imported, result.Skipped),
		"data":    result,
	})
}

// ImportLoans handles CSV file upload for historical loans
// POST /api/v1/import/loans
// CSV format: member_code, amount, purpose, due_date, status, approved_amount
// Example: KKK-0001, 200000, Biashara, 2024-12-31, CLOSED, 200000
func (h *ImportHandler) ImportLoans(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Faili ya CSV haijapatikana",
		})
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Faili lazima iwe ya aina ya CSV",
		})
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kufungua faili",
		})
	}
	defer src.Close()

	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Imeshindikana kusoma faili ya CSV: " + err.Error(),
		})
	}

	if len(records) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Faili ya CSV haina data",
		})
	}

	result := ImportResult{
		TotalRows: len(records) - 1,
	}

	for i, record := range records {
		if i == 0 {
			continue
		}

		if len(record) < 6 {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: columns hazitoshi (inahitaji 6)", i+1))
			result.Skipped++
			continue
		}

		memberCode := strings.TrimSpace(record[0])
		amountStr := strings.TrimSpace(record[1])
		purpose := strings.TrimSpace(record[2])
		dueDateStr := strings.TrimSpace(record[3])
		status := strings.TrimSpace(record[4])
		approvedAmountStr := strings.TrimSpace(record[5])

		// Find member
		var member models.Member
		if err := database.DB.Where("member_no = ? AND deleted_at IS NULL", memberCode).First(&member).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Mwanachama %s hajapatikana", i+1, memberCode))
			result.Skipped++
			continue
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Kiasi si sahihi", i+1))
			result.Skipped++
			continue
		}

		dueDate, err := time.Parse("2006-01-02", dueDateStr)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Tarehe si sahihi", i+1))
			result.Skipped++
			continue
		}

		approvedAmount, _ := strconv.ParseFloat(approvedAmountStr, 64)
		if approvedAmount <= 0 {
			approvedAmount = amount
		}

		// Map status
		var loanStatus models.LoanStatus
		switch strings.ToUpper(status) {
		case "PENDING":
			loanStatus = models.LoanPending
		case "APPROVED":
			loanStatus = models.LoanApproved
		case "OUTSTANDING":
			loanStatus = models.LoanOutstanding
		case "CLOSED":
			loanStatus = models.LoanClosed
		case "REJECTED":
			loanStatus = models.LoanRejected
		default:
			loanStatus = models.LoanClosed
		}

		loan := models.Loan{
			MemberID:      member.ID,
			Amount:        decimal.NewFromFloat(amount),
			ApprovedAmount: func() *decimal.Decimal { d := decimal.NewFromFloat(approvedAmount); return &d }(),
			Purpose:       &purpose,
			DueDate:       dueDate,
			Status:        loanStatus,
			AppliedAt:     dueDate.AddDate(0, -1, 0), // Assume applied 1 month before due
		}

		if loanStatus == models.LoanClosed || loanStatus == models.LoanOutstanding {
			loan.ReviewedBy = &userID
			reviewedAt := dueDate.AddDate(0, -1, 0)
			loan.ReviewedAt = &reviewedAt
		}

		if err := database.DB.Create(&loan).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Safu %d: Imeshindikana kuunda: %v", i+1, err))
			result.Skipped++
			continue
		}

		result.Imported++
	}

	services.LogAudit(c, &userID, models.AuditCreate, "loans_import", nil, nil, map[string]interface{}{
		"total_rows": result.TotalRows,
		"imported":   result.Imported,
		"skipped":    result.Skipped,
	})

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Import imekamilika: %d ziliingizwa, %d zilikataliwa", result.Imported, result.Skipped),
		"data":    result,
	})
}
