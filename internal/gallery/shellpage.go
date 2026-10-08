package main

import (
	"context"
	"io"
	"os"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

const brandHTML = `<a class="g-brand" href="#home" aria-label="Orbital home"><svg class="g-brand__mark" width="24" height="24" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><rect width="24" height="24" rx="7" fill="var(--oui-ink)"></rect><circle cx="9.3" cy="12" r="4.2" fill="var(--oui-brand)"></circle><circle cx="14.7" cy="12" r="4.2" fill="var(--oui-accent)"></circle><path d="M12 8.783A4.2 4.2 0 0 1 12 15.217A4.2 4.2 0 0 1 12 8.783Z" fill="var(--oui-ink)"></path></svg><span>orbital</span></a>`

// shellDemo is the full AppShell with a generic overview page.
func shellDemo(full bool) templ.Component {
	var main templ.Component = overviewPage()
	if full {
		main = fullWidthPage()
	}
	return c.AppShell(c.AppShellOpts{
		Brand:       raw(brandHTML),
		Sidebar:     sidebarNav(),
		Footer:      c.SidebarUser(c.SidebarUserOpts{Name: "Ada Lovelace"}, accountMenu()),
		Breadcrumbs: []c.Crumb{{Label: "Workspace", Href: "#workspace"}, {Label: "Overview"}},
		Actions:     seq(c.TopbarSearch(c.TopbarSearchOpts{Placeholder: "Search or ask…"}), c.TopbarNewMenu(c.TopbarNewMenuOpts{Label: "New"}, newMenu())),
		Overlays: seq(
			c.CommandPalette(paletteOpts()),
			with(c.ToastRegion(), c.Toast(c.ToastOpts{Tone: c.ToneSuccess, Message: "Workspace synced.", Auto: false})),
		),
		FullWidth: full,
	}, main)
}

func overviewPage() templ.Component {
	return seq(
		c.PageHeader("Overview", c.PageHeaderOpts{Description: "A generic page inside the shell: header, tiles and a card list.",
			Actions: c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Icon: "plus", Label: "New document"})}),
		raw(`<div class="g-grid" style="margin:1.5rem 0">`),
		c.KPITile("Open items", "14", c.KPITileOpts{Hint: "waiting on you", Icon: "inbox"}),
		c.KPITile("Completed", "128", c.KPITileOpts{Tone: c.ToneSuccess, Hint: "this quarter"}),
		c.KPITile("Assistant drafts", "23", c.KPITileOpts{Tone: c.ToneAI, Icon: "spark"}),
		raw(`</div>`),
		c.Card(c.CardOpts{Title: "Recent documents", Flush: true}, rowSet(c.ActionListOpts{}, false)),
	)
}

func fullWidthPage() templ.Component {
	return raw(`<div style="flex:1;display:grid;place-items:center;color:var(--oui-text-quaternary)">A full-width sheet: the page owns its layout and scroll.</div>`)
}

func writeShellPage(path string, p preset, full bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	title := "Shell"
	if full {
		title = "Shell, full-width sheet"
	}
	io.WriteString(f, pageHead("Orbital UI / "+p.Name+" / "+title, p.Theme, p.Stylesheet, "../"))
	io.WriteString(f, shellCSS)
	if err := shellDemo(full).Render(context.Background(), f); err != nil {
		return err
	}
	io.WriteString(f, `<script src="../assets/menu.js" defer></script><script src="../assets/orbital-ui.js" defer></script><script src="../assets/shell.js" defer></script></body></html>`)
	return nil
}
