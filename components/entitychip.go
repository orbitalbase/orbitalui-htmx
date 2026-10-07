// EntityChip — a compact "who or what" mark: a 20px avatar and a name, with
// an optional link. It is how a row says which person or organisation it
// belongs to.
package components

import "github.com/a-h/templ"

// EntityChipOpts configures EntityChip.
//
//	Name — the entity; it is also the avatar's initials source.
//	Href — makes the whole chip a link.
//	Organisation — a rounded-square avatar (companies, firms).
//	Anonymous — the silhouette avatar; the name text is still rendered, so
//	    pass a neutral name ("Candidate") when the identity is hidden.
//	Tone — avatar tint: "" (derived from the name), "brand" or "accent".
type EntityChipOpts struct {
	Name         string
	Href         templ.SafeURL
	Organisation bool
	Anonymous    bool
	Tone         string
	Class        string
}

func entityChipClass(o EntityChipOpts) string {
	return joinClass("ui-entity-chip", modifier(o.Href != "", "ui-entity-chip--link"), o.Class)
}

func entityChipAvatar(o EntityChipOpts) AvatarOpts {
	return AvatarOpts{Px: 20, Square: o.Organisation, Anonymous: o.Anonymous, Tone: o.Tone}
}
