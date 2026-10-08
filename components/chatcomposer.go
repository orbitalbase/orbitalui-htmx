// ChatComposer and TypingIndicator (chatcomposer.templ): the message box and
// the "assistant is working" cue that go with ChatThread, AssistantMessage
// and UserMessage. The agent workspace layout stays in the host; these are
// the shared pieces.
package components

import "github.com/a-h/templ"

// ChatComposerOpts configures ChatComposer.
//
//	Action — form action (a plain POST works without JavaScript); HXPost /
//	    HXTarget / HXSwap wire it to htmx instead.
//	Name — the textarea's field name (default "message").
//	Label — the textarea's accessible name (default "Message").
//	Placeholder — hint text. Value — prefilled text.
//	Tools — a slot at the bar's left (attach button, quick actions).
//	Hint — a muted line under the box (a shortcut tip, a disclaimer).
//	Busy — renders the busy state: the send button shows the AI orb and the
//	    textarea stays editable.
//	SendLabel — the send button's accessible name (default "Send").
//	ID — the form id; Class — extra classes; Attrs — extra form attributes.
type ChatComposerOpts struct {
	Action      string
	Method      string
	HXPost      string
	HXTarget    string
	HXSwap      string
	Name        string
	Label       string
	Placeholder string
	Value       string
	Tools       templ.Component
	Hint        string
	Busy        bool
	SendLabel   string
	ID          string
	Class       string
	Attrs       templ.Attributes
}

func composerName(o ChatComposerOpts) string {
	if o.Name == "" {
		return "message"
	}
	return o.Name
}

func composerLabel(o ChatComposerOpts) string {
	if o.Label == "" {
		return "Message"
	}
	return o.Label
}

func composerSendLabel(o ChatComposerOpts) string {
	if o.SendLabel == "" {
		return "Send"
	}
	return o.SendLabel
}

func composerMethod(o ChatComposerOpts) string {
	if o.Method == "" {
		return "post"
	}
	return o.Method
}

func composerClass(o ChatComposerOpts) string {
	return joinClass("ui-composer", modifier(o.Busy, "is-busy"), o.Class)
}

func composerFieldID(o ChatComposerOpts) string {
	if o.ID == "" {
		return "composer-input"
	}
	return o.ID + "-input"
}
