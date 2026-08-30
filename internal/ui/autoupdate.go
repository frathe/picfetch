package ui

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

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

// sweepUpdateBackup deletes the executable an applied update renamed aside,
// and hands back the recorded apply failure it had to read to decide - nil
// when the last apply worked or left nothing readable behind. Returning the
// record is what puts the failure report downstream of the sweep: reporting
// clears the record, and a cleared record reads as a clean install, so a
// reporter that ran first would let the sweep take the last working binary.
//
// It runs at startup rather than in update.CleanupPredecessor because the
// decision needs that record, and it lives in v.app's cache - which does not
// exist yet at the point CleanupPredecessor has to run.
func (v *viewer) sweepUpdateBackup() *autoupdate.ApplyFailure {
	dest, err := os.Executable()
	if err != nil {
		return nil
	}
	return v.sweepUpdateBackupAt(dest)
}

// sweepUpdateBackupAt is sweepUpdateBackup over an explicit path, so the
// record-versus-backup decision can be exercised without writing into the
// directory the test binary itself runs from.
func (v *viewer) sweepUpdateBackupAt(dest string) *autoupdate.ApplyFailure {
	failure, err := autoupdate.LoadApplyFailure(v.app)
	if err != nil {
		// An unreadable record might have said "restore", and keeping a
		// stale backup costs disk while deleting the wrong one costs the
		// application. Nothing to report either.
		return nil
	}
	if failure != nil && failure.Op == "restore" {
		// A failed restore left neither binary in place, so the backup is
		// the only PicFetch known to work.
		return failure
	}
	update.SweepBackup(dest)
	return failure
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

// maybeShowUpdateFailure explains a failed binary replacement on the launch
// after it happened: ApplyStagedUpdate runs from Fyne's stopped callback,
// where no window is left to report into, so the reason is cached and
// surfaced here. rec comes from sweepUpdateBackup rather than from a second
// cache read - see its doc comment for why reporting has to stay downstream
// of the sweep.
//
// Deliberately not version-gated the way maybeShowWhatsNew is: a suppressed
// dialog would never reach the clear below, and a stranded record with Op
// "restore" vetoes the backup sweep on every later launch.
func (v *viewer) maybeShowUpdateFailure(rec *autoupdate.ApplyFailure) {
	if rec == nil {
		return
	}
	// Detail is the raw OS error text, and this is the only place it is
	// ever shown: it stays out of the dialog because an errno-flavoured
	// sentence explains nothing and cannot be translated, and the process
	// that first logged it is gone by now, so this launch's log is the one
	// the user still has.
	fyne.LogError(fmt.Sprintf("update %s was not installed (%s during %q at %s): %s",
		rec.Version, rec.Reason, rec.Op, rec.Path, rec.Detail), nil)
	_ = autoupdate.ClearApplyFailure(v.app)

	dialog.NewCustomConfirm(
		lang.L("Update could not be installed"),
		lang.L("Open download page"),
		lang.L("Close"),
		updateFailureContent(updateFailureMessage(*rec, runtime.GOOS), updateFailureBodySize(v.win.Canvas().Size())),
		func(open bool) {
			if open {
				v.openReleasesPage()
			}
		},
		v.win,
	).Show()
}

// The scrolled message is sized from the canvas it will be shown on, not
// from a fixed box: a modal popup is clamped to its canvas, so anything the
// dialog cannot fit is cut off rather than made reachable, while a fixed box
// small enough for the smallest window leaves a maximised one scrolling
// through a message that had room to be read in full.
//
// updateFailureChromeW and updateFailureChromeH are what the dialog adds
// around the body - title, padding and the button row - measured from
// dialog.MinSize, so canvas minus chrome is the largest body that still fits.
// The maxima stop a full-screen window from stretching one paragraph across
// the display; they are set so the longest wording this dialog carries (the
// German Controlled Folder Access text with a deep install path) is fully
// visible without scrolling. The floor keeps a body worth scrolling when the
// user has shrunk the window below even the startW x startH default.
const (
	updateFailureChromeW  = 32
	updateFailureChromeH  = 108
	updateFailureBodyMaxW = 720
	updateFailureBodyMaxH = 340
	updateFailureBodyMinW = 240
	updateFailureBodyMinH = 120
)

// updateFailureBodySize is the body size for a dialog shown on a canvas of
// this size.
func updateFailureBodySize(canvas fyne.Size) fyne.Size {
	return fyne.NewSize(
		max(updateFailureBodyMinW, min(canvas.Width-updateFailureChromeW, updateFailureBodyMaxW)),
		max(updateFailureBodyMinH, min(canvas.Height-updateFailureChromeH, updateFailureBodyMaxH)),
	)
}

// updateFailureContent lays the explanation out so all of it stays reachable
// at the given body size. Vertical scrolling only: the label wraps to
// whatever width the viewport gives it, so a horizontal bar would have
// nothing to reveal. At the sizes a normal window yields there is nothing
// left to scroll - the scroll is what keeps the tail reachable when the
// window is too small to show the whole message at once.
func updateFailureContent(message string, body fyne.Size) *container.Scroll {
	scroll := container.NewVScroll(&widget.Label{Text: message, Wrapping: fyne.TextWrapWord})
	scroll.SetMinSize(body)
	return scroll
}

// updateFailureMessage picks the explanation for a recorded reason, for the
// platform named by goos. Split out so the wording is testable without
// opening a dialog.
//
// goos is a parameter rather than a read of runtime.GOOS because the
// Controlled Folder Access arm is the only wording in this app that is true
// on one platform and false on the others, and reading the constant here
// would leave whichever arm the test host does not match with no executed
// coverage - CI is Linux, so that would be the arm this whole feature exists
// for. Reading the OS at report time rather than storing it in the record is
// enough: a record travels from one launch of an installed PicFetch to the
// next on the same machine, so the build that writes it is the build that
// reads it.
func updateFailureMessage(f autoupdate.ApplyFailure, goos string) string {
	generic := fmt.Sprintf(lang.L("PicFetch could not install the update at %s."), f.Path)

	var reason string
	switch update.FailureReason(f.Reason) {
	case update.ReasonAccessDenied:
		// Only Windows has Controlled Folder Access. Elsewhere the same
		// classification means nothing more than an install directory the
		// user cannot write to, and naming a Windows feature would send them
		// hunting for a setting their system does not have.
		reason = generic
		if goos == "windows" {
			reason = fmt.Sprintf(lang.L("Windows blocked PicFetch from replacing itself at %s. This is Controlled Folder Access (\"Überwachter Ordnerzugriff\") protecting that folder. Allow PicFetch through it in Windows Security -> Virus & threat protection -> Ransomware protection -> Allow an app through Controlled folder access, or move PicFetch to a folder outside Documents, Pictures, Music, Videos and Desktop."), f.Path)
		}
	case update.ReasonVirusBlocked:
		reason = lang.L("Your antivirus removed or quarantined the downloaded update.")
	case update.ReasonSharingViolation:
		reason = fmt.Sprintf(lang.L("PicFetch could not replace itself at %s because the file was in use."), f.Path)
	default:
		reason = generic
	}
	return reason + "\n\n" + lang.L("The previous version is still installed and running.")
}

// openReleasesPage sends the user to the download page - the manual way out
// of an update PicFetch could not install for itself. Split off the dialog's
// callback so the branch runs in a test at all; the confirm button is
// otherwise unreachable from this suite.
func (v *viewer) openReleasesPage() {
	u, err := url.Parse(update.DownloadPageURL)
	if err != nil {
		fyne.LogError("the releases page URL is unusable", err)
		return
	}
	if err := v.app.OpenURL(u); err != nil {
		fyne.LogError("failed to open the download page", err)
	}
}
