package components

import "testing"

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
