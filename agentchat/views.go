package agentchat

import (
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// Role is the kind of a thread row.
type Role string

// The roles a Message can have.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleNote      Role = "note"
	RoleStopped   Role = "stopped"
	RoleAnswered  Role = "answered"
)

// Step is one settled line of an assistant reply's step list. Count is the
// number of times the step ran (×N shows from 2); Caption is the narration
// the agent wrote before running it.
type Step struct {
	Label   string
	Count   int
	Caption string
}

// Message is one persisted thread row.
//
//	ID — optional; rendered as data-message-id on the row.
//	Text — the user's text, the note text, or the answered summary.
//	Context — user rows only; a small slot above the bubble, right-aligned (a
//	    hand-off chip such as "From the invoice page"). The app styles its content.
//	Content — the assistant body, rendered by the app; nil renders Markdown(Text).
//	Steps — assistant only; shown behind the folded "N steps" toggle.
//	Title — the note title.
type Message struct {
	ID      string
	Role    Role
	Text    string
	Content templ.Component
	Context templ.Component
	Steps   []Step
	Title   string
}

// Variant selects the chat's layout.
type Variant string

// VariantPanel is the compact layout for a side panel or overlay: no rail,
// no docked panel, tighter spacing.
const VariantPanel Variant = "panel"

// ChatOpts configures Chat.
//
//	TurnURL — where the runtime POSTs each turn (also the default action of card forms).
//	CancelURL — optional; the runtime POSTs session_id there when the user presses Stop.
//	StateURL — optional; polled with ?session_id= while InFlight (see Chat).
//	SessionID — the current session; empty for a new chat.
//	InFlight — the page was rendered while a turn is still running; needs StateURL.
//	AutoSend — a message the runtime sends once on load.
//	Transcript — also send the completed turns on screen with each turn.
//	Label — the chat's accessible name (default "Conversation").
//	Variant — "" for a full page, VariantPanel for a compact panel.
//	RailLabel, PanelLabel — accessible names of the rail and panel (defaults
//	    "Conversations" and "Details").
//	Header, Rail, Panel — optional slots. Rail is a left list and Panel a right
//	    panel; both dock on wide screens and slide over below that, with a scrim.
//	Thread — the persisted messages, normally Thread(msgs).
//	Empty — shown while the thread is empty (Hero and Starters).
//	Card — the pending card (a QuestionForm or Card) shown after the thread.
//	Composer — normally Composer(...).
//	Class, Attrs — extra classes and attributes on the root.
type ChatOpts struct {
	TurnURL, CancelURL, StateURL string
	SessionID                    string
	InFlight                     bool
	AutoSend                     string
	Transcript                   bool
	Label                        string
	Variant                      Variant
	RailLabel, PanelLabel        string
	Header, Rail, Panel          templ.Component
	Thread, Empty, Card          templ.Component
	Composer                     templ.Component
	Class                        string
	Attrs                        templ.Attributes
}

func (o ChatOpts) class() string {
	return join("ui-chat", when(o.Variant == VariantPanel, "ui-chat--panel"), o.Class)
}

func (o ChatOpts) label() string { return or(o.Label, "Conversation") }

// resuming reports whether the page must show a pending row and poll for the turn.
func (o ChatOpts) resuming() bool { return o.InFlight && o.StateURL != "" }

func (o ChatOpts) showEmpty() bool { return o.Empty != nil && !o.resuming() && o.AutoSend == "" }

// HeadOpts configures Head: the chat header with the orb and status line the
// runtime updates ("Ready", "Thinking…", "Writing…").
//
//	Title — the heading.
//	Leading — a slot before the orb (a Toggle for the rail).
//	Meta — a slot at the right (chips, a Toggle for the panel).
type HeadOpts struct {
	Title         string
	Leading, Meta templ.Component
}

// Dock names a slide-over region of the chat.
type Dock string

// The docks Toggle can open.
const (
	DockRail  Dock = "rail"
	DockPanel Dock = "panel"
)

// ToggleOpts configures Toggle.
//
//	Dock — the region it opens and closes.
//	Label — the accessible name.
//	Icon — an icon-set name; default "history" for the rail and "panel-right" for the panel.
type ToggleOpts struct {
	Dock  Dock
	Label string
	Icon  string
}

