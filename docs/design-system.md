# Orbital UI design system

Orbital UI ships a complete default theme for every class its components
emit, so a consuming application gets a polished, consistent workspace
without restyling the library. The look is a calm, light, record-first
workspace: cool grays, one indigo brand accent, crisp 1px borders, soft
paper-like elevation and dense-but-readable 14px controls.

## Files

| File | Purpose |
|---|---|
| `assets/tokens.css` | Primitive scales, semantic roles and the full dark mode. Import first. |
| `assets/base.css` | Optional: type scale (`.t-*`), `.ui-eyebrow`, `.ui-link`, `.ui-divider`, `.ui-kbd`, `.ui-prose`, skip link, selection. Class-scoped; no element resets. |
| `assets/motion.css` | Optional: `.ui-enter-up`, `.ui-stagger`, `.ui-press`, `.ui-shimmer`, `.ui-magic-fill`, `.ui-caret`, `.ui-typing`, orb states, ring and meter draw-in, view-transition helpers. Reduced-motion safe. |
| `assets/theme.css` | Default styles for every `ui-*`, `field-*`, `tab-*`, `kpi-*`, `pager__*` and table class. |
| `assets/shell.css` | Optional: `AppShell` frame, topbar, sidebar item extras, `CommandPalette` and toast styles. Import after `theme.css`. |
| `assets/theme.js` | Optional: external pre-paint theme script for `<head>` (see "Theme script and light pinning"). |
| `assets/controls.css` | Inside-the-control focus treatment for library and host fields. |
| `assets/inspector.css`, `inspector.js` | The docked record inspector. |
| `assets/menu.js` | Optional: menu dismissal (outside click, Escape) and remembered sidebar sections. |

With Tailwind 4, import the CSS into the host entrypoint after linking the
module directory (see the README for `@source`):

```css
@import "../path/to/orbitalui-htmx/assets/tokens.css";
@import "../path/to/orbitalui-htmx/assets/theme.css";
```

Or serve `assets.Files` and link the stylesheets directly.

## Theme ownership across applications

The library owns component structure, semantic component selectors and the
default theme. A consuming app owns its product identity: it should override
the primitive and semantic tokens (brand, neutral scale, surface, shape,
density and motion) and compose its own page and outer shell layouts. Prefer
token overrides over redefining `.ui-*` component selectors so the same
component markup behaves consistently in each app while retaining its own
palette and personality. Keep domain-specific page composition in the app.
[docs/adoption](adoption/README.md) spells out the split and has a
migration checklist for each consuming application.

## Completeness contract

`theme.css` styles every class a component emits. `TestThemeStylesEveryEmittedClass`
scans the `.templ` sources and class helpers and fails when a class has no
rule in a shipped stylesheet, so a host never has to style library markup to
make a component look finished. When you add a class to a component, add its
rule here in the same change.

## Tokens

Three layers, all declared in `:where(:root)` so any host rule wins:

1. **Primitive scales** — `--oui-gray-25…950`, `--oui-brand-25…950`,
   `--oui-success-*`, `--oui-warning-*`, `--oui-error-*`, `--oui-info-*`,
   `--oui-ai-*`.
2. **Semantic roles** — components read only these:
   - text: `--oui-text-primary | secondary | tertiary | quaternary | placeholder | brand`
   - surfaces: `--oui-bg-primary | secondary | tertiary | canvas | active | brand | brand-subtle`
   - borders: `--oui-border-primary` (controls) and `--oui-border-secondary` (cards, dividers)
   - icons: `--oui-fg-*`
3. **Shape, elevation, type, motion** — `--oui-radius-*`, `--oui-shadow-xs…xl`,
   `--oui-ring-brand`, `--oui-text-xs…display-sm` with matching
   `--oui-leading-*`, `--oui-duration*`, layout sizes.
4. **Identity hooks** — `--oui-font-display` (every title: page, card,
   dialog, drawer, record header, metric value), `--oui-heading-weight`,
   `--oui-heading-tracking`, `--oui-font-numeric` (money and codes).
5. **Layout** — `--oui-page-measure` / `-narrow` / `-wide`, `--oui-page-gap`,
   `--oui-form-gap-x/y`. Below 42rem, controls (small ones included) grow to
   the 44px touch target.

