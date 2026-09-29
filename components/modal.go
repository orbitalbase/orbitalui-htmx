// ModalShell helper types + classes (modal.templ).
package components

// dialogClass joins the base dialog class + extras.
func dialogClass(o ModalOpts) string {
	if o.Class == "" {
		return "ui-dialog"
	}
	return "ui-dialog " + o.Class
}

// modalCloseLabel resolves the × button's accessible name.
func modalCloseLabel(o ModalOpts) string {
	if o.CloseLabel != "" {
		return o.CloseLabel
	}
	return "Close"
}
