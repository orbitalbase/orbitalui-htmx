package components

import (
	"math/big"
	"strings"
	"testing"
)

func bigRatFromParts(t *testing.T, n, d int64) *big.Rat {
	t.Helper()
	return big.NewRat(n, d)
}

// ── parsing ────────────────────────────────────────────────────────────────

func TestParseAmountOK(t *testing.T) {
	cases := map[string]string{
		// input → canonical exact string
		"0":              "0",
		"1,234":          "1234",
		"1,234,567":      "1234567",
		"12.5":           "12.5",
		".5":             "0.5",
		"0.5":            "0.5",
		"+5":             "5",
		"-5":             "-5",
		"-1,234.50":      "-1234.5",
		"1000.000000001": "1000.000000001",
		"0.000000001":    "0.000000001",
	}
	for in, want := range cases {
		a, err := ParseAmount(in)
		if err != nil {
			t.Errorf("ParseAmount(%q) error: %v", in, err)
			continue
		}
		if got := a.String(); got != want {
			t.Errorf("ParseAmount(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestParseAmountEmptyIsZero(t *testing.T) {
	a, err := ParseAmount("  ")
	if err != nil || !a.IsZero() {
		t.Fatalf("ParseAmount(\"\") = %v, %v; want zero, nil", a, err)
	}
}

func TestParseAmountRejects(t *testing.T) {
	bad := []string{
		"abc", "1,23", "1,234,56", ",123", "123,", "1.2.3",
		"1.2345678901", // 10 decimal places > numeric(24,9) scale
		"1e3",          // exponent is not user input
		"--5", "1 000",
	}
	for _, in := range bad {
		if _, err := ParseAmount(in); err == nil {
			t.Errorf("ParseAmount(%q) should error", in)
		}
	}
}

func TestAmountFromStringDBText(t *testing.T) {
	// numeric(24,9) output keeps trailing zeros to the column scale.
	cases := map[string]string{
		"1234.500000000": "1234.5",
		"-0.000000001":   "-0.000000001",
		"0":              "0",
		"123":            "123",
	}
	for in, want := range cases {
		a, err := AmountFromString(in)
		if err != nil {
			t.Errorf("AmountFromString(%q): %v", in, err)
			continue
		}
		if a.String() != want {
			t.Errorf("AmountFromString(%q) = %s, want %s", in, a.String(), want)
		}
	}
	// Views can widen scale or emit exponents — both decode exactly.
	a, err := AmountFromString("0.12345678901")
	if err != nil || a.String() != "0.12345678901" {
		t.Errorf("wide-scale: %v %v", a, err)
	}
	a, err = AmountFromString("1.5e3")
	if err != nil || a.String() != "1500" {
		t.Errorf("exponent: %v %v", a, err)
	}
	if _, err := AmountFromString(""); err == nil {
		t.Error("AmountFromString(\"\") should error")
	}
}

// ── arithmetic ─────────────────────────────────────────────────────────────

func amt(t *testing.T, s string) Amount {
	t.Helper()
	a, err := ParseAmount(s)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", s, err)
	}
	return a
}

func TestAmountArithmetic(t *testing.T) {
	if got := amt(t, "1.5").Add(amt(t, "2.25")); got.String() != "3.75" {
		t.Errorf("add = %s", got)
	}
	if got := amt(t, "1.5").Sub(amt(t, "2.25")); got.String() != "-0.75" {
		t.Errorf("sub = %s", got)
	}
	if got := amt(t, "-1.5").Neg(); got.String() != "1.5" {
		t.Errorf("neg = %s", got)
	}
	if got := amt(t, "-1.5").Abs(); got.String() != "1.5" {
		t.Errorf("abs = %s", got)
	}
	if got := amt(t, "2.5").Cmp(amt(t, "2.50")); got != 0 {
		t.Errorf("cmp equal = %d", got)
	}
	if got := amt(t, "3").Cmp(amt(t, "2")); got != 1 {
		t.Errorf("cmp greater = %d", got)
	}
	if !amt(t, "0.000").IsZero() || amt(t, "0.001").IsZero() {
		t.Error("IsZero wrong")
	}
	if got := Sum([]Amount{amt(t, "1"), amt(t, "2.5"), amt(t, "-0.5")}); got.String() != "3" {
		t.Errorf("sum = %s", got)
	}
	if got := Sum(nil); !got.IsZero() {
		t.Errorf("empty sum = %s", got)
	}
}

// ── display ────────────────────────────────────────────────────────────────

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		amount string
		code   string
		want   string
	}{
		{"1234.5", "USD", "1,234.50"},
		{"1234.5", "usd", "1,234.50"}, // case-insensitive code
		{"1234.5", "", "1,234.50"},    // unknown → 2 decimals
		{"0", "USD", "0.00"},
		{"-1234.5", "USD", "-1,234.50"},
		{"1234.567", "USD", "1,234.57"}, // half-away rounding
		{"0.0049", "USD", "0.00"},       // below half a cent
		{"0.005", "USD", "0.01"},        // half a cent rounds up
		{"-0.005", "USD", "-0.01"},      // symmetric
		{"1234.5", "JPY", "1,235"},      // zero decimals
		{"1234.5", "VND", "1,235"},      // zero decimals
		{"1234.5", "BHD", "1,234.500"},  // three decimals
		{"999999999.99", "USD", "999,999,999.99"},
		{"12345678901234567890.5", "USD", "12,345,678,901,234,567,890.50"},
	}
	for _, c := range cases {
		a := amt(t, c.amount)
		if got := FormatMoney(a, c.code); got != c.want {
			t.Errorf("FormatMoney(%s, %q) = %q, want %q", c.amount, c.code, got, c.want)
		}
	}
}

func TestFormatMoneyN(t *testing.T) {
	// Unbounded-scale DB text (view arithmetic) rounds at the display scale.
	a, err := AmountFromString("1.2345678905")
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatMoneyN(a, 9); got != "1.234567891" { // rounds at 9dp
		t.Errorf("9dp = %q", got)
	}
	if got := FormatMoneyN(a, 0); got != "1" {
		t.Errorf("0dp = %q", got)
	}
}

func TestFormatSigned(t *testing.T) {
	cases := map[string]string{
		"1234.5": "1,234.50 Dr",
		"-5":     "5.00 Cr",
		"0":      "0.00",
	}
	for in, want := range cases {
		if got := FormatSigned(amt(t, in), "USD"); got != want {
			t.Errorf("FormatSigned(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestCurrencyDecimalsDefaults(t *testing.T) {
	if got := CurrencyDecimals("XXX"); got != 2 {
		t.Errorf("unknown code = %d, want 2", got)
	}
	if got := CurrencyDecimals("JPY"); got != 0 {
		t.Errorf("JPY = %d, want 0", got)
	}
	if got := CurrencyDecimals("KWD"); got != 3 {
		t.Errorf("KWD = %d, want 3", got)
	}
}

func TestAmountStringNonTerminating(t *testing.T) {
	// 1/3 has no finite decimal — String falls back to a/b form.
	r := AmountFromRat(bigRatFromParts(t, 1, 3))
	s := r.String()
	if !strings.Contains(s, "/") {
		t.Errorf("1/3 should render as a fraction, got %q", s)
	}
}
