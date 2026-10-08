package main

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// shellCSS is gallery chrome only: a brand mark, and stages that pin the
// shell's fixed or absolutely positioned parts (palette dialog) in the flow.
const shellCSS = `<style>
.g-brand { display: inline-flex; align-items: center; gap: 9px; color: var(--oui-ink); font-family: var(--oui-font-display); font-size: 19px; font-weight: 600; letter-spacing: -0.04em; text-decoration: none; }
.g-brand__mark { flex: none; }
.g-canvas { width: 15.5rem; padding: 0.75rem 0.5rem; background: var(--oui-canvas); border-radius: 12px; box-shadow: 0 0 0 1px var(--oui-line); }
.g-canvas--tall { min-height: 34rem; }
.g-bar { display: flex; align-items: center; gap: 12px; padding: 8px 16px; background: var(--oui-surface); border-radius: 12px; box-shadow: 0 0 0 1px var(--oui-line); }
.g-bar__grow { flex: 1; }
.g-pin { position: relative; display: grid; place-items: start center; min-height: 26rem; padding: 2rem 1rem; overflow: hidden; background: var(--oui-scrim); border-radius: 12px; isolation: isolate; }
.g-pin .ui-palette { position: static; width: min(38.75rem, 100%); margin: 0; animation: none; }
.g-toasts { display: grid; gap: 8px; justify-items: start; }
.g-toasts .ui-toast { animation: none; }
.g-menu-room { min-height: 15rem; }
</style>`

func init() {
	register(section{ID: "shell-parts", Title: "Shell parts", Note: "Sidebar items (icon, leading avatar, count, percent, done, tones, dense, add), sidebar user with the open account menu, topbar search and the New menu. The full frame is the Shell page in the left nav.", Body: shellPartsBody()})
	register(section{ID: "palette", Title: "Command palette", Note: "Open state, pinned in a stage (the runtime opens it with the keyboard shortcut). Results are rendered by shell.js; this markup mirrors its output.", Body: paletteBody()})
	register(section{ID: "toasts", Title: "Toasts", Note: "Every tone, with and without a title, and a long message. ToastRegion is fixed to the viewport corner; here the toasts sit in the flow.", Body: toastsBody()})
	register(section{ID: "theme-switch", Title: "Theme switch", Note: "Sun / monitor / moon, built on SegmentedChoice; assets/theme.js applies the choice before first paint.", Body: seq(
		group("ThemeSwitch", cell("default", c.ThemeSwitch(c.ThemeSwitchOpts{})), cell("sm", c.ThemeSwitch(c.ThemeSwitchOpts{Size: "sm", StorageKey: "gallery-theme"}))),
		group("In a menu row", cell("MenuRow", raw(`<div class="g-panel" style="width:16rem">`+render(c.MenuRow("Theme", c.ThemeSwitch(c.ThemeSwitchOpts{Size: "sm"})))+`</div>`))),
	)})
}

func render(cs templ.Component) string {
	var b bytes.Buffer
	_ = cs.Render(context.Background(), &b)
	return b.String()
}

func avatarOf(name string, square bool) templ.Component {
	return c.Avatar(name, c.AvatarOpts{Px: 20, Square: square})
}

// with renders comp with children, the way templ's @comp() { ... } does.
func with(comp templ.Component, children ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return comp.Render(templ.WithChildren(ctx, seq(children...)), w)
	})
}

