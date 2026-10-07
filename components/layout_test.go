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

// Starter controls and chat surfaces read theme classes, not colour utilities
// that exist only when a host happens to define matching Tailwind colours.
func TestStarterComponentsUseThemeClasses(t *testing.T) {
	outs := []string{
		renderC(t, SummaryCard("Draft", "spark", "card", SummaryField("Company", "c"))),
		renderC(t, AssistantMessage("Hello")),
		renderC(t, UserMessage("Hi")),
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
	want(t, outs[0], `oui-summary-card__icon`)
	want(t, outs[1], `oui-assistant-message__body`)
	want(t, outs[3], `class="oui-character-count"`)
}

func withChildren(c templ.Component, text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, Text(text)), w)
	})
}
