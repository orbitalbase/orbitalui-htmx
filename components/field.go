// Field types + helpers (field.templ) — the labelled-field wrapper, the
// standalone error paragraph and the shared field-control input, mirroring
// the v2 LabelledField + layout.css .field-control/field-help/field-error
// (docs/ui-v1-plan.md §6).
package components

import (
	"strconv"

	"github.com/a-h/templ"
)

// FieldOpts configures a LabelledField wrapper.
type FieldOpts struct {
	Help     string // muted hint under the control (markup-safe text)
	Error    string // field-level error, rendered red with a stable id
	Required bool   // adds the * marker to the label
	Wide     bool   // spans the whole grid row (.field--wide)
	// Suggested marks the value as pre-filled by an assistant: the label row
	// shows a SuggestedTag and the control takes the AI ring until edited.
	Suggested bool
	// SuggestedTitle is the tag's hover title (default "Suggested").
	SuggestedTitle string
}

// FieldErrorID returns the stable id of the error paragraph LabelledField
// renders for a control (htmlFor + "-error") — pass it to the control's
// aria-describedby so screen readers announce the error:
//
//	@components.FieldInput(components.FieldInputOpts{
//		ID: "j-date", AriaDescribedBy: components.FieldErrorID("j-date"), ...})
func FieldErrorID(htmlFor string) string { return htmlFor + "-error" }

// fieldClass builds the wrapper classes.
func fieldClass(o FieldOpts) string {
	c := "field"
	if o.Wide {
		c += " field--wide"
	}
	if o.Suggested {
		c += " is-suggested"
	}
	return c
}

// FieldInputOpts shapes the shared single-line control (field-control).
type FieldInputOpts struct {
	Type            string // "" → "text" | email | password | date | number | search …
	Name            string
	ID              string
	Value           string
	Placeholder     string
	Required        bool
	Disabled        bool
	Invalid         bool   // sets aria-invalid="true" + the error border class
	AriaDescribedBy string // FieldErrorID(id) when the field has an error
	Autocomplete    string
	InputMode       string
	MaxLength       int
	Pattern         string
	// Min, Max and Step bound number, date and time inputs ("" = unset), so
	// a signed amount or a date range needs no Attrs escape hatch.
	Min   string
	Max   string
	Step  string
	Class string // extra classes appended to .field-control
	// Icon is a leading icon-set name ("search", "mail", …). Icon, Prefix or
	// Suffix wrap the input in a .field-input shell that draws the ring, so
	// the affixes sit inside the control; without them the bare <input> is
	// emitted exactly as before.
	Icon string
	// Prefix and Suffix are inline affixes such as "$", "USD" or "%".
	Prefix string
	Suffix string
	// Suggested marks the value as assistant-filled (AI ring). Prefer
	// FieldOpts.Suggested when the control sits in a LabelledField.
	Suggested bool
	// Attrs are extra attributes spread onto the <input> (data-* markers
	// the client behaviours read, e.g. the create wizard's data-ftype on
	// custom form-field controls).
	Attrs templ.Attributes
}

// fieldControlType resolves the effective input type.
func fieldControlType(t string) string {
	switch t {
	case "text", "email", "password", "date", "number", "search", "tel", "url", "time", "month":
		return t
	case "":
		return "text"
	default:
		return t
	}
}

// fieldControlClass joins the base + extra + invalid classes.
func fieldControlClass(o FieldInputOpts) string {
	c := "field-control"
	if o.Invalid {
		c += " field-control--invalid"
	}
	if o.Suggested {
		c += " field-control--suggested"
	}
	if o.Class != "" {
		c += " " + o.Class
	}
	return c
}

// fieldHasShell reports whether the input needs the .field-input wrapper.
func fieldHasShell(o FieldInputOpts) bool {
	return o.Icon != "" || o.Prefix != "" || o.Suffix != ""
}

// fieldShellClass builds the wrapper classes for an input with an icon or
// affixes.
func fieldShellClass(o FieldInputOpts) string {
	return joinClass("field-input",
		ifClass(o.Icon != "", "has-icon"),
		ifClass(o.Prefix != "", "has-prefix"),
		ifClass(o.Suffix != "", "has-suffix"),
		ifClass(o.Invalid, "is-invalid"),
		ifClass(o.Suggested, "is-suggested"),
		ifClass(o.Disabled, "is-disabled"))
}

// ifClass returns name when cond holds, "" otherwise.
func ifClass(cond bool, name string) string {
	if cond {
		return name
	}
	return ""
}

// fieldMaxLengthAttr renders maxlength only when positive (0 = none).
func fieldMaxLengthAttr(o FieldInputOpts) (string, bool) {
	if o.MaxLength <= 0 {
		return "", false
	}
	return strconv.Itoa(o.MaxLength), true
}

// FieldTextareaOpts shapes the shared multi-line control. Compose it inside
// LabelledField for the label, help and error, exactly like FieldInput.
type FieldTextareaOpts struct {
	Name            string
	ID              string
	Value           string
	Placeholder     string
	Rows            int // 0 → 4
	MaxLength       int
	Required        bool
	Disabled        bool
	Invalid         bool
	AriaDescribedBy string
	// Autogrow lets the textarea grow with its content (data-autogrow;
	// CSS field-sizing where supported, the runtime fallback elsewhere).
	Autogrow  bool
	Suggested bool // assistant-filled value (AI ring)
	Class     string
	Attrs     templ.Attributes
}

func fieldTextareaClass(o FieldTextareaOpts) string {
	return fieldControlClass(FieldInputOpts{Invalid: o.Invalid, Suggested: o.Suggested, Class: o.Class})
}

func fieldTextareaRows(o FieldTextareaOpts) string {
	if o.Rows <= 0 {
		return "4"
	}
	return strconv.Itoa(o.Rows)
}

func fieldTextareaMaxLength(o FieldTextareaOpts) (string, bool) {
	return fieldMaxLengthAttr(FieldInputOpts{MaxLength: o.MaxLength})
}

// FieldSelectOpts shapes the shared native select. Options carry their own
// Value/Label; Selected picks the option whose Value matches. Placeholder,
// when set, renders a first empty option ("Choose an account…").
type FieldSelectOpts struct {
	Name            string
	ID              string
	Options         []SelectOption
	Selected        string
	Placeholder     string
	Required        bool
	Disabled        bool
	Invalid         bool
	AriaDescribedBy string
	Suggested       bool // assistant-filled value (AI ring)
	Class           string
	Attrs           templ.Attributes
}

func fieldSelectClass(o FieldSelectOpts) string {
	return fieldControlClass(FieldInputOpts{Invalid: o.Invalid, Suggested: o.Suggested, Class: o.Class})
}

// fieldSelectShellClass builds the chevron wrapper classes.
func fieldSelectShellClass(o FieldSelectOpts) string {
	return joinClass("field-select", ifClass(o.Invalid, "is-invalid"), ifClass(o.Suggested, "is-suggested"), ifClass(o.Disabled, "is-disabled"))
}

// suggestedTitle resolves the SuggestedTag hover title.
func suggestedTitle(t string) string {
	if t == "" {
		return "Suggested"
	}
	return t
}
