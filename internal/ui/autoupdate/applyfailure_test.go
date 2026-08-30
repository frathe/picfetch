package autoupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/update"
)

func sampleApplyFailure() ApplyFailure {
	return ApplyFailure{
		Version: "v0.2.6",
		Reason:  string(update.ReasonAccessDenied),
		Op:      "copy",
		Path:    `C:\Program Files\PicFetch\picfetch.exe`,
		Detail:  "update apply: copy C:\\Program Files\\PicFetch\\picfetch.exe: permission denied",
	}
}

func TestApplyFailureCache_RoundTrip(t *testing.T) {
	app := test.NewApp()
	want := sampleApplyFailure()

	if err := SaveApplyFailure(app, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("LoadApplyFailure returned nil after a save")
	}
	if *got != want {
		t.Errorf("LoadApplyFailure() = %+v, want %+v", *got, want)
	}
}

func TestLoadApplyFailure_NothingCachedIsNilWithoutError(t *testing.T) {
	got, err := LoadApplyFailure(test.NewApp())
	if err != nil {
		t.Fatalf("LoadApplyFailure with an empty cache: %v", err)
	}
	if got != nil {
		t.Errorf("LoadApplyFailure() = %+v, want nil", got)
	}
}

func TestClearApplyFailure_RemovesTheRecord(t *testing.T) {
	app := test.NewApp()
	if err := SaveApplyFailure(app, sampleApplyFailure()); err != nil {
		t.Fatal(err)
	}

	if err := ClearApplyFailure(app); err != nil {
		t.Fatal(err)
	}

	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("record survived Clear: %+v", got)
	}
}

func TestClearApplyFailure_NothingCachedIsANoop(t *testing.T) {
	if err := ClearApplyFailure(test.NewApp()); err != nil {
		t.Errorf("ClearApplyFailure with an empty cache: %v", err)
	}
}

func TestSaveApplyFailure_ReplacesTheEarlierRecord(t *testing.T) {
	// Only the most recent attempt is worth reporting; a stale reason would
	// send the user after a problem that no longer exists.
	app := test.NewApp()
	if err := SaveApplyFailure(app, sampleApplyFailure()); err != nil {
		t.Fatal(err)
	}
	second := ApplyFailure{Version: "v0.2.7", Reason: string(update.ReasonVirusBlocked), Op: "rename"}
	if err := SaveApplyFailure(app, second); err != nil {
		t.Fatal(err)
	}

	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != second {
		t.Errorf("LoadApplyFailure() = %+v, want %+v", got, second)
	}
}

func TestSaveApplyFailure_UsesStableJSONKeys(t *testing.T) {
	// The record is written by one build of PicFetch and read by the next
	// one, so the wire names are part of the contract, not an implementation
	// detail either side may rename.
	app := test.NewApp()
	if err := SaveApplyFailure(app, sampleApplyFailure()); err != nil {
		t.Fatal(err)
	}

	r, err := app.Cache().Read(ApplyFailureCacheKey)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	var raw map[string]any
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	want := sampleApplyFailure()
	for key, value := range map[string]string{
		"version": want.Version,
		"reason":  want.Reason,
		"op":      want.Op,
		"path":    want.Path,
		"detail":  want.Detail,
	} {
		if raw[key] != value {
			t.Errorf("cached %q = %v, want %q", key, raw[key], value)
		}
	}
}

func TestLoadApplyFailure_UnreadableRecordIsAnError(t *testing.T) {
	// The caller has to be able to tell "nothing failed" from "the record is
	// there but unreadable": only the first one may sweep the backup.
	app := test.NewApp()
	w, err := app.Cache().Write(ApplyFailureCacheKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("{not json")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := LoadApplyFailure(app)
	if err == nil {
		t.Fatalf("LoadApplyFailure() = %+v, want an error for a corrupt record", got)
	}
	if got != nil {
		t.Errorf("LoadApplyFailure() = %+v alongside an error, want nil", got)
	}
}

// --- ApplyStagedUpdate's use of the record -------------------------------

// verifiedStage downloads a real, provenance-verified stage into u's
// directory, the only shape ApplyStagedUpdate will act on.
func verifiedStage(t *testing.T, u *Updater, version string) update.Stage {
	t.Helper()
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	srv := updaterReleaseServer(t, version, asset, archive, hex.EncodeToString(sum[:]), nil)
	client := updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local))
	rel, err := client.Check(context.Background(), "v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("Check found no release to download")
	}
	st, err := client.Download(context.Background(), *rel)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func stubApply(t *testing.T, err error) {
	t.Helper()
	orig := update.Apply
	update.Apply = func(update.Stage, string, update.ApplyOptions) error { return err }
	t.Cleanup(func() { update.Apply = orig })
}

