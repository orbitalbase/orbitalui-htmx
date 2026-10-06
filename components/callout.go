// Callout and DecisionCard (callout.templ).
//
// Alert is one sentence on a tinted strip. A Callout is the same tinted frame
// grown up: it has a heading, a header action slot and a body that can hold
// rows, so the queue a reader must clear before anything else on the page can
// live inside it rather than beside it.
//
// DecisionCard is the single unit of a review queue: the subject, the context
// that identifies it, the sentence that states the proposal, the evidence
// behind it one click away, and the explicit choices.
package components

import "github.com/a-h/templ"

// CalloutOpts configures Callout.
//
//	Title — the heading; required, because a tinted frame with no name is
//	    decoration. HeadingLevel is 2 (default) or 3.
//	Description — one line under the title.
//	Icon — a leading icon in a tile beside the heading.
//	Actions — header-right controls (a dismissal, a "review all" link).
//	ID — when set, the frame is labelled by the heading.
type CalloutOpts struct {
	Title        string
	Description  string
	Icon         string
	Actions      templ.Component
	HeadingLevel int
	ID           string
	Class        string
	Attrs        templ.Attributes
}

// calloutClass tints the frame by tone. The neutral tone maps to the quietest
// tint (info), the same rule Alert follows, so a missing tone degrades to a
// quiet frame rather than a wrong colour.
func calloutClass(tone Tone, o CalloutOpts) string {
	m := tone.Modifier()
	if m == string(ToneNeutral) {
		m = string(ToneInfo)
	}
	return joinClass("ui-callout", "ui-callout--"+m, o.Class)
}

func calloutHeadingID(o CalloutOpts) string {
	if o.ID == "" {
		return ""
	}
	return o.ID + "-title"
}

// DecisionCardOpts configures DecisionCard.
//
//	Subject — what the decision is about (a company, an event).
//	Context — the identifiers that pin it down ("share_transfer · v3").
//	Status — an optional badge beside the subject.
//	Evidence — the source material behind the proposal, normally a quiet
//	    Disclosure, placed under the body and above the actions.
//	Actions — the explicit choices. Keep one filled control and make every
//	    option that needs words first open its own field.
type DecisionCardOpts struct {
	Subject      string
	Context      string
	Status       templ.Component
	Evidence     templ.Component
	Actions      templ.Component
	HeadingLevel int
	ID           string
	Class        string
	Attrs        templ.Attributes
}

func decisionCardHeadingID(o DecisionCardOpts) string {
	if o.ID == "" {
		return ""
	}
	return o.ID + "-subject"
}
