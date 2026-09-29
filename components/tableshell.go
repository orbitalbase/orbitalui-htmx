// TableShell types + helpers (tableshell.templ) — the .ledger-table frame:
// sticky uppercase header over the page-supplied rows, optional totals
// tfoot, empty state and per-column sort links. Data + body always come from
// the page (rows is the page's <tbody> content); this component renders the
// semantics and the shared chrome (docs/ui-components.md §1).
package components

import (
	"net/url"
	"strings"

	"github.com/a-h/templ"
)

// Align is a column's horizontal alignment.
type Align string

const (
	AlignLeft   Align = "left"
	AlignRight  Align = "right"
	AlignCenter Align = "center"
)

// Column describes one table column (header cell only — the page renders
// its own data cells, using the cell helpers below for money/mono values).
type Column struct {
	Key   string // stable id; also the sort key
	Label string // header text (uppercased by CSS)

	Align    Align  // AlignRight for amounts/dates-of-columns; "" = left
	Numeric  bool   // numbers get .amount (tabular, right-aligned) header
	Width    string // optional CSS width ("10rem") — rendered as a style
	Sortable bool   // emits a sort link when opts.Sort is set
	Class    string // extra classes on the <th>
	Optional bool   // may be hidden through per-user view preferences
}

// SortState describes the current sort for the header links. The wire value
// is "{key}" ascending or "{key}-desc"; sorting never changes the filter
// params, only page resets to 1.
type SortState struct {
	Key string // currently sorted column key ("" = natural order)
	Dir string // "asc" | "desc"

	BaseHref  string // page path (no query)
	BaseQuery url.Values
	Param     string // query parameter; default "sort"
}

// SortParam resolves the query parameter name.
func (s *SortState) sortParam() string {
	if s == nil || s.Param == "" {
		return "sort"
	}
	return s.Param
}

// sortValueFor returns the wire value a click on key should set (toggle).
func (s *SortState) sortValueFor(key string) string {
	if s != nil && s.Key == key && s.Dir == "asc" {
		return key + "-desc"
	}
	return key
}

// sortHref builds the sort link for a column (filters preserved, page 1).
func sortHref(s *SortState, key string) templ.SafeURL {
	if s == nil {
		return ""
	}
	q := CloneQuery(s.BaseQuery)
	q.Set(s.sortParam(), s.sortValueFor(key))
	q.Del("page")
	return templ.SafeURL(LinkTo(s.BaseHref, q))
}

// sortAria renders the th aria-sort value for a column ("ascending",
// "descending" or "" when not the active column).
func sortAria(s *SortState, key string) string {
	if s != nil && s.Key == key {
		if s.Dir == "desc" {
			return "descending"
		}
		return "ascending"
	}
	return ""
}

// sortAriaLabel is the active sort link's accessible name ("Sorted
// ascending by date") so screen readers get the direction too.
func sortAriaLabel(s *SortState, key, label string) string {
	dir := "ascending"
	if s != nil && s.Dir == "desc" {
		dir = "descending"
	}
	return "Sorted " + dir + " by " + label
}

// TableOpts configures the shell.
type TableOpts struct {
	ID      string // optional id on the .table-region (htmx target anchor)
	Caption string // sr-only table caption (page description)
	Class   string // extra classes on the .table-region

	// Empty state (rendered instead of rows when Empty is set).
	Empty            bool
	EmptyTitle       string
	EmptyMessage     string
	EmptyIcon        string // icon-set name (icons.go)
	EmptyActionHref  templ.SafeURL
	EmptyActionLabel string

	Sort *SortState // nil = no sort links even on Sortable columns
}

// thClass builds the header-cell classes for a column.
func thClass(c Column) string {
	classes := []string{}
	if c.Align == AlignRight || c.Numeric {
		classes = append(classes, "amount")
	}
	if c.Align == AlignCenter {
		classes = append(classes, "align-center")
	}
	if c.Class != "" {
		classes = append(classes, c.Class)
	}
	out := ""
	for i, cl := range classes {
		if i > 0 {
			out += " "
		}
		out += cl
	}
	return out
}

// tableColStyle renders the inline width style for a column.
func tableColStyle(c Column) string {
	return "width: " + c.Width
}

// tableRegionClass joins the region base class + extras.
func tableRegionClass(o TableOpts) string {
	if o.Class == "" {
		return "table-region"
	}
	return "table-region " + o.Class
}

// tdClass joins the base cell classes + optional extras.
func tdClass(base string, extra ...string) string {
	parts := []string{base}
	for _, e := range extra {
		if e != "" {
			parts = append(parts, e)
		}
	}
	return strings.Join(parts, " ")
}

// colSpanText is the colspan value for empty-row/footer cells.
func colSpanText(columns []Column) string {
	return IntText(len(columns))
}
