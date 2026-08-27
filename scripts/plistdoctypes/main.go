// Command plistdoctypes inserts CFBundleDocumentTypes into a packaged
// macOS Info.plist. `fyne package -os darwin` (see Makefile's package-mac
// target) generates Info.plist from a fixed template
// (fyne.io/fyne/v2/cmd/fyne's templates/data/Info.plist) that has no hook
// for document-type / file-association metadata, so this runs as a
// post-processing step instead of being something FyneApp.toml can
// express.
//
// LSHandlerRank is "Alternate" rather than "Owner" or "Default" for both
// declared types: PicFetch is a viewer, not the authoritative editor for
// any of these formats, so it registers as available in "Open With"
// without trying to become the system default.
//
// Run from the repository root:
//
//	go run ./scripts/plistdoctypes path/to/Info.plist
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		// Ignored explicitly: the process is exiting non-zero on the next
		// line, and there is nowhere left to report a failure to write the
		// failure itself.
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: plistdoctypes <path/to/Info.plist>")
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	out, err := insertDocumentTypes(string(data), bareExtensions())
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(out), 0o644)
}
