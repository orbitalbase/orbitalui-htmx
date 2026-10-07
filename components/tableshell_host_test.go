package components

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestHostTableSortingContracts(t *testing.T) {
	columns := []Column{
		{Label: "Name", SortHref: "/clients?sort=name-desc&search=hello%20world", SortDirection: "ascending", Width: "12rem"},
		{Label: "Status", ClientSort: "status", SortDirection: "invalid"},
		{Key: "amount", Label: "Amount", Numeric: true, Sortable: true},
	}
	out := renderC(t, TableShell(columns, TableOpts{TableClass: "table-wide", SortClass: "table-sort"}, Text("ROW"), nil))
	for _, expected := range []string{`class="ledger-table table-wide"`, `href="/clients?sort=name-desc&amp;search=hello%20world"`, `aria-sort="ascending"`, `style="width: 12rem;"`, `data-table-sort="status"`, `type="button"`} {
		if !strings.Contains(out, expected) {
			t.Errorf("missing %s in %s", expected, out)
		}
	}
	if strings.Contains(out, `aria-sort="invalid"`) || strings.Contains(out, `@click`) {
		t.Fatal("invalid sort semantics or inline runtime expression")
	}
	// Existing Sortable columns remain plain unless the host supplies SortState.
	if strings.Count(out, "<a ") != 1 || strings.Count(out, "<button ") != 1 {
		t.Fatal("new defaults changed existing sortable columns")
	}
}

func TestHostEmptyTableAndZeroColumns(t *testing.T) {
	out := renderC(t, TableShell(nil, TableOpts{Empty: true, EmptyContent: Text("HOST EMPTY")}, Text("HIDDEN ROW"), nil))
	if !strings.Contains(out, `colspan="1"`) || !strings.Contains(out, "HOST EMPTY") || strings.Contains(out, "HIDDEN ROW") {
		t.Fatal(out)
	}
	// Direct URLs remain escaped by templ's URL sanitizer.
	out = renderC(t, TableShell([]Column{{Label: "Unsafe", SortHref: templ.SafeURL("/safe?x=<script>")}}, TableOpts{}, nil, nil))
	if strings.Contains(out, "<script>") {
		t.Fatal("unsafe attribute output")
	}
}
