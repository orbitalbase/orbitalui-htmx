package components

import (
	"os"
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
