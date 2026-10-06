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
| `ProgressBar(value, ProgressOpts)` | Accessible progress with label and value. |
| `Checklist(items)` | Onboarding steps; only open steps are actionable. |
| `Steps(steps)` | Vertical timeline with done/current/upcoming/blocked states. |
| `Feed(items)` / `FeedItem(opts)` | Notifications and activity rows with unread state. |
| `SidebarNav(sections, opts)` | Sidebar navigation with optional collapsible sections and badges. |
| `Breadcrumbs(crumbs)` | Compact trail; the last crumb is current. |
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
| Formatters (`format.go`) | `DateText`, `DateTimeText`, `RelativeDate(Text)`, `DayDuration`, `Count`, `Plural`, `Quantity`, `GroupedAmount`, `Percent` — total functions over stored strings (including PostgreSQL timestamptz text). |

Shell layout classes (`.ui-icon-button`, `.ui-nav*`, `.ui-switcher*`,
`.ui-breadcrumbs`) are styled in `theme.css`; the host owns the outer grid,
responsive drawer behaviour and routes.