**Rebranding** means re-pointing `--oui-brand-*`. **Bridging** an existing
application means aliasing its own names to the semantic layer
(`--ink: var(--oui-text-primary)`), so old and new rules share one source.

## v2 foundation: tone families, action tokens, dark mode

- **Tone families.** Each of `brand`, `accent` (second identity colour),
  `info`, `success`, `warning`, `danger`, `neutral` and `ai` has `--oui-{tone}`,
  `-2` (strong or hover), `-soft` (tint), `-ink` (text on the tint) and
  `--oui-on-{tone}` (text on the solid colour). `ai` also has
  `--oui-ai-gradient`, `--oui-ai-gradient-soft` and `--oui-ai-glow`.
  Components read these, never hex.
- **Primary action.** `--oui-action-bg`, `-bg-hover`, `-fg`, `-ring`, `-soft`
  and `-ink` default to the brand tone. Buttons, selected chips, segmented
  and tab selection, meters and checklists read them. An ink-led identity sets
  `--oui-action-bg: var(--oui-ink)` and `--oui-action-fg: var(--oui-ink-on)`
  and keeps `--oui-action-ring` on its focus colour.
- **Surfaces, ink, lines.** `--oui-canvas`, `--oui-surface`, `-2`, `-3`, `-4`,
  `--oui-raised`; `--oui-ink`, `-2`, `-3`, `-4`, `--oui-ink-on`; translucent
  `--oui-line`, `-2`, `-3`. The older `--oui-bg-*`, `--oui-text-*`,
  `--oui-border-*` and `--oui-fg-*` names resolve onto these.
- **Elevation.** `--oui-shadow-1/2/3` are ring plus layered shadow;
  `--oui-shadow-inset` is the highlight. `--oui-shadow-xs..xl` are the v1
  scale kept ring-less (shadow only) for hosts that pair them with borders.
  Focus: `--oui-focus-ring` (canvas gap plus action ring) and
  `--oui-focus-halo` for inputs. Also `--oui-glass` and `--oui-scrim`.
- **Radius** xs 6, sm 8, md 12, lg 16, xl 22 (`2xl` = 22). **Type scale**
  `--oui-text-display-xl|display-l|display-m|title|body-l|body|label|caption|micro`
  with `--oui-leading-*`; `--oui-text-xs..xl` are aliases. **Controls** are
  30 / 36 / 44px (44px for all on phones). **Motion** `--oui-ease-out`,
  `-in-out`, `-spring` and `--oui-dur-1..4` (120/200/320/520ms, all `0ms` under
  `prefers-reduced-motion`); `--oui-duration(-fast)` are aliases.
- **Tables.** `--oui-table-head-bg|color|size|weight|tracking|transform` and
  `--oui-table-pad-x|y` restyle every table; `TableOpts.Flush` and
  `HeaderStyle: TableHeaderCaps` cover the full-bleed register look.

### Dark mode contract

`tokens.css` flips the primitives and base semantics under
`:root[data-theme="dark"]` and, while no `data-theme` is set, under
`@media (prefers-color-scheme: dark)`; `data-theme="light"` pins light. Dark
selectors outrank `:where(:root)`, so **an app that overrides tokens for light
mode must also provide the dark values**, in `:root[data-theme="dark"]` (and
the media-query block if it supports system dark before a theme script runs).
An override only in `:root {}` is replaced by the library's dark value in dark
mode. Derived tokens (`--oui-bg-primary` etc.) follow automatically, so
override base values (`--oui-surface`, `--oui-brand-600`, tone tokens) rather
than every alias. `internal/gallery/presets/kindred.css` is a worked example.

### Runtime and CSP

See the README for the `orbital-ui.js` load order and CSP-safe factories. Host
markup that adds its own Alpine directives must stay within the CSP build's
grammar (property or method references, no inline expressions) when the page is
served under `script-src 'self'`; `x-model` writes cannot run there.

### New components (v2)

