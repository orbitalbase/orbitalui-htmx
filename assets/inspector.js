/* Shared record inspector. No Alpine factories, application globals or eval.
   The server remains authoritative for content and URL-backed panel state. */
(() => {
  let lastURL = location.href;
  let opener = null;
  let openerRecord = null;
  let openerLinkIndex = 0;
  let request = null;
  let serial = 0;
  let drag = null;
  let motion = null;
  const inertElements = new Map();
  const drafts = new Map();
  const mobile = matchMedia("(max-width: 58rem)");
  const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)");
  const panel = () => document.querySelector("[data-orbital-inspector]");
  const keys = (node) => (node?.dataset.inspectorStateKeys || "selected,panel").split(",");
  const remainder = (url, node) => {
    const copy = new URL(url, location.href);
    keys(node).forEach((key) => copy.searchParams.delete(key));
    if (node?.hasAttribute("data-inspector-ignore-empty-query")) {
      Array.from(copy.searchParams.keys()).forEach((key) => {
        if (copy.searchParams.getAll(key).every((value) => value === "")) copy.searchParams.delete(key);
      });
    }
    copy.searchParams.sort();
    return copy.origin + copy.pathname + copy.search;
  };
  const sameHost = (url, node) => remainder(url, node) === remainder(lastURL, node);
  const overlay = (node) => !!node && !node.hidden && (mobile.matches || node.classList.contains("ui-inspector--expanded"));
  const draftKey = (url, node) => {
    const value = new URL(url, location.href);
    return value.pathname + JSON.stringify(value.searchParams.getAll(keys(node)[0]));
  };
  function slide(node, entering) {
    if (!node || node.hidden || reducedMotion.matches || !node.animate) return null;
    motion?.cancel();
    const animation = node.animate(
      entering
        ? [{ transform: "translateX(100%)", opacity: .6 }, { transform: "translateX(0)", opacity: 1 }]
        : [{ transform: "translateX(0)", opacity: 1 }, { transform: "translateX(100%)", opacity: .6 }],
      { duration: entering ? 160 : 120, easing: "cubic-bezier(.22, 1, .36, 1)", fill: "both" },
    );
    motion = animation;
    animation.finished.then(() => {
      if (entering) {
        animation.cancel();
        if (motion === animation) motion = null;
      }
    }).catch(() => {});
    return animation;
  }
  function saveDraft(node, url) {
    if (!node || node.hidden) return;
    drafts.set(draftKey(url, node), Array.from(node.querySelectorAll("input:not([type=hidden]):not([type=file]), textarea, select"), (control) => ({
      name: control.name, value: control.value, checked: control.checked,
    })));
  }
  function restoreDraft(node, url) {
    const values = drafts.get(draftKey(url, node));
    if (!values) return;
    node.querySelectorAll("input:not([type=hidden]):not([type=file]), textarea, select").forEach((control, i) => {
      const saved = values[i];
      if (!saved || saved.name !== control.name) return;
      control.value = saved.value;
      if ("checked" in control) control.checked = saved.checked;
    });
  }
  function unlock() {
    inertElements.forEach((value, element) => { element.inert = value; });
    inertElements.clear();
    document.body.classList.remove("ui-inspector-overlay");
  }
  function sync() {
    const node = panel();
    unlock();
    if (!node) return;
    syncListNavigation(node);
    if (overlay(node)) {
      // Keep every background branch inert without making the panel's own
      // ancestors inert. This works in any application's shell hierarchy.
      for (let branch = node; branch && branch !== document.body; branch = branch.parentElement) {
        for (const sibling of branch.parentElement?.children || []) {
          if (sibling === branch || /^(SCRIPT|STYLE|LINK)$/.test(sibling.tagName)) continue;
          inertElements.set(sibling, sibling.inert);
          sibling.inert = true;
        }
      }
      node.setAttribute("role", "dialog");
      node.setAttribute("aria-modal", "true");
      document.body.classList.add("ui-inspector-overlay");
    } else {
      node.setAttribute("role", "complementary");
      node.removeAttribute("aria-modal");
    }
    const selected = new URL(location.href).searchParams.getAll(keys(node)[0]).at(-1) || null;
    const host = node.closest("[data-orbital-inspector-host]");
    // Host applications can keep their existing record/card markup. Derive
    // identity from native selection URLs, never from the record's contents.
    host?.querySelectorAll("[data-inspector-list] tr, [data-inspector-list] .record-card").forEach((row) => {
      const links = row.matches("a[href]") ? [row] : Array.from(row.querySelectorAll("a[href]"));
      const selection = links.find((link) => {
        const url = new URL(link.href);
        return url.pathname === location.pathname && url.searchParams.has(keys(node)[0]);
      });
      if (!selection) return;
      row.dataset.inspectorRecord = new URL(selection.href).searchParams.getAll(keys(node)[0]).at(-1);
      links.forEach((link) => {
        const url = new URL(link.href);
        if (url.pathname !== location.pathname || !url.searchParams.has(keys(node)[0])) return;
        // Retain fullscreen state when selecting another record.
        keys(node).slice(1).forEach((key) => {
          url.searchParams.delete(key);
          new URL(location.href).searchParams.getAll(key).forEach((value) => url.searchParams.append(key, value));
        });
        link.href = url.href;
      });
    });
    document.querySelectorAll("[data-inspector-record]").forEach((row) => {
      const active = selected !== null && row.dataset.inspectorRecord === selected;
      row.classList.toggle("is-selected", active);
      if (active) row.setAttribute("aria-current", "true");
      else row.removeAttribute("aria-current");
    });
    if (!node.hidden && !overlay(node)) {
      try {
        const width = Number(localStorage.getItem(node.dataset.inspectorStorageKey));
        if (node.dataset.inspectorStorageKey && width >= 320 && width <= 1000) setWidth(node, width, false);
      } catch (_) { /* Browser storage is optional. */ }
      syncWidth(node);
    }
  }
  function syncListNavigation(node) {
    const state = new URL(location.href).searchParams;
    document.querySelectorAll("[data-inspector-list-nav]").forEach((control) => {
      if (control instanceof HTMLFormElement) {
        // The filter form survives panel swaps, including the first opening
        // from a closed page. Keep its native GET state fields current.
        keys(node).forEach((key) => {
          Array.from(control.querySelectorAll("input[type=hidden]")).filter((input) => input.name === key).forEach((input) => input.remove());
          state.getAll(key).forEach((value) => {
            const input = document.createElement("input");
            input.type = "hidden"; input.name = key; input.value = value;
            control.append(input);
          });
        });
      } else if (control instanceof HTMLAnchorElement) {
        const target = new URL(control.href);
        keys(node).forEach((key) => {
          target.searchParams.delete(key);
          state.getAll(key).forEach((value) => target.searchParams.append(key, value));
        });
        if (control.href !== target.href) {
          control.href = target.href;
          if (control.hasAttribute("hx-get")) control.setAttribute("hx-get", target.pathname + target.search);
          window.htmx?.process(control);
        }
      }
    });
  }
  function syncHostURL() {
    const node = panel();
    if (!node) return;
    lastURL = location.href;
    // List-only HTMX updates can change filters/page while retaining the panel.
    // Rebase its chrome on that new context, preserving each link's state keys.
    const rebase = (href) => {
      const source = new URL(href, location.href), target = new URL(location.href);
      if (source.origin !== target.origin || source.pathname !== target.pathname) return href;
      keys(node).forEach((key) => {
        target.searchParams.delete(key);
        source.searchParams.getAll(key).forEach((value) => target.searchParams.append(key, value));
      });
      return target.href;
    };
    node.querySelectorAll(".ui-inspector__tools a[href]").forEach((link) => { link.href = rebase(link.href); });
    if (node.dataset.inspectorClose) node.dataset.inspectorClose = rebase(node.dataset.inspectorClose);
    sync();
  }
  function focusPanel(node) {
    if (!node.hidden) node.querySelector("[data-inspector-title]")?.focus({ preventScroll: true });
    else {
      let target = opener?.isConnected ? opener : null;
      if (!target && openerRecord) {
        const row = Array.from(document.querySelectorAll("[data-inspector-record]")).find((item) => item.dataset.inspectorRecord === openerRecord);
        target = row?.querySelectorAll("a[href]")[openerLinkIndex] || row?.querySelector("a[href]");
      }
      // Filtering may remove the originating row. Return to search in that case.
      if (!target) target = document.querySelector("form[data-inspector-list-nav] input[type=search]");
      target?.focus({ preventScroll: true });
    }
  }
  async function navigate(url, pop = false) {
    url = new URL(url, location.href).href;
    const current = panel();
    if (!current || !sameHost(url, current)) { location.assign(url); return; }
    request?.abort();
    motion?.cancel();
    request = new AbortController();
    const ticket = ++serial;
    current.setAttribute("aria-busy", "true");
    saveDraft(current, lastURL);
    let exit = !current.hidden && !new URL(url).searchParams.has(keys(current)[0]) ? slide(current, false) : null;
    try {
      const response = await fetch(url, { signal: request.signal, credentials: "same-origin", headers: { Accept: "text/html" } });
      if (ticket !== serial) return;
      if (response.redirected) { location.assign(response.url); return; }
      if (!response.ok) throw new Error("Inspector request failed");
      const documentNext = new DOMParser().parseFromString(await response.text(), "text/html");
      if (ticket !== serial) return;
      const next = documentNext.getElementById(current.id);
      if (!next?.matches("[data-orbital-inspector]")) throw new Error("Inspector mount missing");
      restoreDraft(next, url);
      if (!current.hidden && next.hidden) {
        exit ||= slide(current, false);
        await exit?.finished.catch(() => {});
        if (ticket !== serial) return;
      }
      const entering = current.hidden && !next.hidden;
      unlock();
      window.Alpine?.destroyTree?.(current);
      current.replaceWith(next);
      window.htmx?.process(next);
      window.Alpine?.initTree?.(next);
      if (!pop) {
        const state = { ...history.state, orbitalInspector: true };
        // HTMX must not also restore the whole page for a panel-only entry.
        delete state.htmx;
        history.pushState(state, "", url);
      }
      lastURL = location.href;
      if (documentNext.title) document.title = documentNext.title;
      sync();
      if (entering) slide(next, true);
      focusPanel(next);
      document.dispatchEvent(new CustomEvent("orbitalui:inspector-change", { detail: { panel: next } }));
    } catch (error) {
      if (error.name === "AbortError" || ticket !== serial) return;
      if (pop) location.reload();
      else location.assign(url);
    } finally {
      exit?.cancel();
      if (ticket === serial) panel()?.removeAttribute("aria-busy");
    }
  }
  function widthCap(node) {
    const host = node.closest("[data-orbital-inspector-host]");
    return Math.max(320, Math.min(1000, (host?.getBoundingClientRect().width || innerWidth) - 320));
  }
  function syncWidth(node) {
    const separator = node.querySelector("[data-inspector-resizer]");
    separator?.setAttribute("aria-valuenow", String(Math.round(node.getBoundingClientRect().width)));
    separator?.setAttribute("aria-valuemax", String(Math.round(widthCap(node))));
  }
  function setWidth(node, value, persist) {
    const width = Math.max(320, Math.min(widthCap(node), Math.round(value)));
    node.style.setProperty("--ui-inspector-width", width + "px");
    syncWidth(node);
    if (persist && node.dataset.inspectorStorageKey) {
      try { localStorage.setItem(node.dataset.inspectorStorageKey, String(width)); } catch (_) { /* Optional. */ }
    }
  }
  document.addEventListener("click", (event) => {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return;
    const target = event.target instanceof Element ? event.target : null;
    let link = target?.closest("a[href]");
    if (!link && !target?.closest("button, input, select, textarea, label")) link = target?.closest("[data-inspector-record]")?.querySelector("a[href]");
    if (!link || link.target || link.hasAttribute("download") || !link.closest("[data-orbital-inspector-host]")) return;
    const node = panel();
    if (!node || link.hash || !sameHost(link.href, node)) return;
    event.preventDefault();
    if (link.href === lastURL) { focusPanel(node); return; }
    if (!link.closest("[data-orbital-inspector]")) {
      opener = link;
      const row = link.closest("[data-inspector-record]");
      openerRecord = row?.dataset.inspectorRecord || null;
      openerLinkIndex = row ? Array.from(row.querySelectorAll("a[href]")).indexOf(link) : 0;
    }
    navigate(link.href);
  });
  window.addEventListener("popstate", (event) => {
    if (event.state?.htmx) return; // List/filter history belongs to HTMX.
    const node = panel();
    if (node && sameHost(location.href, node)) navigate(location.href, true);
  });
  document.addEventListener("keydown", (event) => {
    const node = panel();
    if (!node || node.hidden || event.defaultPrevented) return;
    if (event.key === "Escape" && !event.repeat && !event.altKey && !event.ctrlKey && !event.metaKey && !event.shiftKey &&
        !document.querySelector("dialog[open]") && !event.target.closest("input, textarea, select, [contenteditable]:not([contenteditable='false'])")) {
      event.preventDefault();
      navigate(node.dataset.inspectorClose);
      return;
    }
    if (event.target.matches("[data-inspector-resizer]") && !overlay(node)) {
      if (motion?.effect?.target === node) { motion.cancel(); motion = null; }
      const width = node.getBoundingClientRect().width;
      const next = { ArrowLeft: width + 16, ArrowRight: width - 16, Home: 320, End: widthCap(node) }[event.key];
      if (next !== undefined) { event.preventDefault(); setWidth(node, next, true); }
    }
    if (event.key === "Tab" && overlay(node)) {
      const controls = Array.from(node.querySelectorAll("a[href], button, input, select, textarea, [tabindex='0']"))
        .filter((element) => !element.disabled && element.getClientRects().length);
      const first = controls[0], last = controls.at(-1);
      if (event.shiftKey && (document.activeElement === first || !controls.includes(document.activeElement))) {
        event.preventDefault(); last?.focus();
      } else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
    }
  });
  document.addEventListener("pointerdown", (event) => {
    const handle = event.target.closest?.("[data-inspector-resizer]");
    const node = handle?.closest("[data-orbital-inspector]");
    if (!node || event.button !== 0 || overlay(node)) return;
    event.preventDefault();
    if (motion?.effect?.target === node) { motion.cancel(); motion = null; }
    drag = { node, handle, pointer: event.pointerId, right: node.getBoundingClientRect().right };
    handle.setPointerCapture(event.pointerId);
    document.body.classList.add("ui-inspector-resizing");
  });
  document.addEventListener("pointermove", (event) => {
    if (drag && drag.pointer === event.pointerId) setWidth(drag.node, drag.right - event.clientX, false);
  });
  const endDrag = (event) => {
    if (!drag || drag.pointer !== event.pointerId) return;
    setWidth(drag.node, drag.node.getBoundingClientRect().width, true);
    drag = null;
    document.body.classList.remove("ui-inspector-resizing");
  };
  document.addEventListener("pointerup", endDrag);
  document.addEventListener("pointercancel", endDrag);
  window.addEventListener("resize", sync);
  document.addEventListener("htmx:afterSwap", sync);
  document.addEventListener("htmx:pushedIntoHistory", syncHostURL);
  document.addEventListener("htmx:beforeHistorySave", () => saveDraft(panel(), lastURL));
  document.addEventListener("htmx:historyRestore", () => {
    const node = panel();
    if (node) restoreDraft(node, location.href);
    syncHostURL();
  });
  reducedMotion.addEventListener("change", () => { if (reducedMotion.matches) motion?.cancel(); });
  function initialize() {
    sync();
    const node = panel();
    if (node && !node.hidden) { slide(node, true); focusPanel(node); }
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", initialize);
  else initialize();
})();
