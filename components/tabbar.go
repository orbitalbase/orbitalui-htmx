// TabBar types + helpers (tabbar.templ) — the orbital TabBar filter pills
// ported to GET links (docs/ui-v1-plan.md §6): each tab sets one query
// parameter and preserves the sibling filter params, so the server re-renders
// the table (the URL is the state). Count chips mirror v2.
package components

import (
	"net/url"

	"github.com/a-h/templ"
)

// Tab is one filter pill: its Value is the query value, Label the visible
// text and Count an optional row count for the chip (0 hides it unless
// ShowCounts is set — see TabBarOpts).
type Tab struct {
	Value string
	Label string
	Count int64
}

// TabBarOpts configures the bar.
type TabBarOpts struct {
	// Param is the query parameter the tabs set ("status", "category",
	// "kind", …). Empty defaults to "status".
	Param string
	// BaseHref is the page path (no query) tabs link to.
	BaseHref string
	// BaseQuery carries the current request query — sibling filter params
	// (date range, search text, …) are preserved on every tab link.
	BaseQuery url.Values
	// ShowCounts renders the optional per-tab count chip.
	ShowCounts bool
	// HXTarget, when set, upgrades the links to htmx partial re-renders of
	// that region (hx-get + hx-select from the full page response).
	HXTarget  string
	AriaLabel string // accessible name; default "Filters"
	Class     string
}

// tabParam resolves the query parameter name.
func tabParam(o TabBarOpts) string {
	if o.Param != "" {
		return o.Param
	}
	return "status"
}

// tabAriaLabel resolves the nav's accessible name.
func tabAriaLabel(o TabBarOpts) string {
	if o.AriaLabel != "" {
		return o.AriaLabel
	}
	return "Filters"
}

// tabHref builds the link for one tab value (sibling params preserved,
// paging reset).
func tabHref(o TabBarOpts, value string) templ.SafeURL {
	q := SetParam(o.BaseQuery, tabParam(o), value)
	return templ.SafeURL(LinkTo(o.BaseHref, q))
}

// tabItemClass marks the active pill.
func tabItemClass(value, active string) string {
	if value == active {
		return "tab tab--active"
	}
	return "tab"
}

// tabListClass joins the nav base class + extras.
func tabListClass(o TabBarOpts) string {
	if o.Class == "" {
		return "tab-bar"
	}
	return "tab-bar " + o.Class
}
