// Package assets holds the images the UI embeds into the binary.
//
// They live here, beside the package that draws them, because //go:embed
// cannot reference a parent directory - a file can only be embedded by a
// package at or below its own directory. The app's other artwork stays at
// the module root's assets/ directory instead, since it's consumed by the
// build rather than the program: appIcon.png by the Makefile/FyneApp.toml
// packaging step, header.png by the README.
package assets

import _ "embed"

// TraneWebP contains only the 17 gaze cells from the original Codex v2 atlas.
// scripts/appassets retains their 192x208 pixels in a single row.
//
//go:embed trane.webp
var TraneWebP []byte

// WelcomeWebP is the fallback if the welcome pet atlas cannot be decoded.
//
//go:embed welcome.webp
var WelcomeWebP []byte

// PlaceholderWebP replaces it once an error has left the drop zone empty.
//
//go:embed placeholder.webp
var PlaceholderWebP []byte

// DiggingWebP is shown alongside the folder-scan spinner while a drop is
// being scanned.
//
//go:embed digging.webp
var DiggingWebP []byte

// ComparingWebP is shown alongside the about window.
//
//go:embed comparingImages.webp
var ComparingWebP []byte

// ExplorerIntroPNG shows Trane sorting pictures into similar stacks.
//
//go:embed explorer-intro.png
var ExplorerIntroPNG []byte
