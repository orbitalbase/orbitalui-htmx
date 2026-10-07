// Meter — a thin horizontal measure (a score, a fill level, a quota) with an
// animated fill. Unlike ProgressBar it has no label row and reads as a
// property of a record rather than the status of a task.
package components

import "strconv"

// MeterOpts configures Meter.
//
//	Value — 0..100; clamped. aria-valuenow is exact and the fill width is the
//	    nearest 1% class, so no inline style is needed (CSP safe).
//	Tone — "" reads the primary-action colour; brand, accent, info, success,
//	    warning, danger and ai (the AI gradient) tint the fill.
//	Size — "" (6px), "sm" (4px) or "lg" (8px).
//	Label — the accessible name ("Profile strength").
type MeterOpts struct {
	Value int
	Tone  Tone
	Size  string
	Label string
	Class string
}

func meterClass(o MeterOpts) string {
	tone := ""
	if m := visualTone(o.Tone); m != "" {
		tone = "ui-meter--" + m
	}
	size := ""
	if o.Size == "sm" || o.Size == "lg" {
		size = "ui-meter--" + o.Size
	}
	return joinClass("ui-meter", tone, size, o.Class)
}

func meterFillClass(v int) string {
	return "ui-meter__fill ui-meter__fill--" + strconv.Itoa(clampPercent(v))
}
