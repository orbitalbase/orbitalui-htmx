// A stand-in for a host app runtime (ledger-agent's app.js): it defines
// window.LedgerUI with its own dialog/drawer/picker factories and click
// handling. browsercheck loads it BEFORE orbital-ui.js on the ?host=1 page to
// prove the library keeps out of the way while still registering the Alpine
// adapters (ouiDialog/ouiDrawer/ouiPicker) on top of these factories.
(function () {
    "use strict";
    var UI = window.LedgerUI = {host: true, calls: {openDialog: 0, openDrawer: 0, backdrop: 0, drawerBackdrop: 0, choose: 0, picker: 0, dialog: 0}};

    UI.openDialog = function (id) {
        UI.calls.openDialog++;
        var d = document.getElementById(id);
        if (d && !d.open) d.showModal();
    };
    UI.closeDialog = function (id) {
        var d = document.getElementById(id);
        if (d && d.open) d.close();
    };
    UI.dialog = function () {
        UI.calls.dialog++;
        return {
            backdrop: function (e) {
                var el = e.currentTarget;
                if (e.target !== el) return;
                var r = el.getBoundingClientRect();
                if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) {
                    UI.calls.backdrop++;
                    UI.closeDialog(el.id);
                }
            },
        };
    };
    UI.openDrawer = function (id) {
        UI.calls.openDrawer++;
        var p = document.querySelector("#" + id + " .drawer-panel");
        if (p) p.classList.add("is-open");
    };
    UI.closeDrawer = function (id) {
        var p = document.querySelector("#" + id + " .drawer-panel");
        if (p) p.classList.remove("is-open");
    };
    UI.drawer = function () {
        return {
            backdrop: function (e) {
                var el = e.currentTarget;
                var panel = el.querySelector(".drawer-panel");
                if (panel && !panel.contains(e.target)) { UI.calls.drawerBackdrop++; UI.closeDrawer(el.id); }
            },
        };
    };
    UI.picker = function (cfg) {
        UI.calls.picker++;
        cfg = cfg || {};
        cfg.listId = (cfg.id || "picker") + "-list";
        return {
            cfg: cfg, query: "", display: "", picked: "", _open: false, idx: -1, items: [], busy: false, failed: false,
            creating: false, createFailed: false,
            // Like the shared picker: setting open shows or hides the list's top layer (Popover API).
            // The state lives on _open so Alpine tracks it.
            get open() { return this._open; },
            set open(value) { this._open = !!value; this.syncList(); },
            syncList: function () {
                var el = this.list();
                if (!el || typeof el.showPopover !== "function") return;
                var showing = el.matches(":popover-open");
                if (this._open) { if (!showing) el.showPopover(); }
                else if (showing) el.hidePopover();
            },
            list: function () { return this.$refs.list; },
            optId: function (i) { return this.cfg.listId + "-opt-" + i; },
            options: function () { return this.items; },
            activeId: function () { return this.open && this.idx >= 0 ? this.optId(this.idx) : null; },
            place: function () {},
            typing: function () {},
            ensureOpen: function () {
                var self = this;
                this.open = true;
                this.busy = true;
                window.fetch(this.cfg.endpoint + "?q=").then(function (r) { return r.json(); }).then(function (d) {
                    self.items = d.items || [];
                    self.idx = 0;
                    self.busy = false;
                });
            },
            onBlur: function () {},
            onKey: function () {},
            clearPicked: function () { this.picked = ""; this.display = ""; this.query = ""; },
            choose: function (item) {
                UI.calls.choose++;
                this.picked = item.id;
                this.display = item.label;
                this.query = item.label;
                this.open = false;
            },
        };
    };
    document.addEventListener("click", function (e) {
        var t = e.target.closest && e.target.closest("[data-open-dialog]");
        if (t) { e.preventDefault(); UI.openDialog(t.getAttribute("data-open-dialog")); return; }
        t = e.target.closest && e.target.closest("[data-open-drawer]");
        if (t) { UI.openDrawer(t.getAttribute("data-open-drawer")); return; }
        t = e.target.closest && e.target.closest("[data-close-dialog]");
        if (t) UI.closeDialog(t.closest("dialog").id);
    });
})();
