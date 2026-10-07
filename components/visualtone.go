// Tone helpers for the display components (meter, ring, timeline, progress,
// feed). They accept the full identity vocabulary (brand and accent as well as
// the state tones) independent of Tone.Valid, so a display component never
// silently drops a brand or accent tint.
package components

// ToneBrandName and ToneAccentName are the identity tones accepted by the
// display components as Tone("brand") / Tone("accent").
const (
	toneBrandName  Tone = "brand"
	toneAccentName Tone = "accent"
)

// visualTone returns the CSS modifier for t ("brand", "ai", …) and "" for the
// default (neutral or empty) tone, which reads the primary-action colour.
// Unknown values collapse to the default, never to a wrong colour.
func visualTone(t Tone) string {
	switch t {
	case toneBrandName, toneAccentName, ToneInfo, ToneSuccess, ToneWarning, ToneDanger, ToneAI:
		return string(t)
	}
	return ""
}
