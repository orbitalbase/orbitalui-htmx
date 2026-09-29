// SearchField types + helpers (searchfield.templ) — the v2 orbital
// SearchField semantics server-side: a labelled search input inside a GET
// filters bar that re-renders the table region over htmx (debounced), with a
// plain GET submit as the no-JS fallback (docs/ui-components.md §1).
package components

import "github.com/a-h/templ"

// SearchFieldOpts configures the control.
type SearchFieldOpts struct {
	Label       string // accessible name (sr-only); default "Search"
	Name        string // form field name; default "q"
	ID          string // input id (label for); default "<name>" when empty
	Value       string // current query (server state)
	Placeholder string // default "Search…"
	Class       string // extra classes on the wrapper

	// Partial re-render wiring (htmx). HXGet is the filters URL WITHOUT q
	// and WITHOUT page/pageSize (the page handler rebuilds it from the
	// current filters each render); the component serializes the closest
	// form (hx-include) so q and any sibling filter fields travel along,
	// debounces 300ms and pushes the resulting URL (shareable state).
	HXGet    templ.SafeURL // empty = plain GET input (full reload on Enter)
	HXTarget string        // "" → "#rows" (the canonical rows region)
}

// searchName resolves the default field name.
func searchName(o SearchFieldOpts) string {
	if o.Name != "" {
		return o.Name
	}
	return "q"
}

// searchID resolves the input id used by the sr-only label.
func searchID(o SearchFieldOpts) string {
	if o.ID != "" {
		return o.ID
	}
	return "search-" + searchName(o)
}

// searchLabel resolves the accessible name.
func searchLabel(o SearchFieldOpts) string {
	if o.Label != "" {
		return o.Label
	}
	return "Search"
}

// searchPlaceholder resolves the input placeholder.
func searchPlaceholder(o SearchFieldOpts) string {
	if o.Placeholder != "" {
		return o.Placeholder
	}
	return "Search…"
}

// searchTarget resolves the htmx swap target (canonical "#rows").
func searchTarget(o SearchFieldOpts) string {
	if o.HXTarget != "" {
		return o.HXTarget
	}
	return "#rows"
}

// searchClass joins the wrapper base class + extras.
func searchClass(o SearchFieldOpts) string {
	if o.Class == "" {
		return "search-field"
	}
	return "search-field " + o.Class
}
