package main

import (
	"net/http"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// kitPage serves the kit-parity components (buttons with icon placement, avatars,
// orb, cards, data lists, ring, chips, segmented and tab navs) so
// scripts/browser-check.mjs can read their computed layout and styles.
func kitPage(w http.ResponseWriter, r *http.Request) {
	body := seq(
		raw(`<main id="kit" style="padding:24px;display:grid;gap:24px;max-width:720px">`),
		raw(`<div id="k-buttons" style="display:flex;gap:12px;flex-wrap:wrap">`),
		c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Next", Icon: "arrow-right", IconRight: true}),
		c.Button("", "", c.ButtonOpts{Label: "New", Icon: "plus", TrailingIcon: "chevron-down"}),
		c.Button("", "", c.ButtonOpts{Label: "Add", Icon: "plus"}),
		raw(`</div>`),
		raw(`<div id="k-avatars" style="display:flex;gap:12px;align-items:center">`),
		c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 16}), c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 20}), c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 24}),
		c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 32}), c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 40}), c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 56}), c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 72}),
		raw(`</div>`),
		raw(`<div id="k-orb">`), c.Orb(c.OrbOpts{Size: 64}), raw(`</div>`),
		raw(`<div id="k-cards" style="display:grid;gap:12px">`),
		c.Card(c.CardOpts{ID: "k-card-padded", Title: "Padded", Subtitle: "sub", Padded: true, Actions: c.Button("", c.ButtonSizeSM, c.ButtonOpts{Label: "Edit"})}, raw(`<p>Body</p>`)),
		c.Card(c.CardOpts{ID: "k-card-bare", Bare: true, Padded: true, Class: "host-grid"}, raw(`<span class="host-a">a</span><span class="host-b">b</span>`)),
		raw(`</div>`),
		raw(`<div id="k-dl">`),
	)
	dl := with(c.DataList(c.DescriptionListOpts{Columns: 2}),
		with(c.DataItem("Role"), raw(`Engineer`)),
		with(c.DataItem("Notes"), raw(`<span>Line one</span><span>Line two</span><span>Line three</span><span>Line four</span>`)),
		with(c.DataItemWide("Status"), c.Badge(c.ToneSuccess, "Active", c.BadgeOpts{Dot: true})),
	)
	rest := seq(
		raw(`</div>`),
		raw(`<div id="k-ring" style="display:flex;gap:12px;align-items:center">`),
		c.RingMeter(c.RingMeterOpts{Value: 82, Px: 72, Tone: c.ToneAI}), c.RingMeter(c.RingMeterOpts{Value: 40, Px: 28}), c.RingMeter(c.RingMeterOpts{Value: 40, Size: 64}),
		raw(`</div>`),
		c.ChipGroup([]c.ChipOption{{Value: "a", Label: "Alpha", Selected: true}, {Value: "b", Label: "Beta", Suggested: true}}, c.ChipGroupOpts{Name: "k-chip", Legend: "Mode", Required: true, Help: "Pick one.", Error: "Pick one to continue."}),
		raw(`<div id="k-seg">`),
		c.SegmentedNav([]c.SegItem{{Label: "Overview", Href: "#a"}, {Label: "Roles", Href: "#b", Active: true, Count: "4"}, {Label: "Notes", Href: "#c"}}, c.SegNavOpts{AriaLabel: "Company"}),
		raw(`</div><div id="k-tabs">`),
		c.TabNav([]c.TabItem{{Label: "Overview", Href: "#a", Active: true, Count: "12"}, {Label: "Roles", Href: "#b", Count: "4"}}, c.TabNavOpts{AriaLabel: "Sections"}),
		raw(`</div><div id="k-icons" style="display:flex;gap:12px;align-items:center">`),
		c.IconSized("plus", 12), c.IconSized("plus", 16), c.IconSized("plus", 24),
		raw(`</div><div id="k-field" style="display:grid;gap:12px">`),
		c.LabelledField("Name", "k-name", c.FieldOpts{Suggested: true, SuggestedTitle: "Suggested by your agent"}, c.FieldInput(c.FieldInputOpts{ID: "k-name", Value: "Ada"})),
		c.FieldInput(c.FieldInputOpts{ID: "k-plain", Value: "Plain"}),
		raw(`</div>`),
	)
	render(w, r, shellHead(w, r, "Kit parity"), body, dl, rest, kitFinish(), raw(`</main></body></html>`))
}

