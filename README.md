# orbitalui-htmx

Reusable server-rendered UI components for Go applications using templ,
HTMX, Alpine.js, and Tailwind CSS.

## Components

The original starter set provides:

- `TextInputField`, `NumberInputControl`, `DropdownField`, `CompactDropdown`,
  `TextAreaControl`, and `CharacterCount`.
- `HTMXPostForm`.
- `Icon` and `SelectOption`.

The reusable core set extracted from `ledger-agent/internal/ui/components`
adds:

- Buttons, status badges, alerts, state panels, and disclosures.
- Labelled fields, field errors, search fields, and an Alpine entity picker.
- URL-backed tab bars, segmented controls, table shells, sortable columns,
  and paging controls.
- Native dialog and confirmation components, plus a drawer component.
- A docked record `Inspector` with a standalone runtime for resizing,
  fullscreen, panel-only navigation, browser history, focus, and mobile overlays.
- KPI tiles, exact decimal amounts, money cells, and a debit/credit balance
  tool.
- Generic asynchronous regions, dependency panels, selection toolbars, and
  text/number formatting helpers.

Components that accept `templ.Component` children let the application keep
ownership of page content and endpoints. Components do not register routes or
make application data assumptions.

### Agent chat (agentchat)

The `agentchat` package is the shared agentic chat: an SSE `Stream` writer, the
thread, composer, starter, card and question-form components, validated
`ParseQuestions` and a safe Markdown renderer, driven in the browser by
`assets/agent-chat.js` and styled by `assets/agent-chat.css`. The library owns
the chat UI and the wire protocol; the app keeps the agent, tools, persistence
and auth. See `docs/agent-chat.md` for the contract.

### App shell, command palette, toasts and theme (v2 shell)

Optional pieces for a full product frame, styled by `assets/shell.css`
(load after `tokens.css` and `theme.css`) and run by `assets/shell.js` as plain
DOM delegation, so they work under Alpine's CSP build and beside a host
`window.LedgerUI` runtime. `shell.js` is independent of `orbital-ui.js` and of
Alpine (any load order): hosts that bring their own toast or palette system
load the component runtime alone. Toast, palette, theme-switch and shell
markup only works when `shell.js` is loaded; server-rendered toasts and the
ThemeSwitch radios degrade gracefully without it (the toast stays visible and
its close button does nothing), so load it whenever those components render. Full contracts are in `docs/design-system.md`
("App shell and runtime contracts").

```go
@components.AppShell(components.AppShellOpts{
	Brand:       brand(),                       // logo link slot
	Sidebar:     nav(),                         // SidebarMenu > SidebarSection > SidebarItem
	Footer:      components.SidebarUser(components.SidebarUserOpts{Name: "Ada"}, accountMenu()),
	Breadcrumbs: []components.Crumb{{Label: "Home", Href: "/"}, {Label: "Reports"}},
	Actions: templ.Join(components.TopbarSearch(components.TopbarSearchOpts{Placeholder: "Search or ask…"}),
		components.TopbarNewMenu(components.TopbarNewMenuOpts{Label: "New"}, newMenu())),
	Overlays: overlays(),                       // CommandPalette + ToastRegion
}, page)
```

- `AppShell` / `SidebarMenu` / `SidebarSection` / `SidebarItem` / `SidebarUser` / `TopbarSearch` / `TopbarNewMenu` / `MenuRow`: the frame. Sidebar items extend the existing `.ui-nav__*` look (`SidebarNav` stays for data-driven lists).
- `CommandPalette(CommandPaletteOpts{Endpoint, Groups, ...})`: ⌘K / Ctrl+K dialog fed by static groups and/or a JSON endpoint.
- `ToastRegion()` + `Toast(ToastOpts)`; `HX-Trigger: {"toast": {...}}` raises one client-side.
- `ThemeSwitch(ThemeSwitchOpts{StorageKey})` + `assets/theme.js` (pre-paint, external). A light-only app pins `<html data-theme="light">` and skips both. A `<meta name="theme-color" data-light="…" data-dark="…">` follows the active theme.
- Touch sizing tokens: `--oui-touch-target` (44px hit area), `--oui-topbar-control` (visual size of the phone search/New chips; the hit area stays 44px), `--oui-row-touch` (one min-height for phone nav, menu and palette rows; unset, rows inside AppShell keep desktop heights and SidebarNav/menu rows elsewhere keep 44px).
- `components.RegisterIcons(map[string]string)` adds host glyphs at init (a registered name overrides a built-in); palette rows may name them in `Icons`. `PaletteForm.QueryField` posts the palette's typed text under that field at submit time. `PaletteItem.Default` / `Promoted` / `QueryLabel` pick the first-selected row, the highlighted look and the "label “typed query”" row.

