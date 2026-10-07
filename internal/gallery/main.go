// Command gallery writes static HTML pages that render every component and
// state with the shipped stylesheets (no Tailwind, no runtime server). It is
// the visual contract for the library: review it in light, dark and the
// Kindred preset, at desktop and phone width.
//
//	go run ./internal/gallery -out /tmp/oui-gallery
//
// Each preset gets its own directory: default (library tokens), dark
// (data-theme="dark"), kindred and kindred-dark (internal/gallery/presets
// maps Kindred's token values onto --oui-* so shared components can be
// compared with Kindred's own kit). Open <out>/index.html to pick one; each
// preset directory has index.html (every section) and one page per section.
// Pages carry no orbital-ui.js or Alpine: overlays are pinned open statically
// (an open <dialog> would otherwise be promoted to a real modal).
package main

import (
	"context"
	"flag"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"

	"github.com/orbitalbase/orbitalui-htmx/assets"
)

// section is one gallery block: every variant and state of a component family.
// Register sections from init() in sections_<area>.go.
type section struct {
	ID    string // url-safe slug, also the per-section page name
	Title string
	Note  string // optional one-line description shown under the title
	Body  templ.Component
}

var registry []section

func register(s section) { registry = append(registry, s) }

type preset struct {
	Name       string
	Theme      string // data-theme attribute value ("" = none)
	Stylesheet string // preset stylesheet under presets/, loaded after the theme
}

var presets = []preset{
	{Name: "default"},
	{Name: "dark", Theme: "dark"},
	{Name: "kindred", Theme: "light", Stylesheet: "kindred.css"},
	{Name: "kindred-dark", Theme: "dark", Stylesheet: "kindred.css"},
}

var sheets = []string{"tokens.css", "base.css", "motion.css", "controls.css", "theme.css", "inspector.css"}

func main() {
	out := flag.String("out", "gallery-out", "output directory")
	fonts := flag.String("kindred-fonts", "", "directory holding Kindred's woff2 files (optional; the preset falls back to system fonts)")
	flag.Parse()
	if err := run(*out, *fonts); err != nil {
		log.Fatal(err)
	}
	fmt.Println(filepath.Join(*out, "index.html"))
}

func run(out, fontDir string) error {
	if err := os.MkdirAll(filepath.Join(out, "assets"), 0o755); err != nil {
		return err
	}
	for _, name := range append(append([]string{}, sheets...), "orbital-ui.js", "menu.js", "inspector.js") {
		b, err := fs.ReadFile(assets.Files, name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, "assets", name), b, 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(out, "assets", "gallery.css"), []byte(galleryCSS), 0o644); err != nil {
		return err
	}
	if err := copyPresets(out, fontDir); err != nil {
		return err
	}
	var index strings.Builder
	index.WriteString(pageHead("Orbital UI gallery", "", "", "") + `<main class="g-main"><h1>Orbital UI gallery</h1><ul class="g-list">`)
	for _, p := range presets {
		dir := filepath.Join(out, p.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := writePage(filepath.Join(dir, "index.html"), p, "All components", registry); err != nil {
			return err
		}
		for _, s := range registry {
			if err := writePage(filepath.Join(dir, s.ID+".html"), p, s.Title, []section{s}); err != nil {
				return err
			}
		}
		fmt.Fprintf(&index, `<li><a href="%s/index.html">%s</a></li>`, p.Name, p.Name)
	}
	index.WriteString("</ul></main></body></html>")
	return os.WriteFile(filepath.Join(out, "index.html"), []byte(index.String()), 0o644)
}

func copyPresets(out, fontDir string) error {
	dir := filepath.Join(out, "presets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := fs.ReadDir(presetFS, "presets")
	if err != nil {
		return err
	}
	for _, e := range entries {
		b, err := fs.ReadFile(presetFS, "presets/"+e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), b, 0o644); err != nil {
			return err
		}
	}
	if fontDir == "" {
		return nil
	}
	fonts, _ := filepath.Glob(filepath.Join(fontDir, "*.woff2"))
	if len(fonts) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(dir, "fonts"), 0o755); err != nil {
		return err
	}
	for _, f := range fonts {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "fonts", filepath.Base(f)), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func pageHead(title, theme, stylesheet, scriptSrc string) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"`)
	if theme != "" {
		fmt.Fprintf(&b, ` data-theme="%s"`, theme)
	}
	b.WriteString(`><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">`)
	fmt.Fprintf(&b, "<title>%s</title>", html.EscapeString(title))
	for _, s := range sheets {
		fmt.Fprintf(&b, `<link rel="stylesheet" href="%sassets/%s">`, rootPrefix(scriptSrc), s)
	}
	if stylesheet != "" {
		fmt.Fprintf(&b, `<link rel="stylesheet" href="%spresets/%s">`, rootPrefix(scriptSrc), stylesheet)
	}
	fmt.Fprintf(&b, `<link rel="stylesheet" href="%sassets/gallery.css"></head><body>`, rootPrefix(scriptSrc))
	return b.String()
}

// rootPrefix is "../" for pages inside a preset directory, "" for the index.
func rootPrefix(scriptSrc string) string { return scriptSrc }

func writePage(path string, p preset, title string, ss []section) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	io.WriteString(f, pageHead("Orbital UI / "+p.Name+" / "+title, p.Theme, p.Stylesheet, "../"))
	io.WriteString(f, `<div class="g-shell"><nav class="g-nav" aria-label="Sections"><a href="../index.html"><strong>Presets</strong></a>`)
	for _, q := range presets {
		cur := ""
		if q.Name == p.Name {
			cur = ` aria-current="true"`
		}
		fmt.Fprintf(f, `<a href="../%s/index.html"%s>%s</a>`, q.Name, cur, q.Name)
	}
	io.WriteString(f, `<hr>`)
	for _, s := range registry {
		fmt.Fprintf(f, `<a href="%s.html">%s</a>`, s.ID, html.EscapeString(s.Title))
	}
	io.WriteString(f, `</nav><main class="g-main" id="main">`)
	for _, s := range ss {
		fmt.Fprintf(f, `<section class="g-section" id="%s"><h2 class="g-section__title">%s</h2>`, s.ID, html.EscapeString(s.Title))
		if s.Note != "" {
			fmt.Fprintf(f, `<p class="g-section__note">%s</p>`, html.EscapeString(s.Note))
		}
		if err := s.Body.Render(context.Background(), f); err != nil {
			return fmt.Errorf("section %s: %w", s.ID, err)
		}
		io.WriteString(f, `</section>`)
	}
	io.WriteString(f, `</main></div><script src="../assets/menu.js" defer></script></body></html>`)
	return nil
}
