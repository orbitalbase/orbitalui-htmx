package main

import (
	"context"
	"io"
	"net/url"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func amt(s string) c.Amount {
	a, err := c.ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

func init() {
	register(section{ID: "alerts", Title: "Alerts and callouts", Note: "Every tone; ai uses the AI gradient tint.", Body: alertsBody()})
	register(section{ID: "tables", Title: "Tables and pager", Note: "Default, caps headers, compact, flush, sortable, empty, selection toolbar.", Body: tablesBody()})
	register(section{ID: "disclosure", Title: "Disclosure", Body: disclosureBody()})
	register(section{ID: "headers", Title: "Page and record headers", Body: headersBody()})
	register(section{ID: "settings", Title: "Settings sections", Body: settingsBody()})
	register(section{ID: "menus", Title: "Menus and switcher", Note: "Menus are rendered open.", Body: menusBody()})
	register(section{ID: "navigation", Title: "Navigation", Note: "Sidebar, rail, breadcrumbs.", Body: navigationBody()})
	register(section{ID: "regions", Title: "Page frame, balance tool, async regions", Body: regionsBody()})
}

func alertsBody() templ.Component {
	tones := []c.Tone{c.ToneNeutral, c.ToneInfo, c.ToneSuccess, c.ToneWarning, c.ToneDanger, c.ToneBrand, c.ToneAccent, c.ToneAI}
	var alerts, callouts []templ.Component
	for _, t := range tones {
		alerts = append(alerts, cell(string(t), c.Alert(t, "Heads up", c.AlertOpts{}, c.Text("Re-open the period first, then post the entry again."))))
	}
	alerts = append(alerts,
		cell("no title", c.Alert(c.ToneInfo, "", c.AlertOpts{}, c.Text("A short message without a heading."))),
		cell("dismissible", c.Alert(c.ToneWarning, "Dismissible", c.AlertOpts{Dismissible: true}, c.Text("Closes locally."))),
		cell("long unbroken", c.Alert(c.ToneDanger, "Overflow", c.AlertOpts{}, c.Text("https://example.com/a/very/long/path/that/never/breaks/and/keeps/going/and/going/until/it/must/wrap/somewhere/ok"))),
	)
	for _, t := range tones {
		callouts = append(callouts, cell(string(t), c.Callout(t, c.CalloutOpts{Title: "Needs your review", Description: "Three items are waiting before the period can close.", Icon: "alert"},
			c.Text("Callout body content."))))
	}
	return seq(groupGrid("Alert", alerts...), groupGrid("Callout", callouts...))
}

func rowsBody(rows ...[]string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, "<tbody>")
		for _, r := range rows {
			io.WriteString(w, "<tr>")
			for i, v := range r {
				if i == len(r)-1 {
					io.WriteString(w, `<td class="amount mono">`+v+"</td>")
				} else {
					io.WriteString(w, "<td>"+v+"</td>")
				}
			}
			io.WriteString(w, "</tr>")
		}
		_, err := io.WriteString(w, "</tbody>")
		return err
	})
}

