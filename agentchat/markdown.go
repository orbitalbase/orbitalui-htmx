package agentchat

import (
	"bytes"
	"html"
	"net/url"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// chatMarkdown is the shared converter; goldmark converters are safe for concurrent use.
var chatMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(linkRenderer{}, 999), util.Prioritized(tableRenderer{}, 499))),
)

// tableRenderer wraps each GFM table in a scroll container: a table wider than the
// answer scrolls sideways inside it instead of squeezing its cells or widening the page.
// The table itself is written by Goldmark's own renderer, so its escaping is unchanged.
// Priority 499 sits just ahead of the GFM table renderer's 500; a tie would be unordered.
type tableRenderer struct{}

type tableFuncCapture struct{ render renderer.NodeRendererFunc }

func (c *tableFuncCapture) Register(kind ast.NodeKind, fn renderer.NodeRendererFunc) {
	if kind == east.KindTable {
		c.render = fn
	}
}

func (tableRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	var inner tableFuncCapture
	extension.NewTableHTMLRenderer().RegisterFuncs(&inner)
	reg.Register(east.KindTable, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString(`<div class="ui-chat-table">`)
		}
		status, err := inner.render(w, source, node, entering)
		if !entering {
			_, _ = w.WriteString("</div>\n")
		}
		return status, err
	})
}

// linkRenderer replaces Goldmark's default link renderers because its
// dangerous-URL check does not cover autolinks. Both Markdown links and
// autolinks pass through the same explicit scheme allowlist.
type linkRenderer struct{}

func (linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, renderLink)
	reg.Register(ast.KindAutoLink, renderAutoLink)
	reg.Register(ast.KindImage, renderImage)
}

func renderLink(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	link := node.(*ast.Link)
	if !urlAllowed(link.Destination, true) {
		// Keep the link's formatted label but render it as plain text.
		return ast.WalkContinue, nil
	}
	if entering {
		_, _ = w.WriteString(`<a href="`)
		_, _ = w.Write(util.EscapeHTML(util.URLEscape(link.Destination, true)))
		_ = w.WriteByte('"')
		writeLinkRel(w, link.Destination)
		if link.Title != nil {
			_, _ = w.WriteString(` title="`)
			_, _ = w.Write(util.EscapeHTML(link.Title))
			_ = w.WriteByte('"')
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString(`</a>`)
	}
	return ast.WalkContinue, nil
}

func renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	link := node.(*ast.AutoLink)
	destination := link.URL(source)
	label := link.Label(source)
	if link.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(string(destination)), "mailto:") {
		destination = append([]byte("mailto:"), destination...)
	}
	if !urlAllowed(destination, true) {
		_, _ = w.Write(util.EscapeHTML(label))
		return ast.WalkSkipChildren, nil
	}
	_, _ = w.WriteString(`<a href="`)
	_, _ = w.Write(util.EscapeHTML(util.URLEscape(destination, false)))
	_ = w.WriteByte('"')
	writeLinkRel(w, destination)
	_ = w.WriteByte('>')
	_, _ = w.Write(util.EscapeHTML(label))
	_, _ = w.WriteString(`</a>`)
	return ast.WalkSkipChildren, nil
}

// renderImage never loads a remote image: a model-written ![alt](url) would otherwise make the
// browser fetch an attacker-chosen URL as soon as the reply renders. It becomes a plain link (or its alt text).
func renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	image := node.(*ast.Image)
	label := util.EscapeHTML(image.Text(source))
	if len(label) == 0 {
		label = util.EscapeHTML(image.Destination)
	}
	if !urlAllowed(image.Destination, false) {
		_, _ = w.Write(label)
		return ast.WalkSkipChildren, nil
	}
	_, _ = w.WriteString(`<a href="`)
	_, _ = w.Write(util.EscapeHTML(util.URLEscape(image.Destination, true)))
	_ = w.WriteByte('"')
	writeLinkRel(w, image.Destination)
	_ = w.WriteByte('>')
	_, _ = w.Write(label)
	_, _ = w.WriteString(`</a>`)
	return ast.WalkSkipChildren, nil
}

// writeLinkRel opens absolute http(s) links in a new tab, cut off from this page.
func writeLinkRel(w util.BufWriter, destination []byte) {
	d := strings.ToLower(strings.TrimSpace(string(destination)))
	if strings.HasPrefix(d, "http://") || strings.HasPrefix(d, "https://") {
		_, _ = w.WriteString(` target="_blank" rel="noopener noreferrer"`)
	}
}

func urlAllowed(destination []byte, allowMailto bool) bool {
	canonical, ok := canonicalURL(string(destination))
	if !ok {
		return false
	}
	colon := strings.IndexByte(canonical, ':')
	delimiter := strings.IndexAny(canonical, "/?#")
	if colon < 0 || delimiter >= 0 && delimiter < colon {
		return true // Relative paths, query strings, and fragments.
	}
	scheme := canonical[:colon]
	for i, r := range scheme {
		if i == 0 {
			if r < 'a' || r > 'z' {
				return false
			}
		} else if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '+' || r == '.' || r == '-') {
			return false
		}
	}
	switch scheme {
	case "http", "https":
		return true
	case "mailto":
		return allowMailto
	default:
		return false
	}
}

func canonicalURL(value string) (string, bool) {
	value = html.UnescapeString(value)
	for i := 0; i < 32; i++ {
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return "", false
		}
		if decoded == value {
			var canonical strings.Builder
			for _, r := range value {
				if unicode.IsControl(r) || unicode.IsSpace(r) {
					continue
				}
				canonical.WriteRune(unicode.ToLower(r))
			}
			return canonical.String(), true
		}
		value = html.UnescapeString(decoded)
	}
	return "", false
}

// RenderMarkdown renders src as GitHub-flavoured Markdown for an assistant
// reply. Raw HTML is dropped, link destinations pass an allowlist (http,
// https, mailto and relative URLs), external links open in a new tab, and
// images never load: they become links to their source.
func RenderMarkdown(src string) string {
	var rendered bytes.Buffer
	if err := chatMarkdown.Convert([]byte(src), &rendered); err != nil {
		// Goldmark's conversion is deterministic for string input. Keep an escaped
		// text fallback if a future extension introduces a renderer error.
		return "<p>" + html.EscapeString(src) + "</p>"
	}
	return rendered.String()
}

var (
	_ renderer.NodeRenderer = linkRenderer{}
	_ renderer.NodeRenderer = tableRenderer{}
)
