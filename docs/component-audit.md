# Component library review

## Repository history

`git log --all` contains one commit, `c7afe51` (`feat: add reusable templ
HTMX component library`), tagged `v0.1.0`. The tag and `main` point to that
same commit, so there are no earlier revisions to compare.

## Source comparison

The initial module exposed form controls, chat/suggestion surfaces, summary
cards, and a small SVG icon set. `ledger-agent/internal/ui/components` has a
larger reusable set. The generic controls and primitives from that package
were added here: buttons, status and alert surfaces, field helpers, search,
URL-backed tabs and segments, tables and paging, dialogs, confirmations,
drawers, entity picker, KPI tiles, exact decimal helpers, and async/state
surfaces. Application routes, pages, and domain records were not copied.

`agentswarmv2` does not currently import this module. Its `go.mod` has no
templ, HTMX, Alpine, Tailwind, or `orbitalui-htmx` dependency, and the repo
contains no templ UI sources. The current tree therefore does not substantiate
the assumption that it is a consumer.

## Pattern review

- **HTMX:** the URL-backed search/tab/pager patterns in the extracted set use
  ordinary links or forms and progressively add partial requests. The form
  wrapper now keeps native `method="post"` and `action` behavior and sanitizes
  the `hx-post` URL. Applications still need to provide CSRF protection,
  endpoint behavior, response status handling, and correct target/response
  pairing. Full-page response plus `hx-select` and fragment response patterns
  must not be mixed on a page.
- **Alpine.js:** state is scoped to components and ordinary controls use
  `x-model`/`x-text`. Expression parameters such as model names and click
  actions are code configuration, not user data. Dialog, drawer, and entity
  picker markup currently calls the source app's `window.LedgerUI` helpers;
  this is an integration dependency the component package does not yet ship.
- **templ and Go:** generated files are refreshed from `.templ`; user-facing
  text is templ-escaped, URL attributes are sanitized, and exact amounts use
  `big.Rat` instead of floats. HTML ids, labels, and HTMX selectors remain the
  caller's responsibility. New public APIs should prefer typed options
  structs over adding more positional arguments.
- **Tailwind:** utility classes are statically present in template sources,
  so Tailwind can detect them when those files are registered as sources.
  Tailwind 4 ignores many dependency directories by default, so consumer
  builds must explicitly include this module. The extracted semantic classes
  still require a theme stylesheet; `styles/components.css` only styles the
  original `oui-*` subset.

## Follow-up gaps

1. Move `window.LedgerUI` behaviors into a library-owned, documented runtime
   (or make those components runtime-agnostic) before advertising dialogs,
   drawers, and the picker as standalone.
2. Ship or generate styles for the extracted semantic class set and provide a
   Tailwind integration example for a real consuming app.
3. Expand input APIs where needed: `TextAreaControl` has no label/error
   wrapper, and `NumberInputControl` hardcodes `min="0"`, which is not valid
   for every numeric field.
4. Add consumer-level rendering/browser coverage for native form fallback,
   HTMX swaps, Alpine initialization after swaps, and keyboard interaction.

## Verification

`go tool templ generate`, `go test ./...`, and `go vet ./...` pass. The
rendering and money/pager/icon tests were ported alongside the extracted
components. This verifies Go/template compilation and server-rendered markup;
it does not verify a browser integration or a consuming Tailwind build.
