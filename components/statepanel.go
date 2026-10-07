// StatePanel types + helpers (statepanel.templ).
package components

// StatePanelOpts: Loading swaps the content for a skeleton bar (v2), Icon
// adds a muted lead icon above the heading (search, inbox-style empties),
// Compact renders a smaller, left-aligned panel with no heading element —
// the in-card empty state, where the card's own title already supplies the
// outline level and a second heading would compete with it. Class appends
// CSS classes.
type StatePanelOpts struct {
	Loading bool
	Icon    string // icon-set name (icons.go); "" hides it
	Compact bool   // in-card variant: no heading element, left-aligned, denser
	Class   string // extra classes (e.g. a width cap)
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
