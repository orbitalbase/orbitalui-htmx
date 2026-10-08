/* Orbital UI shell runtime: AppShell sidebar, CommandPalette, Toast and
 * ThemeSwitch behaviour. Pairs with shell.css and the shell, palette, toast
 * and theme-switch components. Optional: the component runtime (orbital-ui.js)
 * does not need it, and hosts with their own toast / palette system can leave
 * it out.
 *
 * Load order: any position. It is independent of orbital-ui.js and of Alpine
 * (plain DOM delegation, no eval, no inline styles), so it is CSP-safe and
 * behaves the same under the standard and the CSP Alpine builds.
 *
 *   <script src="/static/shell.js" defer></script>
 *
 * Toast, palette, theme-switch and shell markup only works when this file is
 * loaded. Server-rendered toasts and the ThemeSwitch radios degrade gracefully
 * without it (toasts stay visible, their close button does nothing), so load
 * it whenever those components render.
 *
 * Plain DOM delegation (no Alpine, no eval, no inline styles), so it behaves
 * the same under the standard and the CSP Alpine builds and when a host
 * runtime owns window.LedgerUI.
 * Public surface: window.OrbitalShell = {toast, openPalette, closePalette}.
 *
 *   [data-oui-shell] [data-shell-open|-close|-scrim]   off-canvas sidebar (phone)
 *   dialog[data-oui-palette]  [data-palette-open]      ⌘K / Ctrl+K palette
 *   [data-toast-stack]  HX-Trigger {"toast": {...}}    notifications
 *   [data-oui-theme-switch]                            syncs with window.OrbitalTheme
 */
