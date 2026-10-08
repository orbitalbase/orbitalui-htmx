// AppShell and its parts (shell.templ): the product frame shared by apps — a
// quiet canvas sidebar, an inset rounded main sheet, a sticky glass topbar,
// and an off-canvas sidebar with a scrim on phones. Every region is a slot
// (templ.Component) or a typed option; nothing in it names a product.
//
// The sidebar items are the existing .ui-nav__* family (SidebarNav's look)
// extended with Count / Percent / Done / Leading / Tone, so a data-driven
// SidebarNav and a hand-composed SidebarMenu look identical. Behaviour (burger,
// scrim, Escape, inert while closed) lives in assets/shell.js and is
// plain DOM delegation, so it runs under Alpine's CSP build and alongside a
// host runtime. Styles are in assets/shell.css.
package components

import (
	"strconv"

	"github.com/a-h/templ"
)

// AppShellOpts configures AppShell.
//
//	Brand — the logo link at the top of the sidebar (a slot; the shell adds
//	    the phone-only close button next to it).
//	Sidebar — the scrolling navigation area: SidebarMenu / SidebarSection /
//	    SidebarItem, or any markup.
//	Footer — pinned to the bottom of the sidebar (SidebarUser).
//	Breadcrumbs — the topbar trail (BreadcrumbsWith, phone-collapsing).
//	    Crumbs replaces it with any component (a page title, tabs).
//	Actions — the topbar's right side (TopbarSearch, TopbarNewMenu, buttons).
//	Overlays — rendered after the frame: CommandPalette, ToastRegion, drawers.
//	FullWidth — the sheet's main area is not wrapped in the padded, centred
//	    page column and does not scroll itself, for workspaces (chat, editors,
//	    split views) that own their own layout.
//	ID — id of the sidebar, which the burger controls (default "app-sidebar").
//	SidebarLabel — aria-label of the sidebar landmark (default "Sidebar").
//	Class — extra classes on the frame.
//
// The shell marks its sidebar and sheet with .ui-vt-sidebar / .ui-vt-main, the
// view-transition names motion.css defines; they are inert until the host adds
// `@view-transition { navigation: auto; }`.
type AppShellOpts struct {
	Brand        templ.Component
	Sidebar      templ.Component
	Footer       templ.Component
	Breadcrumbs  []Crumb
	Crumbs       templ.Component
	Actions      templ.Component
	Overlays     templ.Component
	FullWidth    bool
	ID           string
	SidebarLabel string
	Class        string
}

func shellClass(o AppShellOpts) string {
	return joinClass("ui-shell", modifier(o.FullWidth, "ui-shell--full"), o.Class)
}

func shellSidebarID(o AppShellOpts) string {
	if o.ID == "" {
		return "app-sidebar"
	}
	return o.ID
}

func shellSidebarLabel(o AppShellOpts) string {
	if o.SidebarLabel == "" {
		return "Sidebar"
	}
	return o.SidebarLabel
}

// SidebarSectionOpts configures SidebarSection. A section without a Label is
// the unlabelled primary block.
type SidebarSectionOpts struct {
	Label string
	// ID names the heading for aria-labelledby; derived from the label when
	// empty, so set it when two sections share a label.
	ID    string
	Class string
}

func sidebarSectionID(o SidebarSectionOpts) string {
	if o.ID != "" {
		return o.ID
	}
	id := make([]byte, 0, len(o.Label))
	dash := false
	for _, r := range o.Label {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			id = append(id, byte(r))
			dash = false
		case r >= 'A' && r <= 'Z':
			id = append(id, byte(r-'A'+'a'))
			dash = false
		default:
			if !dash && len(id) > 0 {
				id = append(id, '-')
				dash = true
			}
		}
	}
	return "oui-nav-" + string(id)
}

