package components

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestPageFrameWidths(t *testing.T) {
	for width, class := range map[PageWidth]string{
		PageWidthStandard:  `class="ui-page"`,
		PageWidthNarrow:    `class="ui-page ui-page--narrow"`,
		PageWidthWide:      `class="ui-page ui-page--wide"`,
		PageWidthFull:      `class="ui-page ui-page--full"`,
		PageWidth("bogus"): `class="ui-page"`,
	} {
		out := renderC(t, withChildren(PageFrame(PageFrameOpts{Width: width}), "BODY"))
		want(t, out, class)
		want(t, out, "BODY")
	}
	out := renderC(t, withChildren(PageFrame(PageFrameOpts{ID: "main", Class: "settings"}), ""))
	want(t, out, `id="main" class="ui-page settings"`)
}

func TestFormGridColumnsAndActions(t *testing.T) {
	want(t, renderC(t, withChildren(FormGrid(FormGridOpts{}), "")), `class="ui-form-grid ui-form-grid--2"`)
	want(t, renderC(t, withChildren(FormGrid(FormGridOpts{Columns: 3}), "")), `class="ui-form-grid ui-form-grid--3"`)
	want(t, renderC(t, withChildren(FormGrid(FormGridOpts{Columns: 7}), "")), `class="ui-form-grid ui-form-grid--2"`)
	want(t, renderC(t, withChildren(FormActions(FormActionsOpts{End: true, Divided: true}), "SAVE")),
		`<div class="ui-form-actions ui-form-actions--end ui-form-actions--divided">SAVE</div>`)
}

// Starter controls read theme classes, not colour utilities
// that exist only when a host happens to define matching Tailwind colours.
func TestStarterComponentsUseThemeClasses(t *testing.T) {
	outs := []string{
		renderC(t, CharacterCount("n", "200")),
		renderC(t, TextInputField("Name", "n", "n", "m", "", "", "", "", false, "search")),
	}
	for _, out := range outs {
		for _, utility := range []string{"text-ink", "bg-paper", "text-accent"} {
			if strings.Contains(out, utility) {
				t.Errorf("renders host colour utility %s: %s", utility, out)
			}
		}
	}
	want(t, outs[0], `class="oui-character-count"`)
}

func withChildren(c templ.Component, text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, Text(text)), w)
	})
}

func TestFieldInputBoundsAndTypedControls(t *testing.T) {
	in := renderC(t, FieldInput(FieldInputOpts{ID: "amt", Type: "number", Min: "-1000", Max: "1000", Step: "0.01"}))
	want(t, in, `min="-1000"`)
	want(t, in, `max="1000"`)
	want(t, in, `step="0.01"`)
	if strings.Contains(renderC(t, FieldInput(FieldInputOpts{ID: "x"})), "min=") {
		t.Error("unset Min rendered an attribute")
	}

	ta := renderC(t, LabelledField("Notes", "notes", FieldOpts{Error: "Too long"},
		FieldTextarea(FieldTextareaOpts{ID: "notes", Name: "notes", Value: "<b>hi</b>", Invalid: true, AriaDescribedBy: FieldErrorID("notes")})))
	want(t, ta, `<textarea class="field-control field-control--invalid" rows="4" name="notes" id="notes" aria-invalid="true" aria-describedby="notes-error">&lt;b&gt;hi&lt;/b&gt;</textarea>`)
	want(t, ta, `<p class="field-error" id="notes-error">Too long</p>`)

	sel := renderC(t, FieldSelect(FieldSelectOpts{ID: "basis", Placeholder: "Choose…", Selected: "incl",
		Options: []SelectOption{{Value: "excl", Label: "Tax exclusive"}, {Value: "incl", Label: "Tax inclusive"}}}))
	want(t, sel, `<select class="field-control" id="basis">`)
	want(t, sel, `<option value="">Choose…</option>`)
	want(t, sel, `<option value="incl" selected>Tax inclusive</option>`)
	if strings.Contains(sel, `value="excl" selected`) {
		t.Errorf("unselected option marked selected: %s", sel)
	}
}
