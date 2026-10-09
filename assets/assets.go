// Package assets provides portable component styles and the small runtimes
// some components opt into. Serve Files under your application's static
// handler, or import the CSS into your own build:
//
//	tokens.css    — primitive and semantic design tokens, light and dark (import first)
//	base.css      — optional type scale, prose and small utilities
//	motion.css    — optional motion language (enter, press, shimmer, orb, ring)
//	theme.css     — the default look for every class the components emit
//	shell.css     — optional AppShell, command palette and toast styles
//	theme.js      — optional pre-paint theme script (load synchronously in <head>)
//	controls.css  — field focus treatment shared with host-native controls
//	inspector.css — structural styles for the docked record inspector
//	inspector.js  — the inspector runtime (resize, fullscreen, navigation)
//	menu.js       — optional menu dismissal and sidebar-section memory
//	orbital-ui.js — dialog, drawer and entity-picker runtime (load before Alpine)
//	shell.js      — optional shell runtime: sidebar, palette, toasts, theme switch (any order)
//	agent-chat.css — the agentchat package's look: thread, steps, composer, cards (after theme.css and motion.css)
//	agent-chat.js  — the agentchat runtime: turns over SSE, streaming render, cards (plain DOM, any order)
package assets

import "embed"

//go:embed tokens.css base.css motion.css theme.css shell.css theme.js inspector.js inspector.css controls.css menu.js orbital-ui.js shell.js agent-chat.css agent-chat.js
var Files embed.FS
