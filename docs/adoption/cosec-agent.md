# cosec-agent adoption checklist

Audit of 2026-10-08 against `27a4835` (the vendored snapshot). Cosec uses the
shared library heavily (`orbital.Card` ×242, `StatusBadge` ×82,
`DescriptionList` ×81, `StatePanel` ×76, `PageHeader` ×63), but about 7,000
lines of app CSS in 22 files restyle 68 of the theme's `.ui-*` selectors
directly. Rendering Cosec's preview pages against the polished theme changed
**0 pixels**, so none of the shared improvements reach Cosec until those
overrides are removed.

## 1. Refresh the snapshot

`scripts/sync-orbitalui.sh <checkout> <commit>` once this library change is
tagged. `go test ./internal/ui/...` passes against it (verified with a
workspace override).

## 2. Delete overrides the theme now owns

| Cosec rule | Now shared |
|---|---|
| `shell.css:144` `.app-main .ui-page-header__title { font-weight; letter-spacing }` | Set `--oui-heading-weight: 650; --oui-heading-tracking: -0.035em` in `design/tokens.css` |
| `shell.css:155` button font size/weight under `.app-main` | Already the theme default (`.ui-button` is 14px/600) |
| Phone `min-height: 2.75rem` rules in `patterns.css`, `route-refresh.css`, `shell.css`, `company-settings.css` | `--oui-control-height` and `--oui-control-height-sm` are 2.75rem at ≤42rem |
| `firm.css` `.ui-card__body > .disclosure:first-child …` | Shared |
| `firm.css` / `platform.css` phone `.ui-row` wrapping | Shared (`.ui-row` wraps at ≤42rem) |
| `firm-catalogue.css` phone `.ui-decision__actions .ui-button { width: 100% }` | Shared |
| `company-records.css` `.records-person .ui-status { flex: 0 0 auto }` | `.ui-status` never shrinks |
| `chat.css` colour rules for messages, avatar and chips | Starter and chat components are themed through tokens |
| `controls.css` relied on `--accent-dark` | Focus now reads `--oui-fg-brand`; set `--control-focus-border` only if focus must differ from the brand |

## 3. Replace local components that duplicate shared ones

`internal/ui/components` wraps shared `Button`, `Alert`, `Status` (fine), but
re-implements:

- `table.templ` → `orbital.TableShell` + `MoneyCell`/`MonoCell`
- `pager.templ` → `orbital.Pager`
- `dialog.templ`, `register_dialog.templ` → `orbital.ModalShell`; load
  `assets/orbital-ui.js` before Alpine (Cosec has no `window.LedgerUI`, so the
  shared runtime installs it). The same script enables `Drawer`,
  `ConfirmButton` and `EntityPicker`.
- Hand-written `<select>` and `<textarea>` markup → `FieldSelect` /
  `FieldTextarea` inside `LabelledField`
- `form.templ` `FormGrid`/`FormActions` (`.form-grid` in `patterns.css:16`)
  → `orbital.FormGrid` / `orbital.FormActions` with `LabelledField`

## 4. Consistency fixes found in screenshots

- **Status badges render three ways.** `workspace-badge`
  (`company-workspace.css:13`, used in `company_offers.templ` and
  `workspace_components.templ`) adds a different chip on the Companies page.
  Use `StatusBadge` defaults everywhere and map tones consistently: "Active"
  is success on every page (Officers currently shows it as info).
- **Page width changes per page.** Wrap each page in `orbital.PageFrame`:
  narrow for sign-in and settings, standard for dashboards, full for
  registers.
- **Search looks different per page** (icon field vs. field + "Search"
  button). Use `orbital.SearchField` everywhere.
- Breadcrumb renders "Companies › Companies" on `/companies`.

## 5. Identity, after the cleanup

Cosec's look today is the shared default plus a violet brand. To give it a
distinct identity, change tokens only: brand scale, `--oui-font-display`,
radius scale, canvas and surface colours. Then improve page composition.
