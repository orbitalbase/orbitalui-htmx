package main

import (
	"context"
	"io"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// withChildren renders comp with child as its templ children block.
func withChildren(comp, child templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return comp.Render(templ.WithChildren(ctx, child), w)
	})
}

func dispText(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func init() {
	register(section{ID: "avatars", Title: "Avatar and entity chip", Note: "16..72 scale, person circle, organisation square, anonymous silhouette, tones, presence.", Body: seq(
		group("Scale (person)",
			cell("16", c.Avatar("Adri Tan", c.AvatarOpts{Px: 16})),
			cell("20", c.Avatar("Adri Tan", c.AvatarOpts{Px: 20})),
			cell("24", c.Avatar("Adri Tan", c.AvatarOpts{Px: 24})),
			cell("32", c.Avatar("Adri Tan", c.AvatarOpts{Px: 32})),
			cell("40", c.Avatar("Adri Tan", c.AvatarOpts{Px: 40})),
			cell("56", c.Avatar("Adri Tan", c.AvatarOpts{Px: 56})),
			cell("72", c.Avatar("Adri Tan", c.AvatarOpts{Px: 72})),
		),
		group("Legacy size names",
			cell("xs", c.Avatar("Adri Tan", c.AvatarOpts{Size: c.AvatarXS})),
			cell("sm", c.Avatar("Adri Tan", c.AvatarOpts{Size: c.AvatarSM})),
			cell("md (default)", c.Avatar("Adri Tan", c.AvatarOpts{})),
			cell("lg", c.Avatar("Adri Tan", c.AvatarOpts{Size: c.AvatarLG})),
			cell("xl", c.Avatar("Adri Tan", c.AvatarOpts{Size: c.AvatarXL})),
		),
		group("Organisation (square)",
			cell("32", c.Avatar("Northwind", c.AvatarOpts{Px: 32, Square: true, Tone: "accent"})),
			cell("40", c.Avatar("Northwind", c.AvatarOpts{Px: 40, Square: true, Tone: "accent"})),
			cell("56", c.Avatar("Northwind", c.AvatarOpts{Px: 56, Square: true, Tone: "brand"})),
			cell("72", c.Avatar("Northwind", c.AvatarOpts{Px: 72, Square: true, Tone: "brand"})),
		),
		group("Anonymous",
			cell("24", c.Avatar("", c.AvatarOpts{Px: 24, Anonymous: true})),
			cell("40", c.Avatar("", c.AvatarOpts{Px: 40, Anonymous: true})),
			cell("72", c.Avatar("", c.AvatarOpts{Px: 72, Anonymous: true})),
		),
		group("Tones",
			cell("brand", c.Avatar("Brand Person", c.AvatarOpts{Tone: "brand"})),
			cell("accent", c.Avatar("Accent Person", c.AvatarOpts{Tone: "accent"})),
			cell("gray", c.Avatar("Gray Person", c.AvatarOpts{Tone: "gray"})),
			cell("derived 0..7", seq(raw(`<div class="g-row">`),
				c.Avatar("Aaron", c.AvatarOpts{}), c.Avatar("Bea", c.AvatarOpts{}), c.Avatar("Chandra", c.AvatarOpts{}),
				c.Avatar("Dmitri", c.AvatarOpts{}), c.Avatar("Esi", c.AvatarOpts{}), c.Avatar("Farid", c.AvatarOpts{}),
				c.Avatar("Gwen", c.AvatarOpts{}), c.Avatar("Hiro", c.AvatarOpts{}), raw(`</div>`),
			)),
		),
		group("Presence dot",
			cell("online", c.Avatar("Adri Tan", c.AvatarOpts{Px: 40, Status: "online"})),
			cell("away", c.Avatar("Adri Tan", c.AvatarOpts{Px: 40, Status: "away"})),
			cell("none-set", c.Avatar("Adri Tan", c.AvatarOpts{Px: 40, Status: "offline"})),
		),
		group("Entity chip",
			cell("plain", c.EntityChip(c.EntityChipOpts{Name: "Adri Tan"})),
			cell("link", c.EntityChip(c.EntityChipOpts{Name: "Adri Tan", Href: "/people/42"})),
			cell("organisation", c.EntityChip(c.EntityChipOpts{Name: "Northwind Traders", Organisation: true})),
			cell("anonymous", c.EntityChip(c.EntityChipOpts{Name: "Candidate", Anonymous: true})),
		),
	)})

	register(section{ID: "measures", Title: "Meter, ring, progress", Note: "Meter tones incl. the AI gradient; score rings (arc draws in); progress bars.", Body: seq(
		groupStack("Meter tones (72%)",
			cell("default (action)", c.Meter(c.MeterOpts{Value: 72, Label: "Default"})),
			cell("brand", c.Meter(c.MeterOpts{Value: 72, Tone: "brand", Label: "Brand"})),
			cell("accent", c.Meter(c.MeterOpts{Value: 72, Tone: "accent", Label: "Accent"})),
			cell("info", c.Meter(c.MeterOpts{Value: 72, Tone: c.ToneInfo, Label: "Info"})),
			cell("success", c.Meter(c.MeterOpts{Value: 72, Tone: c.ToneSuccess, Label: "Success"})),
			cell("warning", c.Meter(c.MeterOpts{Value: 72, Tone: c.ToneWarning, Label: "Warning"})),
			cell("danger", c.Meter(c.MeterOpts{Value: 72, Tone: c.ToneDanger, Label: "Danger"})),
			cell("ai", c.Meter(c.MeterOpts{Value: 72, Tone: c.ToneAI, Label: "AI"})),
		),
		groupStack("Meter sizes and edges",
			cell("sm 40%", c.Meter(c.MeterOpts{Value: 40, Size: "sm", Label: "Small"})),
			cell("md 40%", c.Meter(c.MeterOpts{Value: 40, Label: "Medium"})),
			cell("lg 40%", c.Meter(c.MeterOpts{Value: 40, Size: "lg", Label: "Large"})),
			cell("0%", c.Meter(c.MeterOpts{Value: 0, Label: "Empty"})),
			cell("100%", c.Meter(c.MeterOpts{Value: 100, Tone: c.ToneSuccess, Label: "Full"})),
		),
		group("Ring meter sizes",
			cell("32", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 32})),
			cell("40", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 40})),
			cell("48", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 48})),
			cell("56", c.RingMeter(c.RingMeterOpts{Value: 64})),
			cell("64", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 64})),
			cell("80", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 80})),
			cell("96", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 96})),
			cell("120", c.RingMeter(c.RingMeterOpts{Value: 64, Size: 120})),
		),
		group("Ring meter, any pixel size (Px)",
			cell("28", c.RingMeter(c.RingMeterOpts{Value: 64, Px: 28, Tone: c.ToneAI})),
			cell("72", c.RingMeter(c.RingMeterOpts{Value: 82, Px: 72, Tone: c.ToneAI})),
			cell("100", c.RingMeter(c.RingMeterOpts{Value: 40, Px: 100})),
			cell("preset size via Px", c.RingMeter(c.RingMeterOpts{Value: 64, Px: 64}))),
		group("Ring meter tones",
			cell("action", c.RingMeter(c.RingMeterOpts{Value: 78, Size: 64})),
			cell("ai (gradient)", c.RingMeter(c.RingMeterOpts{Value: 78, Size: 64, Tone: c.ToneAI})),
			cell("success", c.RingMeter(c.RingMeterOpts{Value: 92, Size: 64, Tone: c.ToneSuccess})),
			cell("warning", c.RingMeter(c.RingMeterOpts{Value: 45, Size: 64, Tone: c.ToneWarning})),
			cell("danger", c.RingMeter(c.RingMeterOpts{Value: 18, Size: 64, Tone: c.ToneDanger})),
			cell("custom label", c.RingMeter(c.RingMeterOpts{Value: 78, Size: 64, Tone: c.ToneAI, Label: "7/9"})),
			cell("0 and 100", seq(c.RingMeter(c.RingMeterOpts{Value: 0, Size: 48}), c.RingMeter(c.RingMeterOpts{Value: 100, Size: 48, Tone: c.ToneSuccess}))),
		),
		groupStack("Progress bar",
			cell("default 40%, label + value", c.ProgressBar(40, c.ProgressOpts{Label: "Profile", ShowValue: true})),
			cell("success 100%", c.ProgressBar(100, c.ProgressOpts{Label: "Imported", ShowValue: true, Tone: c.ToneSuccess})),
			cell("warning 65%", c.ProgressBar(65, c.ProgressOpts{Label: "Quota", ShowValue: true, Tone: c.ToneWarning})),
			cell("danger 90%", c.ProgressBar(90, c.ProgressOpts{Label: "Disk", ShowValue: true, Tone: c.ToneDanger})),
			cell("info 30%", c.ProgressBar(30, c.ProgressOpts{Label: "Sync", ShowValue: true, Tone: c.ToneInfo})),
			cell("ai 55%", c.ProgressBar(55, c.ProgressOpts{Label: "Drafting", ShowValue: true, Tone: c.ToneAI})),
			cell("accent 25% (no label)", c.ProgressBar(25, c.ProgressOpts{AriaLabel: "Upload", Tone: "accent"})),
		),
	)})

	register(section{ID: "timeline", Title: "Timeline, steps, feed", Note: "Timeline tones and current state; steps in every state; feed with unread, tones, long text.", Body: seq(
		groupGrid("Timeline",
			cell("tones + current", withChildren(c.Timeline(), seq(
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "v4", Meta: "Today", Current: true, Mono: true}), dispText("Raised the salary band after feedback.")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Published", Meta: "Mon", Tone: c.ToneSuccess}), dispText("Visible to matched candidates.")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Match proposed", Meta: "Sun", Tone: "brand", Href: "/matches/1"}), dispText("Strong overlap on skills.")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Agent drafted", Meta: "Sat", Tone: c.ToneAI}), dispText("")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Hiring side responded", Meta: "Fri", Tone: "accent"}), dispText("")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Warning raised", Meta: "Thu", Tone: c.ToneWarning}), dispText("Salary outside the stated range.")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Declined", Meta: "Wed", Tone: c.ToneDanger}), dispText("")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Created with a very long unbroken title value that must wrap safely", Meta: "Jan 2"}), dispText("")),
			))),
			cell("current default tone", withChildren(c.Timeline(), seq(
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Latest", Meta: "Now", Current: true}), dispText("The current one.")),
				withChildren(c.TimelineItem(c.TimelineItemOpts{Title: "Earlier", Meta: "Yesterday"}), dispText("An earlier one.")),
			))),
		),
		groupGrid("Steps",
			cell("done / current / upcoming / blocked", c.Steps([]c.Step{
				{Title: "Create account", Meta: "Adri, Mon", State: c.StepDone},
				{Title: "Verify identity", Meta: "In review", State: c.StepCurrent, Content: dispText("Documents received.")},
				{Title: "Sign agreement", Meta: "Waiting", State: c.StepBlocked},
				{Title: "Go live", Meta: "Not started", State: c.StepUpcoming},
			})),
		),
		groupStack("Feed",
			cell("rows", c.Feed(seq(
				c.FeedItem(c.FeedItemOpts{Title: "Invoice posted", Body: "INV-0042 was posted to the ledger.", Time: "2m", Icon: "check", Tone: c.ToneSuccess, Unread: true}),
				c.FeedItem(c.FeedItemOpts{Title: "Review requested", Body: "Adri asked you to review the draft.", Time: "1h", Icon: "user", Tone: "brand", Href: "/reviews/3"}),
				c.FeedItem(c.FeedItemOpts{Title: "Agent summary", Body: "Three open questions remain.", Time: "3h", Icon: "spark", Tone: c.ToneAI}),
				c.FeedItem(c.FeedItemOpts{Title: "Overdue", Body: "Payment is 12 days late.", Time: "1d", Icon: "alert", Tone: c.ToneDanger}),
				c.FeedItem(c.FeedItemOpts{Title: "Heads up", Body: "Quota at 90%.", Time: "2d", Icon: "alert", Tone: c.ToneWarning}),
				c.FeedItem(c.FeedItemOpts{Title: "Accent event", Body: "Hiring side opened the listing.", Time: "3d", Icon: "briefcase", Tone: "accent"}),
				c.FeedItem(c.FeedItemOpts{Title: "Long unbroken", Body: "https://example.com/a/really/long/unbroken/path/that/should/wrap/instead/of/overflowing/the/container/and/pushing/the/layout/sideways?q=1", Time: "4d", Icon: "link"}),
			))),
		),
	)})

	register(section{ID: "data", Title: "Description list, KPI, checklist, orb", Note: "DataList columns 1-3 with empty/mono/wide values, KPI tiles, checklist normal + compact, orb sizes and states, state panels.", Body: seq(
		groupStack("Description list: inline",
			c.DescriptionList([]c.DescriptionItem{
				{Term: "Jurisdiction", Value: "Malaysia"},
				{Term: "Status", Content: c.StatusBadge("Active", c.ToneSuccess, c.StatusBadgeOpts{})},
				{Term: "Registration", Value: "202401012345", Mono: true, Hint: "Issued 12 Jan 2024"},
				{Term: "Notes"},
			}, c.DescriptionListOpts{}),
		),
		groupStack("DataList (composable): rich values, tall row beside short rows",
			cell("DataList + DataItem", with(c.DataList(c.DescriptionListOpts{Columns: 2}),
				with(c.DataItem("Role"), c.Text("Senior engineer")),
				with(c.DataItem("Notes"), raw(`<span>First line</span><span>Second line</span><span>Third line</span><span>Fourth line</span>`)),
				with(c.DataItem("Owner"), seq(c.Avatar("Ada Lovelace", c.AvatarOpts{Px: 20}), c.Text("Ada Lovelace"))),
				with(c.DataItemWide("Status"), c.StatusBadge("Active", c.ToneSuccess, c.StatusBadgeOpts{})),
			))),
		group("Icon sizes (IconSized)",
			cell("12", c.IconSized("plus", 12)), cell("14", c.IconSized("plus", 14)), cell("16", c.IconSized("plus", 16)),
			cell("20", c.IconSized("plus", 20)), cell("24", c.IconSized("plus", 24)), cell("default", c.Icon("plus"))),
		groupStack("Description list: columns 1, 2, 3",
			cell("Columns 1", c.DescriptionList(dlItems(), c.DescriptionListOpts{Columns: 1})),
			cell("Columns 2", c.DescriptionList(dlItems(), c.DescriptionListOpts{Columns: 2})),
			cell("Columns 3", c.DescriptionList(dlItems(), c.DescriptionListOpts{Columns: 3})),
			cell("Layout grid (legacy)", c.DescriptionList(dlItems(), c.DescriptionListOpts{Layout: c.DescriptionGrid})),
		),
		groupGrid("KPI tiles",
			c.KPITile("Open AR", "$48,200.00", c.KPITileOpts{Hint: "unpaid invoices", Icon: "payments"}),
			c.KPITile("Trial balance", "Balanced", c.KPITileOpts{Tone: c.ToneSuccess, Icon: "scale"}),
			c.KPITile("Overdue", "$3,100.00", c.KPITileOpts{Tone: c.ToneDanger, Hint: "12 invoices"}),
			c.KPITile("Pending", "14", c.KPITileOpts{Tone: c.ToneWarning, Href: "/pending"}),
			c.KPITile("Info", "7", c.KPITileOpts{Tone: c.ToneInfo}),
			c.KPITile("Agent suggestions", "23", c.KPITileOpts{Tone: c.ToneAI, Icon: "spark"}),
			c.KPITile("Unbroken value", "A value that is very long and must wrap without overflowing the tile", c.KPITileOpts{}),
		),
		groupGrid("Checklist",
			cell("normal", c.Checklist([]c.ChecklistItem{
				{Title: "Create your profile", Done: true},
				{Title: "Verify your email", Description: "We sent a link to your inbox.", Href: "/verify", ActionLabel: "Resend"},
				{Title: "Connect a bank", Description: "Optional, speeds up reconciliation.", Href: "/bank"},
				{Title: "Invite the team", Description: "Add colleagues who need access."},
			})),
			cell("compact", c.ChecklistWith([]c.ChecklistItem{
				{Title: "Create your profile", Done: true},
				{Title: "Verify your email", Description: "hidden in compact", Href: "/verify"},
				{Title: "Invite the team"},
			}, c.ChecklistOpts{Compact: true})),
		),
		group("Orb sizes (idle)",
			cell("16", c.Orb(c.OrbOpts{Size: 16, State: c.OrbIdle})),
			cell("20", c.Orb(c.OrbOpts{Size: 20, State: c.OrbIdle})),
			cell("28", c.Orb(c.OrbOpts{Size: 28, State: c.OrbIdle})),
			cell("40", c.Orb(c.OrbOpts{Size: 40, State: c.OrbIdle})),
			cell("64", c.Orb(c.OrbOpts{Size: 64, State: c.OrbIdle})),
			cell("120", c.Orb(c.OrbOpts{Size: 120, State: c.OrbIdle})),
		),
		group("Orb on a dark surface (tokens --oui-orb-a/-b/-c keep their colour)",
			raw(`<div data-theme="dark" style="display:flex;gap:16px;padding:16px;background:#14161c;border-radius:12px">`),
			c.Orb(c.OrbOpts{Size: 40, State: c.OrbIdle}), c.Orb(c.OrbOpts{Size: 64}), c.Orb(c.OrbOpts{Size: 28, State: c.OrbThinking}),
			raw(`</div>`)),
		group("Orb states",
			cell("static", c.Orb(c.OrbOpts{Size: 64})),
			cell("idle", c.Orb(c.OrbOpts{Size: 64, State: c.OrbIdle})),
			cell("thinking", c.Orb(c.OrbOpts{Size: 64, State: c.OrbThinking})),
			cell("speaking", c.Orb(c.OrbOpts{Size: 64, State: c.OrbSpeaking})),
		),
		groupGrid("State panels",
			cell("icon", c.StatePanel("No journal entries yet", c.StatePanelOpts{Icon: "journal"}, dispText("Post an entry from the New entry button."))),
			cell("orb + actions", c.StatePanel("Ask your agent", c.StatePanelOpts{Orb: true, Actions: c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Start", Icon: "spark"})}, dispText("The agent can draft the first version for you."))),
			cell("loading", c.StatePanel("Loading", c.StatePanelOpts{Loading: true}, nil)),
			cell("compact", c.StatePanel("You are all caught up", c.StatePanelOpts{Icon: "check", Compact: true}, dispText("Nothing is assigned to you."))),
		),
	)})
}

func dlItems() []c.DescriptionItem {
	return []c.DescriptionItem{
		{Term: "Role", Value: "Senior engineer", Field: "role"},
		{Term: "Salary", Value: "120,000 USD", Mono: true, Field: "salary"},
		{Term: "Location", Value: "Remote, EU time zones"},
		{Term: "Start date"},
		{Term: "Summary", Value: "Builds reliable systems.\nSecond line of the summary.", Wide: true},
		{Term: "Identifier", Value: "https://example.com/very/long/unbroken/value/that/must/wrap/safely/inside/the/column"},
	}
}
