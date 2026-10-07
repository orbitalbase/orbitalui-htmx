# Adopting the shared theme

Orbital UI ships a finished look for every class its components emit
(`assets/tokens.css` + `assets/theme.css`; `TestThemeStylesEveryEmittedClass`
keeps it that way). An application looks better when it *stops* overriding
that theme, not when it adds more rules on top of it.

## What belongs where

| Belongs in **orbitalui-htmx** | Belongs in **the application** |
|---|---|
| How a component looks and behaves: buttons, fields, tables, pager, badges, cards, dialogs, drawer, picker, disclosures, empty and loading states | Brand identity, set **only through tokens**: `--oui-brand-*`, `--oui-gray-*`, `--oui-font-sans/-display/-mono`, `--oui-heading-weight/-tracking`, radius, density |
| Layout primitives every page shares: `PageFrame`, `FormGrid`, `FormActions`, `Card`, `SettingsSection`, `ActionList` | Page composition: which content leads a page, grouping, copy, empty-state wording, which width a page uses |
| Responsive and touch-target behaviour of components | The outer shell (sidebar, topbar), routes, domain-specific widgets |

Rule of thumb: if both apps would want the change, make it here. If only one
app wants it, it is identity (a token) or composition (a page).

## Rules for application CSS

1. **Override tokens, not selectors.** Re-point `--oui-*` in one `:root`
   block. Do not write `.ui-card { … }`, `.ui-page-header__title { … }` and
   similar. If a token is missing for what you need, add the token here.
2. **No component restyles inside page scopes.** `.records-settings-card
   .ui-settings-section { padding: … }` is a sign the shared component is
   missing an option; add the option here instead.
3. **Use the component, not a copy of its markup.** Raw `<details>`,
   hand-built tables, pagers, dialogs and form grids drift from the system.
4. **Load order.** Import `tokens.css`, `controls.css`, `theme.css` first,
   then the app's token overrides, then page CSS. Ledger's
   `@import … layer(components)` pattern is a good model: shared rules sit in
   a layer, so an app rule always wins without specificity fights.

## Developing against a local checkout

Both apps vendor a snapshot under `third_party/orbitalui-htmx`. While
iterating on shared and app design together, point the app at your checkout
without editing its repository:

```sh
cat > /tmp/orbitalui.work <<EOF
go 1.26.2
use /path/to/app
replace github.com/orbitalbase/orbitalui-htmx => /path/to/orbitalui-htmx
EOF
GOWORK=/tmp/orbitalui.work go test ./...          # app tests against the checkout
GOWORK=/tmp/orbitalui.work scripts/ui-css.sh out.css   # Ledger CSS build
```

Cosec's `scripts/tailwind.sh` forces `GOWORK=off`; build from a copy of
`internal/ui/assets/tailwind.css` whose `tmp/orbitalui-htmx` imports point at
the checkout. When the shared change is final, tag it and refresh the snapshot
with each app's `scripts/sync-orbitalui.sh`.

Per-application checklists: [cosec-agent](cosec-agent.md),
[ledger-agent](ledger-agent.md).
