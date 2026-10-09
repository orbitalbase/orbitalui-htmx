// agent-chat.js — the agentic chat runtime. Plain DOM, no Alpine or htmx
// dependency, no eval and no inline handlers (CSP safe). It finds every
// [data-agentchat] root (rendered by agentchat.Chat), sends turns as
// POST + server-sent events, and streams the reply into the thread.
//
//   window.OrbitalChat.init(root)  — bind a root (done automatically)
//   window.OrbitalChat.get(root)   — its controller: send(text, extra), stop(), busy
//
// The wire protocol, the data-agentchat-* attributes and the agentchat:*
// events are documented in docs/agent-chat.md. Everything is looked up inside
// the root by data attribute, so several chats can share a page.
(function (global) {
  "use strict";

  const GENERIC_ERROR = "The agent couldn't finish that reply. Try again.";
  const OFFLINE_ERROR = "Couldn't reach the agent. Check your connection and try again.";
  const DROPPED_ERROR = "The connection dropped before the reply finished.";
  const NEAR_BOTTOM = 96;
  const FOCUSABLE = 'a[href], button:not(:disabled), input:not([type=hidden]):not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])';

  /* ---- pure helpers (exposed on OrbitalChat._internals for tests) ---------------------------------- */

  // Minimal server-sent-events parser for fetch streams. Feed it decoded text chunks; it calls
  // onEvent(name, data) for each complete event whose data is JSON. Comment lines (": ping") are ignored.
  function parseSSE(onEvent) {
    let buffer = "";
    return function feed(chunk) {
      buffer = (buffer + chunk).replace(/\r\n?/g, "\n");
      let end;
      while ((end = buffer.indexOf("\n\n")) >= 0) {
        const block = buffer.slice(0, end);
        buffer = buffer.slice(end + 2);
        let name = "message";
        const data = [];
        for (const line of block.split("\n")) {
          if (!line || line.startsWith(":")) continue;
          const colon = line.indexOf(":");
          const field = colon < 0 ? line : line.slice(0, colon);
          let value = colon < 0 ? "" : line.slice(colon + 1);
          if (value.startsWith(" ")) value = value.slice(1);
          if (field === "event") name = value;
          else if (field === "data") data.push(value);
        }
        if (!data.length) continue;
        let parsed;
        try {
          parsed = JSON.parse(data.join("\n"));
        } catch (e) {
          continue;
        }
        onEvent(name, parsed);
      }
    };
  }

  // A small Markdown renderer for text that is still streaming. It mirrors what the server's Markdown
  // pass produces for the subset assistants use (paragraphs, lists, headings, code, bold, italic, links)
  // so the final swap to server HTML does not shift the layout. Input is always escaped first.
  const escapeHTML = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");

  function inline(raw) {
    const codes = [];
    let s = escapeHTML(raw).replace(/`([^`\n]+)`/g, (_, code) => `\u0000${codes.push(code) - 1}\u0000`);
    s = s.replace(/\[([^\]\n]+)\]\((https?:\/\/[^\s)]+|mailto:[^\s)]+)\)/g, (_, label, href) =>
      `<a href="${href}"${href.startsWith("http") ? ' target="_blank" rel="noopener noreferrer"' : ""}>${label}</a>`);
    s = s.replace(/\*\*([^*\n]+)\*\*/g, "<strong>$1</strong>").replace(/__([^_\n]+)__/g, "<strong>$1</strong>");
    s = s.replace(/(^|[^*\w])\*([^*\s][^*\n]*?)\*(?!\*)/g, "$1<em>$2</em>");
    return s.replace(/\u0000(\d+)\u0000/g, (_, i) => `<code>${codes[Number(i)]}</code>`);
  }

  function renderMarkdown(text) {
    const lines = text.replace(/\r\n?/g, "\n").split("\n");
    const out = [];
    let para = [];
    let list = null;
    let fence = null;
    const flushPara = () => {
      if (para.length) out.push(`<p>${para.map(inline).join("<br>")}</p>`);
      para = [];
    };
    const flushList = () => {
      if (list) out.push(`<${list.tag}>${list.items.map((i) => `<li>${inline(i)}</li>`).join("")}</${list.tag}>`);
      list = null;
    };
    for (const line of lines) {
      if (fence) {
        if (/^```/.test(line)) {
          out.push(`<pre><code>${escapeHTML(fence.join("\n"))}</code></pre>`);
          fence = null;
        } else fence.push(line);
        continue;
      }
      if (/^```/.test(line)) {
        flushPara();
        flushList();
        fence = [];
        continue;
      }
      const bullet = /^\s*[-*+]\s+(.*)$/.exec(line);
      const numbered = /^\s*\d+[.)]\s+(.*)$/.exec(line);
      const heading = /^(#{1,4})\s+(.*)$/.exec(line);
      if (bullet || numbered) {
        flushPara();
        const tag = bullet ? "ul" : "ol";
        if (!list || list.tag !== tag) {
          flushList();
          list = { tag, items: [] };
        }
        list.items.push((bullet || numbered)[1]);
      } else if (heading) {
        flushPara();
        flushList();
        out.push(`<h${heading[1].length}>${inline(heading[2])}</h${heading[1].length}>`);
      } else if (!line.trim()) {
        // A blank line inside a list keeps it open (a loose list); the next non-list line closes it.
        flushPara();
      } else {
        flushList();
        para.push(line);
      }
    }
    if (fence) out.push(`<pre><code>${escapeHTML(fence.join("\n"))}</code></pre>`);
    flushPara();
    flushList();
    return out.join("");
  }

  const trimDots = (label) => label.replace(/[….]+$/, "").trim();

  // The text the model wrote before a tool call becomes the next step's caption: plain, one line, short.
  function captionOf(text, max = 96) {
    const plain = text.replace(/[*_`#>[\]]/g, "").replace(/\s+/g, " ").trim();
    return plain.length > max ? `${plain.slice(0, max - 3).trim()}…` : plain;
  }

  // Folds step events [{label, caption?}] into display steps [{label, count, caption}]: the same step again
  // in a row (parallel lookups) is one line with a count, unless it carries its own caption.
  function foldSteps(events) {
    const out = [];
    for (const e of events) {
      const prev = out[out.length - 1];
      if (prev && prev.label === e.label && !e.caption) prev.count += 1;
      else out.push({ label: e.label, count: 1, caption: e.caption || "" });
    }
    return out;
  }

  // A fresh id per turn, for the app's idempotency key.
  function newTurnId() {
    const crypto = global.crypto;
    if (crypto && crypto.randomUUID) return crypto.randomUUID();
    const bytes = new Uint8Array(16);
    if (crypto && crypto.getRandomValues) crypto.getRandomValues(bytes);
    else for (let i = 0; i < 16; i++) bytes[i] = Math.floor(Math.random() * 256);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  }

  const groupNumber = (v) => {
    const n = Number(v.replace(/[,\s]/g, ""));
    return v.trim() !== "" && Number.isFinite(n) ? n.toLocaleString("en-US", { maximumFractionDigits: 6 }) : v;
  };

  // X-Agentchat-Note may be percent-encoded (headers are Latin-1); plain text passes through.
  function decodeNote(value) {
    if (!value) return "";
    try {
      return decodeURIComponent(value).trim();
    } catch (e) {
      return value.trim();
    }
  }

  /* ---- one chat root ------------------------------------------------------------------------------- */

  const controllers = new WeakMap();
  const reduceMotion = () => !!(global.matchMedia && global.matchMedia("(prefers-reduced-motion: reduce)").matches);

  function init(root) {
    if (controllers.has(root)) return controllers.get(root);
    const $ = (name) => root.querySelector(`[data-agentchat-${name}]`);
    const thread = $("thread");
    const col = $("col");
    const messages = $("messages");
    const slot = $("card-slot");
    const form = $("composer");
    const input = $("input");
    if (!thread || !col || !messages || !slot || !form || !input) return null;
    const status = $("status");
    const pill = $("newpill");
    const sendButton = $("send");
    const count = $("count");
    const headOrb = root.querySelector("[data-agentchat-orb] .ui-orb");

    const st = { sessionId: root.dataset.sessionId || "", busy: false, held: 0, abort: null, stick: true, last: null, turn: null, resume: null, statusText: "Ready", notes: [] };
    // Returns false when a listener cancelled the event.
    const emit = (name, detail, cancelable = false) => root.dispatchEvent(new CustomEvent(`agentchat:${name}`, { bubbles: true, cancelable, detail }));
    const working = () => st.busy || st.held > 0;
    const template = (name) => root.querySelector(`template[data-agentchat-tpl="${name}"]`).content.firstElementChild.cloneNode(true);

    /* scrolling */
    const nearBottom = () => thread.scrollHeight - thread.scrollTop - thread.clientHeight < NEAR_BOTTOM;
    function scrollEnd({ smooth = false, force = false } = {}) {
      if (force) st.stick = true;
      if (!st.stick) {
        if (st.busy && pill) pill.hidden = false;
        return;
      }
      thread.scrollTo({ top: thread.scrollHeight, behavior: smooth && !reduceMotion() ? "smooth" : "auto" });
    }
    thread.addEventListener("scroll", () => {
      st.stick = nearBottom();
      if (st.stick && pill) pill.hidden = true;
    }, { passive: true });
    if (pill) {
      pill.addEventListener("click", () => {
        pill.hidden = true;
        scrollEnd({ smooth: true, force: true });
      });
    }
    scrollEnd({ force: true });

    /* header status, orb, composer state */
    function setStatus(text) {
      if (!status || text === st.statusText) return;
      st.statusText = text;
      status.querySelectorAll(".ui-chat-status__text").forEach((old) => {
        old.classList.remove("is-in");
        old.classList.add("is-out");
        setTimeout(() => old.remove(), 200);
      });
      const next = document.createElement("span");
      next.className = "ui-chat-status__text is-in";
      next.textContent = text;
      status.append(next);
    }

    function setOrb(state) {
      if (!headOrb) return;
      headOrb.classList.remove("is-idle", "is-thinking", "is-speaking");
      headOrb.classList.add(state);
    }

    // The chat is busy while a turn runs (Stop offered) or while the app holds it for its own work (nothing to stop).
    function renderBusy() {
      form.dataset.state = st.busy ? "busy" : st.held ? "held" : "idle";
      form.inert = !st.busy && st.held > 0;
      root.classList.toggle("is-busy", working());
      if (sendButton) {
        sendButton.setAttribute("aria-label", st.busy ? "Stop" : "Send message");
        sendButton.title = st.busy ? "Stop" : "";
      }
      setOrb(working() ? "is-thinking" : "is-idle");
      if (!working()) setStatus("Ready");
    }
    function setBusy(busy) {
      st.busy = busy;
      renderBusy();
    }

    function autogrow() {
      input.style.height = "auto";
      input.style.height = `${Math.min(input.scrollHeight, 220)}px`;
    }
    function updateCount() {
      if (count && input.maxLength > 0) count.textContent = `${input.value.length} / ${input.maxLength}`;
    }
    function setInput(value) {
      input.value = value;
      autogrow();
      updateCount();
    }
    // The composer notice ("Press Enter to send this") goes once the user edits or sends.
    const dropNotice = () => {
      const notice = $("notice");
      if (notice) notice.remove();
    };
    input.addEventListener("input", () => {
      autogrow();
      updateCount();
      dropNotice();
    });

    /* thread building */
    function hideEmpty() {
      const empty = $("empty");
      if (empty) {
        empty.classList.add("is-leaving");
        setTimeout(() => empty.remove(), 180);
      }
      const quick = $("quick");
      if (quick) quick.hidden = true;
    }

    function appendUser(text) {
      const row = template("user");
      row.classList.add("ui-pop-in");
      row.querySelector("[data-agentchat-bubble]").textContent = text;
      messages.append(row);
      return row;
    }

    function appendAssistant() {
      const row = template("assistant");
      row.classList.add("ui-enter-up");
      messages.append(row);
      return row;
    }

    function setNote(note, title, text) {
      note.querySelector("[data-agentchat-note-title]").textContent = title;
      const body = note.querySelector("[data-agentchat-note-text]");
      body.textContent = text || "";
      body.hidden = !text;
    }

    function appendNote(title, text) {
      const note = template("note");
      note.classList.add("ui-pop-in");
      setNote(note, title, text);
      messages.append(note);
      return note;
    }

    // An error row of its own (no assistant reply to hang it on). onRetry adds the Retry button.
    function appendError(message, onRetry) {
      const error = template("error");
      error.querySelector("[data-agentchat-error-text]").textContent = message || GENERIC_ERROR;
      const retry = error.querySelector("[data-agentchat-retry]");
      if (onRetry) retry.addEventListener("click", () => {
        error.remove();
        onRetry();
      });
      else retry.remove();
      messages.append(error);
      scrollEnd();
      return error;
    }

    // The completed turns on screen, for apps without a server session for some users.
    function transcriptOf(untilRow) {
      const turns = [];
      for (const row of messages.children) {
        if (row === untilRow) break;
        if (row.matches(".ui-chat-msg--user")) {
          const bubble = row.querySelector("[data-agentchat-bubble]");
          if (bubble) turns.push({ role: "user", text: bubble.textContent });
        } else if (row.matches(".ui-chat-msg--assistant:not(.is-pending):not(.is-error)")) {
          const body = row.querySelector("[data-agentchat-body]");
          const text = body ? body.textContent.trim() : "";
          if (text) turns.push({ role: "assistant", text });
        }
      }
      return turns;
    }

    /* the turn */
    const composerFields = () => [...new FormData(form)].filter(([key, value]) => typeof value === "string" && key !== input.name && key !== "session_id");

    async function send(payload, { retry = false } = {}) {
      if (working()) return;
      st.last = payload;
      hideEmpty();
      payload.notes = st.notes.splice(0);
      dropNotice();
      if (!retry) {
        payload.transcript = transcriptOf(null);
        payload.userRow = payload.text === undefined ? null : appendUser(payload.text);
        if (payload.clear) setInput("");
      } else {
        payload.transcript = transcriptOf(payload.userRow);
      }
      setBusy(true);
      setStatus("Thinking…");
      const row = appendAssistant();
      scrollEnd({ smooth: true, force: true });

      const body = new URLSearchParams();
      body.set("session_id", st.sessionId);
      for (const [key, value] of payload.entries || []) body.append(key, value);
      // The composer's own fields (a CSRF token, a submission id) ride on every turn; the turn's own fields win.
      for (const [key, value] of composerFields()) if (!body.has(key)) body.append(key, value);
      // Retrying a turn reuses its id (the payload is the turn); every new turn has a fresh one.
      if (!payload.turnId) payload.turnId = newTurnId();
      body.set("turn_id", payload.turnId);
      if (payload.text !== undefined) body.set(input.name || "message", payload.text);
      if (root.hasAttribute("data-agentchat-transcript")) body.set("transcript", JSON.stringify(payload.transcript));
      if (payload.notes.length) body.set("notes", JSON.stringify(payload.notes));
      emit("send", { body, form: payload.form });

      const turn = { row, text: "", renderQueued: false, done: false, failed: false, started: false, note: "", events: [], folded: [] };
      st.turn = turn;
      const controller = new AbortController();
      st.abort = controller;
      try {
        const res = await fetch(payload.url || root.dataset.turnUrl, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded", Accept: "text/event-stream" },
          body,
          signal: controller.signal,
          credentials: "same-origin",
        });
        if (!res.ok) {
          fail(turn, await readError(res));
          return;
        }
        if (!res.body || !/text\/event-stream/i.test(res.headers.get("Content-Type") || "")) {
          // A 2xx that is not a stream is a protocol error (a login page, a proxy): the turn did not run.
          fail(turn, { message: GENERIC_ERROR, retryable: true, restore_text: payload.text });
          return;
        }
        const decoder = new TextDecoder();
        const feed = parseSSE((name, data) => handle(turn, name, data));
        const reader = res.body.getReader();
        for (;;) {
          const { value, done } = await reader.read();
          if (done) break;
          feed(decoder.decode(value, { stream: true }));
        }
        if (!turn.done && !turn.failed) fail(turn, { message: DROPPED_ERROR, retryable: true });
      } catch (error) {
        if (error && error.name === "AbortError") stopped(turn);
        else if (!turn.done && !turn.failed) fail(turn, { message: OFFLINE_ERROR, retryable: true });
      } finally {
        st.abort = null;
        st.turn = null;
        setBusy(false);
        if (!slot.querySelector("[data-agentchat-card]:not(.is-answered)")) input.focus({ preventScroll: true });
      }
    }

    // A non-2xx answer before the stream: text/plain is the message, JSON is {message, restore_text, retryable}.
    async function readError(res) {
      const text = ((await res.text().catch(() => "")) || "").trim();
      const error = { message: GENERIC_ERROR, retryable: res.status >= 500 };
      if (text.startsWith("{")) {
        try {
          const parsed = JSON.parse(text);
          if (parsed.message) error.message = String(parsed.message);
          if (parsed.restore_text) error.restore_text = String(parsed.restore_text);
          if (typeof parsed.retryable === "boolean") error.retryable = parsed.retryable;
        } catch (e) { /* keep the generic error */ }
      } else if (text && text.length < 300 && !text.startsWith("<")) error.message = text;
      return error;
    }

    function handle(turn, name, data) {
      switch (name) {
        case "session":
          applySession(data);
          emit("session", data);
          break;
        case "step":
          setStatus(data.label);
          addStep(turn, data.label);
          break;
        case "delta":
          if (!turn.started) {
            turn.started = true;
            finishSteps(turn);
            const body = bodyOf(turn.row);
            body.hidden = false;
            body.classList.add("is-provisional");
            turn.row.querySelector("[data-agentchat-typing]").hidden = true;
            setStatus("Writing…");
            setOrb("is-speaking");
            const orb = turn.row.querySelector("[data-agentchat-row-orb] .ui-orb");
            if (orb) orb.classList.replace("is-thinking", "is-speaking");
          }
          turn.text += data.text;
          queueRender(turn);
          break;
        case "reset":
          // The text so far was narration before a tool call, not the reply: it becomes the caption of the next step.
          turn.note = captionOf(turn.text);
          turn.text = "";
          turn.started = false;
          bodyOf(turn.row).hidden = true;
          bodyOf(turn.row).replaceChildren();
          turn.row.querySelector("[data-agentchat-typing]").hidden = turn.events.length > 0;
          setOrb("is-thinking");
          break;
        case "done":
          finish(turn, data);
          break;
        case "error":
          fail(turn, data);
          break;
      }
    }

    const bodyOf = (row) => row.querySelector("[data-agentchat-body]");

    /* step list: what the agent is doing, one line per step event */
    function finishSteps(turn) {
      turn.row.querySelectorAll(".ui-chat-step.is-active").forEach((li) => {
        li.classList.replace("is-active", "is-done");
        const label = li.querySelector("[data-agentchat-step-label]");
        label.textContent = trimDots(label.textContent);
      });
    }

    function addStep(turn, label) {
      const list = turn.row.querySelector("[data-agentchat-steps]");
      if (!list) return;
      const before = turn.folded.length;
      turn.events.push({ label, caption: turn.note });
      const caption = turn.note;
      turn.note = "";
      turn.folded = foldSteps(turn.events);
      const detail = { label, count: turn.folded[turn.folded.length - 1].count };
      if (turn.folded.length === before) {
        // The same step again: one line with a count.
        const holder = list.lastElementChild.querySelector(".ui-chat-step__label");
        let badge = holder.querySelector(".ui-chat-step__count");
        if (!badge) {
          badge = document.createElement("span");
          badge.className = "ui-chat-step__count num";
          holder.append(badge);
        }
        badge.textContent = `×${detail.count}`;
      } else {
        finishSteps(turn);
        const li = template("step");
        li.querySelector("[data-agentchat-step-label]").textContent = `${trimDots(label)}…`;
        const note = li.querySelector("[data-agentchat-step-note]");
        note.textContent = caption;
        note.hidden = !caption;
        list.hidden = false;
        list.append(li);
        // The dots give way to the list once there is something concrete to show.
        turn.row.querySelector("[data-agentchat-typing]").hidden = true;
        bodyOf(turn.row).hidden = !turn.started;
        scrollEnd();
      }
      emit("step", detail);
    }

    // After the turn the steps fold into one quiet toggle the reader can open.
    function settleSteps(turn) {
      const list = turn.row.querySelector("[data-agentchat-steps]");
      if (!list || list.hidden || !turn.events.length) return;
      list.querySelectorAll(".ui-chat-step.is-active").forEach((li) => li.classList.replace("is-active", "is-done"));
      const toggle = document.createElement("button");
      toggle.type = "button";
      toggle.className = "ui-chat-steps__toggle";
      toggle.setAttribute("data-agentchat-steps-toggle", "");
      toggle.setAttribute("aria-expanded", "false");
      const total = turn.events.length;
      toggle.textContent = `${total} ${total === 1 ? "step" : "steps"}`;
      list.hidden = true;
      list.before(toggle);
    }

    function queueRender(turn) {
      if (turn.renderQueued) return;
      turn.renderQueued = true;
      requestAnimationFrame(() => {
        turn.renderQueued = false;
        if (turn.done) return;
        const body = bodyOf(turn.row);
        body.innerHTML = renderMarkdown(turn.text);
        let host = body;
        while (host.lastElementChild && !["PRE", "A", "CODE"].includes(host.lastElementChild.tagName)) host = host.lastElementChild;
        const caret = document.createElement("span");
        caret.className = "ui-caret";
        host.append(caret);
        scrollEnd();
      });
    }

    function finalizeRow(turn) {
      turn.row.classList.remove("is-pending");
      turn.row.removeAttribute("data-agentchat-pending");
      turn.row.removeAttribute("aria-busy");
      const typing = turn.row.querySelector("[data-agentchat-typing]");
      if (typing) typing.remove();
      const orb = turn.row.querySelector("[data-agentchat-row-orb] .ui-orb");
      if (orb) orb.classList.remove("is-thinking", "is-speaking");
    }

    function stoppedMark(turn) {
      const mark = document.createElement("p");
      mark.className = "ui-chat-stopped";
      mark.textContent = "Stopped";
      turn.row.querySelector(".ui-chat-msg__col").append(mark);
    }

    function finish(turn, data) {
      turn.done = true;
      const body = bodyOf(turn.row);
      body.hidden = false;
      body.removeAttribute("aria-hidden");
      body.classList.remove("is-provisional");
      body.innerHTML = data.html || (data.stopped ? renderMarkdown(turn.text) : "");
      finalizeRow(turn);
      settleSteps(turn);
      if (data.stopped) stoppedMark(turn);
      else if (!body.firstChild && !turn.events.length) turn.row.remove();
      setCard(data.card_html);
      emit("done", data);
    }

    function setCard(html) {
      slot.innerHTML = html || "";
      slot.querySelectorAll("textarea[data-autogrow]").forEach((t) => {
        t.style.height = "auto";
        t.style.height = `${t.scrollHeight}px`;
      });
      if (html) scrollToCard();
      else scrollEnd();
    }

    function scrollToCard() {
      if (!st.stick) return;
      requestAnimationFrame(() => {
        const card = slot.firstElementChild;
        if (card) card.scrollIntoView({ block: "nearest", behavior: reduceMotion() ? "auto" : "smooth" });
      });
    }

    function fail(turn, data) {
      turn.failed = true;
      const payload = st.last;
      st.notes.unshift(...(payload.notes || []));
      finalizeRow(turn);
      settleSteps(turn);
      const message = data.message || GENERIC_ERROR;
      if (payload.cardEl) {
        // The card was replaced (another tab, or the chat moved on): drop the dead one and show the current one.
        if (data.stale) {
          if (payload.cardEl.answeredRow) payload.cardEl.answeredRow.remove();
          setCard(data.card_html);
        } else expandCard(payload.cardEl);
        turn.row.remove();
        appendError(message, null);
      } else if (data.restore_text) {
        // The message never ran: it goes back into the composer, and Retry sends it again.
        if (!input.value.trim()) setInput(data.restore_text);
        if (payload.userRow) payload.userRow.remove();
        payload.userRow = null;
        turn.row.remove();
        appendError(message, data.retryable ? () => {
          payload.clear = input.value === data.restore_text;
          send(payload);
        } : null);
      } else {
        const body = bodyOf(turn.row);
        if (!turn.text) body.remove();
        else body.removeAttribute("aria-hidden");
        const error = template("error");
        error.querySelector("[data-agentchat-error-text]").textContent = message;
        const retry = error.querySelector("[data-agentchat-retry]");
        if (!data.retryable) retry.remove();
        else retry.addEventListener("click", () => {
          turn.row.remove();
          send(payload, { retry: true });
        });
        turn.row.querySelector(".ui-chat-msg__col").append(error);
        turn.row.classList.add("is-error");
      }
      scrollEnd();
      emit("error", data);
    }

    function stopped(turn) {
      finalizeRow(turn);
      settleSteps(turn);
      const body = bodyOf(turn.row);
      body.classList.remove("is-provisional");
      if (turn.text) {
        body.hidden = false;
        body.innerHTML = renderMarkdown(turn.text);
        body.removeAttribute("aria-hidden");
        stoppedMark(turn);
      } else if (turn.events.length) stoppedMark(turn);
      else turn.row.remove();
      if (st.last && st.last.cardEl) expandCard(st.last.cardEl);
      emit("done", { html: "", stopped: true });
    }

    // Stop is real: tell the server to end the turn, then drop the stream so the composer is free straight away.
    function stop() {
      cancelOnServer();
      if (!st.resume && st.abort) st.abort.abort();
    }

    function cancelOnServer() {
      const url = root.dataset.cancelUrl;
      if (!url || !st.sessionId) return;
      fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({ session_id: st.sessionId }),
        credentials: "same-origin",
        keepalive: true,
      }).catch(() => {});
    }

    /* session handoff */
    function applySession(s) {
      if (!s || !s.session_id) return;
      st.sessionId = s.session_id;
      root.dataset.sessionId = s.session_id;
      root.querySelectorAll('input[name="session_id"]').forEach((i) => { i.value = s.session_id; });
      if (s.url) {
        try {
          const url = new URL(s.url, global.location.href);
          if (url.origin === global.location.origin) global.history.replaceState(global.history.state, "", url.pathname + url.search + url.hash);
        } catch (e) { /* keep the current address */ }
      }
    }

    /* cards: forms that send a turn, forms that run an action */
    function collapseCard(card, summary) {
      const row = template("answered");
      row.classList.add("ui-pop-in");
      row.querySelector("[data-agentchat-summary]").textContent = summary;
      messages.append(row);
      card.answeredRow = row;
      card.inert = true;
      card.classList.add("is-answered");
    }

    function expandCard(card) {
      if (!card || !card.isConnected) return;
      if (card.answeredRow) card.answeredRow.remove();
      card.inert = false;
      card.classList.remove("is-answered");
    }

    // "Label: value" for each question, unless the form or the submit button says what to show.
    function summaryOf(f, submitter) {
      const explicit = (submitter && submitter.dataset.agentchatSummary) ?? f.dataset.agentchatSummary;
      if (explicit !== undefined) return explicit;
      const lines = [];
      f.querySelectorAll("[data-agentchat-q-label]").forEach((q) => {
        const chips = [...q.querySelectorAll("input[type=radio]:checked, input[type=checkbox]:checked")].map((i) => i.value);
        let value = chips.join(", ");
        if (!chips.length) {
          const field = q.querySelector("input:not([type=hidden]):not([type=radio]):not([type=checkbox]), textarea, select");
          value = field ? field.value.trim() : "";
        }
        if (value) lines.push(`${q.dataset.agentchatQLabel}: ${value}`);
      });
      return lines.join(" · ") || "No answers";
    }

    function submitTurnForm(f, submitter) {
      if (working()) return;
      if (!(submitter && submitter.formNoValidate) && !f.reportValidity()) return;
      const data = new FormData(f, submitter || undefined);
      f.querySelectorAll("[data-agentchat-group]").forEach((el) => data.set(el.name, el.value.replace(/[,\s]/g, "")));
      const entries = [...data].filter(([key, value]) => typeof value === "string" && key !== "session_id");
      const card = f.closest("[data-agentchat-card]") || f;
      const detail = { form: f, summary: summaryOf(f, submitter) };
      emit("summarize", detail);
      collapseCard(card, detail.summary);
      send({ entries, form: f, cardEl: card, url: f.getAttribute("action") || undefined });
    }

    function cardError(card, text) {
      let error = card.querySelector("[data-agentchat-card-error]");
      if (!error) {
        error = document.createElement("p");
        error.className = "ui-chat-card__error";
        error.setAttribute("role", "alert");
        error.setAttribute("data-agentchat-card-error", "");
        card.append(error);
      }
      error.textContent = text;
    }

    async function submitAction(f, submitter) {
      if (f.hasAttribute("aria-busy")) return;
      if (!(submitter && submitter.formNoValidate) && !f.reportValidity()) return;
      const card = f.closest("[data-agentchat-card]") || f;
      const error = card.querySelector("[data-agentchat-card-error]");
      if (error) error.remove();
      const buttons = [...f.querySelectorAll("button, input[type=submit]")].filter((b) => !b.disabled);
      buttons.forEach((b) => { b.disabled = true; });
      f.setAttribute("aria-busy", "true");
      const settle = () => {
        buttons.forEach((b) => { b.disabled = false; });
        f.removeAttribute("aria-busy");
      };
      try {
        const data = new FormData(f, submitter || undefined);
        const method = (f.getAttribute("method") || "post").toUpperCase();
        const init = { method, headers: { "X-Agentchat-Action": "1" }, credentials: "same-origin" };
        let url = f.action;
        if (method === "GET") {
          const target = new URL(f.action, global.location.href);
          for (const [key, value] of data) if (typeof value === "string") target.searchParams.append(key, value);
          url = target.toString();
        } else init.body = f.enctype === "multipart/form-data" ? data : new URLSearchParams(data);
        const res = await fetch(url, init);
        const text = await res.text();
        if (!res.ok) {
          const plain = text.trim();
          cardError(card, plain && plain.length < 300 && !plain.startsWith("<") ? plain : "That didn't go through. Try again.");
          settle();
          return;
        }
        const note = decodeNote(res.headers.get("X-Agentchat-Note"));
        const receipt = document.createElement("template");
        receipt.innerHTML = text;
        card.replaceWith(receipt.content);
        if (note) {
          appendNote(note, "");
          st.notes.push(note);
          scrollEnd();
        }
        emit("action", { form: f, response: res, note });
      } catch (e) {
        cardError(card, "Couldn't reach the server. Check your connection and try again.");
        settle();
      }
    }

    root.addEventListener("submit", (event) => {
      const f = event.target;
      if (f === form || !f.getAttribute) return;
      const mode = f.getAttribute("data-agentchat-submit");
      if (mode !== "turn" && mode !== "action") return;
      event.preventDefault();
      if (mode === "turn") submitTurnForm(f, event.submitter);
      else submitAction(f, event.submitter);
    });

    // A card that is one unanswered choice sends as soon as a chip is picked.
    root.addEventListener("change", (event) => {
      const f = event.target.closest && event.target.closest("form[data-agentchat-autosubmit]");
      if (f && event.target.matches(".ui-choice-chip__input")) setTimeout(() => f.requestSubmit(), 220);
    });

    // Number inputs in cards show thousands separators once you leave them; the commas go before they are sent.
    root.addEventListener("focusout", (event) => {
      if (event.target.matches && event.target.matches("[data-agentchat-group]")) event.target.value = groupNumber(event.target.value);
    });
    root.addEventListener("focusin", (event) => {
      if (event.target.matches && event.target.matches("[data-agentchat-group]")) event.target.value = event.target.value.replace(/,/g, "");
    });

    /* composer */
    form.addEventListener("submit", (event) => {
      event.preventDefault();
      if (st.busy) return stop();
      if (working()) return;
      const text = input.value.trim();
      if (!text) return;
      send({ text, entries: [], form, clear: true });
    });

    input.addEventListener("keydown", (event) => {
      if (event.key !== "Enter" || event.shiftKey || event.isComposing || event.keyCode === 229) return;
      event.preventDefault();
      if (!working()) form.requestSubmit();
    });

    /* prompts, steps toggle and the docks */
    // The rail and panel dock on wide screens and slide over below that; the stylesheet decides which, so
    // "sliding over" is read from the computed position. Only a slide-over is a modal dialog.
    const dockEl = (name) => $(name);
    const slideOver = (name) => {
      const el = dockEl(name);
      return !!el && global.getComputedStyle(el).position === "absolute";
    };
    const isOpen = (name) => root.classList.contains(`is-${name}-open`);
    const focusables = (el) => [...el.querySelectorAll(FOCUSABLE)].filter((x) => x.getClientRects().length);
    let returnFocus = null;

    function syncDock(name) {
      const el = dockEl(name);
      if (!el) return;
      if (isOpen(name) && slideOver(name)) {
        el.setAttribute("role", "dialog");
        el.setAttribute("aria-modal", "true");
      } else {
        el.removeAttribute("role");
        el.removeAttribute("aria-modal");
      }
    }

    function setDock(name, open, opener) {
      const was = isOpen(name);
      root.classList.toggle(`is-${name}-open`, open);
      root.querySelectorAll(`[data-agentchat-toggle="${name}"]`).forEach((b) => b.setAttribute("aria-expanded", String(open)));
      syncDock(name);
      const el = dockEl(name);
      if (!el || !slideOver(name) || open === was) return;
      if (open) {
        // One slide-over at a time.
        const other = name === "rail" ? "panel" : "rail";
        if (isOpen(other) && slideOver(other)) setDock(other, false);
        returnFocus = opener || document.activeElement;
        requestAnimationFrame(() => {
          const target = el.querySelector("[data-agentchat-autofocus]") || focusables(el)[0];
          if (target) target.focus();
          else {
            el.tabIndex = -1;
            el.focus();
          }
        });
      } else if (returnFocus && returnFocus.isConnected) {
        returnFocus.focus();
        returnFocus = null;
      }
    }
    const closeDocks = () => {
      setDock("rail", false);
      setDock("panel", false);
    };
    if (global.ResizeObserver) {
      const observer = new global.ResizeObserver(() => {
        scrollEnd();
        syncDock("rail");
        syncDock("panel");
      });
      observer.observe(col);
      observer.observe(root);
    }

    root.addEventListener("click", (event) => {
      const target = event.target;
      if (!target.closest) return;
      const action = target.closest("[data-agentchat-action]");
      if (action) {
        emit("starter", { action: action.getAttribute("data-agentchat-action") }, true);
        return;
      }
      const prompt = target.closest("[data-agentchat-prompt]");
      if (prompt) {
        if (!working()) send({ text: prompt.getAttribute("data-agentchat-prompt"), entries: [], form });
        return;
      }
      const toggle = target.closest("[data-agentchat-steps-toggle]");
      if (toggle) {
        const list = toggle.nextElementSibling;
        const open = list.hidden;
        list.hidden = !open;
        toggle.setAttribute("aria-expanded", String(open));
        return;
      }
      const dock = target.closest("[data-agentchat-toggle]");
      if (dock) {
        const name = dock.getAttribute("data-agentchat-toggle");
        setDock(name, !isOpen(name), dock);
        return;
      }
      if (target.closest("[data-agentchat-scrim]")) closeDocks();
    });
    root.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && (isOpen("rail") || isOpen("panel"))) {
        closeDocks();
        return;
      }
      if (event.key !== "Tab") return;
      const name = ["panel", "rail"].find((n) => isOpen(n) && slideOver(n));
      if (!name) return;
      // Tab stays inside an open slide-over.
      const el = dockEl(name);
      const items = focusables(el);
      if (!items.length) {
        event.preventDefault();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      const inside = el.contains(document.activeElement);
      if (event.shiftKey && (!inside || document.activeElement === first)) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && (!inside || document.activeElement === last)) {
        event.preventDefault();
        first.focus();
      }
    });

    /* a turn this page did not start */
    // After a reload mid-turn the server renders the pending row; poll until the turn is over, then swap in the
    // finished thread and card.
    function resumeTurn(row) {
      const turn = { row };
      st.resume = turn;
      setBusy(true);
      setStatus("Working…");
      row.classList.add("ui-enter-up");
      row.querySelector("[data-agentchat-row-status]").textContent = "Still working on your last message…";
      scrollEnd({ force: true });
      const started = Date.now();
      let misses = 0;
      const tick = async () => {
        if (st.resume !== turn) return;
        let data = null;
        try {
          const res = await fetch(`${root.dataset.stateUrl}${root.dataset.stateUrl.includes("?") ? "&" : "?"}session_id=${encodeURIComponent(st.sessionId)}`, { credentials: "same-origin", cache: "no-store" });
          if (res.ok) data = await res.json();
        } catch (e) { /* counted below */ }
        misses = data ? 0 : misses + 1;
        if (data && !data.inflight) return landResumed(turn, data);
        if (misses > 5 || Date.now() - started > 5 * 60 * 1000) {
          st.resume = null;
          setBusy(false);
          row.remove();
          appendError("Still working. Reload in a moment to see the reply.", null);
          return;
        }
        setTimeout(tick, 2000);
      };
      setTimeout(tick, 1200);
    }

    function landResumed(turn, data) {
      st.resume = null;
      if (data.thread_html) messages.innerHTML = data.thread_html;
      turn.row.remove();
      setCard(data.card_html);
      setBusy(false);
      if (!data.card_html) input.focus({ preventScroll: true });
      emit("done", Object.assign({ resumed: true }, data));
    }

    /* the controller */
    const controller = {
      // Sends text as the user's next message; extra is an object of additional form fields.
      send(text, extra) {
        const entries = [];
        for (const [key, value] of Object.entries(extra || {})) for (const v of [].concat(value)) entries.push([key, String(v)]);
        return send({ text: String(text), entries, form });
      },
      stop,
      // Adds a note row for work the app runs itself (an upload). Returns {update(title, text), working(on), remove()}.
      note(title, text) {
        const el = appendNote(title, text);
        scrollEnd({ smooth: true, force: true });
        return {
          update: (nextTitle, nextText) => setNote(el, nextTitle, nextText),
          working: (on = true) => el.classList.toggle("is-working", !!on),
          remove: () => el.remove(),
        };
      },
      // Replaces the card slot (empty html clears it).
      setCard,
      // Sets the session id; the address bar changes only when url is given.
      setSession: (id, url) => applySession({ session_id: id, url }),
      // Makes the chat busy while the app works: the composer is inert and no Stop is offered. Call the
      // returned release() when done. status is the header text meanwhile.
      hold(status) {
        st.held += 1;
        renderBusy();
        setStatus(status || "Working…");
        let released = false;
        return () => {
          if (released) return;
          released = true;
          st.held -= 1;
          renderBusy();
        };
      },
      get busy() {
        return working();
      },
    };
    controllers.set(root, controller);

    /* boot */
    autogrow();
    updateCount();
    const pending = root.querySelector("[data-agentchat-pending]");
    const autosend = root.dataset.autosend;
    if (pending) resumeTurn(pending);
    else if (autosend) {
      // The server handed this message over: send it once. (Never a URL parameter — a link must not make the agent act.)
      delete root.dataset.autosend;
      requestAnimationFrame(() => send({ text: autosend, entries: [], form }));
    } else if (!slot.firstElementChild && (document.activeElement === document.body || !document.activeElement) && root.getClientRects().length) {
      input.focus({ preventScroll: true });
      input.setSelectionRange(input.value.length, input.value.length);
    }
    return controller;
  }

  function initAll() {
    document.querySelectorAll("[data-agentchat]").forEach(init);
  }

  global.OrbitalChat = {
    init,
    get: (root) => controllers.get(root) || init(root),
    _internals: { parseSSE, renderMarkdown, foldSteps, captionOf, trimDots, newTurnId },
  };

  if (typeof document !== "undefined") {
    if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", initAll);
    else initAll();
    document.addEventListener("htmx:afterSettle", initAll);
    // A page restored from the back/forward cache would show a dead stream: reload it.
    global.addEventListener("pageshow", (event) => {
      if (event.persisted) global.location.reload();
    });
  }
})(typeof window !== "undefined" ? window : globalThis);
