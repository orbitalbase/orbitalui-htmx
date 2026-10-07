# Orbital UI design system

Orbital UI ships a complete default theme for every class its components
emit, so a consuming application gets a polished, consistent workspace
without restyling the library. The look is a calm, light, record-first
workspace: cool grays, one indigo brand accent, crisp 1px borders, soft
paper-like elevation and dense-but-readable 14px controls.

## Files

| File | Purpose |
|---|---|
| `assets/tokens.css` | Primitive scales and semantic roles. Import first. |
| `assets/theme.css` | Default styles for every `ui-*`, `field-*`, `tab-*`, `kpi-*`, `pager__*` and table class. |
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
| `ModalShell`, `ConfirmButton`, `Drawer`, and `EntityPicker` | Alpine plus the source-compatible `window.LedgerUI` helpers referenced by component markup | Not standalone; host must supply matching helper/runtime versions. |
| `BalanceTool` | None for server-rendered totals; Alpine plus the host lines editor for live recalculation | Server totals remain authoritative; the host owns live editor integration. |
| Dismissible `Alert`, `SuggestionChip` and `EntityPicker` states | Alpine.js; `theme.css` hides `[x-cloak]` until Alpine starts | Expressions are emitted in markup; CSP compatibility depends on the host's Alpine build and policy. |

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
