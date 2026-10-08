// StatePanel types + helpers (statepanel.templ).
package components

import "github.com/a-h/templ"

// StatePanelOpts: Loading swaps the content for a skeleton bar (v2), Icon
// adds a muted lead icon above the heading (search, inbox-style empties),
// Compact renders a smaller, left-aligned panel with no heading element —
// the in-card empty state, where the card's own title already supplies the
// outline level and a second heading would compete with it. Class appends
// CSS classes.
//
// Orb replaces Icon with the AI presence orb (for states that lead to the
// assistant); Actions renders a centred button row under the body.
type StatePanelOpts struct {
	Loading bool
	Icon    string // icon-set name (icons.go); "" hides it
	Compact bool   // in-card variant: no heading element, left-aligned, denser
	Orb     bool   // idle AI orb instead of the icon
	// TitleTag is the heading element of a non-compact panel: "h2" (the
	// default) or "h3" for a panel nested under a card's h2. Anything else
	// falls back to "h2".
	TitleTag string
	Actions  templ.Component
	Class    string // extra classes (e.g. a width cap)
}

// statePanelClass joins the base + modifier + extra classes.
func statePanelClass(o StatePanelOpts) string {
	return joinClass("ui-state", modifier(o.Compact, "ui-state--compact"), o.Class)
}

// busyAttr renders the aria-busy value for the loading flag.
func busyAttr(loading bool) string {
	if loading {
		return "true"
	}
	return "false"
}

// statePanelH3 reports whether the title renders as an h3.
func statePanelH3(o StatePanelOpts) bool { return o.TitleTag == "h3" }
