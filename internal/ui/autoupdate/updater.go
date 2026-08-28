// Package autoupdate owns the opt-in release-check/download policy, the
// staged-update lifecycle, and the What's-New cache (whatsnew.go). It has
// no window of its own and does not own cancellation: internal/ui keeps
// its own requestLifecycle for the background check and hands Updater.Start
// a context and a staleness func per call, rather than this package
// promoting that lifecycle type to a shared package.
package autoupdate

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/update"
)

// DefaultDir is the production stage directory: a "picfetch/updates"
// subdirectory of the OS cache dir, or os.TempDir() if that can't be
// resolved.
func DefaultDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "picfetch", "updates")
}

// Updater owns the update.Client (nil until built by Start or a test
// assigns one), the staged-update directory, the background
// check/download's completion signal, the current-version override, and
// the last-check calendar day plus the mutex that guards it.
type Updater struct {
	app fyne.App

	dir    string
	client *update.Client

	done completion.Signal

	currentVersion string

	// dayMu guards lastCheckDay: the check goroutine writes it from its own
	// goroutine while a concurrent reader (internal/ui's currentPreferences,
	// at quit) must not block waiting for it.
	dayMu        sync.Mutex
	lastCheckDay string

	// persist is the day round-trip's seam into internal/preferences,
	// wired by internal/ui to preferences.SaveLastUpdateCheckDay. Keeps
	// this package free of fyne.App for anything but the What's-New cache
	// and reading the build's own version - see CurrentVersion. nil is
	// safe and simply skips persistence.
	persist func(day string)
}

// New builds an Updater for app, staging updates under dir and persisting
// every SetLastCheckDay call through persist - so a crash or quit during a
// background check still records today without the caller waiting for the
// check goroutine.
func New(app fyne.App, dir string, persist func(day string)) *Updater {
	return &Updater{app: app, dir: dir, persist: persist}
}

// Dir/SetDir round-trip the staged-update directory. SetDir exists for
// tests, and for internal/ui's startViewerRuntime, which fills in
// DefaultDir only when nothing set one first - the seam that lets a test
// viewer install a t.TempDir() ahead of it.
func (u *Updater) Dir() string       { return u.dir }
func (u *Updater) SetDir(dir string) { u.dir = dir }

// Client/SetClient round-trip the GitHub Releases client. nil until the
// opt-in is on and Start builds one, or a test assigns one directly
// (httptest + a fake Verifier) to exercise Check/Download without hitting
// GitHub for real.
func (u *Updater) Client() *update.Client     { return u.client }
func (u *Updater) SetClient(c *update.Client) { u.client = c }

// SetCurrentVersion overrides CurrentVersion for tests - production Fyne
// apps ship a real Metadata().Version, but Fyne's test app leaves it empty.
func (u *Updater) SetCurrentVersion(v string) { u.currentVersion = v }

// CurrentVersion is the version update checks compare releases against:
// the test override if one was set, otherwise app's own build metadata.
func (u *Updater) CurrentVersion() string {
	if u.currentVersion != "" {
		return u.currentVersion
	}
	return u.app.Metadata().Version
}

// Now is the update client's own clock once one exists - so a check
// started with a fixed test clock also drives Due/DayString - and
// time.Now otherwise, before any client has been built.
func (u *Updater) Now() time.Time {
	if u.client != nil {
		return u.client.Now()
	}
	return time.Now()
}

// Due reports whether a check should run today, given the last-recorded
// check day.
func (u *Updater) Due(now time.Time) bool {
	return update.Due(u.LastCheckDay(), now)
}

// Done is the background check/download's completion signal - waited on
// by tests, and by internal/ui's harness drain, the same way scan/sort/
// load are.
func (u *Updater) Done() *completion.Signal { return &u.done }

// LastCheckDay/SetLastCheckDay round-trip the local calendar day
// (YYYY-MM-DD) of the last update check, or empty when none has run yet.
func (u *Updater) LastCheckDay() string {
	u.dayMu.Lock()
	defer u.dayMu.Unlock()
	return u.lastCheckDay
}

func (u *Updater) SetLastCheckDay(day string) {
	u.dayMu.Lock()
	defer u.dayMu.Unlock()
	u.lastCheckDay = day
	if u.persist != nil {
		u.persist(day)
	}
}

// RemoveStaleStage removes a staged update whose version is no longer
// newer than CurrentVersion - left behind by, say, downgrading back to an
// older build after already staging a newer one. A no-op when nothing is
// staged.
func (u *Updater) RemoveStaleStage() {
	st, err := update.LoadStage(u.dir)
	if err != nil {
		return
	}
	if !update.Newer(u.CurrentVersion(), st.Version) {
		_ = update.RemoveStage(u.dir)
	}
}

// Start begins the background check/download's completion signal and runs
// it on its own goroutine. It lazily builds the update.Client the first
// time it is called - a Sigstore verifier plus a 30s-timeout HTTP client -
// unless a test has already assigned one via SetClient.
//
// ctx is the caller's per-request context (internal/ui's updateOp
// lifecycle token); stale reports whether that token has since been
// superseded, and is checked once, after Check returns and before the
// result is acted on - mirroring every other requestLifecycle consumer in
// internal/ui. currentVersion is the version to check against, resolved by
// the caller through CurrentVersion so gating decisions (is it due, is
// there an asset for this OS/arch) and the check itself agree on the same
// value.
func (u *Updater) Start(ctx context.Context, stale func() bool, currentVersion string) {
	if u.client == nil {
		ver, err := update.NewSigstoreVerifier()
		if err != nil {
			fyne.LogError("update verifier unavailable", err)
			return
		}
		u.client = update.NewClient(update.Config{
			HTTP:     &http.Client{Timeout: 30 * time.Second},
			Now:      time.Now,
			Verify:   ver,
			StageDir: u.dir,
		})
	}

	done := u.done.Begin()
	go func() {
		defer done()
		rel, err := u.client.Check(ctx, currentVersion)
		if err != nil {
			fyne.LogError("update check failed", err)
			return
		}
		if stale() {
			return
		}
		day := update.DayString(u.client.Now())
		u.SetLastCheckDay(day)
		if rel == nil {
			return
		}
		_, err = u.client.Download(ctx, *rel)
		if err != nil {
			fyne.LogError("update download failed", err)
			return
		}
		// Finished stage is kept even if the token was cancelled mid-download
		// (SetCheckForUpdates(false)): apply-on-stop still needs the bits.
	}()
}

// ApplyStagedUpdate replaces the running binary with a completed stage, if
// one exists and is still newer than CurrentVersion - called from
// internal/ui's shutdown handler, after the event loop has stopped taking
// input. Safe to call with nothing staged.
func (u *Updater) ApplyStagedUpdate() {
	st, err := update.LoadStage(u.dir)
	if err != nil {
		return
	}
	if !update.Newer(u.CurrentVersion(), st.Version) {
		_ = update.RemoveStage(u.dir)
		return
	}
	if err := SaveWhatsNew(u.app, st.Version, st.Notes); err != nil {
		fyne.LogError("failed to store release notes", err)
	}
	dest, err := os.Executable()
	if err != nil {
		fyne.LogError("update apply skipped", err)
		return
	}
	if err := update.Apply(st, dest); err != nil {
		fyne.LogError("failed to apply update", err)
		return
	}
	if runtime.GOOS != "windows" {
		_ = update.RemoveStage(u.dir)
	}
}
