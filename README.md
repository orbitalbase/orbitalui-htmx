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
- KPI tiles, exact decimal amounts, money cells, and a debit/credit balance
  tool.
- Generic asynchronous regions, dependency panels, selection toolbars, and
  text/number formatting helpers.

Components that accept `templ.Component` children let the application keep
ownership of page content and endpoints. Components do not register routes or
make application data assumptions.

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
The starter fields also have optional base styles in
`styles/components.css`. The extracted core components use `ui-*`, `field-*`,
`table-*`, and related semantic class names; import a matching theme stylesheet
or define those classes in the host design system.

`HTMXPostForm` renders a normal POST form (`method` and `action`) and upgrades
it with `hx-post`, so browser submission still works when HTMX is unavailable.
The component leaves target, swap strategy, indicator, authorization, CSRF,
validation, and response rendering to the application.

The dialog, drawer, and entity-picker components currently use the
`window.LedgerUI` Alpine helper contract from the source application. The host
must provide the matching runtime before using those components. Other
components use native HTML or HTMX attributes and need no library-specific
JavaScript.

## Development

Generated `*_templ.go` files are committed. After changing a `.templ` file:

```sh
go tool templ generate
go test ./...
```
