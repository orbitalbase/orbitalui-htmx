package components

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/orbitalbase/orbitalui-htmx/assets"
)

func withBody(c templ.Component, body string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, textC(body)), w)
	})
}

func TestFieldHelpIDErrorRoleAndDescribedBy(t *testing.T) {
	out := renderC(t, LabelledField("Date", "d", FieldOpts{Help: "h", Error: "e"}, nil))
	want(t, out, `<p class="field-error" id="d-error">e</p>`)
	want(t, out, `<p class="field-help" id="d-help">h</p>`)
	out = renderC(t, LabelledField("Date", "d", FieldOpts{Error: "e", ErrorAlert: true}, nil))
	want(t, out, `id="d-error" role="alert"`)
	// A field with neither help nor error renders exactly as before.
	notWant(t, renderC(t, LabelledField("Date", "d", FieldOpts{}, nil)), "id=")
	// No htmlFor, no help id.
	notWant(t, renderC(t, LabelledField("Date", "", FieldOpts{Help: "h"}, nil)), `id="-help"`)
	for _, c := range []struct {
		o    FieldOpts
		want string
	}{{FieldOpts{}, ""}, {FieldOpts{Help: "h"}, "d-help"}, {FieldOpts{Error: "e"}, "d-error"}, {FieldOpts{Help: "h", Error: "e"}, "d-help d-error"}} {
		if got := FieldDescribedBy("d", c.o); got != c.want {
			t.Errorf("FieldDescribedBy(%+v) = %q, want %q", c.o, got, c.want)
		}
	}
}

func TestFieldNumericAndPlaceholderDisabled(t *testing.T) {
	want(t, renderC(t, FieldInput(FieldInputOpts{ID: "n", Numeric: true})), `class="field-control num"`)
	notWant(t, renderC(t, FieldInput(FieldInputOpts{ID: "n"})), " num")
	sel := FieldSelect(FieldSelectOpts{ID: "s", Placeholder: "Choose", PlaceholderDisabled: true, Options: []SelectOption{{Value: "a", Label: "A"}}})
	want(t, renderC(t, sel), `<option value="" selected disabled>Choose</option>`)
	notWant(t, renderC(t, FieldSelect(FieldSelectOpts{ID: "s", Placeholder: "Choose"})), "disabled")
}

func TestStatePanelTitleTag(t *testing.T) {
	want(t, renderC(t, StatePanel("T", StatePanelOpts{}, nil)), "<h2>T</h2>")
	want(t, renderC(t, StatePanel("T", StatePanelOpts{TitleTag: "h3"}, nil)), "<h3>T</h3>")
	want(t, renderC(t, StatePanel("T", StatePanelOpts{TitleTag: "h1"}, nil)), "<h2>T</h2>")
}

func TestChipGroupSuggestedTitle(t *testing.T) {
	opts := []ChipOption{{Value: "a", Label: "A", Suggested: true}}
	want(t, renderC(t, ChipGroup(opts, ChipGroupOpts{Name: "n"})), `title="Suggested"`)
	want(t, renderC(t, ChipGroup(opts, ChipGroupOpts{Name: "n", SuggestedTitle: "By agent"})), `title="By agent"`)
}

func TestDataItemWith(t *testing.T) {
	out := renderC(t, withBody(DataItemWith("L", DataItemOpts{Wide: true, Field: "f"}), "v"))
	want(t, out, `<div class="ui-dl__row ui-dl__row--wide">`)
	want(t, out, `<dd data-field="f">v</dd>`)
	// DataItem keeps its markup.
	out = renderC(t, withBody(DataItem("L"), "v"))
	want(t, out, `<div class="ui-dl__row"><dt>L</dt><dd>v</dd></div>`)
}

func TestCardHeaderMatchesCard(t *testing.T) {
	card := renderC(t, Card(CardOpts{Title: "T", Subtitle: "S", Icon: "plus", Bare: true}, nil))
	hdr := renderC(t, CardHeader(CardHeaderOpts{Title: "T", Subtitle: "S", Icon: "plus"}))
	if !strings.Contains(card, hdr) {
		t.Errorf("Card header differs from CardHeader\ncard: %s\nhdr:  %s", card, hdr)
	}
	want(t, renderC(t, CardHeader(CardHeaderOpts{Title: "T", HeadingLevel: 3, HeadingID: "x"})), `<h3 class="ui-card__title" id="x">`)
}

func TestFinishCSSTokens(t *testing.T) {
	b, err := assets.Files.ReadFile("theme.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	for _, tok := range []string{"--oui-control-pad-x", "--oui-tab-height-phone", "--oui-chip-suggested-ring", "--oui-chip-suggested-halo",
		"--oui-textarea-min-height", "--oui-textarea-resize", "--oui-badge-dot", "--oui-control-ring-inset"} {
		if !strings.Contains(css, tok) {
			t.Errorf("theme.css does not read %s", tok)
		}
	}
}