`Button` variant `ai`, `ButtonSizeLG`, `IconOnly`, `Block`, `IconRight`;
`Badge(tone, label, BadgeOpts)` (Hollow, Muted, Mono); `Card` Interactive,
Href, Edge, header Icon and Subtitle, `CardSection`; field Icon, Prefix,
Suffix, Suggested and `SuggestedTag`, `FieldSelect` chevron element,
`FieldTextarea` Autogrow; `ChipGroup`; `SegmentedChoice`; `Meter`,
`RingMeter`, `Timeline`/`TimelineItem`, `Kbd`, `Orb`, `EntityChip`; `Avatar`
sizes 16..72 and Anonymous; `ChecklistWith` compact; `BreadcrumbsWith`
(phone collapse); `SidebarNavOpts.Rail`; `WorkspaceSwitcherOpts.Compact`;
`TableOpts.Flush|HeaderStyle|Compact`; `DisclosureOpts.SummaryID`;
`SettingsSectionOpts.InCard`; `MenuLabel` and menu `Kbd` hints. Each is shown
in the gallery.

## App shell and runtime contracts (v2 shell)

Files: `assets/shell.css` (frame, topbar, sidebar extras, palette, toasts),
`assets/theme.js` (pre-paint theme), `assets/orbital-ui.js` (shell runtime).
Components: `shell.templ`, `palette.templ`, `toast.templ`, `themeswitch.templ`,
`chatcomposer.templ`. Nothing here names a product: brand, navigation, palette
content and menus are slots or data supplied by the host.

### AppShell

`AppShell(AppShellOpts, main)` renders, inside `<body>`: a skip link, the quiet
canvas sidebar (`Brand` slot, scrolling `Sidebar`, pinned `Footer`), a scrim, and
the inset rounded sheet with a sticky glass topbar (`Breadcrumbs` or `Crumbs`,
`Actions`) over `main`. `Overlays` renders after the frame (palette, toast
region, drawers). `FullWidth` drops the padded, centred page column and the
sheet's own scroll, for workspaces that own their layout. The sidebar and sheet
carry `.ui-vt-sidebar` / `.ui-vt-main`, the view-transition names from
`motion.css`; they do nothing until the host adds
`@view-transition { navigation: auto; }`.

Below 900px the sidebar is off-canvas: the burger (`data-shell-open`) opens it,
focus moves to its first link, the scrim, the close button, a sidebar link or
Escape close it, and focus returns to the burger. While closed it is `inert`
and `visibility: hidden`, so keyboard and screen readers cannot reach it. Targets
are 44px.

Relationship to `SidebarNav`: both emit the `.ui-nav__*` classes, so a
data-driven `SidebarNav` and a hand-composed `SidebarMenu` > `SidebarSection` >
`SidebarItem` look the same. Use `SidebarItem` when a row needs `Count`,
`Percent`/`Done`, a `Leading` slot (avatar), `Tone` (icon tint), `Dense` or
`Add`; `SidebarNav` keeps its collapsible sections and rail mode. Ledger and
Cosec keep their own shells; adopting `AppShell` is opt-in.

`SidebarUser` is the footer: avatar and name as a `Menu` trigger that opens
upward; the panel is the host's (`MenuHeader`, `MenuRow("Theme", ThemeSwitch)`,
`MenuLink`s, a sign-out form). Dismissal comes from `assets/menu.js`.
`TopbarSearch` opens the palette; `TopbarNewMenu` is a small button with a menu
panel aligned right (label hidden on phones).

### Command palette contract

`CommandPalette(CommandPaletteOpts{ID, Endpoint, Groups, Placeholder, Label,
EmptyText, Icons})` is a modal `<dialog>` (default id `command-palette`)
containing a combobox input (`aria-controls`, `aria-activedescendant`), a
listbox of `role=option` rows and a polite status line. Open it with Ctrl+K /
⌘K (⌘ is shown as Ctrl off Apple platforms), `TopbarSearch`, or any element with
`data-palette-open` (its value is the palette id; empty opens the first). Arrow
keys move the active row (wrapping), Home/End jump, Enter activates it, Escape
or a backdrop click closes it, and focus returns to the opener.

Results are JSON, not an HTML fragment: rows are built with `textContent` (no
injection surface), no inline script is needed, and static and remote results
share one shape. The runtime sends `GET <Endpoint>?q=<query>` with
`Accept: application/json` and same-origin credentials, once on open (empty
`q`) and then debounced (150ms) as the query changes; a newer query aborts the
older request. The response:

```json
{"groups": [
  {"label": "Documents", "items": [
    {"label": "Report archive", "href": "/docs/7", "icon": "document", "hint": "Document",
     "tone": "ai", "keywords": "annual", "avatar": "Acme Inc", "square": true,
     "form": {"action": "/assistant/new", "method": "post", "fields": {"prompt": "…"}}}
  ]}
]}
```

