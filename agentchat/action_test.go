package agentchat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestActionRequestAndNote(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/authorize", nil)
	if IsActionRequest(r) {
		t.Fatal("a plain request is not an action request")
	}
	r.Header.Set(HeaderAction, "1")
	if !IsActionRequest(r) {
		t.Fatal("X-Agentchat-Action marks an action request")
	}
	w := httptest.NewRecorder()
	SetNote(w, "Paid RM 1,200 to Café Müller")
	if got := w.Header().Get(HeaderNote); got != "Paid%20RM%201%2C200%20to%20Caf%C3%A9%20M%C3%BCller" {
		t.Errorf("note header = %q", got)
	}
}
