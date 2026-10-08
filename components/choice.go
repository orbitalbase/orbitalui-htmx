// ChipGroup and SegmentedChoice types + helpers (choice.templ): the two
// native-input choice controls. Both are real radio/checkbox inputs, so they
// submit with their form, work without JavaScript and are keyboard operable
// out of the box; the visual state is CSS only (:checked, :has(:checked)).
package components

import (
	"strings"

	"github.com/a-h/templ"
)

// ChipOption is one chip in a ChipGroup.
type ChipOption struct {
	Value     string
	Label     string
	Icon      string // optional leading icon-set name
	Selected  bool
	Disabled  bool
	Suggested bool // assistant-suggested choice: AI tint and spark mark
}

// ChipGroupOpts configures a ChipGroup.
//
//	Name — form field name shared by every input.
//	Multiple — checkboxes (several chips) instead of radios (one chip).
//	Legend — the group's accessible name, shown above the chips unless
//	    HideLegend is set (it is then visually hidden but still announced).
//	Required — adds the * marker to the legend (display only; radios still
//	    need a required attribute from the host form to block submission).
//	Help — a muted hint under the chips. Error — an error under the chips,
//	    rendered like a field error (role alert) and announced with the group.
//	    Both are tied to the fieldset with aria-describedby (ids derive from
//	    Name: Name-help and Name-error).
//	Disabled — disables every chip.
//	Class — extra classes on the fieldset.
type ChipGroupOpts struct {
	Name       string
	Multiple   bool
	Legend     string
	HideLegend bool
	Required   bool
	Help       string
	Error      string
	Disabled   bool
	Class      string
	// SuggestedTitle is the hover title of the spark on a Suggested chip
	// (default "Suggested").
	SuggestedTitle string
}

// chipGroupDescribedBy lists the ids of the help and error paragraphs.
func chipGroupDescribedBy(o ChipGroupOpts) string {
	var ids []string
	if o.Error != "" {
		ids = append(ids, FieldErrorID(o.Name))
	}
	if o.Help != "" {
		ids = append(ids, o.Name+"-help")
	}
	return strings.Join(ids, " ")
}

func chipGroupClass(o ChipGroupOpts) string {
	return joinClass("ui-chip-group", ifClass(o.Error != "", "has-error"), o.Class)
}

func chipInputType(o ChipGroupOpts) string {
	if o.Multiple {
		return "checkbox"
	}
	return "radio"
}

func choiceChipClass(c ChipOption) string {
	return joinClass("ui-choice-chip", ifClass(c.Suggested, "is-suggested"))
}

// SegChoice is one option of a SegmentedChoice.
type SegChoice struct {
	Value    string
	Label    string
	Icon     string // optional leading icon-set name
	IconOnly bool   // hide the label visually (it stays the accessible name)
	Disabled bool
}

// SegmentedChoiceOpts configures a SegmentedChoice.
//
//	Name — form field name of the radio group.
//	AriaLabel — accessible name of the group (required for a11y).
//	Size — "" (30px) or "sm" (26px).
//	Class — extra classes on the group.
//	Attrs — extra attributes spread onto the group (data hooks).
type SegmentedChoiceOpts struct {
	Name      string
	AriaLabel string
	Size      string
	Class     string
	Attrs     templ.Attributes
}

var segChoiceCounts = [...]string{"ui-seg-choice--n2", "ui-seg-choice--n3", "ui-seg-choice--n4", "ui-seg-choice--n5", "ui-seg-choice--n6"}

// segChoiceClass carries the option count (the thumb is 1/n of the track;
// n is clamped to 2..6) and the size modifier.
func segChoiceClass(n int, o SegmentedChoiceOpts) string {
	if n < 2 {
		n = 2
	}
	if n > 6 {
		n = 6
	}
	return joinClass("ui-seg-choice", segChoiceCounts[n-2], ifClass(o.Size == "sm", "ui-seg-choice--sm"), o.Class)
}
