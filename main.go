// Command picfetch is a desktop image viewer: drop files or folders onto
// it (or open them from the file dialog) and page through them.
//
// This file is the whole of package main - app setup, translations, and
// the command-line arguments. Everything else lives in internal/ui; see
// ARCHITECTURE.md for the package map.
package main

import (
	"embed"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/openwith"
	"github.com/frathe/picfetch/internal/ui"
	"github.com/frathe/picfetch/internal/update"
)

// translationsFS stays here rather than moving into internal/ui with the
// rest of the app: lang.AddTranslationsFS loads into fyne's process-wide
// bundle, which every lang.L call reads from wherever it lives, so this is
// app setup rather than UI code.
//
//go:embed translations/*.json
var translationsFS embed.FS

// appID is the stable key Fyne uses for application-scoped preferences and
// cache data. Keep it in sync with FyneApp.toml's ID and the Makefile's
// PACKAGE_ID (the bundle identifier packaging stamps in); changing it would
// make an existing installation appear to lose its saved settings and
// session. It is reverse-DNS because it doubles as the macOS
// CFBundleIdentifier, which allows only alphanumerics, hyphens, and dots.
const appID = "io.github.frathe.picfetch"

// argsToURIs converts command-line paths (os.Args[1:]) into file URIs
// handleDrop can ingest, so launching the binary with paths - the way a
// macOS file association or "Open With" launches it - works the same as a
// drag-and-drop. Relative paths are resolved against the current working
// directory; anything that fails to resolve is skipped rather than aborting
// the whole batch, since one bad argument shouldn't stop the rest from
// loading. Existence/format isn't checked here - handleDrop's own scan and
// attemptLoad's retry chain already handle a bad path gracefully, the same
// as a bad drag-drop.
func argsToURIs(args []string) []fyne.URI {
	uris := make([]fyne.URI, 0, len(args))
	for _, a := range args {
		// filepath.Abs("") resolves to the working directory. An explicitly
		// empty command-line argument should not unexpectedly scan that entire
		// directory as though the user had opened it.
		if a == "" {
			continue
		}
		abs, err := filepath.Abs(a)
		if err != nil {
			continue
		}
		uris = append(uris, storage.NewFileURI(abs))
	}
	return uris
}

func main() {
	// First statement in the process, before the fyne.App exists.
	// openwith.Install grafts the "Open With" methods onto GLFW's
	// application delegate class, and -[NSApplication setDelegate:] caches
	// which selectors its delegate answers at the moment it is called -
	// which GLFW does inside glfw.Init(), so a method grafted after that
	// is never consulted no matter that the runtime can now find it.
	// app.NewWithID below creates no window and so never reaches initGLFW,
	// which is what makes this position both early enough and safe.
	//
	// The result is deliberately dropped rather than logged: it is false on
	// every non-macOS build by design, so logging it would put a line in
	// every Linux and Windows launch that means nothing, and on macOS false
	// says only that the Cocoa driver isn't linked or that the methods were
	// already grafted. Neither is actionable, and neither is a reason not
	// to start - the app simply behaves as it did before, ignoring
	// "Open With".
	openwith.Install()

	// Before app.NewWithID: an update relaunch must not read or write
	// preferences while the process it replaced is still flushing its own.
	// Microsoft Store builds never stage or apply GitHub-delivered binaries,
	// so they must not inspect or clean that channel's predecessor files.
	// Qodana analyzes the default build, where StoreManaged is a constant false;
	// the microsoftstore build tag replaces it with the true variant.
	//goland:noinspection GoBoolExpressions
	if !distribution.StoreManaged {
		update.CleanupPredecessor()
	}

	application := app.NewWithID(appID)

	if err := lang.AddTranslationsFS(translationsFS, "translations"); err != nil {
		fyne.LogError("failed to load translations", err)
	}

	ui.Run(application, argsToURIs(os.Args[1:]))
}
