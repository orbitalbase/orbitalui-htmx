package components

import (
	"strings"
	"testing"
	"time"
)

func TestActionRowLinkedAndPlain(t *testing.T) {
	out := renderC(t, ActionList(ActionListOpts{}, ActionRow(ActionRowOpts{
		Title: "Board minutes", Meta: "Resolution", Href: "/documents/42", Icon: "document", IconTone: ToneInfo, Value: "3",
		Trailing: Text("BADGE"),
	})))
	want(t, out, `<ul class="ui-rows ui-rows__list" role="list">`)
	want(t, out, `<a class="ui-row ui-row--link" href="/documents/42">`)
	want(t, out, `class="ui-row__icon ui-row__icon--info" aria-hidden="true"`)
	want(t, out, `<span class="ui-row__title">Board minutes</span>`)
	want(t, out, `<span class="ui-row__value">3</span> BADGE`)
	want(t, out, `class="ui-row__chevron"`)

	plain := renderC(t, ActionRow(ActionRowOpts{Title: "Mark read", Trailing: Text("FORM"), Below: Text("BELOW"), Stacked: true}))
	want(t, plain, `<li class="ui-rows__item ui-rows__item--stacked"><div class="ui-row">`)
	want(t, plain, `<div class="ui-row__below">BELOW</div>`)
	if strings.Contains(plain, "ui-row__chevron") {
		t.Errorf("an unlinked row must not render a chevron: %s", plain)
	}

	nav := renderC(t, ActionList(ActionListOpts{Label: "Firm records", Variant: ActionListCards}, ActionRow(ActionRowOpts{Title: "Clients", Href: "/admin/clients", Current: true})))
	want(t, nav, `<nav class="ui-rows ui-rows--cards" aria-label="Firm records">`)
	want(t, nav, `aria-current="page"`)
}

func TestFactListAndCriteria(t *testing.T) {
	if renderC(t, FactList(nil)) != "" {
		t.Errorf("an empty fact list rendered markup")
	}
	out := renderC(t, FactList([]Fact{{Term: "Price", Value: "MYR 350.00"}}))
	want(t, out, `<dl class="ui-facts"><div class="ui-facts__item"><dt>Price</dt><dd>MYR 350.00</dd>`)

	crit := renderC(t, CriteriaList([]Criterion{{Met: true, Label: "Registered in Malaysia"}, {Label: "Revenue under MYR 50m", Note: "Recorded: MYR 80m"}}))
	want(t, crit, `class="ui-criteria__item is-met"`)
	want(t, crit, `<span class="sr-only">Met: </span>Registered in Malaysia`)
	want(t, crit, `<span class="sr-only">Not met: </span>Revenue under MYR 50m`)
	want(t, crit, `<span class="ui-criteria__note">Recorded: MYR 80m</span>`)
}

func TestChoiceRowAndGroup(t *testing.T) {
	out := renderC(t, ChoiceGroup("Email alerts", ChoiceGroupOpts{}, ChoiceRow(ChoiceRowOpts{
		Name: "alert_deadlines", Value: "1", Label: "Statutory deadlines", Description: "Filing dates as they approach.", Checked: true,
	})))
	want(t, out, `<fieldset class="ui-choices"><legend class="sr-only">Email alerts</legend>`)
	want(t, out, `<input type="checkbox" name="alert_deadlines" value="1" checked>`)
	want(t, out, `<span class="ui-choice__hint">Filing dates as they approach.</span>`)

	radio := renderC(t, ChoiceRow(ChoiceRowOpts{Type: "radio", Name: "plan", Value: "pro", Label: "Pro", Disabled: true}))
	want(t, radio, `class="ui-choice is-disabled"`)
	want(t, radio, `type="radio"`)
	want(t, radio, ` disabled>`)
}

func TestCalloutAndDecisionCard(t *testing.T) {
	out := renderC(t, Callout(ToneWarning, CalloutOpts{ID: "queue", Title: "Companies waiting", Description: "Two asked to join.", Icon: "link", Actions: Text("ACT")}, Text("ROWS")))
	want(t, out, `class="ui-callout ui-callout--warning" id="queue" aria-labelledby="queue-title"`)
	want(t, out, `<h2 class="ui-callout__title" id="queue-title">Companies waiting</h2>`)
	want(t, out, `<div class="ui-callout__actions">ACT</div>`)
	want(t, out, `<div class="ui-callout__body">ROWS</div>`)

	neutral := renderC(t, Callout(ToneNeutral, CalloutOpts{Title: "Note"}, nil))
	want(t, neutral, `class="ui-callout ui-callout--info"`)

	card := renderC(t, DecisionCard(DecisionCardOpts{ID: "d1", Subject: "Share transfer", Context: "v3", Evidence: Text("EVIDENCE"), Actions: Text("ACTIONS")}, Text("BODY")))
	want(t, card, `<article class="ui-decision" id="d1" aria-labelledby="d1-subject">`)
	want(t, card, `<h3 class="ui-decision__subject" id="d1-subject">Share transfer</h3>`)
	want(t, card, `<div class="ui-decision__evidence">EVIDENCE</div><div class="ui-decision__actions">ACTIONS</div>`)
}

