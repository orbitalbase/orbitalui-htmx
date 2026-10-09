package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/a-h/templ"

	"github.com/orbitalbase/orbitalui-htmx/agentchat"
	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// chatStub records what the agent chat sent, so scripts/browser-check.mjs can
// assert on the requests the runtime made.
var chatStub struct {
	sync.Mutex
	requests []url.Values // every turn, in order
	cancels  int          // Stop presses that reached the server
	failed   bool         // the "fail" message has failed once
	resumes  []string     // session ids the resume page polled with
}

func resetChatStub() {
	chatStub.Lock()
	defer chatStub.Unlock()
	chatStub.requests, chatStub.cancels, chatStub.failed, chatStub.resumes = nil, 0, false, nil
}

func registerAgentChat(mux *http.ServeMux) {
	mux.HandleFunc("GET /agentchat", agentChatPage)
	mux.HandleFunc("GET /agentchat/empty", agentChatEmptyPage)
	mux.HandleFunc("GET /agentchat/resume", agentChatResumePage)
	mux.HandleFunc("GET /agentchat/resume-state", agentChatResumeState)
	mux.HandleFunc("POST /agentchat/turn", agentChatTurn)
	mux.HandleFunc("POST /agentchat/cancel", func(w http.ResponseWriter, r *http.Request) {
		chatStub.Lock()
		chatStub.cancels++
		chatStub.Unlock()
	})
	mux.HandleFunc("POST /agentchat/authorize", agentChatAuthorize)
	mux.HandleFunc("GET /agentchat/state", func(w http.ResponseWriter, r *http.Request) {
		chatStub.Lock()
		defer chatStub.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"requests": chatStub.requests, "cancels": chatStub.cancels, "resumes": chatStub.resumes})
	})
}

