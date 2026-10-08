package main

import (
	"fmt"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func init() {
	register(section{ID: "buttons", Title: "Buttons", Note: "Variants x sizes, icon placement, icon-only, block, loading, disabled, link buttons.", Body: buttonsBody()})
	register(section{ID: "badges", Title: "Badges and status", Note: "StatusBadge (pill) and Badge (rectangle) across every tone and option.", Body: badgesBody()})
	register(section{ID: "cards", Title: "Cards", Note: "Frame variants, header icon and subtitle, edges, interactive and link cards, sections.", Body: cardsBody()})
	register(section{ID: "kbd", Title: "Kbd", Body: group("Key caps",
		cell("single", c.Kbd("Esc")), cell("combination", c.Kbd("⌘", "K")), cell("group of three", c.KbdGroup("Ctrl", "Shift", "P")))})
}

type variantRow struct {
	name string
	v    c.ButtonVariant
}

var buttonVariants = []variantRow{
	{"primary", c.ButtonVariantPrimary}, {"secondary", c.ButtonVariantSecondary}, {"quiet", c.ButtonVariantQuiet},
	{"danger", c.ButtonVariantDanger}, {"ai", c.ButtonVariantAI}, {"text", c.ButtonVariantText}, {"text-danger", c.ButtonVariantTextDanger},
}

func buttonsBody() templ.Component {
	var parts []templ.Component
	for _, size := range []struct {
		name string
		s    c.ButtonSize
	}{{"sm (30)", c.ButtonSizeSM}, {"md (36)", ""}, {"lg (44)", c.ButtonSizeLG}} {
		var cells []templ.Component
		for _, v := range buttonVariants {
			cells = append(cells, c.Button(v.v, size.s, c.ButtonOpts{Label: v.name, Icon: "plus"}))
		}
		parts = append(parts, group("Size "+size.name, cells...))
	}
	var disabled, loading, icons, iconOnly []templ.Component
	for _, v := range buttonVariants {
		disabled = append(disabled, c.Button(v.v, "", c.ButtonOpts{Label: v.name, Disabled: true}))
		loading = append(loading, c.Button(v.v, "", c.ButtonOpts{Label: "Saving", Loading: true}))
		iconOnly = append(iconOnly, c.Button(v.v, "", c.ButtonOpts{Icon: "plus", Label: "Add " + v.name, IconOnly: true}))
	}
	icons = append(icons,
		c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Next", Icon: "arrow-right", IconRight: true}),
		c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Plain, no icon"}),
		c.Button(c.ButtonVariantAI, "", c.ButtonOpts{Label: "Ask the assistant", Icon: "spark"}),
		c.Button(c.ButtonVariantSecondary, c.ButtonSizeLG, c.ButtonOpts{Label: "Link button", Icon: "external", Href: "/x"}),
		c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "A very long label that should keep a single line and not wrap at all today"}),
	)
	parts = append(parts,
		group("Trailing icon pair and IconRight order",
			c.Button(c.ButtonVariantSecondary, c.ButtonSizeSM, c.ButtonOpts{Label: "New", Icon: "plus", TrailingIcon: "chevron-down"}),
			c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "New", Icon: "plus", TrailingIcon: "chevron-down"}),
			c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Next (IconRight: icon on the right)", Icon: "arrow-right", IconRight: true}),
			c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Only trailing", TrailingIcon: "chevron-down"})),
		group("Disabled", disabled...), group("Loading (aria-busy)", loading...),
		group("Icon placement and link", icons...), group("Icon-only (label becomes aria-label)", iconOnly...),
		group("Icon-only sizes", c.Button(c.ButtonVariantSecondary, c.ButtonSizeSM, c.ButtonOpts{Icon: "plus", Label: "Add", IconOnly: true}),
			c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Icon: "plus", Label: "Add", IconOnly: true}),
			c.Button(c.ButtonVariantSecondary, c.ButtonSizeLG, c.ButtonOpts{Icon: "plus", Label: "Add", IconOnly: true}),
			raw(`<button type="button" class="ui-icon-button" aria-label="Legacy icon button">`), c.Icon("close"), raw(`</button>`)),
		groupStack("Block (full width)", c.Button(c.ButtonVariantPrimary, c.ButtonSizeLG, c.ButtonOpts{Label: "Continue", Block: true}),
			c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Cancel", Block: true})),
		group("Count pill", c.CountBadge(3), c.CountBadge(128)),
	)
	return seq(parts...)
}

