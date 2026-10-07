// Render smoke tests: every component renders end-to-end through templ with
// key markup/copy present. These are unit-ish guards that the signatures and
// generated code stay wired — they are NOT visual tests (pages prove visual
// composition in their own phases).
package components

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderC(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func want(t *testing.T, out, needle string) {
	t.Helper()
	if !strings.Contains(out, needle) {
		t.Errorf("rendered output missing %q\n---\n%s", needle, out)
	}
}

func TestButtonRenders(t *testing.T) {
	out := renderC(
		t, Button(
			ButtonVariantPrimary, "", ButtonOpts{
				Icon: "plus", Label: "New entry", Href: templ.SafeURL("/journal/new"),
			},
		),
	)
	want(t, out, `<a`)
	want(t, out, `class="ui-button ui-button--primary"`)
	want(t, out, `href="/journal/new"`)
	want(t, out, `>New entry</span>`)

	out = renderC(
		t, Button(
			ButtonVariantDanger, ButtonSizeSM, ButtonOpts{
				Label: "Void entry", Type: "submit",
			},
		),
	)
	want(t, out, `<button type="submit"`)
	want(t, out, `class="ui-button ui-button--danger ui-button--sm"`)

	// Loading disables + spins.
	out = renderC(t, Button(ButtonVariantPrimary, "", ButtonOpts{Label: "Save", Loading: true}))
	want(t, out, `disabled aria-busy`)
	want(t, out, `class="ui-button__progress"`)
}

func TestInteractionContractsRenderTruthfulAccessibleStates(t *testing.T) {
	out := renderC(t, RecordStateSummary(RecordStateSummaryOpts{Posting: "Posted", Review: "Not reviewed", Provenance: "Imported"}))
	want(t, out, `aria-label="Record state"`)
	want(t, out, "Posted")
	want(t, out, "Not reviewed")
	want(t, out, "Imported")

	out = renderC(t, AsyncRegion(AsyncRegionOpts{ID: "bank-list", Label: "Bank transactions", State: "stale", Message: "Refresh failed. Showing the last successful data.", RetryHref: "/bank"}, Text("MYR 24.00")))
	want(t, out, `aria-busy="false"`)
	want(t, out, "Refresh failed")
	want(t, out, "MYR 24.00")
	want(t, out, `href="/bank"`)

	out = renderC(t, AsyncRegion(AsyncRegionOpts{ID: "bad", Label: "Bad retry", State: "error", Message: "Unavailable", RetryHref: "//evil.example"}, nil))
	if strings.Contains(out, "evil.example") {
		t.Fatal("external RetryHref passed the internal-route guard")
	}

	out = renderC(t, DependencyPanel("missing-tax", "Complete tax setup", []Dependency{{Label: "Registration", Detail: "Add the applicable registration", Href: "/settings/tax/setup"}, {Label: "Bad link", Href: "javascript:alert(1)"}}))
	want(t, out, `id="missing-tax-title"`)
	want(t, out, `href="/settings/tax/setup"`)
	if strings.Contains(out, "javascript:") {
		t.Fatal("external dependency URL passed the internal-route guard")
	}

	out = renderC(t, SelectionToolbar(SelectionToolbarOpts{Selected: 3, Eligible: 2, Excluded: 1, ExcludedReason: "One item is already booked", AllMatching: true}, Text("Review")))
	want(t, out, "3 on this page")
	want(t, out, "2 eligible")
	want(t, out, "One item is already booked")

	out = renderC(t, SelectionToolbar(SelectionToolbarOpts{Selected: 3, Eligible: 2, AllMatching: true, SnapshotID: "frozen-17"}, Text("Review")))
	want(t, out, "All matching records")
}

func TestInteractionLinksStayOnTheCurrentOrigin(t *testing.T) {
	for _, raw := range []string{"//evil.example", "/\\evil.example", "/%5cevil.example", "javascript:alert(1)", "/\nHost: evil.example"} {
		if got := internalInteractionHref(raw); got != "" {
			t.Errorf("unsafe link %q accepted as %q", raw, got)
		}
	}
	if got := internalInteractionHref("/bank/12?d=txn:19"); got != "/bank/12?d=txn:19" {
		t.Fatalf("valid internal link changed to %q", got)
	}
}

func TestButtonQuietAndAttrs(t *testing.T) {
	out := renderC(
		t, Button(
			ButtonVariantQuiet, "", ButtonOpts{
				Label: "Make current",
				Attrs: templ.Attributes{"data-open-dialog": "dlg-x", "data-test": ""},
			},
		),
	)
	want(t, out, `class="ui-button ui-button--quiet"`)
	want(t, out, `data-open-dialog="dlg-x"`)
}

func TestStatusBadgeRenders(t *testing.T) {
	out := renderC(t, StatusBadge("Posted", ToneSuccess, StatusBadgeOpts{}))
	want(t, out, `class="ui-status ui-status--success"`)
	want(t, out, `class="ui-status__mark"`)
	want(t, out, `Posted`)

	// Subtle = neutral chip without the tone modifier; no dot when asked.
	out = renderC(t, StatusBadge("Manual", ToneInfo, StatusBadgeOpts{Subtle: true, Dot: boolp(false)}))
	want(t, out, `class="ui-status"`)
	if strings.Contains(out, "ui-status__mark") {
		t.Error("dot should be hidden when Dot=false")
	}
	if strings.Contains(out, "ui-status--info") {
		t.Error("subtle badge must not carry the tinted tone class")
	}
}

func boolp(b bool) *bool { return &b }

func TestAlertRenders(t *testing.T) {
	out := renderC(
		t, Alert(
			ToneDanger, "The entry wasn't posted", AlertOpts{},
			Text("Re-open the period first."),
		),
	)
	want(t, out, `class="ui-alert ui-alert--danger"`)
	want(t, out, `role="alert"`)
	want(t, out, `The entry`)
	want(t, out, `Re-open the period first.`)

	// Info default role is status; dismissible adds Alpine + close.
	out = renderC(t, Alert(ToneInfo, "Note", AlertOpts{Dismissible: true}, nil))
	want(t, out, `role="status"`)
	want(t, out, `x-data="ouiDismiss"`)
	want(t, out, `class="ui-alert ui-alert--info ui-alert--dismissible"`)
	want(t, out, `aria-label="Dismiss"`)
}

func TestStatePanelRenders(t *testing.T) {
	out := renderC(t, StatePanel("No journal entries yet", StatePanelOpts{Icon: "journal"}, nil))
	want(t, out, `class="ui-state"`)
	want(t, out, `>No journal entries yet</h2>`)
	want(t, out, `ui-state__icon`)
	out = renderC(t, StatePanel("Loading", StatePanelOpts{Loading: true}, nil))
	want(t, out, `class="ui-state__skeleton"`)
	want(t, out, `aria-busy="true"`)
}

func TestDisclosureRenders(t *testing.T) {
	out := renderC(
		t, Disclosure(
			"Audit trail", DisclosureOpts{Summary: "created 3 Sep 2026"},
			Text("Entry #42"),
		),
	)
	want(t, out, `<details class="disclosure"`)
	want(t, out, `Audit trail`)
	want(t, out, `class="disclosure__hint">created 3 Sep 2026`)
	want(t, out, `Entry #42`)

	out = renderC(t, Disclosure("Details", DisclosureOpts{Open: true}, Text("open body")))
	want(t, out, `class="disclosure" open>`)
	if strings.Contains(out, "disclosure__hint") {
		t.Error("hint must hide when open")
	}
}

func TestFieldRenders(t *testing.T) {
	in := FieldInput(
		FieldInputOpts{
			Type: "date", Name: "entry_date", ID: "j-date",
			Value: "2026-09-03", Invalid: true, AriaDescribedBy: FieldErrorID("j-date"),
		},
	)
	out := renderC(
		t, LabelledField(
			"Entry date", "j-date", FieldOpts{
				Required: true, Error: "That date is in a closed period.",
			}, in,
		),
	)
	want(t, out, `<label class="field-label" for="j-date">`)
	want(t, out, `<span class="field-label__req" aria-hidden="true">*</span>`)
	want(t, out, `id="j-date-error">That date is in a closed period.</p>`)
	want(t, out, `aria-invalid`)
	want(t, out, `aria-describedby="j-date-error"`)
	want(t, out, `class="field-control field-control--invalid"`)

	out = renderC(t, FieldError("too long"))
	want(t, out, `<p class="field-error">too long</p>`)
}

func TestSearchFieldRenders(t *testing.T) {
	out := renderC(
		t, SearchField(
			SearchFieldOpts{
				Value: "cash", HXGet: templ.SafeURL("/journal?status=POSTED"),
			},
		),
	)
	want(t, out, `class="search-field"`)
	want(t, out, `type="search"`)
	want(t, out, `name="q"`)
	want(t, out, `value="cash"`)
	want(t, out, `hx-trigger="input changed delay:300ms"`)
	want(t, out, `hx-include="closest form"`)
	want(t, out, `hx-target="#rows"`)

	// Plain mode (no HXGet) carries no htmx attributes.
	out = renderC(t, SearchField(SearchFieldOpts{Name: "q", Placeholder: "Find…"}))
	if strings.Contains(out, "hx-") {
		t.Error("plain search must not emit htmx attributes")
	}
	want(t, out, `placeholder="Find…"`)
}

func TestTabBarRenders(t *testing.T) {
	q := url.Values{"status": {"POSTED"}, "q": {"cash"}, "page": {"3"}}
	tabs := []Tab{{Value: "", Label: "All", Count: 12}, {Value: "DRAFT", Label: "Draft"}}
	out := renderC(
		t, TabBar(
			tabs, "DRAFT", TabBarOpts{
				Param: "status", BaseHref: "/journal", BaseQuery: q, ShowCounts: true,
			},
		),
	)
	want(t, out, `<nav class="tab-bar" aria-label="Filters">`)
	// Active pill: aria-current + no status param; sibling q kept; page dropped.
	want(t, out, `href="/journal?q=cash&amp;status=DRAFT"`)
	want(t, out, `aria-current="true"`)
	want(t, out, `class="tab tab--active"`)
	// "All" pill sets status="" → param deleted.
	want(t, out, `href="/journal?q=cash"`)
	want(t, out, `class="tab__count">12</span>`)
}

func TestSegmentedRenders(t *testing.T) {
	opts := []SegOption{{Value: "ar", Label: "AR"}, {Value: "ap", Label: "AP", Disabled: true}}
	out := renderC(
		t, SegmentedControl(
			opts, "ar", SegmentedOpts{
				Param: "side", BaseHref: "/payments",
			},
		),
	)
	want(t, out, `<nav class="segmented" aria-label="View">`)
	want(t, out, `class="segmented__item segmented__item--active"`)
	want(t, out, `href="/payments?side=ar"`)
	want(t, out, `class="segmented__item segmented__item--disabled"`)
	want(t, out, `aria-disabled="true"`)
}

func TestKPITileRenders(t *testing.T) {
	out := renderC(
		t, KPITile(
			"Open AR", "$1,234.56", KPITileOpts{
				Hint: "unpaid invoices", Icon: "payments",
			},
		),
	)
	want(t, out, `<div class="kpi-tile">`)
	want(t, out, `class="kpi-tile__label"`)
	want(t, out, `>$1,234.56</p>`)
	want(t, out, `unpaid invoices`)
	out = renderC(
		t, KPITile("Shortfall", "-5.00", KPITileOpts{Tone: ToneDanger, Href: templ.SafeURL("/reports/trial-balance")}),
	)
	want(t, out, `<a class="kpi-tile" href="/reports/trial-balance">`)
	want(t, out, `kpi-tile__value kpi-tile__value--danger`)
}

func TestBalanceToolVerdicts(t *testing.T) {
	dr := mustAmt("100")
	cr := mustAmt("100")
	out := renderC(t, BalanceTool(dr, cr, BalanceToolOpts{Currency: "USD", Ready: true}))
	want(t, out, `beam--balanced`)
	want(t, out, `aria-label="Balanced at 100.00"`)
	want(t, out, `>Balanced`)

	// Unbalanced: verdict + diff, danger state.
	out = renderC(t, BalanceTool(mustAmt("150"), mustAmt("100"), BalanceToolOpts{Currency: "USD"}))
	want(t, out, `beam--unbalanced`)
	want(t, out, `>Out of balance by`)
	want(t, out, `>50.00</span>`)
	want(t, out, `aria-label="Out of balance by 50.00"`)

	// Empty.
	out = renderC(t, BalanceTool(mustAmt("0"), mustAmt("0"), BalanceToolOpts{Currency: "USD"}))
	want(t, out, `beam--empty`)
	want(t, out, `Enter debit and credit lines`)

	// Balanced but not ready appends the v2 hint.
	out = renderC(t, BalanceTool(dr, cr, BalanceToolOpts{Currency: "USD", Ready: false}))
	want(t, out, `— complete the lines above`)
}

func mustAmt(s string) Amount {
	a, err := ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestModalShellRenders(t *testing.T) {
	out := renderC(
		t, ModalShell(
			ModalOpts{ID: "dlg-party", Title: "New party"},
			Text("modal body"),
		),
	)
	want(t, out, `<dialog id="dlg-party"`)
	want(t, out, `aria-labelledby="dlg-party__title"`)
	// The dialog x-data names the ouiDialog FACTORY (Alpine.data): each dialog
	// gets its own scope object, and every directive is a bare reference so the
	// markup runs under Alpine's CSP build.
	want(t, out, `x-data="ouiDialog"`)
	want(t, out, `x-on:click="backdrop"`)
	want(t, out, `x-on:cancel.prevent="cancel"`)
	if strings.Contains(out, `LedgerUI.`) || strings.Contains(out, `($event)`) {
		t.Error(`dialog markup must not call LedgerUI.* or pass arguments (CSP-unsafe)`)
	}
	want(t, out, `>New party</h2>`)
	want(t, out, `data-close-dialog`)
	want(t, out, `modal body`)
}

func TestConfirmButtonRenders(t *testing.T) {
	out := renderC(
		t, ConfirmButton(
			"Void", "Void entry", "Void entry #12?",
			"It can't be undone; the entry keeps its number and shows as voided.",
			ConfirmOpts{
				ID: "void-12", Action: templ.SafeURL("/journal/12/void"),
				Fields: []KV{{Name: "reason", Value: "entered twice"}},
			}, nil,
		),
	)
	want(t, out, `<form id="void-12-form" method="post" action="/journal/12/void">`)
	want(t, out, `<input type="hidden" name="reason" value="entered twice">`)
	want(t, out, `data-open-dialog="void-12"`)
	want(t, out, `<dialog id="void-12" class="ui-dialog ui-dialog--confirm"`)
	want(t, out, `>Void entry #12?</h2>`)
	want(t, out, `shows as voided.`)
	// Same CSP-safe factory reference as ModalShell (see TestModalShellRenders).
	want(t, out, `x-data="ouiDialog"`)
	want(t, out, `<button type="button" class="ui-button ui-button--secondary" data-close-dialog>`)
	want(t, out, `<button type="submit" form="void-12-form" class="ui-button ui-button--danger">`)
	want(t, out, `<span class="confirm-label">Void entry</span>`)

	// Positive confirmations pick the primary variant.
	out = renderC(
		t, ConfirmButton(
			"Post", "Post entry", "Post entry #3?",
			"It is posted and numbered; edits after posting need a reversal.",
			ConfirmOpts{
				ID: "post-3", Tone: ToneInfo,
				Action: templ.SafeURL("/journal/3/post"),
			}, nil,
		),
	)
	want(t, out, `class="ui-button ui-button--primary"`)
}

func TestDrawerRenders(t *testing.T) {
	out := renderC(
		t, Drawer(
			DrawerOpts{ID: "drw-tpl", Title: "Edit template"},
			Text("drawer body"),
		),
	)
	want(t, out, `<div id="drw-tpl" class="drawer-overlay"`)
	// The overlay x-data names the ouiDrawer factory; directives are bare
	// references (CSP-safe).
	want(t, out, `x-data="ouiDrawer"`)
	want(t, out, `x-on:click="backdrop"`)
	want(t, out, `x-on:keydown.escape="escape"`)
	if strings.Contains(out, `LedgerUI.`) || strings.Contains(out, `($event)`) {
		t.Error(`drawer markup must not call LedgerUI.* or pass arguments (CSP-unsafe)`)
	}
	want(t, out, `class="drawer-panel"`)
	want(t, out, `role="dialog"`)
	want(t, out, `aria-modal="true"`)
	want(t, out, `aria-labelledby="drw-tpl__title"`)
	want(t, out, `tabindex="-1"`)
	want(t, out, `id="drw-tpl__title">Edit template</h2>`)
	want(t, out, `data-close-drawer`)
	want(t, out, `aria-label="Close"`)
	want(t, out, `class="drawer-content" id="drw-tpl-content"`)
	want(t, out, `drawer body`)

	// Subtitle + custom close label render alongside the title.
	out = renderC(
		t, Drawer(
			DrawerOpts{
				ID: "drw-x", Title: "New template",
				Subtitle: "Templates of ACME", CloseLabel: "Dismiss",
			}, Text("x"),
		),
	)
	want(t, out, `id="drw-x__title">New template</h2>`)
	want(t, out, `class="drawer-panel__subtitle">Templates of ACME</p>`)
	want(t, out, `aria-label="Dismiss"`)

	// The open/close wiring contract is attribute-driven (one app.js click
	// delegation serves every sheet): pages put data-open-drawer on any
	// trigger — here a Button carrying the canonical htmx fragment wiring
	// (docs/ui-components.md §3d, target = the {ID}-content region) — and the
	// component itself emits data-close-drawer on the × button.
	out = renderC(
		t, Button(
			ButtonVariantPrimary, "", ButtonOpts{
				Label: "Add new template",
				Attrs: templ.Attributes{
					"hx-get":           "/settings/templates/new",
					"hx-target":        "#drw-tpl-content",
					"hx-swap":          "innerHTML",
					"data-open-drawer": "drw-tpl",
				},
			},
		),
	)
	want(t, out, `data-open-drawer="drw-tpl"`)
	want(t, out, `hx-target="#drw-tpl-content"`)
}

func TestTableShellRenders(t *testing.T) {
	cols := []Column{
		{Key: "date", Label: "Date"},
		{Key: "total", Label: "Total", Numeric: true, Sortable: true},
	}
	q := url.Values{"status": {"POSTED"}}
	rows := templ.Raw("<tr><td>2026-09-03</td><td class=\"amount mono\">1,234.50</td></tr>")
	out := renderC(
		t, TableShell(
			cols, TableOpts{
				ID: "rows", Caption: "Journal entries",
				Sort: &SortState{Key: "total", Dir: "desc", BaseHref: "/journal", BaseQuery: q},
			}, rows, nil,
		),
	)
	want(t, out, `<div class="table-region" id="rows">`)
	want(t, out, `<caption class="visually-hidden">Journal entries</caption>`)
	want(t, out, `<th scope="col" data-column-key="total" class="amount"`)
	want(t, out, `aria-sort="descending"`)
	want(t, out, `href="/journal?sort=total&amp;status=POSTED"`)
	want(t, out, `<td>2026-09-03</td>`)

	// Empty state replaces rows with a full-width row + action.
	out = renderC(
		t, TableShell(
			cols, TableOpts{
				Empty: true, EmptyTitle: "No entries", EmptyMessage: "None yet.",
				EmptyActionHref: templ.SafeURL("/journal/new"), EmptyActionLabel: "New entry",
			}, nil, nil,
		),
	)
	want(t, out, `<td colspan="2">`)
	want(t, out, `No entries`)
	want(t, out, `href="/journal/new"`)

	// Footer (totals) renders inside tfoot.
	footer := templ.Raw("<tr><td colspan=\"2\">Total</td></tr>")
	out = renderC(t, TableShell(cols, TableOpts{}, nil, footer))
	want(t, out, `<tfoot>`)
	want(t, out, `Total`)
}

func TestTableShellMarksOptionalColumnsForScopedPreferences(t *testing.T) {
	out := renderC(t, TableShell(
		[]Column{{Key: "bank-created", Label: "Created", Optional: true}},
		TableOpts{}, templ.Raw(`<tr><td data-column-key="bank-created">Today</td></tr>`), nil,
	))
	want(t, out, `data-column-key="bank-created"`)
	want(t, out, `data-column-optional`)
}

func TestTableCellHelpers(t *testing.T) {
	out := renderC(t, MoneyCell(mustAmt("1234.5"), "USD"))
	want(t, out, `<td class="amount mono">1,234.50</td>`)
	out = renderC(t, SignedMoneyCell(mustAmt("-5"), "USD"))
	want(t, out, `>5.00 Cr</td>`)
	out = renderC(t, MonoCell("ACME"))
	want(t, out, `<td class="mono">ACME</td>`)
}

func TestPagerRenders(t *testing.T) {
	q := url.Values{"status": {"POSTED"}}
	out := renderC(
		t, Pager(
			2, 25, 217, PagerOpts{
				BaseHref: "/journal", BaseQuery: q,
			},
		),
	)
	want(t, out, `<p class="pager__summary">26–50 of 217</p>`)
	want(t, out, `href="/journal?page=1&amp;status=POSTED"`)
	want(t, out, `href="/journal?page=3&amp;status=POSTED"`)
	want(t, out, `aria-current="page"`) // current page link
	want(t, out, `pager__size--active`)
	want(t, out, `href="/journal?pageSize=50&amp;status=POSTED"`)

	// Zero rows: no pager at all.
	out = renderC(t, Pager(1, 25, 0, PagerOpts{BaseHref: "/journal"}))
	if strings.TrimSpace(out) != "" {
		t.Errorf("pager must be empty for zero rows, got %q", out)
	}
}

func TestEntityPickerRenders(t *testing.T) {
	out := renderC(
		t, EntityPicker(
			EntityPickerOpts{
				ID: "acct-0", Label: "Account", Name: "account_id",
				Endpoint: templ.SafeURL("/api/accounts?book=ACME"),
				Value:    "5", ValueLabel: "1000 Cash",
			},
		),
	)
	want(t, out, `class="entity-picker"`)
	want(t, out, `x-data="ouiPicker"`)
	want(t, out, `data-oui-picker="{`)
	want(t, out, `type="hidden"`)
	want(t, out, `name="account_id"`)
	want(t, out, `role="combobox"`)
	want(t, out, `aria-controls="acct-0-list"`)
	want(t, out, `role="listbox"`)
	// The Alpine config carries the endpoint + restored state (HTML-escaped
	// JSON inside the attribute).
	want(t, out, `/api/accounts?book=ACME`)
	want(t, out, `1000 Cash`)

	// Create is opt-in and MUST stay invisible until a call site asks for it:
	// the config key is omitted from the JSON (omitempty on the pointer, so the
	// substring cannot appear anywhere), the option list iterates the pre-create
	// rows, and none of the create-only rows exist. This is the guarantee the
	// ~15 untouched call sites depend on.
	if strings.Contains(out, `create`) {
		t.Errorf("a picker without Create must not render any create config:\n%s", out)
	}
	for _, absent := range []string{
		`x-show="creating"`, `x-show="createFailed"`, `Creating`,
	} {
		if strings.Contains(out, absent) {
			t.Errorf("a picker without Create must not render %q:\n%s", absent, out)
		}
	}
}

// Create arms the same markup for "create in place": the Alpine config grows a
// `create` sub-object, and the list switches to options() — the one iterable the
// keyboard model also walks, so the synthetic row is reachable by ArrowUp/Down
// and by aria-activedescendant rather than being a decorative extra.
func TestEntityPickerCreateOptIn(t *testing.T) {
	out := renderC(
		t, EntityPicker(
			EntityPickerOpts{
				ID: "acct-0", Label: "Account", Name: "account_id",
				Endpoint: templ.SafeURL("/api/accounts?book=ACME"),
				Create: &EntityPickerCreateOpts{
					Endpoint: templ.SafeURL("/api/accounts?book=ACME"),
					Field:    "name",
					Noun:     "bank account",
				},
			},
		),
	)
	// templ HTML-escapes the attribute's quotes, so assert the JSON fragments
	// unescaped — they are what app.js parses.
	want(t, out, `create`)
	want(t, out, `&#34;field&#34;:&#34;name&#34;`)
	want(t, out, `&#34;noun&#34;:&#34;bank account&#34;`)
	want(t, out, `x-show="creating"`)
	want(t, out, `x-show="createFailed"`)
	// The rows, the empty state and aria-activedescendant all read members the
	// runtime derives from options() (items plus the create row), so they share
	// one index space; the markup itself carries only bare references.
	want(t, out, `x-for="item in rows"`)
	want(t, out, `x-show="showEmpty"`)
	want(t, out, `x-bind:aria-activedescendant="activeDescendant"`)
	if strings.Contains(out, `activeId()`) || strings.Contains(out, `options()`) {
		t.Error(`picker markup must not call methods (CSP-unsafe)`)
	}
}

// A Create with no endpoint is a config error, not a dead button: the row would
// arm and then POST nowhere, so the picker degrades to the plain combobox.
func TestEntityPickerCreateWithoutEndpointIsDropped(t *testing.T) {
	out := renderC(
		t, EntityPicker(
			EntityPickerOpts{
				ID: "acct-0", Label: "Account", Name: "account_id",
				Endpoint: templ.SafeURL("/api/accounts?book=ACME"),
				Create:   &EntityPickerCreateOpts{Field: "name", Noun: "bank account"},
			},
		),
	)
	if strings.Contains(out, `&#34;create&#34;`) || strings.Contains(out, `Creating`) {
		t.Errorf("Create without an endpoint must not arm the row:\n%s", out)
	}
}

// The full-screen toggle is opt-in: a plain drawer must not grow a control it
// cannot honour, and an opted-in drawer must ship both icons (CSS reveals the
// right one per state, so no JavaScript touches icon markup — the same pattern
// as the shell's nav-toggle).
func TestDrawerFullscreenToggle(t *testing.T) {
	plain := renderC(t, Drawer(DrawerOpts{ID: "drw-plain", Title: "Plain"}, Text("body")))
	if strings.Contains(plain, "data-drawer-fullscreen") {
		t.Error("a drawer without Fullscreen must not render the toggle")
	}

	out := renderC(t, Drawer(DrawerOpts{ID: "drw-log", Title: "Log", Fullscreen: true}, Text("body")))
	want(t, out, `data-drawer-fullscreen`)
	want(t, out, `aria-pressed="false"`)
	want(t, out, `aria-label="Expand this panel to the whole screen"`)
	want(t, out, `drawer-panel__tool-icon--expand`)
	want(t, out, `drawer-panel__tool-icon--contract`)
	// The tools group must sit with the close button, inside the sheet header.
	want(t, out, `class="drawer-panel__tools"`)
}

// Tabbed drawers (DrawerOpts.Tabbed) trade the visible title for a tab strip:
// the header becomes one row of tabs plus the tools, and the h2 goes
// screen-reader-only (each tab's content carries its own heading). The strip is
// server-rendered but empty — tab state is per-reading-session client state — so
// its id and role are the whole server-side contract app.js depends on.
func TestDrawerTabbedStripsTheTitleAndMountsTheStrip(t *testing.T) {
	out := renderC(
		t, Drawer(
			DrawerOpts{
				ID: "log", Title: "Agent log", Subtitle: "Everything", Tabbed: true,
			}, Text("body"),
		),
	)

	if !strings.Contains(out, "drawer-panel drawer-panel--tabbed") {
		t.Error("a tabbed drawer must mark its panel so CSS and app.js can key the strip layout")
	}
	if !strings.Contains(out, `id="log-tabs"`) {
		t.Errorf("the strip must render at the id app.js looks for (log-tabs):\n%s", out)
	}
	if !strings.Contains(out, `class="drawer-tabs__track"`) || !strings.Contains(out, `role="tablist"`) {
		t.Error("the tab track must be a tablist")
	}
	// The scroll chevrons ship server-side (their icons come from the component
	// set); app.js only toggles their visibility per direction.
	if !strings.Contains(out, `data-log-tabs-wrap`) {
		t.Error("the strip needs its frame element for app.js to hang the chevron state on")
	}
	for _, want := range []string{
		`data-log-scroll="-1"`, `data-log-scroll="1"`, `aria-label="Scroll to earlier open records"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the strip must render its scroll navigation: %s", want)
		}
	}
	if !strings.Contains(out, `aria-label="Agent log — open records"`) {
		t.Error("the tab strip needs an accessible name")
	}
	// The h2 stays as the panel's accessible name, but hidden: the visible
	// heading is inside each tab's own content.
	if !strings.Contains(out, `id="log__title" class="visually-hidden">Agent log</h2>`) {
		t.Errorf(
			"a tabbed drawer's title must be screen-reader-only (app.js keeps it in step with the active tab):\n%s",
			out,
		)
	}
	if strings.Contains(out, "drawer-panel__subtitle") {
		t.Error("a tabbed drawer must not render the static subtitle — each record carries its own meta line")
	}
	// Progressive shape: the sheet is still openable/closable by the same
	// controls it had before.
	for _, want := range []string{`data-close-drawer`, `class="drawer-content" id="log-content"`} {
		if !strings.Contains(out, want) {
			t.Errorf("the tabbed drawer lost %q", want)
		}
	}
}

// A non-tabbed drawer must not grow a strip, and must keep its visible title.
func TestDrawerUntabbedKeepsTheVisibleTitle(t *testing.T) {
	out := renderC(t, Drawer(DrawerOpts{ID: "d", Title: "Template", Subtitle: "sub"}, Text("x")))
	if strings.Contains(out, "drawer-tabs") || strings.Contains(out, "drawer-panel--tabbed") {
		t.Error("an untabbed drawer must not render the tab strip")
	}
	if !strings.Contains(out, `id="d__title">Template</h2>`) {
		t.Error("an untabbed drawer keeps its visible title")
	}
	if !strings.Contains(out, "drawer-panel__subtitle") {
		t.Error("an untabbed drawer keeps its subtitle")
	}
}

// DrawerTabsID must stay in lockstep with the markup (app.js resolves the strip
// by id from the drawer id alone).
func TestDrawerTabsID(t *testing.T) {
	if got := DrawerTabsID("agent-log"); got != "agent-log-tabs" {
		t.Errorf("DrawerTabsID = %q", got)
	}
	if got := DrawerContentID("agent-log"); got != "agent-log-content" {
		t.Errorf("DrawerContentID = %q", got)
	}
}
