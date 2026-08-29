// Package autoupdate owns the opt-in release-check/download policy, the
// staged-update lifecycle, and the What's-New cache (whatsnew.go). It has
// no window of its own and does not own cancellation: internal/ui keeps
// its own requestLifecycle for the background check and hands Updater.Start
// or StartManual a context and a staleness func per call, rather than this package
// promoting that lifecycle type to a shared package.
package autoupdate

import (
	"context"
	"errors"
	"fmt"
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

var errClientNotPrepared = errors.New("update client not prepared")

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

// Updater owns the lazily prepared update.Client, the staged-update
// directory, the background check/download completion signals, the
// current-version override, and the last-check calendar day plus the mutex
// that guards it.
type Updater struct {
	app fyne.App

	dir string

	// clientMu keeps lazy client preparation single-shot when a manual
	// request (which prepares on its worker) meets an automatic request.
	// The client itself is immutable after construction.
	clientMu sync.Mutex
	client   *update.Client

	// Instance-owned to keep preparation lazy and testable without a mutable
	// package seam.
	verifierFactory func() (update.Verifier, error)

	done completion.Signal

	// transaction serializes the whole Check -> optional staged-release reuse
	// -> DownloadWithProgress operation. Both automatic and manual requests
	// share the stage directory, whose remove/extract/write sequence must never
	// overlap another request.
	transaction chan struct{}
	// applyOptions is protected by transaction. Its zero value preserves the
	// automatic/normal-shutdown behavior; RequestApplyAndRelaunch records the
	// explicit intent used by the existing shutdown-time apply point.
	applyOptions update.ApplyOptions

	// workers tracks every generation, not only done's latest one. A manual
	// request can supersede an automatic request while the latter is still
	// unwinding; Settle gives tests and shutdown cleanup an observable barrier
	// for both.
	workersMu   sync.Mutex
	workers     int
	workersDone chan struct{}

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
	workersDone := make(chan struct{})
	close(workersDone)
	transaction := make(chan struct{}, 1)
	transaction <- struct{}{}
	return &Updater{
		app:             app,
		dir:             dir,
		verifierFactory: update.NewSigstoreVerifier,
		persist:         persist,
		transaction:     transaction,
		workersDone:     workersDone,
	}
}

// Dir and SetDir round-trip the staged-update directory. SetDir exists for
// tests, and for internal/ui's startViewerRuntime, which fills in
// DefaultDir only when nothing set one first - the seam that lets a test
// viewer install a t.TempDir() ahead of it.
func (u *Updater) Dir() string       { return u.dir }
func (u *Updater) SetDir(dir string) { u.dir = dir }

// Client and SetClient round-trip the GitHub Releases client. nil until
// EnsureClient prepares one, or a test assigns one directly (httptest + a
// fake Verifier) to exercise Check/Download without hitting GitHub for real.
func (u *Updater) Client() *update.Client {
	u.clientMu.Lock()
	defer u.clientMu.Unlock()
	return u.client
}

func (u *Updater) SetClient(c *update.Client) {
	u.clientMu.Lock()
	defer u.clientMu.Unlock()
	u.client = c
}

// SetVerifierFactory replaces the lazy verifier constructor. It is a narrow
// test seam for preparation failures; nil restores the production factory.
func (u *Updater) SetVerifierFactory(factory func() (update.Verifier, error)) {
	u.clientMu.Lock()
	defer u.clientMu.Unlock()
	if factory == nil {
		factory = update.NewSigstoreVerifier
	}
	u.verifierFactory = factory
}

// EnsureClient lazily prepares the GitHub Releases client. Successful
// preparation is idempotent; a verifier-construction failure leaves the client
// nil so a later call can retry.
func (u *Updater) EnsureClient() error {
	u.clientMu.Lock()
	defer u.clientMu.Unlock()
	if u.client != nil {
		return nil
	}
	ver, err := u.verifierFactory()
	if err != nil {
		return err
	}
	u.client = update.NewClient(update.Config{
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		Now:      time.Now,
		Verify:   ver,
		StageDir: u.dir,
	})
	return nil
}

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
	if client := u.Client(); client != nil {
		return client.Now()
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

// Busy reports whether any update worker is still running. The automatic
// viewer entry point uses it to avoid waiting on a manual request's verifier
// preparation from the UI thread; the manual request already performs the
// same check and records the successful check day.
func (u *Updater) Busy() bool {
	u.workersMu.Lock()
	defer u.workersMu.Unlock()
	return u.workers != 0
}

// Settle waits until every update worker that is currently running has
// stopped, including superseded generations no longer named by Done.
func (u *Updater) Settle(ctx context.Context) error {
	for {
		u.workersMu.Lock()
		if u.workers == 0 {
			u.workersMu.Unlock()
			return nil
		}
		done := u.workersDone
		u.workersMu.Unlock()

		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// LastCheckDay and SetLastCheckDay round-trip the local calendar day
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
	select {
	case <-u.transaction:
		defer func() { u.transaction <- struct{}{} }()
		u.removeStaleStage(u.CurrentVersion())
	default:
		// A worker holding the transaction has already run the same cleanup
		// before its GitHub check. This public call runs from viewer startup
		// and preference callbacks, so it must not freeze the UI behind an
		// in-flight download or signature verification.
	}
}

func (u *Updater) removeStaleStage(currentVersion string) {
	st, err := update.LoadStage(u.dir)
	if err != nil {
		return
	}
	if !update.Newer(currentVersion, st.Version) {
		_ = update.RemoveStage(u.dir)
	}
}

// Events describes observable states of a manual update request. Callbacks
// run synchronously on the update worker; callers that own UI state must
// marshal them to their UI thread.
type Events struct {
	Downloading func(version string)
	Progress    func(update.DownloadProgress)
	Current     func()
	Ready       func(update.Stage)
	Failed      func(error)
}

// Start begins the background check/download's completion signal and runs it
// on its own goroutine. The caller must first prepare a client with EnsureClient
// or assign one directly via SetClient. Without one, Start returns an error
// without beginning the completion signal or starting a goroutine.
//
// ctx is the caller's per-request context (internal/ui's updateOp
// lifecycle token); stale reports whether that token has since been
// superseded, and is checked at every expensive-work and event boundary.
// currentVersion is the version to check against, resolved by
// the caller through CurrentVersion so gating decisions (is it due, is
// there an asset for this OS/arch) and the check itself agree on the same
// value.
func (u *Updater) Start(ctx context.Context, stale func() bool, currentVersion string) error {
	client := u.Client()
	if client == nil {
		return errClientNotPrepared
	}
	u.start(ctx, stale, currentVersion, client, Events{}, false)
	return nil
}

// StartManual starts a manually requested check. Unlike Start it prepares the
// verifier/client inside the tracked worker, so construction cannot block the
// caller's UI thread. Opt-in and daily-due policy belong to the automatic
// viewer entry point and are deliberately absent here.
func (u *Updater) StartManual(ctx context.Context, stale func() bool, currentVersion string, events Events) {
	u.start(ctx, stale, currentVersion, nil, events, true)
}

func (u *Updater) start(
	ctx context.Context,
	stale func() bool,
	currentVersion string,
	client *update.Client,
	events Events,
	prepare bool,
) {
	done := u.done.Begin()
	u.workerStarted()
	go func() {
		defer u.workerFinished()
		defer done()
		u.run(ctx, stale, currentVersion, client, events, prepare)
	}()
}

func (u *Updater) run(
	ctx context.Context,
	stale func() bool,
	currentVersion string,
	client *update.Client,
	events Events,
	prepare bool,
) {
	if requestStopped(ctx, stale) {
		return
	}
	if prepare {
		if err := u.EnsureClient(); err != nil {
			u.reportFailure(ctx, stale, events, "update verifier unavailable", err)
			return
		}
		client = u.Client()
	}
	if client == nil {
		u.reportFailure(ctx, stale, events, "update check failed", errClientNotPrepared)
		return
	}
	if requestStopped(ctx, stale) {
		return
	}

	select {
	case <-u.transaction:
		defer func() { u.transaction <- struct{}{} }()
	case <-ctx.Done():
		return
	}
	if requestStopped(ctx, stale) {
		return
	}

	u.removeStaleStage(currentVersion)
	if requestStopped(ctx, stale) {
		return
	}
	rel, err := client.Check(ctx, currentVersion)
	if err != nil {
		u.reportFailure(ctx, stale, events, "update check failed", err)
		return
	}
	if requestStopped(ctx, stale) {
		return
	}

	u.SetLastCheckDay(update.DayString(client.Now()))
	if rel == nil {
		u.emit(ctx, stale, events.Current)
		return
	}
	if st, ok := u.matchingUsableStage(*rel); ok {
		u.emitStage(ctx, stale, events.Ready, st)
		return
	}
	if requestStopped(ctx, stale) {
		return
	}
	u.emitVersion(ctx, stale, events.Downloading, rel.Version)
	if requestStopped(ctx, stale) {
		return
	}

	st, err := client.DownloadWithProgress(ctx, *rel, func(progress update.DownloadProgress) {
		if requestStopped(ctx, stale) || events.Progress == nil {
			return
		}
		events.Progress(progress)
	})
	if err != nil {
		u.reportFailure(ctx, stale, events, "update download failed", err)
		return
	}
	// DownloadWithProgress has fully verified, extracted, and written the
	// stage at this point. Keep it if this request became stale at the final
	// boundary; only its terminal UI event is suppressed.
	u.emitStage(ctx, stale, events.Ready, st)
}

func (u *Updater) matchingUsableStage(rel update.Release) (update.Stage, bool) {
	st, err := update.LoadStage(u.dir)
	if err != nil || !update.StageMatchesRelease(st, rel) || update.ValidateStageForPlatform(st, runtime.GOOS, runtime.GOARCH) != nil {
		return update.Stage{}, false
	}
	return st, true
}

func (u *Updater) reportFailure(ctx context.Context, stale func() bool, events Events, message string, err error) {
	fyne.LogError(message, err)
	u.emitError(ctx, stale, events.Failed, err)
}

func requestStopped(ctx context.Context, stale func() bool) bool {
	return ctx == nil || ctx.Err() != nil || stale == nil || stale()
}

func (u *Updater) emit(ctx context.Context, stale func() bool, callback func()) {
	if callback != nil && !requestStopped(ctx, stale) {
		callback()
	}
}

func (u *Updater) emitVersion(ctx context.Context, stale func() bool, callback func(string), version string) {
	if callback != nil && !requestStopped(ctx, stale) {
		callback(version)
	}
}

func (u *Updater) emitStage(ctx context.Context, stale func() bool, callback func(update.Stage), stage update.Stage) {
	if callback != nil && !requestStopped(ctx, stale) {
		callback(stage)
	}
}

func (u *Updater) emitError(ctx context.Context, stale func() bool, callback func(error), err error) {
	if callback != nil && !requestStopped(ctx, stale) {
		callback(err)
	}
}

func (u *Updater) workerStarted() {
	u.workersMu.Lock()
	defer u.workersMu.Unlock()
	if u.workers == 0 {
		u.workersDone = make(chan struct{})
	}
	u.workers++
}

func (u *Updater) workerFinished() {
	u.workersMu.Lock()
	defer u.workersMu.Unlock()
	u.workers--
	if u.workers == 0 {
		close(u.workersDone)
	}
}

// RequestApplyAndRelaunch validates that the shared update worker's staged
// output is still present, usable, and newer, then records explicit relaunch
// intent for ApplyStagedUpdate. It does not replace files or quit the app.
func (u *Updater) RequestApplyAndRelaunch() error {
	<-u.transaction
	defer func() { u.transaction <- struct{}{} }()

	st, err := update.LoadStage(u.dir)
	if err != nil {
		return fmt.Errorf("load staged update: %w", err)
	}
	if !update.Newer(u.CurrentVersion(), st.Version) {
		return fmt.Errorf("staged update %q is not newer than the current version", st.Version)
	}
	if err := update.ValidateStageForPlatform(st, runtime.GOOS, runtime.GOARCH); err != nil {
		return fmt.Errorf("staged update is not verified or usable: %w", err)
	}
	u.applyOptions.Relaunch = true
	return nil
}

// ApplyStagedUpdate replaces the running binary with a completed stage, if
// one exists and is still newer than CurrentVersion - called from
// internal/ui's shutdown handler, after the event loop has stopped taking
// input. Safe to call with nothing staged.
func (u *Updater) ApplyStagedUpdate() {
	<-u.transaction
	defer func() { u.transaction <- struct{}{} }()

	st, err := update.LoadStage(u.dir)
	if err != nil {
		return
	}
	if !update.Newer(u.CurrentVersion(), st.Version) {
		_ = update.RemoveStage(u.dir)
		return
	}
	if err := update.ValidateStageForPlatform(st, runtime.GOOS, runtime.GOARCH); err != nil {
		fyne.LogError("update apply skipped: staged update is not verified or usable", err)
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
	if err := update.Apply(st, dest, u.applyOptions); err != nil {
		fyne.LogError("failed to apply update", err)
		return
	}
	if runtime.GOOS != "windows" {
		_ = update.RemoveStage(u.dir)
	}
}