func stagePresent(t *testing.T, dir string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, "stage.json"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

func newApplyUpdater(t *testing.T) (*Updater, fyne.App) {
	t.Helper()
	app := test.NewApp()
	u := New(app, t.TempDir(), nil)
	u.SetCurrentVersion("0.2.5")
	verifiedStage(t, u, "v0.2.6")
	return u, app
}

func TestApplyStagedUpdate_RecordsWhyTheSwapFailed(t *testing.T) {
	u, app := newApplyUpdater(t)
	applyErr := &update.ApplyError{Op: "copy", Path: "/Applications/PicFetch.app", Err: fs.ErrPermission}
	stubApply(t, applyErr)

	u.ApplyStagedUpdate()

	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("a failed apply recorded nothing for the next launch to report")
	}
	if got.Version != "v0.2.6" {
		t.Errorf("Version = %q, want v0.2.6", got.Version)
	}
	if got.Reason != string(update.ReasonAccessDenied) {
		t.Errorf("Reason = %q, want %q", got.Reason, update.ReasonAccessDenied)
	}
	if got.Op != "copy" {
		t.Errorf("Op = %q, want copy", got.Op)
	}
	if got.Path == "" {
		t.Error("Path is empty; the report cannot name the executable it could not replace")
	}
	if got.Detail != applyErr.Error() {
		t.Errorf("Detail = %q, want %q", got.Detail, applyErr.Error())
	}
}

func TestApplyStagedUpdate_FailureKeepsTheStageForARetry(t *testing.T) {
	u, _ := newApplyUpdater(t)
	stubApply(t, &update.ApplyError{Op: "copy", Err: fs.ErrPermission})

	u.ApplyStagedUpdate()

	if !stagePresent(t, u.Dir()) {
		t.Error("a denied write threw the download away; a later launch cannot retry it")
	}
}

func TestApplyStagedUpdate_PlainErrorIsRecordedAsUnknown(t *testing.T) {
	// Nothing outside Apply constructs an ApplyError, so a bare error still
	// has to produce a usable record rather than none at all.
	u, app := newApplyUpdater(t)
	stubApply(t, errors.New("boom"))

	u.ApplyStagedUpdate()

	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("a failed apply recorded nothing")
	}
	if got.Reason != string(update.ReasonUnknown) {
		t.Errorf("Reason = %q, want %q", got.Reason, update.ReasonUnknown)
	}
	if got.Op != "" {
		t.Errorf("Op = %q, want empty for an error that names no step", got.Op)
	}
	if got.Detail != "boom" {
		t.Errorf("Detail = %q, want boom", got.Detail)
	}
}

// TestApplyStagedUpdate_AFailedRelaunchIsNotAFailedInstall covers the one
// ApplyError swapBinary raises after the update is already on disk: the copy
// and the SHA-256 verify both passed, only starting the successor failed.
// Recording it would greet the user - on the new build - with "PicFetch
// could not install the update" and "The previous version is still installed
// and running", both false, stacked on top of the What's New dialog for the
// version the other dialog denies.
func TestApplyStagedUpdate_AFailedRelaunchIsNotAFailedInstall(t *testing.T) {
	u, app := newApplyUpdater(t)
	stubApply(t, &update.ApplyError{Op: "relaunch", Path: "/Applications/PicFetch.app", Err: errors.New("fork/exec: no such file")})

	u.ApplyStagedUpdate()

	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("recorded %+v for a relaunch that failed after the install succeeded", got)
	}
	if stagePresent(t, u.Dir()) {
		t.Error("the stage survived an update that was installed; the next shutdown would apply it again")
	}
}

// TestApplyStagedUpdate_AFailedRelaunchClearsAnEarlierRecord pins the other
// half: the install succeeded, so a record from a previous attempt must not
// outlive it and veto the backup sweep.
func TestApplyStagedUpdate_AFailedRelaunchClearsAnEarlierRecord(t *testing.T) {
	u, app := newApplyUpdater(t)
	if err := SaveApplyFailure(app, ApplyFailure{Version: "v0.2.6", Op: "restore"}); err != nil {
		t.Fatal(err)
	}
	stubApply(t, &update.ApplyError{Op: "relaunch", Err: errors.New("no")})

	u.ApplyStagedUpdate()

	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("record after an installed update = %+v, want nil", got)
	}
}

func TestApplyStagedUpdate_SuccessRemovesTheStageOnEveryPlatform(t *testing.T) {
	// Windows used to keep the stage because the cmd.exe script deleted the
	// staged file itself, after Apply had already returned. The in-process
	// swap finishes first, so there is no platform left that needs it.
	u, app := newApplyUpdater(t)
	// Seeded from an earlier attempt, and the worst one to leave behind: a
	// stale "restore" vetoes the backup sweep on every later launch and
	// reports a failure for an update that worked.
	if err := SaveApplyFailure(app, ApplyFailure{Version: "v0.2.6", Op: "restore"}); err != nil {
		t.Fatal(err)
	}
	stubApply(t, nil)

	u.ApplyStagedUpdate()

	if stagePresent(t, u.Dir()) {
		t.Error("an applied stage was left on disk")
	}
	got, err := LoadApplyFailure(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("a successful apply left a failure record behind: %+v", got)
	}
}