func (o ToggleOpts) icon() string {
	switch {
	case o.Icon != "":
		return o.Icon
	case o.Dock == DockRail:
		return "history"
	}
	return "panel-right"
}

// AssistantOpts configures AssistantMessage: Content is the app-rendered body
// and Steps the settled step list.
type AssistantOpts struct {
	Content templ.Component
	Steps   []Step
}

// ComposerOpts configures Composer.
//
//	Name — the textarea's field name (default "message").
//	Label — its accessible name (default "Message").
//	Placeholder, Value — hint text and prefilled text.
//	MaxLength — when positive, caps the text (the maxlength attribute).
//	Counter — also shows "n / MaxLength" in the bar; needs MaxLength.
//	Notice — a slot inside the box under the textarea (a line such as "Press
//	    Enter to send this"); the runtime removes it once the user edits or sends.
//	Hint — a muted line under the box, after the key hints.
//	Tools — a slot at the bar's left (an attach button, a Toggle).
//	Quick — a slot above the box for QuickPrompts; hidden while a turn runs.
//	Hidden — extra hidden fields sent with every typed turn.
//	Attrs — extra attributes on the form (for example a no-JavaScript action).
type ComposerOpts struct {
	Name, Label, Placeholder, Value string
	MaxLength                       int
	Counter                         bool
	Hint                            string
	Tools, Quick, Notice            templ.Component
	Hidden                          map[string]string
	Attrs                           templ.Attributes
}

func (o ComposerOpts) name() string  { return or(o.Name, "message") }
func (o ComposerOpts) label() string { return or(o.Label, "Message") }

func (o ComposerOpts) maxLength() string { return strconv.Itoa(o.MaxLength) }

func (o ComposerOpts) counts() bool { return o.Counter && o.MaxLength > 0 }

func (o ComposerOpts) hiddenKeys() []string { return sortedKeys(o.Hidden) }

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Prompt is a quick prompt button: Prompt is sent as the user's message. With
// Action set it sends nothing; the runtime dispatches agentchat:starter instead.
type Prompt struct {
	Label, Prompt, Action, Icon string
}

// Starter is a card in the empty state. Prompt is sent as the user's message;
// with Action set it sends nothing and the runtime dispatches agentchat:starter
// with that action name (an upload, a dialog) for the app to handle.
//
//	Eyebrow — a small line above the title; Body — a line below it.
//	Icon — an icon-set name; Tone — ToneBrand (default) or ToneAccent tints it.
type Starter struct {
	Eyebrow, Title, Body, Prompt, Action, Icon string
	Tone                                       c.Tone
}

// trigger is the attribute pair that makes a starter or quick prompt do its job.
func trigger(prompt, action string) templ.Attributes {
	if action != "" {
		return templ.Attributes{"data-agentchat-action": action}
	}
	return templ.Attributes{"data-agentchat-prompt": prompt}
}

func (s Starter) iconClass() string {
	if s.Tone == c.ToneAccent {
		return "ui-chat-starter__icon ui-chat-starter__icon--accent"
	}
	return "ui-chat-starter__icon"
}

// HeroOpts configures Hero: the orb, a title and a line of context.
type HeroOpts struct{ Title, Body string }

// CardOpts configures Card, the shell for app-owned interactive bodies.
//
//	Body — the card content; forms in it choose a submit mode (see the docs).
//	Label — the card's accessible name.
//	Class — extra classes.
type CardOpts struct {
	Body  templ.Component
	Label string
	Class string
}

// stepTotal is the number of times steps ran, repeats included.
func stepTotal(steps []Step) int {
	n := 0
	for _, s := range steps {
		n += max(s.Count, 1)
	}
	return n
}

func stepsLabel(steps []Step) string {
	if n := stepTotal(steps); n != 1 {
		return strconv.Itoa(n) + " steps"
	}
	return "1 step"
}

func rowAttrs(id string) templ.Attributes {
	if id == "" {
		return nil
	}
	return templ.Attributes{"data-message-id": id}
}

func join(parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " ")
}

func when(cond bool, class string) string {
	if cond {
		return class
	}
	return ""
}

func or(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
