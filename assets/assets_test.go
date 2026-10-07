package assets

import (
	"io/fs"
	"strings"
	"testing"
)

// The runtime and every stylesheet ship in Files, so a host serving the
// embedded assets gets the whole contract.
func TestFilesShipRuntimeAndStyles(t *testing.T) {
	for _, name := range []string{"tokens.css", "base.css", "motion.css", "theme.css", "controls.css", "inspector.css", "inspector.js", "menu.js", "orbital-ui.js"} {
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
		// CSP-safe markup: factories registered with Alpine, config from data-oui-picker.
		`A.data("ouiDialog"`, `A.data("ouiDrawer"`, `A.data("ouiPicker"`, `"alpine:init"`, `data-oui-picker`,
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("orbital-ui.js is missing %s", needle)
		}
	}
}

// The runtime stays out of a host's way (no click delegation of its own) yet
// still registers the Alpine adapters, and the markup directives it serves are
// bare references only, which is all Alpine's CSP build evaluates.
func TestRuntimeAdaptsToHostRuntime(t *testing.T) {
	raw, err := fs.ReadFile(Files, "orbital-ui.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(raw)
	hosted := strings.Index(js, "registerAlpine(existing);")
	delegation := strings.Index(js, `document.addEventListener("click"`)
	if hosted < 0 || delegation < 0 || hosted > delegation {
		t.Fatal("the host-runtime path must register the adapters and return before the click delegation is installed")
	}
	for _, needle := range []string{"after any host", "alpinejs-csp", "data-oui-picker"} {
		if !strings.Contains(js, needle) {
			t.Errorf("the load-order documentation in orbital-ui.js is missing %q", needle)
		}
	}
}