func badgesBody() templ.Component {
	tones := []c.Tone{c.ToneNeutral, c.ToneBrand, c.ToneAccent, c.ToneInfo, c.ToneSuccess, c.ToneWarning, c.ToneDanger, c.ToneAI}
	var status, statusNoDot, statusSubtle, badge, badgeDot, badgeHollow, badgeMuted, badgeMono, badgeIcon []templ.Component
	no := false
	for _, t := range tones {
		n := string(t)
		status = append(status, c.StatusBadge(n, t, c.StatusBadgeOpts{}))
		statusNoDot = append(statusNoDot, c.StatusBadge(n, t, c.StatusBadgeOpts{Dot: &no}))
		badge = append(badge, c.Badge(t, n, c.BadgeOpts{}))
		badgeDot = append(badgeDot, c.Badge(t, n, c.BadgeOpts{Dot: true}))
		badgeHollow = append(badgeHollow, c.Badge(t, n, c.BadgeOpts{Hollow: true}))
		badgeMuted = append(badgeMuted, c.Badge(t, n, c.BadgeOpts{Muted: true, Dot: true}))
		badgeMono = append(badgeMono, c.Badge(t, fmt.Sprintf("%.3s-0042", n), c.BadgeOpts{Mono: true}))
		badgeIcon = append(badgeIcon, c.Badge(t, n, c.BadgeOpts{Icon: "spark"}))
	}
	statusSubtle = append(statusSubtle, c.StatusBadge("Current", c.ToneInfo, c.StatusBadgeOpts{Subtle: true}))
	return seq(
		group("StatusBadge (pill, dot)", status...), group("StatusBadge, no dot", statusNoDot...), group("StatusBadge subtle", statusSubtle...),
		group("Badge", badge...), group("Badge with dot", badgeDot...), group("Badge hollow dot", badgeHollow...),
		group("Badge muted", badgeMuted...), group("Badge mono", badgeMono...), group("Badge with icon", badgeIcon...),
		group("Badge dot tint (--oui-badge-dot)", wrap(`<span style="--oui-badge-dot: var(--oui-accent)">`, `</span>`, c.Badge(c.ToneBrand, "Dot tinted apart from the text", c.BadgeOpts{Dot: true}))),
		group("In a sentence", raw(`<span>Posted by `), c.Badge(c.ToneBrand, "Talent", c.BadgeOpts{Dot: true}), raw(` for `), c.Badge(c.ToneAccent, "Hiring", c.BadgeOpts{Dot: true}), raw(`</span>`)),
	)
}

func lorem(n int) templ.Component {
	s := "A short paragraph of card body copy that wraps onto a second line to show the text rhythm inside the frame."
	if n > 1 {
		s += " A second sentence keeps the body honest about line length and spacing in this card."
	}
	return c.Text(s)
}

