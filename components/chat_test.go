package components

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestChatPrimitivesSupportCitedContentAndNativeSuggestions(t *testing.T) {
	var output bytes.Buffer
	thread := ChatThread("conversation", "Company conversation", "thread", AssistantMessage("<script>untrusted answer</script>"))
	if err := thread.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`role="log"`, `aria-live="polite"`, `oui-assistant-message__body`, `&lt;script&gt;`} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing %s", expected)
		}
	}
	output.Reset()
	content := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<a href="/documents/one">Source [1]</a>`)
		return err
	})
	if err := AssistantMessageContent(content).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `href="/documents/one"`) || !strings.Contains(output.String(), `aria-label="Assistant message"`) {
		t.Fatal("rich message lost its source link or accessible label")
	}
	output.Reset()
	if err := SuggestionSubmitChip("Ask", "suggestion", "What approvals?").Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `type="submit"`) || !strings.Contains(output.String(), `name="suggestion"`) || strings.Contains(output.String(), "@click") {
		t.Fatal("suggestion must work without Alpine or JavaScript")
	}
}
