// Alert helper types + classes (alert.templ).
package components

// AlertOpts: Dismissible adds the × control that hides the band for the rest
// of the page visit; DismissLabel is its accessible name.
type AlertOpts struct {
	Dismissible  bool
	DismissLabel string
	Class        string // extra classes
}

// alertDismissLabel resolves the accessible close name.
func alertDismissLabel(o AlertOpts) string {
	if o.DismissLabel != "" {
		return o.DismissLabel
	}
	return "Dismiss"
}

// alertClass assembles the band classes: ui-alert + tone modifier + extras.
func alertClass(t Tone, o AlertOpts) string {
	c := "ui-alert " + t.AlertClass()
	if o.Dismissible {
		c += " ui-alert--dismissible"
	}
	if o.Class != "" {
		c += " " + o.Class
	}
	return c
}

// roleFor mirrors v2: danger is assertive ("alert"), everything else is
// polite ("status").
func roleFor(t Tone) string {
	if t == ToneDanger {
		return "alert"
	}
	return "status"
}
