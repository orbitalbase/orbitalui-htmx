package components

import (
	"strings"
	"testing"
)

func TestHTMXPostFormKeepsNativePostFallback(t *testing.T) {
	out := renderC(t, HTMXPostForm("save", "/items", "#result", "innerHTML", "#spinner", "form", Text("Save")))
	for _, want := range []string{
		`method="post"`,
		`action="/items"`,
		`hx-post="/items"`,
		`hx-target="#result"`,
		`hx-swap="innerHTML"`,
		`hx-indicator="#spinner"`,
		`Save`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered form missing %q\n---\n%s", want, out)
		}
	}
}

func TestHTMXPostFormSanitizesRequestURLAndOmitsEmptyIndicator(t *testing.T) {
	out := renderC(t, HTMXPostForm("save", "javascript:alert(1)", "#result", "", "", "", Text("Save")))
	if strings.Contains(out, `hx-indicator=""`) {
		t.Fatalf("empty indicator should be omitted: %s", out)
	}
	if !strings.Contains(out, `hx-post="about:invalid#TemplFailedSanitizationURL"`) {
		t.Fatalf("unsafe hx-post URL should be sanitized: %s", out)
	}
	if !strings.Contains(out, `action="about:invalid#TemplFailedSanitizationURL"`) {
		t.Fatalf("unsafe form action should be sanitized: %s", out)
	}
}
