// Money — the exact decimal core of the UI. The ledger stores money as
// numeric(24,9) and every derived view is exact; this package mirrors that
// with Amount (a *big.Rat). Parsing, sums and formatting never pass through
// float64 — the only floats anywhere are decorative ratios (BalanceTool's
// bar widths) computed from an exact pair at render time.
//
// The page-facing API is re-exported by internal/ui as ui.Amount /
// ui.ParseAmount / ui.FormatMoney … (ui/money.go) — use the ui names in
// handlers; this file is the implementation and what templ components type
// against.
package components

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Amount is an exact decimal quantity. The zero value is zero (a nil *big.Rat
// is treated as 0/1), so `var a Amount` is safe. Amounts are immutable: Add,
// Sub, Neg, Abs return new values.
type Amount struct {
	r *big.Rat
}

// MaxInputDecimals is the most fractional digits ParseAmount accepts — the
// scale of numeric(24,9). Form inputs past this are rejected with a friendly
// error instead of being silently rounded.
const MaxInputDecimals = 9

// ErrAmount is the shared error family for money parsing failures; the
// wrapped messages are stable user copy (docs/ui-v1-plan.md §9) suitable for
// FieldError.
var ErrAmount = errors.New("invalid amount")

func rat(a Amount) *big.Rat {
	if a.r == nil {
		return new(big.Rat)
	}
	return a.r
}

// ZeroAmount returns the additive identity.
func ZeroAmount() Amount { return Amount{r: new(big.Rat)} }

// AmountFromRat wraps an exact *big.Rat into an Amount (used by code paths
// that already computed exact rationals). It never rounds.
func AmountFromRat(r *big.Rat) Amount { return Amount{r: new(big.Rat).Set(r)} }

// Rat returns a copy of the wrapped *big.Rat (Amounts are immutable).
func (a Amount) Rat() *big.Rat { return new(big.Rat).Set(rat(a)) }

// Add returns a + b.
func (a Amount) Add(b Amount) Amount { return Amount{r: new(big.Rat).Add(rat(a), rat(b))} }

// Sub returns a - b.
func (a Amount) Sub(b Amount) Amount { return Amount{r: new(big.Rat).Sub(rat(a), rat(b))} }

// Neg returns -a.
func (a Amount) Neg() Amount { return Amount{r: new(big.Rat).Neg(rat(a))} }

// Abs returns |a|.
func (a Amount) Abs() Amount { return Amount{r: new(big.Rat).Abs(rat(a))} }

// Cmp compares a with b: -1, 0 or +1 when a < b, a == b, a > b.
func (a Amount) Cmp(b Amount) int { return rat(a).Cmp(rat(b)) }

// Sign returns -1, 0 or +1 per a's sign.
func (a Amount) Sign() int { return rat(a).Sign() }

// IsZero reports whether a is exactly zero.
func (a Amount) IsZero() bool { return rat(a).Sign() == 0 }

// IsNegative reports whether a < 0.
func (a Amount) IsNegative() bool { return rat(a).Sign() < 0 }

// String renders a's exact canonical decimal ("1234.5", "-0.25") with no
// padding — for debugging and canonical round-trips, not display (display
// goes through FormatMoney).
func (a Amount) String() string {
	r := rat(a)
	if r.IsInt() {
		return r.Num().String()
	}
	// The smallest scale that represents a exactly: denominator 2^k·5^m
	// terminates at max(k, m) digits.
	d := new(big.Int).Set(r.Denom())
	two, five := 0, 0
	for d.Bit(0) == 0 {
		d.Rsh(d, 1)
		two++
	}
	five = countFactor5(d)
	if five < 0 {
		// Non-terminating decimal — fall back to a/b form.
		return r.String()
	}
	scale := two
	if five > scale {
		scale = five
	}
	return r.FloatString(scale)
}

// countFactor5 strips factors of 5 from d, returning how many were removed
// or -1 if anything else remains (non-terminating decimal).
func countFactor5(d *big.Int) int {
	n := 0
	five := big.NewInt(5)
	rem := new(big.Int)
	for {
		q, m := new(big.Int).QuoRem(d, five, rem)
		if m.Sign() != 0 {
			if d.Cmp(big.NewInt(1)) == 0 {
				return n
			}
			return -1
		}
		d = q
		n++
	}
}

// Sum returns the exact total of amounts (zero for an empty slice).
func Sum(amounts []Amount) Amount {
	total := ZeroAmount()
	for _, a := range amounts {
		total = total.Add(a)
	}
	return total
}

// ── parsing ────────────────────────────────────────────────────────────────

// errAmount builds a stable user-facing parse error.
func errAmount(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrAmount, fmt.Sprintf(format, args...))
}

