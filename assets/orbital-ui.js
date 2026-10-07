/* Orbital UI runtime: the behaviour behind ModalShell, ConfirmButton, Drawer
 * and EntityPicker.
 *
 * Load order (the script must run before Alpine starts, and after any host
 * runtime that defines window.LedgerUI):
 *
 *   <script src="/static/app.js" defer></script>          <!-- host runtime, if any -->
 *   <script src="/static/orbital-ui.js" defer></script>
 *   <script src="/static/alpine.min.js" defer></script>   <!-- or alpinejs-csp -->
 *
 * Component markup is CSP-safe: x-data="ouiDialog" | "ouiDrawer" | "ouiPicker"
 * name factories registered here with Alpine.data() on `alpine:init`, and every
 * directive is a bare property or method reference (x-on:click="backdrop",
 * x-show="open", x-bind:class="row.cls"), which is all Alpine's CSP build
 * evaluates. The picker reads its config from the data-oui-picker JSON
 * attribute. Both the standard and the CSP Alpine build run the same markup.
 *
 * The factories also work for the older markup that called window.LedgerUI.*
 * (x-data="LedgerUI.dialog()"): this file installs those names under
 * window.OrbitalUI and aliases window.LedgerUI to the same object.
 *
 * Host runtime. When the host already ships window.LedgerUI (ledger-agent's
 * app.js), this file does NOT install its own functions or click delegation,
 * so the two never double-handle a click. It still registers ouiDialog /
 * ouiDrawer / ouiPicker, delegating to the host's LedgerUI.dialog() /
 * drawer() / picker(cfg) factories, so the new markup runs on the host's
 * behaviour.
 *
 * Contract:
 *   [data-open-dialog="id"]   opens <dialog id> modally, focus moves inside
 *   [data-close-dialog]       closes the enclosing dialog, focus returns
 *   [data-open-drawer="id"]   opens the drawer overlay id (htmx may fill it)
 *   [data-close-drawer]       closes the enclosing drawer
 *   [data-drawer-fullscreen]  toggles .drawer-panel--full on its panel
 *   HX-Trigger: closeDialog | drawerClose   closes the open overlay after a
 *                                           successful htmx swap
 * The runtime needs no inline script and no eval beyond Alpine's own
 * expression evaluation.
 */
