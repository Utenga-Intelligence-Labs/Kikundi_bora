package services

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

const reportDir = "./uploads/reports"

type ReportData struct {
	Filename string
	Path     string
	Size     int64
}

func EnsureReportDir() {
	os.MkdirAll(reportDir, 0750)
}

func safeStat(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func GenerateMembersReport() (*ReportData, error) {
	EnsureReportDir()

	var members []models.Member
	database.DB.Where("deleted_at IS NULL").Order("member_no ASC").Find(&members)

	filename := fmt.Sprintf("wanachama_%s.csv", time.Now().Format("2006_01_02"))
	path := filepath.Join(reportDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"Namba", "Jina Kamili", "Simu", "Anwani", "Tarehe ya Kujiunga", "Hali"})

	for _, m := range members {
		status := "Hai"
		if !m.IsActive {
			status = "Hahai"
		}
		w.Write([]string{
			m.MemberNo,
			m.FullName,
			m.Phone,
			deref(m.Address),
			m.JoinedAt.Format("2006-01-02"),
			status,
		})
	}

	return &ReportData{Filename: filename, Path: path, Size: safeStat(path)}, nil
}

func GenerateContributionsReport(month string) (*ReportData, error) {
	EnsureReportDir()

	var contribs []models.Contribution
	query := database.DB
	if month != "" {
		query = query.Where("month = ?", month)
	}
	query.Order("paid_at DESC").Find(&contribs)

	for i := range contribs {
		var member models.Member
		if err := database.DB.Select("id, member_no, full_name, phone").First(&member, contribs[i].MemberID).Error; err == nil {
			contribs[i].Member = &member
		}
	}

	filename := fmt.Sprintf("michango_%s.csv", time.Now().Format("2006_01_02"))
	path := filepath.Join(reportDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"Mwanachama", "Namba", "Simu", "Kiasi", "Mwezi", "Tarehe ya Kulipa", "Njia", "Hali"})

	for _, c := range contribs {
		memberName := ""
		memberNo := ""
		memberPhone := ""
		if c.Member != nil {
			memberName = c.Member.FullName
			memberNo = c.Member.MemberNo
			memberPhone = c.Member.Phone
		}
		w.Write([]string{
			memberName,
			memberNo,
			memberPhone,
			c.Amount.StringFixed(0),
			c.Month.Format("2006-01"),
			c.PaidAt.Format("2006-01-02"),
			c.PaymentMethod,
			c.Status,
		})
	}

	return &ReportData{Filename: filename, Path: path, Size: safeStat(path)}, nil
}

func GenerateLoansReport(status string) (*ReportData, error) {
	EnsureReportDir()

	var loans []models.Loan
	query := database.DB
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Order("applied_at DESC").Find(&loans)

	for i := range loans {
		var member models.Member
		if err := database.DB.Select("id, member_no, full_name, phone").First(&member, loans[i].MemberID).Error; err == nil {
			loans[i].Member = &member
		}
	}

	filename := fmt.Sprintf("mikopo_%s.csv", time.Now().Format("2006_01_02"))
	path := filepath.Join(reportDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"Mwanachama", "Namba", "Kiasi", "Kiasi Ilioidhinishwa", "Salio", "Kusudi", "Tarehe ya Kukamilisha", "Hali"})

	for _, l := range loans {
		memberName := ""
		memberNo := ""
		if l.Member != nil {
			memberName = l.Member.FullName
			memberNo = l.Member.MemberNo
		}
		approved := l.Amount.String()
		if l.ApprovedAmount != nil && l.ApprovedAmount.GreaterThan(decimal.Zero) {
			approved = l.ApprovedAmount.String()
		}
		balance := "0"
		if l.BalanceRemaining != nil {
			balance = l.BalanceRemaining.String()
		}
		purpose := ""
		if l.Purpose != nil {
			purpose = *l.Purpose
		}
		w.Write([]string{
			memberName,
			memberNo,
			l.Amount.StringFixed(0),
			approved,
			balance,
			purpose,
			l.DueDate.Format("2006-01-02"),
			string(l.Status),
		})
	}

	return &ReportData{Filename: filename, Path: path, Size: safeStat(path)}, nil
}

