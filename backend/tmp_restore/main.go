package main

import (
	"fmt"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

func main() {
	config.Load()
	database.Connect()
	var chair models.User
	database.DB.Where("role = ? AND deleted_at IS NULL", models.RoleChair).First(&chair)
	var members []models.Member
	database.DB.Where("deleted_at IS NULL AND is_active = TRUE").Order("member_no").Find(&members)

	months := []time.Time{}
	for y, m := 2025, time.October; ; {
		months = append(months, time.Date(y, m, 1, 0, 0, 0, 0, time.UTC))
		if y == 2026 && m == time.August {
			break
		}
		m++
		if m > time.December {
			m = time.January
			y++
		}
	}
	amt := decimal.NewFromInt(2020000)
	tenK := decimal.NewFromInt(10000)
	added, skipped := 0, 0
	put := func(mem models.Member, month time.Time, a decimal.Decimal, day int) {
		var n int64
		database.DB.Model(&models.Contribution{}).Where("member_id = ? AND month = ?", mem.ID, month.Format("2006-01-02")).Count(&n)
		if n > 0 {
			skipped++
			return
		}
		note := "Demo backfill"
		database.DB.Create(&models.Contribution{
			MemberID: mem.ID, RecordedBy: chair.ID, Amount: a, Month: month,
			PaidAt: time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.UTC),
			PaymentMethod: "CASH", Status: "PAID", ConfirmedBy: &chair.ID, Notes: &note,
		})
		added++
	}
	for _, month := range months {
		for _, mem := range members {
			put(mem, month, amt, 28)
		}
	}
	sept := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	for _, mem := range members {
		put(mem, sept, tenK, 5)
	}
	var sum string
	database.DB.Raw("SELECT COALESCE(SUM(amount),0) FROM contributions WHERE status='PAID'").Scan(&sum)
	fmt.Printf("added=%d skipped=%d PAID total=%s\n", added, skipped, sum)
}
