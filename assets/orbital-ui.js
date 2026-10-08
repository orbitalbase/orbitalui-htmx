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
 *
 * The shell runtime (AppShell sidebar, CommandPalette, toasts, ThemeSwitch) is
 * a second scope at the end of this file; it is plain DOM delegation, needs no
 * Alpine and runs even when a host runtime owns window.LedgerUI.
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

/* ── Shell runtime: AppShell, CommandPalette, Toast, ThemeSwitch ─────────────
 *
 * Plain DOM delegation (no Alpine, no eval, no inline styles), so it behaves
 * the same under the standard and the CSP Alpine builds and when a host
 * runtime owns window.LedgerUI. It is a separate scope from the runtime above
 * on purpose: the host early-return there must not switch the shell off.
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
