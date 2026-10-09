package main

import (
	"context"

	"github.com/a-h/templ"

	"github.com/orbitalbase/orbitalui-htmx/agentchat"
	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func init() {
	register(section{ID: "agent-chat", Title: "Agent chat", Note: "agentchat.Chat as the runtime leaves it: thread rows, folded and live steps, question and action cards, starters, composer and the compact panel variant.", Body: agentChatBody()})
}

func chatFrame(chat templ.Component) templ.Component {
	return wrap(`<div class="g-chatframe">`, `</div>`, chat)
}

func agentChatBody() templ.Component {
	thread := agentchat.Thread([]agentchat.Message{
		{Role: agentchat.RoleUser, Text: "Find me a remote product design role in Kuala Lumpur."},
		{Role: agentchat.RoleAssistant, Text: "I found **three** matches. The strongest is *Product Designer* at Northwind:\n\n- Remote, KL timezone\n- RM 9,000 to 12,000 a month\n- [Read the listing](https://example.com/listing)\n\nWant me to draft your application?", Steps: []agentchat.Step{
			{Label: "Reading your profile", Count: 1},
			{Label: "Searching listings", Count: 2, Caption: "Let me check the open roles"},
			{Label: "Ranking matches", Count: 1},
		}},
		{Role: agentchat.RoleAnswered, Text: "Work arrangement: Remote · Minimum pay: 9,000"},
		{Role: agentchat.RoleNote, Title: "Draft saved", Text: "Product Designer, version 1"},
		{Role: agentchat.RoleUser, Text: "Actually, stop. I changed my mind.", Context: handoffChip("From the roles page →")},
		{Role: agentchat.RoleAssistant, Text: "Okay, pausing here."},
		{Role: agentchat.RoleStopped},
	})
	card := agentchat.QuestionForm(agentchat.QuestionCardView{SessionID: "s1", Card: agentchat.QuestionCard{
		Token: "gallery", Title: "A few details", Description: "So I can narrow the matches.",
		Questions: []agentchat.Question{
			{Key: "mode", Label: "Work arrangement", Type: agentchat.TypeSelect, Options: []string{"Remote", "Hybrid", "On site"}, Value: "Remote", Required: true},
			{Key: "skills", Label: "Skills", Type: agentchat.TypeMultiselect, Options: []string{"Figma", "Research", "Prototyping"}, Values: []string{"Figma"}},
			{Key: "min", Label: "Minimum pay", Type: agentchat.TypeNumber, Value: "9000", Help: "Per month, in RM."},
			{Key: "about", Label: "Anything else?", Type: agentchat.TypeTextarea, Placeholder: "Team size, tools, travel"},
		},
	}})
	composer := agentchat.Composer(agentchat.ComposerOpts{
		Placeholder: "Message your agent…", MaxLength: 8000, Counter: true, Notice: seq(c.IconSized("ai", 12), c.Kbd("↵"), raw(" to send this to your agent")), Hint: "drafts stay private until you publish",
		Quick: agentchat.QuickPrompts([]agentchat.Prompt{{Label: "Explore trade-offs", Prompt: "x", Icon: "ai"}, {Label: "Find gaps", Prompt: "y", Icon: "search"}}),
		Tools: agentchat.Toggle(agentchat.ToggleOpts{Dock: agentchat.DockPanel, Label: "Show details"}),
	})
	full := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/turn", Label: "Assistant",
		Header: agentchat.Head(agentchat.HeadOpts{Title: "Assistant", Leading: agentchat.Toggle(agentchat.ToggleOpts{Dock: agentchat.DockRail, Label: "Show chats"}), Meta: c.Badge(c.ToneBrand, "Talent", c.BadgeOpts{})}),
		Thread: thread, Card: card, Composer: composer,
	})
	empty := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/turn", Label: "Assistant",
		Header: agentchat.Head(agentchat.HeadOpts{Title: "Assistant"}),
		Empty: seq(
			agentchat.Hero(agentchat.HeroOpts{Title: "Hi Ada. What should we work on?", Body: "Tell me in your own words. I build the listing as we talk."}),
			agentchat.Starters([]agentchat.Starter{
				{Title: "Find my next role", Body: "Describe the work you want.", Icon: "target", Prompt: "a"},
				{Title: "Hire for a role", Body: "Describe who you need.", Icon: "briefcase", Prompt: "b", Tone: c.ToneAccent},
				{Eyebrow: "Import", Title: "Bring my CV", Body: "Upload a CV to fill your profile.", Icon: "upload", Action: "import"},
				{Title: "Set up a company", Icon: "building", Prompt: "d", Tone: c.ToneAccent},
			}),
		),
		Thread: agentchat.Thread(nil), Composer: agentchat.Composer(agentchat.ComposerOpts{Placeholder: "Message your agent…"}),
	})
	compact := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/turn", Label: "Ask", Variant: agentchat.VariantPanel,
		Header: agentchat.Head(agentchat.HeadOpts{Title: "Ask about this page"}),
		Thread: agentchat.Thread([]agentchat.Message{
			{Role: agentchat.RoleUser, Text: "Why is the balance off by RM 40?"},
			{Role: agentchat.RoleAssistant, Text: "A bank fee on 3 March was not matched. Open **Reconcile** to post it.", Steps: []agentchat.Step{{Label: "Checking the bank feed", Count: 1}}},
		}),
		Composer: agentchat.Composer(agentchat.ComposerOpts{Placeholder: "Ask…"}),
	})
	action := agentchat.Card(agentchat.CardOpts{Label: "Payment proposal", Body: seq(
		raw(`<form method="post" action="/authorize" data-agentchat-submit="action"><div class="ui-chat-card__head"><div><h3 class="ui-chat-card__title">Pay Acme Sdn Bhd</h3><p class="ui-chat-card__desc">Invoice INV-204, due 12 Oct. RM 1,200.00</p></div></div><div class="ui-chat-card__foot"><span class="ui-chat-card__spacer"></span>`),
		c.Button(c.ButtonVariantQuiet, c.ButtonSizeSM, c.ButtonOpts{Type: "button", Label: "Not now"}),
		c.Button(c.ButtonVariantPrimary, c.ButtonSizeSM, c.ButtonOpts{Type: "submit", Label: "Authorize"}),
		raw(`</div></form>`),
	)})
	return seq(
		groupStack("Full page: thread, steps, notes, answered, stopped, question card, composer", chatFrame(full)),
		groupStack("Empty state: hero and starters", chatFrame(empty)),
		groupStack("Panel variant", wrap(`<div class="g-chatframe g-chatframe--narrow">`, `</div>`, compact)),
		groupStack("Live step list while a turn runs (active and done)", wrap(`<div class="g-panel">`, `</div>`, liveSteps())),
		groupStack("App-run work: a working note (controller.note)", wrap(`<div class="g-panel"><div class="g-stack">`, `</div></div>`, workingNote("Reading cv.pdf…", "Pulling out your roles, skills and preferences."), workingNote("Profile imported", ""))),
		groupStack("Card shell with an action form", wrap(`<div class="g-panel">`, `</div>`, action)),
	)
}

