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
	// ButtonVariantTextDanger is the text variant for a destructive per-row
	// mutation ("Remove", "Revoke"): still a quiet link, but red on hover
	// and focus so the consequence is legible before the click.
	ButtonVariantTextDanger ButtonVariant = "text-danger"
	// ButtonVariantAI is the agent entry point: a secondary surface with an
	// AI-tinted icon that picks up the AI ring and glow on hover and focus.
	// Use it for every control that starts or invokes the assistant, and
	// nowhere else.
	ButtonVariantAI ButtonVariant = "ai"
)

// ButtonSize is a size preset; "" = md (36px). sm is 30px and lg is 44px;
// on a phone every size grows to the 44px touch target.
type ButtonSize string

const (
	ButtonSizeSM ButtonSize = "sm"
	ButtonSizeLG ButtonSize = "lg"
)

// ButtonOpts carries everything except the variant/size/label:
//
//	Type/Name/Value/Form — native <button> wiring (Type "" → "button").
//	Disabled/Loading — Loading disables and shows the spinner + aria-busy.
//	Href/Target — when Href is set the Button renders an <a href> (v2 link
//	    buttons); type/disabled/loading are ignored for anchors.
//	Icon — icon-set name rendered before the label ("plus", "check", …).
//	IconRight — put the icon after the label ("arrow-right" on a next step).
//	IconOnly — a square icon button: the label is not drawn but becomes the
//	    accessible name (aria-label and title). Requires Icon and Label.
//	Block — fill the container's width (forms on a phone, dialog footers).
//	HXGet/HXPost/… — htmx attributes passed through verbatim (empty = omit).
//	Attrs — extra attributes (data-open-dialog="…", …), spread last.
//	Class — extra CSS classes appended to the .ui-button list.
type ButtonOpts struct {
	Label string // button text (escaped); may be empty for icon-only
	Icon  string // optional leading icon name (components/icons.go)
	// IconRight moves Icon after the label; IconOnly draws only the icon and
	// uses Label as the accessible name; Block fills the container's width.
	IconRight bool
	IconOnly  bool
	Block     bool
	Type      string // "" (button) | "submit" | "reset"
	Disabled  bool
	Loading   bool
	Name      string // <button name> — payload key
	Value     string // <button value> — payload value
	Form      string // associate with a form elsewhere (form attribute)
	Href      templ.SafeURL
	Target    string

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

// buttonClass builds the full class list: variant, size, layout modifiers
// and caller extras.
func buttonClass(variant ButtonVariant, size ButtonSize, o ButtonOpts) string {
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
	case ButtonVariantTextDanger:
		parts = append(parts, "ui-button--text", "ui-button--text-danger")
	case ButtonVariantAI:
		parts = append(parts, "ui-button--ai")
	default:
		parts = append(parts, "ui-button--secondary")
	}
	switch size {
	case ButtonSizeSM:
		parts = append(parts, "ui-button--sm")
	case ButtonSizeLG:
		parts = append(parts, "ui-button--lg")
	}
	if o.IconOnly {
		parts = append(parts, "ui-button--icon")
	}
	if o.Block {
		parts = append(parts, "ui-button--block")
	}
	if o.IconRight && !o.IconOnly {
		parts = append(parts, "ui-button--icon-right")
	}
	if strings.TrimSpace(o.Class) != "" {
		parts = append(parts, o.Class)
	}
	return strings.Join(parts, " ")
}

// buttonAriaLabel is the accessible name an icon-only button needs: its
// Label, unless the caller already supplied aria-label through Attrs.
func buttonAriaLabel(o ButtonOpts) string {
	if !o.IconOnly || o.Label == "" {
		return ""
	}
	if _, ok := o.Attrs["aria-label"]; ok {
		return ""
	}
	return o.Label
}

// buttonShowsLabel reports whether the visible label span is drawn.
func buttonShowsLabel(o ButtonOpts) bool { return o.Label != "" && !o.IconOnly }

// buttonType resolves the effective type attribute value.
func buttonType(t string) string {
	if t == "submit" || t == "reset" {
		return t
	}
	return "button"
}

// buttonTrailingIcon renders the icon after the label for IconRight buttons
// and nothing otherwise. It is a component call rather than a trailing if
// block so the default markup carries no extra whitespace node.
func buttonTrailingIcon(o ButtonOpts) templ.Component {
	if o.Icon == "" || !o.IconRight {
		return templ.NopComponent
	}
	return Icon(o.Icon)
}
