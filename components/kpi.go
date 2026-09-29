// KPITile types + helpers (kpi.templ) — the stat-card tile used on the
// overview and report headers (new in v1; markup matches the dashboard
// .kpi-tile classes of phase 1).
package components

import "github.com/a-h/templ"

// KPITileOpts: Tone tints the value (danger for shortfalls/negative);
// Hint is the muted sub-line; Icon renders a small leading glyph; Href wraps
// the tile in a link to a drill-down page.
type KPITileOpts struct {
	Tone  Tone
	Hint  string
	Icon  string // icon-set name (icons.go)
	Href  templ.SafeURL
	Class string
}

// kpiClass joins the base tile class + extras.
func kpiClass(o KPITileOpts) string {
	if o.Class == "" {
		return "kpi-tile"
	}
	return "kpi-tile " + o.Class
}

// kpiValueClass appends the tone modifier to the mono value.
func kpiValueClass(o KPITileOpts) string {
	c := "kpi-tile__value"
	if m := o.Tone.KPIValueClass(); m != "" {
		c += " " + m
	}
	return c
}
