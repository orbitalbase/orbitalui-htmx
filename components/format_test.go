package components

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestCountTextAndDayDurationAreTotal(t *testing.T) {
	counts := map[int64]string{
		0:             "0",
		-1:            "-1",
		-1234:         "-1,234",
		1234567:       "1,234,567",
		math.MaxInt64: "9,223,372,036,854,775,807",
		math.MinInt64: "-9,223,372,036,854,775,808",
	}
	for in, wantText := range counts {
		if got := CountText(in); got != wantText {
			t.Errorf("CountText(%d) = %q, want %q", in, got, wantText)
		}
	}

	days := map[int]string{
		0:    "0 days",
		1:    "1 day",
		-1:   "1 day",
		20:   "20 days",
		-30:  "4 weeks",
		400:  "1 year",
		-800: "2 years",
	}
	for in, wantText := range days {
		if got := DayDuration(in); got != wantText {
			t.Errorf("DayDuration(%d) = %q, want %q", in, got, wantText)
		}
	}
	if got, other := DayDuration(math.MinInt), DayDuration(math.MaxInt); !strings.HasSuffix(got, " years") || !strings.HasSuffix(other, " years") {
		t.Errorf("extreme durations = %q, %q", got, other)
	}
}

func TestPercentIsTotal(t *testing.T) {
	cases := map[string]string{
		"":       "",
		"  ":     "",
		"80":     "80%",
		"80.000": "80%",
		"12.50":  "12.5%",
		"0.0":    "0%",
		".5":     "0.5%",
		"-5":     "-5%",
		"+5.0":   "+5%",
		" 33.3 ": "33.3%",
		"50%":    "50%",
		" 50% ":  "50%",
		"50.00%": "50.00%",
		"abc":    "abc",
		"n/a":    "n/a",
		"1.2.3":  "1.2.3",
		"-":      "-",
		".":      ".",
		"12abc":  "12abc",
	}
	for in, wantText := range cases {
		if got := Percent(in); got != wantText {
			t.Errorf("Percent(%q) = %q, want %q", in, got, wantText)
		}
	}
}

func TestQuantityStripsLeadingZeros(t *testing.T) {
	cases := map[string]string{
		"0000123.10": "123.1",
		"0000123":    "123",
		"000":        "0",
		"0.5000":     "0.5",
		"-007.50":    "-7.5",
		"1000000":    "1,000,000",
		"0001000000": "1,000,000",
		"60000.0000": "60,000",
		"abc":        "abc",
		" 12 ":       "12",
		"":           "",
		"1,234":      "1,234",
	}
	for in, wantText := range cases {
		if got := Quantity(in); got != wantText {
			t.Errorf("Quantity(%q) = %q, want %q", in, got, wantText)
		}
	}
}

func TestRelativeDateSurvivesFarDates(t *testing.T) {
	now := time.Date(2026, time.October, 6, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		t    time.Time
		want string
	}{
		{"same day", time.Date(2026, time.October, 6, 23, 59, 0, 0, time.UTC), "today"},
		{"tomorrow", time.Date(2026, time.October, 7, 0, 0, 0, 0, time.UTC), "tomorrow"},
		{"yesterday", time.Date(2026, time.October, 5, 23, 0, 0, 0, time.UTC), "yesterday"},
		{"12 days", time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC), "in 12 days"},
		{"leap year span", time.Date(2028, time.October, 6, 0, 0, 0, 0, time.UTC), "in 2 years"},
		{"500 years ahead", time.Date(2526, time.October, 6, 0, 0, 0, 0, time.UTC), "in 500 years"},
		{"500 years back", time.Date(1526, time.October, 6, 0, 0, 0, 0, time.UTC), "500 years ago"},
		{"year 1", time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), "2027 years ago"},
		{"year 9999", time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC), "in 7979 years"},
		{"zone is reduced to UTC day", time.Date(2026, time.October, 7, 1, 0, 0, 0, time.FixedZone("x", 3*3600)), "today"},
	}
	for _, tc := range cases {
		if got := RelativeDate(tc.t, now); got != tc.want {
			t.Errorf("%s: RelativeDate = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCivilDayMatchesTimePackage(t *testing.T) {
	for _, d := range []time.Time{
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2000, 2, 29, 12, 0, 0, 0, time.UTC),
		time.Date(1900, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 6, 23, 59, 59, 0, time.UTC),
	} {
		if got, want := civilDay(d), d.Unix()/86400; d.Unix() >= 0 && got != want {
			t.Errorf("civilDay(%s) = %d, want %d", d, got, want)
		}
		if got, want := civilDay(d), floorDiv(d.Unix(), 86400); got != want {
			t.Errorf("civilDay(%s) = %d, want %d", d, got, want)
		}
	}
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}