`href` is required; `icon` names an icon from the set (the dialog ships the
common ones in a `<template>`; list extras in `Icons`); `avatar` draws initials
of that name instead of an icon; `form` makes Enter submit a hidden form (a link
cannot POST; include any CSRF token in `fields`). Static `Groups` use the same
keys, are filtered client-side (prefix, word, substring, subsequence) and are
shown first; endpoint groups follow unfiltered, so the server decides what a
query matches. A non-2xx or non-JSON response leaves only the static results.
Enter follows the row's real link, so `hx-boost` and view transitions apply.
An `href` or `form.action` that does not resolve to http(s) (`javascript:`,
`data:`) is dropped: the row renders with `href="#"` and no form is built, so
results that echo user-entered data cannot run script.

### Toast contract

`ToastRegion()` once per page (children are server-rendered `Toast`s, which
show on first paint; set `Auto` to have the runtime dismiss them). From a
response:

```
HX-Trigger: {"toast": {"tone": "success", "title": "Saved", "message": "Invoice 42 posted.", "auto": true}}
```

`tone` is `neutral | info | success | warning | danger | ai` (`positive` and
`spark` are accepted aliases); `title` is optional; a bare string
(`{"toast": "Saved"}`) is the message. `auto` omitted means auto-dismiss for
every tone except `warning` and `danger`, which stay until dismissed. The same
payload works from script: `window.OrbitalShell.toast({...})`. Auto-dismiss
waits ~4s (`data-lifetime="ms"` on `[data-toast-stack]` overrides) and pauses
while the pointer is over the toast or focus is inside it; the close button, and
Escape on a focused toast, dismiss at once. The stack is `aria-live="polite"`; a
danger toast is `role="alert"`. Reduced motion replaces the movement with a
short fade.

### Theme script and light pinning

`assets/theme.js` is an external blocking script for `<head>`, before the
stylesheets:

```html
<script src="/static/theme.js"></script>
<script src="/static/theme.js" data-storage-key="myapp-theme" data-default="system"></script>
```

It reads the preference (`light`, `dark` or `system`) from `localStorage` under
the key (script `data-storage-key`, else `<html data-oui-theme-key>`, else
`oui-theme`; first-visit default from `data-default` /
`<html data-oui-theme-default>`, else `system`), sets `<html data-theme>` and
`color-scheme` before first paint, and follows `prefers-color-scheme` and other
tabs while the preference is `system`. It exposes `window.OrbitalTheme`
(`getPreference`, `getTheme`, `setPreference`, `storageKey`) and fires
`oui:theme` on `document`. `ThemeSwitch(ThemeSwitchOpts{StorageKey})` is a
`SegmentedChoice` (sun, monitor, moon) the runtime keeps in sync; give it the
same `StorageKey` as the script. If `theme.js` is not loaded the switch still
stores the choice and applies it after first paint.

An app that is light-only (or dark-only) does not load `theme.js`, does not
render `ThemeSwitch`, and pins the theme in markup: `<html data-theme="light">`.
`tokens.css` follows the OS only while `data-theme` is unset, so a pinned page
never flips.

### Chat composer

`ChatComposer(ChatComposerOpts)` is the message box (autogrowing textarea,
`Tools` slot, round send button, AI glow on focus, `Busy` state with the orb);
`TypingIndicator(label)` and `StreamingCaret()` are the working cues. The
host owns the endpoint, Enter-to-send behaviour and conversation state.

## Principles

- **Progressive disclosure.** Show identity and the next decision; put
  detail one click away (menus, collapsible nav sections, disclosures,
  inspector tabs). Never print raw field dumps.
- **One primary action per view.** Primary buttons are brand-filled;
  everything else is secondary or quiet.
- **Color carries state, never decoration.** Every tinted badge has a label;
  brand color marks action, selection and focus.
- **Hierarchy through weight and tone, not size.** Titles 24px/600, card
  titles 18px/600, body 14px, metadata 12–14px in tertiary/quaternary text.
- **Cards frame working areas; settings use two-column rows.** Labels and
  explanations on the left, controls on the right.

## Components added for the workspace shell

