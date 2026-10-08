// The app's single icon set: 24×24 stroke SVG paths (1.5px, round caps),
// ported from the ledgerv2 Icon component
// (ledgerv2/ledger-app/src/lib/components/ui/Icon.svelte). Icons render with
// stroke=currentColor and no fill, so they inherit the surrounding text
// color — tone comes from CSS, never baked into the artwork. Custom additions
// (payments, tax, cashflow, recon, periods/calendar, alert, chevrons, book,
// refresh, external, search) follow the same geometric idiom so the set reads
// as one family.
package components

import (
	"strconv"
	"strings"
	"sync"
)

// iconPaths maps icon name → inner SVG markup (one or more path `d`
// subpaths). Names are stable across phases — the sidebar nav (internal/ui
// nav.go), page headers and component primitives all reference them.
var iconPaths = map[string]string{
	// Core conversation and form icons from the original orbitalui set.
	"spark":           "M12 3l1.7 5.3L19 10l-5.3 1.7L12 17l-1.7-5.3L5 10l5.3-1.7L12 3 M19 16l.8 2.2L22 19l-2.2.8L19 22l-.8-2.2L16 19l2.2-.8L19 16",
	"send":            "M22 2l-7 20-4-9-9-4z M22 2L11 13",
	"paperclip":       "M21.4 11.1l-8.5 8.5a5.5 5.5 0 0 1-7.8-7.8l9.2-9.2a3.7 3.7 0 0 1 5.2 5.2l-9.2 9.2a1.8 1.8 0 0 1-2.6-2.6l8.5-8.5",
	"pin":             "M20 10c0 5-8 12-8 12S4 15 4 10a8 8 0 1 1 16 0z M12 7.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5",
	"briefcase":       "M3 7h18v14H3z M8 7V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M3 12h18M10 12v2h4v-2",
	"building":        "M4 3h16v18H4z M9 7h1m4 0h1M9 11h1m4 0h1M9 15h1m4 0h1M10 21v-3h4v3",
	"money":           "M3 5h18v14H3z M7 9h.01m10 6h.01M12 9v6m2-4.5c0-.8-.9-1.5-2-1.5s-2 .7-2 1.5 1 1.2 2 1.5 2 .7 2 1.5-.9 1.5-2 1.5-2-.7-2-1.5",
	"chevron":         "M7 10l5 5 5-5",
	"info":            "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z M12 11v5m0-8h.01",
	"clock":           "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z M12 7v5l3 2",
	"arrow-right":     "M5 12h14m-6-6 6 6-6 6",
	"sparkle-small":   "M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8L12 3",
	"briefcase-small": "M4 7h16v13H4z M9 7V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2M4 12h16",
	// ── ported from ledgerv2 Icon.svelte (verbatim paths) ──────────────
	"dashboard": "M3.75 4.5h6v6h-6zM14.25 4.5h6v6h-6zM3.75 13.5h6v6h-6zM14.25 13.5h6v6h-6z",
	"accounts":  "M4 5.25h16M4 9.25h16M4 13.25h10M4 17.25h7",
	"bank":      "M3.75 9.75 12 4.5l8.25 5.25M5.25 9.75v8.5m4.5-8.5v8.5m4.5-8.5v8.5m4.5-8.5v8.5M3.75 18.25h16.5",
	"journal":   "M5.25 4.5h11a2 2 0 0 1 2 2v13h-11a2 2 0 0 1-2-2zM18.25 19.5v-13M8.5 9h6m-6 3.5h6",
	"document":  "M6.75 3.75h7.5l3 3v13.5h-10.5zM14.25 3.75v3h3M9.5 12h5m-5 3.5h5",
	"recon":     "M9.5 14.5 5 19m0 0v-4m0 4h4M14.5 9.5 19 5m0 0v4m0-4h-4M5 5l14 14",
	"parties":   "M9 11.25a3.25 3.25 0 1 0 0-6.5 3.25 3.25 0 0 0 0 6.5zM3.75 19.5c0-2.9 2.35-5.25 5.25-5.25s5.25 2.35 5.25 5.25M15.5 5.1a3.25 3.25 0 0 1 0 5.8M17 14.6c1.95.75 3.25 2.7 3.25 4.9",
	"assets":    "M12 3.75 20.25 8 12 12.25 3.75 8zM3.75 8v8l8.25 4.25L20.25 16V8M12 12.25v8",
	"reports":   "M4.5 19.5h15M6.75 16.5v-6m4.5 6V7.5m4.5 9v-3.75M12 4.5l7.5 3",
	"ai":        "M12 4.25 14 10l5.75 2L14 14l-2 5.75L10 14l-5.75-2L10 10z",
	"chat":      "M4.5 5.75h15v10h-8.5L7 19.25v-3.5H4.5zM8 9.25h8m-8 3h5",
	"settings":  "M5.5 7h13M5.5 12h13M5.5 17h13M9 5.25V8.75M15 10.25v3.5M7.5 15.25v3.5",
	"plus":      "M12 5v14M5 12h14",
	"close":     "M6 6l12 12M18 6 6 18",
	"check":     "M5 12.5 10 17.5 19 7",
	"scale":     "M12 4.5v15m-7-13h14M7.5 6.5 5 12.5a2.6 2.6 0 0 0 5 0zM16.5 6.5 14 12.5a2.6 2.6 0 0 0 5 0zM8.5 19.5h7",
	"keyboard":  "M3.75 7h16.5v10H3.75zM6.5 10h1m2.5 0h1m2.5 0h1m2.5 0h1M7.5 14h9",

	// ── crafted additions, same 24×24 / 1.5px stroke idiom ──────────────
	// Calendar/periods — the fiscal-period month grid.
	"periods":  "M4.5 6.25h15V19h-15zM4.5 10.25h15M8.25 4v4.75M15.75 4v4.75",
	"calendar": "M4.5 6.25h15V19h-15zM4.5 10.25h15M8.25 4v4.75M15.75 4v4.75",
	// Tax — percent mark.
	"tax": "M10 7.75a2.25 2.25 0 1 0-4.5 0 2.25 2.25 0 0 0 4.5 0zM18.5 16.25a2.25 2.25 0 1 0-4.5 0 2.25 2.25 0 0 0 4.5 0zM6.75 17.25l10.5-10.5",
	// Cash flow — money moving both directions.
	"cashflow": "M4.25 9.75h9.5M13.75 9.75l-3-3M13.75 9.75l-3 3M19.75 14.25h-9.5M10.25 14.25l3-3M10.25 14.25l3 3",
	// Payments — a banknote (settling money).
	"payments": "M3.75 8.25h16.5v7.5H3.75zM12 13.5a1.25 1.25 0 1 0 0-2.5 1.25 1.25 0 0 0 0 2.5zM7.5 11.5h.01M16.5 12.5h.01",
	// Alert — warning triangle.
	"alert": "M12 4.5 20.75 19.25H3.25zM12 9.75v4M12 16.75v.25",
	// Chevrons.
	"chevron-down":  "M6 9.75l6 6 6-6",
	"chevron-up":    "M6 14.25l6-6 6 6",
	"chevron-left":  "M14.25 6.75l-5.25 5.25 5.25 5.25",
	"chevron-right": "M9.75 6.75l5.25 5.25-5.25 5.25",
	// Book — the ledger itself.
	"book": "M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z",
	// Refresh — circular arrows (feather refresh-cw idiom).
	"refresh": "M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15",
	// External — opens in a new context.
	"external": "M14.25 5.25H5.25v13.5h13.5V9.75M9.75 14.25 18.75 5.25M14.25 5.25h4.5v4.5",
	// Search — magnifier.
	"search": "M18.5 11a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0zM20.25 20.25l-4.35-4.35",
	// Expand / contract — the drill stack's full-screen toggle (corners).
	"expand":   "M8.25 4.5H4.5v3.75M15.75 4.5h3.75v3.75M8.25 19.5H4.5v-3.75M15.75 19.5h3.75v-3.75",
	"contract": "M8.25 8.25V4.5H4.5M15.75 8.25V4.5h3.75M8.25 15.75V19.5H4.5M15.75 15.75V19.5h3.75",
	// Provider — a chip: the model/provider behind an LLM call (agent
	// observability dashboard).
	"provider": "M8.25 8.25h7.5v7.5h-7.5zM9.75 4.5v3.75M14.25 4.5v3.75M9.75 15.75v3.75M14.25 15.75v3.75M4.5 9.75h3.75M4.5 14.25h3.75M15.75 9.75h3.75M15.75 14.25h3.75",
	// Tool — a wrench: one agent tool call (agent observability dashboard).
	"tool": "M19.5 6.75a4.25 4.25 0 0 1-5.6 5.6l-6.4 6.4a1.75 1.75 0 1 1-2.5-2.5l6.4-6.4a4.25 4.25 0 0 1 5.6-5.6l-2.6 2.6 1.85 1.85z",
	// Masterdata — a reference table: a framed grid with a header row and
	// two body rows. Reads as "the catalog of values other records point
	// at", distinct from the plain rule-list of "accounts".
	"masterdata": "M4.5 5.25h15v13.5h-15zM4.5 9.75h15M4.5 14.25h15M10.5 9.75v9",

	// ── workspace shell additions (same idiom) ─────────────────────────
	// Selector — the up/down chevrons of a switcher or select.
	"selector": "M8 9.5l4-4 4 4M8 14.5l4 4 4-4",
	// Bell — notifications.
	"bell": "M6.75 9.5a5.25 5.25 0 0 1 10.5 0c0 5 2.25 6.75 2.25 6.75h-15S6.75 14.5 6.75 9.5zM10.25 19.25a1.9 1.9 0 0 0 3.5 0",
	// Home — the root of a breadcrumb trail.
	"home": "M4.5 10.5 12 4.5l7.5 6V19.5h-5v-5h-5v5h-5z",
	// User / users — one person, a team.
	"user":  "M12 11.25a3.75 3.75 0 1 0 0-7.5 3.75 3.75 0 0 0 0 7.5zM5.25 20.25c0-3.73 3.02-6.75 6.75-6.75s6.75 3.02 6.75 6.75",
	"users": "M9 11.25a3.25 3.25 0 1 0 0-6.5 3.25 3.25 0 0 0 0 6.5zM3.75 19.5c0-2.9 2.35-5.25 5.25-5.25s5.25 2.35 5.25 5.25M15.5 5.1a3.25 3.25 0 0 1 0 6.3M17 14.5a5.25 5.25 0 0 1 3.25 5",
	// Logout — leaving the session.
	"logout": "M9.75 19.5h-4.5v-15h4.5M15 16.5l4.5-4.5L15 7.5M19.5 12H9.75",
	// Shield — compliance and protection.
	"shield": "M12 3.75 19.5 6.5v5.25c0 4.5-3.25 7.75-7.5 8.75-4.25-1-7.5-4.25-7.5-8.75V6.5zM9 12l2.25 2.25L15.5 10",
	// Mail — email preferences and invitations.
	"mail": "M3.75 6h16.5v12H3.75zM4.25 6.5 12 12.75l7.75-6.25",
	// Gift — partner offers and perks.
	"gift": "M4.5 9.75h15v3h-15zM5.75 12.75v6.75h12.5v-6.75M12 9.75v9.75M12 9.75S10.5 4.5 8.25 5.25 8.5 9.75 12 9.75zM12 9.75s1.5-5.25 3.75-4.5-.25 4.5-3.75 4.5z",
	// Trending — opportunities and growth.
	"trending": "M3.75 16.5 9.75 10.5l3.75 3.75 6.75-6.75M15 7.5h5.25v5.25",
	// Pie — shares and capitalisation.
	"pie": "M12 3.75v8.25h8.25A8.25 8.25 0 1 1 12 3.75zM15 3.95A8.27 8.27 0 0 1 20.05 9H15z",
	// Folder — document folders.
	"folder": "M3.75 6.75a1.5 1.5 0 0 1 1.5-1.5h4.5l2 2.25h7a1.5 1.5 0 0 1 1.5 1.5v8.25a1.5 1.5 0 0 1-1.5 1.5H5.25a1.5 1.5 0 0 1-1.5-1.5z",
	// Inbox — work queues.
	"inbox": "M3.75 12.75 6 5.25h12l2.25 7.5v6H3.75zM3.75 12.75h4.5l1.5 2.25h4.5l1.5-2.25h4.5",
	// Layers — templates and packages.
	"layers": "M12 3.75 20.25 8.25 12 12.75 3.75 8.25zM3.75 12.25 12 16.75l8.25-4.5M3.75 16.25 12 20.75l8.25-4.5",
	// More — an overflow menu: three solid 3px dots (a stroke dot would be too faint).
	"more": "M5.25 12a.75.75 0 1 0 1.5 0 .75.75 0 1 0-1.5 0M11.25 12a.75.75 0 1 0 1.5 0 .75.75 0 1 0-1.5 0M17.25 12a.75.75 0 1 0 1.5 0 .75.75 0 1 0-1.5 0",
	// Edit — a pencil.
	"edit": "M15.75 4.5 19.5 8.25 8.25 19.5H4.5v-3.75zM13.5 6.75l3.75 3.75",
	// Trash — permanent removal.
	"trash": "M4.5 6.75h15M9.75 6.75V4.5h4.5v2.25M6.75 6.75l.75 12.75h9l.75-12.75M10.25 10.5v5.25M13.75 10.5v5.25",
	// Archive — reversible removal from the active register.
	"archive": "M3.75 4.5h16.5v3.75H3.75zM5.25 8.25v11.25h13.5V8.25M9.75 12h4.5",
	// Lock — restricted or private.
	"lock": "M6 10.5h12v9.75H6zM8.25 10.5V7.5a3.75 3.75 0 0 1 7.5 0v3",
	// Globe — public network and jurisdictions.
	"globe": "M12 3.75a8.25 8.25 0 1 0 0 16.5 8.25 8.25 0 0 0 0-16.5zM3.75 12h16.5M12 3.75c2.25 2.25 3.25 5 3.25 8.25s-1 6-3.25 8.25c-2.25-2.25-3.25-5-3.25-8.25s1-6 3.25-8.25z",
	// Help — guidance.
	"help": "M12 3.75a8.25 8.25 0 1 0 0 16.5 8.25 8.25 0 0 0 0-16.5zM9.75 9.5a2.25 2.25 0 1 1 3.4 1.95c-.7.4-1.15.95-1.15 1.8v.25M12 16.5h.01",
	// Signature — a signing request.
	"signature": "M3.75 17.25c2.25 0 3-6 4.5-6s.75 4.5 2.25 4.5 2.25-3 3.75-3 1.5 2.25 3 2.25M3.75 20.25h16.5",
	// Filter — narrowing a register.
	"filter": "M4.5 5.25h15l-6 7.5v6l-3-1.5v-4.5z",
	// Download / upload.
	"download": "M12 4.5v10.5M7.5 10.5 12 15l4.5-4.5M4.5 19.5h15",
	"upload":   "M12 15V4.5M7.5 9 12 4.5 16.5 9M4.5 19.5h15",
	// Eye — preview.
	"eye": "M2.75 12S6 5.75 12 5.75 21.25 12 21.25 12 18 18.25 12 18.25 2.75 12 2.75 12zM12 14.75a2.75 2.75 0 1 0 0-5.5 2.75 2.75 0 0 0 0 5.5z",
	// Link — a connection between records or organisations.
	"link": "M10 14a3.5 3.5 0 0 0 5 0l3-3a3.5 3.5 0 0 0-5-5l-.75.75M14 10a3.5 3.5 0 0 0-5 0l-3 3a3.5 3.5 0 0 0 5 5l.75-.75",
	// Workflow — a process with steps.
	"workflow": "M4.5 4.5h6v4.5h-6zM13.5 15h6v4.5h-6zM7.5 9v3.75a2.25 2.25 0 0 0 2.25 2.25h3.75",
	// Sidebar — the collapse toggle for the navigation panel.
	"sidebar": "M3.75 4.5h16.5v15H3.75zM9.75 4.5v15",
	// Shell and feedback glyphs: burger, theme switch, toast tones.
	"menu":           "M4 7h16M4 12h16M4 17h16",
	"sun":            "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z M12 2.5v2m0 15v2M4.6 4.6l1.4 1.4m12 12 1.4 1.4M2.5 12h2m15 0h2M4.6 19.4 6 18m12-12 1.4-1.4",
	"moon":           "M20.5 13.2A8.5 8.5 0 1 1 10.8 3.5a6.8 6.8 0 0 0 9.7 9.7z",
	"monitor":        "M3.5 4.5h17v11.5h-17z M8.5 20h7M12 16v4",
	"circle-check":   "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z M8.5 12.2l2.4 2.4 4.6-4.9",
	"circle-x":       "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z M9 9l6 6m0-6-6 6",
	"triangle-alert": "M12 3.5 21.5 20h-19z M12 10v4.5m0 2.5h.01",
	// Target (concentric rings) and list (bulleted rows).
	"target": "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z M12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10z M12 11a1 1 0 1 0 0 2 1 1 0 0 0 0-2",
	"list":   "M9 6.5h11M9 12h11M9 17.5h11M4.5 6.5h.01M4.5 12h.01M4.5 17.5h.01",
}

