// Tone — the app's semantic colour vocabulary. Status chips, alerts and
// verdicts map a Tone to CSS modifier classes (ui-status--*, ui-alert--*);
// colours are never used alone: every tinted surface carries a text label
// (docs/ui-v1-plan.md §4/§9, docs/ui-components.md §5). Domain label maps
// (enum → label + tone) stay in the domain page packages, not here.
package components

// Tone is one semantic status colour. The empty value is ToneNeutral.
type Tone string

const (
	ToneNeutral Tone = "neutral" // base chip — no tinted modifier
	ToneInfo    Tone = "info"
	ToneWarning Tone = "warning"
	ToneDanger  Tone = "danger"
	ToneSuccess Tone = "success" // accounting-positive states only
	ToneAI      Tone = "ai"      // iris — reserved for AI provenance
	// ToneBrand and ToneAccent are the two identity colours. They tint
	// badges, alerts and card edges when a label belongs to the product
	// (a side, a plan, a highlight) rather than to a status.
	ToneBrand  Tone = "brand"
	ToneAccent Tone = "accent"
)

// Valid reports whether t is one of the known enum values.
func (t Tone) Valid() bool {
	switch t {
	case ToneNeutral, ToneInfo, ToneWarning, ToneDanger, ToneSuccess, ToneAI, ToneBrand, ToneAccent:
		return true
	}
	return false
}

// Modifier returns the CSS class suffix for t ("danger"). Unknown/empty
// tones collapse to the neutral base so a typo degrades to a quiet chip,
// never a wrong colour.
func (t Tone) Modifier() string {
	if !t.Valid() {
		return string(ToneNeutral)
	}
	return string(t)
}

// StatusClass returns the ui-status--{tone} modifier class for a status
// chip: "" for the neutral tone (the base .ui-status chip is already the
// neutral look) and "ui-status--" + modifier otherwise.
func (t Tone) StatusClass() string {
	m := t.Modifier()
	if m == string(ToneNeutral) {
		return ""
	}
	return "ui-status--" + m
}

// AlertClass returns the ui-alert--{tone} modifier class. Alerts always take
// a tinted surface; the neutral tone maps to the info tint (the quietest).
func (t Tone) AlertClass() string {
	m := t.Modifier()
	if m == string(ToneNeutral) {
		m = string(ToneInfo)
	}
	return "ui-alert--" + m
}

// KPIValueClass returns the optional colour modifier for a KPI tile value
// ("" for neutral — the default ink). Danger is the common case: a negative
// balance or a closing shortfall rendered red.
func (t Tone) KPIValueClass() string {
	m := t.Modifier()
	switch m {
	case string(ToneDanger), string(ToneSuccess), string(ToneAI), string(ToneInfo), string(ToneWarning):
		return "kpi-tile__value--" + m
	}
	return ""
}