| Component | Use |
|---|---|
| `PageHeader(title, PageHeaderOpts)` | Eyebrow, title, description, meta row, actions, optional tabs. |
| `Card(CardOpts, body)` | Framed working area with header, actions, footer; `Flush` for tables; `Tone` brand/danger/subtle. |
| `SettingsSection(SettingsSectionOpts, body)` | Two-column settings row. |
| `DescriptionList(items, opts)` | Record facts; inline, stacked or grid; visible placeholder for missing values. |
| `Avatar(name, AvatarOpts)` / `Initials(name)` | Initials avatar with a stable derived hue; square for organisations. |
| `ProgressBar(value, ProgressOpts)` | Accessible progress with label and value. The bar is always named: `Label`, else `AriaLabel`, else "Progress". |
| `Checklist(items)` | Onboarding steps; only open steps are actionable. |
| `Steps(steps)` | Vertical timeline with done/current/upcoming/blocked states. |
| `Feed(items)` / `FeedItem(opts)` | Notifications and activity rows with unread state. |
| `SidebarNav(sections, opts)` | Sidebar navigation with optional collapsible sections and badges. |
| `Breadcrumbs(crumbs)` | Compact trail; only the last crumb is the current page (`aria-current`), earlier crumbs without an `Href` render as plain text. |
| `Menu`, `MenuLink`, `MenuHeader`, `MenuSeparator` | Native `<details>` popover menus. |
| `WorkspaceSwitcher(opts, panel)` | One-line scope identity that discloses details and switching. |
| `CountBadge(n)` | Neutral count pill for titles. |
| `RecordHeader(RecordHeaderOpts)` | Identity block an inspector opens with: mark, eyebrow, title, subtitle, status. No heading element. |
| `PageHeaderOpts.Leading` | Avatar or logo beside a record page's title. |
| `ActionList` / `ActionRow` | The workspace list shape: leading icon or avatar, title + one meta line, trailing value/badge/controls; linked rows get a chevron. Variants: divided, plain, cards. |
| `FactList([]Fact)` | One-line term/value facts too small for a DescriptionList. |
| `CriteriaList([]Criterion)` | Pass/fail evidence with sr-only "Met:/Not met:" prefixes. |
| `ChoiceRow` / `ChoiceGroup` | Checkbox or radio framed as a row with a one-line description; tinted when checked. |
| `Callout(tone, CalloutOpts, body)` | A titled, toned frame that can hold rows — the queue to clear first. |
| `DecisionCard(DecisionCardOpts, body)` | One review decision: subject, context, proposal, evidence disclosure, explicit actions. |
| `FilterChips([]ChipLink, active, opts)` | GET-link chips that narrow a list; shareable and script-free. |
| `SectionNav([]SectionLink, opts)` | Sticky in-page index; a scrolling strip on phones. |
| `Disclosure` variants | `DisclosurePanel` (editor surface), `DisclosureQuiet` (text-link evidence), `DisclosureDanger` (danger zone); `Hint` for a trailing count or consequence. |
| `StatePanelOpts.Compact` | In-card empty state without a competing heading. |
| `ButtonVariantText` | A link-styled submit for per-row mutations. |
| `PageFrame(PageFrameOpts)` | The page's content column: `PageWidthNarrow` (forms, sign-in), standard, `PageWidthWide` (dense registers), `PageWidthFull`; shared vertical rhythm. |
| `FormGrid(FormGridOpts)` | 1–3 column grid for `LabelledField`s; `FieldOpts.Wide` spans the row; one column on a phone. |
| `FormActions(FormActionsOpts)` | Submit/cancel row; `End` aligns right, `Divided` closes a long form; full-width actions on a phone. |
| Formatters (`format.go`) | `DateText`, `DateTimeText`, `RelativeDate(Text)`, `DayDuration`, `Count`, `Plural`, `Quantity`, `GroupedAmount`, `Percent` — total functions over stored strings (including PostgreSQL timestamptz text). |

Shell layout classes (`.ui-icon-button`, `.ui-nav*`, `.ui-switcher*`,
`.ui-breadcrumbs`) are styled in `theme.css`; the host owns the outer grid,
responsive drawer behaviour and routes.

## Slot and attribute conventions

