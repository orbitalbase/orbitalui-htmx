// ThemeSwitch (themeswitch.templ) and the pre-paint script assets/theme.js.
//
// Theme model. The preference is "light", "dark" or "system" (default),
// stored in localStorage under StorageKey (default "oui-theme"). theme.js,
// loaded synchronously in <head> before the stylesheets, resolves it to
// data-theme="light"|"dark" on <html> (and color-scheme) before first paint,
// and follows prefers-color-scheme and other tabs while the preference is
// "system". The switch is a SegmentedChoice (sun / monitor / moon); the
// runtime in orbital-ui.js syncs it with window.OrbitalTheme.
//
//	<script src="/static/theme.js"></script>                           <!-- default key "oui-theme" -->
//	<script src="/static/theme.js" data-storage-key="myapp-theme"></script>
//
// The key can also be set on <html data-oui-theme-key="…">, and the first-visit
// preference with data-default="dark" / data-oui-theme-default="dark".
//
// A light-only (or dark-only) app does not load theme.js and does not render
// ThemeSwitch: it pins the theme statically with <html data-theme="light">.
// tokens.css follows prefers-color-scheme only while data-theme is unset, so a
// pinned page never flips with the OS.
package components

// ThemeSwitchOpts configures ThemeSwitch.
//
//	StorageKey — must match theme.js's key; rendered as data-storage-key so
//	    the runtime can still apply a choice if theme.js was not loaded.
//	    Default "oui-theme".
//	AriaLabel — the group's name (default "Color theme").
//	Size — "" or "sm". Class — extra classes.
type ThemeSwitchOpts struct {
	StorageKey string
	AriaLabel  string
	Size       string
	Class      string
}

func themeStorageKey(o ThemeSwitchOpts) string {
	if o.StorageKey == "" {
		return "oui-theme"
	}
	return o.StorageKey
}

func themeAriaLabel(o ThemeSwitchOpts) string {
	if o.AriaLabel == "" {
		return "Color theme"
	}
	return o.AriaLabel
}