// kitFinish is the markup of the final kit-parity options (field pad token,
// textarea tokens, suggested shell, chips, bare flex card, data list spans,
// badge dot, numeric field, icon shrink, state panel tag, select placeholder,
// field help/error ids, standalone card header, inset ring).
func kitFinish() templ.Component {
	finishDL := with(c.DataList(c.DescriptionListOpts{Columns: 2}),
		with(c.DataItem("A"), raw(`one`)),
		with(c.DataItem("B"), raw(`two`)),
		with(c.DataItemWith("Wide", c.DataItemOpts{Wide: true, Field: "wide"}), raw(`spans`)),
		with(c.DataItemWith("Plain", c.DataItemOpts{Field: "plain"}), raw(`x`)),
	)
	barecard := c.Card(c.CardOpts{ID: "kf-bcard", Bare: true, Padded: true, Class: "kf-flex"}, seq(
		c.CardHeader(c.CardHeaderOpts{Icon: "search", Title: "Standalone", Subtitle: "header", Actions: c.Button("", c.ButtonSizeSM, c.ButtonOpts{Label: "Edit"})}),
		raw(`<span>a</span><span>b</span>`),
	))
	return seq(
		raw(`<style>.kf-flex { display: flex; }</style><div id="k-finish" style="display:grid;gap:12px">`),
		c.LabelledField("Suggested icon", "kf-sug", c.FieldOpts{Suggested: true}, c.FieldInput(c.FieldInputOpts{ID: "kf-sug", Icon: "search", Value: "x"})),
		c.LabelledField("Plain affix", "kf-pad", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "kf-pad", Prefix: "$", Value: "1"})),
		c.FieldInput(c.FieldInputOpts{ID: "kf-bare", Value: "bare"}),
		c.FieldInput(c.FieldInputOpts{ID: "kf-num", Value: "1,204.50", Numeric: true}),
		c.FieldInput(c.FieldInputOpts{ID: "kf-numtype", Type: "number", Value: "5"}),
		c.FieldTextarea(c.FieldTextareaOpts{ID: "kf-ta", Value: "x"}),
		c.FieldTextarea(c.FieldTextareaOpts{ID: "kf-ta-grow", Value: "x", Autogrow: true}),
		c.LabelledField("Required select", "kf-sel", c.FieldOpts{Help: "Pick one.", Error: "Required.", ErrorAlert: true}, c.FieldSelect(c.FieldSelectOpts{ID: "kf-sel", Options: []c.SelectOption{{Value: "a", Label: "A"}}, Placeholder: "Choose", PlaceholderDisabled: true, Required: true, AriaDescribedBy: c.FieldDescribedBy("kf-sel", c.FieldOpts{Help: "Pick one.", Error: "Required."})})),
		c.LabelledField("Calm", "kf-calm", c.FieldOpts{Error: "Bad."}, c.FieldInput(c.FieldInputOpts{ID: "kf-calm"})),
		c.ChipGroup([]c.ChipOption{{Value: "a", Label: "Alpha", Selected: true, Suggested: true}, {Value: "b", Label: "Beta", Suggested: true}}, c.ChipGroupOpts{Name: "kf-chip", Legend: "Finish", SuggestedTitle: "Suggested by your agent"}),
		raw(`<div id="kf-dl">`), finishDL, raw(`</div>`),
		barecard,
		raw(`<div id="kf-badge">`), c.Badge(c.ToneBrand, "Dot", c.BadgeOpts{Dot: true}), raw(`</div>`),
		raw(`<div id="kf-iconrow" style="display:flex;width:8px">`), c.Icon("plus"), raw(`</div>`),
		raw(`<div id="kf-state">`), c.StatePanel("Nothing here", c.StatePanelOpts{TitleTag: "h3"}, nil), raw(`</div>`),
		raw(`</div>`),
	)
}
