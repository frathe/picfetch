// Command picfetch is a desktop image viewer: drop files or folders onto
// it (or open them from the file dialog) and page through them.
//
// Package main owns app setup, translations and command-line arguments;
// main_startup.go keeps the production startup order observable. Everything
// else lives in internal/ui; see ARCHITECTURE.md for the package map.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/openwith"
	"github.com/frathe/picfetch/internal/similarity"
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

// thirdPartyNotices is the same document distributed beside the executable.
// Embedding it keeps Help -> Licenses available offline in every distribution.
//
//go:embed THIRD-PARTY-NOTICES.md
var thirdPartyNotices string

// privacyPolicy keeps the installed build's canonical policy available offline.
//
//go:embed PRIVACY.md
var privacyPolicy string

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

// launchArgs turns the raw command line into the paths and options main
// starts with. exit is negative when the launch should go ahead, and
// otherwise the process's exit status: 0 for --help, which is a successful
// answer to a question rather than a failed launch, and 2 for a bad flag.
//
// A bad flag stops the launch instead of being ignored, because the failure
// this guards is a typo in an autostart unit or a shell script, where a
// window that opens with the flag quietly dropped looks like the flag did
// nothing. The writers are parameters so a test can read both streams
// without touching the process's own.
func launchArgs(args []string, stdout, stderr io.Writer) (paths []string, opts launch.Options, exit int) {
	paths, opts, err := launch.Parse(args)
	switch {
	case errors.Is(err, launch.ErrHelp):
		_, _ = io.WriteString(stdout, launch.Usage())

		return nil, launch.Options{}, 0
	case err != nil:
		_, _ = fmt.Fprintf(stderr, "picfetch: %v\n\n%s", err, launch.Usage())

		return nil, launch.Options{}, 2
	}

	return paths, opts, -1
}

// cleanupLaunchPredecessor admits the pre-app filesystem cleanup only for
// ordinary portable launches. The callback keeps this boundary desktop-free.
func cleanupLaunchPredecessor(policy launch.Policy, cleanup func()) {
	if policy.Updates().Allowed() {
		cleanup()
	}
}

func main() {
	exit, err := runStartup(os.Args[1:], os.Stdout, os.Stderr, productionStartup())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
	if exit != 0 {
		os.Exit(exit)
	}
}

func productionStartup() startupOps {
	return startupOps{
		heicWorker:       heic.WorkerMain,
		similarityWorker: similarity.WorkerMain,
		installOpenWith: func() {
			// Cocoa caches delegate selectors during glfw.Init. Installation
			// belongs before app construction. False is normal off macOS.
			openwith.Install()
		},
		cleanup: update.CleanupPredecessor,
		capture: func(opts launch.Options) (launch.Policy, error) {
			return launch.NewPolicy(opts, appID, distribution.StoreManaged)
		},
		prerequisites: func(ctx context.Context, policy launch.Policy) error {
			if policy.Purpose() == launch.ExplorerTrial {
				return similarity.VerifyOffline(ctx)
			}
			return nil
		},
		newApp: func(identity string) (fyne.App, error) {
			application := app.NewWithID(identity)
			if err := lang.AddTranslationsFS(translationsFS, "translations"); err != nil {
				fyne.LogError("failed to load translations", err)
			}
			return application, nil
		},
		run: func(application fyne.App, initial []fyne.URI, opts launch.Options, policy launch.Policy) error {
			return ui.Run(application, initial, opts, policy, thirdPartyNotices, privacyPolicy)
		},
	}
}
