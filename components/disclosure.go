// Disclosure types (disclosure.templ).
package components

// DisclosureOpts: Open renders the <details> expanded; Summary is a muted
// hint shown next to the label while collapsed (v2 "Posting details" style).
type DisclosureOpts struct {
	Open    bool
	Summary string
}
