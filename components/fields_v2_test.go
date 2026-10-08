package components

import (
	"strings"
	"testing"
)

// Icon, Prefix and Suffix wrap the input in the .field-input shell; a plain
// input stays the bare element it always was.
func TestFieldInputShellForIconAndAffixes(t *testing.T) {
	plain := renderC(t, FieldInput(FieldInputOpts{ID: "a", Name: "a"}))
	if strings.Contains(plain, "field-input") {
		t.Errorf("plain input must not render the shell: %s", plain)
	}
	out := renderC(t, FieldInput(FieldInputOpts{ID: "amt", Name: "amt", Type: "number", Icon: "search", Prefix: "$", Suffix: "USD", Invalid: true}))
	want(t, out, `class="field-input has-icon has-prefix has-suffix is-invalid"`)
	want(t, out, `<span class="field-input__affix">$</span>`)
	want(t, out, `<span class="field-input__affix field-input__affix--post">USD</span>`)
	want(t, out, `class="field-control field-control--invalid"`)
	want(t, out, `field-input__icon`)
}

func TestFieldSuggestedAndTag(t *testing.T) {
	tag := renderC(t, SuggestedTag(""))
	want(t, tag, `class="ui-suggested"`)
	want(t, tag, `Suggested`)
	want(t, renderC(t, SuggestedTag("From CV")), `From CV`)

	out := renderC(t, LabelledField("Title", "t", FieldOpts{Suggested: true}, FieldInput(FieldInputOpts{ID: "t"})))
	want(t, out, `class="field is-suggested"`)
	want(t, out, `class="field-head"`)
	want(t, out, `class="ui-suggested"`)
	if strings.Contains(renderC(t, LabelledField("Title", "t", FieldOpts{}, nil)), "field-head") {
		t.Error("unsuggested field must keep the bare label")
	}
	want(t, renderC(t, FieldInput(FieldInputOpts{ID: "t", Suggested: true})), `field-control--suggested`)
}

func TestFieldSelectChevronIsAnElement(t *testing.T) {
	out := renderC(t, FieldSelect(FieldSelectOpts{ID: "s", Options: []SelectOption{{Value: "a", Label: "A"}}, Invalid: true}))
	want(t, out, `<span class="field-select is-invalid"><select class="field-control field-control--invalid" id="s"`)
	want(t, out, `class="field-select__chevron" aria-hidden="true"`)
	if strings.Contains(out, "background-image") {
		t.Error("chevron must not be a background image")
	}
}

func TestFieldTextareaAutogrow(t *testing.T) {
	want(t, renderC(t, FieldTextarea(FieldTextareaOpts{ID: "n", Autogrow: true})), `data-autogrow`)
	if strings.Contains(renderC(t, FieldTextarea(FieldTextareaOpts{ID: "n"})), "data-autogrow") {
		t.Error("autogrow must be opt-in")
	}
}

func TestSearchFieldInputIsAFullControl(t *testing.T) {
	want(t, renderC(t, SearchField(SearchFieldOpts{})), `class="field-control search-field__input"`)
}

func TestChipGroup(t *testing.T) {
	out := renderC(t, ChipGroup([]ChipOption{
		{Value: "a", Label: "Alpha", Selected: true},
		{Value: "b", Label: "Beta", Suggested: true},
		{Value: "c", Label: "Gamma", Disabled: true},
	}, ChipGroupOpts{Name: "greek", Legend: "Letters"}))
	want(t, out, `<fieldset class="ui-chip-group">`)
	want(t, out, `<legend class="field-label ui-chip-group__legend">`)
	want(t, out, `type="radio" name="greek" value="a" checked`)
	want(t, out, `class="ui-choice-chip is-suggested"`)
	want(t, out, `ui-choice-chip__spark`)
	want(t, out, `value="c" disabled`)

	multi := renderC(t, ChipGroup([]ChipOption{{Value: "a", Label: "A"}}, ChipGroupOpts{Name: "m", Multiple: true, Legend: "L", HideLegend: true, Disabled: true}))
	want(t, multi, `type="checkbox"`)
	want(t, multi, `<legend class="visually-hidden">L</legend>`)
	want(t, multi, `disabled`)
}

func TestSegmentedChoice(t *testing.T) {
	out := renderC(t, SegmentedChoice([]SegChoice{
		{Value: "l", Label: "Light", Icon: "check", IconOnly: true},
		{Value: "d", Label: "Dark"},
		{Value: "x", Label: "Off", Disabled: true},
	}, "d", SegmentedChoiceOpts{Name: "theme", AriaLabel: "Theme", Size: "sm"}))
	want(t, out, `class="ui-seg-choice ui-seg-choice--n3 ui-seg-choice--sm" role="radiogroup" aria-label="Theme"`)
	want(t, out, `value="d" checked`)
	if strings.Contains(out, `value="l" checked`) {
		t.Error("only the selected value is checked")
	}
	want(t, out, `<span class="visually-hidden">Light</span>`)
	want(t, out, `value="x" disabled`)
	if got := segChoiceClass(1, SegmentedChoiceOpts{}); !strings.Contains(got, "--n2") {
		t.Errorf("count clamps to 2: %s", got)
	}
	if got := segChoiceClass(9, SegmentedChoiceOpts{}); !strings.Contains(got, "--n6") {
		t.Errorf("count clamps to 6: %s", got)
	}
}
