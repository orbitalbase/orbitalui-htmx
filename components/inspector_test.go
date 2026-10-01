package components

import (
	"strings"
	"testing"
)

func TestInspectorHasNativeLinksAndStableClosedMount(t *testing.T) {
	closed := renderC(t, Inspector(InspectorOpts{ID: "record-inspector"}, Text("Private body")))
	want(t, closed, `id="record-inspector"`)
	want(t, closed, "hidden")
	if strings.Contains(closed, "Private body") {
		t.Fatal("closed mount renders record content")
	}

	open := renderC(t, Inspector(InspectorOpts{
		ID: "record-inspector", Title: "Record", Open: true, Resizable: true,
		CloseHref: "/records?q=active", ExpandHref: "/records?q=active&selected=1&panel=full",
	}, Text("Body")))
	want(t, open, `aria-labelledby="record-inspector-title"`)
	want(t, open, `role="separator"`)
	want(t, open, `aria-orientation="vertical"`)
	want(t, open, `href="/records?q=active"`)
	want(t, open, `href="/records?q=active&amp;selected=1&amp;panel=full"`)
	if strings.Contains(open, "x-data") || strings.Contains(open, "style=") {
		t.Fatal("inspector requires inline Alpine/style evaluation")
	}
}
