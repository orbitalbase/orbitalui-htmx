package components

import (
	"strings"
	"testing"
)

func TestPageHeaderRendersSlots(t *testing.T) {
	out := renderC(t, PageHeader("Officers", PageHeaderOpts{
		Eyebrow:     "Acme Sdn Bhd",
		Description: "People on the statutory register.",
		Actions:     Text("ACTIONS"),
		Compact:     true,
	}))
	want(t, out, `class="ui-page-header ui-page-header--compact"`)
	want(t, out, `<p class="ui-page-header__eyebrow">Acme Sdn Bhd</p>`)
	want(t, out, `<h1 class="ui-page-header__title" tabindex="-1">Officers</h1>`)
	want(t, out, `<div class="ui-page-header__actions">ACTIONS</div>`)
	if strings.Contains(out, "ui-page-header__tabs") {
		t.Errorf("empty tabs slot rendered: %s", out)
	}
}

func TestCardHeaderAndToneAreOptional(t *testing.T) {
	out := renderC(t, Card(CardOpts{ID: "docs", Title: "Recent documents", Description: "Latest six.", Flush: true, Footer: Text("FOOT")}, Text("BODY")))
	want(t, out, `class="ui-card ui-card--flush"`)
	want(t, out, `aria-labelledby="docs-title"`)
	want(t, out, `<h2 class="ui-card__title" id="docs-title">Recent documents</h2>`)
	want(t, out, `<footer class="ui-card__footer">FOOT</footer>`)

	bare := renderC(t, Card(CardOpts{Tone: CardToneDanger}, Text("BODY")))
	want(t, bare, `class="ui-card ui-card--danger"`)
	if strings.Contains(bare, "ui-card__header") {
		t.Errorf("title-less card rendered a header: %s", bare)
	}

	nested := renderC(t, Card(CardOpts{Title: "Nested", HeadingLevel: 3}, nil))
	want(t, nested, `<h3 class="ui-card__title">Nested</h3>`)
}

func TestSettingsSectionLabelsContent(t *testing.T) {
	out := renderC(t, SettingsSection(SettingsSectionOpts{ID: "alerts", Title: "Email alerts", Description: "What reaches your inbox."}, Text("FIELDS")))
	want(t, out, `aria-labelledby="alerts-title"`)
	want(t, out, `id="alerts-title">Email alerts</h2>`)
	want(t, out, `<div class="ui-settings-section__content">FIELDS</div>`)
}

func TestDescriptionListShowsPlaceholderForMissingValues(t *testing.T) {
	out := renderC(t, DescriptionList([]DescriptionItem{
		{Term: "Jurisdiction", Value: "Malaysia"},
		{Term: "Registration", Wide: true},
		{Term: "Status", Content: Text("BADGE")},
	}, DescriptionListOpts{Layout: DescriptionGrid}))
	want(t, out, `class="ui-dl ui-dl--grid"`)
	want(t, out, `<dt>Jurisdiction</dt><dd><span class="ui-dl__value">Malaysia</span>`)
	want(t, out, `class="ui-dl__row ui-dl__row--wide"`)
	want(t, out, `<span class="ui-dl__empty">Not set</span>`)
	want(t, out, `<dd>BADGE</dd>`)
}

func TestInitials(t *testing.T) {
	cases := map[string]string{
		"Adri":                "A",
		"james tan":           "JT",
		"Mary Ann de la Cruz": "MC",
		"  ":                  "?",
		"cosec@hotmail.com":   "CC",
		"ünïcode name":        "ÜN",
	}
	for in, wantOut := range cases {
		if got := Initials(in); got != wantOut {
			t.Errorf("Initials(%q) = %q, want %q", in, got, wantOut)
		}
	}
}

func TestAvatarIsStableAndDecorativeByDefault(t *testing.T) {
	a := renderC(t, Avatar("Adri Tan", AvatarOpts{Size: AvatarSM}))
	b := renderC(t, Avatar("Adri Tan", AvatarOpts{Size: AvatarSM}))
	if a != b {
		t.Fatalf("avatar hue is not stable:\n%s\n%s", a, b)
	}
	want(t, a, `aria-hidden="true"`)
	want(t, a, `ui-avatar--sm`)
	want(t, a, `<span class="ui-avatar__initials">AT</span>`)

	labelled := renderC(t, Avatar("Acme", AvatarOpts{Square: true, Tone: "brand", Label: "Acme"}))
	want(t, labelled, `class="ui-avatar ui-avatar--square ui-avatar--brand"`)
	want(t, labelled, `role="img" aria-label="Acme"`)
}

func TestProgressBarClampsValue(t *testing.T) {
	out := renderC(t, ProgressBar(140, ProgressOpts{Label: "Profile", ShowValue: true}))
	want(t, out, `aria-valuenow="100"`)
	want(t, out, `>100%</span>`)
	want(t, out, `class="ui-progress__fill ui-progress__fill--100"`)
	if strings.Contains(out, "style=") {
		t.Errorf("progress bar must not need an inline style: %s", out)
	}
	want(t, renderC(t, ProgressBar(11, ProgressOpts{})), `ui-progress__fill--10`)
}

