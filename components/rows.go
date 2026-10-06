// Row primitives (rows.templ): the one list shape a record workspace uses
// over and over — a leading mark, a title with one line of meta, and
// something on the trailing edge (a badge, a figure, a button row). An
// ActionRow with an Href becomes a whole-row link with a chevron; without
// one it is a plain row that carries its own controls.
//
// They own spacing, truncation and hit area only. Every slot takes a
// templ.Component so the host keeps ownership of content, endpoints and
// permissions.
package components

import "github.com/a-h/templ"

// ActionListVariant selects how the rows are separated.
type ActionListVariant string

const (
	// ActionListDivided is the default: hairline-separated rows carrying the
	// card's own gutter, for a list that runs edge to edge inside a Card with
	// Flush set.
	ActionListDivided ActionListVariant = ""
	// ActionListPlain drops the inline padding, for a list nested inside an
	// already-padded card body or disclosure.
	ActionListPlain ActionListVariant = "plain"
	// ActionListCards frames each row as its own bordered tile and lays the
	// tiles out in an auto-fitting grid — the "where to go next" link group.
	ActionListCards ActionListVariant = "cards"
)

// ActionListOpts configures ActionList.
//
//	Variant — divided (default), plain or cards; see ActionListVariant.
//	Label — when set, the list becomes a labelled navigation landmark.
//	Compact — a denser row height for long reference lists.
type ActionListOpts struct {
	Variant ActionListVariant
	Label   string
	Compact bool
	ID      string
	Class   string
}

func actionListClass(o ActionListOpts) string {
	variant := ""
	if o.Variant != ActionListDivided {
		variant = "ui-rows--" + string(o.Variant)
	}
	return joinClass("ui-rows", variant, modifier(o.Compact, "ui-rows--compact"), o.Class)
}

// ActionRowOpts configures ActionRow.
//
//	Title/Meta — the two lines of copy; Meta is one line of context, truncated
//	    on a desktop and wrapped on a phone.
//	Href — makes the whole row a link and adds the trailing chevron. Leave it
//	    empty for a row whose actions live in Trailing.
//	Icon/IconTone — a leading icon in a tinted tile; the tone tints the tile.
//	Leading — replaces Icon with host markup (an Avatar, a checkbox, a
//	    monogram). Icon is ignored when Leading is set.
//	Value — a right-aligned figure, rendered tabular so a column lines up.
//	Trailing — a badge, a button row or a menu on the trailing edge. On a
//	    linked row keep it to non-interactive content: a nested control inside
//	    an anchor is not reachable by keyboard in a predictable order.
//	Below — secondary content under the copy (a disclosure, a note), spanning
//	    the copy column. Rows with Below usually want Stacked.
//	Stacked — align the row's columns to the top instead of centring them.
//	Title stays plain text; a row that needs rich copy renders Below instead.
type ActionRowOpts struct {
	Title    string
	Meta     string
	Href     templ.SafeURL
	Icon     string
	IconTone Tone
	Leading  templ.Component
	Value    string
	Trailing templ.Component
	Below    templ.Component
	Stacked  bool
	Current  bool // marks the row as the current page in a navigation list
	ID       string
	Class    string
	Attrs    templ.Attributes
}

func actionRowItemClass(o ActionRowOpts) string {
	return joinClass("ui-rows__item", modifier(o.Stacked, "ui-rows__item--stacked"))
}

func actionRowClass(o ActionRowOpts) string {
	linked := modifier(o.Href != "", "ui-row--link")
	return joinClass("ui-row", linked, modifier(o.Current, "is-current"), o.Class)
}

func actionRowIconClass(o ActionRowOpts) string {
	tone := ""
	if o.IconTone != "" && o.IconTone != ToneNeutral {
		tone = "ui-row__icon--" + o.IconTone.Modifier()
	}
	return joinClass("ui-row__icon", tone)
}

func actionRowHasTrail(o ActionRowOpts) bool {
	return o.Value != "" || o.Trailing != nil || o.Href != ""
}

// Fact is one short term/value pair for FactList.
type Fact struct {
	Term  string
	Value string
}

// Criterion is one pass/fail line in a CriteriaList: the stated test, whether
// the record met it, and optionally the value that was read.
type Criterion struct {
	Met   bool
	Label string
	Note  string
}

func criterionClass(c Criterion) string {
	return joinClass("ui-criteria__item", modifier(c.Met, "is-met"))
}

// criterionPrefix is the screen-reader-only word that carries the verdict the
// tick or cross only shows. Colour and shape never carry meaning alone.
func criterionPrefix(c Criterion) string {
	if c.Met {
		return "Met: "
	}
	return "Not met: "
}

func criterionIcon(c Criterion) string {
	if c.Met {
		return "check"
	}
	return "close"
}

// ChoiceRowOpts configures ChoiceRow: one checkbox or radio framed as a row,
// with the sentence that explains what choosing it does.
//
//	Type — "checkbox" (default) or "radio".
//	Name/Value — native form wiring; Value may be empty for a lone checkbox.
//	Label — the choice itself; Description — the one line that explains it.
//	ID — set it when something else must point at the input (a field error).
type ChoiceRowOpts struct {
	Type        string
	Name        string
	Value       string
	Label       string
	Description string
	Checked     bool
	Required    bool
	Disabled    bool
	ID          string
	Class       string
	Attrs       templ.Attributes
}

func choiceRowType(o ChoiceRowOpts) string {
	if o.Type == "radio" {
		return "radio"
	}
	return "checkbox"
}

func choiceRowClass(o ChoiceRowOpts) string {
	return joinClass("ui-choice", modifier(o.Disabled, "is-disabled"), o.Class)
}
