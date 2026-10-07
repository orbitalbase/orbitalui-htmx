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
	if o.Wide {
		return "field field--wide"
	}
	return "field"
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
	if o.Class != "" {
		c += " " + o.Class
	}
	return c
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
	Class           string
	Attrs           templ.Attributes
}

func fieldTextareaClass(o FieldTextareaOpts) string {
	return fieldControlClass(FieldInputOpts{Invalid: o.Invalid, Class: o.Class})
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
	Class           string
	Attrs           templ.Attributes
}

func fieldSelectClass(o FieldSelectOpts) string {
	return fieldControlClass(FieldInputOpts{Invalid: o.Invalid, Class: o.Class})
}
