package agentchat

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func render(t *testing.T, comp templ.Component) string {
	t.Helper()
	var out bytes.Buffer
	if err := comp.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func has(t *testing.T, out string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(out, n) {
			t.Errorf("output omitted %q:\n%s", n, out)
		}
	}
}

func lacks(t *testing.T, out string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(out, n) {
			t.Errorf("output contains %q:\n%s", n, out)
		}
	}
}

func TestChatRootAttributes(t *testing.T) {
	out := render(t, Chat(ChatOpts{
		TurnURL: "/turn", CancelURL: "/cancel", StateURL: "/state", SessionID: "s1", InFlight: true,
		Transcript: true, Label: "Ask", Variant: VariantPanel, Class: "host",
		Thread: Thread(nil), Composer: Composer(ComposerOpts{}), Empty: Hero(HeroOpts{Title: "Hi"}),
		Attrs: templ.Attributes{"data-host": "1"},
	}))
	has(t, out, `class="ui-chat ui-chat--panel host"`, "data-agentchat", `data-turn-url="/turn"`, `data-cancel-url="/cancel"`,
		`data-state-url="/state"`, `data-session-id="s1"`, "data-inflight", "data-agentchat-transcript", `aria-label="Ask"`, `data-host="1"`,
		"data-agentchat-pending", `aria-busy="true"`, "data-agentchat-card-slot", "data-agentchat-newpill", "data-agentchat-scrim")
	lacks(t, out, "data-agentchat-empty", "data-autosend", "data-agentchat-rail", "data-agentchat-panel")
}

func TestChatOptionalPartsAndEmptyState(t *testing.T) {
	out := render(t, Chat(ChatOpts{
		TurnURL: "/turn", Header: Head(HeadOpts{Title: "Agent"}), Rail: Markdown("rail"), Panel: Markdown("panel"),
		RailLabel: "Chats", Empty: Hero(HeroOpts{Title: "Hi", Body: "Ask me"}), Card: Markdown("card"),
		Thread: Thread(nil), Composer: Composer(ComposerOpts{}),
	}))
	has(t, out, `data-agentchat-rail aria-label="Chats"`, `data-agentchat-panel aria-label="Details"`, "data-agentchat-empty", "ui-chat-hero__title",
		`data-agentchat-status`, `data-agentchat-orb`, "<h1", "<p>card</p>")
	lacks(t, out, "data-inflight", "data-cancel-url", "data-state-url", "data-agentchat-pending")

	// An in-flight page or an auto-sent message replaces the empty state.
	for _, o := range []ChatOpts{{InFlight: true, StateURL: "/s", Empty: Hero(HeroOpts{})}, {AutoSend: "hello", Empty: Hero(HeroOpts{})}} {
		o.Thread = Thread(nil)
		lacks(t, render(t, Chat(o)), "data-agentchat-empty")
	}
	// InFlight without a StateURL cannot be resumed: no pending row.
	lacks(t, render(t, Chat(ChatOpts{InFlight: true})), "data-inflight", "data-agentchat-pending")
	has(t, render(t, Chat(ChatOpts{AutoSend: "hello <b>"})), `data-autosend="hello &lt;b&gt;"`)
}

func TestChatShipsEveryRuntimeTemplate(t *testing.T) {
	out := render(t, Chat(ChatOpts{}))
	for _, name := range []string{"user", "assistant", "step", "error", "answered", "note"} {
		has(t, out, `<template data-agentchat-tpl="`+name+`">`)
	}
	has(t, out, "data-agentchat-bubble", "data-agentchat-steps", "data-agentchat-typing", "data-agentchat-body", "data-agentchat-step-label",
		"data-agentchat-step-note", "data-agentchat-error-text", "data-agentchat-retry", "data-agentchat-summary", "data-agentchat-note-title")
}

func TestThreadRowsRolesAndEscaping(t *testing.T) {
	out := render(t, Thread([]Message{
		{ID: "m1", Role: RoleUser, Text: `<script>alert(1)</script>`},
		{Role: RoleAssistant, Text: "A **bold** answer", Steps: []Step{{Label: "Reading", Count: 3, Caption: "Let me look"}, {Label: "Saving"}}},
		{Role: RoleAssistant, Content: Markdown("custom *body*")},
		{Role: RoleNote, Title: "Done", Text: "Sent <1>"},
		{Role: RoleStopped},
		{Role: RoleAnswered, Text: "Remote"},
	}))
	has(t, out, `role="log"`, `aria-live="polite"`, "data-agentchat-messages", `data-message-id="m1"`,
		"&lt;script&gt;alert(1)&lt;/script&gt;", `aria-label="You said"`, "<strong>bold</strong>", "<em>body</em>",
		`data-agentchat-steps-toggle`, ">4 steps<", `×3`, "Let me look", "ui-chat-step is-done",
		"ui-chat-note", "Sent &lt;1&gt;", "ui-chat-stopped--row", "ui-chat-answered__text", "Remote")
	lacks(t, out, "<script>", "is-pending", "aria-busy")
}

