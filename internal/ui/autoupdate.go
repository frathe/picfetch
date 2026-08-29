package ui

import (
	"errors"
	"runtime"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/autoupdate"
	"github.com/frathe/picfetch/internal/ui/settingswin"
	"github.com/frathe/picfetch/internal/update"
)

// CheckForUpdates and SetCheckForUpdates are the settings window's
// getter/setter pair for the opt-in updates preference. Turning the setting
// on starts a check when due; turning it off cancels an in-flight check but
// leaves an already-complete stage on disk for apply-on-stop.
func (v *viewer) CheckForUpdates() bool { return v.settings.checkForUpdates }

func (v *viewer) SetCheckForUpdates(on bool) {
	v.settings.checkForUpdates = on
	if on {
		v.maybeStartUpdateCheck()
		return
	}
	v.updateOp.invalidate()
}

// LastUpdateCheckDay and SetLastUpdateCheckDay round-trip the local
// calendar day (YYYY-MM-DD) of the last update check, or empty when none
// has run yet - storage and persistence live on v.updater, see
// internal/ui/autoupdate.Updater.
func (v *viewer) LastUpdateCheckDay() string { return v.updater.LastCheckDay() }

func (v *viewer) SetLastUpdateCheckDay(day string) { v.updater.SetLastCheckDay(day) }

// currentUpdateVersion is the version update checks compare releases
// against - see autoupdate.Updater.CurrentVersion for the test-override
// rule.
func (v *viewer) currentUpdateVersion() string { return v.updater.CurrentVersion() }

// maybeStartUpdateCheck is the viewer-side entry point into
// internal/ui/autoupdate: it removes any now-stale staged update, gates on
// the opt-in setting, a usable current version, today's due-ness, and
// asset availability for this OS/arch, then prepares the verifier/client
// before beginning updateOp's lifecycle token and handing Updater.Start its
// context and a staleness func.
func (v *viewer) maybeStartUpdateCheck() {
	v.updater.RemoveStaleStage()

	if !v.CheckForUpdates() {
		return
	}
	cur := v.currentUpdateVersion()
	if update.NormalizeVersion(cur) == "" {
		return
	}
	// A manual request already performs the same GitHub check, records the
	// successful day, and owns the shared stage transaction. Avoid waiting
	// on its background verifier preparation from this UI-thread entry point.
	if v.updater.Busy() {
		return
	}
	if !v.updater.Due(v.updater.Now()) {
		return
	}
	if _, ok := update.AssetName(runtime.GOOS, runtime.GOARCH); !ok {
		return
	}
	if err := v.updater.EnsureClient(); err != nil {
		fyne.LogError("update verifier unavailable", err)
		return
	}

	token := v.updateOp.begin()
	if err := v.updater.Start(token.context(), func() bool { return !token.current() }, cur); err != nil {
		fyne.LogError("update check failed to start", err)
	}
}

// CheckForUpdatesNow starts a manual update request. It deliberately bypasses
// the automatic-update preference and daily Due gate. Client/verifier
// preparation, stale-stage cleanup, the GitHub check, and any download all run
// on Updater's tracked worker; this entry point only validates cheap local
// prerequisites and adapts worker events onto Fyne's UI thread.
func (v *viewer) CheckForUpdatesNow(callbacks settingswin.UpdateCallbacks) {
	token := v.updateOp.begin()
	runOnUI := func(callback func()) {
		if callback == nil {
			return
		}
		fyne.Do(currentUpdateCallback(token, callback))
	}
	fail := func(err error) {
		runOnUI(func() {
			if callbacks.Failed != nil {
				callbacks.Failed(err)
			}
		})
	}

	cur := v.currentUpdateVersion()
	if update.NormalizeVersion(cur) == "" {
		err := errors.New("current update version is unavailable")
		fyne.LogError("update check failed", err)
		if callbacks.Failed != nil && token.current() {
			callbacks.Failed(err)
		}
		return
	}
	if _, ok := update.AssetName(runtime.GOOS, runtime.GOARCH); !ok {
		err := errors.New("updates are unavailable for this platform")
		fyne.LogError("update check failed", err)
		if callbacks.Failed != nil && token.current() {
			callbacks.Failed(err)
		}
		return
	}

	v.updater.StartManual(token.context(), func() bool { return !token.current() }, cur, autoupdate.Events{
		Downloading: func(version string) {
			runOnUI(func() {
				if callbacks.Downloading != nil {
					callbacks.Downloading(version)
				}
			})
		},
		Progress: func(progress update.DownloadProgress) {
			runOnUI(func() {
				if callbacks.Progress != nil {
					callbacks.Progress(progress.Downloaded, progress.Total)
				}
			})
		},
		Current: func() {
			runOnUI(callbacks.Current)
		},
		Ready: func(stage update.Stage) {
			runOnUI(func() {
				if callbacks.Ready != nil {
					callbacks.Ready(update.NormalizeVersion(stage.Version))
				}
			})
		},
		Failed: fail,
	})
}

// PerformUpdate accepts the disruptive Settings action only while a newer,
// usable staged update still exists. The actual file replacement remains in
// Run's SetOnStopped callback, after session and preference persistence.
func (v *viewer) PerformUpdate() error {
	if err := v.updater.RequestApplyAndRelaunch(); err != nil {
		return err
	}
	v.quit()
	return nil
}

// currentUpdateCallback is the second staleness boundary for an update event:
// a token can be superseded after its worker queues a fyne.Do closure but
// before the real UI driver runs it. Kept as a named closure builder so tests
// can exercise that delayed-driver ordering even though Fyne's test driver
// executes fyne.Do inline.
func currentUpdateCallback(token requestToken, callback func()) func() {
	return func() {
		if token.current() {
			callback()
		}
	}
}

// maybeShowWhatsNew opens the release-notes window once after an applied
// update, when the cached version matches this build. The cache is cleared
// before show so a later launch does not reopen it. newTestUI does not run
// SetOnStarted; tests call this directly.
func (v *viewer) maybeShowWhatsNew() {
	wn, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil || wn == nil || wn.Version == "" {
		return
	}
	cur := update.NormalizeVersion(v.currentUpdateVersion())
	if cur == "" || update.NormalizeVersion(wn.Version) != cur {
		return
	}
	_ = autoupdate.ClearWhatsNew(v.app)
	v.help.ShowWhatsNew(wn.Version, wn.Body)
}
