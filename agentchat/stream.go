package agentchat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrClientGone is returned by every Stream write after the client has gone
// (or an earlier write failed). The turn itself should keep running.
var ErrClientGone = errors.New("agentchat: client gone")

// ErrNoFlush is returned by NewStream when the response writer chain cannot
// flush, so events would be buffered until the handler returns.
var ErrNoFlush = errors.New("agentchat: response writer cannot flush")

const writeTimeout = 30 * time.Second

// heartbeat is the interval of the ": ping" comment that keeps proxies from
// closing a quiet stream. A variable so tests can shorten it.
var heartbeat = 15 * time.Second

// SessionEvent announces the chat session, usually on the first turn of a new chat.
//
//	SessionID — the id the runtime sends back with every later turn.
//	URL — a same-origin address to show in the address bar (history.replaceState).
//	Data — opaque app payload, delivered in the agentchat:session DOM event.
type SessionEvent struct {
	SessionID string `json:"session_id"`
	URL       string `json:"url,omitempty"`
	Data      any    `json:"data,omitempty"`
}

// DoneEvent ends a turn. HTML is authoritative: it replaces the provisional
// streamed text. CardHTML fills the card slot; Stopped marks a cancelled turn.
// Data is opaque app payload, delivered in the agentchat:done DOM event.
type DoneEvent struct {
	HTML     string `json:"html"`
	CardHTML string `json:"card_html,omitempty"`
	Stopped  bool   `json:"stopped,omitempty"`
	Data     any    `json:"data,omitempty"`
}

// ErrorEvent ends a turn with an error the user can see.
//
//	Retryable — show a Retry button that sends the turn again.
//	RestoreText — put the user's text back into an empty composer.
//	Stale, CardHTML — the card the user answered is out of date; swap in CardHTML.
type ErrorEvent struct {
	Message     string `json:"message"`
	Retryable   bool   `json:"retryable,omitempty"`
	RestoreText string `json:"restore_text,omitempty"`
	CardHTML    string `json:"card_html,omitempty"`
	Stale       bool   `json:"stale,omitempty"`
}

// Stream writes one turn to the browser as server-sent events. It is safe for
// concurrent use. Once a write fails the stream is dead: later writes do
// nothing and return ErrClientGone, while the app keeps running (and
// persisting) the turn — the library never cancels anything.
type Stream struct {
	mu   sync.Mutex
	w    http.ResponseWriter
	rc   *http.ResponseController
	dead bool
	stop chan struct{}
	once sync.Once
}

// IsStreamRequest reports whether the client asked for an event stream
// (Accept contains text/event-stream), as the runtime's turn request does.
func IsStreamRequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// NewStream starts an event stream on w: it sets the stream headers, sends
// ": open", flushes, and starts the heartbeat. The writer chain must be able
// to flush (http.Flusher, or Unwrap down to one); otherwise NewStream returns
// ErrNoFlush before writing anything. Call Close when the handler is done.
func NewStream(w http.ResponseWriter) (*Stream, error) {
	if !canFlush(w) {
		return nil, ErrNoFlush
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	s := &Stream{w: w, rc: http.NewResponseController(w), stop: make(chan struct{})}
	if err := s.write([]byte(": open\n\n")); err != nil {
		return nil, err
	}
	go s.beat(heartbeat)
	return s, nil
}

func canFlush(w http.ResponseWriter) bool {
	for {
		switch v := w.(type) {
		case http.Flusher:
			return true
		case interface{ FlushError() error }:
			return true
		case interface{ Unwrap() http.ResponseWriter }:
			w = v.Unwrap()
		default:
			return false
		}
	}
}

func (s *Stream) beat(every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			_ = s.write([]byte(": ping\n\n"))
		case <-s.stop:
			return
		}
	}
}

func (s *Stream) write(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead {
		return ErrClientGone
	}
	_ = s.rc.SetWriteDeadline(time.Now().Add(writeTimeout))
	if _, err := s.w.Write(b); err != nil {
		s.dead = true
		return fmt.Errorf("%w: %v", ErrClientGone, err)
	}
	if err := s.rc.Flush(); err != nil {
		s.dead = true
		return fmt.Errorf("%w: %v", ErrClientGone, err)
	}
	return nil
}

// Session sends a session event.
func (s *Stream) Session(p SessionEvent) error { return s.Event("session", p) }

// Step sends a step event: the agent started doing label.
func (s *Stream) Step(label string) error {
	return s.Event("step", struct {
		Label string `json:"label"`
	}{label})
}

// Delta sends a chunk of provisional reply text.
func (s *Stream) Delta(text string) error {
	return s.Event("delta", struct {
		Text string `json:"text"`
	}{text})
}

// Reset tells the runtime that the text streamed so far was narration before a
// tool call: it becomes the caption of the next step.
func (s *Stream) Reset() error { return s.Event("reset", struct{}{}) }

// Done sends the authoritative end of the turn.
func (s *Stream) Done(p DoneEvent) error { return s.Event("done", p) }

// Error sends a user-visible error that ends the turn.
func (s *Stream) Error(p ErrorEvent) error { return s.Event("error", p) }

// Event sends any event; the typed methods use it. name must be a single line.
func (s *Stream) Event(name string, payload any) error {
	if name == "" || strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("agentchat: invalid event name %q", name)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.write([]byte("event: " + name + "\ndata: " + string(data) + "\n\n"))
}

// Close stops the heartbeat and any further writes. Call it before the
// handler returns, so the heartbeat never touches a finished response. It is
// safe to call more than once.
func (s *Stream) Close() {
	s.once.Do(func() { close(s.stop) })
	s.mu.Lock()
	s.dead = true
	s.mu.Unlock()
}