func shellPartsBody() templ.Component {
	return seq(
		raw(shellCSS),
		group("Sidebar", wrap(`<div class="g-canvas g-canvas--tall">`, `</div>`, sidebarNav())),
		group("Sidebar user, menu open", wrap(`<div class="g-canvas g-menu-room" style="padding-top:15rem">`, `</div>`,
			c.SidebarUser(c.SidebarUserOpts{Name: "Ada Lovelace", Open: true}, accountMenu()))),
		group("Topbar pieces", wrap(`<div class="g-bar" style="min-width:34rem">`, `</div>`,
			c.BreadcrumbsWith([]c.Crumb{{Label: "Reports", Href: "/reports"}, {Label: "Quarterly summary"}}, c.BreadcrumbsOpts{}),
			raw(`<span class="g-bar__grow"></span>`),
			c.TopbarSearch(c.TopbarSearchOpts{Placeholder: "Search or ask…"}),
			c.TopbarNewMenu(c.TopbarNewMenuOpts{Label: "New"}, newMenu()),
		)),
		group("Topbar New menu, open", wrap(`<div class="g-bar g-menu-room" style="min-width:34rem;align-items:flex-start;justify-content:flex-end">`, `</div>`,
			c.Menu(c.MenuOpts{Align: "end", Open: true, Label: "New", TriggerClass: "ui-button ui-button--secondary ui-button--sm ui-topbar-new__btn", Class: "ui-topbar-new"},
				seq(c.Icon("plus"), raw(`<span class="ui-topbar-new__label">New</span>`)), newMenu()))),
	)
}

