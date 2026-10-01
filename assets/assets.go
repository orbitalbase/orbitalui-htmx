// Package assets provides portable component styles and the inspector runtime.
// Serve Files under your application's authenticated or public static handler.
package assets

import "embed"

//go:embed inspector.js inspector.css controls.css
var Files embed.FS
