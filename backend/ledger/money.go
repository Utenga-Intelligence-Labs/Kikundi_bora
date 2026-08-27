package ledger

import (
	"fmt"
	"math/big"
)

// CurrencyTZS is the single base currency for Kikundibora's ledger (v1).
const CurrencyTZS = "TZS"

// SupportedCurrencies lists every currency accepted by the ledger core.
// v1 intentionally supports only TZS; multi-currency FX is out of scope.
var SupportedCurrencies = map[string]bool{
	CurrencyTZS: true,
}

// Money is an exact monetary amount held in integer minor units (senti)
// plus its ISO-4217 currency code. Never use float64 for money.
type Money struct {
	AmountMinor int64  // minor units, e.g. cents-of-TZS (senti)
	Currency    string // always "TZS" in v1
}

// NewMoney constructs a validated Money value.
func NewMoney(amountMinor int64, currency string) (Money, error) {
	if !SupportedCurrencies[currency] {
		return Money{}, fmt.Errorf("%w: %q", ErrUnsupportedCurrency, currency)
	}
	return Money{AmountMinor: amountMinor, Currency: currency}, nil
}

// NewTZS constructs a TZS Money value.
func NewTZS(amountMinor int64) Money {
	return Money{AmountMinor: amountMinor, Currency: CurrencyTZS}
}

// Add returns a + b, erroring if currencies differ.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{AmountMinor: m.AmountMinor + other.AmountMinor, Currency: m.Currency}, nil
}

// Negate returns -m.
func (m Money) Negate() Money { return Money{AmountMinor: -m.AmountMinor, Currency: m.Currency} }

// IsZero reports whether the amount is exactly zero.
func (m Money) IsZero() bool { return m.AmountMinor == 0 }

// String renders a stable, deterministic decimal string, e.g. "-1500" for
// TZS 1,500 with a 0-decimal convention (minor units are still stored).
func (m Money) String() string {
	v := big.NewInt(m.AmountMinor)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(moneyScale(m.Currency)), nil)
	neg := false
	if v.Sign() < 0 {
		neg = true
		v.Neg(v)
	}
	whole, frac := new(big.Int).QuoRem(v, scale, new(big.Int))
	s := whole.String()
	if scale.Cmp(big.NewInt(1)) > 0 {
		fs := frac.String()
		for len(fs) < int(moneyScale(m.Currency)) {
			fs = "0" + fs
		}
		s += "." + fs
	}
	if neg && s != "0" {
		return "-" + s
	}
	return s
}

func moneyScale(currency string) int64 {
	switch currency {
	case CurrencyTZS:
		return 0 // TZS has no subunit in practice; amounts are whole shillings
	default:
		return 2
	}
}
