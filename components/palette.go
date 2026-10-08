// CommandPalette (palette.templ): a host-driven ⌘K / Ctrl+K dialog. The host
// supplies results either as static Groups, a JSON Endpoint, or both; the
// runtime in assets/orbital-ui.js owns the dialog, the debounced query, the
// grouped listbox, keyboard navigation and focus return.
//
// Endpoint contract. The runtime issues
//
//	GET <Endpoint>?q=<query>      (Accept: application/json, same-origin cookies)
//
// once on open (q is empty: suggestions) and again, debounced, as the query
// changes. The response is JSON:
//
//	{"groups": [
//	  {"label": "Pages", "items": [
//	    {"label": "Reports", "href": "/reports", "icon": "reports", "hint": "Go to page",
//	     "tone": "ai", "keywords": "analytics charts", "avatar": "Acme Inc", "square": true}
//	  ]},
//	  {"label": "Actions", "items": [
//	    {"label": "Ask the assistant", "href": "/assistant",
//	     "form": {"action": "/assistant/new", "method": "post", "fields": {"prompt": "…"}}}
//	  ]}
//	]}
//
// An item with "form" submits a hidden form instead of following href (a
// link can never POST); href is still its fallback target. JSON, not an HTML
// fragment, because rows are built with textContent (no injection surface),
// the page needs no inline script, and static and remote results share one
// shape. Static Groups use the same keys and are filtered client-side
// (prefix > word > substring > subsequence); remote groups follow them
// unfiltered, so the server decides what a query matches. A response that is
// not 2xx, or not JSON, shows only the static results.
package components

import (
	"encoding/json"

	"github.com/a-h/templ"
)

// PaletteItem is one palette result. Icon is an icon-set name (see Icons in
// CommandPaletteOpts). Tone tints the icon (ai, brand, accent). Avatar draws
// initials of that name instead of an icon; Square gives it the organisation
// shape. Keywords are matched but not shown.
type PaletteItem struct {
	Label    string       `json:"label"`
	Href     string       `json:"href"`
	Icon     string       `json:"icon,omitempty"`
	Hint     string       `json:"hint,omitempty"`
	Tone     string       `json:"tone,omitempty"`
	Keywords string       `json:"keywords,omitempty"`
	Avatar   string       `json:"avatar,omitempty"`
	Square   bool         `json:"square,omitempty"`
	Form     *PaletteForm `json:"form,omitempty"`
}

// PaletteForm makes Enter on an item submit a hidden form. Include any CSRF
// token in Fields.
type PaletteForm struct {
	Action string            `json:"action"`
	Method string            `json:"method,omitempty"` // default "post"
	Fields map[string]string `json:"fields,omitempty"`
}

// PaletteGroup is a labelled run of results.
type PaletteGroup struct {
	Label string        `json:"label"`
	Items []PaletteItem `json:"items"`
}

// CommandPaletteOpts configures CommandPalette.
//
//	ID — dialog id (default "command-palette").
//	Endpoint — GET results URL (see the package comment); empty = static only.
//	Groups — static groups, filtered client-side; with no Endpoint they are
//	    the whole palette, with one they are the quick links shown while the
//	    query is empty or matches.
//	Placeholder — the input's placeholder. Label — the dialog's accessible
//	    name (default "Command palette").
//	EmptyText — shown when nothing matches (the query is appended in quotes).
//	Icons — extra icon names the results may use; the defaults cover the
//	    common set (search, home, user, users, document, settings, plus, …).
type CommandPaletteOpts struct {
	ID          string
	Endpoint    string
	Groups      []PaletteGroup
	Placeholder string
	Label       string
	EmptyText   string
	Icons       []string
}

var paletteDefaultIcons = []string{"search", "home", "dashboard", "user", "users", "document", "folder", "settings", "plus", "arrow-right", "spark", "chat", "bell", "building", "briefcase", "reports"}

func paletteID(o CommandPaletteOpts) string {
	if o.ID == "" {
		return "command-palette"
	}
	return o.ID
}

func paletteLabel(o CommandPaletteOpts) string {
	if o.Label == "" {
		return "Command palette"
	}
	return o.Label
}

func palettePlaceholder(o CommandPaletteOpts) string {
	if o.Placeholder == "" {
		return "Search…"
	}
	return o.Placeholder
}

func paletteEmpty(o CommandPaletteOpts) string {
	if o.EmptyText == "" {
		return "No results for"
	}
	return o.EmptyText
}

func paletteGroupsJSON(o CommandPaletteOpts) string {
	if len(o.Groups) == 0 {
		return ""
	}
	b, err := json.Marshal(o.Groups)
	if err != nil {
		return ""
	}
	return string(b)
}

func paletteIcons(o CommandPaletteOpts) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, n := range paletteDefaultIcons {
		add(n)
	}
	for _, n := range o.Icons {
		add(n)
	}
	for _, g := range o.Groups {
		for _, it := range g.Items {
			add(it.Icon)
		}
	}
	return out
}

func paletteAttrs(o CommandPaletteOpts) templ.Attributes {
	a := templ.Attributes{"data-empty": paletteEmpty(o)}
	if o.Endpoint != "" {
		a["data-endpoint"] = o.Endpoint
	}
	if g := paletteGroupsJSON(o); g != "" {
		a["data-groups"] = g
	}
	return a
}
