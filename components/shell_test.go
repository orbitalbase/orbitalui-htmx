package components

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func withKids(comp templ.Component, kids ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return comp.Render(templ.WithChildren(ctx, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for _, k := range kids {
				if err := k.Render(ctx, w); err != nil {
					return err
				}
			}
			return nil
		})), w)
	})
}

func TestAppShellFrameAndSlots(t *testing.T) {
	out := renderC(t, AppShell(AppShellOpts{
		Brand:       Text("brand-slot"),
		Sidebar:     Text("sidebar-slot"),
		Footer:      Text("footer-slot"),
		Actions:     Text("actions-slot"),
		Overlays:    Text("overlay-slot"),
		Breadcrumbs: []Crumb{{Label: "Home", Href: "/"}, {Label: "Here"}},
	}, Text("main-slot")))
	for _, needle := range []string{
		`data-oui-shell`, `data-nav-open="false"`, `id="app-sidebar"`, `ui-vt-sidebar`, `ui-vt-main`,
		`brand-slot`, `sidebar-slot`, `footer-slot`, `actions-slot`, `overlay-slot`, `main-slot`,
		`data-shell-open`, `aria-controls="app-sidebar"`, `data-shell-scrim`, `data-shell-close`,
		`id="content"`, `ui-shell__page`, `aria-current="page"`, `skip-link`,
	} {
		want(t, out, needle)
	}
	if strings.Contains(out, "ui-shell--full") {
		t.Error("default shell must not be full width")
	}
	full := renderC(t, AppShell(AppShellOpts{FullWidth: true, ID: "side"}, Text("x")))
	want(t, full, "ui-shell--full")
	want(t, full, `id="side"`)
	if strings.Contains(full, "ui-shell__page") {
		t.Error("full-width shell must not wrap main in the page column")
	}
}

func TestSidebarItemVariants(t *testing.T) {
	out := renderC(t, SidebarItem(SidebarItemOpts{Label: "Reviews", Href: "/r", Icon: "inbox", Count: 120, CountLabel: "waiting", Active: true, Tone: ToneAI, Percent: Pct(0), Key: "reviews"}))
	for _, needle := range []string{`is-active`, `aria-current="page"`, `ui-nav__link--ai`, `99+`, `aria-label="120 waiting"`, `0%`, `data-nav-key="reviews"`, `ui-nav__icon`} {
		want(t, out, needle)
	}
	done := renderC(t, SidebarItem(SidebarItemOpts{Label: "Acme", Href: "/a", Leading: Text("L"), Done: true, Percent: Pct(50)}))
	want(t, done, `ui-nav__lead`)
	want(t, done, `ui-nav__done`)
	if strings.Contains(done, "ui-nav__pct") || strings.Contains(done, "ui-nav__icon") {
		t.Error("Done replaces the percentage and Leading replaces the icon")
	}
	if strings.Contains(renderC(t, SidebarItem(SidebarItemOpts{Label: "x", Href: "/x"})), "ui-nav__count") {
		t.Error("zero count must render nothing")
	}
	add := renderC(t, SidebarItem(SidebarItemOpts{Label: "Add", Href: "/n", Add: true, Dense: true}))
	want(t, add, `ui-nav__link--add`)
	want(t, add, `ui-nav__link--dense`)
	want(t, add, `ui-nav__add`)
}

func TestSidebarSectionLabelsItself(t *testing.T) {
	out := renderC(t, withKids(SidebarSection(SidebarSectionOpts{Label: "Recent chats"}), Text("kid")))
	want(t, out, `aria-labelledby="oui-nav-recent-chats"`)
	want(t, out, `id="oui-nav-recent-chats"`)
	want(t, out, "kid")
	bare := renderC(t, withKids(SidebarSection(SidebarSectionOpts{}), Text("kid")))
	if strings.Contains(bare, "aria-labelledby") || strings.Contains(bare, "<h2") {
		t.Error("an unlabelled section has no heading")
	}
}

func TestTopbarPieces(t *testing.T) {
	out := renderC(t, TopbarSearch(TopbarSearchOpts{Placeholder: "Search or ask…"}))
	for _, needle := range []string{`data-palette-open`, `aria-haspopup="dialog"`, `Search or ask…`, `⌘`, `ui-kbd`} {
		want(t, out, needle)
	}
	if strings.Contains(renderC(t, TopbarSearch(TopbarSearchOpts{NoShortcut: true})), "ui-kbd") {
		t.Error("NoShortcut hides the key caps")
	}
	menu := renderC(t, TopbarNewMenu(TopbarNewMenuOpts{Label: "New"}, Text("panel")))
	for _, needle := range []string{`data-ui-menu`, `ui-menu--end`, `ui-topbar-new__label`, `panel`} {
		want(t, menu, needle)
	}
	user := renderC(t, SidebarUser(SidebarUserOpts{Name: "Ada Lovelace"}, Text("menu-panel")))
	for _, needle := range []string{`ui-menu--top`, `ui-sidebar-user`, `Ada Lovelace`, `menu-panel`, `aria-label="Account menu"`} {
		want(t, user, needle)
	}
}