func GenerateIncomeExpenseReport() (*ReportData, error) {
	EnsureReportDir()

	var totalContributions float64
	var totalRepayments float64
	var totalLoans float64

	database.DB.Model(&models.Contribution{}).Where("status = ?", "PAID").Select("COALESCE(SUM(amount),0)").Scan(&totalContributions)
	database.DB.Model(&models.Repayment{}).Select("COALESCE(SUM(amount),0)").Scan(&totalRepayments)
	database.DB.Model(&models.Loan{}).Where("status IN ?", []string{string(models.LoanOutstanding), string(models.LoanClosed)}).Select("COALESCE(SUM(approved_amount),0)").Scan(&totalLoans)

	filename := fmt.Sprintf("mapato_matumizi_%s.csv", time.Now().Format("2006_01_02"))
	path := filepath.Join(reportDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"Kitengo", "Kiasi (TZS)"})
	w.Write([]string{"Jumla ya Michango", fmt.Sprintf("%.0f", totalContributions)})
	w.Write([]string{"Jumla ya Marejesho", fmt.Sprintf("%.0f", totalRepayments)})
	w.Write([]string{"Mapato Yote", fmt.Sprintf("%.0f", totalContributions+totalRepayments)})
	w.Write([]string{"Jumla ya Mikopo Iliyotolewa", fmt.Sprintf("%.0f", totalLoans)})
	w.Write([]string{"Faida/Hasara", fmt.Sprintf("%.0f", totalContributions+totalRepayments-totalLoans)})

	return &ReportData{Filename: filename, Path: path, Size: safeStat(path)}, nil
}

func GenerateSummaryReport() (*ReportData, error) {
	EnsureReportDir()

	var activeMembers int64
	var totalMembers int64
	var totalContributions float64
	var totalLoans float64
	var totalRepayments float64
	var outstandingBalance float64
	var pendingLoans int64

	database.DB.Model(&models.Member{}).Where("is_active = TRUE AND deleted_at IS NULL").Count(&activeMembers)
	database.DB.Model(&models.Member{}).Where("deleted_at IS NULL").Count(&totalMembers)
	database.DB.Model(&models.Contribution{}).Where("status = ?", "PAID").Select("COALESCE(SUM(amount),0)").Scan(&totalContributions)
	database.DB.Model(&models.Loan{}).Select("COALESCE(SUM(amount),0)").Scan(&totalLoans)
	database.DB.Model(&models.Repayment{}).Select("COALESCE(SUM(amount),0)").Scan(&totalRepayments)
	database.DB.Model(&models.Loan{}).Where("status = ?", string(models.LoanOutstanding)).Select("COALESCE(SUM(balance_remaining),0)").Scan(&outstandingBalance)
	database.DB.Model(&models.Loan{}).Where("status = ?", string(models.LoanPending)).Count(&pendingLoans)

	filename := fmt.Sprintf("muhtasari_%s.csv", time.Now().Format("2006_01_02"))
	path := filepath.Join(reportDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"Kitengo", "Thamani"})
	w.Write([]string{"Wanachama Hai", fmt.Sprintf("%d", activeMembers)})
	w.Write([]string{"Wanachama Wote", fmt.Sprintf("%d", totalMembers)})
	w.Write([]string{"Jumla ya Michango", fmt.Sprintf("%.0f TZS", totalContributions)})
	w.Write([]string{"Jumla ya Mikopo Iliyotolewa", fmt.Sprintf("%.0f TZS", totalLoans)})
	w.Write([]string{"Jumla ya Marejesho", fmt.Sprintf("%.0f TZS", totalRepayments)})
	w.Write([]string{"Salio la Mikopo Wazi", fmt.Sprintf("%.0f TZS", outstandingBalance)})
	w.Write([]string{"Mikopo Inayosubiri", fmt.Sprintf("%d", pendingLoans)})

	return &ReportData{Filename: filename, Path: path, Size: safeStat(path)}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