func tablesBody() templ.Component {
	cols := []c.Column{{Key: "ref", Label: "Reference"}, {Key: "party", Label: "Counterparty"}, {Key: "status", Label: "Status"}, {Key: "amount", Label: "Amount", Align: c.AlignRight}}
	sortCols := []c.Column{{Key: "ref", Label: "Reference", Sortable: true}, {Key: "party", Label: "Counterparty", Sortable: true}, {Key: "status", Label: "Status"}, {Key: "amount", Label: "Amount", Align: c.AlignRight, Sortable: true}}
	body := rowsBody(
		[]string{"JE-1042", "Acme Sdn Bhd", "Posted", "1,250.00"},
		[]string{"JE-1043", "Orbital Holdings with a deliberately long counterparty name", "Draft", "88.40"},
		[]string{"JE-1044", "Kindred Labs", "Voided", "-310.00"},
	)
	q := url.Values{"q": {"acme"}}
	return seq(
		groupStack("Default",
			c.TableShell(cols, c.TableOpts{Caption: "Entries"}, body, nil),
			c.Pager(2, 10, 47, c.PagerOpts{BaseHref: "/entries", BaseQuery: q}),
		),
		groupStack("Caps header", c.TableShell(cols, c.TableOpts{Caption: "Entries", HeaderStyle: c.TableHeaderCaps}, body, nil)),
		groupStack("Caps header + compact", c.TableShell(cols, c.TableOpts{Caption: "Entries", HeaderStyle: c.TableHeaderCaps, Compact: true}, body, nil)),
		groupStack("Sortable (descending on amount)", c.TableShell(sortCols, c.TableOpts{Caption: "Entries", Sort: &c.SortState{Key: "amount", Dir: "desc", BaseHref: "/entries", BaseQuery: q}}, body, nil)),
		groupStack("Flush inside a flush card",
			c.Card(c.CardOpts{Title: "Recent entries", Flush: true},
				c.TableShell(cols, c.TableOpts{Caption: "Entries", Flush: true, HeaderStyle: c.TableHeaderCaps}, body, nil)),
		),
		groupStack("Empty", c.TableShell(cols, c.TableOpts{Caption: "Entries", Empty: true, EmptyTitle: "No entries yet", EmptyMessage: "Post your first journal entry to see it here.", EmptyIcon: "journal"}, nil, nil)),
		groupStack("Selection toolbar", c.SelectionToolbar(c.SelectionToolbarOpts{Selected: 3, Eligible: 3, Excluded: 1, ExcludedReason: "One entry is already posted."},
			c.Button(c.ButtonVariantSecondary, c.ButtonSizeSM, c.ButtonOpts{Label: "Post selected"}))),
		groupStack("Pager (first page, ellipsis)", c.Pager(1, 25, 200, c.PagerOpts{BaseHref: "/entries"})),
	)
}

func disclosureBody() templ.Component {
	body := c.Text("The revealed content sits here and may run to several lines of explanatory copy.")
	return seq(groupStack("Variants",
		c.Disclosure("Posting details", c.DisclosureOpts{Hint: "4 fields", SummaryID: "gal-disc-summary"}, body),
		c.Disclosure("Open by default", c.DisclosureOpts{Open: true, Summary: "unused when open"}, body),
		c.Disclosure("Edit service information", c.DisclosureOpts{Variant: c.DisclosurePanel, Hint: "4 fields"}, body),
		c.Disclosure("Panel, open", c.DisclosureOpts{Variant: c.DisclosurePanel, Open: true}, body),
		c.Disclosure("Show the evidence", c.DisclosureOpts{Variant: c.DisclosureQuiet}, body),
		c.Disclosure("Delete this package", c.DisclosureOpts{Variant: c.DisclosureDanger}, body),
		c.Disclosure("Danger, open", c.DisclosureOpts{Variant: c.DisclosureDanger, Open: true}, body),
	))
}

