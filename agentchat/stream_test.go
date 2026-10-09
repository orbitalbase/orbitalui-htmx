package agentchat

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsStreamRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/turn", nil)
	if IsStreamRequest(r) {
		t.Fatal("a request without Accept is not a stream request")
	}
	r.Header.Set("Accept", "text/html, text/event-stream")
	if !IsStreamRequest(r) {
		t.Fatal("Accept text/event-stream is a stream request")
	}
}

func TestStreamHeadersAndFraming(t *testing.T) {
	rec := httptest.NewRecorder()
	s, err := NewStream(rec)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h := rec.Result().Header
	for k, want := range map[string]string{
		"Content-Type": "text/event-stream", "Cache-Control": "no-cache, no-transform",
		"X-Accel-Buffering": "no", "Connection": "keep-alive",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	for _, err := range []error{
		s.Session(SessionEvent{SessionID: "s1", URL: "/chat?id=s1", Data: map[string]int{"n": 1}}),
		s.Step("Reading"), s.Delta("Hi <b>"), s.Reset(),
		s.Done(DoneEvent{HTML: "<p>Hi</p>", CardHTML: "<form></form>", Data: "x"}),
		s.Error(ErrorEvent{Message: "no", Retryable: true, RestoreText: "again"}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	want := ": open\n\n" +
		"event: session\ndata: {\"session_id\":\"s1\",\"url\":\"/chat?id=s1\",\"data\":{\"n\":1}}\n\n" +
		"event: step\ndata: {\"label\":\"Reading\"}\n\n" +
		"event: delta\ndata: {\"text\":\"Hi \\u003cb\\u003e\"}\n\n" +
		"event: reset\ndata: {}\n\n" +
		"event: done\ndata: {\"html\":\"\\u003cp\\u003eHi\\u003c/p\\u003e\",\"card_html\":\"\\u003cform\\u003e\\u003c/form\\u003e\",\"data\":\"x\"}\n\n" +
		"event: error\ndata: {\"message\":\"no\",\"retryable\":true,\"restore_text\":\"again\"}\n\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body:\n%q\nwant:\n%q", got, want)
	}
	if !rec.Flushed {
		t.Error("events were not flushed")
	}
}

func TestStreamEventRejectsBadNamesAndPayloads(t *testing.T) {
	s, err := NewStream(httptest.NewRecorder())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Event("a\nb", nil); err == nil {
		t.Error("an event name with a newline must be rejected")
	}
	if err := s.Event("x", make(chan int)); err == nil {
		t.Error("an unmarshalable payload must be rejected")
	}
	if err := s.Event("custom", map[string]string{"k": "v"}); err != nil {
		t.Errorf("a custom event: %v", err)
	}
}

// wrapper hides Flush behind Unwrap, like a logging or gzip middleware.
type wrapper struct{ http.ResponseWriter }

func (w wrapper) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func TestStreamWorksThroughAnUnwrapWrapper(t *testing.T) {
	rec := httptest.NewRecorder()
	s, err := NewStream(wrapper{rec})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Delta("x"); err != nil {
		t.Fatal(err)
	}
	if !rec.Flushed || !strings.Contains(rec.Body.String(), "event: delta") {
		t.Errorf("flushed=%v body=%q", rec.Flushed, rec.Body.String())
	}
}

// bare has neither Flush nor Unwrap.
type bare struct{ http.ResponseWriter }

func TestNewStreamNeedsAFlushableWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, err := NewStream(bare{rec}); !errors.Is(err, ErrNoFlush) {
		t.Fatalf("err = %v, want ErrNoFlush", err)
	}
	if rec.Body.Len() != 0 || rec.Header().Get("Content-Type") != "" {
		t.Error("NewStream wrote before failing")
	}
}

// deadWriter fails every write after the first n.
type deadWriter struct {
	*httptest.ResponseRecorder
	ok int
}

func (d *deadWriter) Write(b []byte) (int, error) {
	if d.ok == 0 {
		return 0, errors.New("broken pipe")
	}
	d.ok--
	return d.ResponseRecorder.Write(b)
}

func TestStreamIsADeadNoOpAfterAFailedWrite(t *testing.T) {
	w := &deadWriter{ResponseRecorder: httptest.NewRecorder(), ok: 1}
	s, err := NewStream(w) // ": open" uses the one good write
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Delta("a"); !errors.Is(err, ErrClientGone) {
		t.Fatalf("first failed write: %v", err)
	}
	w.ok = 10 // the client "comes back": the stream must stay dead
	before := w.Body.String()
	if err := s.Done(DoneEvent{}); !errors.Is(err, ErrClientGone) {
		t.Fatalf("write after death: %v", err)
	}
	if w.Body.String() != before {
		t.Error("a dead stream wrote")
	}
}

func TestStreamHeartbeatAndClose(t *testing.T) {
	old := heartbeat
	heartbeat = 10 * time.Millisecond
	defer func() { heartbeat = old }()
	rec := httptest.NewRecorder()
	s, err := NewStream(rec)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(s.snapshot(rec), ": ping\n\n") {
		time.Sleep(5 * time.Millisecond)
	}
	if !strings.Contains(s.snapshot(rec), ": ping\n\n") {
		t.Fatal("no heartbeat was written")
	}
	s.Close()
	s.Close() // idempotent
	n := len(s.snapshot(rec))
	time.Sleep(40 * time.Millisecond)
	if len(s.snapshot(rec)) != n {
		t.Error("the heartbeat wrote after Close")
	}
	if err := s.Step("late"); !errors.Is(err, ErrClientGone) {
		t.Errorf("write after Close: %v", err)
	}
}

// snapshot reads the recorder body under the stream's lock, so the heartbeat
// goroutine and the test never touch it together.
func (s *Stream) snapshot(rec *httptest.ResponseRecorder) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rec.Body.String()
}