// ParseAmount parses user-typed money text: optional leading +/-, integer
// part with optional thousands commas (every group after the first must be
// exactly 3 digits), optional '.' fraction of at most MaxInputDecimals
// digits. An empty string parses to zero (callers decide whether a field is
// required). Exact — never rounded, never float.
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ZeroAmount(), nil
	}
	return parseDecimal(s, MaxInputDecimals, true)
}

// AmountFromString parses DB numeric text — the result of SELECTing a
// numeric(24,9) column or one of the derived views. Permissive: optional
// sign, any number of fractional digits (view expressions can widen scale)
// and an optional decimal exponent (e notation is decoded exactly, digit by
// digit). Empty input is an error. Exact — never float.
func AmountFromString(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Amount{}, errAmount("The database returned an empty amount.")
	}
	return parseDecimal(s, -1, false)
}

// parseDecimal is the shared exact decimal scanner. maxFrac < 0 allows any
// scale; strictCommas validates thousands grouping (user input). Returns an
// error whose message is safe to show next to a field.
func parseDecimal(s string, maxFrac int, strictCommas bool) (Amount, error) {
	body := s
	neg := false
	switch body[0] {
	case '+':
		body = body[1:]
	case '-':
		neg = true
		body = body[1:]
	}
	if body == "" {
		return Amount{}, errAmount("Enter an amount like 1,234.56.")
	}

	// Optional exponent (DB path): split mantissa | exponent, then apply the
	// shift when building the rational.
	mant := body
	exp := 0
	if i := strings.IndexAny(body, "eE"); i >= 0 {
		if maxFrac >= 0 {
			return Amount{}, errAmount("Enter an amount like 1,234.56.")
		}
		mant = body[:i]
		expPart := body[i+1:]
		expSign := 1
		if strings.HasPrefix(expPart, "+") {
			expPart = expPart[1:]
		} else if strings.HasPrefix(expPart, "-") {
			expSign = -1
			expPart = expPart[1:]
		}
		if expPart == "" {
			return Amount{}, errAmount("The database returned an unreadable amount.")
		}
		for _, c := range expPart {
			if c < '0' || c > '9' {
				return Amount{}, errAmount("The database returned an unreadable amount.")
			}
		}
		exp = expSign * atoi(expPart)
	}

	intPart, fracPart, ok := strings.Cut(mant, ".")
	if !ok {
		fracPart = ""
	} else if strings.Contains(fracPart, ".") {
		return Amount{}, errAmount("Enter an amount like 1,234.56.")
	}

	// Integer part: digits with optional thousands grouping.
	var intDigits strings.Builder
	if intPart != "" {
		groups := strings.Split(intPart, ",")
		if strictCommas && len(groups) > 1 {
			// Grouping only matters when commas are present; a bare run of
			// digits is always fine.
			for gi, g := range groups {
				if g == "" {
					return Amount{}, errAmount("Enter an amount like 1,234.56.")
				}
				if gi == 0 {
					if len(g) > 3 {
						return Amount{}, errAmount("Enter an amount like 1,234.56.")
					}
				} else if len(g) != 3 {
					return Amount{}, errAmount("Enter an amount like 1,234.56.")
				}
			}
		}
		for _, g := range groups {
			if g == "" {
				return Amount{}, errAmount("Enter an amount like 1,234.56.")
			}
			for _, c := range g {
				if c < '0' || c > '9' {
					return Amount{}, errAmount("Enter an amount like 1,234.56.")
				}
			}
			intDigits.WriteString(g)
		}
	}
	// Non-strict path (DB text never carries commas — reject them if present
	// so junk can't slip through as grouping).
	if !strictCommas && strings.Contains(intPart, ",") {
		return Amount{}, errAmount("The database returned an unreadable amount.")
	}
	if intDigits.Len() > 1000 {
		return Amount{}, errAmount("That amount is too long.")
	}

	// Fraction: digits only, capped on the user path.
	if len(fracPart) > 1000 {
		return Amount{}, errAmount("That amount is too long.")
	}
	if maxFrac >= 0 && len(fracPart) > maxFrac {
		return Amount{}, errAmount("Enter at most %d decimal places.", maxFrac)
	}
	for _, c := range fracPart {
		if c < '0' || c > '9' {
			return Amount{}, errAmount("Enter an amount like 1,234.56.")
		}
	}

	// Exact rational: digits × 10^(exp−fracLen).
	digits := strings.TrimLeft(intDigits.String()+fracPart, "0")
	if digits == "" {
		return ZeroAmount(), nil
	}
	num := new(big.Int)
	if _, ok := num.SetString(digits, 10); !ok {
		return Amount{}, errAmount("The database returned an unreadable amount.")
	}
	if neg {
		num.Neg(num)
	}
	scale := len(fracPart) - exp
	switch {
	case scale > 0:
		den := pow10(scale)
		return Amount{r: new(big.Rat).SetFrac(num, den)}, nil
	case scale == 0:
		return Amount{r: new(big.Rat).SetInt(num)}, nil
	default:
		num.Mul(num, pow10(-scale))
		return Amount{r: new(big.Rat).SetInt(num)}, nil
	}
}

