package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

// CSV format: member_code, amount, type, date, status
// Example: KKK-0001, 50000, AKIBA, 2024-01-15, CONFIRMED

func main() {
	filePath := flag.String("file", "", "Path to CSV file")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("Tafadhali toa -file=<path>")
	}

	config.Load()
	database.Connect()

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("Imeshindikana kufungua faili: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Imeshindikana kusoma CSV: %v", err)
	}

	imported := 0
	skipped := 0

	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}

		if len(record) < 5 {
			log.Printf("Row %d: columns hazitoshi", i+1)
			skipped++
			continue
		}

		memberCode := strings.TrimSpace(record[0])
		amountStr := strings.TrimSpace(record[1])
		contribType := strings.TrimSpace(record[2])
		dateStr := strings.TrimSpace(record[3])
		status := strings.TrimSpace(record[4])

		// Find member
		var member models.Member
		if err := database.DB.Where("member_no = ? AND deleted_at IS NULL", memberCode).First(&member).Error; err != nil {
			log.Printf("Row %d: Mwanachama %s hajapatikana", i+1, memberCode)
			skipped++
			continue
		}

		// Parse amount
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			log.Printf("Row %d: Kiasi si sahihi: %s", i+1, amountStr)
			skipped++
			continue
		}

		// Validate type
		if contribType != "AKIBA" && contribType != "MFUKO_WA_KIJAMII" {
			log.Printf("Row %d: Aina si sahihi: %s", i+1, contribType)
			skipped++
			continue
		}

		// Parse date
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Row %d: Tarehe si sahihi: %s", i+1, dateStr)
			skipped++
			continue
		}

		// Validate status
		if status != "CONFIRMED" && status != "PENDING_VERIFICATION" && status != "REJECTED" {
			log.Printf("Row %d: Status si sahihi: %s", i+1, status)
			skipped++
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
			ReviewedAt:         time.Now(),
		}

		if status == "CONFIRMED" || status == "REJECTED" {
			contribution.ReviewedByMemberID = member.ID // Self-reviewed for historical
		}

		if err := database.DB.Create(&contribution).Error; err != nil {
			log.Printf("Row %d: Imeshindikana kuunda: %v", i+1, err)
			skipped++
			continue
		}

		imported++
	}

	fmt.Printf("\n✅ Import imekamilika!\n")
	fmt.Printf("   Iliyoingizwa: %d\n", imported)
	fmt.Printf("   Iliyokataliwa: %d\n", skipped)
}
