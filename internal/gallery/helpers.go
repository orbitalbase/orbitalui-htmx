package main

import (
	"context"
	"embed"
	"html"
	"io"

	"github.com/a-h/templ"
)

//go:embed presets/*.css
var presetFS embed.FS

// galleryCSS is the gallery's own chrome (g-* classes). It never styles a
// library class, so what the screenshots show is the shipped theme.
const galleryCSS = `
*, ::before, ::after { box-sizing: border-box; }
body { margin: 0; background: var(--oui-canvas); color: var(--oui-text-primary); font-family: var(--oui-font-sans); font-size: var(--oui-text-body); line-height: var(--oui-leading-body); }
.g-card-center { --oui-card-header-align: center; }
.g-shell { display: grid; grid-template-columns: 13rem minmax(0, 1fr); min-height: 100vh; }
.g-nav { position: sticky; top: 0; align-self: start; display: grid; gap: 2px; max-height: 100vh; overflow: auto; padding: 1rem 0.75rem; border-right: 1px solid var(--oui-line); font-size: 13px; }
.g-nav a { padding: 4px 8px; border-radius: 6px; color: var(--oui-text-tertiary); text-decoration: none; }
.g-nav a:hover { background: var(--oui-surface-2); color: var(--oui-text-primary); }
.g-nav a[aria-current] { color: var(--oui-text-primary); font-weight: 600; }
.g-nav hr { width: 100%; border: 0; border-top: 1px solid var(--oui-line); margin: 8px 0; }
.g-main { min-width: 0; padding: 1.5rem 2rem 4rem; }
.g-section { margin-bottom: 3rem; }
.g-section__title { margin: 0 0 0.25rem; font-size: 20px; line-height: 28px; font-weight: 650; letter-spacing: -0.01em; }
.g-section__note { margin: 0 0 1rem; color: var(--oui-text-quaternary); font-size: 13px; }
.g-group { margin: 1.25rem 0; }
.g-group__title { margin: 0 0 0.5rem; color: var(--oui-text-quaternary); font-size: 11px; font-weight: 600; letter-spacing: 0.06em; text-transform: uppercase; }
.g-row { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.g-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr)); gap: 16px; align-items: start; }
.g-cell { display: grid; gap: 6px; min-width: 0; }
.g-cell__label { color: var(--oui-text-quaternary); font-size: 11px; font-family: var(--oui-font-mono); }
.g-stack { display: grid; gap: 12px; }
.g-panel { padding: 16px; background: var(--oui-surface); border-radius: 12px; box-shadow: 0 0 0 1px var(--oui-line); }
.g-list { padding-left: 1.2rem; }
@media (max-width: 42rem) { .g-shell { grid-template-columns: 1fr; } .g-nav { position: static; max-height: none; grid-auto-flow: column; overflow-x: auto; border-right: 0; border-bottom: 1px solid var(--oui-line); } .g-nav hr { display: none; } .g-main { padding: 1rem; } }
`

// group renders a titled block of cells (a flowing row by default).
func group(title string, cells ...templ.Component) templ.Component {
	return wrap(`<div class="g-group"><h3 class="g-group__title">`+html.EscapeString(title)+`</h3><div class="g-row">`, `</div></div>`, cells...)
}

// groupGrid is group laid out as a grid of equal columns (cards, panels).
func groupGrid(title string, cells ...templ.Component) templ.Component {
	return wrap(`<div class="g-group"><h3 class="g-group__title">`+html.EscapeString(title)+`</h3><div class="g-grid">`, `</div></div>`, cells...)
}

// groupStack stacks cells vertically at full width (tables, alerts, forms).
func groupStack(title string, cells ...templ.Component) templ.Component {
	return wrap(`<div class="g-group"><h3 class="g-group__title">`+html.EscapeString(title)+`</h3><div class="g-stack">`, `</div></div>`, cells...)
}

// cell labels one component instance with its variant name.
func cell(label string, c templ.Component) templ.Component {
	return wrap(`<div class="g-cell"><span class="g-cell__label">`+html.EscapeString(label)+`</span>`, `</div>`, c)
}

// plain renders a component with no label (use inside group when the group
// title already names the variant).
func plain(c templ.Component) templ.Component { return c }

// seq renders components one after another.
func seq(cs ...templ.Component) templ.Component { return wrap("", "", cs...) }

// raw is trusted static markup.
func raw(s string) templ.Component { return templ.Raw(s) }

func wrap(open, closing string, cs ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, open); err != nil {
			return err
		}
		for _, c := range cs {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, closing)
		return err
	})
}