(function () {
    "use strict";

    var existing = window.LedgerUI;
    if (existing && typeof existing.openDialog === "function") {
        window.OrbitalUI = window.OrbitalUI || existing;
        registerAlpine(existing);
        return;
    }

    // The CSP build of Alpine evaluates only bare property or method
    // references: no calls with arguments, negation, ternaries, assignments
    // or x-model. The picker markup therefore reads these precomputed members,
    // and they are added to whichever picker implementation is active (this
    // file's or the host's), so both run the same markup.
    function pickerView() {
        return {
            get rows() {
                var self = this;
                return this.options().map(function (o, i) {
                    return {
                        id: o.id,
                        label: o.label,
                        sublabel: o.sublabel || "",
                        index: i,
                        domId: self.optId(i),
                        selected: self.picked === o.id,
                        cls: "entity-picker__option" + (i === self.idx ? " entity-picker__option--active" : ""),
                        btnCls: o.create ? "entity-picker__option-btn--create-row" : "",
                    };
                });
            },
            get expanded() { return this.open ? "true" : "false"; },
            get activeDescendant() { return this.activeId(); },
            get listLabel() { return this.cfg.label; },
            get showClear() { return !!this.picked && !this.open; },
            get showEmpty() { return !this.busy && !this.failed && !this.creating && this.open && this.options().length === 0; },
            onInput: function (e) {
                this.query = e.target.value;
                this.typing();
            },
            pick: function (e) {
                this.choose(this.options()[parseInt(e.currentTarget.getAttribute("data-index"), 10)]);
            },
            hover: function (e) {
                this.idx = parseInt(e.currentTarget.getAttribute("data-index"), 10);
            },
        };
    }

    function pickerConfig(el) {
        try {
            return JSON.parse(el.getAttribute("data-oui-picker") || "{}") || {};
        } catch (err) {
            return {};
        }
    }

    // Registers the factories the markup names. The data callbacks run with
    // Alpine's magics on `this`, so $el reaches the element that asked.
    function registerAlpine(api) {
        function add() {
            var A = window.Alpine;
            A.data("ouiDialog", function () {
                var d = (api.dialog && api.dialog()) || {};
                if (!d.cancel) d.cancel = function (e) { api.closeDialog(e.currentTarget.id); };
                return d;
            });
            A.data("ouiDrawer", function () {
                var d = (api.drawer && api.drawer()) || {};
                if (!d.escape) d.escape = function (e) { api.closeDrawer(e.currentTarget.id); };
                return d;
            });
            // A dismissible Alert: argument-free references keep it CSP-safe.
            A.data("ouiDismiss", function () {
                return { visible: true, dismiss: function () { this.visible = false; } };
            });
            A.data("ouiPicker", function () {
                var p = api.picker(pickerConfig(this.$el));
                Object.defineProperties(p, Object.getOwnPropertyDescriptors(pickerView()));
                return p;
            });
        }
        if (window.Alpine && typeof window.Alpine.data === "function") add();
        else document.addEventListener("alpine:init", add);
    }

    var UI = window.OrbitalUI = window.OrbitalUI || {};
    window.LedgerUI = UI;

    var FOCUSABLE = "[autofocus], input:not([type=hidden]):not([disabled]), select:not([disabled]), textarea:not([disabled]), button:not([disabled]), a[href]";

    function restoreFocus(store, id) {
        var opener = store[id];
        if (opener && opener.isConnected && opener.focus) opener.focus();
        delete store[id];
    }

    function outside(rect, e) {
        return e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom;
    }

    // ── dialogs (ModalShell, ConfirmButton) ─────────────────────────────────
    UI._openers = {};

    UI.openDialog = function (id) {
        var dialog = document.getElementById(id);
        if (!dialog || dialog.open || typeof dialog.showModal !== "function") return;
        UI._openers[id] = document.activeElement;
        dialog.showModal();
        // The body's first field, not the header's close button.
        var body = dialog.querySelector(".ui-dialog__body");
        var first = (body && body.querySelector(FOCUSABLE)) || dialog.querySelector(FOCUSABLE);
        if (first && first.focus) first.focus();
    };

    UI.closeDialog = function (id) {
        var dialog = document.getElementById(id);
        if (dialog && dialog.open) dialog.close();
        restoreFocus(UI._openers, id);
    };

    // A full-page 422 re-render arrives with <dialog open> in the markup. The
    // open attribute alone is non-modal (no ::backdrop, no inert page), so
    // promote it to a real modal.
    UI.promoteOpenDialog = function () {
        var dialog = document.querySelector("dialog[open][id]:not(:modal)");
        if (!dialog || typeof dialog.showModal !== "function") return;
        dialog.close();
        UI.openDialog(dialog.id);
    };

    // Alpine factory on every dialog. A factory, not a shared object: Alpine
    // defines non-configurable magics on whatever x-data evaluates to.
    UI.dialog = function () {
        return {
            backdrop: function (e) {
                var el = e.currentTarget;
                // Keyboard activation of a child reports (0, 0); only clicks
                // whose target is the dialog box itself can be backdrop clicks.
                if (e.target !== el) return;
                if (outside(el.getBoundingClientRect(), e)) UI.closeDialog(el.id);
            },
        };
    };

    // ── drawers (Drawer) ────────────────────────────────────────────────────
    UI._drawerOpeners = {};

    UI.openDrawer = function (id) {
        var overlay = document.getElementById(id);
        var panel = overlay && overlay.querySelector(".drawer-panel");
        if (!panel || panel.classList.contains("is-open")) return;
        UI._drawerOpeners[id] = document.activeElement;
        var region = overlay.querySelector(".drawer-content");
        if (region) region.scrollTop = 0;
        panel.classList.add("is-open");
        document.body.classList.add("drawer-lock");
        var content = overlay.querySelector(".drawer-content");
        var first = (content && content.querySelector(FOCUSABLE)) || panel.querySelector(FOCUSABLE);
        if (first && first.focus) first.focus();
        else if (panel.focus) panel.focus();
        updateTabsOverflow(panel.querySelector("[data-log-tabs-wrap]"));
    };

    UI.closeDrawer = function (id) {
        var overlay = document.getElementById(id);
        var panel = overlay && overlay.querySelector(".drawer-panel");
        if (panel && panel.classList.contains("is-open")) {
            panel.classList.remove("is-open", "drawer-panel--full");
            document.body.classList.remove("drawer-lock");
            var toggle = panel.querySelector("[data-drawer-fullscreen]");
            if (toggle) setFullscreenLabel(toggle, false);
        }
        restoreFocus(UI._drawerOpeners, id);
    };

    UI.drawer = function () {
        return {
            backdrop: function (e) {
                var el = e.currentTarget;
                var panel = el.querySelector(".drawer-panel");
                if (!panel || (e.target && panel.contains(e.target))) return;
                if (outside(panel.getBoundingClientRect(), e)) UI.closeDrawer(el.id);
            },
        };
    };

    // Kept in step with components.drawerFullscreenLabel.
    function setFullscreenLabel(btn, full) {
        var label = full ? "Contract back to the sheet" : "Expand this panel to the whole screen";
        btn.setAttribute("aria-pressed", full ? "true" : "false");
        btn.setAttribute("aria-label", label);
        btn.title = label;
    }

    // Tabbed drawers: the host fills the track with tabs; the runtime only
    // reveals the scroll buttons when the strip overflows.
    function updateTabsOverflow(wrap) {
        if (!wrap) return;
        var track = wrap.querySelector(".drawer-tabs__track");
        if (!track) return;
        var max = track.scrollWidth - track.clientWidth;
        var state = "none";
        if (max > 1) {
            var atStart = track.scrollLeft <= 1;
            var atEnd = track.scrollLeft >= max - 1;
            state = atStart ? "end" : atEnd ? "start" : "both";
        }
        wrap.setAttribute("data-overflow", state);
    }
    UI.updateDrawerTabs = updateTabsOverflow;

    // ── entity picker (EntityPicker) ────────────────────────────────────────
    var CREATE_ROW_ID = "@create";

    function newRequestKey() {
        var c = window.crypto;
        if (c && c.randomUUID) return c.randomUUID();
        if (!c || !c.getRandomValues) return "";
        var b = new Uint8Array(16);
        c.getRandomValues(b);
        b[6] = (b[6] & 0x0f) | 0x40;
        b[8] = (b[8] & 0x3f) | 0x80;
        var h = Array.prototype.map.call(b, function (x) { return x.toString(16).padStart(2, "0"); }).join("");
        return h.slice(0, 8) + "-" + h.slice(8, 12) + "-" + h.slice(12, 16) + "-" + h.slice(16, 20) + "-" + h.slice(20);
    }

    function csrfFor(el) {
        var form = el && el.closest("form");
        var field = form && form.querySelector('[name="csrf"], [name="csrf_token"], [name="_csrf"]');
        if (field) return {name: field.getAttribute("name"), value: field.value};
        var meta = document.querySelector('meta[name="csrf-token"]');
        return meta ? {name: "csrf", value: meta.getAttribute("content") || "", header: true} : null;
    }

    UI.picker = function (cfg) {
        var DEBOUNCE_MS = 300;
        cfg = cfg || {};
        cfg.listId = (cfg.id || "picker") + "-list";
        return {
            cfg: cfg,
            query: cfg.valueLabel || "",
            display: cfg.valueLabel || "",
            picked: cfg.value || "",
            open: false,
            idx: -1,
            items: [],
            busy: false,
            failed: false,
            creating: false,
            createFailed: false,
            createRequestKey: "",
            createRequestText: "",
            timer: null,
            pending: false, // typed text not yet searched (debounce window)
            loadedOnce: false,
            seq: 0,
            list: function () { return this.$refs.list; },
            optId: function (i) { return this.cfg.listId + "-opt-" + i; },
            // One reader for both row kinds: the template iterates it and the
            // keyboard walks it, so aria-activedescendant never points at a row
            // that is not on screen. The create row goes last so Enter keeps
            // picking the first real match.
            options: function () {
                var create = this.createOption();
                return create ? this.items.concat([create]) : this.items;
            },
            createOption: function () {
                var c = this.cfg.create;
                if (!c || !c.endpoint) return null;
                var q = this.query.trim();
                if (!q) return null;
                var ql = q.toLowerCase();
                for (var i = 0; i < this.items.length; i++) {
                    if (((this.items[i] && this.items[i].label) || "").trim().toLowerCase() === ql) return null;
                }
                return {
                    id: CREATE_ROW_ID,
                    label: c.noun ? "Create " + c.noun + ' "' + q + '"' : 'Create "' + q + '"',
                    sublabel: "",
                    create: true,
                    text: q,
                };
            },
            activeId: function () {
                var n = this.options().length;
                if (!this.open || this.idx < 0 || this.idx >= n) return null;
                return this.optId(this.idx);
            },
            // A transformed ancestor (the drawer's slide) becomes the fixed
            // containing block, so offset the list by that ancestor's box.
            fixedOrigin: function () {
                var el = this.list() && this.list().parentElement;
                while (el && el !== document.documentElement) {
                    var cs = window.getComputedStyle(el);
                    if (cs.transform !== "none" || cs.perspective !== "none" ||
                        (cs.translate && cs.translate !== "none") || (cs.rotate && cs.rotate !== "none") ||
                        (cs.scale && cs.scale !== "none") || cs.filter !== "none" ||
                        (cs.backdropFilter && cs.backdropFilter !== "none") ||
                        /(paint|layout|strict|content)/.test(cs.contain || "")) {
                        var r = el.getBoundingClientRect();
                        return {left: r.left, top: r.top};
                    }
                    el = el.parentElement;
                }
                return null;
            },
            place: function () {
                var input = this.$refs.input;
                var listEl = this.list();
                if (!input || !listEl) return;
                var rect = input.getBoundingClientRect();
                var height = Math.min(256, window.innerHeight * 0.4);
                var below = window.innerHeight - rect.bottom;
                var up = below < height && rect.top > below;
                var top = up ? Math.max(8, rect.top - 4 - height) : rect.bottom + 4;
                var left = rect.left;
                var origin = this.fixedOrigin();
                if (origin) {
                    top -= origin.top;
                    left -= origin.left;
                }
                listEl.style.top = top + "px";
                listEl.style.left = left + "px";
                listEl.style.width = rect.width + "px";
                listEl.style.maxHeight = height + "px";
            },
            fetch: function (q) {
                var self = this;
                var mine = ++this.seq;
                this.busy = true;
                this.failed = false;
                var url = this.cfg.endpoint + (this.cfg.endpoint.indexOf("?") === -1 ? "?" : "&") + "q=" + encodeURIComponent(q);
                window.fetch(url, {headers: {Accept: "application/json"}, credentials: "same-origin"})
                    .then(function (res) {
                        if (!res.ok) throw new Error("picker HTTP " + res.status);
                        return res.json();
                    })
                    .then(function (data) {
                        if (mine !== self.seq) return; // a newer query answered first
                        self.items = (data && data.items) || [];
                        self.idx = self.options().length ? 0 : -1;
                        self.busy = false;
                        self.open = true;
                        self.$nextTick(function () { self.place(); });
                    })
                    .catch(function () {
                        if (mine !== self.seq) return;
                        self.items = [];
                        self.idx = -1;
                        self.busy = false;
                        self.failed = true;
                        self.open = true;
                    });
            },
            typing: function () {
                var self = this;
                if (this.query.trim() !== this.display && this.picked) {
                    this.picked = "";
                    this.display = "";
                }
                this.createFailed = false;
                this.pending = true;
                clearTimeout(this.timer);
                this.timer = setTimeout(function () {
                    self.pending = false;
                    self.open = true;
                    self.fetch(self.query.trim());
                }, DEBOUNCE_MS);
            },
            ensureOpen: function () {
                if (this.open) return;
                this.open = true;
                if (!this.loadedOnce && !this.busy) {
                    this.loadedOnce = true;
                    this.fetch(this.query.trim());
                } else {
                    var self = this;
                    this.$nextTick(function () { self.place(); });
                }
            },
            invalidate: function () {
                this.loadedOnce = false;
                this.items = [];
                this.createFailed = false;
            },
            onBlur: function () {
                var self = this;
                // Delay so a mousedown on an option can still choose() it.
                setTimeout(function () {
                    // Focus came back (click, clear button) before this ran:
                    // the field is live again, so neither close nor revert it.
                    if (self.$refs.input && document.activeElement === self.$refs.input) return;
                    self.open = false;
                    if (self.query.trim() !== self.display && !self.picked) self.query = self.display;
                }, 120);
            },
            choose: function (item) {
                if (!item) return;
                if (item.create) return this.createNew(item);
                this.picked = item.id;
                this.display = item.label;
                this.query = item.label;
                this.open = false;
                this.idx = -1;
                if (this.$refs.input) this.$refs.input.focus();
                var hidden = this.$refs.hidden;
                if (hidden) {
                    var self = this;
                    this.$nextTick(function () {
                        hidden.dispatchEvent(new Event("change", {bubbles: true}));
                        self.$el.dispatchEvent(new CustomEvent("entity-picked", {bubbles: true, detail: {id: item.id, label: item.label}}));
                    });
                }
            },
            // POST the typed text; the response is just another option, so a
            // created record and a selected one end in the same state.
            createNew: function (item) {
                if (this.creating) return; // one request in flight
                var c = this.cfg.create;
                if (!c || !c.endpoint) return;
                var text = (item && item.text) || this.query.trim();
                if (!text) return;
                if (this.createRequestText !== text || !this.createRequestKey) {
                    this.createRequestText = text;
                    this.createRequestKey = newRequestKey();
                    if (!this.createRequestKey) {
                        this.createFailed = true;
                        return;
                    }
                }
                var self = this;
                this.creating = true;
                this.createFailed = false;
                var body = {request_key: this.createRequestKey};
                body[c.field || "name"] = text;
                var headers = {"Content-Type": "application/json", Accept: "application/json"};
                var csrf = csrfFor(this.$el);
                if (csrf) {
                    body[csrf.name] = csrf.value;
                    if (csrf.header) headers["X-CSRF-Token"] = csrf.value;
                }
                var form = this.$el && this.$el.closest("form");
                window.fetch(c.endpoint, {method: "POST", headers: headers, body: JSON.stringify(body), credentials: "same-origin"})
                    .then(function (res) {
                        if (!res.ok) throw new Error("picker create HTTP " + res.status);
                        return res.json();
                    })
                    .then(function (data) {
                        if (!data || !data.id) throw new Error("picker create returned an incomplete record");
                        if (c.requireBookScope) {
                            var book = form && form.querySelector('[name="book_id"]');
                            if (!data.book_id || !book || String(data.book_id) !== String(book.value)) {
                                throw new Error("picker create returned a record from another scope");
                            }
                        }
                        self.creating = false;
                        self.invalidate();
                        self.createRequestKey = "";
                        self.createRequestText = "";
                        self.choose({id: data.id, label: data.label || text, sublabel: data.sublabel || ""});
                    })
                    .catch(function () {
                        // Keep the text and the row: retry is a second Enter
                        // with the same request key, so it cannot mint twice.
                        self.creating = false;
                        self.createFailed = true;
                        self.open = true;
                        self.$nextTick(function () { self.place(); });
                    });
            },
            clearPicked: function () {
                this.picked = "";
                this.display = "";
                this.query = "";
                this.open = false;
                this.idx = -1;
                this.invalidate();
                if (this.$refs.input) this.$refs.input.focus();
            },
            onKey: function (e) {
                var key = e.key;
                var rows = this.options();
                if (!this.open && (key === "ArrowDown" || key === "ArrowUp" || key === "Enter")) {
                    e.preventDefault();
                    this.ensureOpen();
                    return;
                }
                if (!this.open) return;
                if (key === "ArrowDown") {
                    e.preventDefault();
                    if (rows.length) this.idx = Math.min(this.idx + 1, rows.length - 1);
                } else if (key === "ArrowUp") {
                    e.preventDefault();
                    this.idx = Math.max(this.idx - 1, 0);
                } else if (key === "Enter") {
                    if (this.pending || this.busy) {
                        // The rows on screen belong to an older query; never let
                        // Enter submit the form or pick a stale row meanwhile.
                        e.preventDefault();
                    } else if (this.idx >= 0 && this.idx < rows.length) {
                        e.preventDefault();
                        this.choose(rows[this.idx]);
                    }
                    // Idle with no active option: Enter submits the form.
                } else if (key === "Escape") {
                    e.preventDefault();
                    this.open = false;
                    this.idx = -1;
                    if (!this.picked) this.query = this.display;
                } else if (key === "Tab") {
                    this.open = false;
                    this.idx = -1;
                }
            },
        };
    };

    function repositionPickers() {
        document.querySelectorAll(".entity-picker[x-data]").forEach(function (el) {
            var scope = el._x_dataStack && el._x_dataStack[0];
            if (scope && scope.open && scope.place) scope.place();
        });
    }

    // ── Alpine registration (CSP-safe markup) ───────────────────────────────
    registerAlpine(UI);

    // ── delegation ──────────────────────────────────────────────────────────
    document.addEventListener("click", function (e) {
        var t = e.target;
        if (!t || !t.closest) return;
        var el = t.closest("[data-open-dialog]");
        if (el) {
            e.preventDefault();
            UI.openDialog(el.getAttribute("data-open-dialog"));
            return;
        }
        el = t.closest("[data-open-drawer]");
        if (el) {
            // Openers double as htmx triggers; htmx handles the request on its
            // own listener, so only a default navigation is stopped here.
            e.preventDefault();
            UI.openDrawer(el.getAttribute("data-open-drawer"));
            return;
        }
        el = t.closest("[data-close-dialog]");
        if (el) {
            var dialog = el.closest("dialog");
            if (dialog) UI.closeDialog(dialog.id);
            return;
        }
        el = t.closest("[data-close-drawer]");
        if (el) {
            var overlay = el.closest(".drawer-overlay");
            if (overlay) UI.closeDrawer(overlay.id);
            return;
        }
        el = t.closest("[data-drawer-fullscreen]");
        if (el) {
            var panel = el.closest(".drawer-panel");
            if (panel) setFullscreenLabel(el, panel.classList.toggle("drawer-panel--full"));
            return;
        }
        el = t.closest("[data-log-scroll]");
        if (el) {
            var wrap = el.closest("[data-log-tabs-wrap]");
            var track = wrap && wrap.querySelector(".drawer-tabs__track");
            if (track) {
                track.scrollBy({left: Math.max(120, track.clientWidth * 0.8) * parseInt(el.getAttribute("data-log-scroll"), 10), behavior: "smooth"});
            }
        }
    });

    document.addEventListener("scroll", function (e) {
        repositionPickers();
        var t = e.target;
        if (t && t.classList && t.classList.contains("drawer-tabs__track")) updateTabsOverflow(t.closest("[data-log-tabs-wrap]"));
    }, true);
    window.addEventListener("resize", function () {
        repositionPickers();
        document.querySelectorAll("[data-log-tabs-wrap]").forEach(updateTabsOverflow);
    });

    // Server-driven close after a successful swap. 422 validation swaps carry
    // no such header, so a failed submit keeps its overlay open. This runs on
    // afterSettle, after htmx restores focus into the swapped markup, so the
    // focus returned to the opener is not taken back into a closed overlay.
    document.addEventListener("htmx:afterSettle", function (evt) {
        var xhr = evt.detail && evt.detail.xhr;
        var header = xhr && xhr.getResponseHeader && xhr.getResponseHeader("HX-Trigger");
        if (!header) return;
        if (header.indexOf("closeDialog") !== -1) {
            var open = document.querySelector("dialog[open]");
            if (open) UI.closeDialog(open.id);
        }
        if (header.indexOf("drawerClose") !== -1) {
            var sheet = document.querySelector(".drawer-panel.is-open");
            var overlay = sheet && sheet.closest(".drawer-overlay");
            if (overlay) UI.closeDrawer(overlay.id);
        }
    });

    document.addEventListener("htmx:afterSwap", function (evt) {
        var detail = evt.detail || {};
        // A freshly loaded drawer editor starts at its heading.
        var target = detail.target;
        var verb = String((detail.requestConfig && detail.requestConfig.verb) || "").toUpperCase();
        if (target && target.matches && target.matches(".drawer-content") && verb === "GET" &&
            target.closest(".drawer-overlay") && target.closest(".drawer-overlay").querySelector(".drawer-panel.is-open")) {
            target.scrollTop = 0;
            var heading = target.querySelector("h2, h3, h4");
            if (heading) {
                heading.setAttribute("tabindex", "-1");
                heading.focus({preventScroll: true});
            }
        }
    });

    // Autogrow textareas (data-autogrow): CSS field-sizing does the work where
    // supported; this fallback sizes the box to its content elsewhere.
    if (!(window.CSS && CSS.supports && CSS.supports("field-sizing", "content"))) {
        var grow = function (el) {
            el.style.height = "auto";
            el.style.height = el.scrollHeight + "px";
        };
        document.addEventListener("input", function (e) {
            if (e.target && e.target.matches && e.target.matches("textarea[data-autogrow]")) grow(e.target);
        });
        var growAll = function (root) {
            (root || document).querySelectorAll("textarea[data-autogrow]").forEach(grow);
        };
        document.addEventListener("htmx:afterSwap", function (e) { growAll(e.target); });
        if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { growAll(); });
        else growAll();
    }

    // htmx swaps validation errors (422) into the form so dialogs and drawers
    // stay open with inline errors; other 4xx/5xx keep htmx's default.
    document.addEventListener("htmx:beforeSwap", function (evt) {
        var xhr = evt.detail && evt.detail.xhr;
        if (xhr && xhr.status === 422) {
            evt.detail.shouldSwap = true;
            evt.detail.isError = false;
        }
    });

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", UI.promoteOpenDialog);
    } else {
        UI.promoteOpenDialog();
    }
})();