func sidebarNav() templ.Component {
	return with(c.SidebarMenu("Primary"),
		with(c.SidebarSection(c.SidebarSectionOpts{}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Ask the assistant", Href: "#ask", Icon: "spark", Tone: c.ToneAI}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Overview", Href: "#overview", Icon: "dashboard", Active: true}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Reviews", Href: "#reviews", Icon: "inbox", Count: 4, CountLabel: "waiting on you"}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Documents", Href: "#documents", Icon: "folder"}),
		),
		with(c.SidebarSection(c.SidebarSectionOpts{Label: "Profiles"}),
			c.SidebarItem(c.SidebarItemOpts{Label: "You", Href: "#you", Leading: avatarOf("Ada Lovelace", false), Percent: c.Pct(64)}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Acme Sdn Bhd", Href: "#acme", Leading: avatarOf("Acme Sdn Bhd", true), Done: true, DoneLabel: "Profile complete"}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Northstar Technologies", Href: "#northstar", Leading: avatarOf("Northstar Technologies", true), Percent: c.Pct(38)}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Add company", Href: "#add", Add: true}),
		),
		with(c.SidebarSection(c.SidebarSectionOpts{Label: "Recent chats"}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Quarterly summary draft", Href: "#chat1", Dense: true, Leading: c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 16})}),
			c.SidebarItem(c.SidebarItemOpts{Label: "Share transfer questions", Href: "#chat2", Dense: true, Leading: c.Avatar("Acme Sdn Bhd", c.AvatarOpts{Px: 16, Square: true})}),
		),
	)
}

func accountMenu() templ.Component {
	return seq(
		c.MenuHeader("Ada Lovelace", "ada@example.com"),
		c.MenuRow("Theme", c.ThemeSwitch(c.ThemeSwitchOpts{Size: "sm"})),
		c.MenuSeparator(),
		c.MenuLink("Your profile", "#profile", c.MenuItemOpts{Icon: "user"}),
		c.MenuLink("Settings", "#settings", c.MenuItemOpts{Icon: "settings"}),
		c.MenuSeparator(),
		c.MenuLink("Sign out", "#signout", c.MenuItemOpts{Icon: "logout", Danger: true}),
	)
}

func newMenu() templ.Component {
	return seq(
		c.MenuLink("Chat with the assistant", "#new-chat", c.MenuItemOpts{Icon: "spark"}),
		c.MenuSeparator(),
		c.MenuLabel("Create"),
		c.MenuLink("Document", "#new-doc", c.MenuItemOpts{Icon: "document"}),
		c.MenuLink("Company profile", "#new-company", c.MenuItemOpts{Icon: "building", Hint: "3 left"}),
	)
}

var paletteGroups = []c.PaletteGroup{
	{Label: "Pages", Items: []c.PaletteItem{
		{Label: "Overview", Href: "#overview", Icon: "dashboard", Hint: "Go to page"},
		{Label: "Reviews", Href: "#reviews", Icon: "inbox", Keywords: "approvals waiting"},
		{Label: "Documents", Href: "#documents", Icon: "folder"},
		{Label: "Settings", Href: "#settings", Icon: "settings"},
	}},
	{Label: "Companies", Items: []c.PaletteItem{
		{Label: "Acme Sdn Bhd", Href: "#acme", Avatar: "Acme Sdn Bhd", Square: true, Hint: "Company"},
		{Label: "Northstar Technologies", Href: "#northstar", Avatar: "Northstar Technologies", Square: true, Hint: "Company"},
	}},
	{Label: "Actions", Items: []c.PaletteItem{
		{Label: "Ask the assistant", Href: "#ask", Icon: "spark", Tone: "ai", Keywords: "chat help"},
	}},
}

func paletteOpts() c.CommandPaletteOpts {
	return c.CommandPaletteOpts{Placeholder: "Search or ask the assistant…", Groups: paletteGroups}
}

// paletteBody pins the real dialog markup open with rows shaped like the
// runtime's output (the runtime needs a script; the gallery pages carry none).
func paletteBody() templ.Component {
	var rows strings.Builder
	n := 0
	for _, g := range paletteGroups {
		rows.WriteString(`<li class="ui-palette__group" role="presentation">` + g.Label + `</li>`)
		for _, it := range g.Items {
			sel := "false"
			if n == 1 {
				sel = "true"
			}
			lead := `<span class="ui-palette__icon` + toneSuffix(it.Tone) + `" aria-hidden="true">` + render(c.Icon(it.Icon)) + `</span>`
			if it.Avatar != "" {
				lead = `<span class="ui-avatar ui-avatar--20 ui-avatar--square ui-avatar--hue-2" aria-hidden="true"><span class="ui-avatar__initials">` + c.Initials(it.Avatar) + `</span></span>`
			}
			hint := ""
			if it.Hint != "" {
				hint = `<span class="ui-palette__hint">` + it.Hint + `</span>`
			}
			rows.WriteString(`<li id="g-pal-opt-` + string(rune('0'+n)) + `" class="ui-palette__item" role="option" aria-selected="` + sel + `"><a href="` + it.Href + `" tabindex="-1">` + lead + `<span class="ui-palette__label">` + it.Label + `</span>` + hint + `</a></li>`)
			n++
		}
	}
	html := render(c.CommandPalette(paletteOpts()))
	html = strings.Replace(html, `<dialog id="command-palette"`, `<dialog open id="command-palette"`, 1)
	html = strings.Replace(html, `data-palette-list></ul>`, `data-palette-list>`+rows.String()+`</ul>`, 1)
	html = strings.Replace(html, `aria-expanded="false"`, `aria-expanded="true" aria-activedescendant="g-pal-opt-1"`, 1)
	return seq(raw(shellCSS), raw(`<div class="g-pin">`+html+`</div>`))
}

func toneSuffix(t string) string {
	if t == "" {
		return ""
	}
	return " ui-palette__icon--" + t
}

func toastsBody() templ.Component {
	return seq(
		raw(shellCSS),
		wrap(`<div class="g-toasts">`, `</div>`,
			c.Toast(c.ToastOpts{Message: "Draft saved."}),
			c.Toast(c.ToastOpts{Tone: c.ToneInfo, Title: "Sync paused", Message: "We will retry in a minute."}),
			c.Toast(c.ToastOpts{Tone: c.ToneSuccess, Title: "Invoice posted", Message: "Invoice 42 was posted to the ledger.", Auto: false}),
			c.Toast(c.ToastOpts{Tone: c.ToneWarning, Title: "Period closes soon", Message: "Post remaining entries before Friday."}),
			c.Toast(c.ToastOpts{Tone: c.ToneDanger, Title: "Something needs attention", Message: "The transfer could not be saved because the total does not balance."}),
			c.Toast(c.ToastOpts{Tone: c.ToneAI, Title: "Draft ready", Message: "The assistant prepared a first version for you to review."}),
			c.Toast(c.ToastOpts{Tone: c.ToneSuccess, Message: "A very long unbroken message aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}),
		),
	)
}
