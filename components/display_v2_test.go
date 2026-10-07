package components

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func notWant(t *testing.T, out, needle string) {
	t.Helper()
	if strings.Contains(out, needle) {
		t.Errorf("rendered output must not contain %q\n---\n%s", needle, out)
	}
}

func TestMeterRendersExactValueAndStepClass(t *testing.T) {
	out := renderC(t, Meter(MeterOpts{Value: 72, Tone: ToneAI, Size: "sm", Label: "Profile strength"}))
	want(t, out, `role="meter"`)
	want(t, out, `aria-valuenow="72"`)
	want(t, out, `aria-label="Profile strength"`)
	want(t, out, `ui-meter--ai`)
	want(t, out, `ui-meter--sm`)
	want(t, out, `ui-meter__fill--72`)
	notWant(t, out, "style=")
	over := renderC(t, Meter(MeterOpts{Value: 400}))
	want(t, over, `aria-valuenow="100"`)
	want(t, over, `ui-meter__fill--100`)
	notWant(t, over, "aria-label")
	// Unknown tone degrades to the default fill, never a wrong colour.
	notWant(t, renderC(t, Meter(MeterOpts{Value: 1, Tone: "bogus"})), "ui-meter--bogus")
}

func TestRingMeterUsesAttributesNotInlineStyle(t *testing.T) {
	out := renderC(t, RingMeter(RingMeterOpts{Value: 82, Size: 64, Tone: ToneAI, Name: "Match score"}))
	want(t, out, `role="img"`)
	want(t, out, `aria-label="Match score: 82 percent"`)
	want(t, out, `ui-ring--64`)
	want(t, out, `ui-ring--ai`)
	want(t, out, `pathLength="100"`)
	want(t, out, `stroke-dasharray="100"`)
	want(t, out, `stroke-dashoffset="18"`)
	want(t, out, `stroke="url(#oui-ring-grad-64)"`)
	want(t, out, `<linearGradient id="oui-ring-grad-64"`)
	want(t, out, `>82</span>`)
	notWant(t, out, "style=")

	plain := renderC(t, RingMeter(RingMeterOpts{Value: 0, Size: 999, Label: "0/9"}))
	want(t, plain, `ui-ring--56`)
	want(t, plain, `ui-ring--action`)
	want(t, plain, `>0/9</span>`)
	notWant(t, plain, "ui-ring__arc") // an empty ring draws no arc
	notWant(t, plain, "linearGradient")
	want(t, plain, `aria-label="Score: 0 percent"`)
}

func TestTimelineItemStatesAndLinks(t *testing.T) {
	body := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "Published.")
		return err
	})
	ctx := templ.WithChildren(context.Background(), body)
	var sb strings.Builder
	if err := TimelineItem(TimelineItemOpts{Title: "v3", Meta: "Today", Current: true, Tone: ToneSuccess, Mono: true, Href: "/v/3"}).Render(ctx, &sb); err != nil {
		t.Fatal(err)
	}
	out := sb.String()
	want(t, out, `ui-timeline__item ui-timeline__item--success is-current`)
	want(t, out, `aria-current="step"`)
	want(t, out, `<a href="/v/3" class="ui-timeline__title ui-timeline__title--mono">v3</a>`)
	want(t, out, `<span class="ui-timeline__meta">Today</span>`)
	want(t, out, `Published.`)

	list := renderC(t, Timeline())
	want(t, list, `<ol class="ui-timeline">`)
	plain := renderC(t, TimelineItem(TimelineItemOpts{Title: "v1"}))
	notWant(t, plain, "aria-current")
	notWant(t, plain, "<a ")
}

func TestOrbSizesStatesAndAccessibility(t *testing.T) {
	out := renderC(t, Orb(OrbOpts{Size: 40, State: OrbThinking, Label: "Assistant is thinking"}))
	want(t, out, `ui-orb ui-orb--40 is-thinking`)
	want(t, out, `role="img"`)
	want(t, out, `aria-label="Assistant is thinking"`)
	want(t, out, `ui-orb__core`)
	notWant(t, out, "ui-orb--xs")

	small := renderC(t, Orb(OrbOpts{Size: 16}))
	want(t, small, `ui-orb--16 ui-orb--xs`)
	want(t, small, `aria-hidden="true"`)
	notWant(t, small, "is-idle")
	want(t, renderC(t, Orb(OrbOpts{Size: 33})), `ui-orb--28`)
}

