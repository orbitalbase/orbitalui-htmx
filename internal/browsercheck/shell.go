package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func with(comp templ.Component, children ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return comp.Render(templ.WithChildren(ctx, seq(children...)), w)
	})
}

// shellHead mirrors a host's <head>: theme.js synchronously before the
// stylesheets, orbital-ui.js before Alpine. variant selects the same three
// builds as the component page (?csp=1, ?host=1).
func shellHead(w http.ResponseWriter, r *http.Request, title string) templ.Component {
	alpine := "/vendor/alpine.min.js"
	hostScript := ""
	if r.URL.Query().Get("csp") == "1" {
		alpine = "/vendor/alpine-csp.min.js"
		w.Header().Set("Content-Security-Policy", "script-src 'self'")
	}
	if r.URL.Query().Get("host") == "1" {
		hostScript = `<script src="/vendor/host-runtime.js" defer></script>`
	}
	return raw(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + title + `</title>
<script src="/static/theme.js"></script>
<link rel="stylesheet" href="/static/tokens.css"><link rel="stylesheet" href="/static/motion.css"><link rel="stylesheet" href="/static/controls.css"><link rel="stylesheet" href="/static/theme.css"><link rel="stylesheet" href="/static/shell.css">
<script src="/vendor/htmx.min.js" defer></script>` + hostScript + `<script src="/static/menu.js" defer></script><script src="/static/orbital-ui.js" defer></script><script src="` + alpine + `" defer></script>
</head><body>`)
}

func shellPage(w http.ResponseWriter, r *http.Request) {
	nav := with(c.SidebarMenu("Primary"),
		with(c.SidebarSection(c.SidebarSectionOpts{}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Overview", Href: "/shell", Icon: "dashboard", Active: true}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Reviews", Href: "/shell/target?to=reviews", Icon: "inbox", Count: 4, CountLabel: "waiting on you"}),
		),
		with(c.SidebarSection(c.SidebarSectionOpts{Label: "Profiles"}),
			c.SidebarItem(c.SidebarItemOpts{Label: "You", Href: "/shell/target?to=you", Leading: c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 20}), Percent: c.Pct(64)}),
		),
	)
	account := seq(
		c.MenuHeader("Ada Lovelace", "ada@example.com"),
		c.MenuLink("Your profile", "/shell/target?to=profile", c.MenuItemOpts{Icon: "user"}),
	)
	shell := c.AppShell(c.AppShellOpts{
		Brand:       raw(`<a href="/shell" id="brand">orbital</a>`),
		Sidebar:     nav,
		Footer:      c.SidebarUser(c.SidebarUserOpts{Name: "Ada Lovelace"}, account),
		Breadcrumbs: []c.Crumb{{Label: "Workspace", Href: "/shell"}, {Label: "Overview"}},
		Actions: seq(
			c.TopbarSearch(c.TopbarSearchOpts{Placeholder: "Search or ask…"}),
			c.ThemeSwitch(c.ThemeSwitchOpts{Size: "sm"}),
			c.TopbarNewMenu(c.TopbarNewMenuOpts{Label: "New"}, c.MenuLink("Document", "/shell/target?to=new", c.MenuItemOpts{Icon: "document"})),
		),
		Overlays: seq(
			c.CommandPalette(c.CommandPaletteOpts{
				Endpoint:    "/shell/palette",
				Placeholder: "Search…",
				Groups: []c.PaletteGroup{{Label: "Pages", Items: []c.PaletteItem{
					{Label: "Overview", Href: "/shell", Icon: "dashboard"},
					{Label: "Reports", Href: "/shell/target?to=reports", Icon: "reports"},
					{Label: "Settings", Href: "/shell/target?to=settings", Icon: "settings"},
				}}},
			}),
			with(c.ToastRegion(), c.Toast(c.ToastOpts{Tone: c.ToneInfo, Message: "Server toast", Auto: false})),
		),
	}, seq(
		c.PageHeader("Overview", c.PageHeaderOpts{}),
		raw(`<p><button type="button" id="fire-success" class="ui-button" hx-post="/shell/toast?tone=success" hx-swap="none">Success toast</button>
<button type="button" id="fire-danger" class="ui-button" hx-post="/shell/toast?tone=danger" hx-swap="none">Danger toast</button>
<button type="button" id="fire-string" class="ui-button" hx-post="/shell/toast?tone=string" hx-swap="none">String toast</button></p>`),
	))
	render(w, r, shellHead(w, r, "Shell check"), shell, raw(`</body></html>`))
}

func shellTarget(w http.ResponseWriter, r *http.Request) {
	render(w, r, shellHead(w, r, "Target"), raw(`<main><h1 id="target">Target `+templ.EscapeString(r.URL.Query().Get("to"))+`</h1></main></body></html>`))
}

// shellPalette answers the palette contract: GET ?q= returns {"groups": [...]}.
func shellPalette(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	docs := []c.PaletteItem{
		{Label: "Report archive 2025", Href: "/shell/target?to=archive", Icon: "document", Hint: "Document"},
		{Label: "Quarterly figures", Href: "/shell/target?to=figures", Icon: "document", Hint: "Document"},
	}
	var items []c.PaletteItem
	for _, d := range docs {
		if q != "" && strings.Contains(strings.ToLower(d.Label), q) {
			items = append(items, d)
		}
	}
	// "unsafe" returns targets a result must never follow: the runtime drops
	// non-http(s) hrefs and form actions.
	if q == "unsafe" {
		items = append(items,
			c.PaletteItem{Label: "Unsafe link", Href: "javascript:window.__ouiPwned=1"},
			c.PaletteItem{Label: "Unsafe form", Href: "javascript:window.__ouiPwned=2", Form: &c.PaletteForm{Action: "javascript:window.__ouiPwned=3"}})
	}
	groups := []c.PaletteGroup{}
	if len(items) > 0 {
		groups = append(groups, c.PaletteGroup{Label: "Documents", Items: items})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"groups": groups})
}

// shellToast answers with HX-Trigger toast payloads.
func shellToast(w http.ResponseWriter, r *http.Request) {
	var payload any
	switch r.URL.Query().Get("tone") {
	case "danger":
		payload = map[string]any{"tone": "danger", "title": "Failed", "message": "The save did not go through."}
	case "string":
		payload = "Plain message"
	default:
		payload = map[string]any{"tone": "success", "title": "Saved", "message": "Your changes are live.", "auto": true}
	}
	b, _ := json.Marshal(map[string]any{"toast": payload})
	w.Header().Set("HX-Trigger", string(b))
	w.WriteHeader(http.StatusNoContent)
}
