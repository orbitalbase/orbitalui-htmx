# agentchat — shared agentic chat for orbitalui-htmx

Reference for `github.com/orbitalbase/orbitalui-htmx/agentchat`, `assets/agent-chat.js` and
`assets/agent-chat.css`. Consumers: Kindred (agentswarmv2), Cosec (cosec-agent), Ledger (ledger-agent).

## 1. Principle

orbitalui-htmx owns the **chat UI and the wire protocol**. It does **not** own the agent.

| Shared (orbitalui-htmx) | App |
|---|---|
| `agentchat` Go package: SSE `Stream` writer, view types, question-card parsing, safe Markdown | Agent loop, LLM provider, tools, step labels |
| templ components: chat frame, thread, messages, composer, starters, question form, card shell, runtime templates | Persistence, auth, CSRF, access rules |
| `assets/agent-chat.js`: one runtime (fetch + SSE parse, streaming render, steps, stop, retry, resume, card submit, autoscroll) | Mapping its DB rows to `agentchat.Message` |
| `assets/agent-chat.css`: Kindred's agent look on `--oui-*` tokens | Domain card bodies, action endpoints, header/rail/panel content |

**The contract is the wire protocol and the `data-agentchat-*` attributes.** Components are the convenient,
correct way to emit that markup. The runtime never knows what an app's card means.

Non-goals: no LLM client, no tool registry, no DB schema, no routes registered by the library.

Load order: `tokens.css`, `motion.css`, `controls.css`, `theme.css`, `agent-chat.css`; then `agent-chat.js`
(plain DOM, any order, no Alpine or htmx needed, CSP safe: no eval, no inline handlers).

## 2. Go package `agentchat`

### 2.1 Stream (server to browser)

```go
func IsStreamRequest(r *http.Request) bool              // Accept contains text/event-stream
func NewStream(w http.ResponseWriter) (*Stream, error)  // headers, ": open", flush, 15s ": ping" heartbeat
func (s *Stream) Session(p SessionEvent) error
func (s *Stream) Step(label string) error
func (s *Stream) Delta(text string) error
func (s *Stream) Reset() error
func (s *Stream) Done(p DoneEvent) error
func (s *Stream) Error(p ErrorEvent) error
func (s *Stream) Event(name string, payload any) error  // escape hatch, used by the typed methods
func (s *Stream) Close()                                 // stops the heartbeat; idempotent
var ErrClientGone, ErrNoFlush error
```

- Headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-transform`,
  `X-Accel-Buffering: no`, `Connection: keep-alive`.
- Flushing and per-write deadlines (30s) go through `http.ResponseController`, so wrapped writers only need
  `Unwrap()` or `Flush()`. `NewStream` returns `ErrNoFlush`, before writing anything, if the chain cannot flush.
- Concurrency-safe. After the first failed write the stream is dead; later writes are no-ops returning
  `ErrClientGone`. The **app must keep running the turn** (persist it): the library never cancels anything.
  Call `Close` before the handler returns.
- Each event is `event: <name>\ndata: <json>\n\n`.

Action cards (2.6) have two helpers: `IsActionRequest(r)` and `SetNote(w, text)`, with the header names
`HeaderAction` and `HeaderNote`.

### 2.2 Events (the protocol)

| event | payload | runtime behaviour |
|---|---|---|
| `session` | `SessionEvent{session_id, url?, data?}` | sets the session id (root `data-session-id` and every `input[name=session_id]` in the root), `history.replaceState(url)` when `url` is same-origin, dispatches `agentchat:session` |
| `step` | `{label}` | starts a step row (the previous active step becomes done). The same label again in a row gives `×N`. Dispatches `agentchat:step` |
| `delta` | `{text}` | appends to the provisional reply, rendered with the client Markdown subset once per animation frame, with a `.ui-caret` at the end |
| `reset` | `{}` | the text so far was narration before a tool call: it becomes the caption of the next step and the reply is cleared |
| `done` | `DoneEvent{html, card_html?, stopped?, data?}` | **authoritative**: replaces the provisional reply with `html`, folds the steps into an "N steps" toggle, replaces the card slot with `card_html` (cleared when absent), dispatches `agentchat:done` with the whole payload. `stopped` appends a "Stopped" mark |
| `error` | `ErrorEvent{message, retryable?, restore_text?, card_html?, stale?}` | see below; dispatches `agentchat:error` |

Error handling by case:

- A typed turn with `restore_text`: the user row and the reply row are removed, the text goes back into an empty
  composer, and an error row shows (with Retry when `retryable`; Retry sends the text again and clears the composer
  if it still holds that text).
- A typed turn without it: the error sits in the reply row (Retry only when `retryable`; the user row stays).
- A card turn: the card expands again and a standalone error row shows. With `stale` and `card_html` the outdated
  card is replaced by `card_html` instead.
- Retry shows only when `retryable` is true.

Rules:

- `delta` text is **always provisional**. Apps may rewrite the answer; `done.html` wins. Apps that must not show
  some text simply do not send those deltas.
- A stream that ends without `done` or `error`, and a 2xx response that is not `text/event-stream`, are retryable
  errors (the second also restores the text).
- `data` is opaque app payload, delivered in the DOM event. The runtime never reads it.
- The streaming reply row is `aria-busy="true"` with its body `aria-hidden` until `done`, so screen readers
  announce the final reply once.

### 2.3 Request (browser to server)

`POST <data-turn-url>` (or the form's `action`), `application/x-www-form-urlencoded`,
`Accept: text/event-stream`. Fields:

- `session_id` (may be empty) and `message` (the composer textarea's name).
- `turn_id`: a fresh UUID for every new turn (typed, prompt, card, `controller.send`). Retry reuses the id of the
  turn it retries, so the app can use `r.FormValue("turn_id")` as its idempotency key.
- `transcript`: only when the root has `data-agentchat-transcript`; JSON
  `[{"role":"user"|"assistant","text":"..."}]` of the completed turns on screen before this one.
- `notes`: JSON array of strings, the outcome notes queued by action cards since the last turn (2.6). They are
  put back in the queue if the turn fails.
- every field of the composer form except the message textarea (a CSRF token, a submission id) on **every** turn,
  whatever started it (typed, starter, quick prompt, card, `controller.send`); a field the turn already set, such
  as a card form's own, wins.
- every field of the submitting form (question answers, hidden fields, the pressed submit button), and for the
  composer its hidden fields.
- whatever an `agentchat:send` listener changes in `event.detail.body` (the `URLSearchParams`).

Errors **before** the stream starts: any non-2xx. The body may be `text/plain` (the message) or JSON
`{message, restore_text?, retryable?}`; `retryable` defaults to true for 5xx.

Optional endpoints (root attributes; omit to disable):

- `data-cancel-url`: `POST session_id` when the user presses Stop (sent with `keepalive`). The runtime always
  aborts its own fetch; the app should finish the turn and send `done` with `stopped` or nothing.
- `data-state-url`: `GET ?session_id=` returns `{inflight:bool, thread_html?, card_html?}`. Used when the page
  renders with `InFlight` (a reload mid-turn): polled every 2s (5 minutes at most). Once `inflight` is false the
  runtime sets the **inner HTML of the existing log** (`[data-agentchat-messages]`) to `thread_html`, so
  `thread_html` must be bare rows: render `agentchat.Rows(msgs)`, never `Thread` (which would nest a second log).
  `card_html` fills the card slot.

### 2.4 Views

```go
type Role string // RoleUser, RoleAssistant, RoleNote, RoleStopped, RoleAnswered
type Step struct{ Label string; Count int; Caption string }
type Message struct {
    ID      string          // data-message-id on the row
    Role    Role
    Text    string          // user text, note text, answered summary
    Content templ.Component // assistant body (app-rendered); nil renders Markdown(Text)
    Steps   []Step          // assistant only; the folded "N steps" list
    Title   string          // note title
}
```

### 2.5 Components (templ, package agentchat)

- `Chat(ChatOpts)`: the root `<section data-agentchat>` and frame. `ChatOpts`: `TurnURL`, `CancelURL`, `StateURL`,
  `SessionID`, `InFlight`, `AutoSend` (sent once on load), `Transcript`, `Label`, `Variant` (`""` full page |
  `VariantPanel`), `RailLabel`, `PanelLabel`, slots `Header`, `Rail`, `Panel`, `Thread`, `Empty`, `Card`,
  `Composer`, plus `Class` and `Attrs`. `InFlight` needs `StateURL`. The empty state is dropped when `InFlight` or
  `AutoSend` is set, and hidden by CSS once the thread has rows or the card slot has content. It renders the
  runtime `<template>`s itself and carries no global ids.
- `Head(HeadOpts{Title, Leading, Meta})`: a header with an orb and the status line the runtime keeps current.
- `Toggle(ToggleOpts{Dock, Label, Icon})`: opens and closes the rail (`DockRail`, icon `history`) or panel
  (`DockPanel`, icon `panel-right`; override with `Icon` or `RegisterIcons`). The panel docks from 1280px and the
  rail from 1600px; below that they slide over with a scrim. A toggle hides itself where its region is docked.
  A slide-over is a modal dialog: while open it has `role="dialog" aria-modal="true"` and its label, focus moves
  to its `[data-agentchat-autofocus]` element (else its first focusable), Tab and Shift+Tab stay inside, Escape,
  the scrim or the toggle close it, focus returns to the opener, and the toggles' `aria-expanded` follows. Docked,
  none of that applies (no role, no trap). Only one slide-over is open at a time.
- `Message.Context` and `UserMessageWith(text, context)`: a slot above a user bubble, right-aligned, for a hand-off
  chip; the app styles the content.
- `Thread([]Message)`: `role="log" aria-live="polite"` list around `Rows([]Message)`, the bare role switch
  (use it for `thread_html`); `UserMessage(text)`,
  `AssistantMessage(AssistantOpts{Content, Steps})`, `NoteMessage(title, text)`, `StoppedMessage()`,
  `AnsweredMessage(summary)` render the same rows.
- `Composer(ComposerOpts{Name, Label, Placeholder, Value, MaxLength, Counter, Notice, Hint, Tools, Quick, Hidden, Attrs})`:
  Enter sends, Shift+Enter adds a line, autogrow, a Stop button while busy. `MaxLength > 0` always sets the
  `maxlength` attribute; the visible `n / max` count shows only with `Counter`. `Notice` is a slot inside the box,
  under the textarea (for example "Press Enter to send this" under a prompt prefilled from a URL); the runtime
  removes it on the first edit and when a turn is sent.
- `QuickPrompts([]Prompt{Label, Prompt, Action, Icon})`, `Hero(HeroOpts{Title, Body})` and
  `Starters([]Starter{Eyebrow, Title, Body, Prompt, Action, Icon, Tone})`: an element with `data-agentchat-prompt`
  sends its value as the user's message. With `Action` set the button carries `data-agentchat-action=<name>`
  instead: clicking it sends nothing and dispatches a cancelable `agentchat:starter` with `{action}` for the app
  to handle (an upload picker, a dialog).
- `Card(CardOpts{Body, Label, Class})`: the card shell (`data-agentchat-card`). CSS hooks for bodies:
  `ui-chat-card__head`, `__title`, `__desc`, `__body`, `__foot`, `__spacer`.
- `QuestionForm(QuestionCardView)`: the agent-driven form (2.6).
- `Markdown(src)` and `RenderMarkdown(src) string`: Goldmark GFM, raw HTML dropped, link URLs limited to
  http, https, mailto and relative, images become links, external links `target=_blank rel="noopener noreferrer"`.

### 2.6 Interactive cards

Any `<form>` inside the chat root declares one submit mode:

- `data-agentchat-submit="turn"`: the runtime sends the form's fields as a **turn** (to the form `action` or the
  turn URL, the pressed submitter included) and streams the reply. A submitter with `formnovalidate` skips
  validation. The card collapses into an inert "Answered" row whose text is, in order: the submitter's
  `data-agentchat-summary`, the form's `data-agentchat-summary`, or `Label: value` for each
  `[data-agentchat-q-label]` element joined with " · ". A listener on `agentchat:summarize` may replace
  `event.detail.summary`. The Answered row stands in for a user bubble. A failed turn expands the card again.
  `data-agentchat-autosubmit` submits on the first chip change. Inputs with `data-agentchat-group` show thousands
  separators and are sent without them. The card is the closest `[data-agentchat-card]` (or the form).
- `data-agentchat-submit="action"`: the runtime `fetch`es the form (method and action as written, header
  `X-Agentchat-Action: 1`, same-origin credentials, buttons disabled meanwhile) **without** a turn. A 2xx HTML
  response replaces the closest `[data-agentchat-card]` (a receipt). A response header `X-Agentchat-Note: <text>`
  (percent-encoding allowed; use `SetNote`) adds a note row and queues the text in `notes` for the next turn.
  A non-2xx response puts its text in an inline error inside the card and re-enables the buttons.
  CSRF stays the app's (a normal form field).
- A form without the attribute behaves natively.

Question cards:

```go
type Question struct {
    Key, Label, Help, Type, Placeholder, Value string // Type: TypeButtons|TypeSelect|TypeMultiselect|TypeNumber|TypeText|TypeTextarea
    Options, Values []string
    Required bool
}
type QuestionCard struct{ Token, Title, Description string; Questions []Question }
type QuestionSchema struct {
    AllowedKeys  []string // nil = any [a-z0-9_]{1,40}
    MaxQuestions int      // default 8
}
func ParseQuestions(raw json.RawMessage, s QuestionSchema) (QuestionCard, error)
func QuestionToolSchema(s QuestionSchema) map[string]any
type QuestionCardView struct{ SessionID string; Card QuestionCard }
```

`raw` is the tool call's arguments object `{title?, description?, questions:[...]}`. Validation: keys from
`AllowedKeys` and unique; non-empty labels; known types; `buttons` only as the single question of a card;
choices need 2 to 20 distinct non-empty options; a suggested `value` must be one of the options (snapped to their
spelling; a list for `multiselect`) or a non-negative number. HTML entities in copy are unescaped (templ escapes
on output). The returned card has **no Token**: the app sets it (for example a hash of session and questions) and
refuses answers that come back with another one.

Rendering: every choice type is a `ChipGroup` (radio, or checkbox for `multiselect`); `number`, `text` and
`textarea` use `FieldInput` and `FieldTextarea` inside `LabelledField`; a card that is one unanswered `buttons`
question auto-submits. Fields are named `q_<key>`; the form also carries `session_id`, `answer_questions=1` and
`question_token`, and its Skip button `skip_questions=1`. The model never writes HTML.

### 2.7 DOM events (runtime to app), on the root, `bubbles: true`

| event | detail |
|---|---|
| `agentchat:send` | `{body, form}`: mutate `body` (`URLSearchParams`) before the request |
| `agentchat:session` | the session payload |
| `agentchat:step` | `{label, count}` |
| `agentchat:done` | the `done` payload; after a local Stop `{html:"", stopped:true}`; after a resumed turn lands, the state response plus `resumed:true` |
| `agentchat:error` | the `error` payload |
| `agentchat:summarize` | `{form, summary}`: replace `summary` |
| `agentchat:action` | `{form, response, note}` after an action card response |
| `agentchat:starter` | `{action}`: an action starter or quick prompt was clicked (cancelable; nothing is sent) |

`window.OrbitalChat = {init(root), get(root), _internals}`. `get` returns the controller, for app buttons and
app-run work such as an upload:

| member | behaviour |
|---|---|
| `send(text, extraFields?)` | sends a turn as if typed; a no-op while busy or held |
| `stop()` | the Stop button's action |
| `busy` | true while a turn runs or the chat is held |
| `note(title, text)` | appends a note row and returns a handle `{update(title, text), working(bool), remove()}`; `working(true)` gives the icon its breathing AI tint |
| `setCard(html)` | replaces the card slot; `""` clears it |
| `setSession(id, url?)` | sets the session id on the root and every `input[name=session_id]`; the address bar changes only when `url` (same-origin) is given |
| `hold(status?)` | makes the chat busy for the app's own work: composer inert, no Stop offered, `status` in the header. Returns `release()`; holds nest and `release` is idempotent |
`_internals` (`parseSSE`, `renderMarkdown`, `foldSteps`, `captionOf`, `trimDots`) is for tests.

### 2.8 Markup attributes the runtime reads

Root: `data-agentchat`, `data-turn-url`, `data-cancel-url`, `data-state-url`, `data-session-id`, `data-inflight`,
`data-autosend`, `data-agentchat-transcript`. Inside the root (all looked up by attribute, never by id):
`data-agentchat-thread`, `-col`, `-messages`, `-card-slot`, `-composer`, `-input`, `-send`, `-count`, `-quick`,
`-empty`, `-status`, `-orb`, `-newpill`, `-scrim`, `-rail`, `-panel`, `-toggle="rail|panel"`, `-prompt`,
`-submit`, `-autosubmit`, `-group`, `-q-label`, `-summary`, `-card`, `-steps-toggle`, `-pending`, and the
`<template data-agentchat-tpl="user|assistant|step|error|answered|note">` rows.

## 3. Assets

- `assets/agent-chat.js`: standalone IIFE, auto-inits every `[data-agentchat]` on load and on `htmx:afterSettle`,
  reloads on bfcache restore. Several roots can share a page.
- `assets/agent-chat.css`: `ui-chat-*` classes (the bubble and send button shadows use `--oui-sh-ink`, `-2` and `-3`, defined in `tokens.css` and identical in both themes), every value an `--oui-*` token (no hard-coded colour; a test
  enforces both). Dark mode, reduced motion, rail and panel docking, and the compact `ui-chat--panel` variant.
- Both are embedded in `assets.Files`. `node --test assets/*.test.mjs` runs the helper tests;
  `scripts/browser-check.mjs` drives the `/agentchat` page of `internal/browsercheck` in headless Chrome.

## 4. Removed components

The earlier `components` chat primitives (`ChatThread`, `SummaryCard`, `SummaryField`, `AssistantAvatar`,
`AssistantMessage`, `AssistantMessageContent`, `UserMessage`, `SuggestionChip`, `SuggestionSubmitChip`,
`LoadingSpinner`, `ChatComposer`, `TypingIndicator`, `StreamingCaret`) and their `theme.css` rules were removed in
favour of `agentchat`.

## 5. App adoption (summary)

- **Kindred**: keep `AgentPage` composition (header, chat list, live draft panel, uploads, owner/side) and render
  it through `agentchat.Chat` with the shared thread, composer and question form; rename `status` to `step`,
  `questions_html` to `card_html`; Kindred extras (`refresh`, `title`, `listing_html`, owner/side) move to `data`
  and its own DOM-event listeners. `ui/js/chat.js` shrinks to those listeners.
- **Cosec**: SSE branch in the `/ask` handler (keep the no-JS path), hold back deltas until a sentinel is ruled
  out, `done.html` is the rendered turn; the instrument becomes a `Card` body with a `turn` form.
- **Ledger**: agentd opt-in SSE relayed through `Stream`; page and panel use `agentchat.Chat`
  (`VariantPanel`, `Transcript`); `ProposalCard` becomes an `action` card whose endpoint returns a receipt and
  `SetNote`.