func TestEntityChipLinkAndAnonymous(t *testing.T) {
	out := renderC(t, EntityChip(EntityChipOpts{Name: "Adri Tan", Href: "/people/42"}))
	want(t, out, `<a class="ui-entity-chip ui-entity-chip--link" href="/people/42">`)
	want(t, out, `ui-avatar--20`)
	want(t, out, `>AT</span>`)
	want(t, out, `<span class="ui-entity-chip__name">Adri Tan</span>`)

	org := renderC(t, EntityChip(EntityChipOpts{Name: "Northwind", Organisation: true}))
	want(t, org, `<span class="ui-entity-chip">`)
	want(t, org, `ui-avatar--square`)

	anon := renderC(t, EntityChip(EntityChipOpts{Name: "Candidate", Anonymous: true}))
	want(t, anon, `ui-avatar--anon`)
	want(t, anon, `aria-label="Anonymous"`)
	notWant(t, anon, "ui-avatar__initials")
}

func TestAvatarScaleAnonymousAndTones(t *testing.T) {
	out := renderC(t, Avatar("Adri Tan", AvatarOpts{Px: 72, Tone: "accent"}))
	want(t, out, `ui-avatar ui-avatar--72 ui-avatar--accent`)
	want(t, out, `aria-hidden="true"`)
	// An unsupported pixel size falls back to the named size.
	want(t, renderC(t, Avatar("A", AvatarOpts{Px: 50, Size: AvatarLG, Tone: "brand"})), `ui-avatar ui-avatar--lg ui-avatar--brand`)
	// Legacy names keep working.
	want(t, renderC(t, Avatar("A", AvatarOpts{Size: AvatarXS, Tone: "gray"})), `ui-avatar--xs ui-avatar--gray`)
	img := renderC(t, Avatar("A", AvatarOpts{Src: "/a.png", Label: "Adri"}))
	want(t, img, `<img class="ui-avatar__img" src="/a.png"`)
	want(t, img, `aria-label="Adri"`)
	dot := renderC(t, Avatar("A", AvatarOpts{Status: "online"}))
	want(t, dot, `ui-avatar__status--online`)
}

func TestDescriptionListColumnsAndFieldHook(t *testing.T) {
	out := renderC(t, DescriptionList([]DescriptionItem{
		{Term: "Salary", Value: "120,000", Field: "salary", Mono: true},
		{Term: "Notes", Value: "", Wide: true},
	}, DescriptionListOpts{Columns: 3}))
	want(t, out, `class="ui-dl ui-dl--stacked ui-dl--cols-3"`)
	want(t, out, `<dd data-field="salary">`)
	want(t, out, `ui-dl__value ui-dl__value--mono`)
	want(t, out, `ui-dl__empty`)
	want(t, out, `ui-dl__row ui-dl__row--wide`)
	// Columns clamp to three; the old layouts are untouched.
	want(t, renderC(t, DescriptionList(nil, DescriptionListOpts{Columns: 9})), `ui-dl--cols-3`)
	want(t, renderC(t, DescriptionList(nil, DescriptionListOpts{})), `ui-dl ui-dl--inline`)
	want(t, renderC(t, DescriptionList(nil, DescriptionListOpts{Layout: DescriptionGrid})), `ui-dl ui-dl--grid`)
}

func TestStatePanelOrbAndActions(t *testing.T) {
	actions := templ.Raw(`<button>Ask the agent</button>`)
	out := renderC(t, StatePanel("Nothing here", StatePanelOpts{Orb: true, Icon: "check", Actions: actions}, nil))
	want(t, out, `ui-orb ui-orb--40 is-idle`)
	notWant(t, out, `ui-state__icon`)
	want(t, out, `<div class="ui-state__actions"><button>Ask the agent</button></div>`)
	icon := renderC(t, StatePanel("Nothing here", StatePanelOpts{Icon: "check"}, nil))
	want(t, icon, `ui-state__icon`)
	notWant(t, icon, "ui-orb")
	notWant(t, icon, "ui-state__actions")
}

func TestChecklistCompactVariant(t *testing.T) {
	items := []ChecklistItem{{Title: "Profile", Done: true}, {Title: "Verify", Description: "ID check"}}
	want(t, renderC(t, ChecklistWith(items, ChecklistOpts{Compact: true})), `class="ui-checklist ui-checklist--compact"`)
	plain := renderC(t, Checklist(items))
	want(t, plain, `class="ui-checklist"`)
	notWant(t, plain, "compact")
}

func TestProgressAndFeedAcceptIdentityTones(t *testing.T) {
	want(t, renderC(t, ProgressBar(40, ProgressOpts{Tone: toneAccentName})), `ui-progress ui-progress--accent`)
	want(t, renderC(t, FeedItem(FeedItemOpts{Title: "Saved", Icon: "check", Tone: toneBrandName})), `ui-feed__icon ui-feed__icon--brand`)
	notWant(t, renderC(t, ProgressBar(40, ProgressOpts{})), "ui-progress--")
}
