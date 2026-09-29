// BalanceTool types + verdict math (balancetool.templ) — the v2 BalanceBeam
// ported to pure server-side rendering: journal-entry forms echo their
// current line totals to a #beam fragment (htmx) and this component renders
// the exact Dr/Cr verdict. All math is on Amounts (exact); the only floats
// are the decorative bar widths, derived from an exact pair.
package components

import (
	"strconv"
)

// BalanceToolOpts configures the beam.
type BalanceToolOpts struct {
	Currency string // format code for the displayed amounts (ui.FormatMoney)
	Ready    bool   // false appends " — complete the lines above" to a
	// balanced verdict (the v2 copy) when other constraints are unmet
	Class string
}

// beamVerdict mirrors the v2 BalanceBeam copy exactly:
//
//	balanced → "Balanced" (+ " — complete the lines above" when !ready)
//	empty    → "Enter debit and credit lines"
//	else     → "Out of balance by X"
type beamVerdict struct {
	balanced, empty bool
	diff            Amount
}

// balanceVerdict computes the beam state from the exact line totals.
func balanceVerdict(debits, credits Amount) beamVerdict {
	if debits.IsZero() && credits.IsZero() {
		return beamVerdict{empty: true}
	}
	balanced := debits.Sign() > 0 && debits.Cmp(credits) == 0
	var diff Amount
	if !balanced {
		if debits.Cmp(credits) > 0 {
			diff = debits.Sub(credits)
		} else {
			diff = credits.Sub(debits)
		}
	}
	return beamVerdict{balanced: balanced, diff: diff}
}

// beamShare returns the fill width percentage for one side: the v2 model
// fills each half proportionally, so a balanced beam shows two equal 50%
// fills. exact→float only for the decorative bar.
func beamShare(side, other Amount) float64 {
	larger := side
	if other.Cmp(larger) > 0 {
		larger = other
	}
	if larger.IsZero() {
		return 0
	}
	f, _ := side.Abs().Rat().Float64()
	l, _ := larger.Rat().Float64()
	pct := f / (2 * l) * 100
	if pct > 50 {
		pct = 50
	}
	return pct
}

// beamClass joins the state classes (balanced/empty) + extras.
func beamClass(v beamVerdict, o BalanceToolOpts) string {
	c := "beam"
	switch {
	case v.balanced:
		c += " beam--balanced"
	case v.empty:
		c += " beam--empty"
	default:
		c += " beam--unbalanced"
	}
	if o.Class != "" {
		c += " " + o.Class
	}
	return c
}

// beamWidthStyle renders the CSS width for a beam fill (decorative float).
func beamWidthStyle(pct float64) string {
	return "width:" + strconv.FormatFloat(pct, 'f', 1, 64) + "%"
}

// beamAriaLabel mirrors the v2 aria-label: the verdict with the exact
// money so screen readers get the numbers the bar only hints at.
func beamAriaLabel(debits, credits Amount, o BalanceToolOpts) string {
	v := balanceVerdict(debits, credits)
	switch {
	case v.balanced:
		return "Balanced at " + FormatMoney(debits, o.Currency)
	case v.empty:
		return "Enter debit and credit lines"
	default:
		return "Out of balance by " + FormatMoney(v.diff, o.Currency)
	}
}

// beamVerdictText returns the visible verdict: the amount stays inline (its
// own .amount span) except for the balanced/empty one-liners.
func beamVerdictText(v beamVerdict, o BalanceToolOpts) string {
	switch {
	case v.balanced:
		return "Balanced"
	case v.empty:
		return "Enter debit and credit lines"
	default:
		return "Out of balance by"
	}
}

// beamBalancedNote appends the v2 not-ready suffix to a balanced verdict.
func beamBalancedNote(v beamVerdict, o BalanceToolOpts) string {
	if v.balanced && !o.Ready {
		return " — complete the lines above"
	}
	return ""
}
