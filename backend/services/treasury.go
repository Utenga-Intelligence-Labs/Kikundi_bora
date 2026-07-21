package services

import (
	"kikundibora/database"
	"kikundibora/models"
)

// TreasuryService handles treasury balance calculations
type TreasuryService struct{}

func NewTreasuryService() *TreasuryService {
	return &TreasuryService{}
}

// HazinaBalance represents the treasury balance breakdown
type HazinaBalance struct {
	TotalContributions   float64 `json:"total_contributions"`
	TotalRepayments      float64 `json:"total_repayments"`
	TotalDisbursed       float64 `json:"total_disbursed"`
	AvailableBalance     float64 `json:"available_balance"`
}

// CalculateHazinaBalance calculates the available treasury balance
// Formula: (Confirmed Contributions) + (Loan Repayments) - (Disbursed Loans)
func (s *TreasuryService) CalculateHazinaBalance() (*HazinaBalance, error) {
	var balance HazinaBalance

	// 1. Total confirmed contributions
	var totalContributions float64
	err := database.DB.Model(&models.Contribution{}).
		Where("status = ?", "PAID").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalContributions).Error
	if err != nil {
		return nil, err
	}
	balance.TotalContributions = totalContributions

	// 2. Total loan repayments
	var totalRepayments float64
	err = database.DB.Model(&models.Repayment{}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRepayments).Error
	if err != nil {
		return nil, err
	}
	balance.TotalRepayments = totalRepayments

	// 3. Total disbursed loans (OUTSTANDING or CLOSED loans that were actually disbursed)
	var totalDisbursed float64
	err = database.DB.Model(&models.Loan{}).
		Where("status IN ?", []models.LoanStatus{models.LoanOutstanding, models.LoanClosed}).
		Select("COALESCE(SUM(approved_amount), 0)").
		Scan(&totalDisbursed).Error
	if err != nil {
		return nil, err
	}
	balance.TotalDisbursed = totalDisbursed

	// 4. Calculate available balance
	balance.AvailableBalance = balance.TotalContributions + balance.TotalRepayments - balance.TotalDisbursed

	return &balance, nil
}

// CanAffordLoan checks if the treasury can afford a loan of the given amount
func (s *TreasuryService) CanAffordLoan(amount float64) (bool, float64, error) {
	balance, err := s.CalculateHazinaBalance()
	if err != nil {
		return false, 0, err
	}

	if amount > balance.AvailableBalance {
		return false, balance.AvailableBalance, nil
	}

	return true, balance.AvailableBalance, nil
}