// SidebarItemOpts configures SidebarItem.
//
//	Label, Href — the destination. Active marks the current page.
//	Icon — icon-set name; Leading is a slot for an avatar or any lead element
//	    and replaces the icon.
//	Tone — tints the icon (ToneAI for the assistant entry, ToneBrand,
//	    ToneAccent); neutral by default.
//	Count — a pill with a number (0 hides it; above 99 shows "99+").
//	    CountLabel is the phrase after the number for screen readers
//	    ("waiting on you" reads "4 waiting on you").
//	Percent — a quiet percentage (use Pct); Done replaces it with a check,
//	    for a completed record. DoneLabel is the check's accessible name.
//	Dense — a 30px row for long lists (recent chats). Add — the quiet
//	    "add another" row with a boxed plus.
//	Key — rendered as data-nav-key.
type SidebarItemOpts struct {
	Label      string
	Href       templ.SafeURL
	Icon       string
	Leading    templ.Component
	Tone       Tone
	Active     bool
	Count      int
	CountLabel string
	Percent    *int
	Done       bool
	DoneLabel  string
	Dense      bool
	Add        bool
	Key        string
	Class      string
}

// Pct returns a pointer to n, for SidebarItemOpts.Percent.
func Pct(n int) *int { return &n }

func sidebarItemClass(o SidebarItemOpts) string {
	tone := ""
	if o.Tone != "" && o.Tone != ToneNeutral {
		tone = "ui-nav__link--" + o.Tone.Modifier()
	}
	return joinClass("ui-nav__link", modifier(o.Active, "is-active"), modifier(o.Dense, "ui-nav__link--dense"), modifier(o.Add, "ui-nav__link--add"), tone, o.Class)
}

func sidebarCount(n int) string {
	if n > 99 {
		return "99+"
	}
	return strconv.Itoa(n)
}

func sidebarCountLabel(o SidebarItemOpts) string {
	if o.CountLabel == "" {
		return ""
	}
	return strconv.Itoa(o.Count) + " " + o.CountLabel
}

func sidebarPercent(p int) string {
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return strconv.Itoa(p) + "%"
}

// SidebarUserOpts configures SidebarUser.
//
//	Name — shown beside the avatar.
//	Avatar — a slot; defaults to an initials Avatar of Name.
//	Label — the trigger's accessible name (default "Account menu").
//	Open — render the menu expanded (documentation, screenshots).
type SidebarUserOpts struct {
	Name   string
	Avatar templ.Component
	Label  string
	Open   bool
}

func sidebarUserLabel(o SidebarUserOpts) string {
	if o.Label == "" {
		return "Account menu"
	}
	return o.Label
}

// TopbarSearchOpts configures TopbarSearch, the button that opens a
// CommandPalette.
//
//	Placeholder — the visible hint ("Search or ask…").
//	Shortcut — key caps shown on the right (default ⌘ K; the runtime shows
//	    Ctrl instead of ⌘ off Apple platforms). Set NoShortcut to hide them.
//	Label — the accessible name (default Placeholder).
//	Target — id of the palette to open; empty opens the first one.
type TopbarSearchOpts struct {
	Placeholder string
	Shortcut    []string
	NoShortcut  bool
	Label       string
	Target      string
	Class       string
}

func topbarSearchKeys(o TopbarSearchOpts) []string {
	if o.NoShortcut {
		return nil
	}
	if len(o.Shortcut) == 0 {
		return []string{"⌘", "K"}
	}
	return o.Shortcut
}

func topbarSearchPlaceholder(o TopbarSearchOpts) string {
	if o.Placeholder == "" {
		return "Search…"
	}
	return o.Placeholder
}

func topbarSearchLabel(o TopbarSearchOpts) string {
	if o.Label != "" {
		return o.Label
	}
	return topbarSearchPlaceholder(o)
}

// TopbarNewMenuOpts configures TopbarNewMenu: a small secondary button with a
// leading icon that opens a menu panel (MenuLink / MenuLabel / MenuSeparator
// children, or forms) aligned to the right edge.
//
//	Label — the button text ("New"); hidden on phones, where the icon stays
//	    and Label remains the accessible name.
//	Icon — default "plus".
type TopbarNewMenuOpts struct {
	Label string
	Icon  string
	Class string
}

func topbarNewIcon(o TopbarNewMenuOpts) string {
	if o.Icon == "" {
		return "plus"
	}
	return o.Icon
}

func sidebarAddIcon(o SidebarItemOpts) string {
	if o.Icon == "" {
		return "plus"
	}
	return o.Icon
}

func sidebarDoneLabel(o SidebarItemOpts) string {
	if o.DoneLabel == "" {
		return "Complete"
	}
	return o.DoneLabel
}
