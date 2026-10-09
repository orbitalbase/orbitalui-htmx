package agentchat

import (
	"strings"
	"testing"
)

func TestMarkdownRendersCommonChatFormatting(t *testing.T) {
	input := "# Role overview\n\n" +
		"**Product designer** with *research* experience.\n\n" +
		"- Lead discovery\n  - Interview customers\n    - Share findings\n\n" +
		"```go\nfmt.Println(\"hello\")\n```\n\n" +
		"| Need | Detail |\n| --- | --- |\n| Location | Remote |\n"
	got := RenderMarkdown(input)
	for _, want := range []string{
		"<h1>Role overview</h1>", "<strong>Product designer</strong>", "<em>research</em>",
		"<ul>", "<li>Interview customers", "<li>Share findings", "<pre><code class=\"language-go\">",
		"<table>", "<th>Need</th>", "<td>Remote</td>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered Markdown omitted %q:\n%s", want, got)
		}
	}
}

func TestMarkdownRejectsRawHTMLAndUnsafeLinks(t *testing.T) {
	unsafe := []string{
		`[x](javascript:alert(1))`, `[x](JaVaScRiPt:alert(1))`,
		`[x](vbscript:alert(1))`, `[x](data:text/html,alert(1))`,
		`[x](java&#x73;cript:alert(1))`, `[x](java%73cript:alert(1))`,
		`[x](java%0ascript:alert(1))`, `<javascript:alert(1)>`,
		`<JaVaScRiPt:alert(1)>`, `<vbscript:alert(1)>`, `<data:text/html,alert(1)>`,
		`![x](data:image/svg+xml;base64,PHN2Zz4=)`,
	}
	for _, input := range unsafe {
		got := strings.ToLower(RenderMarkdown(input))
		if strings.Contains(got, "<a ") || strings.Contains(got, "<img ") {
			t.Errorf("unsafe destination remained clickable for %q: %s", input, got)
		}
	}

	rawHTML := strings.ToLower(RenderMarkdown(`<script>alert(1)</script><img src=x onerror="alert(1)">`))
	if strings.Contains(rawHTML, "<script") || strings.Contains(rawHTML, "<img") || strings.Contains(rawHTML, "onerror=") {
		t.Errorf("raw HTML was emitted: %s", rawHTML)
	}
	code := RenderMarkdown("```html\n<script>alert(1)</script>\n```")
	if !strings.Contains(code, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("code sample was not preserved as escaped code: %s", code)
	}
}

func TestMarkdownAllowsSafeLinkSchemesAndRelativeURLs(t *testing.T) {
	got := RenderMarkdown(`[https](https://example.com) [http](http://example.com) [email](mailto:a@example.com) [root](/jobs) [relative](../jobs) [fragment](#role) [query](?q=role) <https://example.com>`)
	for _, want := range []string{
		`href="https://example.com"`, `href="http://example.com"`,
		`href="mailto:a@example.com"`, `href="/jobs"`, `href="../jobs"`,
		`href="#role"`, `href="?q=role"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("safe URL omitted %q: %s", want, got)
		}
	}
}

func TestMarkdownNeverLoadsRemoteImagesAndOpensLinksSafely(t *testing.T) {
	got := RenderMarkdown("![logo](https://evil.example/t.png) and [docs](https://example.com/a)")
	if strings.Contains(got, "<img") || strings.Contains(got, "src=") {
		t.Fatalf("images must not render: %s", got)
	}
	if !strings.Contains(got, `<a href="https://evil.example/t.png" target="_blank" rel="noopener noreferrer">logo</a>`) {
		t.Fatalf("image should become a link: %s", got)
	}
	if !strings.Contains(got, `<a href="https://example.com/a" target="_blank" rel="noopener noreferrer">docs</a>`) {
		t.Fatalf("links open in a new tab: %s", got)
	}
}

func TestMarkdownComponentRendersTheSameHTML(t *testing.T) {
	got := render(t, Markdown("A **bold** answer."))
	if got != "<p>A <strong>bold</strong> answer.</p>\n" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownWrapsTablesInAScrollContainer(t *testing.T) {
	got := RenderMarkdown("Intro\n\n| Account | <b>Debit</b> |\n| --- | --- |\n| Cash | 1 < 2 |\n\nAfter")
	if !strings.Contains(got, `<div class="ui-chat-table"><table>`) || !strings.Contains(got, "</table>\n</div>") {
		t.Fatalf("table is not wrapped: %s", got)
	}
	if strings.Count(got, `class="ui-chat-table"`) != 1 || strings.Contains(got, "<b>") || !strings.Contains(got, "1 &lt; 2") {
		t.Fatalf("table output changed or is unsafe: %s", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "<p>After</p>") {
		t.Fatalf("content after the table lost: %s", got)
	}
}
