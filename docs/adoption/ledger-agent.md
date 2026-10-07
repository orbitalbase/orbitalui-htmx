# ledger-agent adoption checklist

Audit of 2026-10-08 against `27a4835` (the vendored snapshot). Ledger imports
the theme correctly (`app.css` loads `tokens.css`, then `theme.css`,
`controls.css` and `inspector.css` into `layer(components)`) and re-skins it
mostly through tokens (107 of 143). Most of its 12,200-line `app.css` is page
CSS, plus copies of component styles that the shared theme now ships.
`go test ./internal/adapters/ui/...` passes against the polished library
(verified with a workspace override); Ledger's preview pages changed only in
the pager and the date-picker icon.

## 1. Refresh the snapshot

`scripts/sync-orbitalui.sh <checkout> <commit>` once this library change is
tagged.

## 2. Bugs to fix in app.css (not shared)

- **Display font never loads.** `app.css:84` sets
  `--font-display: "Schibsted Grotesk Variable"`, but Google Fonts serves the
  family as `"Schibsted Grotesk"`. Every heading falls back to the system
  font. Fix the name and map it into the theme:
  `--oui-font-display: var(--font-display);`
- **Sidebar group items are 16px; links are 14px.** `.nav-item--parent`
  (~`app.css:728`) sets `font: inherit`, which resets the size from
  `.nav-item`. Put `font-size: 0.875rem` after it, or move the sidebar to
  `SidebarNav`.

## 3. Delete copies of component styles the theme now owns

Ledger's `app.css` styles these shared component families itself. The theme
now ships token-driven versions, so remove the Ledger copies and keep only
tone adjustments as tokens:

- Drawer (`.drawer-*`, about 116 references; keep the page-specific
  `#tpl-drawer` side-panel variant)
- Entity picker (`.entity-picker*`, from ~`app.css:2541`)
- Balance tool (`.beam*`, from ~`app.css:2324`)
- Pager (`.pager*`, from ~`app.css:1853`)
- `.amount`, `.mono`, `.confirm-busy`/`.confirm-label`,
  `.record-state-summary`, `.async-region*`, `.dependency-panel*`,
  `.ui-alert--dismissible`, `.field-label`, `.field-help`, `.field-error`,
  `.field--wide`

The "Ledger skin for shared components" block (`app.css:90`) is the right
pattern; keep identity there as token values.

## 4. Use components instead of raw markup

- **Raw `<details>`** appears in 33 templates (for example
  `pages/dashboard/dashboard.templ:43,66`, `tax-setup__section`). These show
  the browser's ▶ marker. Use `components.Disclosure` (default, `DisclosurePanel`
  or `DisclosureQuiet`). Tree expanders in `accounts/hierarchy.templ` are a
  legitimate custom widget.
- **Bare inputs** inside `.ui-field`, `.tax-setup__form` and `.workspace`
  forms are styled ad hoc. Use `LabelledField` + `FieldInput`, or the
  `field-control` class, inside `components.FormGrid`.
- **Invoice and payment line editors**: long labels such as "Tax code
  (optional; book OUTPUT codes, e.g. GST-S)" wrap to three lines and push the
  inputs out of line. Keep labels short and move the explanation to
  `FieldOpts.Help`.
- **Dimensions setup**: "Back to starters" floats mid-page inside a
  disclosure body. Put it in a `FormActions` row or in the page header actions.
- Page frames: `PageFrame` (wide/full) replaces the per-page width rules for
  workspace, drill and report pages.

## 5. Runtime

`ModalShell`, `ConfirmButton`, `Drawer` and `EntityPicker` call the
`window.LedgerUI` helpers in Ledger's `app.js`. Keep that contract until the
library ships its own runtime.