// hostIcons holds glyphs registered by the host app; lookups consult it
// before iconPaths.
var (
	hostIconsMu sync.RWMutex
	hostIcons   = map[string]string{}
)

// RegisterIcons adds host glyphs to the icon set: name → inner SVG path data,
// the same format as the built-ins (24×24 viewBox, stroke artwork, one or more
// path `d` subpaths). Call it at init, before serving; it is safe for
// concurrent use but is not meant for per-request changes. A registered name
// replaces a built-in of the same name (the host's glyph wins), and a later
// registration of a name replaces an earlier one. Empty names and empty path
// data are ignored. Registered icons work anywhere an icon name does,
// including CommandPaletteOpts.Icons.
func RegisterIcons(icons map[string]string) {
	hostIconsMu.Lock()
	defer hostIconsMu.Unlock()
	for name, d := range icons {
		if name != "" && d != "" {
			hostIcons[name] = d
		}
	}
}

// fallbackIcon is rendered for unknown icon names — a neutral circle so a
// typo degrades to an empty outline, never a blank space.
const fallbackIcon = "M18.5 11a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0z"

// Path returns the SVG path data for name, or the fallback circle when the
// name is unknown. Exported so tests and later phases can enumerate the set.
func Path(name string) string {
	hostIconsMu.RLock()
	d, ok := hostIcons[name]
	hostIconsMu.RUnlock()
	if ok {
		return d
	}
	if d, ok := iconPaths[name]; ok {
		return d
	}
	return fallbackIcon
}

// iconClass joins optional CSS classes for the Icon svg. Separate from Path
// so templ stays declarative and the class list stays testable.
func iconClass(class ...string) string {
	return strings.Join(class, " ")
}

// iconSizedClass puts the size modifier for px (12, 14, 16, 20, 24) ahead of
// the caller's classes; any other px adds none.
func iconSizedClass(px int, class []string) []string {
	switch px {
	case 12, 14, 16, 20, 24:
		return append([]string{"ui-icon--" + strconv.Itoa(px)}, class...)
	}
	return class
}
