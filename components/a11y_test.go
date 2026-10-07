package components

import (
	"strings"
	"testing"
)

func TestBreadcrumbsOnlyLastCrumbIsCurrent(t *testing.T) {
	out := renderC(t, Breadcrumbs([]Crumb{
		{Label: "Home"},
		{Label: "Companies", Href: "/companies"},
		{Label: "Acme"},
	}))
	if n := strings.Count(out, `aria-current="page"`); n != 1 {
		t.Fatalf("aria-current on %d crumbs, want 1: %s", n, out)
	}
	want(t, out, `<span>Home</span>`)
	want(t, out, `<a href="/companies">Companies</a>`)
	want(t, out, `<span aria-current="page">Acme</span>`)
}

func TestProgressBarAlwaysHasAccessibleName(t *testing.T) {
	cases := []struct {
		opts ProgressOpts
		want string
	}{
		{ProgressOpts{}, `aria-label="Progress"`},
		{ProgressOpts{ShowValue: true}, `aria-label="Progress"`},
		{ProgressOpts{AriaLabel: "Upload progress"}, `aria-label="Upload progress"`},
		{ProgressOpts{Label: "Profile", AriaLabel: "ignored"}, `aria-label="Profile"`},
	}
	for _, tc := range cases {
		out := renderC(t, ProgressBar(40, tc.opts))
		want(t, out, tc.want)
		if n := strings.Count(out, "aria-label="); n != 1 {
			t.Errorf("%+v: %d aria-label attributes", tc.opts, n)
		}
	}
}
