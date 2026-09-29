// StatePanel types + helpers (statepanel.templ).
package components

// StatePanelOpts: Loading swaps the content for a skeleton bar (v2), Icon
// adds a muted lead icon above the heading (search, inbox-style empties),
// Class appends CSS classes.
type StatePanelOpts struct {
	Loading bool
	Icon    string // icon-set name (icons.go); "" hides it
	Class   string // extra classes (e.g. a width cap)
}

// statePanelClass joins the base + extra classes.
func statePanelClass(o StatePanelOpts) string {
	if o.Class == "" {
		return "ui-state"
	}
	return "ui-state " + o.Class
}

// busyAttr renders the aria-busy value for the loading flag.
func busyAttr(loading bool) string {
	if loading {
		return "true"
	}
	return "false"
}
