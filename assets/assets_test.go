package assets

import (
	"io/fs"
	"strings"
	"testing"
)

// The runtime and every stylesheet ship in Files, so a host serving the
// embedded assets gets the whole contract.
func TestFilesShipRuntimeAndStyles(t *testing.T) {
	for _, name := range []string{"tokens.css", "theme.css", "controls.css", "inspector.css", "inspector.js", "menu.js", "orbital-ui.js"} {
		if _, err := fs.Stat(Files, name); err != nil {
			t.Errorf("%s is not embedded: %v", name, err)
		}
	}
}

// The runtime implements every helper the component markup calls, and stays
// inert when a host already provides its own LedgerUI runtime.
func TestRuntimeCoversMarkupContract(t *testing.T) {
	raw, err := fs.ReadFile(Files, "orbital-ui.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(raw)
	for _, needle := range []string{
		"UI.dialog = function", "UI.closeDialog = function", "UI.openDialog = function",
		"UI.drawer = function", "UI.closeDrawer = function", "UI.openDrawer = function",
		"UI.picker = function", "window.LedgerUI = UI",
		`"[data-open-dialog]"`, `"[data-close-dialog]"`, `"[data-open-drawer]"`, `"[data-close-drawer]"`, `"[data-drawer-fullscreen]"`,
		`"closeDialog"`, `"drawerClose"`,
		`typeof existing.openDialog === "function"`,
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("orbital-ui.js is missing %s", needle)
		}
	}
}
