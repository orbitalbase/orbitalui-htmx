// SegmentedControl types + helpers (segmented.templ) — the orbital
// SegmentedControl "radio-like link group" server-side: one of N options
// active, each a GET link setting one query param (AR/AP side, report
// variant, direction). Sibling params preserved; paging reset.
package components

import (
	"net/url"

	"github.com/a-h/templ"
)

// SegOption is one segment.
type SegOption struct {
	Value    string
	Label    string
	Disabled bool // rendered as a disabled span (kept visible, not a link)
}

// SegmentedOpts configures the group.
type SegmentedOpts struct {
	Param     string // query parameter the segments set; default "view"
	BaseHref  string // page path (no query)
	BaseQuery url.Values
	HXTarget  string // optional htmx partial re-render target
	AriaLabel string // accessible name; default "View"
	Class     string
}

// segParam resolves the query parameter name.
func segParam(o SegmentedOpts) string {
	if o.Param != "" {
		return o.Param
	}
	return "view"
}

// segAriaLabel resolves the group's accessible name.
func segAriaLabel(o SegmentedOpts) string {
	if o.AriaLabel != "" {
		return o.AriaLabel
	}
	return "View"
}

// segHref builds the link for one segment value.
func segHref(o SegmentedOpts, value string) templ.SafeURL {
	q := SetParam(o.BaseQuery, segParam(o), value)
	return templ.SafeURL(LinkTo(o.BaseHref, q))
}

// segItemClass marks the active segment.
func segItemClass(value, active string) string {
	if value == active {
		return "segmented__item segmented__item--active"
	}
	return "segmented__item"
}

// segmentedClass joins the group base class + extras.
func segmentedClass(o SegmentedOpts) string {
	if o.Class == "" {
		return "segmented"
	}
	return "segmented " + o.Class
}
