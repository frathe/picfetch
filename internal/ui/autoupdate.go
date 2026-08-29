package ui

import (
	"runtime"

	"github.com/frathe/picfetch/internal/ui/autoupdate"
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
// asset availability for this OS/arch, then begins updateOp's lifecycle
// token and hands Updater.Start its context and a staleness func.
func (v *viewer) maybeStartUpdateCheck() {
	v.updater.RemoveStaleStage()

	if !v.CheckForUpdates() {
		return
	}
	cur := v.currentUpdateVersion()
	if update.NormalizeVersion(cur) == "" {
		return
	}
	if !v.updater.Due(v.updater.Now()) {
		return
	}
	if _, ok := update.AssetName(runtime.GOOS, runtime.GOARCH); !ok {
		return
	}

	token := v.updateOp.begin()
	v.updater.Start(token.context(), func() bool { return !token.current() }, cur)
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
