// Package assets provides portable component styles and the small runtimes
// some components opt into. Serve Files under your application's static
// handler, or import the CSS into your own build:
//
//	tokens.css    — primitive and semantic design tokens (import first)
//	theme.css     — the default look for every class the components emit
//	controls.css  — field focus treatment shared with host-native controls
//	inspector.css — structural styles for the docked record inspector
//	inspector.js  — the inspector runtime (resize, fullscreen, navigation)
//	menu.js       — optional menu dismissal and sidebar-section memory
//	orbital-ui.js — dialog, drawer and entity-picker runtime (load before Alpine)
package assets

import "embed"

//go:embed tokens.css theme.css inspector.js inspector.css controls.css menu.js orbital-ui.js
var Files embed.FS
