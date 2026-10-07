// RingMeter — a circular score: a track, an arc that draws in, and a centre
// value. The arc is plain SVG attributes (pathLength 100, dash array and
// offset), so it works under a strict style-src Content-Security-Policy.
package components

import (
	"fmt"
	"math"
	"strconv"
)

// RingMeterOpts configures RingMeter.
//
//	Value — 0..100; clamped.
//	Size — pixel size from 32/40/48/56/64/80/96/120 (default 56).
//	Tone — "" reads the primary-action colour; ai draws the arc in the AI
//	    gradient (brand, AI hue, accent); other tones tint it.
//	Label — replaces the centre number ("7/9").
//	Name — accessible name (default "Score"); the value is appended.
type RingMeterOpts struct {
	Value int
	Size  int
	Tone  Tone
	Label string
	Name  string
	Class string
}

func (o RingMeterOpts) size() int {
	switch o.Size {
	case 32, 40, 48, 56, 64, 80, 96, 120:
		return o.Size
	}
	return 56
}

func (o RingMeterOpts) stroke() float64 {
	switch s := o.size(); {
	case s >= 96:
		return 7
	case s >= 64:
		return 5
	case s >= 44:
		return 4
	}
	return 3
}

func (o RingMeterOpts) radius() float64 { return (float64(o.size()) - o.stroke()) / 2 }

func (o RingMeterOpts) percent() int { return clampPercent(o.Value) }

func (o RingMeterOpts) centre() string {
	if o.Label != "" {
		return o.Label
	}
	return strconv.Itoa(o.percent())
}

func (o RingMeterOpts) aria() string {
	name := o.Name
	if name == "" {
		name = "Score"
	}
	return fmt.Sprintf("%s: %d percent", name, o.percent())
}

func (o RingMeterOpts) ai() bool { return visualTone(o.Tone) == "ai" }

func (o RingMeterOpts) class() string {
	tone := "action"
	if m := visualTone(o.Tone); m != "" {
		tone = m
	}
	return joinClass("ui-ring", "ui-ring--"+strconv.Itoa(o.size()), "ui-ring--"+tone, o.Class)
}

// gradientID is shared by every AI ring of a size: the definitions are
// identical, so one id per size is enough and needs no counter.
func (o RingMeterOpts) gradientID() string { return "oui-ring-grad-" + strconv.Itoa(o.size()) }

func (o RingMeterOpts) viewBox() string {
	return fmt.Sprintf("0 0 %d %d", o.size(), o.size())
}

func (o RingMeterOpts) centreXY() string { return fmt.Sprintf("%.1f", float64(o.size())/2) }

func (o RingMeterOpts) r() string { return fmt.Sprintf("%.2f", o.radius()) }

func (o RingMeterOpts) strokeWidth() string { return fmt.Sprintf("%.0f", o.stroke()) }

func (o RingMeterOpts) rotate() string {
	c := float64(o.size()) / 2
	return fmt.Sprintf("rotate(-90 %.1f %.1f)", c, c)
}

func (o RingMeterOpts) dashOffset() string {
	return strconv.FormatFloat(math.Round((100-float64(o.percent()))*100)/100, 'f', -1, 64)
}
