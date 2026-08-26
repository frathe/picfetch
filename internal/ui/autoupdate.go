package ui

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/update"
)

const whatsNewCacheKey = "whatsnew.json"

type whatsNewCache struct {
	Version string `json:"version"`
	Body    string `json:"body"`
}

func defaultUpdateDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "picfetch", "updates")
}

func (v *viewer) currentUpdateVersion() string {
	if v.updateCurrentVersion != "" {
		return v.updateCurrentVersion
	}
	return v.app.Metadata().Version
}

// CheckForUpdates/SetCheckForUpdates are the settings window's getter/setter
// pair for the opt-in updates preference. Turning the setting on starts a
// check when due; turning it off cancels an in-flight check but leaves an
// already-complete stage on disk for apply-on-stop.
func (v *viewer) CheckForUpdates() bool { return v.settings.checkForUpdates }

func (v *viewer) SetCheckForUpdates(on bool) {
	v.settings.checkForUpdates = on
	if on {
		v.maybeStartUpdateCheck()
		return
	}
	v.updateOp.lifecycle.invalidate()
}

// LastUpdateCheckDay/SetLastUpdateCheckDay round-trip the local calendar
// day (YYYY-MM-DD) of the last update check, or empty when none has run
// yet. The setter also persists via SaveLastUpdateCheckDay so a crash or
// quit during a background check still records today without OnStopped
// waiting for the check goroutine.
func (v *viewer) LastUpdateCheckDay() string {
	v.updateDayMu.Lock()
	defer v.updateDayMu.Unlock()
	return v.settings.lastUpdateCheckDay
}

func (v *viewer) SetLastUpdateCheckDay(day string) {
	v.updateDayMu.Lock()
	defer v.updateDayMu.Unlock()
	v.settings.lastUpdateCheckDay = day
	if v.app != nil {
		preferences.SaveLastUpdateCheckDay(v.app, day)
	}
}

func (v *viewer) removeStaleUpdateStage() {
	st, err := update.LoadStage(v.updateDir)
	if err != nil {
		return
	}
	if !update.Newer(v.currentUpdateVersion(), st.Version) {
		_ = update.RemoveStage(v.updateDir)
	}
}

func (v *viewer) updateNow() time.Time {
	if v.update != nil {
		return v.update.Now()
	}
	return time.Now()
}

func (v *viewer) maybeStartUpdateCheck() {
	v.removeStaleUpdateStage()

	if !v.CheckForUpdates() {
		return
	}
	if update.NormalizeVersion(v.currentUpdateVersion()) == "" {
		return
	}
	if !update.Due(v.LastUpdateCheckDay(), v.updateNow()) {
		return
	}
	if _, ok := update.AssetName(runtime.GOOS, runtime.GOARCH); !ok {
		return
	}
	if v.update == nil {
		ver, err := update.NewSigstoreVerifier()
		if err != nil {
			fyne.LogError("update verifier unavailable", err)
			return
		}
		v.update = update.NewClient(update.Config{
			HTTP:     &http.Client{Timeout: 30 * time.Second},
			Now:      time.Now,
			Verify:   ver,
			StageDir: v.updateDir,
		})
	}

	token := v.updateOp.lifecycle.begin()
	done := v.updateDone.Begin()
	go func() {
		defer done()
		rel, err := v.update.Check(token.context(), v.currentUpdateVersion())
		if err != nil {
			fyne.LogError("update check failed", err)
			return
		}
		if !token.current() {
			return
		}
		day := update.DayString(v.update.Now())
		v.SetLastUpdateCheckDay(day)
		if rel == nil {
			return
		}
		_, err = v.update.Download(token.context(), *rel)
		if err != nil {
			fyne.LogError("update download failed", err)
			return
		}
		// Finished stage is kept even if the token was cancelled mid-download
		// (SetCheckForUpdates(false)): apply-on-stop still needs the bits.
	}()
}

func (v *viewer) applyStagedUpdate() {
	st, err := update.LoadStage(v.updateDir)
	if err != nil {
		return
	}
	if !update.Newer(v.currentUpdateVersion(), st.Version) {
		_ = update.RemoveStage(v.updateDir)
		return
	}
	if err := saveWhatsNew(v.app, st.Version, st.Notes); err != nil {
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
		_ = update.RemoveStage(v.updateDir)
	}
}

func saveWhatsNew(app fyne.App, version, body string) error {
	w, err := app.Cache().Write(whatsNewCacheKey)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(whatsNewCache{Version: version, Body: body}); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func loadWhatsNew(app fyne.App) (*whatsNewCache, error) {
	cache := app.Cache()
	if !cache.Exists(whatsNewCacheKey) {
		return nil, nil
	}
	r, err := cache.Read(whatsNewCacheKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	var s whatsNewCache
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func clearWhatsNew(app fyne.App) error {
	cache := app.Cache()
	if !cache.Exists(whatsNewCacheKey) {
		return nil
	}
	return cache.Remove(whatsNewCacheKey)
}

// maybeShowWhatsNew opens the release-notes window once after an applied
// update, when the cached version matches this build. The cache is cleared
// before show so a later launch does not reopen it. newTestUI does not run
// SetOnStarted; tests call this directly.
func (v *viewer) maybeShowWhatsNew() {
	wn, err := loadWhatsNew(v.app)
	if err != nil || wn == nil || wn.Version == "" {
		return
	}
	cur := update.NormalizeVersion(v.currentUpdateVersion())
	if cur == "" || update.NormalizeVersion(wn.Version) != cur {
		return
	}
	_ = clearWhatsNew(v.app)
	v.help.ShowWhatsNew(wn.Version, wn.Body)
}
