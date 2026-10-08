package components

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/orbitalbase/orbitalui-htmx/assets"
)

func textC(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func idx(t *testing.T, out, needle string) int {
	t.Helper()
	i := strings.Index(out, needle)
	if i < 0 {
		t.Fatalf("missing %q in %s", needle, out)
	}
	return i
}

func TestButtonTrailingIconPair(t *testing.T) {
	out := renderC(t, Button("", "", ButtonOpts{Icon: "plus", TrailingIcon: "chevron-down", Label: "New"}))
	if strings.Count(out, "<svg") != 2 {
		t.Fatalf("want two icons: %s", out)
	}
	if !(idx(t, out, "<svg") < idx(t, out, "<span>New</span>") && idx(t, out, "<span>New</span>") < strings.LastIndex(out, "<svg")) {
		t.Errorf("leading icon, label, trailing icon expected: %s", out)
	}
	if strings.Contains(out, "ui-button--icon-right") {
		t.Errorf("TrailingIcon must not add the icon-right modifier: %s", out)
	}
	both := renderC(t, Button("", "", ButtonOpts{Icon: "arrow-right", IconRight: true, TrailingIcon: "chevron-down", Label: "Next"}))
	if strings.Index(both, "<svg") < strings.Index(both, "<span>Next</span>") || strings.Count(both, "<svg") != 2 {
		t.Errorf("IconRight + TrailingIcon should both sit after the label: %s", both)
	}
	// An icon-only button has no label to trail.
	only := renderC(t, Button("", "", ButtonOpts{Icon: "close", TrailingIcon: "chevron-down", IconOnly: true, Label: "Close"}))
	if strings.Count(only, "<svg") != 1 {
		t.Errorf("icon-only button drew a trailing icon: %s", only)
	}
}

func TestCardPaddedAndBare(t *testing.T) {
	out := renderC(t, Card(CardOpts{Title: "T", Padded: true}, textC("body")))
	want(t, out, `class="ui-card ui-card--padded"`)
	want(t, out, `class="ui-card__body"`)
	bare := renderC(t, Card(CardOpts{Bare: true, Padded: true, Class: "page-grid"}, textC("<i>x</i>")))
	want(t, bare, `class="ui-card ui-card--padded ui-card--bare page-grid"`)
	if strings.Contains(bare, "ui-card__body") {
		t.Errorf("Bare card kept the body wrapper: %s", bare)
	}
	def := renderC(t, Card(CardOpts{Title: "T"}, textC("b")))
	want(t, def, `class="ui-card"`)
}

func TestDataListComposable(t *testing.T) {
	out := renderC(t, withKids(DataList(DescriptionListOpts{Columns: 2}),
		withKids(DataItem("Role"), textC("Engineer")),
		withKids(DataItemWide("Status"), textC("<b>Active</b>")),
	))
	want(t, out, `<dl class="ui-dl ui-dl--stacked ui-dl--cols-2">`)
	want(t, out, `<div class="ui-dl__row"><dt>Role</dt><dd>Engineer</dd></div>`)
	want(t, out, `ui-dl__row ui-dl__row--wide`)
	want(t, out, `<b>Active</b>`)
}

func TestRingMeterArbitraryPx(t *testing.T) {
	out := renderC(t, RingMeter(RingMeterOpts{Value: 50, Px: 72}))
	want(t, out, `ui-ring--px`)
	want(t, out, `style="--ring-size:72px;"`)
	want(t, out, `width="72"`)
	preset := renderC(t, RingMeter(RingMeterOpts{Value: 50, Px: 64}))
	want(t, preset, `ui-ring--64`)
	if strings.Contains(preset, "style=") || strings.Contains(preset, "ui-ring--px") {
		t.Errorf("preset size must not set a style: %s", preset)
	}
	want(t, renderC(t, RingMeter(RingMeterOpts{Value: 50, Px: 28})), `--ring-size:28px`)
	want(t, renderC(t, RingMeter(RingMeterOpts{Value: 50, Px: 5})), `--ring-size:16px`)
	def := renderC(t, RingMeter(RingMeterOpts{Value: 50}))
	want(t, def, `ui-ring--56`)
}

func TestChipGroupOrderHelpError(t *testing.T) {
	out := renderC(t, ChipGroup([]ChipOption{{Value: "b", Label: "Beta", Suggested: true, Icon: "plus"}}, ChipGroupOpts{
		Name: "g", Legend: "Mode", Required: true, Help: "Pick one.", Error: "Required.",
	}))
	tick, label, spark := idx(t, out, "ui-choice-chip__tick"), idx(t, out, "ui-choice-chip__label"), idx(t, out, "ui-choice-chip__spark")
	if !(tick < label && label < spark) {
		t.Errorf("want tick, label, spark order: %s", out)
	}
	want(t, out, `<legend class="field-label ui-chip-group__legend">`)
	want(t, out, `field-label__req`)
	want(t, out, `<p class="field-error" id="g-error" role="alert">Required.</p>`)
	want(t, out, `<p class="field-help" id="g-help">Pick one.</p>`)
	want(t, out, `aria-describedby="g-error g-help"`)
	want(t, out, `class="ui-chip-group has-error"`)
	want(t, out, `title="Suggested"`)
}

func TestSegmentedNavAndTabNav(t *testing.T) {
	out := renderC(t, SegmentedNav([]SegItem{
		{Label: "One", Href: "/a"}, {Label: "Two", Href: "/b", Active: true, Count: "12", Icon: "plus"}, {Label: "Three", Href: "/c", Attrs: templ.Attributes{"data-x": "1"}},
	}, SegNavOpts{AriaLabel: "Sections"}))
	want(t, out, `class="ui-seg-nav ui-seg-nav--n3 ui-seg-nav--i1"`)
	want(t, out, `aria-label="Sections"`)
	want(t, out, `class="ui-seg-nav__item is-active" href="/b" aria-current="page"`)
	want(t, out, `<span class="ui-seg-nav__count">12</span>`)
	want(t, out, `data-x="1"`)
	none := renderC(t, SegmentedNav([]SegItem{{Label: "a", Href: "/a"}, {Label: "b", Href: "/b"}}, SegNavOpts{}))
	want(t, none, `class="ui-seg-nav ui-seg-nav--n2"`)
	many := make([]SegItem, 10)
	want(t, renderC(t, SegmentedNav(many, SegNavOpts{})), `ui-seg-nav--flow`)

	tabs := renderC(t, TabNav([]TabItem{{Label: "A", Href: "/a", Active: true, Count: "99+"}, {Label: "B", Href: "/b"}}, TabNavOpts{}))
	want(t, tabs, `<nav class="tab-bar" aria-label="Sections">`)
	want(t, tabs, `class="tab tab--active" href="/a" aria-current="page"`)
	want(t, tabs, `<span class="tab__count">99+</span>`)
	want(t, tabs, `<a class="tab" href="/b"`)
}

func TestSuggestedTagTitle(t *testing.T) {
	want(t, renderC(t, SuggestedTag("")), `<span class="ui-suggested" title="Suggested">`)
	want(t, renderC(t, SuggestedTagTitled("From CV", "Suggested by your agent")), `title="Suggested by your agent"`)
	f := renderC(t, LabelledField("L", "x", FieldOpts{Suggested: true, SuggestedTitle: "Agent"}, nil))
	want(t, f, `title="Agent"`)
}

func TestIconSized(t *testing.T) {
	want(t, renderC(t, IconSized("plus", 16)), `class="ui-icon--16"`)
	want(t, renderC(t, IconSized("plus", 12, "x")), `class="ui-icon--12 x"`)
	if strings.Contains(renderC(t, IconSized("plus", 17)), "ui-icon--") {
		t.Error("an unsupported size added a modifier")
	}
	want(t, renderC(t, Icon("plus")), `class=""`)
}

func TestKitParityStyles(t *testing.T) {
	css := func(name string) string {
		b, err := assets.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	theme, tokens := css("theme.css"), css("tokens.css")
	for _, needle := range []string{
		".ui-badge__dot { width: 0.375rem; height: 0.375rem; flex: 0 0 auto; border-radius: 50%; background: var(--oui-badge-dot, currentColor); }",
		"inset 0 0 0 1.5px var(--oui-badge-dot, currentColor)",
		"--oui-avatar-fs: 8px", "--oui-avatar-fs: 14.5px", "--oui-avatar-fs: 26px; letter-spacing: -0.02em",
		"var(--oui-orb-a)", "var(--oui-card-title-family", "var(--oui-state-title-family", "var(--oui-ring-glow, none)",
		".ui-dl__row { align-content: start; }", "svg.ui-icon--24",
	} {
		if !strings.Contains(theme, needle) {
			t.Errorf("theme.css missing %q", needle)
		}
	}
	// The orb stops are defined once, in the light block, and never again for dark.
	if strings.Count(tokens, "--oui-orb-a:") != 1 {
		t.Errorf("--oui-orb-a must be defined exactly once (not redefined in dark), got %d", strings.Count(tokens, "--oui-orb-a:"))
	}
	if strings.Contains(theme, ".ui-button--icon-right { flex-direction: row-reverse") {
		t.Error("IconRight still reverses the row")
	}
	js := css("shell.js")
	for _, needle := range []string{`"oui:palette-submit"`, "cancelable: true", "detail"} {
		if !strings.Contains(js, needle) {
			t.Errorf("shell.js missing %s", needle)
		}
	}
}
