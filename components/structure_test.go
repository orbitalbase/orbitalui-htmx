package components

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func structContains(t *testing.T, html string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(html, w) {
			t.Errorf("output is missing %q\n%s", w, html)
		}
	}
}

func structOmits(t *testing.T, html string, banned ...string) {
	t.Helper()
	for _, b := range banned {
		if strings.Contains(html, b) {
			t.Errorf("output must not contain %q\n%s", b, html)
		}
	}
}

var emptyTableCols = []Column{{Key: "a", Label: "A"}}

func TestTableShellFlushCapsCompact(t *testing.T) {
	plain := renderC(t, TableShell(emptyTableCols, TableOpts{}, nil, nil))
	structContains(t, plain, `class="table-region"`, `class="ledger-table"`)
	structOmits(t, plain, "table-region--flush", "ledger-table--caps", "ledger-table--compact")

	html := renderC(t, TableShell(emptyTableCols, TableOpts{Flush: true, HeaderStyle: TableHeaderCaps, Compact: true, Class: "host", TableClass: "wide"}, nil, nil))
	structContains(t, html, `class="table-region table-region--flush host"`, `class="ledger-table ledger-table--caps ledger-table--compact wide"`)
}

func TestDisclosureSummaryID(t *testing.T) {
	html := renderC(t, Disclosure("Audit", DisclosureOpts{SummaryID: "audit-summary"}, templ.NopComponent))
	structContains(t, html, `<summary id="audit-summary">`)
	structOmits(t, renderC(t, Disclosure("Audit", DisclosureOpts{}, templ.NopComponent)), "<summary id=")
}

func TestSettingsSectionInCardModifier(t *testing.T) {
	html := renderC(t, SettingsSection(SettingsSectionOpts{Title: "General", InCard: true}, nil))
	structContains(t, html, "ui-settings-section ui-settings-section--card")
	structOmits(t, renderC(t, SettingsSection(SettingsSectionOpts{Title: "General"}, nil)), "--card")
}

func TestBreadcrumbsWithCollapseOnPhone(t *testing.T) {
	crumbs := []Crumb{{Label: "Company", Href: "/c"}, {Label: "Journal"}}
	html := renderC(t, BreadcrumbsWith(crumbs, BreadcrumbsOpts{CollapseOnPhone: true, Label: "Trail"}))
	structContains(t, html, `class="ui-breadcrumbs ui-breadcrumbs--collapse"`, `aria-label="Trail"`, `aria-current="page"`)
	// The original constructor is unchanged.
	old := renderC(t, Breadcrumbs(crumbs))
	structContains(t, old, `class="ui-breadcrumbs"`, `aria-label="Breadcrumb"`)
	structOmits(t, old, "--collapse")
	if renderC(t, BreadcrumbsWith(nil, BreadcrumbsOpts{})) != "" {
		t.Error("an empty trail must render nothing")
	}
}

func TestSidebarNavRail(t *testing.T) {
	sections := []NavSection{
		{Links: []NavLink{{Label: "Overview", Href: "/", Icon: "dashboard", Active: true}}},
		{Label: "Accounting", ID: "acc", Collapsible: true, Links: []NavLink{{Label: "Journal", Href: "/j", Badge: "3", Tone: ToneDanger}}},
	}
	full := renderC(t, SidebarNav(sections, SidebarNavOpts{}))
	structContains(t, full, `class="ui-nav"`, "<details", "ui-nav__heading-chevron")

	rail := renderC(t, SidebarNav(sections, SidebarNavOpts{Rail: true}))
	structContains(t, rail, `class="ui-nav ui-nav--rail"`, `title="Journal"`, `<span class="ui-nav__label">Journal</span>`, "ui-nav__badge--danger")
	// A rail has no heading to toggle, so collapsible sections render open.
	structOmits(t, rail, "<details", "ui-nav__heading-chevron")
}

func TestWorkspaceSwitcherCompact(t *testing.T) {
	html := renderC(t, WorkspaceSwitcher(WorkspaceSwitcherOpts{Name: "Acme", Kind: "Company", Compact: true}, templ.NopComponent))
	structContains(t, html, "ui-switcher ui-switcher--compact", "Company workspace menu: ", `class="ui-switcher__name">Acme`)
	structOmits(t, renderC(t, WorkspaceSwitcher(WorkspaceSwitcherOpts{Name: "Acme"}, templ.NopComponent)), "--compact")
}

func TestMenuOpenKbdAndLabel(t *testing.T) {
	closed := renderC(t, Menu(MenuOpts{Label: "Account"}, nil, nil))
	structOmits(t, closed, " open")
	open := renderC(t, Menu(MenuOpts{Label: "Account", Open: true}, nil, nil))
	structContains(t, open, "<details", " open")

	item := renderC(t, MenuLink("Palette", "/p", MenuItemOpts{Kbd: []string{"⌘", "K"}}))
	structContains(t, item, `class="ui-menu__kbd ui-kbd-group"`, `<kbd class="ui-kbd">⌘</kbd>`, `<kbd class="ui-kbd">K</kbd>`)
	structOmits(t, renderC(t, MenuLink("Plain", "/p", MenuItemOpts{})), "ui-kbd")

	structContains(t, renderC(t, MenuLabel("Workspace")), `class="ui-menu__section"`, "Workspace")
}