(function () {
    "use strict";
    if (window.OrbitalShell) return;

    function ready(fn) {
        if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", fn);
        else fn();
    }

    function reducedMotion() {
        return !!(window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches);
    }

    // ── shell: off-canvas sidebar below 900px ───────────────────────────────
    var PHONE = window.matchMedia ? window.matchMedia("(max-width: 899.98px)") : {matches: false};

    function shellOf(el) { return el && el.closest ? el.closest("[data-oui-shell]") : null; }
    function navOpen(shell) { return shell.getAttribute("data-nav-open") === "true"; }

    // A closed off-canvas sidebar must be unreachable by keyboard and screen reader.
    function syncShell(shell) {
        var sidebar = shell.querySelector(".ui-shell__sidebar");
        if (sidebar) sidebar.inert = PHONE.matches && !navOpen(shell);
        var burger = shell.querySelector("[data-shell-open]");
        if (burger) burger.setAttribute("aria-expanded", navOpen(shell) ? "true" : "false");
    }

    function openNav(shell) {
        shell.setAttribute("data-nav-open", "true");
        syncShell(shell);
        var first = shell.querySelector(".ui-shell__sidebar a[href]");
        if (first) first.focus({preventScroll: true});
    }

    function closeNav(shell, returnFocus) {
        if (!navOpen(shell)) return;
        shell.setAttribute("data-nav-open", "false");
        syncShell(shell);
        var burger = shell.querySelector("[data-shell-open]");
        if (returnFocus && PHONE.matches && burger) burger.focus();
    }

    function openShells() {
        return Array.prototype.filter.call(document.querySelectorAll("[data-oui-shell]"), navOpen);
    }

    if (PHONE.addEventListener) {
        PHONE.addEventListener("change", function () {
            document.querySelectorAll("[data-oui-shell]").forEach(function (s) {
                s.setAttribute("data-nav-open", "false");
                syncShell(s);
            });
        });
    }

    // ── palette ─────────────────────────────────────────────────────────────
    var IS_APPLE = /Mac|iPhone|iPad|iPod/.test((navigator.platform || "") + " " + (navigator.userAgent || ""));
    var CLOSE_MS = 110;
    var DEBOUNCE_MS = 150;
    var RESULT_LIMIT = 12;

    function mod(el) {
        // ⌘ is the Apple key; show Ctrl everywhere else.
        if (IS_APPLE) return;
        el.querySelectorAll("kbd").forEach(function (k) { if (k.textContent === "⌘") k.textContent = "Ctrl"; });
    }

    function score(query, text) {
        var t = text.toLowerCase();
        if (t.indexOf(query) === 0) return 100 - t.length / 100;
        var word = t.indexOf(" " + query);
        if (word >= 0) return 80 - word / 100;
        var sub = t.indexOf(query);
        if (sub >= 0) return 60 - sub / 100;
        var i = 0;
        for (var c = 0; c < t.length; c++) if (t.charAt(c) === query.charAt(i) && ++i === query.length) return 30;
        return 0;
    }

    function itemScore(query, it) {
        return Math.max(score(query, it.label || ""), score(query, (it.label || "") + " " + (it.hint || "") + " " + (it.keywords || "")) * 0.8);
    }

    // Static groups: everything while the query is empty, else ranked matches.
    function filterStatic(groups, query) {
        var q = query.trim().toLowerCase();
        var out = [];
        groups.forEach(function (g) {
            var items = (g.items || []).map(function (it) { return {it: it, s: q ? itemScore(q, it) : 1}; })
                .filter(function (r) { return r.s > 0; })
                .sort(function (a, b) { return b.s - a.s; })
                .map(function (r) { return r.it; });
            if (items.length) out.push({label: g.label, items: items.slice(0, q ? RESULT_LIMIT : 6)});
        });
        return out;
    }

    function initials(name) {
        var words = String(name || "").split(/[^A-Za-z0-9À-￿]+/).filter(Boolean);
        if (!words.length) return "?";
        var first = words[0].charAt(0).toUpperCase();
        return words.length > 1 ? first + words[words.length - 1].charAt(0).toUpperCase() : first;
    }

    function hue(name) {
        var sum = 0;
        for (var i = 0; i < name.length; i++) sum += name.toLowerCase().charCodeAt(i);
        return sum % 6;
    }

    function highlight(text, query) {
        var frag = document.createDocumentFragment();
        var at = query ? text.toLowerCase().indexOf(query) : -1;
        if (at < 0) {
            frag.appendChild(document.createTextNode(text));
            return frag;
        }
        var mark = document.createElement("mark");
        mark.textContent = text.slice(at, at + query.length);
        frag.appendChild(document.createTextNode(text.slice(0, at)));
        frag.appendChild(mark);
        frag.appendChild(document.createTextNode(text.slice(at + query.length)));
        return frag;
    }

    function parseJSON(text, fallback) {
        try { return JSON.parse(text); } catch (err) { return fallback; }
    }

    // Results can echo user-entered record data, so a link or form target is
    // only used when it resolves to http(s); javascript:, data: and the like
    // are dropped rather than trusted.
    function safeURL(value) {
        if (!value) return "";
        try {
            var u = new URL(String(value), window.location.href);
            return u.protocol === "http:" || u.protocol === "https:" ? u.href : "";
        } catch (err) {
            return "";
        }
    }

    function formTarget(item) {
        return item && item.form ? safeURL(item.form.action) : "";
    }

    function Palette(dialog) {
        var input = dialog.querySelector("[data-palette-input]");
        var list = dialog.querySelector("[data-palette-list]");
        var status = dialog.querySelector("[data-palette-status]");
        var endpoint = dialog.getAttribute("data-endpoint") || "";
        var emptyText = dialog.getAttribute("data-empty") || "No results for";
        var staticGroups = parseJSON(dialog.getAttribute("data-groups") || "[]", []);
        var remote = [];
        var rows = [];
        var active = -1;
        var opener = null;
        var timer = 0;
        var closing = 0;
        var seq = 0;
        var controller = null;
        var self = this;

        function icon(name) {
            var holder = dialog.querySelector('template[data-palette-icons]');
            var src = holder && holder.content.querySelector('[data-icon="' + name + '"]');
            return src ? src.firstElementChild.cloneNode(true) : null;
        }

        function lead(item) {
            var el = document.createElement("span");
            if (item.avatar) {
                el.className = "ui-avatar ui-avatar--20 ui-avatar--hue-" + hue(item.avatar) + (item.square ? " ui-avatar--square" : "");
                el.setAttribute("aria-hidden", "true");
                var text = document.createElement("span");
                text.className = "ui-avatar__initials";
                text.textContent = initials(item.avatar);
                el.appendChild(text);
                return el;
            }
            el.className = "ui-palette__icon" + (item.tone ? " ui-palette__icon--" + item.tone : "");
            el.setAttribute("aria-hidden", "true");
            var svg = icon(item.icon || "arrow-right");
            if (svg) el.appendChild(svg);
            return el;
        }

        function setActive(index) {
            if (!rows.length) {
                active = -1;
                input.removeAttribute("aria-activedescendant");
                return;
            }
            active = (index + rows.length) % rows.length;
            rows.forEach(function (row, i) {
                var on = i === active;
                row.el.setAttribute("aria-selected", on ? "true" : "false");
                if (on) {
                    input.setAttribute("aria-activedescendant", row.el.id);
                    if (row.el.scrollIntoView) row.el.scrollIntoView({block: "nearest"});
                }
            });
        }

        function render() {
            var query = input.value.trim();
            var q = query.toLowerCase();
            var groups = filterStatic(staticGroups, input.value).concat(remote);
            list.textContent = "";
            rows = [];
            groups.forEach(function (g, gi) {
                if (!g.items || !g.items.length) return;
                if (g.label) {
                    var label = document.createElement("li");
                    label.className = "ui-palette__group";
                    label.setAttribute("role", "presentation");
                    label.textContent = g.label;
                    list.appendChild(label);
                }
                g.items.forEach(function (item) {
                    var li = document.createElement("li");
                    li.id = dialog.id + "-opt-" + rows.length;
                    li.className = "ui-palette__item";
                    li.setAttribute("role", "option");
                    li.setAttribute("aria-selected", "false");
                    var a = document.createElement("a");
                    a.href = safeURL(item.href) || "#";
                    a.tabIndex = -1;
                    var text = document.createElement("span");
                    text.className = "ui-palette__label";
                    text.appendChild(highlight(item.label || "", q));
                    a.appendChild(lead(item));
                    a.appendChild(text);
                    if (item.hint) {
                        var hint = document.createElement("span");
                        hint.className = "ui-palette__hint";
                        hint.textContent = item.hint;
                        a.appendChild(hint);
                    }
                    li.appendChild(a);
                    list.appendChild(li);
                    rows.push({el: li, a: a, item: item});
                });
            });
            if (!rows.length && query) {
                var none = document.createElement("li");
                none.className = "ui-palette__none";
                none.setAttribute("role", "presentation");
                none.textContent = emptyText + " “" + query + "”.";
                list.appendChild(none);
            }
            input.setAttribute("aria-expanded", rows.length ? "true" : "false");
            status.textContent = rows.length ? rows.length + (rows.length === 1 ? " result" : " results") : (query ? emptyText + " " + query : "");
            setActive(rows.length ? 0 : -1);
        }

        function load() {
            if (!endpoint) return;
            var mine = ++seq;
            if (controller && controller.abort) controller.abort();
            controller = window.AbortController ? new AbortController() : null;
            var url = endpoint + (endpoint.indexOf("?") < 0 ? "?" : "&") + "q=" + encodeURIComponent(input.value.trim());
            dialog.setAttribute("data-busy", "");
            window.fetch(url, {headers: {Accept: "application/json"}, credentials: "same-origin", signal: controller ? controller.signal : undefined})
                .then(function (res) { return res.ok ? res.json() : {}; })
                .then(function (data) {
                    if (mine !== seq) return;
                    remote = (data && data.groups) || [];
                    dialog.removeAttribute("data-busy");
                    if (dialog.open) render();
                })
                .catch(function (err) {
                    if (mine !== seq || (err && err.name === "AbortError")) return;
                    remote = [];
                    dialog.removeAttribute("data-busy");
                    if (dialog.open) render();
                });
        }

        // Enter on a row: a real click on its link (so hx-boost and view
        // transitions still apply), or a hidden form for a POST row.
        function activate(row) {
            if (!row) return;
            var f = row.item.form;
            var action = formTarget(row.item);
            if (action) {
                var form = document.createElement("form");
                form.method = String(f.method).toLowerCase() === "get" ? "get" : "post";
                form.action = action;
                form.hidden = true;
                Object.keys(f.fields || {}).forEach(function (name) {
                    var field = document.createElement("input");
                    field.type = "hidden";
                    field.name = name;
                    field.value = f.fields[name];
                    form.appendChild(field);
                });
                dialog.appendChild(form);
                form.submit();
                return;
            }
            row.a.click();
        }

        function open() {
            window.clearTimeout(closing);
            closing = 0;
            dialog.classList.remove("is-closing");
            if (dialog.open) return;
            opener = document.activeElement;
            dialog.showModal();
            input.value = "";
            remote = [];
            render();
            input.focus();
            load();
        }

        function close() {
            if (!dialog.open || closing) return;
            if (reducedMotion()) { dialog.close(); return; }
            dialog.classList.add("is-closing");
            closing = window.setTimeout(function () {
                dialog.close();
                dialog.classList.remove("is-closing");
                closing = 0;
            }, CLOSE_MS);
        }

        self.open = open;
        self.close = close;
        self.closing = function () { return !!closing; };
        self.dialog = dialog;

        input.addEventListener("input", function () {
            render();
            window.clearTimeout(timer);
            timer = window.setTimeout(load, DEBOUNCE_MS);
        });
        input.addEventListener("keydown", function (e) {
            if (e.key === "ArrowDown") { e.preventDefault(); setActive(active + 1); }
            else if (e.key === "ArrowUp") { e.preventDefault(); setActive(active - 1); }
            else if (e.key === "Home" && rows.length) { e.preventDefault(); setActive(0); }
            else if (e.key === "End" && rows.length) { e.preventDefault(); setActive(rows.length - 1); }
            else if (e.key === "Enter") { e.preventDefault(); activate(rows[active]); }
        });
        list.addEventListener("mousemove", function (e) {
            var li = e.target.closest && e.target.closest("[role=option]");
            if (!li) return;
            rows.forEach(function (row, i) { if (row.el === li && i !== active) setActive(i); });
        });
        // Following a result closes the dialog even when the page stays mounted (hx-boost).
        list.addEventListener("click", function (e) {
            var li = e.target.closest && e.target.closest("[role=option]");
            if (!li) return;
            var row = rows.filter(function (r) { return r.el === li; })[0];
            if (row && formTarget(row.item)) {
                e.preventDefault();
                activate(row);
                return;
            }
            opener = null;
            dialog.close();
        });
        dialog.addEventListener("click", function (e) { if (e.target === dialog) close(); });
        dialog.addEventListener("cancel", function (e) { e.preventDefault(); close(); });
        dialog.addEventListener("close", function () {
            window.clearTimeout(timer);
            seq++;
            if (controller && controller.abort) controller.abort();
            if (opener && opener.isConnected && opener.focus) opener.focus();
            opener = null;
        });
        mod(dialog);
    }

    var palettes = [];
    function palette(id) {
        var el = id ? document.getElementById(id) : document.querySelector("dialog[data-oui-palette]");
        if (!el || !el.matches("dialog[data-oui-palette]")) return null;
        for (var i = 0; i < palettes.length; i++) if (palettes[i].dialog === el) return palettes[i];
        if (typeof el.showModal !== "function") return null;
        var p = new Palette(el);
        palettes.push(p);
        return p;
    }

    // ── toast ───────────────────────────────────────────────────────────────
    var TONES = {neutral: "neutral", info: "info", success: "success", positive: "success", warning: "warning", danger: "danger", error: "danger", ai: "ai", spark: "ai"};
    var LIFETIME_MS = 4000;

    function dismissToast(toast) {
        if (!toast || !toast.isConnected || toast._leaving) return;
        toast._leaving = true;
        window.clearTimeout(toast._timer);
        if (reducedMotion()) { toast.remove(); return; }
        toast.classList.add("is-leaving");
        window.setTimeout(function () { toast.remove(); }, 200);
    }

    // Auto-dismiss that waits while the pointer is over the toast or focus is inside it.
    function autoDismiss(toast) {
        if (toast._adopted) return;
        toast._adopted = true;
        // data-lifetime (ms) on [data-toast-stack] tunes the delay for a host.
        var stack = toast.closest("[data-toast-stack]");
        var remaining = (stack && parseInt(stack.getAttribute("data-lifetime"), 10)) || LIFETIME_MS;
        var floor = Math.min(1000, remaining);
        var started = 0;
        function start() {
            window.clearTimeout(toast._timer);
            started = Date.now();
            toast._timer = window.setTimeout(function () { dismissToast(toast); }, remaining);
        }
        function pause() {
            window.clearTimeout(toast._timer);
            remaining = Math.max(floor, remaining - (Date.now() - started));
        }
        toast.addEventListener("mouseenter", pause);
        toast.addEventListener("focusin", pause);
        toast.addEventListener("mouseleave", function () { if (!toast.contains(document.activeElement)) start(); });
        toast.addEventListener("focusout", function (e) { if (!toast.contains(e.relatedTarget) && !toast.matches(":hover")) start(); });
        start();
    }

    function adoptToasts() {
        document.querySelectorAll("[data-toast-stack] [data-toast-auto]").forEach(autoDismiss);
    }

    function showToast(d) {
        if (typeof d === "string") d = {message: d};
        d = d || {};
        if (d.value !== undefined && d.message === undefined) d.message = d.value; // {"toast": "Saved"}
        var title = d.title ? String(d.title) : "";
        var message = d.message ? String(d.message) : "";
        var stack = document.querySelector("[data-toast-stack]");
        var tpl = document.querySelector("template[data-toast-template]");
        if (!stack || !tpl || !(title || message)) return null;
        var tone = TONES[d.tone] || "neutral";
        var toast = tpl.content.firstElementChild.cloneNode(true);
        if (tone !== "neutral") toast.classList.add("ui-toast--" + tone);
        toast.setAttribute("role", tone === "danger" ? "alert" : "status");
        var iconTpl = document.querySelector('template[data-toast-icon-for="' + tone + '"]');
        var slot = toast.querySelector("[data-toast-icon]");
        if (iconTpl && slot) slot.appendChild(iconTpl.content.cloneNode(true));
        toast.querySelector("[data-toast-title]").textContent = title;
        toast.querySelector("[data-toast-message]").textContent = message;
        stack.appendChild(toast);
        // Warnings and errors stay until dismissed unless the payload says otherwise.
        var auto = d.auto === undefined ? (tone !== "danger" && tone !== "warning") : !!d.auto;
        if (auto) autoDismiss(toast);
        return toast;
    }

    // ── theme switch ────────────────────────────────────────────────────────
    var SWITCH = "[data-oui-theme-switch] input[type=radio]";

    // theme.js normally owns this; the fallback keeps a switch working (after
    // first paint) on a page that did not load it.
    function themeApi(el) {
        if (window.OrbitalTheme) return window.OrbitalTheme;
        var group = el.closest("[data-oui-theme-switch]");
        var key = (group && group.getAttribute("data-storage-key")) || "oui-theme";
        function read() {
            try { var v = window.localStorage.getItem(key); return v === "light" || v === "dark" ? v : "system"; } catch (err) { return "system"; }
        }
        return {
            getPreference: read,
            setPreference: function (v) {
                try { window.localStorage.setItem(key, v); } catch (err) { /* this visit only */ }
                var dark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
                var theme = v === "system" ? (dark ? "dark" : "light") : v;
                document.documentElement.setAttribute("data-theme", theme);
                document.documentElement.style.colorScheme = theme;
            },
        };
    }

    function syncSwitches() {
        document.querySelectorAll(SWITCH).forEach(function (radio) {
            radio.checked = radio.value === themeApi(radio).getPreference();
        });
    }

    // ── wiring ──────────────────────────────────────────────────────────────
    document.addEventListener("click", function (e) {
        var t = e.target;
        if (!t || !t.closest) return;
        var opener = t.closest("[data-palette-open]");
        if (opener) {
            var p = palette(opener.getAttribute("data-palette-open"));
            if (p) p.open();
            return;
        }
        var shell = shellOf(t);
        if (t.closest("[data-shell-open]") && shell) openNav(shell);
        else if ((t.closest("[data-shell-close]") || t.closest("[data-shell-scrim]")) && shell) closeNav(shell, true);
        else if (shell && navOpen(shell) && t.closest(".ui-shell__sidebar a[href]")) closeNav(shell, false);
        var close = t.closest("[data-toast-close]");
        if (close) dismissToast(close.closest(".ui-toast"));
    });

    document.addEventListener("keydown", function (e) {
        if ((e.metaKey || e.ctrlKey) && !e.altKey && String(e.key).toLowerCase() === "k") {
            var p = palette();
            if (!p) return;
            e.preventDefault();
            // A press during the close animation reopens it rather than being lost.
            if (p.dialog.open && !p.closing()) p.close();
            else p.open();
            return;
        }
        if (e.key !== "Escape") return;
        var toast = e.target && e.target.closest ? e.target.closest(".ui-toast") : null;
        if (toast) { dismissToast(toast); return; }
        if (document.querySelector("dialog[open]")) return;
        openShells().forEach(function (s) { closeNav(s, true); });
    });

    // HX-Trigger: {"toast": {...}} reaches here as a "toast" event on the document.
    document.addEventListener("toast", function (e) { showToast(e.detail); });
    document.addEventListener("htmx:afterSettle", adoptToasts);

    document.addEventListener("change", function (e) {
        var t = e.target;
        if (!t || !t.matches || !t.matches(SWITCH)) return;
        themeApi(t).setPreference(t.value);
        syncSwitches();
    });
    document.addEventListener("oui:theme", syncSwitches);

    window.OrbitalShell = {
        toast: showToast,
        openPalette: function (id) { var p = palette(id); if (p) p.open(); },
        closePalette: function (id) { var p = palette(id); if (p) p.close(); },
    };

    ready(function () {
        document.querySelectorAll("[data-oui-shell]").forEach(syncShell);
        document.querySelectorAll("[data-palette-open]").forEach(mod);
        adoptToasts();
        syncSwitches();
    });
})();
