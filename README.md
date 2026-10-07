# orbitalui-htmx

Reusable server-rendered UI components for Go applications using templ,
HTMX, Alpine.js, and Tailwind CSS.

## Components

The original starter set provides:

- `TextInputField`, `NumberInputControl`, `DropdownField`, `CompactDropdown`,
  `TextAreaControl`, and `CharacterCount`.
- `HTMXPostForm`, chat messages and threads, summary cards, suggestion chips,
  assistant avatars, and loading indicators.
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

### Chat surfaces

`ChatThread` supplies a labelled, polite live log. Compose it with `UserMessage`,
`AssistantMessage`, and `AssistantAvatar`. For cited answers, task links or other
server-rendered content, `AssistantMessageContent(templ.Component)` provides the
same message chrome without converting application-owned content to plain text.
All plain message text remains escaped by templ.

`SuggestionSubmitChip(label, name, value)` submits a named value through its
containing native form. Unlike the Alpine `SuggestionChip`, it needs no runtime
expression and works without JavaScript under a strict CSP. `HTMXPostForm` can
append reply fragments to the thread; the host owns endpoints, CSRF, authorization,
conversation context, loading/error recovery, citations and any actual tool calls.
These components provide a chat interface, not an agent backend.

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

The dialog, drawer, and entity-picker components currently use the
`window.LedgerUI` Alpine helper contract from the source application. The host
must provide the matching runtime before using those components. Other
components use native HTML or HTMX attributes and need no library-specific
JavaScript.

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
treatment. The active border changes color without an outer outline; keyboard
focus uses an additional inset pixel. Invalid fields retain their error color.
Set `--control-focus-border` to override the focus color, or provide the host's
`--accent-dark` token. Native controls in the host application should use the
same treatment, while buttons, links, checkboxes and resize handles keep their
own visible keyboard focus states.

## Development

Generated `*_templ.go` files are committed. After changing a `.templ` file:

```sh
go tool templ generate
go test ./...
```
