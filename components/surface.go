// Page and surface primitives: the page header, the card that frames a
// working area, and the settings row that pairs a label column with its
// controls (surface.templ). They own spacing and hierarchy only; every slot
// takes a templ.Component so the host keeps ownership of content, endpoints
// and permissions.
package components

import (
	"strings"

	"github.com/a-h/templ"
)

// PageHeaderOpts configures PageHeader.
//
//	Eyebrow — a short context label above the title ("Company · Acme Sdn Bhd").
//	Description — one sentence under the title; keep it to the page's purpose.
//	Meta — optional inline status row under the title (badges, timestamps).
//	Actions — right-aligned buttons; place the single primary action last.
//	Tabs — optional tab bar rendered flush under the header.
//	Leading — a mark before the copy: an Avatar for a record page (square for
//	    an organisation), a logo, a monogram. It is decoration beside the
//	    title, so give the Avatar no Label.
//	Compact — a denser header for registers and split views.
type PageHeaderOpts struct {
	Eyebrow     string
	Description string
	Meta        templ.Component
	Actions     templ.Component
	Tabs        templ.Component
	Leading     templ.Component
	Compact     bool
	ID          string
	Class       string
}

func pageHeaderClass(o PageHeaderOpts) string {
	return joinClass("ui-page-header", modifier(o.Compact, "ui-page-header--compact"), o.Class)
}

// CardTone tints a card's frame for consequence: danger frames a destructive
// zone, brand highlights the one card that leads the page.
type CardTone string

const (
	CardToneDefault CardTone = ""
	CardToneBrand   CardTone = "brand"
	CardToneDanger  CardTone = "danger"
	CardToneSubtle  CardTone = "subtle"
)

// CardOpts configures Card.
//
//	Title/Description — the card header. Title "" renders a header-less card.
//	Eyebrow — a small label above the title.
//	Badge — inline component beside the title (a count or status chip).
//	Actions — header-right controls (a text link or a small button).
//	Footer — a separated footer row (pagination, "View all", a save bar).
//	Flush — remove body padding so a table or list can run edge to edge.
//	HeadingLevel — 2 (default) or 3, so nested cards keep a sound outline.
type CardOpts struct {
	Title        string
	Description  string
	Eyebrow      string
	Badge        templ.Component
	Actions      templ.Component
	Footer       templ.Component
	Flush        bool
	Tone         CardTone
	HeadingLevel int
	ID           string
	Class        string
	Attrs        templ.Attributes
}

func cardClass(o CardOpts) string {
	tone := ""
	if o.Tone != CardToneDefault {
		tone = "ui-card--" + string(o.Tone)
	}
	return joinClass("ui-card", tone, modifier(o.Flush, "ui-card--flush"), o.Class)
}

func cardHeadingID(o CardOpts) string {
	if o.ID == "" {
		return ""
	}
	return o.ID + "-title"
}

// SettingsSectionOpts configures SettingsSection: a two-column row where the
// left column names and explains a group of settings and the right column
// holds the controls. It collapses to one column on narrow screens.
type SettingsSectionOpts struct {
	Title       string
	Description string
	Aside       templ.Component // optional extra content under the description
	ID          string
	Class       string
}

func settingsSectionClass(o SettingsSectionOpts) string {
	return joinClass("ui-settings-section", o.Class)
}

// joinClass joins non-empty class names with single spaces.
func joinClass(parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " ")
}

func modifier(on bool, class string) string {
	if on {
		return class
	}
	return ""
}

// RecordHeaderOpts configures RecordHeader.
//
//	Title — the record's name. It renders as a paragraph, not a heading: the
//	    inspector chrome already owns the panel heading, and a record header
//	    inside it must not compete in the outline.
//	Eyebrow — the record type ("Statutory officer", "Company action").
//	Subtitle — one line of context ("Director · authorised signatory").
//	Leading — an Avatar or logo; Icon is used when Leading is nil, in a tile
//	    tinted by IconTone.
//	Status — a badge on the trailing edge.
type RecordHeaderOpts struct {
	Title    string
	Eyebrow  string
	Subtitle string
	Leading  templ.Component
	Icon     string
	IconTone Tone
	Status   templ.Component
	Class    string
}

func recordHeaderIconClass(o RecordHeaderOpts) string {
	tone := ""
	if o.IconTone != "" && o.IconTone != ToneNeutral {
		tone = "ui-record-header__icon--" + o.IconTone.Modifier()
	}
	return joinClass("ui-record-header__icon", tone)
}
