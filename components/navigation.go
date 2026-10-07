// Navigation primitives (navigation.templ): the sidebar navigation, a
// breadcrumb trail, a native <details> menu and the workspace switcher built
// on it. Menus work without JavaScript; assets/menu.js adds outside-click and
// Escape dismissal and remembers collapsed sidebar sections.
package components

import "github.com/a-h/templ"

// NavLink is one sidebar destination.
type NavLink struct {
	Label  string
	Href   templ.SafeURL
	Icon   string // icon-set name
	Key    string // stable id, rendered as data-nav-key
	Active bool
	Badge  string // optional count or short label ("4", "New")
	Tone   Tone   // badge tone; "" neutral
}

// NavSection groups links under an optional heading. A section without a
// label renders as the primary block at the top of the sidebar. Collapsible
// sections use <details>; the section containing the active link is always
// rendered open so the current page is never hidden.
type NavSection struct {
	Label       string
	ID          string // stable id used to remember the collapsed state
	Links       []NavLink
	Collapsible bool
	Open        bool
}

// SidebarNavOpts configures SidebarNav.
type SidebarNavOpts struct {
	Label string // aria-label for the nav landmark; default "Primary"
	Class string
	// Rail renders the compact icon rail (.ui-nav--rail): labels become
	// visually hidden accessible names (links keep their title tooltip),
	// section headings collapse to a divider, badges to a dot. Collapsible
	// sections render open, since a rail has no heading to toggle.
	Rail bool
}

func sidebarNavClass(o SidebarNavOpts) string {
	return joinClass("ui-nav", modifier(o.Rail, "ui-nav--rail"), o.Class)
}

func sidebarNavLabel(o SidebarNavOpts) string {
	if o.Label == "" {
		return "Primary"
	}
	return o.Label
}

func navSectionOpen(s NavSection) bool {
	if s.Open {
		return true
	}
	for _, l := range s.Links {
		if l.Active {
			return true
		}
	}
	return false
}

func navLinkClass(l NavLink) string {
	return joinClass("ui-nav__link", modifier(l.Active, "is-active"))
}

func navBadgeClass(l NavLink) string {
	tone := ""
	if l.Tone != "" && l.Tone != ToneNeutral {
		tone = "ui-nav__badge--" + l.Tone.Modifier()
	}
	return joinClass("ui-nav__badge", tone)
}

// BreadcrumbsOpts configures BreadcrumbsWith.
//
//	CollapseOnPhone — at 42rem and below only the current crumb is shown, so
//	    a long trail never wraps in a topbar.
//	Label — the landmark name (default "Breadcrumb").
type BreadcrumbsOpts struct {
	CollapseOnPhone bool
	Label           string
	Class           string
}

func breadcrumbsClass(o BreadcrumbsOpts) string {
	return joinClass("ui-breadcrumbs", modifier(o.CollapseOnPhone, "ui-breadcrumbs--collapse"), o.Class)
}

func breadcrumbsLabel(o BreadcrumbsOpts) string {
	if o.Label == "" {
		return "Breadcrumb"
	}
	return o.Label
}

// Crumb is one breadcrumb. The last crumb is the current page and should
// have no Href.
type Crumb struct {
	Label string
	Href  templ.SafeURL
}

// MenuOpts configures Menu.
//
//	Label — accessible name for the trigger when it has no visible text.
//	Align — "end" right-aligns the panel to the trigger (default "start").
//	Placement — "top" opens upward (sidebar footers); default "bottom".
//	TriggerClass — extra classes for the <summary> trigger.
//	Width — "wide" for richer panels (workspace switcher).
type MenuOpts struct {
	Label        string
	Align        string
	Placement    string
	TriggerClass string
	Width        string
	// Open renders the panel expanded (documentation, screenshots, a menu
	// that must start open). Normal menus open on click.
	Open  bool
	ID    string
	Class string
}

func menuClass(o MenuOpts) string {
	align := ""
	if o.Align == "end" {
		align = "ui-menu--end"
	}
	place := ""
	if o.Placement == "top" {
		place = "ui-menu--top"
	}
	width := ""
	if o.Width == "wide" {
		width = "ui-menu--wide"
	}
	return joinClass("ui-menu", align, place, width, o.Class)
}

// MenuItemOpts configures MenuLink.
type MenuItemOpts struct {
	Icon     string
	Hint     string   // right-aligned muted text (a count, a note)
	Kbd      []string // keyboard shortcut keys shown right-aligned as keycaps ("⌘", "K")
	Danger   bool
	Selected bool
	Class    string
}

func menuItemClass(o MenuItemOpts) string {
	return joinClass("ui-menu__item", modifier(o.Danger, "ui-menu__item--danger"), modifier(o.Selected, "is-selected"), o.Class)
}

// WorkspaceSwitcherOpts configures WorkspaceSwitcher, the compact scope
// control at the top of a sidebar. It shows one line of identity (name plus
// a short subtitle) and discloses details and switching in its menu panel.
type WorkspaceSwitcherOpts struct {
	Name       string
	Subtitle   string
	Kind       string // accessible prefix, e.g. "Company", "Firm"
	StatusTone Tone   // optional status dot on the avatar
	// Compact shows the avatar only (.ui-switcher--compact), for a nav rail;
	// the name and subtitle stay as the trigger's accessible name.
	Compact bool
	Class   string
}

func switcherLabel(o WorkspaceSwitcherOpts) string {
	if o.Kind == "" {
		return "Workspace menu: "
	}
	return o.Kind + " workspace menu: "
}
