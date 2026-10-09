// Unit tests for the pure helpers of agent-chat.js. The IIFE runs in node:vm
// against a minimal window stub, so no DOM is needed: the DOM behaviour is
// covered by scripts/browser-check.mjs in a real browser.
//
//   node --test assets/*.test.mjs
import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";

function load() {
  const listeners = [];
  const sandbox = {
    document: { readyState: "complete", querySelectorAll: () => [], addEventListener: (name) => listeners.push(name) },
    addEventListener: (name) => listeners.push(name),
  };
  sandbox.window = sandbox;
  vm.createContext(sandbox);
  vm.runInContext(readFileSync(new URL("./agent-chat.js", import.meta.url), "utf8"), sandbox);
  return { chat: sandbox.OrbitalChat, listeners };
}

// Values built inside the vm context have that realm's prototypes; compare them as plain JSON.
const plain = (value) => JSON.parse(JSON.stringify(value));

const { chat, listeners } = load();
const { parseSSE, renderMarkdown, foldSteps, captionOf, trimDots, newTurnId } = chat._internals;

test("loading the runtime exposes OrbitalChat and wires the page hooks", () => {
  assert.equal(typeof chat.init, "function");
  assert.equal(typeof chat.get, "function");
  assert.deepEqual(Object.keys(chat._internals).sort(), ["captionOf", "foldSteps", "newTurnId", "parseSSE", "renderMarkdown", "trimDots"]);
  assert.ok(listeners.includes("htmx:afterSettle") && listeners.includes("pageshow"));
});

test("parseSSE parses events split across chunks and ignores heartbeats", () => {
  const seen = [];
  const feed = parseSSE((name, data) => seen.push([name, data]));
  feed(": open\n\nevent: step\ndata: {\"label\":\"Rea");
  feed("ding\"}\n\n: ping\n\nevent: delta\r\ndata: {\"text\":\"Hi\"}\r\n\r\n");
  assert.deepEqual(plain(seen), [["step", { label: "Reading" }], ["delta", { text: "Hi" }]]);
});

test("parseSSE joins multi-line data, skips malformed json and unnamed data", () => {
  const seen = [];
  const feed = parseSSE((name, data) => seen.push([name, data]));
  feed("event: x\ndata: {bad\n\nevent: done\ndata: {\"a\":\ndata: 1}\n\nevent: empty\n\ndata: {\"n\":1}\n\n");
  assert.deepEqual(plain(seen), [["done", { a: 1 }], ["message", { n: 1 }]]);
});

test("renderMarkdown renders paragraphs, bold, code and lists", () => {
  const html = renderMarkdown("Hello **there** `x`\n\n- one\n- two\n\n1. a\n2. b");
  assert.equal(html, "<p>Hello <strong>there</strong> <code>x</code></p><ul><li>one</li><li>two</li></ul><ol><li>a</li><li>b</li></ol>");
});

test("renderMarkdown escapes first and only links safe URLs", () => {
  assert.equal(renderMarkdown("<script>alert(1)</script>"), "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>");
  assert.equal(renderMarkdown('<img src=x onerror="alert(1)">'), "<p>&lt;img src=x onerror=&quot;alert(1)&quot;&gt;</p>");
  assert.equal(renderMarkdown("[x](javascript:alert(1))"), "<p>[x](javascript:alert(1))</p>");
  assert.equal(renderMarkdown("[x](data:text/html,hi)"), "<p>[x](data:text/html,hi)</p>");
  assert.equal(renderMarkdown("[ok](https://a.example/b)"), '<p><a href="https://a.example/b" target="_blank" rel="noopener noreferrer">ok</a></p>');
  // A quote in the URL stays inside the escaped href: it cannot open a new attribute.
  assert.equal(renderMarkdown('[q](https://a.example/"onmouseover="x)'), '<p><a href="https://a.example/&quot;onmouseover=&quot;x" target="_blank" rel="noopener noreferrer">q</a></p>');
  assert.equal(renderMarkdown("[m](mailto:a@b.example)"), '<p><a href="mailto:a@b.example">m</a></p>');
});

test("renderMarkdown keeps an unterminated fence, partial emphasis and loose lists sensible while streaming", () => {
  assert.equal(renderMarkdown("```\nlet a = 1 < 2"), "<pre><code>let a = 1 &lt; 2</code></pre>");
  assert.equal(renderMarkdown("This is **bol"), "<p>This is **bol</p>");
  assert.equal(renderMarkdown("1. a\n\n2. b\n\nDone"), "<ol><li>a</li><li>b</li></ol><p>Done</p>");
  assert.equal(renderMarkdown("## Title\ntext"), "<h2>Title</h2><p>text</p>");
});

test("foldSteps counts the same step in a row and keeps captioned repeats apart", () => {
  assert.deepEqual(plain(foldSteps([])), []);
  assert.deepEqual(
    plain(foldSteps([{ label: "Reading" }, { label: "Reading" }, { label: "Reading", caption: "Once more" }, { label: "Saving" }, { label: "Reading" }])),
    [
      { label: "Reading", count: 2, caption: "" },
      { label: "Reading", count: 1, caption: "Once more" },
      { label: "Saving", count: 1, caption: "" },
      { label: "Reading", count: 1, caption: "" },
    ],
  );
  const events = [{ label: "a" }, { label: "a" }];
  foldSteps(events);
  assert.deepEqual(events, [{ label: "a" }, { label: "a" }], "the input is not modified");
});

test("trimDots drops a trailing ellipsis", () => {
  assert.equal(trimDots("Reading your profile…"), "Reading your profile");
  assert.equal(trimDots("Checking Penang..."), "Checking Penang");
  assert.equal(trimDots("Saving"), "Saving");
});

test("captionOf flattens markdown and shortens long narration", () => {
  assert.equal(captionOf("**Let me** look\n\nat your `profile`."), "Let me look at your profile.");
  const long = captionOf("word ".repeat(60));
  assert.ok(long.length <= 96 && long.endsWith("…"));
  assert.equal(captionOf("   "), "");
});

test("newTurnId is a fresh UUID v4, with or without crypto.randomUUID", () => {
  const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
  const make = (crypto) => {
    const sandbox = { document: { readyState: "complete", querySelectorAll: () => [], addEventListener() {} }, addEventListener() {}, crypto };
    sandbox.window = sandbox;
    vm.createContext(sandbox);
    vm.runInContext(readFileSync(new URL("./agent-chat.js", import.meta.url), "utf8"), sandbox);
    return sandbox.OrbitalChat._internals.newTurnId;
  };
  const ids = new Set();
  for (const gen of [newTurnId, make(undefined), make({ getRandomValues: (a) => a.map(() => Math.floor(Math.random() * 256)) }), make({ randomUUID: () => "11111111-2222-4333-8444-555555555555" })]) {
    const id = gen();
    assert.match(id, uuid);
    ids.add(id);
  }
  assert.equal(ids.size, 4);
  assert.notEqual(newTurnId(), newTurnId());
});