func TestCommandPaletteSemanticsAndContract(t *testing.T) {
	out := renderC(t, CommandPalette(CommandPaletteOpts{
		Endpoint: "/palette",
		Groups:   []PaletteGroup{{Label: "Pages", Items: []PaletteItem{{Label: "Home", Href: "/", Icon: "home"}, {Label: "Odd & <ends>", Href: "/x", Icon: "custom-icon"}}}},
	}))
	for _, needle := range []string{
		`<dialog id="command-palette"`, `data-oui-palette`, `data-endpoint="/palette"`, `role="combobox"`, `aria-controls="command-palette-list"`,
		`aria-autocomplete="list"`, `role="listbox"`, `data-palette-input`, `data-palette-list`, `role="status"`, `data-palette-status`,
		`data-palette-icons`, `data-icon="home"`, `data-icon="custom-icon"`,
	} {
		want(t, out, needle)
	}
	// The static groups travel as JSON in one attribute and round-trip.
	m := regexp.MustCompile(`data-groups="([^"]*)"`).FindStringSubmatch(out)
	if m == nil {
		t.Fatal("no data-groups attribute")
	}
	raw := strings.NewReplacer("&#34;", `"`, "&amp;", "&", "&lt;", "<", "&gt;", ">", "&#39;", "'", `&`, "&", `<`, "<", `>`, ">").Replace(m[1])
	var groups []PaletteGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil {
		t.Fatalf("data-groups is not JSON: %v\n%s", err, raw)
	}
	if len(groups) != 1 || len(groups[0].Items) != 2 || groups[0].Items[1].Label != "Odd & <ends>" {
		t.Errorf("groups did not round-trip: %+v", groups)
	}
	if strings.Contains(renderC(t, CommandPalette(CommandPaletteOpts{})), "data-endpoint") {
		t.Error("no endpoint, no attribute")
	}
}

func TestToastRendersRolesAndRegionTemplates(t *testing.T) {
	danger := renderC(t, Toast(ToastOpts{Tone: ToneDanger, Title: "Failed", Message: "Nope", Auto: true}))
	for _, needle := range []string{`role="alert"`, `ui-toast--danger`, `data-toast-auto`, `Failed`, `Nope`, `data-toast-close`, `aria-label="Dismiss"`} {
		want(t, danger, needle)
	}
	plain := renderC(t, Toast(ToastOpts{Message: "Saved"}))
	want(t, plain, `role="status"`)
	if strings.Contains(plain, "ui-toast--") || strings.Contains(plain, "data-toast-auto") || strings.Contains(plain, "ui-toast__title") {
		t.Errorf("neutral toast without title or auto carries no modifiers:\n%s", plain)
	}
	region := renderC(t, withKids(ToastRegion(), Toast(ToastOpts{Message: "Hello"})))
	for _, needle := range []string{`role="region"`, `aria-label="Notifications"`, `data-toast-stack`, `aria-live="polite"`, `data-toast-template`, `data-toast-icon-for="danger"`, `data-toast-icon-for="ai"`, `Hello`} {
		want(t, region, needle)
	}
}

func TestThemeSwitchOptions(t *testing.T) {
	out := renderC(t, ThemeSwitch(ThemeSwitchOpts{StorageKey: "myapp-theme"}))
	for _, needle := range []string{`role="radiogroup"`, `aria-label="Color theme"`, `data-oui-theme-switch`, `data-storage-key="myapp-theme"`, `value="light"`, `value="system"`, `value="dark"`, `ui-seg-choice--n3`} {
		want(t, out, needle)
	}
	if !regexp.MustCompile(`value="system"[^>]*checked`).MatchString(out) {
		t.Error("system is the server-rendered choice")
	}
	want(t, renderC(t, ThemeSwitch(ThemeSwitchOpts{})), `data-storage-key="oui-theme"`)
}

// Both shell scripts must reach hosts that serve assets.Files, and the
// pre-paint script must stay an external, blocking-friendly file.
func TestShellRuntimeContract(t *testing.T) {
	js, err := os.ReadFile("../assets/shell.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"data-oui-shell", "data-palette-open", "data-oui-palette", "data-toast-stack", "data-oui-theme-switch", "window.OrbitalShell",
		`document.addEventListener("toast"`, "aria-activedescendant", "OrbitalTheme",
	} {
		if !strings.Contains(string(js), needle) {
			t.Errorf("shell.js is missing %s", needle)
		}
	}
	theme, err := os.ReadFile("../assets/theme.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"oui-theme"`, "data-storage-key", "prefers-color-scheme", "window.OrbitalTheme", "document.currentScript"} {
		if !strings.Contains(string(theme), needle) {
			t.Errorf("theme.js is missing %s", needle)
		}
	}
}
