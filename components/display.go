// Read-only display primitives (display.templ): description lists for record
// facts, initials avatars, progress bars, onboarding checklists, step
// timelines and activity feed rows. Each is plain server HTML with no runtime.
package components

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/a-h/templ"
)

// DescriptionItem is one term/value pair. Value is plain text; Content, when
// set, replaces it with host-rendered markup (a badge, a link). Empty values
// render the Placeholder of the list ("Not set" by default) so a missing fact
// is visible rather than silently absent.
type DescriptionItem struct {
	Term    string
	Value   string
	Content templ.Component
	Hint    string // optional muted line under the value
	Wide    bool   // span both columns in the grid layout
}

// DescriptionLayout selects how terms and values are arranged.
type DescriptionLayout string

const (
	// DescriptionInline puts the term in a fixed left column (inspectors).
	DescriptionInline DescriptionLayout = ""
	// DescriptionStacked puts the term above the value (narrow rails).
	DescriptionStacked DescriptionLayout = "stacked"
	// DescriptionGrid lays stacked pairs out in two columns (summaries).
	DescriptionGrid DescriptionLayout = "grid"
)

type DescriptionListOpts struct {
	Layout      DescriptionLayout
	Placeholder string
	Class       string
}

func descriptionListClass(o DescriptionListOpts) string {
	layout := "ui-dl--inline"
	if o.Layout != DescriptionInline {
		layout = "ui-dl--" + string(o.Layout)
	}
	return joinClass("ui-dl", layout, o.Class)
}

func descriptionPlaceholder(o DescriptionListOpts) string {
	if o.Placeholder == "" {
		return "Not set"
	}
	return o.Placeholder
}

// AvatarSize is a size preset for Avatar.
type AvatarSize string

const (
	AvatarXS AvatarSize = "xs"
	AvatarSM AvatarSize = "sm"
	AvatarMD AvatarSize = ""
	AvatarLG AvatarSize = "lg"
	AvatarXL AvatarSize = "xl"
)

// AvatarOpts configures Avatar. Square renders a rounded tile, used for
// organisations (companies, firms) so they read differently from people.
// Tone "" derives a stable hue from the name so lists stay scannable.
type AvatarOpts struct {
	Size   AvatarSize
	Square bool
	Tone   string // "", "brand", "gray"
	Status string // "", "online", "away" — a small presence dot
	Label  string // accessible name; "" keeps the avatar decorative
	Class  string
}

// avatarHues are the derived tones; each pairs with a .ui-avatar--hue-N rule.
const avatarHues = 6

func avatarClass(name string, o AvatarOpts) string {
	size := ""
	if o.Size != AvatarMD {
		size = "ui-avatar--" + string(o.Size)
	}
	tone := ""
	switch o.Tone {
	case "brand", "gray":
		tone = "ui-avatar--" + o.Tone
	default:
		tone = "ui-avatar--hue-" + strconv.Itoa(nameHue(name))
	}
	return joinClass("ui-avatar", size, modifier(o.Square, "ui-avatar--square"), tone, o.Class)
}

// nameHue maps a name to a stable hue bucket.
func nameHue(name string) int {
	sum := 0
	for _, r := range strings.ToLower(name) {
		sum = (sum*31 + int(r)) % 9973
	}
	return sum % avatarHues
}

// Initials returns up to two uppercase initials for a display name. It
// prefers the first letters of the first and last words and falls back to
// "?" for blank names.
func Initials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return unicode.IsSpace(r) || r == '.' || r == '-' || r == '_' || r == '@'
	})
	letters := make([]rune, 0, 2)
	for i, w := range words {
		if i != 0 && i != len(words)-1 {
			continue
		}
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				letters = append(letters, unicode.ToUpper(r))
				break
			}
		}
	}
	if len(letters) == 0 {
		return "?"
	}
	return string(letters)
}

// ProgressOpts configures ProgressBar.
type ProgressOpts struct {
	Label     string // visible label left of the value
	ShowValue bool   // render "40%" at the end of the label row
	Tone      Tone   // "" brand; success/warning/danger for consequence
	Class     string
}

func progressClass(o ProgressOpts) string {
	tone := ""
	if o.Tone != "" && o.Tone != ToneNeutral {
		tone = "ui-progress--" + o.Tone.Modifier()
	}
	return joinClass("ui-progress", tone, o.Class)
}

func clampPercent(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func progressStyle(v int) templ.SafeCSS {
	return templ.SafeCSS("--ui-progress:" + strconv.Itoa(clampPercent(v)) + "%")
}

// ChecklistItem is one onboarding or readiness step. Href/ActionLabel make
// the next open step actionable; done steps stay visible but quiet.
type ChecklistItem struct {
	Title       string
	Description string
	Done        bool
	Href        templ.SafeURL
	ActionLabel string
}

// StepState is the state of one timeline step.
type StepState string

const (
	StepDone     StepState = "done"
	StepCurrent  StepState = "current"
	StepUpcoming StepState = "upcoming"
	StepBlocked  StepState = "blocked"
)

// Step is one row in a Steps timeline.
type Step struct {
	Title   string
	Meta    string // owner, date or state text under the title
	State   StepState
	Content templ.Component // optional detail under the meta line
}

func stepClass(s Step) string {
	state := s.State
	if state == "" {
		state = StepUpcoming
	}
	return "ui-steps__item ui-steps__item--" + string(state)
}

func stepStateLabel(s StepState) string {
	switch s {
	case StepDone:
		return "Complete"
	case StepCurrent:
		return "Current step"
	case StepBlocked:
		return "Blocked"
	default:
		return "Upcoming"
	}
}

// FeedItemOpts configures FeedItem, one row of an activity or notification
// feed. Unread rows show a brand dot and stronger title.
type FeedItemOpts struct {
	Title   string
	Body    string
	Time    string
	Href    templ.SafeURL
	Unread  bool
	Icon    string
	Tone    Tone
	Actions templ.Component
	Class   string
}

func feedItemClass(o FeedItemOpts) string {
	return joinClass("ui-feed__item", modifier(o.Unread, "ui-feed__item--unread"), o.Class)
}

func feedIconClass(o FeedItemOpts) string {
	tone := ""
	if o.Tone != "" && o.Tone != ToneNeutral {
		tone = "ui-feed__icon--" + o.Tone.Modifier()
	}
	return joinClass("ui-feed__icon", tone)
}
