// Disclosure types (disclosure.templ).
package components

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// DisclosureVariant selects the frame a disclosure wears. The shape is always
// the same native <details>; only its weight changes, so one component covers
// a text link, an editing surface and a danger zone.
type DisclosureVariant string

const (
	// DisclosureDefault is a hairline-ruled row whose summary reads as a
	// heading — the "Posting details", "Audit trail", "Advanced" expander.
	DisclosureDefault DisclosureVariant = ""
	// DisclosurePanel frames the whole disclosure as a bordered surface and
	// turns the summary into its header, so an open editor reads as its own
	// surface without needing a second card around it.
	DisclosurePanel DisclosureVariant = "panel"
	// DisclosureQuiet reads as a text link until it is opened — for evidence
	// (a quote, a payload, a long explanation) that would otherwise crowd the
	// decision it belongs to.
	DisclosureQuiet DisclosureVariant = "quiet"
	// DisclosureDanger frames the block as a danger zone, so a destructive
	// control never needs a tinted slab in the main flow.
	DisclosureDanger DisclosureVariant = "danger"
)

// DisclosureOpts configures Disclosure.
//
//	Open — render the <details> expanded.
//	Variant — the frame; see DisclosureVariant.
//	Hint — muted text on the trailing edge of the summary, shown open or
//	    closed. Use it for a count or a consequence ("Creates a linked
//	    request", "4 fields").
//	Summary — a muted hint shown only while collapsed, for a value the open
//	    body repeats anyway ("created 3 Sep 2026").
//	Name — the native <details name> grouping: disclosures sharing a name
//	    behave as an exclusive accordion in browsers that support it.
//	BodyClass — extra classes on the body wrapper (a form grid, a flush
//	    table) so host layout does not need a second wrapper element.
//	SummaryID — an id on the <summary>, so a host can label a region by the
//	    disclosure heading (aria-labelledby) or target the trigger in a script.
//	Attrs — extra attributes on the <details> (data-* hooks a host script
//	    reads), spread last.
type DisclosureOpts struct {
	Open      bool
	Variant   DisclosureVariant
	Hint      string
	Summary   string
	Name      string
	BodyClass string
	SummaryID string
	ID        string
	Class     string
	Attrs     templ.Attributes
}

func disclosureClass(o DisclosureOpts) string {
	variant := ""
	if o.Variant != DisclosureDefault {
		variant = "disclosure--" + string(o.Variant)
	}
	return joinClass("disclosure", variant, o.Class)
}

// DisclosureBlock is Disclosure with its body taken from a templ children
// block, so a call site keeps an inline form in place:
//
//	@components.DisclosureBlock("Add a fee", components.DisclosureOpts{Variant: components.DisclosurePanel}) {
//		<form method="post">…</form>
//	}
//
// It only hands the block to Disclosure; frame, variants and open state are
// Disclosure's.
func DisclosureBlock(label string, opts DisclosureOpts) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		body := templ.GetChildren(ctx)
		if body == nil {
			body = templ.NopComponent
		}
		return Disclosure(label, opts, body).Render(templ.ClearChildren(ctx), w)
	})
}