- **Nil slots render nothing.** Any `templ.Component` slot or body argument
  (`Alert`, `Callout`, `DecisionCard`, `ConfirmButton`, `Card`,
  `SettingsSection`, `LabelledField`, `ActionList`, `Feed`, `TableShell` rows
  and footer, `Menu` trigger and panel, `WorkspaceSwitcher` panel,
  `Disclosure`, `ChoiceGroup`, `Inspector`, `AsyncRegion`, `SelectionToolbar`,
  `Drawer`, `ModalShell`, `HTMXPostForm`, `ChatThread`, `SummaryCard`, and
  `AssistantMessageContent`) may be nil; the wrapper still renders and the
  slot is empty.
- **`Attrs` never produces a second `class`.** Components that take
  `Attrs templ.Attributes` fold an `Attrs["class"]` value into their own class
  list (after the computed classes) and spread the remaining attributes.
  `Class` and `Attrs["class"]` can be used together.

## Runtime and CSP requirements

| Component set | Runtime requirement | CSP/integration note |
|---|---|---|
| Server-rendered primitives, fields, rows, navigation, tables and state surfaces | None beyond Go/templ; HTMX attributes only when used | Native HTML remains the fallback for links and forms. |
| `Inspector` | Serve and load the external `assets/inspector.js` and `assets/inspector.css` | Does not require Alpine or inline scripts; host supplies CSP policy and asset URLs. |
| `ModalShell`, `ConfirmButton`, `Drawer`, and `EntityPicker` | Alpine plus `assets/orbital-ui.js`, loaded before Alpine | No inline scripts. The runtime installs `window.OrbitalUI` and the `window.LedgerUI` names the markup calls, and stays inert when a host already provides `window.LedgerUI`. Server-driven close: `HX-Trigger: closeDialog` / `drawerClose`; 422 responses swap inline errors. |
| `AppShell` sidebar, `CommandPalette`, `ToastRegion` / `HX-Trigger` toasts, `ThemeSwitch` | `assets/orbital-ui.js` (any position relative to Alpine); `assets/theme.js` in `<head>` for the pre-paint theme | Plain DOM delegation: no Alpine, no eval, no inline script or style, so it runs under `script-src 'self'` and beside a host `window.LedgerUI`. The palette fetches its endpoint with same-origin `fetch`. |
| `BalanceTool` | None for server-rendered totals; Alpine plus the host lines editor for live recalculation | Server totals remain authoritative; the host owns live editor integration. |
| Dismissible `Alert`, `SuggestionChip` and `EntityPicker` states | Alpine.js; `theme.css` hides `[x-cloak]` until Alpine starts | Expressions are emitted in markup; CSP compatibility depends on the host's Alpine build and policy. |

`scripts/browser-check.mjs` drives the interactive set in headless Chrome
with real pointer and keyboard input (modal focus, 422 swaps, server-driven
close and focus return, Escape and backdrop dismissal, drawer lock and
full-screen, picker search, keyboard pick and create, Alpine after htmx
swaps), then the shell (Ctrl+K palette with endpoint results, arrows, Enter
and Escape with focus return; `HX-Trigger` toasts, auto-dismiss, hover pause and
close; theme switch persistence, system following and no-flash reload; the phone
burger and scrim) on `/shell`, in all three passes. CI runs it on every push.

The library does not set an application's CSP headers or serve assets
automatically. Consumers should verify component interaction under their
actual CSP and after HTMX swaps; successful templ rendering alone does not
verify browser runtime behavior.

For chat suggestions, prefer `SuggestionSubmitChip` inside a native form when
the action should work without JavaScript. `SuggestionChip` requires Alpine
to evaluate its action expression. `AssistantMessageContent` accepts
application-rendered content such as cited links; the host owns its validation
and escaping contract (plain text should use `AssistantMessage`).

## Host table sorting and composition

`TableShell` supports both library `SortState` GET links and host-owned
`Column.SortHref` links; direct links take precedence and preserve the host
query convention. Use `SortDirection` (`ascending`, `descending`, or `other`)
for direct-link accessibility, or explicit `ClientSort` keys for a host
external sorting runtime. No new sorting behavior is enabled by default.
`TableClass` and `SortClass` retain host layout and runtime hooks without
duplicating the frame; optional `EmptyContent` supplies a host empty state.

Sidebar hosts can scope navigation density and state skin with `--oui-nav-*`
and switcher surfaces with `--oui-switcher-*` properties. These are optional
component-local overrides with semantic fallbacks in `theme.css`, so hosts
need no selector copies. Only set the properties needed by the app; component
structure, keyboard focus, and default states remain shared.