// pow10 returns 10^n for n ≥ 0 (exact).
func pow10(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

// atoi parses an unsigned digit string (already validated).
func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// ── display ────────────────────────────────────────────────────────────────

// currencyDecimals is the compiled decimals table for the currency codes the
// ledger seeds (db/migrations/00003, 00014) plus common 3-decimal Gulf
// codes. Unknown codes fall back to 2 — and a book whose currency row
// carries a non-standard decimals value can format exactly through
// Deps.Currencies (ui/currencies.go). Keys are upper-case 3-letter codes.
var currencyDecimals = map[string]int{
	"USD": 2, "EUR": 2, "GBP": 2, "CHF": 2, "CAD": 2, "AUD": 2,
	"NZD": 2, "CNY": 2, "HKD": 2, "SGD": 2, "INR": 2, "BRL": 2,
	"MXN": 2, "SEK": 2, "NOK": 2, "DKK": 2, "PLN": 2, "ZAR": 2,
	"MYR": 2, "THB": 2, "PHP": 2, "TWD": 2, "AED": 2, "SAR": 2,
	"JPY": 0, "KRW": 0, "VND": 0, "IDR": 0,
	"BHD": 3, "KWD": 3, "OMR": 3,
}

// CurrencyDecimals returns the fixed decimal places for a currency code
// (compiled ISO table; 2 for unknown or empty codes).
func CurrencyDecimals(code string) int {
	if d, ok := currencyDecimals[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return d
	}
	return 2
}

// FormatMoney renders a exactly, thousands-separated, with the currency's
// fixed decimals (CurrencyDecimals): FormatMoney(1234.5, "USD") ==
// "1,234.50". The fraction is rounded half-away-from-zero to the display
// scale — display rounding only; the Amount itself is untouched.
func FormatMoney(a Amount, code string) string {
	return FormatMoneyN(a, CurrencyDecimals(code))
}

// FormatMoneyN renders a with an explicit decimals count (DB-driven display
// for custom currencies via Deps.Currencies). decimals is clamped to 0..9.
func FormatMoneyN(a Amount, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	if decimals > MaxInputDecimals {
		decimals = MaxInputDecimals
	}
	sign := ""
	r := rat(a)
	if r.Sign() < 0 {
		sign = "-"
		r = new(big.Rat).Abs(r)
	}
	scale := pow10(decimals)
	num := new(big.Int).Mul(r.Num(), scale)
	q, rem := new(big.Int).QuoRem(num, r.Denom(), new(big.Int))
	// Round half away from zero on the first dropped digit.
	if new(big.Int).Lsh(rem, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return sign + groupFixed(q, decimals)
}

// groupFixed renders the scaled integer q (value × 10^decimals) with
// thousands grouping and a fixed decimal point: "1,234.50".
func groupFixed(q *big.Int, decimals int) string {
	s := q.String()
	if decimals == 0 {
		return groupInt(s)
	}
	for len(s) <= decimals {
		s = "0" + s
	}
	intPart := s[:len(s)-decimals]
	frac := s[len(s)-decimals:]
	if intPart == "" {
		intPart = "0"
	}
	return groupInt(intPart) + "." + frac
}

// groupInt inserts thousands commas into an unsigned digit string.
func groupInt(s string) string {
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	b.WriteString(s[:lead])
	for i := lead; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatSigned renders a dr/cr-column value: positive amounts are debits,
// negative amounts are credits (the trial-balance convention — a credit
// balance is a negative debit balance). Zero renders without a side.
//
//	FormatSigned(1234.5, "USD") == "1,234.50 Dr"
//	FormatSigned(-5, "USD")     == "5.00 Cr"
func FormatSigned(a Amount, code string) string {
	switch {
	case a.IsNegative():
		return FormatMoney(a.Abs(), code) + " Cr"
	case a.IsZero():
		return FormatMoney(a, code)
	default:
		return FormatMoney(a, code) + " Dr"
	}
}
