// Modal + Confirm types (modal.templ, confirm.templ) — the native <dialog>
// pattern documented in docs/ui-components.md §3. Both shells use the
// .ui-dialog classes; behaviour (open/close/focus/backdrop/Esc) lives in
// assets/static/app.js under window.LedgerUI and is wired via data-*
// attributes — no modal framework, no JS in templates.
package components

import "github.com/a-h/templ"

// ModalOpts configures a ModalShell.
type ModalOpts struct {
	ID         string // dialog id — triggers open it via data-open-dialog={ID}
	Title      string // heading; the dialog's aria-labelledby target
	CloseLabel string // accessible name of the × button (default "Close")
	Class      string // extra classes on the <dialog> (e.g. a size preset)

	// Open renders the dialog already open — a full-page 422 re-render of a
	// form page (docs/ui-components.md §2a) keeps a modal's values and
	// errors visible. app.js promotes the freshly parsed non-modal dialog
	// to the real modal on load (promoteOpenDialog); JS-disabled renders
	// still show the open form in the page flow.
	Open bool
}

// ModalTitleID returns the id of a modal's h2 (ID + "__title") for
// aria-labelledby wiring.
func ModalTitleID(id string) string { return id + "__title" }

// KV is one hidden form input rendered by ConfirmButton (opts.Fields).
type KV struct {
	Name  string
	Value string
}

// ConfirmOpts configures ConfirmButton; the visible copy is passed as
// parameters (submitLabel/confirmLabel/subject/consequence).
type ConfirmOpts struct {
	ID string // dialog id (unique per instance); form id = ID + "-form"

	// Tone colours the trigger + confirm buttons: ToneDanger (default) or
	// TonePrimary-equivalent — any non-danger tone renders the primary
	// button (this component never mixes quiet/secondary confirmations).
	Tone Tone

	// Method/action of the real submit ("POST" default — destructive
	// actions are POST + PRG; docs/ui-components.md §2/§3).
	Method string
	Action templ.SafeURL // required

	// Fields are hidden inputs carried on the submit (record ids etc).
	Fields []KV

	// HX switches the form to an htmx partial submit (hx-post to Action,
	// swapping HXTarget). While a request is in flight the dialog's buttons
	// disable (hx-disabled-elt) and the confirm button shows BusyLabel.
	HX        bool
	HXTarget  string // required when HX
	HXSwap    string // optional htmx swap ("outerHTML", …)
	BusyLabel string // optional: replaces ConfirmLabel during the request

	CancelLabel string // default "Cancel"
	Class       string // extra classes on the trigger button
}

// confirmToneClass maps the tone to the button variant class used by the
// trigger AND the confirm button. Destructive confirmations are the default
// (v2 semantics — danger); an explicit success/info/neutral tone renders the
// primary action for positive confirms (Post, Approve, …).
func confirmToneClass(o ConfirmOpts) string {
	if o.Tone == "" || o.Tone == ToneDanger {
		return "ui-button--danger"
	}
	return "ui-button--primary"
}

// confirmButtonClass builds the full class list for the trigger + confirm
// buttons.
func confirmButtonClass(o ConfirmOpts) string {
	c := "ui-button " + confirmToneClass(o)
	if o.Class != "" {
		c += " " + o.Class
	}
	return c
}

// confirmFormID returns the submit form id for a dialog id — pages reference
// it on any extra inputs they place inside the dialog:
//
//	<input name="reason" form={ components.ConfirmFormID("void-1") } .../>
func ConfirmFormID(id string) string { return id + "-form" }

// confirmMethod normalizes the form method.
func confirmMethod(o ConfirmOpts) string {
	if o.Method == "get" || o.Method == "GET" {
		return "get"
	}
	return "post"
}

// confirmCancelLabel resolves the cancel button label.
func confirmCancelLabel(o ConfirmOpts) string {
	if o.CancelLabel != "" {
		return o.CancelLabel
	}
	return "Cancel"
}

// confirmDisabledAttr is the hx-disabled-elt selector: every button in the
// dialog disables while the partial request is in flight.
func confirmDisabledAttr(o ConfirmOpts) string {
	if !o.HX {
		return ""
	}
	return "#" + o.ID + " button"
}
