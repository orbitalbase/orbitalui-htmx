// Drawer types (drawer.templ) — the generic full-screen bottom-sheet
// component, sibling of ModalShell/dialog conventions (docs/ui-components.md
// §3d). NOT a native <dialog>: a plain fixed overlay + panel so a partial-
// height sheet with a transform transition behaves predictably (native
// dialog cannot do that cleanly). Behaviour lives entirely in
// assets/orbital-ui.js (window.LedgerUI.drawer): openDrawer/closeDrawer,
// Esc + backdrop hit-test via the Alpine factory, focus restore, body scroll
// lock — wired with data-* attributes, zero per-page JS.
package components

// DrawerOpts configures Drawer. Pages keep the field set small — everything
// else (subtitle copy, content) is template/children territory.
type DrawerOpts struct {
	ID    string // overlay id — openers target it via data-open-drawer={ID}; the scrollable content region id = ID + "-content"
	Title string // heading; the panel's aria-labelledby target
	// Subtitle is an optional muted line under the title (renders nothing
	// when empty).
	Subtitle string
	// CloseLabel is the × button's accessible name (default "Close").
	CloseLabel string
	// Fullscreen renders a full-screen toggle in the header: the panel starts
	// at its normal height (calc(100% - 3vh)) and the toggle expands it to the
	// whole viewport with the grab-handle and top rounding removed. It exists
	// for reading surfaces — a long log or reasoning trace is the reason to
	// take the whole screen, and taking it must not cost a navigation.
	Fullscreen bool

	// Tabbed turns the sheet into a multi-document surface: the header renders
	// a horizontally scrollable TAB STRIP (and the visible title becomes
	// screen-reader-only, because each tab's own content carries its heading).
	//
	// Pages fill it with elements carrying data-log-url / data-log-key /
	// data-log-title; a host tab runtime (ledger-agent's LedgerUI.logTabs) opens one tab per record, keeps
	// the strip scrolled to the active tab, and closes it from the × (or the
	// keyboard). See docs/ui-components.md §3d for the full contract.
	Tabbed bool
}

// DrawerTabsID returns the id of a tabbed drawer's tab strip (ID + "-tabs") —
// the element the host tab runtime renders tabs into. It is server-rendered and empty on
// purpose: tab state is per-reading-session client state (like the nav rail and
// the drill width), while each tab's CONTENT stays server-rendered by the
// drawer's own fragment endpoint.
func DrawerTabsID(id string) string { return id + "-tabs" }

// DrawerTitleID returns the id of a drawer's h2 (ID + "__title") for
// aria-labelledby wiring — the same scheme as dialogs (ModalTitleID).
func DrawerTitleID(id string) string { return id + "__title" }

// DrawerContentID returns the scrollable content region id (ID +
// "-content"). By convention this region is the htmx target for
// fragment-loaded drawer content: pages mount one drawer shell and load the
// inner form with hx-get="{url}" hx-target="#{ID}-content" hx-swap="innerHTML"
// (docs/ui-components.md §3d).
func DrawerContentID(id string) string { return id + "-content" }

// drawerPanelClass joins the panel base class with the tabbed modifier, which
// is what CSS and the runtime key the strip layout off.
func drawerPanelClass(o DrawerOpts) string {
	if o.Tabbed {
		return "drawer-panel drawer-panel--tabbed"
	}
	return "drawer-panel"
}

// drawerTabsLabel is the tab strip's accessible name.
func drawerTabsLabel(o DrawerOpts) string {
	if o.Title != "" {
		return o.Title + " — open records"
	}
	return "Open records"
}

// drawerCloseLabel resolves the × button's accessible name.
func drawerCloseLabel(o DrawerOpts) string {
	if o.CloseLabel != "" {
		return o.CloseLabel
	}
	return "Close"
}

// drawerFullscreenLabel is the full-screen toggle's accessible name and tooltip
// for a state. orbital-ui.js uses the same two strings from this owner's behaviour, so
// the server-rendered label and the label after a toggle can never disagree —
// the toggle is driven by the button's OWN state, and both icons ship in the
// markup (CSS reveals one per state), so no JavaScript touches icon markup.
func drawerFullscreenLabel(full bool) string {
	if full {
		return "Contract back to the sheet"
	}
	return "Expand this panel to the whole screen"
}
