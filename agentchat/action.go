package agentchat

import (
	"net/http"
	"net/url"
)

// Request and response headers of action cards (forms with
// data-agentchat-submit="action").
const (
	// HeaderAction is sent by the runtime with every action form request.
	HeaderAction = "X-Agentchat-Action"
	// HeaderNote carries the outcome of an action; see SetNote.
	HeaderNote = "X-Agentchat-Note"
)

// IsActionRequest reports whether the runtime sent r for an action card.
func IsActionRequest(r *http.Request) bool { return r.Header.Get(HeaderAction) != "" }

// SetNote sets HeaderNote on an action response: the runtime adds text to the
// thread as a note and sends it with the next turn, so the agent learns the
// outcome. The text is percent-encoded because header values are not UTF-8.
func SetNote(w http.ResponseWriter, text string) {
	w.Header().Set(HeaderNote, url.PathEscape(text))
}
