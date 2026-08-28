package services

import (
	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

// TreasuryService handles treasury balance calculations
type TreasuryService struct{}

func NewTreasuryService() *TreasuryService {
	return &TreasuryService{}
}

// HazinaBalance represents the treasury balance breakdown
type HazinaBalance struct {
	TotalContributions   decimal.Decimal `json:"total_contributions"`
	TotalRepayments      decimal.Decimal `json:"total_repayments"`
	TotalDisbursed       decimal.Decimal `json:"total_disbursed"`
	AvailableBalance     decimal.Decimal `json:"available_balance"`
}

// CalculateHazinaBalance calculates the available treasury balance
// Formula: (Confirmed Contributions) + (Loan Repayments) - (Disbursed Loans)
func (s *TreasuryService) CalculateHazinaBalance() (*HazinaBalance, error) {
	var balance HazinaBalance

	// 1. Total confirmed contributions from NEW MemberContribution table (Phase 4)
	var totalContributions decimal.Decimal
	err := database.DB.Model(&models.MemberContribution{}).
		Where("status = ?", models.ContributionConfirmed).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalContributions).Error
	if err != nil {
		return nil, err
	}

	// Also include legacy contributions from OLD Contribution table
	var legacyContributions decimal.Decimal
	err = database.DB.Model(&models.Contribution{}).
		Where("status = ?", "PAID").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&legacyContributions).Error
	if err != nil {
		return nil, err
	}
	balance.TotalContributions = totalContributions.Add(legacyContributions)

	// 2. Total loan repayments
	var totalRepayments decimal.Decimal
	err = database.DB.Model(&models.Repayment{}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRepayments).Error
	if err != nil {
		return nil, err
	}
	balance.TotalRepayments = totalRepayments

	// 3. Total disbursed loans (OUTSTANDING or CLOSED loans that were actually disbursed)
	var totalDisbursed decimal.Decimal
	err = database.DB.Model(&models.Loan{}).
		Where("status IN ?", []models.LoanStatus{models.LoanOutstanding, models.LoanClosed}).
		Select("COALESCE(SUM(approved_amount), 0)").
		Scan(&totalDisbursed).Error
	if err != nil {
		return nil, err
	}
	balance.TotalDisbursed = totalDisbursed

	// 4. Calculate available balance
	balance.AvailableBalance = balance.TotalContributions.Add(balance.TotalRepayments).Sub(balance.TotalDisbursed)

	return &balance, nil
}

// CanAffordLoan checks if the treasury can afford a loan of the given amount
func (s *TreasuryService) CanAffordLoan(amount decimal.Decimal) (bool, decimal.Decimal, error) {
	balance, err := s.CalculateHazinaBalance()
	if err != nil {
		return false, decimal.Zero, err
	}

	if amount.GreaterThan(balance.AvailableBalance) {
		return false, balance.AvailableBalance, nil
	}

	return true, balance.AvailableBalance, nil
}
