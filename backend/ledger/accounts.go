package ledger

// AccountType is the double-entry classification of an account.
type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Income    AccountType = "income"
	Expense   AccountType = "expense"
	Equity    AccountType = "equity"
)

// Valid reports whether t is one of the five fundamental account types.
func (t AccountType) Valid() bool {
	switch t {
	case Asset, Liability, Income, Expense, Equity:
		return true
	}
	return false
}

// DebitPositive reports whether the normal balance side of this type is debit.
// Assets and Expenses increase with debits; Liabilities, Income and Equity
// increase with credits. This weighting is what makes the reconciliation
// invariant ("weighted balances sum to zero") checkable per event.
func (t AccountType) DebitPositive() bool {
	switch t {
	case Asset, Expense:
		return true
	default:
		return false
	}
}

// Chart of accounts — Swahili naming convention (as agreed for Kikundibora).
//
//	name format:                 kind:reference            type
//	────────────────────────────────────────────────────────────────────
//	akiba_ya_mwanachama:{id}     member savings           liability (kikundi kinadaiwa na mwanachama? hapana — kikundi kinamdaiwa? taarifa: kikundi kina DENI kwa mwanachama)
//	dai_la_mkopo:{member}        loan receivable          asset
//	hazina_taslimu               group cash               asset
//	hazina_benki:{acct}          group bank account       asset
//	mapato_ya_riba               interest income          income
//	mapato_ya_faini              fines income             income
//	hifadhi_ya_hasara_ya_mkopo   loan-loss provision      contra-asset (expense-ish reserve)
//	mtaji_wa_kikundi             group capital            equity
//	faida_tulizo                 retained earnings        equity

// Chart constants and name prefixes used when constructing account names.
const (
	PrefixMemberSavings       = "akiba_ya_mwanachama" // liability — kikundi kinadeni mwanachama
	PrefixLoanReceivable      = "dai_la_mkopo"        // asset — mwanachama anadeni kikundi
	NameGroupCash             = "hazina_taslimu"      // asset
	PrefixGroupBank           = "hazina_benki"        // asset
	NameInterestIncome        = "mapato_ya_riba"      // income
	NameFinesIncome           = "mapato_ya_faini"     // income
	NameLoanLossProvision     = "hifadhi_ya_hasara_ya_mkopo" // contra-asset reserve
	NameGroupCapital          = "mtaji_wa_kikundi"    // equity
	NameRetainedEarnings      = "faida_tulizo"        // equity
)

// MemberSavingsName renders the savings account name for a member reference.
func MemberSavingsName(memberRef string) string { return PrefixMemberSavings + ":" + memberRef }

// LoanReceivableName renders the loan-receivable account name for a member reference.
func LoanReceivableName(memberRef string) string { return PrefixLoanReceivable + ":" + memberRef }

// GroupBankName renders a bank account name for a bank-account reference.
func GroupBankName(bankRef string) string { return PrefixGroupBank + ":" + bankRef }
