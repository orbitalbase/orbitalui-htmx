package main

import (
	"net/http"

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
		raw(`</div></main></body></html>`),
	)
	render(w, r, shellHead(w, r, "Kit parity"), body, dl, rest)
}
