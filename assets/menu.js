// Orbital UI menu runtime. CSP-safe (no eval, no inline handlers) and
// optional: menus are native <details> elements that already open and close
// without it. This adds the behaviour people expect from a popover menu:
//   - opening one menu closes any other open menu;
//   - a click outside, or Escape, closes the open menu and restores focus;
//   - ArrowDown / ArrowUp / Home / End move focus between the items of an
//     open menu (ArrowDown on its closed trigger opens it);
//   - collapsible sidebar sections remember their state per browser.
(function () {
  "use strict";
  if (window.OrbitalMenu) return;

  var MENU = "details[data-ui-menu]";
  var SECTION = "details[data-nav-section]";
  var STORE_KEY = "oui.nav.sections";

  function openMenus() {
    return Array.prototype.slice.call(document.querySelectorAll(MENU + "[open]"));
  }

  function closeAll(except) {
    openMenus().forEach(function (menu) {
      if (menu !== except && !menu.contains(except)) menu.removeAttribute("open");
    });
  }

  document.addEventListener("toggle", function (event) {
    var target = event.target;
    if (!(target instanceof HTMLElement)) return;
    if (target.matches(MENU) && target.open) closeAll(target);
    if (target.matches(SECTION)) rememberSection(target);
  }, true);

  document.addEventListener("click", function (event) {
    var inside = event.target instanceof Element ? event.target.closest(MENU) : null;
    closeAll(inside);
    // Following a link inside a menu should also dismiss it (HTMX swaps keep
    // the page mounted, so the menu would otherwise stay open).
    if (inside && event.target.closest("a[href], button[type='submit']")) {
      inside.removeAttribute("open");
    }
  });

  document.addEventListener("keydown", function (event) {
    if (event.key !== "Escape") return;
    var menus = openMenus();
    if (!menus.length) return;
    var last = menus[menus.length - 1];
    last.removeAttribute("open");
    var summary = last.querySelector("summary");
    if (summary) summary.focus();
  });

  var ITEM = ".ui-menu__item, [role='menuitem']";

  document.addEventListener("keydown", function (event) {
    var keys = {ArrowDown: 1, ArrowUp: 1, Home: 1, End: 1};
    if (!keys[event.key] || event.altKey || event.ctrlKey || event.metaKey) return;
    var target = event.target instanceof Element ? event.target : null;
    var menu = target && target.closest(MENU);
    if (!menu) return;
    var summary = menu.querySelector(":scope > summary");
    if (!menu.open) {
      if (target === summary && event.key === "ArrowDown") {
        event.preventDefault();
        menu.setAttribute("open", "");
        var first = menu.querySelector(ITEM);
        if (first) first.focus();
      }
      return;
    }
    // Only the trigger and the items navigate: a control inside the panel
    // (ThemeSwitch radios, a search field) keeps its own arrow keys.
    if (target !== summary && !target.closest(ITEM)) return;
    var items = Array.prototype.slice.call(menu.querySelectorAll(ITEM)).filter(function (el) {
      return !el.disabled && el.getAttribute("aria-disabled") !== "true" && el.getClientRects().length > 0;
    });
    if (!items.length) return;
    event.preventDefault();
    var at = items.indexOf(target.closest(ITEM));
    var next;
    if (event.key === "Home") next = 0;
    else if (event.key === "End") next = items.length - 1;
    else if (event.key === "ArrowDown") next = at < 0 ? 0 : (at + 1) % items.length;
    else next = at < 0 ? items.length - 1 : (at - 1 + items.length) % items.length;
    items[next].focus();
  });

  function readStore() {
    try {
      return JSON.parse(window.localStorage.getItem(STORE_KEY) || "{}") || {};
    } catch (_) {
      return {};
    }
  }

  function rememberSection(section) {
    // The server always opens the section that holds the current page; never
    // persist a state that would hide it.
    if (section.querySelector("[aria-current='page']")) return;
    var store = readStore();
    store[section.getAttribute("data-nav-section")] = section.open;
    try {
      window.localStorage.setItem(STORE_KEY, JSON.stringify(store));
    } catch (_) { /* storage unavailable: the default state still works */ }
  }

  function restoreSections() {
    var store = readStore();
    document.querySelectorAll(SECTION).forEach(function (section) {
      var id = section.getAttribute("data-nav-section");
      if (!(id in store) || section.querySelector("[aria-current='page']")) return;
      if (store[id]) section.setAttribute("open", "");
      else section.removeAttribute("open");
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", restoreSections);
  } else {
    restoreSections();
  }

  window.OrbitalMenu = { closeAll: function () { closeAll(null); } };
})();
