package components

import (
	"encoding/json"
	"html"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// The overlay markup must contain only what Alpine's CSP build can evaluate:
// bare property or method references. Any call with arguments, negation,
// ternary, assignment or x-model silently does nothing there.
func TestOverlayMarkupIsCSPSafe(t *testing.T) {
	outs := map[string]string{
		"modal":   renderC(t, ModalShell(ModalOpts{ID: "d", Title: "T"}, Text("x"))),
		"confirm": renderC(t, ConfirmButton("Go", "Go", "Go?", "Sure.", ConfirmOpts{ID: "c", Action: "/x"}, nil)),
		"drawer":  renderC(t, Drawer(DrawerOpts{ID: "w", Title: "T", Fullscreen: true, Tabbed: true}, nil)),
		"picker": renderC(t, EntityPicker(EntityPickerOpts{
			ID: "p", Label: "P", Name: "p", Endpoint: templ.SafeURL("/e"), AllowClear: true, LabelName: "pl",
			Create: &EntityPickerCreateOpts{Endpoint: templ.SafeURL("/e"), Noun: "thing"},
		})),
	}
	directive := regexp.MustCompile(`\b(?:x-data|x-on:[a-z.]+|x-bind:[a-z-]+|x-show|x-text|x-for|x-model|:key)="([^"]*)"`)
	bare := regexp.MustCompile(`^(?:\(?\w+(?:, \w+)?\)? in )?[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*$`)
	for name, out := range outs {
		if strings.Contains(out, "x-model") {
			t.Errorf("%s: x-model does not write in Alpine's CSP build", name)
		}
		for _, m := range directive.FindAllStringSubmatch(out, -1) {
			if v := html.UnescapeString(m[1]); !bare.MatchString(v) {
				t.Errorf("%s: directive value %q is not a bare reference (CSP-unsafe)", name, v)
			}
		}
	}
}

// The picker's config travels as JSON in data-oui-picker, which the runtime
// parses; a Create without an endpoint is dropped.
func TestPickerConfigJSON(t *testing.T) {
	var cfg struct {
		ID, Label, Endpoint, ValueLabel string
		AllowClear                      bool
		Create                          *struct{ Endpoint, Noun string }
	}
	raw := pickerConfigJSON(EntityPickerOpts{
		ID: "p", Label: `Cust "A"`, Endpoint: "/api?x=1&y=2", ValueLabel: "Acme", AllowClear: true,
		Create: &EntityPickerCreateOpts{Endpoint: "/api", Noun: "customer"},
	})
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	if cfg.ID != "p" || cfg.Label != `Cust "A"` || cfg.Endpoint != "/api?x=1&y=2" || cfg.ValueLabel != "Acme" || !cfg.AllowClear {
		t.Errorf("config lost fields: %+v", cfg)
	}
	if cfg.Create == nil || cfg.Create.Noun != "customer" {
		t.Errorf("create config missing: %+v", cfg.Create)
	}
	if strings.Contains(pickerConfigJSON(EntityPickerOpts{ID: "p", Create: &EntityPickerCreateOpts{Noun: "x"}}), "create") {
		t.Error("a Create without an endpoint must be dropped from the config")
	}
	// The attribute value is escaped once by templ, so a quote in a label cannot end it.
	out := renderC(t, EntityPicker(EntityPickerOpts{ID: "p", Label: `A "quoted" label`, Name: "p"}))
	want(t, out, `data-oui-picker="{`)
	if strings.Contains(out, `"quoted" label`) {
		t.Errorf("label quotes must be escaped inside the attribute:\n%s", out)
	}
}

// Drawer placement is CSS: a right slide-over from 40rem up, a bottom sheet
// below, with a clipped (not scrollable) overlay.
func TestDrawerPlacementCSS(t *testing.T) {
	css := readThemeCSS(t)
	for _, needle := range []string{
		"@media (min-width: 40rem)",
		"transform: translateX(102%)",
		"transform: translateY(102%)",
		"overflow: clip",
	} {
		if !strings.Contains(css, needle) {
			t.Errorf("theme.css drawer rules are missing %q", needle)
		}
	}
}

func readThemeCSS(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../assets/theme.css")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
