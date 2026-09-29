package components

import (
	"fmt"
	"net/url"
	"strings"
)

// RecordStateSummaryOpts expresses independent accounting facts. Empty strings
// are omitted, so a presentation summary never invents a lifecycle state.
type RecordStateSummaryOpts struct {
	Posting, Review, Reconciliation, Period, Provenance string
}

type AsyncRegionOpts struct {
	ID, Label, State, Message, RetryHref string
}

type Dependency struct{ Label, Detail, Href string }

type SelectionToolbarOpts struct {
	Selected, Eligible, Excluded int
	AllMatching                  bool
	SnapshotID                   string
	ExcludedReason               string
}

func asyncBusy(state string) string {
	if state == "loading" {
		return "true"
	}
	return "false"
}
func selectionPopulation(opts SelectionToolbarOpts) string {
	if opts.AllMatching && strings.TrimSpace(opts.SnapshotID) != "" {
		return "All matching records"
	}
	return fmt.Sprintf("%d on this page", opts.Selected)
}

func internalInteractionHref(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\r\n\t") {
		return ""
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.IsAbs() || u.Host != "" || strings.Contains(u.Path, "\\") {
		return ""
	}
	return raw
}
