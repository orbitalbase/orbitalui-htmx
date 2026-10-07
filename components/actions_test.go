package components

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestButtonAIVariantAndLargeSize(t *testing.T) {
	out := renderC(t, Button(ButtonVariantAI, ButtonSizeLG, ButtonOpts{Icon: "spark", Label: "Ask"}))
	want(t, out, `class="ui-button ui-button--ai ui-button--lg"`)
	want(t, out, `<span>Ask</span>`)
}

func TestButtonIconOnlyUsesLabelAsAccessibleName(t *testing.T) {
	out := renderC(t, Button(ButtonVariantQuiet, "", ButtonOpts{Icon: "close", Label: "Dismiss", IconOnly: true}))
	want(t, out, `class="ui-button ui-button--quiet ui-button--icon"`)
	want(t, out, `aria-label="Dismiss"`)
	want(t, out, `title="Dismiss"`)
	if strings.Contains(out, "<span>Dismiss</span>") {
		t.Errorf("icon-only button drew its label: %s", out)
	}
	// A caller-supplied aria-label wins; no duplicate attribute is emitted.
	own := renderC(t, Button("", "", ButtonOpts{Icon: "close", Label: "Dismiss", IconOnly: true, Attrs: templ.Attributes{"aria-label": "Close panel"}}))
	if strings.Count(own, "aria-label=") != 1 || !strings.Contains(own, `aria-label="Close panel"`) {
		t.Errorf("aria-label from Attrs not preserved exactly once: %s", own)
	}
	link := renderC(t, Button("", "", ButtonOpts{Icon: "close", Label: "Back", IconOnly: true, Href: "/x"}))
	want(t, link, `<a href="/x"`)
	want(t, link, `aria-label="Back"`)
}

func TestButtonBlockAndIconRight(t *testing.T) {
	out := renderC(t, Button(ButtonVariantPrimary, "", ButtonOpts{Icon: "arrow-right", Label: "Next", IconRight: true, Block: true}))
	want(t, out, `ui-button--block`)
	want(t, out, `ui-button--icon-right`)
	if strings.Index(out, "<span>Next</span>") > strings.Index(out, "<svg") {
		t.Errorf("IconRight did not place the icon after the label: %s", out)
	}
	left := renderC(t, Button("", "", ButtonOpts{Icon: "plus", Label: "Add"}))
	if strings.Index(left, "<svg") > strings.Index(left, "<span>Add</span>") {
		t.Errorf("default icon is not before the label: %s", left)
	}
}

func TestBadgeTonesAndOptions(t *testing.T) {
	out := renderC(t, Badge(ToneBrand, "Talent", BadgeOpts{Dot: true}))
	want(t, out, `class="ui-badge ui-badge--brand"`)
	want(t, out, `<span class="ui-badge__dot" aria-hidden="true"></span>`)
	want(t, out, `Talent`)

	for _, tone := range []Tone{ToneAccent, ToneInfo, ToneSuccess, ToneWarning, ToneDanger, ToneAI} {
		want(t, renderC(t, Badge(tone, "x", BadgeOpts{})), "ui-badge--"+string(tone))
	}
	neutral := renderC(t, Badge(ToneNeutral, "Draft", BadgeOpts{}))
	want(t, neutral, `class="ui-badge"`)
	if strings.Contains(neutral, "ui-badge__dot") {
		t.Errorf("dot rendered without Dot: %s", neutral)
	}

	hollow := renderC(t, Badge(ToneWarning, "Pending", BadgeOpts{Hollow: true}))
	want(t, hollow, `ui-badge--hollow`)
	want(t, hollow, `ui-badge__dot`)

	muted := renderC(t, Badge(ToneDanger, "Archived", BadgeOpts{Muted: true, Mono: true, Class: "host"}))
	want(t, muted, `class="ui-badge ui-badge--muted ui-badge--mono host"`)
	if strings.Contains(muted, "ui-badge--danger") {
		t.Errorf("muted badge kept its tone: %s", muted)
	}

	icon := renderC(t, Badge(ToneAI, "Suggested", BadgeOpts{Icon: "spark"}))
	want(t, icon, `<svg`)
}

func TestStatusBadgeAcceptsBrandAndAccent(t *testing.T) {
	want(t, renderC(t, StatusBadge("Lead", ToneBrand, StatusBadgeOpts{})), `class="ui-status ui-status--brand"`)
	want(t, renderC(t, StatusBadge("Hiring", ToneAccent, StatusBadgeOpts{})), `class="ui-status ui-status--accent"`)
	if !ToneBrand.Valid() || !ToneAccent.Valid() {
		t.Error("brand and accent tones are not valid")
	}
	if got := Tone("bogus").Modifier(); got != "neutral" {
		t.Errorf("unknown tone modifier = %q", got)
	}
}

func TestCardInteractiveEdgeAndSubtitle(t *testing.T) {
	out := renderC(t, Card(CardOpts{Title: "Match", Subtitle: "Senior engineer", Icon: "briefcase", Interactive: true, Edge: ToneAccent}, Text("B")))
	want(t, out, `class="ui-card ui-card--edge ui-card--edge-accent ui-card--interactive"`)
	want(t, out, `<p class="ui-card__subtitle">Senior engineer</p>`)
	want(t, out, `class="ui-card__icon"`)
	plain := renderC(t, Card(CardOpts{Edge: ToneNeutral}, Text("B")))
	if strings.Contains(plain, "ui-card--edge") {
		t.Errorf("neutral edge rendered: %s", plain)
	}
}

func TestCardHrefStretchesTitleLink(t *testing.T) {
	out := renderC(t, Card(CardOpts{ID: "c1", Title: "Acme", Href: "/companies/acme", Actions: Text("ACT")}, Text("B")))
	want(t, out, `ui-card--interactive ui-card--link`)
	want(t, out, `<a class="ui-card__link" href="/companies/acme">Acme</a>`)
	want(t, out, `<div class="ui-card__actions">ACT</div>`)
	if strings.Count(out, `href="/companies/acme"`) != 1 {
		t.Errorf("expected exactly one link: %s", out)
	}
	bare := renderC(t, Card(CardOpts{Href: "/x", LinkLabel: "Open Acme"}, Text("B")))
	want(t, bare, `<a class="ui-card__link" href="/x" aria-label="Open Acme"></a>`)
}

func TestCardSection(t *testing.T) {
	want(t, renderC(t, CardSection(CardSectionOpts{}, Text("S"))), `<div class="ui-card__section">S</div>`)
	want(t, renderC(t, CardSection(CardSectionOpts{Divided: true, Class: "x"}, Text("S"))), `class="ui-card__section ui-card__section--divided x"`)
}

func TestKbd(t *testing.T) {
	want(t, renderC(t, Kbd("Esc")), `<kbd class="ui-kbd">Esc</kbd>`)
	multi := renderC(t, Kbd("⌘", "K"))
	want(t, multi, `<span class="ui-kbd-group">`)
	if strings.Count(multi, "<kbd") != 2 {
		t.Errorf("combination did not render two keys: %s", multi)
	}
	want(t, renderC(t, KbdGroup("G")), `ui-kbd-group`)
	want(t, renderC(t, Kbd("<")), `&lt;`)
}
