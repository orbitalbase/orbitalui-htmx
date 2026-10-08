package main

import (
	"context"
	"io"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func text(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func badge(label string, tone c.Tone) templ.Component {
	return c.StatusBadge(label, tone, c.StatusBadgeOpts{})
}

func rowSet(opts c.ActionListOpts, below bool) templ.Component {
	var extra templ.Component = seq()
	if below {
		extra = c.ActionRow(c.ActionRowOpts{Title: "Row with below content", Meta: "Stacked", Icon: "bell", Stacked: true, Below: text(`<span class="g-cell__label">A note under the copy column.</span>`)})
	}
	return c.ActionList(opts, seq(
		c.ActionRow(c.ActionRowOpts{Title: "Board minutes", Meta: "Signed · 3 Mar 2024", Icon: "document", Href: "/a", Trailing: badge("Signed", c.ToneSuccess)}),
		c.ActionRow(c.ActionRowOpts{Title: "Share transfer register", Meta: "Needs a second approver before it can be filed", Icon: "scale", IconTone: c.ToneWarning, Href: "/b", Value: "€ 12,400.00"}),
		c.ActionRow(c.ActionRowOpts{Title: "Assistant draft", Meta: "Prepared from the uploaded articles", Icon: "spark", IconTone: c.ToneAI, Href: "/c", Current: true}),
		c.ActionRow(c.ActionRowOpts{Title: "A very long row title that never fits on one line and must wrap instead of squeezing the trailing column into nothing", Meta: "Meta line that is also long enough to be truncated on desktop and wrapped on a phone screen", Icon: "folder", IconTone: c.ToneInfo, Href: "/d"}),
		c.ActionRow(c.ActionRowOpts{Title: "Row without an icon", Meta: "Its value drops flush to the copy on a phone", Href: "/e", Value: "MYR 350"}),
		c.ActionRow(c.ActionRowOpts{Title: "Static row with actions", Meta: "No link; controls live on the trailing edge", Icon: "mail", IconTone: c.ToneDanger,
			Trailing: seq(c.Button(c.ButtonVariantSecondary, c.ButtonSizeSM, c.ButtonOpts{Label: "Resend"}), c.Button(c.ButtonVariantQuiet, c.ButtonSizeSM, c.ButtonOpts{Label: "Revoke"}))}),
		extra,
	))
}

func init() {
	register(section{ID: "rows", Title: "Rows, facts and criteria", Note: "ActionList variants, FactList, CriteriaList.", Body: seq(
		groupStack("ActionList divided (in a flush card)", c.Card(c.CardOpts{Title: "Documents", Flush: true}, rowSet(c.ActionListOpts{}, true))),
		groupStack("ActionList plain, compact", c.Card(c.CardOpts{Title: "Compact"}, rowSet(c.ActionListOpts{Variant: c.ActionListPlain, Compact: true}, true))),
		groupGrid("ActionList cards (labelled nav)", rowSet(c.ActionListOpts{Variant: c.ActionListCards, Label: "Next steps"}, false)),
		group("FactList", c.FactList([]c.Fact{{Term: "Quota", Value: "50 GB"}, {Term: "Period", Value: "Mar 2024"}, {Term: "Cap", Value: "€ 100,000"}, {Term: "Reference", Value: "a-very-long-unbroken-reference-string-0123456789-abcdefghijklmnopqrstuvwxyz"}})),
		group("CriteriaList", c.CriteriaList([]c.Criterion{
			{Met: true, Label: "Two directors approved", Note: "Read: 2 of 2"},
			{Met: false, Label: "Registered office is in the jurisdiction", Note: "Read: not provided"},
			{Met: true, Label: "Articles on file"},
		})),
	)})

	register(section{ID: "choices", Title: "Choice rows and decision card", Body: seq(
		groupGrid("ChoiceGroup radio",
			c.ChoiceGroup("Delivery", c.ChoiceGroupOpts{ShowLegend: true}, seq(
				c.ChoiceRow(c.ChoiceRowOpts{Type: "radio", Name: "d", Value: "a", Label: "Email", Description: "Sent as soon as it is approved.", Checked: true}),
				c.ChoiceRow(c.ChoiceRowOpts{Type: "radio", Name: "d", Value: "b", Label: "Post", Description: "Arrives in three to five working days."}),
				c.ChoiceRow(c.ChoiceRowOpts{Type: "radio", Name: "d", Value: "c", Label: "Courier (unavailable)", Description: "Disabled choice.", Disabled: true}),
			)),
			c.ChoiceGroup("Alerts", c.ChoiceGroupOpts{ShowLegend: true}, seq(
				c.ChoiceRow(c.ChoiceRowOpts{Name: "al", Value: "1", Label: "Filing due", Description: "Seven days before the deadline.", Checked: true}),
				c.ChoiceRow(c.ChoiceRowOpts{Name: "al", Value: "2", Label: "Approval requested"}),
				c.ChoiceRow(c.ChoiceRowOpts{Name: "al", Value: "3", Label: "A choice with a very long label that has to wrap across several lines on a narrow phone screen without breaking the row"}),
			)),
		),
		groupGrid("DecisionCard",
			c.DecisionCard(c.DecisionCardOpts{ID: "dc1", Subject: "Share transfer", Context: "share_transfer · v3", Status: badge("Needs review", c.ToneWarning),
				Evidence: text(`Source: transfer form, page 2`),
				Actions:  seq(c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Approve"}), c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Reject"}))},
				text(`<p>Move 1,000 ordinary shares from A. Holder to B. Buyer at par value.</p>`)),
			c.DecisionCard(c.DecisionCardOpts{ID: "dc2", Subject: "Unbroken_identifier_that_is_extremely_long_and_has_no_spaces_at_all_0123456789", Actions: c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Open"})},
				text(`<p>Minimal card.</p>`)),
		),
		groupStack("SectionNav (sticky in real pages)", c.SectionNav([]c.SectionLink{{Label: "Team", ID: "team"}, {Label: "Billing", ID: "billing"}, {Label: "Security", ID: "security"}}, c.SectionNavOpts{})),
	)})

	register(section{ID: "chat", Title: "Chat and starters", Note: "Thread, messages, suggestion chips, starter controls.", Body: seq(
		groupStack("ChatThread",
			c.ChatThread("g-thread", "Conversation", "g-stack", seq(
				c.UserMessage("Draft a board resolution for the share transfer."),
				c.AssistantMessage("Here is a first draft. I used the transfer details from the register; please check the figures before approving."),
				c.UserMessage("A very long unbroken message: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				c.AssistantMessageContent(text(`<p>Sources: <a href="/documents/one">Transfer form</a>, <a href="/documents/two">Articles</a>.</p>`)),
				c.AssistantMessageContent(seq(text(`<p>Streaming a reply`), c.StreamingCaret(), text(`</p>`))),
				c.TypingIndicator("Assistant is typing"),
			)),
		),
		groupStack("ChatComposer",
			cell("empty (send dims)", c.ChatComposer(c.ChatComposerOpts{ID: "g-comp-a", Action: "/send", Placeholder: "Ask the assistant, or describe what you need…", Hint: "The assistant can make mistakes. Check important figures.",
				Tools: c.Button(c.ButtonVariantQuiet, c.ButtonSizeSM, c.ButtonOpts{Icon: "paperclip", Label: "Attach"})})),
			cell("with text", c.ChatComposer(c.ChatComposerOpts{ID: "g-comp-b", Action: "/send", Value: "Draft a board resolution for the share transfer."})),
			cell("busy", c.ChatComposer(c.ChatComposerOpts{ID: "g-comp-c", Action: "/send", Value: "Summarise the register", Busy: true})),
		),
		group("SuggestionChip and submit chip",
			c.SuggestionSubmitChip("Summarise the register", "q", "summary"),
			c.SuggestionSubmitChip("Who approved this?", "q", "who"),
			raw(`<button type="button" class="oui-suggestion-chip" disabled>Disabled suggestion</button>`),
		),
		groupGrid("SummaryCard", c.SummaryCard("Draft", "spark", "", seq(
			raw(`<div class="contents"><dt class="oui-summary-card__term">Company</dt><dd class="oui-summary-card__value">Acme Ltd</dd></div>`),
		))),
		groupGrid("Starter fields",
			cell("oui-control", raw(`<label class="oui-field-label" for="g-s1">Company name</label><input id="g-s1" class="oui-control" placeholder="Acme Ltd"><div class="oui-character-count">12 / 80</div>`)),
			cell("disabled", raw(`<label class="oui-field-label" for="g-s2">Locked</label><input id="g-s2" class="oui-control" value="Read only" disabled>`)),
		),
	)})
}