// agentChatPage serves two chats under Content-Security-Policy script-src
// 'self': a full page with header, rail, panel and an action card in its thread,
// and a compact panel chat that must not interfere with it.
func agentChatPage(w http.ResponseWriter, r *http.Request) {
	resetChatStub()
	w.Header().Set("Content-Security-Policy", "script-src 'self'")
	proposal := agentchat.Card(agentchat.CardOpts{Label: "Proposal", Body: seq(
		raw(`<form method="post" action="/agentchat/authorize" data-agentchat-submit="action" id="authorize"><p>Pay Acme RM 1,200?</p><input type="hidden" name="csrf" value="tok"/>`),
		c.Button(c.ButtonVariantPrimary, c.ButtonSizeSM, c.ButtonOpts{Type: "submit", Label: "Authorize"}),
		raw(`</form><form method="post" action="/agentchat/authorize?fail=1" data-agentchat-submit="action" id="refuse"><input type="hidden" name="csrf" value="tok"/>`),
		c.Button(c.ButtonVariantQuiet, c.ButtonSizeSM, c.ButtonOpts{Type: "submit", Label: "Refuse"}),
		raw(`</form>`),
	)})
	full := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/agentchat/turn", CancelURL: "/agentchat/cancel", Transcript: true, Label: "Assistant",
		Header: agentchat.Head(agentchat.HeadOpts{
			Title:   "Assistant",
			Leading: agentchat.Toggle(agentchat.ToggleOpts{Dock: agentchat.DockRail, Label: "Show chats"}),
			Meta:    agentchat.Toggle(agentchat.ToggleOpts{Dock: agentchat.DockPanel, Label: "Show details"}),
		}),
		Rail:  raw(`<a href="/agentchat" id="rail-link">Earlier chat</a> <a href="/agentchat?older=1" id="rail-link2">Older chat</a>`),
		Panel: raw(`<div class="ui-chat-panel__body" id="panel-body"><button type="button" id="panel-a">First</button> <button type="button" id="panel-b" data-agentchat-autofocus>Second</button></div>`),
		Thread: agentchat.Thread([]agentchat.Message{
			{Role: agentchat.RoleAssistant, Text: "Welcome back."},
			{Role: agentchat.RoleAssistant, Content: proposal},
		}),
		Composer: agentchat.Composer(agentchat.ComposerOpts{Placeholder: "Message", MaxLength: 200, Counter: true, Hidden: map[string]string{"scope": "all"}}),
	})
	compact := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/agentchat/turn", Variant: agentchat.VariantPanel, Label: "Quick ask",
		Empty: seq(
			agentchat.Hero(agentchat.HeroOpts{Title: "How can I help?", Body: "Ask anything."}),
			agentchat.Starters([]agentchat.Starter{{Title: "Import", Action: "import", Icon: "upload"}, {Title: "Say hello", Prompt: "hello", Icon: "spark"}}),
		),
		Thread:   agentchat.Thread(nil),
		Composer: agentchat.Composer(agentchat.ComposerOpts{Placeholder: "Ask", Hidden: map[string]string{"csrf": "c1"}, Notice: raw("Press Enter to send this")}),
	})
	render(w, r,
		raw(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Agent chat browser check</title>
<link rel="stylesheet" href="/static/tokens.css"><link rel="stylesheet" href="/static/motion.css"><link rel="stylesheet" href="/static/controls.css"><link rel="stylesheet" href="/static/theme.css"><link rel="stylesheet" href="/static/agent-chat.css">
<script src="/static/agent-chat.js" defer></script>
<style>body{margin:0}.host{display:flex;height:640px}</style>
</head><body><div class="host" id="host-full">`),
		full, raw(`</div><div class="host" id="host-compact">`), compact, raw(`</div></body></html>`),
	)
}

// agentChatTurn is the stub agent. The message picks the behaviour:
//
//	fail   an error with restore_text, once; afterwards a normal reply
//	slow   steps and deltas that go on until the client leaves (Stop)
//	other  session, three steps (one repeated), narration and a reset, a
//	       fourth step, delta chunks, then done with html and a question card
//
// A submitted question card (answer_questions=1) gets a short confirmation.
func agentChatTurn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	chatStub.Lock()
	chatStub.requests = append(chatStub.requests, r.PostForm)
	failNow := r.PostFormValue("message") == "fail" && !chatStub.failed
	if failNow {
		chatStub.failed = true
	}
	chatStub.Unlock()

	s, err := agentchat.NewStream(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer s.Close()
	pause := func(d time.Duration) bool {
		select {
		case <-time.After(d):
			return true
		case <-r.Context().Done():
			return false
		}
	}
	switch msg := r.PostFormValue("message"); {
	case failNow:
		s.Error(agentchat.ErrorEvent{Message: "The agent couldn't finish that reply.", Retryable: true, RestoreText: msg})
	case r.PostFormValue("answer_questions") == "1":
		s.Done(agentchat.DoneEvent{HTML: "<p>Thanks, I have your answers.</p>"})
	case msg == "slow":
		s.Session(agentchat.SessionEvent{SessionID: "s-1"})
		s.Step("Reading the profile…")
		for pause(100 * time.Millisecond) {
			if s.Delta("still thinking ") != nil {
				return
			}
		}
	default:
		s.Session(agentchat.SessionEvent{SessionID: "s-1", URL: "/agentchat?session_id=s-1", Data: map[string]string{"owner": "me"}})
		s.Step("Reading the profile…")
		s.Step("Checking listings")
		s.Step("Checking listings")
		s.Delta("Let me look at that.")
		s.Reset()
		s.Step("Drafting the answer")
		for _, chunk := range []string{"Here is ", "**your** ", "provisional ", "answer."} {
			if !pause(180 * time.Millisecond) {
				return
			}
			s.Delta(chunk)
		}
		pause(180 * time.Millisecond)
		card := agentchat.QuestionCard{Token: "tok-1", Title: "A few details", Questions: []agentchat.Question{
			{Key: "mode", Label: "Work arrangement", Type: agentchat.TypeSelect, Options: []string{"Remote", "Hybrid"}, Value: "Hybrid"},
			{Key: "min", Label: "Minimum pay", Type: agentchat.TypeNumber},
		}}
		html, _ := templ.ToGoHTML(r.Context(), agentchat.Markdown("Final **server** answer."))
		cardHTML, _ := templ.ToGoHTML(r.Context(), agentchat.QuestionForm(agentchat.QuestionCardView{SessionID: "s-1", Card: card}))
		s.Done(agentchat.DoneEvent{HTML: string(html), CardHTML: string(cardHTML)})
	}
}

// agentChatAuthorize is an action card endpoint: a receipt and an outcome note,
// or (fail=1) a refusal. Only requests the runtime marks as actions are served.
func agentChatAuthorize(w http.ResponseWriter, r *http.Request) {
	if !agentchat.IsActionRequest(r) || r.FormValue("csrf") != "tok" {
		http.Error(w, "not an action request", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("fail") == "1" {
		http.Error(w, "The proposal is already closed.", http.StatusUnprocessableEntity)
		return
	}
	agentchat.SetNote(w, "Payment authorised")
	receipt := agentchat.Card(agentchat.CardOpts{Label: "Receipt", Body: raw(`<p id="receipt">Authorised RM 1,200</p>`)})
	render(w, r, receipt)
}

// agentChatResumePage is a page rendered while a turn is still running: the
// runtime polls the state endpoint, then swaps in the finished rows and card.
func agentChatResumePage(w http.ResponseWriter, r *http.Request) {
	chatStub.Lock()
	chatStub.resumes = nil
	chatStub.Unlock()
	w.Header().Set("Content-Security-Policy", "script-src 'self'")
	chat := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/agentchat/turn", StateURL: "/agentchat/resume-state", SessionID: "s-r", InFlight: true, Label: "Resumed",
		Thread:   agentchat.Thread([]agentchat.Message{{Role: agentchat.RoleUser, Text: "Pending question", Context: raw(`<span id="from-chip">From the invoice page</span>`)}}),
		Composer: agentchat.Composer(agentchat.ComposerOpts{Placeholder: "Message"}),
	})
	render(w, r,
		raw(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Agent chat resume</title>
<link rel="stylesheet" href="/static/tokens.css"><link rel="stylesheet" href="/static/motion.css"><link rel="stylesheet" href="/static/controls.css"><link rel="stylesheet" href="/static/theme.css"><link rel="stylesheet" href="/static/agent-chat.css">
<script src="/static/agent-chat.js" defer></script>
<style>body{margin:0}.host{display:flex;height:640px}</style>
</head><body><div class="host">`), chat, raw(`</div></body></html>`))
}

