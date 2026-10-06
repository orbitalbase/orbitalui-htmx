// Small number/format helpers shared by components (templ expressions must
// be strings, so counts and page numbers are converted here) plus the
// presentation helpers every record workspace reimplemented: a stored date
// read back as a readable day or a relative phrase, a stored decimal grouped
// for a register column, and that decimal paired with its currency code.
//
// Every string-parsing helper is total: a value it cannot read is returned
// trimmed but otherwise unchanged, so support never loses the stored text.
package components

import (
	"strconv"
	"strings"
	"time"
)

// CountText renders an int64 count with thousands separators ("1,234") — the
// tab-bar count chips and table summaries.
func CountText(n int64) string {
	if n < 0 {
		return "-" + CountText(-n)
	}
	return groupInt(strconv.FormatInt(n, 10))
}

// IntText renders an int for markup (page numbers, aria values).
func IntText(n int) string { return strconv.Itoa(n) }

// dateLayouts are the stored date shapes a Go service hands a template: a
// plain ISO date, an ISO timestamp with or without a zone and with or without
// a 'T', PostgreSQL's text form of timestamptz ("2026-10-06
// 01:44:34.745836+00", hour-only offset), and the day-first strings older
// fixtures carry.
var dateLayouts = []string{
	"2006-01-02",
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999-07",
	"2006-01-02 15:04:05-07",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2 Jan 2006",
	"2 January 2006",
}

// ParseDate reads any of the stored date or timestamp shapes. The second
// result reports whether the value was understood, so a caller can fall back
// to the stored text rather than invent a date.
func ParseDate(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range dateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// DateText renders a stored date as the readable day a register shows
// ("4 Mar 2024"). An unparseable value comes back trimmed.
func DateText(raw string) string {
	if parsed, ok := ParseDate(raw); ok {
		return parsed.Format("2 Jan 2006")
	}
	return strings.TrimSpace(raw)
}

// DateTimeText renders a stored timestamp as the day plus the time of day
// ("4 Mar 2024, 3:04 pm") — the record form, where the clock matters.
func DateTimeText(raw string) string {
	if parsed, ok := ParseDate(raw); ok {
		return parsed.Format("2 Jan 2006, 3:04 pm")
	}
	return strings.TrimSpace(raw)
}

// RelativeDate phrases the calendar distance from now to t the way a deadline
// list speaks: "today", "tomorrow", "in 12 days", "3 months ago". Both
// instants are reduced to their UTC calendar day first, so a timestamp late
// in the day is still "today" and never rounds to "tomorrow".
//
// Pass the clock explicitly (the caller's `time.Now()`) so a page's output is
// a pure function of its inputs and stays testable.
func RelativeDate(t time.Time, now time.Time) string {
	days := calendarDaysBetween(now, t)
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	case days == -1:
		return "yesterday"
	case days > 0:
		return "in " + DayDuration(days)
	default:
		return DayDuration(-days) + " ago"
	}
}

// RelativeDateText is RelativeDate over a stored string. It returns "" when
// the value is not a date, so a caller can show the absolute form alone
// rather than a phrase it cannot justify.
func RelativeDateText(raw string, now time.Time) string {
	parsed, ok := ParseDate(raw)
	if !ok {
		return ""
	}
	return RelativeDate(parsed, now)
}

// calendarDaysBetween counts whole days from now's UTC calendar day to t's.
func calendarDaysBetween(now, t time.Time) int {
	from := now.UTC()
	to := t.UTC()
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(end.Sub(start) / (24 * time.Hour))
}

// DayDuration spells a non-negative day count at the coarsest honest unit:
// days under three weeks, then weeks, months and years.
func DayDuration(days int) string {
	switch {
	case days < 0:
		return DayDuration(-days)
	case days < 21:
		return Plural(days, "day")
	case days < 60:
		return Plural((days+3)/7, "week")
	case days < 365:
		return Plural((days+15)/30, "month")
	default:
		return Plural((days+182)/365, "year")
	}
}

// Plural renders a count with its noun, pluralised by appending "s": the
// shape "3 days" / "1 day" that every summary line needs.
func Plural(n int, noun string) string {
	if n == 1 || n == -1 {
		return strconv.Itoa(n) + " " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// Count renders "3 deadlines" / "1 deadline" for a toolbar summary, with the
// plural spelled out so irregular nouns ("entries", "people") read correctly.
func Count(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return CountText(int64(n)) + " " + plural
}

// Quantity renders a stored numeric string for a register column: trailing
// zeros in the fraction are dropped and thousands are grouped, so
// "60000.0000" reads as "60,000" and "1.5000" as "1.5". A value that is not a
// plain decimal is returned trimmed, never mangled.
func Quantity(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	sign := ""
	if strings.HasPrefix(value, "-") {
		sign, value = "-", value[1:]
	}
	whole, fraction, _ := strings.Cut(value, ".")
	if whole == "" {
		return strings.TrimSpace(raw)
	}
	for _, r := range whole {
		if r < '0' || r > '9' {
			return strings.TrimSpace(raw)
		}
	}
	for _, r := range fraction {
		if r < '0' || r > '9' {
			return strings.TrimSpace(raw)
		}
	}
	grouped := groupInt(whole)
	if fraction = strings.TrimRight(fraction, "0"); fraction == "" {
		return sign + grouped
	}
	return sign + grouped + "." + fraction
}

// GroupedAmount pairs a currency code with a grouped amount, trimming
// trailing fractional zeros the way Quantity does ("37500.00" → "MYR 37,500",
// "2.50" → "MYR 2.5"). An empty amount yields "" so a caller can choose its
// own placeholder wording; an empty currency yields the figure alone.
//
// Use it for unit prices and terms where the stored precision is noise. For
// an amount that must show the currency's own decimals ("MYR 1,500.50"),
// parse it with AmountFromString and render it with FormatMoney.
func GroupedAmount(currency, raw string) string {
	amount := Quantity(raw)
	if amount == "" {
		return ""
	}
	if currency = strings.TrimSpace(currency); currency == "" {
		return amount
	}
	return currency + " " + amount
}

// Percent trims a stored percentage's trailing zeros and appends the sign:
// "80.000" → "80%". An empty value yields "".
func Percent(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.Contains(value, ".") {
		value = strings.TrimSuffix(strings.TrimRight(value, "0"), ".")
	}
	return value + "%"
}
