// Pager types + window math (pager.templ) — the v2 DataTable pager ported to
// GET links (docs/ui-v1-plan.md §6, docs/ui-components.md §1): prev/1…n/next
// with a windowed number list, a row-range summary and per-page-size links,
// all preserving the filter query params. Pure math here (pagerItems) is
// unit-tested; the templ renders links.
package components

import (
	"net/url"

	"github.com/a-h/templ"
)

// PagerOpts configures the pager.
type PagerOpts struct {
	BaseHref  string // page path (no query)
	BaseQuery url.Values

	// PageParam/PageSizeParam name the paging query params ("page",
	// "pageSize" are the repo-wide defaults).
	PageParam     string
	PageSizeParam string

	// PageSizeOptions are the offered sizes; default {10, 25, 50, 100}.
	PageSizeOptions []int
	// ShowPageSize renders the rows-per-page links. nil/true = show (the
	// repo default); set a false pointer to hide the size menu.
	ShowPageSize *bool

	// HXTarget, when set, upgrades every link to an htmx partial re-render
	// of that region (full page response + hx-select) with URL push.
	HXTarget string
	Class    string
}

// pagerParam resolves the page param name.
func pagerParam(o PagerOpts) string {
	if o.PageParam != "" {
		return o.PageParam
	}
	return "page"
}

// pagerSizeParam resolves the page-size param name.
func pagerSizeParam(o PagerOpts) string {
	if o.PageSizeParam != "" {
		return o.PageSizeParam
	}
	return "pageSize"
}

// pageSizeOptions resolves the offered sizes (default 10/25/50/100).
func pageSizeOptions(o PagerOpts) []int {
	if len(o.PageSizeOptions) > 0 {
		return o.PageSizeOptions
	}
	return []int{10, 25, 50, 100}
}

// pagerClass joins the base class + extras.
func pagerClass(o PagerOpts) string {
	if o.Class == "" {
		return "pager"
	}
	return "pager " + o.Class
}

// showPageSizes resolves the rows-per-page visibility (default true).
func showPageSizes(o PagerOpts) bool {
	return o.ShowPageSize == nil || *o.ShowPageSize
}

// pageCount is the number of pages (0 when there are no rows).
func pageCount(total, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

// pageItem is one pager nav entry.
type pageItem struct {
	number int  // 1-based page number
	gap    bool // an ellipsis marker between ranges
}

// pagerItems returns the windowed page list: the first and last pages, a
// window of at most 5 numbers around current (2 each side) and ellipsis
// markers for the skipped ranges. pageCount <= 1 yields a single entry.
func pagerItems(current, count int) []pageItem {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return []pageItem{{number: 1}}
	}
	if current < 1 {
		current = 1
	}
	if current > count {
		current = count
	}
	from := current - 2
	if from < 1 {
		from = 1
	}
	to := current + 2
	if to > count {
		to = count
	}
	var items []pageItem
	if from > 1 {
		items = append(items, pageItem{number: 1})
		if from > 2 {
			items = append(items, pageItem{gap: true})
		}
	}
	for n := from; n <= to; n++ {
		items = append(items, pageItem{number: n})
	}
	if to < count {
		if to < count-1 {
			items = append(items, pageItem{gap: true})
		}
		items = append(items, pageItem{number: count})
	}
	return items
}

// pageHref builds the link for one page (filters + page size preserved).
func pageHref(o PagerOpts, n int) templ.SafeURL {
	q := CloneQuery(o.BaseQuery)
	q.Set(pagerParam(o), IntText(n))
	return templ.SafeURL(LinkTo(o.BaseHref, q))
}

// sizeHref builds a page-size link (resets to page 1).
func sizeHref(o PagerOpts, size int) templ.SafeURL {
	q := CloneQuery(o.BaseQuery)
	q.Set(pagerSizeParam(o), IntText(size))
	q.Del(pagerParam(o))
	return templ.SafeURL(LinkTo(o.BaseHref, q))
}

// pagerRangeText renders the row-range summary: "1–25 of 217" (en dash).
func pagerRangeText(currentPage, pageSize, total int) string {
	if total <= 0 {
		return "0 of 0"
	}
	start := (currentPage-1)*pageSize + 1
	if start < 1 {
		start = 1
	}
	end := currentPage * pageSize
	if end > total {
		end = total
	}
	return IntText(start) + "–" + IntText(end) + " of " + IntText(total)
}

// pagerPageClass marks the current page link.
func pagerPageClass(number, current int) string {
	if number == current {
		return "pager__page pager__page--current"
	}
	return "pager__page"
}