func TestAssistantMessageWithoutStepsHasNoToggle(t *testing.T) {
	out := render(t, AssistantMessage(AssistantOpts{Content: Markdown("hi")}))
	has(t, out, "ui-chat-msg--assistant", `aria-label="Assistant message"`, "data-agentchat-body")
	lacks(t, out, "data-agentchat-steps-toggle", "ui-chat-steps")
}

func TestComposerAttributes(t *testing.T) {
	out := render(t, Composer(ComposerOpts{
		Placeholder: "Ask <anything>", Value: "draft & more", MaxLength: 500, Counter: true, Notice: Markdown("Press **Enter**"), Hint: "Private until published",
		Hidden: map[string]string{"scope": "a\"b", "area": "x"}, Quick: QuickPrompts([]Prompt{{Label: "Explore", Prompt: "Explore this", Icon: "ai"}}),
		Tools: Toggle(ToggleOpts{Dock: DockPanel, Label: "Show details"}), Attrs: templ.Attributes{"action": "/turn"},
	}))
	has(t, out, "data-agentchat-composer", `data-state="idle"`, `action="/turn"`, `name="message"`, `aria-label="Message"`, "data-agentchat-input", `maxlength="500"`,
		`placeholder="Ask &lt;anything&gt;"`, "draft &amp; more", "data-agentchat-count", "0 / 500", "data-agentchat-send", `aria-label="Send message"`,
		"Private until published", "data-agentchat-notice", "<strong>Enter</strong>", `name="scope" value="a&#34;b"`, `data-agentchat-prompt="Explore this"`, `data-agentchat-toggle="panel"`, "<kbd")
	if strings.Index(out, `name="area"`) > strings.Index(out, `name="scope"`) {
		t.Error("hidden fields are not in key order")
	}
	lacks(t, render(t, Composer(ComposerOpts{Name: "q"})), "data-agentchat-count", "data-agentchat-quick", "maxlength")
	has(t, render(t, Composer(ComposerOpts{Name: "q", Label: "Ask"})), `name="q"`, `aria-label="Ask"`)
}

func TestToggleStartersAndCards(t *testing.T) {
	has(t, render(t, Toggle(ToggleOpts{Dock: DockRail, Label: "Chats"})), `data-agentchat-toggle="rail"`, `aria-label="Chats"`, `aria-expanded="false"`, "ui-chat-toggle--rail")

	out := render(t, Starters([]Starter{
		{Title: "Find a role", Body: "Describe it", Prompt: "I want a role", Icon: "target", Eyebrow: "Talent"},
		{Title: "Hire", Prompt: "I want to hire", Tone: c.ToneAccent, Icon: "briefcase"},
		{Title: "No icon", Prompt: "p"},
	}))
	has(t, out, `data-agentchat-prompt="I want a role"`, "ui-chat-starter__eyebrow", "ui-chat-starter__icon--accent", "ui-chat-starter__body")
	if strings.Count(out, "ui-chat-starter__icon") != 3 { // two icons, one of them the accent modifier
		t.Errorf("an empty Icon must render no icon box:\n%s", out)
	}

	card := render(t, Card(CardOpts{Label: "Proposal", Class: "host", Body: Markdown("body")}))
	has(t, card, "data-agentchat-card", `role="group"`, `aria-label="Proposal"`, "ui-chat-card host", "<p>body</p>")
	lacks(t, render(t, Card(CardOpts{})), "role=", "aria-label")
}

func TestQuestionFormRendersEveryFieldType(t *testing.T) {
	card := QuestionCard{Token: "tok-123456789", Title: "Details", Description: "Two things", Questions: []Question{
		{Key: "mode", Label: "Where?", Type: TypeSelect, Options: []string{"Remote", "Hybrid"}, Value: "Hybrid", Help: "Pick one"},
		{Key: "skills", Label: "Skills", Type: TypeMultiselect, Options: []string{"Go", "SQL", "Rust"}, Values: []string{"Go", "SQL"}},
		{Key: "min", Label: "Minimum", Type: TypeNumber, Value: "11000", Required: true},
		{Key: "about", Label: "About", Type: TypeTextarea, Placeholder: "Roles"},
		{Key: "name", Label: "Name <x>", Type: TypeText, Value: "Ada", Help: "Full name"},
	}}
	out := render(t, QuestionForm(QuestionCardView{SessionID: "s1", Card: card}))
	has(t, out, `data-agentchat-card`, `data-agentchat-submit="turn"`, `aria-label="Details"`,
		`name="session_id" value="s1"`, `name="answer_questions" value="1"`, `name="question_token" value="tok-123456789"`,
		`data-agentchat-q-label="Where?"`, `data-agentchat-q-label="Name &lt;x&gt;"`,
		`name="q_mode"`, `type="radio"`, `name="q_skills"`, `type="checkbox"`, `value="Hybrid" checked`,
		`name="q_min"`, `value="11,000"`, "data-agentchat-group", `inputmode="decimal"`, `name="q_about"`, "data-autogrow", `maxlength="4000"`,
		`name="q_name"`, `value="Ada"`, `maxlength="500"`, `id="q-tok12345-name"`, `for="q-tok12345-name"`, `aria-describedby="q-tok12345-name-help"`,
		"Prefilled by your agent", `name="skip_questions"`, `data-agentchat-summary="Skipped"`, "formnovalidate", "Send answers")
	lacks(t, out, "data-agentchat-autosubmit", "<select")

	one := QuestionCard{Questions: []Question{{Key: "intent", Label: "Which?", Type: TypeButtons, Options: []string{"A", "B"}}}}
	autoOut := render(t, QuestionForm(QuestionCardView{Card: one}))
	has(t, autoOut, "data-agentchat-autosubmit", "A few quick questions")
	lacks(t, autoOut, "Prefilled by your agent")
	one.Questions[0].Value = "A"
	lacks(t, render(t, QuestionForm(QuestionCardView{Card: one})), "data-agentchat-autosubmit")
}

