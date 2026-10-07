package components

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// A nil body, rows, trigger or panel renders nothing for that slot instead of
// panicking, and the wrapper stays well formed.
func TestNilSlotsRenderWithoutPanicking(t *testing.T) {
	cases := map[string]templ.Component{
		"Alert":                   Alert(ToneInfo, "Note", AlertOpts{}, nil),
		"Callout":                 Callout(ToneInfo, CalloutOpts{Title: "Note"}, nil),
		"DecisionCard":            DecisionCard(DecisionCardOpts{Subject: "Review"}, nil),
		"ConfirmButton":           ConfirmButton("Delete", "Confirm", "Delete item?", "This cannot be undone.", ConfirmOpts{ID: "confirm"}, nil),
		"Drawer":                  Drawer(DrawerOpts{ID: "drawer", Title: "Edit"}, nil),
		"ModalShell":              ModalShell(ModalOpts{ID: "modal", Title: "Edit"}, nil),
		"HTMXPostForm":            HTMXPostForm("form", "/save", "#target", "innerHTML", "", "", nil),
		"ChatThread":              ChatThread("thread", "Conversation", "", nil),
		"SummaryCard":             SummaryCard("Summary", "spark", "", nil),
		"AssistantMessageContent": AssistantMessageContent(nil),
		"LabelledField":           LabelledField("Name", "name", FieldOpts{}, nil),
		"TableShell":              TableShell([]Column{{Key: "name", Label: "Name"}}, TableOpts{}, nil, nil),
		"SettingsSection":         SettingsSection(SettingsSectionOpts{Title: "Alerts"}, nil),
		"Feed":                    Feed(nil),
		"ActionList":              ActionList(ActionListOpts{}, nil),
		"ActionListNav":           ActionList(ActionListOpts{Label: "Documents"}, nil),
		"MenuTrigger":             Menu(MenuOpts{Label: "More"}, nil, Text("PANEL")),
		"MenuPanel":               Menu(MenuOpts{Label: "More"}, Text("TRIGGER"), nil),
		"Disclosure":              Disclosure("Audit", DisclosureOpts{}, nil),
		"ChoiceGroup":             ChoiceGroup("Alerts", ChoiceGroupOpts{}, nil),
		"Inspector":               Inspector(InspectorOpts{ID: "inspector", Open: true, CloseHref: "/x"}, nil),
		"AsyncRegionIdle":         AsyncRegion(AsyncRegionOpts{}, nil),
		"AsyncRegionStale":        AsyncRegion(AsyncRegionOpts{State: "stale", Message: "old"}, nil),
		"SelectionToolbar":        SelectionToolbar(SelectionToolbarOpts{}, nil),
		"WorkspaceSwitcher":       WorkspaceSwitcher(WorkspaceSwitcherOpts{Name: "Workspace"}, nil),
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if out := renderC(t, c); out == "" {
				t.Errorf("%s rendered nothing at all", name)
			}
		})
	}

	want(t, renderC(t, Feed(nil)), `<ul class="ui-feed"></ul>`)
	want(t, renderC(t, Disclosure("Audit", DisclosureOpts{}, nil)), `<summary>`)
	want(t, renderC(t, Menu(MenuOpts{}, nil, Text("PANEL"))), `<div class="ui-menu__panel">PANEL</div>`)
}

// Attrs["class"] joins the computed class list; no element carries two class
// attributes, and the rest of Attrs still spreads.
func TestAttrsClassMergesIntoComputedClass(t *testing.T) {
	attrs := templ.Attributes{"class": "host-hook", "data-x": "1"}
	cases := map[string]struct {
		c    templ.Component
		want string
	}{
		"Card":          {Card(CardOpts{Attrs: attrs}, Text("b")), `class="ui-card host-hook"`},
		"Callout":       {Callout(ToneWarning, CalloutOpts{Title: "T", Attrs: attrs}, nil), `class="ui-callout ui-callout--warning host-hook"`},
		"DecisionCard":  {DecisionCard(DecisionCardOpts{Subject: "S", Attrs: attrs}, nil), `class="ui-decision host-hook"`},
		"ActionRowLink": {ActionRow(ActionRowOpts{Title: "T", Href: "/x", Attrs: attrs}), `class="ui-row ui-row--link host-hook"`},
		"ActionRowDiv":  {ActionRow(ActionRowOpts{Title: "T", Attrs: attrs}), `class="ui-row host-hook"`},
		"Disclosure":    {Disclosure("L", DisclosureOpts{Attrs: attrs}, Text("b")), `class="disclosure host-hook"`},
		"ButtonLink":    {Button(ButtonVariantPrimary, "", ButtonOpts{Label: "Go", Href: "/x", Attrs: attrs}), `host-hook"`},
		"ButtonButton":  {Button(ButtonVariantPrimary, "", ButtonOpts{Label: "Go", Attrs: attrs}), `host-hook"`},
		"FieldInput":    {FieldInput(FieldInputOpts{Name: "n", Attrs: attrs}), `host-hook"`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			out := renderC(t, tc.c)
			want(t, out, tc.want)
			want(t, out, `data-x="1"`)
			// Only the page-level element count matters: the first element's
			// opening tag is the one that took Attrs.
			opening := out[:strings.Index(out, ">")+1]
			if n := strings.Count(opening, " class="); n != 1 {
				t.Errorf("%s opening tag has %d class attributes: %s", name, n, opening)
			}
		})
	}

	if got := mergeClass("a", nil); got != "a" {
		t.Errorf("mergeClass with nil attrs = %q", got)
	}
	in := templ.Attributes{"class": "x", "id": "y"}
	_ = attrsWithoutClass(in)
	if _, ok := in["class"]; !ok {
		t.Error("attrsWithoutClass mutated its input")
	}
}