// liveSteps is the markup the runtime builds while a turn streams: a done
// step, one with a count and a caption, and the active step.
func liveSteps() templ.Component {
	tick, _ := templ.ToGoHTML(context.Background(), c.IconSized("check", 12))
	mark := `<span class="ui-chat-step__mark">` + string(tick) + `</span>`
	return raw(`<div class="ui-chat-msg ui-chat-msg--assistant"><span class="ui-chat-msg__orb"><span class="ui-orb ui-orb--20 ui-orb--xs is-thinking"><span class="ui-orb__core"></span></span></span><div class="ui-chat-msg__col"><ol class="ui-chat-steps" aria-label="Progress">` +
		`<li class="ui-chat-step is-done">` + mark + `<span class="ui-chat-step__text"><span class="ui-chat-step__label"><span>Reading your profile</span></span></span></li>` +
		`<li class="ui-chat-step is-done">` + mark + `<span class="ui-chat-step__text"><span class="ui-chat-step__label"><span>Searching listings</span><span class="ui-chat-step__count num">×2</span></span><span class="ui-chat-step__note">Let me check the open roles</span></span></li>` +
		`<li class="ui-chat-step is-active">` + mark + `<span class="ui-chat-step__text"><span class="ui-chat-step__label"><span>Ranking matches…</span></span></span></li></ol>` +
		`<div class="ui-chat-msg__body ui-chat-md is-provisional">Three matches so far<span class="ui-caret"></span></div></div></div>`)
}

// workingNote is the note row the runtime builds for controller.note: the
// first is in progress (working(true)), the second finished.
func workingNote(title, text string) templ.Component {
	icon, _ := templ.ToGoHTML(context.Background(), c.IconSized("document", 14))
	class, body := "ui-chat-note is-working", `<p data-agentchat-note-text>`+text+`</p>`
	if text == "" {
		class, body = "ui-chat-note", `<p data-agentchat-note-text hidden></p>`
	}
	return raw(`<div class="` + class + `"><span class="ui-chat-note__icon">` + string(icon) + `</span><div class="ui-chat-note__body"><strong>` + title + `</strong>` + body + `</div></div>`)
}

func handoffChip(label string) templ.Component { return c.Badge(c.ToneBrand, label, c.BadgeOpts{}) }