func TestFilterChipsAndSectionNav(t *testing.T) {
	out := renderC(t, FilterChips([]ChipLink{{Label: "All", Href: "/s"}, {Label: "Filings", Href: "/s?c=f", Value: "f", Count: 4}}, "f", FilterChipsOpts{Label: "Filter by category"}))
	want(t, out, `<nav class="ui-chips" aria-label="Filter by category">`)
	want(t, out, `<a class="ui-chip" href="/s" aria-current="false"><span class="ui-chip__label">All</span>`)
	want(t, out, `<a class="ui-chip is-active" href="/s?c=f" aria-current="page"><span class="ui-chip__label">Filings</span> <span class="ui-chip__count">4</span></a>`)
	if renderC(t, FilterChips(nil, "", FilterChipsOpts{})) != "" {
		t.Errorf("empty chips rendered markup")
	}

	nav := renderC(t, SectionNav([]SectionLink{{Label: "Team", ID: "settings-team"}, {Label: "Billing", Href: "/billing"}}, SectionNavOpts{}))
	want(t, nav, `<nav class="ui-section-nav" aria-label="On this page">`)
	want(t, nav, `<a href="#settings-team">Team</a>`)
	want(t, nav, `<a href="/billing">Billing</a>`)
}

func TestDisclosureVariantsAndHint(t *testing.T) {
	out := renderC(t, Disclosure("Danger zone", DisclosureOpts{Variant: DisclosureDanger, Hint: "Archive this officer", ID: "dz"}, Text("BODY")))
	want(t, out, `<details class="disclosure disclosure--danger" id="dz">`)
	want(t, out, `<span class="disclosure__label">Danger zone</span> <span class="disclosure__hint">Archive this officer</span>`)

	// Summary only shows while closed; Hint always shows and wins.
	closed := renderC(t, Disclosure("Audit trail", DisclosureOpts{Summary: "created 3 Sep"}, Text("x")))
	want(t, closed, `<span class="disclosure__hint">created 3 Sep</span>`)
	open := renderC(t, Disclosure("Audit trail", DisclosureOpts{Summary: "created 3 Sep", Open: true}, Text("x")))
	if strings.Contains(open, "created 3 Sep") {
		t.Errorf("an open disclosure repeated its collapsed summary: %s", open)
	}
}

func TestCompactStatePanelHasNoHeading(t *testing.T) {
	out := renderC(t, StatePanel("All caught up", StatePanelOpts{Icon: "check", Compact: true}, Text("Nothing assigned.")))
	want(t, out, `class="ui-state ui-state--compact"`)
	want(t, out, `<p class="ui-state__title">All caught up</p>`)
	if strings.Contains(out, "<h2") {
		t.Errorf("a compact state panel must not add a heading: %s", out)
	}
}

func TestTextButtonAndHeaders(t *testing.T) {
	out := renderC(t, Button(ButtonVariantText, "", ButtonOpts{Label: "Mark read", Type: "submit"}))
	want(t, out, `class="ui-button ui-button--text"`)

	header := renderC(t, PageHeader("Acme", PageHeaderOpts{Leading: Text("AVATAR")}))
	want(t, header, `<div class="ui-page-header__leading">AVATAR</div><div class="ui-page-header__copy">`)

	record := renderC(t, RecordHeader(RecordHeaderOpts{Title: "Adri", Eyebrow: "Statutory officer", Subtitle: "Director", Icon: "user", IconTone: ToneInfo, Status: Text("STATUS")}))
	want(t, record, `class="ui-record-header__icon ui-record-header__icon--info"`)
	want(t, record, `<p class="ui-record-header__title">Adri</p>`)
	want(t, record, `<div class="ui-record-header__status">STATUS</div>`)
	if strings.Contains(record, "<h1") || strings.Contains(record, "<h2") || strings.Contains(record, "<h3") {
		t.Errorf("a record header must not add a heading inside the inspector: %s", record)
	}
}

func TestFormatHelpers(t *testing.T) {
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	due := map[string]string{
		"2026-10-06": "today",
		"2026-10-07": "tomorrow",
		"2026-10-12": "in 6 days",
		"2026-09-30": "6 days ago",
	}
	for raw, wantText := range due {
		if got := RelativeDateText(raw, now); got != wantText {
			t.Errorf("RelativeDateText(%q) = %q, want %q", raw, got, wantText)
		}
	}
	if got := RelativeDateText("not a date", now); got != "" {
		t.Errorf("RelativeDateText(garbage) = %q, want empty", got)
	}
	if got := DateText("2026-10-06 01:44:34.745836+00"); got != "6 Oct 2026" {
		t.Errorf("DateText(timestamp) = %q", got)
	}
	if got := Quantity("100000.0000"); got != "100,000" {
		t.Errorf("Quantity = %q", got)
	}
	if got := GroupedAmount("MYR", "37500.00"); got != "MYR 37,500" {
		t.Errorf("GroupedAmount = %q", got)
	}
	if got := Count(1, "deadline", "deadlines"); got != "1 deadline" {
		t.Errorf("Count(1) = %q", got)
	}
}
