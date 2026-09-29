// Query-URL helpers shared by the link-style controls (TabBar, Segmented,
// Pager, TableShell sorting) and the page list templates: every filter
// state change is a plain GET link that preserves the sibling query params
// — URLs are the state, servers re-render (docs/ui-v1-plan.md §3,
// docs/ui-components.md §1). Page packages used to carry their own copies
// of this family; the component library owns the one implementation now.
package components

import (
	"net/url"

	"github.com/a-h/templ"
)

// CloneQuery copies q so link helpers never mutate the caller's map.
func CloneQuery(q url.Values) url.Values {
	if q == nil {
		return url.Values{}
	}
	out := make(url.Values, len(q))
	for k, vs := range q {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

// LinkTo renders "baseHref?query" (or the bare href when the query is
// empty). q must already be Encode-safe (url.Values.Encode output).
func LinkTo(baseHref string, q url.Values) string {
	if len(q) == 0 {
		return baseHref
	}
	return baseHref + "?" + q.Encode()
}

// SetParam returns a copy of q with key=value set and the page param
// dropped — changing a filter resets to page 1 at the current page size.
func SetParam(q url.Values, key, value string) url.Values {
	out := CloneQuery(q)
	if value == "" {
		out.Del(key)
	} else {
		out.Set(key, value)
	}
	out.Del("page")
	return out
}

// UnsetParams returns a copy of q without any of keys (used by clear
// links); the page param is dropped too.
func UnsetParams(q url.Values, keys ...string) url.Values {
	out := CloneQuery(q)
	for _, k := range keys {
		out.Del(k)
	}
	out.Del("page")
	return out
}

// SearchHXGet builds a SearchField's partial-render URL: the base path with
// every current filter EXCEPT q and the paging params (q rides along via
// the field's hx-include; a changed query resets to page 1 —
// docs/ui-components.md §1b).
func SearchHXGet(base string, q url.Values) templ.SafeURL {
	out := CloneQuery(q)
	out.Del("q")
	out.Del("page")
	out.Del("pageSize")
	return templ.SafeURL(LinkTo(base, out))
}
