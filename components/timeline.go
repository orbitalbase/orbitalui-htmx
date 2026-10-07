// Timeline — a vertical, hairline-connected list of events: versions, match
// events, work history. Compose Timeline with TimelineItem children.
package components

import "github.com/a-h/templ"

// TimelineItemOpts configures TimelineItem.
//
//	Title — the event; Href makes it a link; Mono sets it in the numeric face
//	    (version numbers).
//	Meta — right-aligned detail, usually a timestamp.
//	Current — emphasises the dot (the current version).
//	Tone — brand, accent, info, success, warning, danger or ai tint the dot;
//	    "" is a quiet neutral dot (the action colour when Current).
//
// The item's children render as the body text under the title row.
type TimelineItemOpts struct {
	Title   string
	Meta    string
	Current bool
	Tone    Tone
	Href    templ.SafeURL
	Mono    bool
	Class   string
}

func timelineItemClass(o TimelineItemOpts) string {
	tone := ""
	if m := visualTone(o.Tone); m != "" {
		tone = "ui-timeline__item--" + m
	}
	return joinClass("ui-timeline__item", tone, modifier(o.Current, "is-current"), o.Class)
}

func timelineTitleClass(o TimelineItemOpts) string {
	return joinClass("ui-timeline__title", modifier(o.Mono, "ui-timeline__title--mono"))
}
