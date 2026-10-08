/* Orbital UI pre-paint theme script.
 *
 * Load it synchronously in <head>, before the stylesheets, so the right theme
 * is on <html> before first paint (no flash of the wrong theme). It is a
 * plain external file: no inline script, so it passes `script-src 'self'`.
 *
 *   <script src="/static/theme.js"></script>
 *   <script src="/static/theme.js" data-storage-key="myapp-theme" data-default="system"></script>
 *
 * Preference: "light" | "dark" | "system", kept in localStorage under the
 * storage key (script attribute data-storage-key, else <html
 * data-oui-theme-key>, else "oui-theme"). The first-visit preference is
 * data-default / <html data-oui-theme-default> (default "system").
 *
 * Effect: <html data-theme="light|dark" data-theme-preference="…"> and
 * style.colorScheme. While the preference is "system" it follows
 * prefers-color-scheme live; a change made in another tab is picked up too.
 *
 * API: window.OrbitalTheme = {getPreference, getTheme, setPreference,
 * storageKey}. setPreference also fires `oui:theme` on document
 * (detail: {preference, theme}). ThemeSwitch (orbital-ui.js) drives it.
 *
 * Browser UI colour: a <meta name="theme-color" data-light="#fff"
 * data-dark="#111"> gets its content set to the active theme's value whenever
 * the theme is applied (first paint, changes, other tabs). Without the data
 * attributes the meta is left untouched.
 *
 * A light-only app skips this file and pins <html data-theme="light">.
 */
(function () {
    "use strict";
    var root = document.documentElement;
    var script = document.currentScript;
    function setting(attr, htmlAttr, fallback) {
        return (script && script.getAttribute(attr)) || root.getAttribute(htmlAttr) || fallback;
    }
    var key = setting("data-storage-key", "data-oui-theme-key", "oui-theme");
    var fallback = normalize(setting("data-default", "data-oui-theme-default", "system"));

    function normalize(value) {
        return value === "light" || value === "dark" ? value : "system";
    }

    function read() {
        try {
            var stored = window.localStorage.getItem(key);
            return stored ? normalize(stored) : fallback;
        } catch (err) {
            return fallback; // storage blocked: the default still works
        }
    }

    var media = null;
    try { media = window.matchMedia("(prefers-color-scheme: dark)"); } catch (err) { /* light default */ }
    var preference = read();

    function resolve() {
        if (preference === "system") return media && media.matches ? "dark" : "light";
        return preference;
    }

    function paintChrome(theme) {
        var metas = document.querySelectorAll('meta[name="theme-color"]');
        for (var i = 0; i < metas.length; i++) {
            var color = metas[i].getAttribute("data-" + theme);
            if (color) metas[i].setAttribute("content", color);
        }
    }

    function apply() {
        var theme = resolve();
        root.setAttribute("data-theme", theme);
        root.setAttribute("data-theme-preference", preference);
        root.style.colorScheme = theme;
        paintChrome(theme);
        return theme;
    }

    function onSystem() { if (preference === "system") apply(); }
    if (media) {
        if (media.addEventListener) media.addEventListener("change", onSystem);
        else if (media.addListener) media.addListener(onSystem);
    }
    window.addEventListener("storage", function (e) {
        if (e.key !== key) return;
        preference = read();
        var theme = apply();
        document.dispatchEvent(new CustomEvent("oui:theme", {detail: {preference: preference, theme: theme}}));
    });

    window.OrbitalTheme = {
        storageKey: key,
        getPreference: function () { return preference; },
        getTheme: resolve,
        setPreference: function (value) {
            preference = normalize(value);
            try { window.localStorage.setItem(key, preference); } catch (err) { /* applies for this visit only */ }
            var theme = apply();
            document.dispatchEvent(new CustomEvent("oui:theme", {detail: {preference: preference, theme: theme}}));
            return preference;
        },
    };
    apply();
    // A theme-color meta placed after this script is not parsed yet: paint it
    // once the document is.
    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", function () { paintChrome(resolve()); });
    }
})();
