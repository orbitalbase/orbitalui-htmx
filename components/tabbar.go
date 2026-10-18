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
	HXTarget string
	// ID is the nav's element id. With HXTarget set it makes the bar
	// refreshable: each link also sends hx-select-oob="#<ID>", so the
	// server-rendered bar (active pill, counts, hrefs built from the new
	// query) is swapped in from the same full-page response. Needed when the
	// bar sits outside HXTarget, where it would otherwise keep a stale
	// highlight and stale sibling-filter hrefs. Leave it empty for a bar
	// INSIDE HXTarget: htmx pulls OOB elements out of the response before the
	// main swap, so the bar would vanish from the re-rendered region.
	ID string
	// SelectOOB lists extra comma-separated selectors to swap out-of-band
	// alongside the bar (sibling filter controls, hidden filter state).
	// Ignored unless HXTarget and ID are set.
	SelectOOB string
	AriaLabel string // accessible name; default "Filters"
	Class     string
}

// oobSelect joins the bar's own id selector with the caller's extras for
// hx-select-oob; empty when the bar is not refreshable (no id or no target).
func oobSelect(target, id, extra string) string {
	if target == "" || id == "" {
		return ""
	}
	if extra == "" {
		return "#" + id
	}
	return "#" + id + ", " + extra
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

// TabItem is one tab of a TabNav: a link to any href.
//
//	Label — visible text. Icon — optional icon-set name before it.
//	Count — optional count pill after the label (a string, so "99+" works).
//	Active — marks the current tab (aria-current="page"); at most one.
//	Attrs — extra attributes on the link (hx-*, data-*, class).
type TabItem struct {
	Label  string
	Href   templ.SafeURL
	Icon   string
	Count  string
	Active bool
	Attrs  templ.Attributes
}

// TabNavOpts configures TabNav.
//
//	AriaLabel — the nav's accessible name (default "Sections").
//	Class — extra classes on the nav.
type TabNavOpts struct {
	AriaLabel string
	Class     string
}

func tabNavLabel(o TabNavOpts) string {
	if o.AriaLabel != "" {
		return o.AriaLabel
	}
	return "Sections"
}

func tabNavClass(o TabNavOpts) string { return joinClass("tab-bar", o.Class) }

func tabNavItemClass(it TabItem) string {
	return mergeClass(joinClass("tab", ifClass(it.Active, "tab--active")), it.Attrs)
}