func headersBody() templ.Component {
	badge := c.StatusBadge("Posted", c.ToneSuccess, c.StatusBadgeOpts{})
	return seq(groupStack("PageHeader",
		cell("full", c.PageHeader("Journal entries", c.PageHeaderOpts{Eyebrow: "Company · Acme Sdn Bhd", Description: "Post, review and reverse entries for the current period.", Meta: badge,
			Actions: seq(c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Export"}), c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "New entry", Icon: "plus"}))})),
		cell("compact", c.PageHeader("Compact header", c.PageHeaderOpts{Compact: true, Description: "Denser header for registers and split views."})),
		cell("long unbroken text", c.PageHeader("A_very_long_unbroken_title_that_must_wrap_instead_of_overflowing_the_container_at_phone_width", c.PageHeaderOpts{Description: "https://example.com/very/long/unbroken/description/path/that/should/wrap/anywhere/at/narrow/widths"})),
	), groupStack("RecordHeader",
		c.RecordHeader(c.RecordHeaderOpts{Eyebrow: "Journal entry", Title: "JE-1042 · Accrued revenue", Subtitle: "Posted 3 Sep 2026 by Mei", Icon: "journal", Status: badge}),
		c.RecordHeader(c.RecordHeaderOpts{Eyebrow: "Suggestion", Title: "Match this invoice", Subtitle: "Proposed by the assistant", Icon: "sparkles", IconTone: c.ToneAI}),
		c.RecordHeader(c.RecordHeaderOpts{Eyebrow: "Failed", Title: "Bank feed disconnected", Icon: "alert", IconTone: c.ToneDanger}),
	))
}

func settingsBody() templ.Component {
	field := func(label, id string) templ.Component {
		return c.LabelledField(label, id, c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: id, Name: id}))
	}
	sec := func(title, desc, id string, in bool) templ.Component {
		return c.SettingsSection(c.SettingsSectionOpts{Title: title, Description: desc, InCard: in}, field("Value", id))
	}
	return seq(
		groupStack("Plain", seq(sec("Profile", "How your name appears on records.", "s1", false), sec("Notifications", "Choose what reaches your inbox.", "s2", false))),
		groupStack("Inside a padded card",
			c.Card(c.CardOpts{Title: "Workspace settings"}, seq(sec("General", "Name and locale.", "s3", false), sec("Billing", "Plan and invoices.", "s4", false), sec("Danger zone", "Irreversible actions.", "s5", false)))),
		groupStack("Inside a flush card (InCard)",
			c.Card(c.CardOpts{Title: "Workspace settings", Flush: true}, seq(sec("General", "Name and locale.", "s6", true), sec("Billing", "Plan and invoices.", "s7", true)))),
	)
}

func menuPanel() templ.Component {
	return seq(
		c.MenuHeader("Mei Tan", "mei@example.com"),
		c.MenuSeparator(),
		c.MenuLabel("Workspace"),
		c.MenuLink("Profile", "/profile", c.MenuItemOpts{Icon: "user"}),
		c.MenuLink("Command palette", "/palette", c.MenuItemOpts{Icon: "search", Kbd: []string{"⌘", "K"}}),
		c.MenuLink("Current item", "/current", c.MenuItemOpts{Icon: "check", Selected: true, Hint: "active"}),
		c.MenuSeparator(),
		c.MenuLink("Sign out", "/logout", c.MenuItemOpts{Icon: "logout", Danger: true}),
	)
}

func menusBody() templ.Component {
	trigger := c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Account"})
	tall := func(cs ...templ.Component) templ.Component {
		return wrap(`<div style="min-height:20rem;padding-top:.5rem">`, `</div>`, cs...)
	}
	return seq(
		groupGrid("Menu (open)", cell("start", tall(c.Menu(c.MenuOpts{Label: "Account", Open: true}, trigger, menuPanel()))),
			cell("end-aligned", wrap(`<div style="min-height:20rem;display:flex;align-items:flex-start;justify-content:flex-end">`, `</div>`, c.Menu(c.MenuOpts{Label: "Account", Open: true, Align: "end"}, trigger, menuPanel())))),
		groupGrid("Workspace switcher",
			cell("closed", c.WorkspaceSwitcher(c.WorkspaceSwitcherOpts{Name: "Acme Sdn Bhd", Subtitle: "Company · MYR", Kind: "Company", StatusTone: c.ToneSuccess}, menuPanel())),
			cell("long name", c.WorkspaceSwitcher(c.WorkspaceSwitcherOpts{Name: "Orbital Holdings Berhad (Group Consolidated)", Subtitle: "Firm", Kind: "Firm"}, menuPanel())),
			cell("compact", c.WorkspaceSwitcher(c.WorkspaceSwitcherOpts{Name: "Acme Sdn Bhd", Subtitle: "Company", Compact: true, StatusTone: c.ToneWarning}, menuPanel())),
		),
	)
}

func navSections() []c.NavSection {
	return []c.NavSection{
		{Links: []c.NavLink{{Label: "Overview", Href: "/", Icon: "dashboard", Active: true}, {Label: "Inbox", Href: "/inbox", Icon: "inbox", Badge: "4", Tone: c.ToneDanger}}},
		{Label: "Accounting", ID: "acc", Collapsible: true, Links: []c.NavLink{{Label: "Journal", Href: "/journal", Icon: "journal"}, {Label: "Reports", Href: "/reports", Icon: "chart", Badge: "New", Tone: c.ToneAI}}},
		{Label: "Company", Links: []c.NavLink{{Label: "Settings", Href: "/settings", Icon: "settings"}, {Label: "A very long navigation label that truncates", Href: "/long", Icon: "document"}}},
	}
}

func navigationBody() templ.Component {
	crumbs := []c.Crumb{{Label: "Company", Href: "/c"}, {Label: "Accounting", Href: "/a"}, {Label: "Journal", Href: "/j"}, {Label: "JE-1042"}}
	narrow := func(cs ...templ.Component) templ.Component {
		return wrap(`<div style="width:16rem;padding:.5rem;background:var(--oui-canvas);border-radius:12px;box-shadow:0 0 0 1px var(--oui-line)">`, `</div>`, cs...)
	}
	rail := func(cs ...templ.Component) templ.Component {
		return wrap(`<div style="width:4.25rem;padding:.5rem .25rem;background:var(--oui-canvas);border-radius:12px;box-shadow:0 0 0 1px var(--oui-line)">`, `</div>`, cs...)
	}
	return seq(
		group("Sidebar nav / rail",
			cell("full", narrow(c.SidebarNav(navSections(), c.SidebarNavOpts{}))),
			cell("rail", rail(c.WorkspaceSwitcher(c.WorkspaceSwitcherOpts{Name: "Acme", Compact: true}, menuPanel()), c.SidebarNav(navSections(), c.SidebarNavOpts{Rail: true, Label: "Rail"}))),
		),
		groupStack("Breadcrumbs", cell("default", c.Breadcrumbs(crumbs)), cell("collapse on phone (resize to 390 to see only the current crumb)", c.BreadcrumbsWith(crumbs, c.BreadcrumbsOpts{CollapseOnPhone: true}))),
		groupStack("Section nav + chips",
			c.SectionNav([]c.SectionLink{{Label: "General", ID: "a"}, {Label: "Team", ID: "b"}}, c.SectionNavOpts{}),
			c.FilterChips([]c.ChipLink{{Label: "All", Href: "/x"}, {Label: "Filings", Href: "/x?c=f", Value: "f", Count: 4}}, "f", c.FilterChipsOpts{}),
		),
	)
}

func regionsBody() templ.Component {
	return seq(
		groupStack("PageFrame (narrow) with FormGrid",
			kids(c.PageFrame(c.PageFrameOpts{Width: c.PageWidthNarrow}), seq(
				kids(c.FormGrid(c.FormGridOpts{Columns: 2}), seq(
					c.LabelledField("First name", "pf1", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "pf1", Name: "pf1"})),
					c.LabelledField("Last name", "pf2", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "pf2", Name: "pf2"})),
				)),
				kids(c.FormActions(c.FormActionsOpts{Divided: true}), seq(
					c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Cancel"}),
					c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Save"}),
				)),
			)),
		),
		groupGrid("Balance tool",
			cell("out of balance", c.BalanceTool(amt("1250.00"), amt("1000.00"), c.BalanceToolOpts{Currency: "MYR"})),
			cell("balanced", c.BalanceTool(amt("500.00"), amt("500.00"), c.BalanceToolOpts{Currency: "MYR", Ready: true})),
			cell("empty", c.BalanceTool(c.ZeroAmount(), c.ZeroAmount(), c.BalanceToolOpts{Currency: "MYR"})),
		),
		groupStack("Record state, dependencies, async",
			c.RecordStateSummary(c.RecordStateSummaryOpts{Posting: "Posted", Review: "Reviewed", Period: "Open", Provenance: "Bank feed"}),
			c.DependencyPanel("deps", "Linked records", []c.Dependency{{Label: "Invoice INV-9", Detail: "Blocks voiding", Href: "/i"}, {Label: "Payment P-3", Detail: "Allocated", Href: "/p"}}),
			c.AsyncRegion(c.AsyncRegionOpts{ID: "a1", Label: "Loading", State: "loading"}, nil),
			c.AsyncRegion(c.AsyncRegionOpts{ID: "a2", Label: "Failed", State: "error", Message: "Could not load.", RetryHref: "/retry"}, nil),
		),
	)
}

// kids renders comp with kids as its templ children block.
func kids(comp, children templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return comp.Render(templ.WithChildren(ctx, children), w)
	})
}