func TestGroupDigits(t *testing.T) {
	for in, want := range map[string]string{"11000": "11,000", "1234567.5": "1,234,567.5", "-4500": "-4,500", "12": "12", "1,000": "1,000", "": "", "abc": "abc"} {
		if got := groupDigits(in); got != want {
			t.Errorf("groupDigits(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUserContextRendersAboveTheBubble(t *testing.T) {
	out := render(t, UserMessageWith("hi <b>", Markdown("From **page**")))
	has(t, out, "ui-chat-user__context", "data-agentchat-user-context", "<strong>page</strong>", "hi &lt;b&gt;", "data-agentchat-bubble")
	if strings.Index(out, "ui-chat-user__context") > strings.Index(out, "ui-chat-bubble") {
		t.Error("the context must come before the bubble")
	}
	thread := render(t, Rows([]Message{{Role: RoleUser, Text: "a", Context: Markdown("ctx")}, {Role: RoleUser, Text: "b"}}))
	if strings.Count(thread, "ui-chat-user__context") != 1 {
		t.Errorf("only the message with Context gets the slot:\n%s", thread)
	}
	lacks(t, render(t, UserMessage("plain")), "ui-chat-user")
}

func TestComposerCounterAndNoticeAreOptIn(t *testing.T) {
	capped := render(t, Composer(ComposerOpts{MaxLength: 300}))
	has(t, capped, `maxlength="300"`)
	lacks(t, capped, "data-agentchat-count", "data-agentchat-notice")
	lacks(t, render(t, Composer(ComposerOpts{Counter: true})), "data-agentchat-count", "maxlength")
	has(t, render(t, Composer(ComposerOpts{MaxLength: 300, Counter: true})), `maxlength="300"`, "0 / 300")
}

func TestRowsAreBareAndThreadWrapsThem(t *testing.T) {
	msgs := []Message{{ID: "a", Role: RoleUser, Text: "hi"}, {Role: RoleAssistant, Text: "yo"}, {Role: RoleStopped}}
	rows := render(t, Rows(msgs))
	has(t, rows, `data-message-id="a"`, "ui-chat-msg--user", "ui-chat-msg--assistant", "ui-chat-stopped--row")
	lacks(t, rows, `role="log"`, "data-agentchat-messages")
	if thread := render(t, Thread(msgs)); !strings.Contains(thread, rows) || !strings.Contains(thread, `role="log"`) {
		t.Errorf("Thread must be the log around Rows:\n%s", thread)
	}
	if render(t, Rows(nil)) != "" {
		t.Error("no messages must render nothing")
	}
}

func TestActionStartersAndPromptsRaiseAnActionInsteadOfAPrompt(t *testing.T) {
	out := render(t, Starters([]Starter{{Title: "Import my CV", Action: "import", Prompt: "ignored", Icon: "upload"}, {Title: "Hello", Prompt: "hello"}}))
	has(t, out, `data-agentchat-action="import"`, `data-agentchat-prompt="hello"`)
	lacks(t, out, `data-agentchat-prompt="ignored"`)
	quick := render(t, QuickPrompts([]Prompt{{Label: "Import", Action: "import"}, {Label: "Ask", Prompt: "ask"}}))
	has(t, quick, `data-agentchat-action="import"`, `data-agentchat-prompt="ask"`)
	if strings.Count(quick, "data-agentchat-prompt") != 1 {
		t.Errorf("an action prompt must not carry a prompt:\n%s", quick)
	}
}

func TestToggleDefaultIconsAndOverride(t *testing.T) {
	has(t, render(t, Toggle(ToggleOpts{Dock: DockRail, Label: "Chats"})), c.Path("history"))
	has(t, render(t, Toggle(ToggleOpts{Dock: DockPanel, Label: "Details"})), c.Path("panel-right"))
	has(t, render(t, Toggle(ToggleOpts{Dock: DockPanel, Label: "Close", Icon: "close"})), c.Path("close"))
	for _, name := range []string{"history", "panel-right", "sparkles"} {
		if c.Path(name) == "" {
			t.Errorf("icon %s is missing", name)
		}
	}
}
