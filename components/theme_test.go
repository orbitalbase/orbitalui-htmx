package components

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every tone class a component can emit has a rule in the shipped theme, so
// a tone never falls back to the untinted base silently.
func TestThemeStylesEmittableToneClasses(t *testing.T) {
	raw, err := os.ReadFile("../assets/theme.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	for _, selector := range []string{
		".ui-callout--info",
		".ui-progress--info .ui-progress__fill",
		".ui-progress--ai .ui-progress__fill",
		".ui-nav__badge--success",
		".ui-nav__badge--ai",
		".ui-record-header__icon--ai",
	} {
		if !strings.Contains(css, selector) {
			t.Errorf("theme.css has no rule for %s", selector)
		}
	}
}

// A host can rebrand focus feedback by overriding its brand scale; the ring
// must not retain the default indigo as a hardcoded color.
func TestHostBrandOverrideControlsFocusRing(t *testing.T) {
	raw, err := os.ReadFile("../assets/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	tokens := string(raw)
	if !strings.Contains(tokens, "--oui-ring-brand: 0 0 0 4px var(--oui-brand-100);") {
		t.Error("focus ring has no brand-token fallback")
	}
	if !strings.Contains(tokens, "--oui-ring-brand: 0 0 0 4px color-mix(in srgb, var(--oui-brand-600) 18%, transparent);") {
		t.Error("focus ring does not derive its enhanced color from the host brand")
	}
	if !strings.Contains(tokens, "@supports (color: color-mix(in srgb, black, transparent))") {
		t.Error("enhanced ring must be guarded to preserve the fallback in older browsers")
	}
	if !strings.Contains(tokens, "--oui-ring-brand:") {
		t.Fatal("tokens.css does not define --oui-ring-brand")
	}
	if strings.Contains(tokens, "--oui-ring-brand: 0 0 0 4px rgb(68 76 231") {
		t.Error("focus ring retains the default indigo instead of host tokens")
	}

	cssRaw, err := os.ReadFile("../assets/theme.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(cssRaw)
	for _, selector := range []string{".ui-button:focus-visible", ".ui-icon-button:focus-visible", ".ui-menu__trigger:focus-visible"} {
		start := strings.Index(css, selector)
		if start < 0 {
			t.Errorf("theme.css is missing %s", selector)
			continue
		}
		end := strings.Index(css[start:], "}")
		if end < 0 || !strings.Contains(css[start:start+end], "var(--oui-ring-brand)") {
			t.Errorf("%s does not use the shared brand focus ring", selector)
		}
	}
}

func TestDialogThemeOwnsResponsiveNativeDialogFrame(t *testing.T) {
	raw, err := os.ReadFile("../assets/theme.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	assertRuleContains := func(selector string, declarations ...string) {
		t.Helper()
		start := strings.Index(css, selector+" {")
		if start < 0 {
			t.Errorf("theme.css is missing %s", selector)
			return
		}
		end := strings.Index(css[start:], "}")
		if end < 0 {
			t.Errorf("theme.css has unterminated %s rule", selector)
			return
		}
		rule := css[start : start+end]
		for _, declaration := range declarations {
			if !strings.Contains(rule, declaration) {
				t.Errorf("%s is missing %q", selector, declaration)
			}
		}
	}
	assertRuleContains(".ui-dialog", "position: fixed", "inset: 0", "margin: auto", "width: min(var(--oui-dialog-width", "max-height: var(--oui-dialog-max-height", "overflow-y: auto", "var(--oui-radius-2xl)")
	assertRuleContains(".ui-dialog__panel", "width: 100%", "min-width: 0")
	assertRuleContains(".ui-dialog__close", "place-items: center", "var(--oui-control-height-sm)")
}

// Every class a component emits has a rule in the shipped stylesheets, so a
// host that imports tokens.css and theme.css gets a finished component and
// never has to style library markup itself. Classes styled only through a
// parent selector are listed with the selector that owns them.
func TestThemeStylesEveryEmittedClass(t *testing.T) {
	styledVia := map[string]string{
		"tab":                     ".tab-bar a",
		"ui-row--link":            "a.ui-row",
		"drawer-panel__tool-icon": ".drawer-panel__tool-icon--expand/--contract",
		"oui-assistant-message":   "layout utilities on the element; body is .oui-assistant-message__body",
	}
	library := regexp.MustCompile(`^(ui-|oui-|field|form-|tab|kpi-|pager|table-|ledger-table|search-field|segmented|disclosure|selection-toolbar|beam|amount|mono|drawer|entity-picker|async-region|dependency-panel|record-state-summary|confirm-|th-sort)`)

	emitted := map[string]string{}
	collect := func(file, list string) {
		for _, token := range strings.Fields(list) {
			token = strings.Trim(token, "'")
			if library.MatchString(token) && !strings.HasSuffix(token, "-") {
				emitted[token] = file
			}
		}
	}
	templs, _ := filepath.Glob("*.templ")
	classAttr := regexp.MustCompile(`class="([^"{}]+)"`)
	for _, file := range templs {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range classAttr.FindAllStringSubmatch(string(raw), -1) {
			collect(file, m[1])
		}
	}
	// Class helpers build names from string literals: "ui-card ui-card--flush".
	sources, _ := filepath.Glob("*.go")
	literal := regexp.MustCompile(`"([a-z][a-z0-9_ -]*)"`)
	for _, file := range sources {
		if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "_templ.go") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range literal.FindAllStringSubmatch(string(raw), -1) {
			if strings.ContainsAny(m[1], "_-") {
				collect(file, m[1])
			}
		}
	}

	var css strings.Builder
	sheets, _ := filepath.Glob("../assets/*.css")
	for _, sheet := range sheets {
		raw, err := os.ReadFile(sheet)
		if err != nil {
			t.Fatal(err)
		}
		css.Write(raw)
	}
	selectors := map[string]bool{}
	for _, m := range regexp.MustCompile(`\.([a-z][a-z0-9_-]*)`).FindAllStringSubmatch(css.String(), -1) {
		selectors[m[1]] = true
	}
	for class, file := range emitted {
		if selectors[class] {
			continue
		}
		if _, ok := styledVia[class]; ok {
			continue
		}
		t.Errorf("%s emits .%s but no shipped stylesheet styles it", file, class)
	}
}

// The runtime rewrites the drawer's full-screen label after a toggle; it must
// use the same two strings the server renders.
func TestRuntimeFullscreenLabelsMatchServer(t *testing.T) {
	raw, err := os.ReadFile("../assets/orbital-ui.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, full := range []bool{true, false} {
		if !strings.Contains(string(raw), `"`+drawerFullscreenLabel(full)+`"`) {
			t.Errorf("orbital-ui.js does not use the label %q", drawerFullscreenLabel(full))
		}
	}
}