### Kit parity

The shared defaults match Kindred's kit (see `docs/design-system.md`, "Kit
parity"): `ButtonOpts.TrailingIcon` (and `IconRight` now draws on the right),
`CardOpts.Padded` and `Bare`, composable `DataList` / `DataItem`,
`RingMeterOpts.Px` with `--oui-ring-glow`, `ChipGroupOpts.Help` / `Error` /
`Required`, `SegmentedNav` and `TabNav` (href-driven), `SuggestedTagTitled`,
`IconSized`, the orb tokens `--oui-orb-a/-b/-c`, `--oui-card-title-family`,
`--oui-state-title-family`, `--oui-chip-height` and
`--oui-tab-count-active-bg/-fg`, and a cancelable `oui:palette-submit` event on
the command palette dialog (`detail.item`, `detail.query`, `detail.row`).

### Kit finish

`--oui-control-pad-x`, `--oui-control-ring-inset`, `--oui-tab-height-phone`,
`--oui-textarea-min-height` / `-resize`, `--oui-chip-suggested-ring` / `-halo`,
`--oui-chip-spark-checked` and `--oui-badge-dot` tune the remaining Kindred
differences; defaults keep existing hosts unchanged. New options:
`FieldOpts.ErrorAlert`, `FieldDescribedBy` (help gets the id `{id}-help`),
`FieldInputOpts.Numeric`, `FieldSelectOpts.PlaceholderDisabled`,
`ChipGroupOpts.SuggestedTitle`, `DataItemWith` (`Wide`, `Field`),
`StatePanelOpts.TitleTag` and the standalone `CardHeader`. See
`docs/design-system.md`, "Kit finish".

## Use

```go
import "github.com/orbitalbase/orbitalui-htmx/components"
```

Use the `.templ` sources to include this module in Tailwind's source scan.
Tailwind 4 CSS can register a local checkout with `@source`:

```css
@import "tailwindcss";
@source "../path/to/orbitalui-htmx/components";
```

For module-cache or vendored builds, point `@source` at the resolved module
directory or ship a generated stylesheet from the host application's build.

Import the design system before the application's own CSS:

```css
@import "<module>/assets/tokens.css";
@import "<module>/assets/base.css";   /* optional: type scale, .ui-prose, .ui-eyebrow, .ui-link */
@import "<module>/assets/motion.css"; /* optional: enter/press/shimmer, orb, ring and meter motion */
@import "<module>/assets/controls.css";
@import "<module>/assets/theme.css";
@import "<module>/assets/inspector.css"; /* only with Inspector */
```

`theme.css` styles every class the components emit (a test enforces this), so
an application restyles through tokens rather than component selectors. See
[docs/design-system.md](docs/design-system.md) for the token layers and
[docs/adoption](docs/adoption/README.md) for what belongs in the library versus
an application. `styles/components.css` is a deprecated alias that imports the
tokens and theme.

Layout primitives: `PageFrame` fixes a page's width (narrow, standard, wide,
full) and vertical rhythm; `FormGrid` and `FormActions` lay out
`LabelledField`s and their submit row.

`HTMXPostForm` renders a normal POST form (`method` and `action`) and upgrades
it with `hx-post`, so browser submission still works when HTMX is unavailable.
The component leaves target, swap strategy, indicator, authorization, CSRF,
validation, and response rendering to the application.

The dialog, drawer, confirm, entity-picker and dismissible-alert components run
on `assets/orbital-ui.js`. Serve it from `assets.Files`. **Load order:** a host
runtime that defines `window.LedgerUI` (if any) first, then `orbital-ui.js`,
then Alpine (`<script src="/static/orbital-ui.js" defer>` ahead of Alpine's own
script). The markup is safe for Alpine's CSP build (`alpinejs-csp`, for
`script-src 'self'`): it names `Alpine.data` factories (`ouiDialog`,
`ouiDrawer`, `ouiPicker`, `ouiDismiss`) registered on `alpine:init` and uses
argument-free handler references only. When the host already provides
`window.LedgerUI`, `orbital-ui.js` still registers those factories, delegating
to the host's `dialog()`, `drawer()` and `picker()`, and skips its own click
delegation. `orbital-ui.js` is the component runtime only; the shell runtime
(sidebar, palette, toasts, ThemeSwitch) is `assets/shell.js` and is loaded
separately, in any order (`<script src="/static/shell.js" defer>`).
`data-autogrow` textareas use CSS `field-sizing`, with a small
fallback in the same file. Other components use native HTML or HTMX attributes
and need no library-specific JavaScript. `node scripts/browser-check.mjs`
drives them in headless Chrome against both the standard and the CSP Alpine
builds, and in host-runtime mode.

