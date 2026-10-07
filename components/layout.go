// Layout primitives: the page frame that fixes a page's reading width and
// vertical rhythm, and the form grid and action row that every form shares
// (layout.templ). They own width, columns and spacing only; the host keeps
// ownership of content, endpoints and permissions.
package components

import "strconv"

// PageWidth picks the frame's maximum inline size. Choose by content, not by
// page: a sign-in or settings form reads best narrow, a dashboard standard, a
// dense register wide, and a full-bleed table or split view full.
type PageWidth string

const (
	PageWidthStandard PageWidth = ""
	PageWidthNarrow   PageWidth = "narrow"
	PageWidthWide     PageWidth = "wide"
	PageWidthFull     PageWidth = "full"
)

// PageFrameOpts configures PageFrame.
//
//	Width — the frame's measure; the zero value is the standard measure.
//	ID/Class — host hooks on the frame element.
type PageFrameOpts struct {
	Width PageWidth
	ID    string
	Class string
}

func pageFrameClass(o PageFrameOpts) string {
	width := ""
	switch o.Width {
	case PageWidthNarrow, PageWidthWide, PageWidthFull:
		width = "ui-page--" + string(o.Width)
	}
	return joinClass("ui-page", width, o.Class)
}

// FormGridOpts configures FormGrid.
//
//	Columns — 1, 2 or 3 field columns (0 means 2). The grid collapses to one
//	    column on a phone. A LabelledField with FieldOpts.Wide spans the row.
//	Class — a host hook on the grid element.
type FormGridOpts struct {
	Columns int
	Class   string
}

func formGridClass(o FormGridOpts) string {
	cols := o.Columns
	if cols < 1 || cols > 3 {
		cols = 2
	}
	return joinClass("ui-form-grid", "ui-form-grid--"+strconv.Itoa(cols), o.Class)
}

// FormActionsOpts configures FormActions.
//
//	End — align the actions to the inline end (dialogs and save bars).
//	Divided — draw a rule above the row to close a long form.
//	Class — a host hook on the row element.
type FormActionsOpts struct {
	End     bool
	Divided bool
	Class   string
}

func formActionsClass(o FormActionsOpts) string {
	return joinClass("ui-form-actions",
		modifier(o.End, "ui-form-actions--end"),
		modifier(o.Divided, "ui-form-actions--divided"),
		o.Class)
}
