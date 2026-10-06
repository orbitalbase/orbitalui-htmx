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
