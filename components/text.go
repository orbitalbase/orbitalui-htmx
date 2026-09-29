// Text — a plain escaped-text component. Many components take their content
// as a templ.Component; Text(s) is the easy "just this sentence" body for
// Alert/StatePanel/Disclosure, so pages never need a one-off templ for a
// single paragraph (docs/ui-components.md §0).
package components

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// Text returns a component that renders s HTML-escaped (the same escaping as
// templ text interpolation — safe for user copy with quotes/angle brackets).
func Text(s string) templ.Component {
	return templ.ComponentFunc(
		func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, templ.EscapeString(s))
			return err
		},
	)
}
