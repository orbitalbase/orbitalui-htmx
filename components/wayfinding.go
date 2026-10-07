// In-page wayfinding (wayfinding.templ): the chip row that narrows a list and
// the index that jumps around a long page.
//
// Both are plain links. A filter chip sets a GET parameter, so the narrowed
// view is shareable, survives a reload and needs no JavaScript; a section
// link is a fragment anchor. Neither needs a client runtime, which is what
// lets them render under a strict script-src 'self' policy.
package components

import "github.com/a-h/templ"

// ChipLink is one filter chip. Value is matched against FilterChips' active
// argument — give the "everything" chip the zero value so no filter is the
// selected state. Count renders a quiet figure after the label.
type ChipLink struct {
	Label string
	Href  templ.SafeURL
	Value string
	Count int
}

// FilterChipsOpts configures FilterChips. Label names the navigation landmark
// and is required: a bare row of links is not navigable without it.
type FilterChipsOpts struct {
	Label string
	ID    string
	Class string
}

func filterChipsLabel(o FilterChipsOpts) string {
	if o.Label == "" {
		return "Filter"
	}
	return o.Label
}

func chipClass(c ChipLink, active string) string {
	return joinClass("ui-chip", modifier(c.Value == active, "is-active"))
}

// chipCurrent marks the selected chip for assistive technology. aria-current
// is the honest attribute here: the chip is the view the reader is on.
func chipCurrent(c ChipLink, active string) string {
	if c.Value == active {
		return "page"
	}
	return "false"
}

// SectionLink is one entry in a SectionNav. ID is the id of the heading or
// section it jumps to; Href overrides the derived "#"+ID when the target
// lives on another page.
type SectionLink struct {
	Label string
	ID    string
	Href  templ.SafeURL
}

func sectionLinkHref(l SectionLink) templ.SafeURL {
	if l.Href != "" {
		return l.Href
	}
	return templ.SafeURL("#" + l.ID)
}

// SectionNavOpts configures SectionNav. Label names the landmark.
type SectionNavOpts struct {
	Label string
	ID    string
	Class string
}

func sectionNavLabel(o SectionNavOpts) string {
	if o.Label == "" {
		return "On this page"
	}
	return o.Label
}
