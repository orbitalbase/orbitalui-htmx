// Toasts (toast.templ): a fixed notification region plus the toast itself.
// Server-rendered toasts (a flash message) go inside ToastRegion's children;
// client-raised ones come from the runtime in assets/shell.js.
//
// Runtime contract (all plain DOM, so it works under Alpine's CSP build):
//
//	HX-Trigger: {"toast": {"tone": "success", "title": "Saved", "message": "Invoice 42 posted.", "auto": true}}
//	    htmx fires a "toast" event; the runtime clones the region's template
//	    into the stack. tone is neutral | info | success | warning | danger |
//	    ai ("positive" and "spark" are accepted aliases). A bare string,
//	    {"toast": "Saved"}, is the message. auto omitted = auto-dismiss for
//	    tones other than warning and danger, which stay until dismissed.
//	window.OrbitalShell.toast({tone, title, message, auto}) does the same from
//	    script (any host script, htmx event handlers, fetch callbacks).
//
// Auto-dismiss waits ~4s and pauses while the pointer is over the toast or
// focus is inside it; the close button, and Escape on a focused toast, dismiss
// at once. The stack is aria-live="polite"; a danger toast carries
// role="alert" so it interrupts. Reduced motion removes the enter/exit
// movement.
package components

// ToastOpts configures Toast.
//
//	Tone — neutral (default), info, success, warning, danger, ai.
//	Title — optional bold first line. Message — the body text.
//	Auto — the runtime dismisses it after ~4s (server-rendered toasts only;
//	    runtime-raised toasts take auto from their payload).
type ToastOpts struct {
	Tone    Tone
	Title   string
	Message string
	Auto    bool
	Class   string
}

// toastIcon maps a tone to its glyph.
func toastIcon(t Tone) string {
	switch t {
	case ToneSuccess:
		return "circle-check"
	case ToneWarning:
		return "triangle-alert"
	case ToneDanger:
		return "circle-x"
	case ToneAI:
		return "spark"
	}
	return "info"
}

func toastClass(o ToastOpts) string {
	tone := ""
	if o.Tone != "" && o.Tone != ToneNeutral {
		tone = "ui-toast--" + o.Tone.Modifier()
	}
	return joinClass("ui-toast", tone, o.Class)
}

func toastRole(o ToastOpts) string {
	if o.Tone == ToneDanger {
		return "alert"
	}
	return "status"
}

// toastTones are the tones the region ships icons for.
var toastTones = []Tone{ToneNeutral, ToneInfo, ToneSuccess, ToneWarning, ToneDanger, ToneAI}
