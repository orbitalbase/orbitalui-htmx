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

// SegItem is one segment of a SegmentedNav: a link to any href, so the
// destination (a page, a tab route, a filtered list) is the caller's.
//
//	Label — visible text. Icon — optional icon-set name before it.
//	Count — optional count pill after it ("12"; a string so "99+" works).
//	Active — marks the current segment (aria-current="page"); at most one.
//	Attrs — extra attributes on the link (hx-*, data-*, class).
type SegItem struct {
	Label  string
	Href   templ.SafeURL
	Icon   string
	Count  string
	Active bool
	Attrs  templ.Attributes
}

// SegNavOpts configures SegmentedNav.
//
//	AriaLabel — the nav's accessible name (default "View").
//	Class — extra classes on the nav.
type SegNavOpts struct {
	AriaLabel string
	Class     string
}

// segNavMax is how many segments the sliding thumb supports (one class per
// count and index in theme.css); a longer nav falls back to a filled active
// segment with no slide.
const segNavMax = 8

var (
	segNavCounts  = [...]string{"ui-seg-nav--n2", "ui-seg-nav--n3", "ui-seg-nav--n4", "ui-seg-nav--n5", "ui-seg-nav--n6", "ui-seg-nav--n7", "ui-seg-nav--n8"}
	segNavIndexes = [...]string{"ui-seg-nav--i0", "ui-seg-nav--i1", "ui-seg-nav--i2", "ui-seg-nav--i3", "ui-seg-nav--i4", "ui-seg-nav--i5", "ui-seg-nav--i6", "ui-seg-nav--i7"}
)

func segNavClass(items []SegItem, o SegNavOpts) string {
	n := len(items)
	if n < 2 {
		n = 2
	}
	if n > segNavMax {
		return joinClass("ui-seg-nav", "ui-seg-nav--flow", o.Class)
	}
	active := ""
	for i, it := range items {
		if it.Active {
			active = segNavIndexes[i]
			break
		}
	}
	return joinClass("ui-seg-nav", segNavCounts[n-2], active, o.Class)
}

func segNavLabel(o SegNavOpts) string {
	if o.AriaLabel != "" {
		return o.AriaLabel
	}
	return "View"
}

func segNavItemClass(it SegItem) string {
	return mergeClass(joinClass("ui-seg-nav__item", ifClass(it.Active, "is-active")), it.Attrs)
}
