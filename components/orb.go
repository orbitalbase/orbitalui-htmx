// Orb — the AI presence mark: a living sphere in the AI gradient. Swap the
// state class client-side (is-idle / is-thinking / is-speaking) to reflect
// streaming; motion.css animates them and honours prefers-reduced-motion.
package components

import "strconv"

// OrbState selects the orb's motion. OrbStatic does not animate.
type OrbState string

const (
	OrbStatic   OrbState = ""
	OrbIdle     OrbState = "is-idle"
	OrbThinking OrbState = "is-thinking"
	OrbSpeaking OrbState = "is-speaking"
)

// OrbOpts configures Orb.
//
//	Size — pixels from 16/20/28/40/64/120 (default 28). 16 and 20 drop the
//	    glow, which only smears at that size.
//	State — OrbStatic, OrbIdle, OrbThinking or OrbSpeaking.
//	Label — an accessible name; "" keeps the orb decorative (aria-hidden),
//	    the right choice beside a visible label.
type OrbOpts struct {
	Size  int
	State OrbState
	Label string
	Class string
}

func (o OrbOpts) size() int {
	switch o.Size {
	case 16, 20, 28, 40, 64, 120:
		return o.Size
	}
	return 28
}

func (o OrbOpts) class() string {
	return joinClass("ui-orb", "ui-orb--"+strconv.Itoa(o.size()), modifier(o.size() <= 20, "ui-orb--xs"), string(o.State), o.Class)
}