func TestChecklistLinksOnlyOpenSteps(t *testing.T) {
	out := renderC(t, Checklist([]ChecklistItem{
		{Title: "Add officers", Done: true, Href: "/officers"},
		{Title: "Complete details", Href: "/profile", ActionLabel: "Continue"},
	}))
	want(t, out, `ui-checklist__item ui-checklist__item--done`)
	want(t, out, `<span class="sr-only">(complete)</span>`)
	want(t, out, `href="/profile"`)
	if strings.Contains(out, `href="/officers"`) {
		t.Errorf("completed step is still linked: %s", out)
	}
}

func TestStepsMarkCurrentStep(t *testing.T) {
	out := renderC(t, Steps([]Step{
		{Title: "Review", State: StepDone},
		{Title: "Approve", Meta: "Directors", State: StepCurrent},
		{Title: "File"},
	}))
	want(t, out, `class="ui-steps__item ui-steps__item--current" aria-current="step"`)
	want(t, out, `ui-steps__item--upcoming`)
	want(t, out, `Current step. </span><span>Directors</span>`)
}

func TestFeedItemUnread(t *testing.T) {
	out := renderC(t, Feed(FeedItem(FeedItemOpts{Title: "Firm connected", Body: "Accepted.", Href: "/n/1", Unread: true, Icon: "bell", Tone: ToneInfo})))
	want(t, out, `<ul class="ui-feed">`)
	want(t, out, `class="ui-feed__item ui-feed__item--unread"`)
	want(t, out, `class="ui-feed__icon ui-feed__icon--info"`)
	want(t, out, `<a class="ui-feed__title" href="/n/1">Firm connected</a>`)
}

func TestSidebarNavOpensActiveSection(t *testing.T) {
	out := renderC(t, SidebarNav([]NavSection{
		{Links: []NavLink{{Label: "Overview", Href: "/", Icon: "dashboard", Key: "home"}}},
		{Label: "Company", ID: "company", Collapsible: true, Links: []NavLink{{Label: "Officers", Href: "/officers", Active: true, Badge: "2"}}},
		{Label: "Finance", ID: "finance", Collapsible: true, Links: []NavLink{{Label: "Fees", Href: "/fees"}}},
		{Label: "Static", Links: []NavLink{{Label: "Guide", Href: "/help"}}},
	}, SidebarNavOpts{Label: "Workspace"}))
	want(t, out, `aria-label="Workspace"`)
	want(t, out, `<details class="ui-nav__section" open data-nav-section="company">`)
	want(t, out, `<details class="ui-nav__section" data-nav-section="finance">`)
	want(t, out, `class="ui-nav__link is-active" href="/officers" title="Officers" aria-current="page"`)
	want(t, out, `<span class="ui-nav__badge">2</span>`)
	want(t, out, `data-nav-key="home"`)
	want(t, out, `<p class="ui-nav__heading">`)
}

func TestBreadcrumbsMarkLastCrumbCurrent(t *testing.T) {
	out := renderC(t, Breadcrumbs([]Crumb{{Label: "Companies", Href: "/companies"}, {Label: "Officers", Href: "/ignored"}}))
	want(t, out, `<a href="/companies">Companies</a>`)
	want(t, out, `<span aria-current="page">Officers</span>`)
	if strings.Contains(out, "/ignored") {
		t.Errorf("current crumb rendered as a link: %s", out)
	}
	if renderC(t, Breadcrumbs(nil)) != "" {
		t.Errorf("empty breadcrumbs rendered markup")
	}
}

func TestMenuAndSwitcher(t *testing.T) {
	out := renderC(t, Menu(MenuOpts{Label: "Account", Align: "end", Placement: "top"}, Text("TRIGGER"), MenuLink("Settings", "/settings", MenuItemOpts{Icon: "settings", Selected: true})))
	want(t, out, `<details class="ui-menu ui-menu--end ui-menu--top" data-ui-menu>`)
	want(t, out, `<summary class="ui-menu__trigger" aria-label="Account">TRIGGER</summary>`)
	want(t, out, `class="ui-menu__item is-selected" href="/settings" aria-current="true"`)

	sw := renderC(t, WorkspaceSwitcher(WorkspaceSwitcherOpts{Name: "Acme", Subtitle: "Malaysia · Manager", Kind: "Company", StatusTone: ToneWarning}, MenuHeader("Acme", "Pending incorporation")))
	want(t, sw, `class="ui-menu ui-menu--wide ui-switcher"`)
	want(t, sw, `aria-label="Company: Acme Malaysia · Manager. Open workspace menu"`)
	want(t, sw, `ui-switcher__status--warning`)
	want(t, sw, `<span class="ui-switcher__subtitle">Malaysia · Manager</span>`)
	want(t, sw, `<span class="ui-menu__header-subtitle">Pending incorporation</span>`)
}
