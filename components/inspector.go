package components

// InspectorOpts describes a record panel. The application owns authorization,
// loaded content and URLs; the library owns chrome and progressive behavior.
// StateKeys lists query parameters that may change without replacing the list.
// CloseHref and ExpandHref must preserve the host page's filters and paging.
type InspectorOpts struct {
	ID, Title, Subtitle, CloseHref, BackHref, ExpandHref string
	Open, Expanded, Resizable                            bool
	StorageKey, StateKeys                                string
	// IgnoreEmptyQuery treats empty host filter values like missing values when
	// deciding whether navigation can retain the list. Opt in only when the
	// application's GET handlers assign the same meaning to both.
	IgnoreEmptyQuery bool
}

func inspectorStateKeys(o InspectorOpts) string {
	if o.StateKeys != "" {
		return o.StateKeys
	}
	return "selected,panel"
}

func inspectorExpandLabel(expanded bool) string {
	if expanded {
		return "Restore panel beside the list"
	}
	return "Expand panel to full screen"
}
