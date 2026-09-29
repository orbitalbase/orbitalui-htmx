// Small number/format helpers shared by components (templ expressions must
// be strings, so counts and page numbers are converted here).
package components

import "strconv"

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
