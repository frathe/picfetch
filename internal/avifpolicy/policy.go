//go:build nodynamic && !wasm2go

// Package avifpolicy makes the viewer's required AVIF build selection a compile
// dependency. Builds without nodynamic, or with wasm2go, have no buildable files
// in this package and fail before any decoder can initialize. Use the Makefile
// or pass -tags=no_emoji,nodynamic to direct Go commands.
package avifpolicy
