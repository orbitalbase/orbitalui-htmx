package main

import (
	"net/url"

	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

func init() {
	opts := []c.SelectOption{{Value: "a", Label: "Tax exclusive"}, {Value: "b", Label: "Tax inclusive"}}
	lf := func(label, id string, o c.FieldOpts, control templ.Component) templ.Component {
		return c.LabelledField(label, id, o, control)
	}
	register(section{ID: "fields", Title: "Fields", Note: "Labelled controls in every state (hover and focus need a pointer or keyboard).", Body: seq(
		groupGrid("Text input",
			lf("Default", "f1", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f1", Placeholder: "Placeholder"})),
			lf("Filled + help", "f2", c.FieldOpts{Help: "Shown on your public profile."}, c.FieldInput(c.FieldInputOpts{ID: "f2", Value: "Ada Lovelace"})),
			lf("Required", "f3", c.FieldOpts{Required: true}, c.FieldInput(c.FieldInputOpts{ID: "f3", Required: true})),
			lf("Invalid", "f4", c.FieldOpts{Error: "That date is in a closed period."}, c.FieldInput(c.FieldInputOpts{ID: "f4", Value: "2026-09-03", Invalid: true, AriaDescribedBy: c.FieldErrorID("f4")})),
			lf("Disabled", "f5", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f5", Value: "Read only", Disabled: true})),
			lf("Suggested", "f6", c.FieldOpts{Suggested: true, Help: "Filled from your CV."}, c.FieldInput(c.FieldInputOpts{ID: "f6", Value: "Staff engineer"})),
			lf("Leading icon", "f7", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f7", Icon: "search", Placeholder: "Search people"})),
			lf("Prefix", "f8", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f8", Type: "number", Prefix: "$", Value: "120000"})),
			lf("Suffix", "f9", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f9", Type: "number", Suffix: "%", Value: "12.5"})),
			lf("Prefix + suffix, invalid", "f10", c.FieldOpts{Error: "Out of range."}, c.FieldInput(c.FieldInputOpts{ID: "f10", Type: "number", Prefix: "$", Suffix: "USD", Value: "-1", Invalid: true})),
			lf("Icon, disabled", "f11", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f11", Icon: "search", Disabled: true, Placeholder: "Unavailable"})),
			lf("Number (mono, no spinner)", "f12", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f12", Type: "number", Value: "1042.50", Step: "0.01"})),
			lf("A very long label that wraps onto a second line when the column is narrow", "f13", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "f13", Value: "A value that is long enough to overflow the control box on a narrow phone screen"})),
		),
		groupGrid("Select",
			lf("Default", "s1", c.FieldOpts{}, c.FieldSelect(c.FieldSelectOpts{ID: "s1", Options: opts, Placeholder: "Choose…"})),
			lf("Selected", "s2", c.FieldOpts{}, c.FieldSelect(c.FieldSelectOpts{ID: "s2", Options: opts, Selected: "b"})),
			lf("Invalid", "s3", c.FieldOpts{Error: "Pick one."}, c.FieldSelect(c.FieldSelectOpts{ID: "s3", Options: opts, Placeholder: "Choose…", Invalid: true})),
			lf("Disabled", "s4", c.FieldOpts{}, c.FieldSelect(c.FieldSelectOpts{ID: "s4", Options: opts, Selected: "a", Disabled: true})),
			lf("Suggested", "s5", c.FieldOpts{Suggested: true}, c.FieldSelect(c.FieldSelectOpts{ID: "s5", Options: opts, Selected: "b"})),
		),
		groupGrid("Field options",
			lf("Suggested with icon (one ring)", "o1", c.FieldOpts{Suggested: true}, c.FieldInput(c.FieldInputOpts{ID: "o1", Icon: "search", Value: "Ada"})),
			lf("Numeric text input", "o2", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "o2", Numeric: true, Value: "1,204.50"})),
			lf("Help + error wired by FieldDescribedBy", "o3", c.FieldOpts{Help: "Pick the closing account.", Error: "Required.", ErrorAlert: true}, c.FieldSelect(c.FieldSelectOpts{ID: "o3", Options: opts, Placeholder: "Choose…", PlaceholderDisabled: true, Required: true, Invalid: true, AriaDescribedBy: c.FieldDescribedBy("o3", c.FieldOpts{Help: "x", Error: "x"})})),
			cell("inset ring (--oui-control-ring-inset: inset)", wrap(`<div style="--oui-control-ring-inset: inset; overflow: hidden">`, `</div>`, c.FieldInput(c.FieldInputOpts{ID: "o4", Value: "Ring drawn inside"}))),
		),
		groupGrid("Textarea",
			lf("Default", "t1", c.FieldOpts{}, c.FieldTextarea(c.FieldTextareaOpts{ID: "t1", Placeholder: "Write something"})),
			lf("Autogrow (grows with content)", "t2", c.FieldOpts{Help: "field-sizing: content, JS fallback elsewhere."}, c.FieldTextarea(c.FieldTextareaOpts{ID: "t2", Autogrow: true, Value: "Line one\nLine two\nLine three\nLine four\nLine five\nLine six\nLine seven"})),
			lf("Invalid", "t3", c.FieldOpts{Error: "Too long."}, c.FieldTextarea(c.FieldTextareaOpts{ID: "t3", Value: "Over the limit", Invalid: true})),
			lf("Disabled", "t4", c.FieldOpts{}, c.FieldTextarea(c.FieldTextareaOpts{ID: "t4", Value: "Locked", Disabled: true})),
		),
		groupGrid("Search field",
			cell("default", c.SearchField(c.SearchFieldOpts{Placeholder: "Search…"})),
			cell("with value", c.SearchField(c.SearchFieldOpts{Value: "board resolution", ID: "sf2"})),
		),
		group("Suggested tag", plain(c.SuggestedTag("")), plain(c.SuggestedTag("From CV")), plain(c.SuggestedTagTitled("From CV", "Suggested by your agent"))),
	)})

	reg := func() url.Values { return url.Values{} }
	register(section{ID: "chips-segments", Title: "Chips, segments and tabs", Body: seq(
		groupStack("ChipGroup, single",
			c.ChipGroup([]c.ChipOption{{Value: "r", Label: "Remote", Selected: true}, {Value: "h", Label: "Hybrid"}, {Value: "o", Label: "On-site"}, {Value: "x", Label: "Unavailable", Disabled: true}}, c.ChipGroupOpts{Name: "mode", Legend: "Work mode"})),
		groupStack("ChipGroup, multiple with icons",
			c.ChipGroup([]c.ChipOption{{Value: "a", Label: "Design", Icon: "plus", Selected: true}, {Value: "b", Label: "Engineering", Selected: true}, {Value: "c", Label: "Product"}, {Value: "d", Label: "Sales"}}, c.ChipGroupOpts{Name: "teams", Multiple: true, Legend: "Teams"})),
		groupStack("ChipGroup, suggested",
			c.ChipGroup([]c.ChipOption{{Value: "a", Label: "Full-time", Suggested: true}, {Value: "b", Label: "Contract", Suggested: true, Selected: true}, {Value: "c", Label: "Part-time"}}, c.ChipGroupOpts{Name: "kind", Legend: "Contract type"})),
		groupStack("ChipGroup, tick first and spark last, with help and error",
			c.ChipGroup([]c.ChipOption{{Value: "a", Label: "Full-time", Suggested: true, Selected: true}, {Value: "b", Label: "Contract", Icon: "plus"}, {Value: "c", Label: "Part-time"}}, c.ChipGroupOpts{Name: "kind2", Legend: "Contract type", Required: true, Help: "Pick the closest match.", Error: "Choose one to continue."})),
		groupStack("ChipGroup, SuggestedTitle and suggestion tokens",
			c.ChipGroup([]c.ChipOption{{Value: "a", Label: "Full-time", Suggested: true, Selected: true}, {Value: "b", Label: "Contract", Suggested: true}}, c.ChipGroupOpts{Name: "sug-title", Legend: "Type", SuggestedTitle: "Suggested by your agent"})),
		groupStack("ChipGroup, disabled",
			c.ChipGroup([]c.ChipOption{{Value: "a", Label: "One", Selected: true}, {Value: "b", Label: "Two"}}, c.ChipGroupOpts{Name: "dis", Legend: "Disabled group", Disabled: true})),
		group("SegmentedChoice",
			cell("2 options", c.SegmentedChoice([]c.SegChoice{{Value: "m", Label: "Monthly"}, {Value: "y", Label: "Yearly"}}, "m", c.SegmentedChoiceOpts{Name: "bill", AriaLabel: "Billing"})),
			cell("3 options", c.SegmentedChoice([]c.SegChoice{{Value: "a", Label: "All"}, {Value: "b", Label: "Open"}, {Value: "c", Label: "Closed"}}, "b", c.SegmentedChoiceOpts{Name: "st", AriaLabel: "Status"})),
			cell("small, icon only", c.SegmentedChoice([]c.SegChoice{{Value: "l", Label: "Light", Icon: "check", IconOnly: true}, {Value: "s", Label: "System", Icon: "settings", IconOnly: true}, {Value: "d", Label: "Dark", Icon: "close", IconOnly: true}}, "s", c.SegmentedChoiceOpts{Name: "th", AriaLabel: "Theme", Size: "sm"})),
			cell("with a disabled option", c.SegmentedChoice([]c.SegChoice{{Value: "a", Label: "One"}, {Value: "b", Label: "Two", Disabled: true}, {Value: "c", Label: "Three"}}, "a", c.SegmentedChoiceOpts{Name: "dsg", AriaLabel: "Disabled option"})),
		),
		group("SegmentedNav (href-driven, sliding thumb)",
			cell("3 segments, count", c.SegmentedNav([]c.SegItem{{Label: "Overview", Href: "#o"}, {Label: "Roles", Href: "#r", Active: true, Count: "4"}, {Label: "Notes", Href: "#n"}}, c.SegNavOpts{AriaLabel: "Company"})),
			cell("icons", c.SegmentedNav([]c.SegItem{{Label: "List", Href: "#l", Icon: "list", Active: true}, {Label: "Board", Href: "#b", Icon: "dashboard"}}, c.SegNavOpts{AriaLabel: "Layout"})),
		),
		group("Segmented (links)",
			cell("active + disabled", c.SegmentedControl([]c.SegOption{{Value: "ar", Label: "AR"}, {Value: "ap", Label: "AP"}, {Value: "x", Label: "Other", Disabled: true}}, "ar", c.SegmentedOpts{Param: "side", BaseHref: "/p", BaseQuery: reg()})),
		),
		groupStack("TabBar with counts",
			c.TabBar([]c.Tab{{Value: "", Label: "All", Count: 128}, {Value: "draft", Label: "Draft", Count: 12}, {Value: "posted", Label: "Posted", Count: 1043}, {Value: "void", Label: "Void", Count: 0}}, "draft", c.TabBarOpts{Param: "status", BaseHref: "/j", BaseQuery: reg(), ShowCounts: true})),
		groupStack("TabBar without counts",
			c.TabBar([]c.Tab{{Value: "a", Label: "Overview"}, {Value: "b", Label: "Activity"}, {Value: "c", Label: "Settings"}}, "a", c.TabBarOpts{Param: "tab", BaseHref: "/j", BaseQuery: reg()})),
		groupStack("TabNav (href-driven) with string counts",
			c.TabNav([]c.TabItem{{Label: "Overview", Href: "#o", Active: true, Count: "12"}, {Label: "Roles", Href: "#r", Count: "99+"}, {Label: "Notes", Href: "#n", Icon: "document"}}, c.TabNavOpts{AriaLabel: "Sections"})),
		groupStack("FilterChips",
			c.FilterChips([]c.ChipLink{{Label: "All", Href: "/s"}, {Label: "Filings", Href: "/s?c=f", Value: "f", Count: 4}, {Label: "Notices", Href: "/s?c=n", Value: "n", Count: 11}}, "f", c.FilterChipsOpts{Label: "Filter by category"})),
	)})
}
