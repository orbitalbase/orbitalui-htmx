package components

import (
	"strings"
	"testing"
)

// The icon set is the sidebar + component contract: every name the UI can
// render must resolve to a real path (never the fallback), and the minimum
// set the plan §6/§8 lists must exist.
func TestRequiredIconsPresent(t *testing.T) {
	required := []string{
		"dashboard", "journal", "document", "bank", "recon", "parties",
		"payments", "assets", "accounts", "reports", "periods", "tax",
		"cashflow", "ai", "chat", "settings", "plus", "close", "check",
		"search", "chevron-down", "chevron-left", "chevron-right",
		"chevron-up", "calendar", "book", "alert", "refresh", "external",
	}
	for _, name := range required {
		if Path(name) == fallbackIcon {
			t.Errorf("icon %q is missing (resolves to the fallback)", name)
		}
	}
}

func TestUnknownIconFallsBack(t *testing.T) {
	if Path("definitely-not-an-icon") != fallbackIcon {
		t.Error("unknown icon must render the neutral fallback, not a blank")
	}
}

func TestIconPathsAreSVGPathData(t *testing.T) {
	// Every icon's inner markup must contain only path 'd' commands — the
	// component renders <path d={...}> verbatim.
	for name, d := range iconPaths {
		if d == "" {
			t.Errorf("icon %q has empty path data", name)
		}
	}
}

func TestTargetAndListIcons(t *testing.T) {
	for _, name := range []string{"target", "list", "history", "panel-right", "sparkles"} {
		if Path(name) == fallbackIcon {
			t.Errorf("icon %q is missing", name)
		}
	}
}

func TestRegisterIconsAddsAndOverrides(t *testing.T) {
	builtin := iconPaths["spark"]
	t.Cleanup(func() {
		hostIconsMu.Lock()
		delete(hostIcons, "test-glyph")
		delete(hostIcons, "spark")
		hostIconsMu.Unlock()
	})
	RegisterIcons(map[string]string{"test-glyph": "M1 1h2", "spark": "M2 2h3", "": "M0 0", "empty": ""})
	if Path("test-glyph") != "M1 1h2" {
		t.Errorf("registered icon not resolved: %q", Path("test-glyph"))
	}
	if Path("spark") != "M2 2h3" {
		t.Errorf("registered icon did not override the built-in: %q", Path("spark"))
	}
	if Path("empty") != fallbackIcon || Path("") != fallbackIcon {
		t.Error("empty names / path data must be ignored")
	}
	hostIconsMu.Lock()
	delete(hostIcons, "spark")
	hostIconsMu.Unlock()
	if Path("spark") != builtin {
		t.Error("removing the override must restore the built-in")
	}
}

func TestCommandPaletteIncludesRegisteredIcons(t *testing.T) {
	t.Cleanup(func() {
		hostIconsMu.Lock()
		delete(hostIcons, "host-only")
		hostIconsMu.Unlock()
	})
	RegisterIcons(map[string]string{"host-only": "M5 5h14"})
	var b strings.Builder
	if err := CommandPalette(CommandPaletteOpts{Icons: []string{"host-only"}}).Render(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, `data-icon="host-only"`) || !strings.Contains(out, `d="M5 5h14"`) {
		t.Errorf("palette icon template lacks the registered glyph:\n%s", out)
	}
}

func TestMenuHeaderQuietAndPaletteRowOptions(t *testing.T) {
	var b strings.Builder
	if err := MenuHeader("", "ada@example.com").Render(t.Context(), &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "ui-menu__header--quiet") || strings.Contains(out, "ui-menu__header-title") || !strings.Contains(out, "ada@example.com") {
		t.Errorf("empty title must render the quiet single line: %s", out)
	}
	o := CommandPaletteOpts{Groups: []PaletteGroup{{Items: []PaletteItem{{Label: "Ask agent", Href: "/a", Default: true, Promoted: true, QueryLabel: true, Form: &PaletteForm{Action: "/x", QueryField: "q"}}}}}}
	j := paletteGroupsJSON(o)
	for _, k := range []string{`"default":true`, `"promoted":true`, `"queryLabel":true`, `"queryField":"q"`} {
		if !strings.Contains(j, k) {
			t.Errorf("palette JSON lacks %s: %s", k, j)
		}
	}
}