func cardsBody() templ.Component {
	act := c.Button(c.ButtonVariantSecondary, c.ButtonSizeSM, c.ButtonOpts{Label: "Edit"})
	return seq(
		groupGrid("Frame",
			cell("default", c.Card(c.CardOpts{Title: "Default", Description: "Ring plus a hairline of elevation."}, lorem(1))),
			cell("subtle", c.Card(c.CardOpts{Title: "Subtle", Tone: c.CardToneSubtle}, lorem(1))),
			cell("brand", c.Card(c.CardOpts{Title: "Brand", Tone: c.CardToneBrand}, lorem(1))),
			cell("danger", c.Card(c.CardOpts{Title: "Danger zone", Tone: c.CardToneDanger, Description: "Irreversible actions."}, lorem(1))),
			cell("headerless", c.Card(c.CardOpts{}, lorem(2))),
		),
		groupGrid("Header",
			cell("icon + subtitle + actions", c.Card(c.CardOpts{Title: "Profile", Subtitle: "Senior engineer · Kuala Lumpur", Icon: "briefcase", Actions: act}, lorem(1))),
			cell("eyebrow + badge", c.Card(c.CardOpts{Title: "Quarterly review", Eyebrow: "Company action", Badge: c.StatusBadge("Open", c.ToneSuccess, c.StatusBadgeOpts{})}, lorem(1))),
			cell("footer", c.Card(c.CardOpts{Title: "With footer", Footer: seq(c.Text("3 of 12 shown"), c.Button(c.ButtonVariantText, "", c.ButtonOpts{Label: "View all"}))}, lorem(1))),
			cell("long title", c.Card(c.CardOpts{Title: "A very long card title that has to wrap inside the narrow column without breaking the header", Subtitle: "Subtitle", Actions: act}, lorem(1))),
		),
		groupGrid("Padded and Bare",
			cell("Padded (20px gutter on the card)", c.Card(c.CardOpts{Title: "Padded", Subtitle: "Header, body and footer share one inset", Padded: true, Actions: act, Footer: c.Text("Footer")}, lorem(1))),
			cell("Bare + Padded (no body wrapper, host class lays it out)", c.Card(c.CardOpts{Bare: true, Padded: true, Class: "g-row"}, seq(c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 40}), c.Text("A plain padded surface, laid out by its own class.")))),
			cell("subtitle + actions (--oui-card-header-align: center)", c.Card(c.CardOpts{Title: "Offers", Subtitle: "3 open, 1 expiring today", Actions: act, Class: "g-card-center"}, lorem(1))),
		),
		groupGrid("Standalone CardHeader",
			cell("CardHeader inside Bare + Padded", c.Card(c.CardOpts{Bare: true, Padded: true}, seq(
				c.CardHeader(c.CardHeaderOpts{Icon: "building", Title: "Acme Ltd", Subtitle: "Company", Actions: act}),
				c.Text("The host lays out the rest of the card."),
			))),
			cell("Bare card with a host flex-row class stays a row", c.Card(c.CardOpts{Bare: true, Padded: true, Class: "g-row"}, seq(c.Text("One"), c.Text("Two"), c.Text("Three")))),
		),
		groupGrid("Edge tone (2px left edge)",
			cell("brand", c.Card(c.CardOpts{Title: "Talent", Edge: c.ToneBrand}, lorem(1))),
			cell("accent", c.Card(c.CardOpts{Title: "Hiring", Edge: c.ToneAccent}, lorem(1))),
			cell("success", c.Card(c.CardOpts{Title: "Complete", Edge: c.ToneSuccess}, lorem(1))),
			cell("warning", c.Card(c.CardOpts{Title: "Needs attention", Edge: c.ToneWarning}, lorem(1))),
			cell("danger", c.Card(c.CardOpts{Title: "Blocked", Edge: c.ToneDanger}, lorem(1))),
			cell("ai", c.Card(c.CardOpts{Title: "Suggested", Edge: c.ToneAI}, lorem(1))),
		),
		groupGrid("Interactive and link",
			cell("interactive (hover lifts)", c.Card(c.CardOpts{Title: "Interactive", Interactive: true, Edge: c.ToneBrand}, lorem(1))),
			cell("href: stretched title link", c.Card(c.CardOpts{ID: "g-link", Title: "Acme Sdn Bhd", Subtitle: "Company", Href: "#acme", Actions: act}, lorem(1))),
			cell("href without title", c.Card(c.CardOpts{Href: "#x", LinkLabel: "Open record"}, lorem(1))),
		),
		groupGrid("Flush with sections",
			cell("sections", c.Card(c.CardOpts{Title: "Candidate", Flush: true, Icon: "parties"},
				seq(c.CardSection(c.CardSectionOpts{}, lorem(1)), c.CardSection(c.CardSectionOpts{Divided: true}, lorem(1)), c.CardSection(c.CardSectionOpts{Divided: true}, c.Text("Last section"))))),
			cell("flush list", c.Card(c.CardOpts{Title: "Flush body", Flush: true}, raw(`<div style="padding:16px">Edge-to-edge content (a table or list runs here)</div>`))),
		),
	)
}