// agentChatResumeState reports the turn as running once, then finished with
// bare rows (agentchat.Rows) and a card, the shape thread_html must have.
func agentChatResumeState(w http.ResponseWriter, r *http.Request) {
	chatStub.Lock()
	chatStub.resumes = append(chatStub.resumes, r.URL.Query().Get("session_id"))
	n := len(chatStub.resumes)
	chatStub.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if n < 2 {
		json.NewEncoder(w).Encode(map[string]any{"inflight": true})
		return
	}
	rows, _ := templ.ToGoHTML(r.Context(), agentchat.Rows([]agentchat.Message{
		{Role: agentchat.RoleUser, Text: "Pending question", Context: raw(`<span id="from-chip">From the invoice page</span>`)},
		{Role: agentchat.RoleAssistant, Text: "Resumed **answer**."},
	}))
	card, _ := templ.ToGoHTML(r.Context(), agentchat.QuestionForm(agentchat.QuestionCardView{SessionID: "s-r", Card: agentchat.QuestionCard{
		Token: "tok-r", Questions: []agentchat.Question{{Key: "note", Label: "Anything else?", Type: agentchat.TypeText}},
	}}))
	json.NewEncoder(w).Encode(map[string]any{"inflight": false, "thread_html": string(rows), "card_html": string(card)})
}

// agentChatEmptyPage is a chat that fills the viewport with nothing in the
// thread but a short hero and starters: the empty state must not scroll. A
// 100px host header leaves the thread only about 440px, as in a real app.
func agentChatEmptyPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", "script-src 'self'")
	chat := agentchat.Chat(agentchat.ChatOpts{
		TurnURL: "/agentchat/turn", Label: "Empty",
		Header: agentchat.Head(agentchat.HeadOpts{Title: "Assistant"}),
		Empty: seq(
			agentchat.Hero(agentchat.HeroOpts{Title: "How can I help?", Body: "Ask anything."}),
			agentchat.Starters([]agentchat.Starter{{Title: "One", Body: "First", Icon: "spark", Prompt: "1"}, {Title: "Two", Body: "Second", Icon: "target", Prompt: "2"}, {Title: "Three", Body: "Third", Icon: "upload", Prompt: "3"}, {Title: "Four", Body: "Fourth", Icon: "building", Prompt: "4"}}),
		),
		Thread:   agentchat.Thread(nil),
		Composer: agentchat.Composer(agentchat.ComposerOpts{Placeholder: "Message", Hint: "Private", Quick: agentchat.QuickPrompts([]agentchat.Prompt{{Label: "Quick", Prompt: "q"}})}),
	})
	render(w, r,
		raw(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Agent chat empty</title>
<link rel="stylesheet" href="/static/tokens.css"><link rel="stylesheet" href="/static/motion.css"><link rel="stylesheet" href="/static/controls.css"><link rel="stylesheet" href="/static/theme.css"><link rel="stylesheet" href="/static/agent-chat.css">
<script src="/static/agent-chat.js" defer></script>
<style>body{margin:0;height:100vh;box-sizing:border-box;padding-top:100px;display:flex}</style>
</head><body>`), chat, raw(`</body></html>`))
}
