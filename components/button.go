// Button types + class helpers (button.templ). The Button is the v2
// ui/Button + .ui-button semantics ported to server HTML (docs/ui-v1-plan.md
// §6): variants primary/secondary/danger/quiet, sizes, loading spinner,
// optional href (renders an <a>), optional leading icon from the icon set
// and explicit htmx passthrough fields for partial behaviours.
package components

import (
	"strings"

	"github.com/a-h/templ"
)

// ButtonVariant is a .ui-button variant. Empty value = secondary (the v2
// default).
type ButtonVariant string

const (
	ButtonVariantPrimary   ButtonVariant = "primary" // the page's single main action (accent green)
	ButtonVariantSecondary ButtonVariant = "secondary"
	ButtonVariantDanger    ButtonVariant = "danger"
	ButtonVariantQuiet     ButtonVariant = "quiet"
	// ButtonVariantText is a submit control styled as a text link: no frame,
	// no height of its own. It is how a per-row mutation ("Mark read",
	// "Resend", "Revoke") stays a real form submission without reading as a
	// button and competing with the page's one primary action.
	ButtonVariantText ButtonVariant = "text"
)

// ButtonSize is a size preset; "" = md (the default height).
type ButtonSize string

const ButtonSizeSM ButtonSize = "sm"

// ButtonOpts carries everything except the variant/size/label:
//
//	Type/Name/Value/Form — native <button> wiring (Type "" → "button").
//	Disabled/Loading — Loading disables and shows the spinner + aria-busy.
//	Href/Target — when Href is set the Button renders an <a href> (v2 link
//	    buttons); type/disabled/loading are ignored for anchors.
//	Icon — icon-set name rendered before the label ("plus", "check", …).
//	HXGet/HXPost/… — htmx attributes passed through verbatim (empty = omit).
//	Attrs — extra attributes (data-open-dialog="…", …), spread last.
//	Class — extra CSS classes appended to the .ui-button list.
type ButtonOpts struct {
	Label    string // button text (escaped); may be empty for icon-only
	Icon     string // optional leading icon name (components/icons.go)
	Type     string // "" (button) | "submit" | "reset"
	Disabled bool
	Loading  bool
	Name     string // <button name> — payload key
	Value    string // <button value> — payload value
	Form     string // associate with a form elsewhere (form attribute)
	Href     templ.SafeURL
	Target   string

	// htmx passthrough (empty = attribute omitted).
	HXGet       string
	HXPost      string
	HXDelete    string
	HXTarget    string
	HXSwap      string
	HXTrigger   string
	HXIndicator string

	Class string
	Attrs templ.Attributes // extra data-* attributes, spread onto the element
}

// buttonClass builds the full class list: variant + optional sm size +
// caller extras.
func buttonClass(variant ButtonVariant, size ButtonSize, extra string) string {
	parts := []string{"ui-button"}
	switch variant {
	case ButtonVariantPrimary:
		parts = append(parts, "ui-button--primary")
	case ButtonVariantDanger:
		parts = append(parts, "ui-button--danger")
	case ButtonVariantQuiet:
		parts = append(parts, "ui-button--quiet")
	case ButtonVariantText:
		parts = append(parts, "ui-button--text")
	default:
		parts = append(parts, "ui-button--secondary")
	}
	if size == ButtonSizeSM {
		parts = append(parts, "ui-button--sm")
	}
	if strings.TrimSpace(extra) != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, " ")
}

// buttonType resolves the effective type attribute value.
func buttonType(t string) string {
	if t == "submit" || t == "reset" {
		return t
	}
	return "button"
}