## Gallery

`go run ./internal/gallery -out /tmp/oui-gallery` writes static pages that
render every component and state with the shipped CSS (no Tailwind), in four
token presets: `default`, `dark`, `kindred` and `kindred-dark`
(`internal/gallery/presets/kindred.css` maps Kindred's brand, accent, AI,
ink-action, radii and fonts onto `--oui-*`; pass `-kindred-fonts DIR` to copy
its woff2 files, otherwise system fonts are used). Each preset also has
`shell.html` and `shell-full.html`, whole `AppShell` pages that load the
runtime (press Ctrl/Cmd K, resize below 900px for the off-canvas sidebar), and
sections for shell parts, the command palette, toasts and the theme switch.

## Record inspectors

`Inspector` is the shared foundation for keeping record review beside a list.
It is separate from `Drawer`, which is a modal bottom sheet. Its browser runtime
does not depend on `window.LedgerUI`, Alpine expressions, or inline scripts.

The application renders a `.ui-inspector-host` element with
`data-orbital-inspector-host`, a list child with `data-inspector-list`, and one
always-mounted `Inspector`. Pass `Open: false` for the closed state so its ID
remains a stable swap target. Mark selectable rows with
`data-inspector-record={recordID}` and give their native links the host page's
selection URL.

```go
components.Inspector(components.InspectorOpts{
    ID: "template-inspector", Title: "Board resolution",
    Open: true, Resizable: true,
    CloseHref: "/templates?q=board",
    ExpandHref: "/templates?q=board&selected=123&panel=full",
    StorageKey: "orbitalui:templates:width",
}, approvedTemplateBody)
```

Serve the embedded `assets.Files` entries `inspector.css` and `inspector.js`
through the host's static handler, and load them as an external stylesheet and
deferred script. The stylesheet supplies structure using semantic CSS tokens
with fallbacks; each app supplies its theme. The parent workspace must constrain
its height and allow its flex children to shrink (`min-height: 0`) so the list
and panel body can scroll independently.

Native GETs render the same open/closed state for refresh, deep links and
JavaScript-free use. Enhanced navigation fetches that URL and replaces only the
inspector with the matching ID; it retains the list DOM, updates browser history
and processes new HTMX content. Forms must specify their application-owned POST
action explicitly. The runtime retains unsaved text/select/check values in memory
while switching panels, expanding, closing and returning during that page visit.
It does not autosave them or retain them after a full page reload.

`StateKeys` defaults to `"selected,panel"`; only changes to those parameters
qualify for panel-only navigation. The first key identifies the record or stack
for draft retention and row highlighting. Ledger can use `StateKeys: "d,dw"`
and its existing server-built stack URLs, with `BackHref` for drilling back.
`IgnoreEmptyQuery: true` can normalize empty filters such as `search=` to
missing values for this comparison. Enable it only when the host's GET handlers
give those states the same meaning; it is disabled by default.
Application-specific providers, stack validation, permissions and mutations stay
in the application. Mark native GET filter forms and pagination/clear links with
`data-inspector-list-nav` to keep their selection fields/URLs synchronized as
panels open and close. HTMX list/filter swaps can preserve the active panel;
the runtime rebases its chrome when `htmx:pushedIntoHistory` changes that context.

The resize handle supports pointer dragging and ArrowLeft/ArrowRight/Home/End.
Preferred width is stored under the supplied `StorageKey`. Fullscreen and mobile
overlays make the surrounding page inert and retain keyboard focus inside the
panel. Closing restores focus to the originating row link.

Opening slides in over 160 ms; closing slides out over 120 ms. Only transform
and opacity animate, with an ease-out curve and no bounce. Navigation cancels
obsolete motion, and `prefers-reduced-motion: reduce` skips it entirely.
Programmatic heading focus does not draw an outline; actionable controls retain
a visible keyboard focus indicator.

### Field focus

Load the embedded `controls.css` after the host theme to give `FieldInput` and
the shared `oui-control` input/select/textarea controls a consistent focus
treatment. `FieldInput` and the other `.field-control` fields draw a 1px ring
as a box-shadow (the box keeps its size); focus takes the action color with a
4px halo, and invalid fields retain their error color. `controls.css` itself
now styles `.oui-control` only, since `.field-control` is handled in
`theme.css`. Set `--control-focus-border` to override the focus color, or provide the host's
`--accent-dark` token. Native controls in the host application should use the
same treatment, while buttons, links, checkboxes and resize handles keep their
own visible keyboard focus states.

## Development

Generated `*_templ.go` files are committed. After changing a `.templ` file:

```sh
go tool templ generate
go test ./...
```
