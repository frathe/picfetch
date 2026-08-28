package autoupdate

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/update"
)

// --- Due / not-due -----------------------------------------------------

func TestUpdater_Due(t *testing.T) {
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		lastDay string
		want    bool
	}{
		{"never checked is due", "", true},
		{"same local calendar day is not due", "2026-08-26", false},
		{"previous local calendar day is due", "2026-08-25", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := New(test.NewApp(), t.TempDir(), nil)
			if tc.lastDay != "" {
				u.SetLastCheckDay(tc.lastDay)
			}
			if got := u.Due(now); got != tc.want {
				t.Errorf("Due(%v) with lastCheckDay %q = %v, want %v", now, tc.lastDay, got, tc.want)
			}
		})
	}
}

func TestUpdater_LastCheckDayZeroValueIsEmpty(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	if got := u.LastCheckDay(); got != "" {
		t.Errorf("LastCheckDay() on a fresh Updater = %q, want empty", got)
	}
}

// --- LastCheckDay persistence seam --------------------------------------

func TestUpdater_SetLastCheckDayCallsPersist(t *testing.T) {
	var got []string
	var mu sync.Mutex
	u := New(test.NewApp(), t.TempDir(), func(day string) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, day)
	})

	u.SetLastCheckDay("2026-08-26")
	u.SetLastCheckDay("2026-08-27")

	mu.Lock()
	defer mu.Unlock()
	want := []string{"2026-08-26", "2026-08-27"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("persisted days = %v, want %v", got, want)
	}
	if u.LastCheckDay() != "2026-08-27" {
		t.Errorf("LastCheckDay() = %q, want 2026-08-27", u.LastCheckDay())
	}
}

// TestUpdater_NilPersistIsSafe pins that persist is an optional seam: a
// nil func (the zero value, and what most package-level unit tests here
// pass) must not panic SetLastCheckDay.
func TestUpdater_NilPersistIsSafe(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetLastCheckDay("2026-08-26")
	if got := u.LastCheckDay(); got != "2026-08-26" {
		t.Errorf("LastCheckDay() = %q, want 2026-08-26", got)
	}
}

// TestUpdater_LastCheckDayConcurrent proves dayMu actually guards
// lastCheckDay end to end through the exported pair, the same invariant
// internal/ui's TestLastUpdateCheckDay_ConcurrentWithCurrentPreferences
// pins at the viewer layer.
func TestUpdater_LastCheckDayConcurrent(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	const n = 200
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range n {
			u.SetLastCheckDay("2026-08-26")
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			_ = u.LastCheckDay()
		}
	}()
	wg.Wait()
}

// --- stale-stage removal -------------------------------------------------

func TestUpdater_RemoveStaleStage_NoStagePresentIsNoop(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage() // must not panic

	if _, err := update.LoadStage(u.Dir()); err == nil {
		t.Fatal("expected no stage to load")
	}
}

func TestUpdater_RemoveStaleStage_OlderStagedVersionIsRemoved(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.5", Notes: "stale"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage()

	if _, err := update.LoadStage(dir); err == nil {
		t.Error("stale stage must be removed")
	}
}

func TestUpdater_RemoveStaleStage_SameStagedVersionIsRemoved(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.6", Notes: "same"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage()

	if _, err := update.LoadStage(dir); err == nil {
		t.Error("a stage matching the current version must be removed too - Newer is strict")
	}
}

func TestUpdater_RemoveStaleStage_NewerStagedVersionIsKept(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.7", Notes: "fresh"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage()

	st, err := update.LoadStage(dir)
	if err != nil {
		t.Fatalf("expected the still-relevant stage to survive, got: %v", err)
	}
	if st.Version != "v0.2.7" {
		t.Errorf("LoadStage().Version = %q, want v0.2.7", st.Version)
	}
}

// --- version-normalisation gates -----------------------------------------

func TestUpdater_CurrentVersion_OverrideWins(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetCurrentVersion("0.2.5")

	if got := u.CurrentVersion(); got != "0.2.5" {
		t.Errorf("CurrentVersion() = %q, want 0.2.5", got)
	}
	if got := update.NormalizeVersion(u.CurrentVersion()); got != "v0.2.5" {
		t.Errorf("NormalizeVersion(CurrentVersion()) = %q, want v0.2.5", got)
	}
}

// TestUpdater_CurrentVersion_FallsBackToAppMetadata pins the gate
// maybeStartUpdateCheck relies on: Fyne's test app ships an empty
// Metadata().Version, so an Updater with no override normalises to "" and
// every caller must treat that as "no usable version, do not check".
func TestUpdater_CurrentVersion_FallsBackToAppMetadata(t *testing.T) {
	app := test.NewApp()
	u := New(app, t.TempDir(), nil)

	if got := u.CurrentVersion(); got != app.Metadata().Version {
		t.Errorf("CurrentVersion() = %q, want app.Metadata().Version %q", got, app.Metadata().Version)
	}
	if got := update.NormalizeVersion(u.CurrentVersion()); got != "" {
		t.Errorf("NormalizeVersion(CurrentVersion()) = %q, want empty (test app metadata is unset)", got)
	}
}

func TestUpdater_CurrentVersion_EmptyOverrideDoesNotStick(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetCurrentVersion("0.2.5")
	u.SetCurrentVersion("") // an empty override falls back, same as never set

	if got := u.CurrentVersion(); got != "" {
		t.Errorf("CurrentVersion() = %q, want empty (Fyne test app metadata)", got)
	}
}

// --- Dir / Client / Done round trips --------------------------------------

func TestUpdater_DirRoundTrip(t *testing.T) {
	u := New(test.NewApp(), "", nil)
	if got := u.Dir(); got != "" {
		t.Fatalf("Dir() = %q, want empty before SetDir", got)
	}

	dir := filepath.Join(t.TempDir(), "updates")
	u.SetDir(dir)
	if got := u.Dir(); got != dir {
		t.Errorf("Dir() = %q, want %q", got, dir)
	}
}

func TestUpdater_ClientRoundTrip(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	if u.Client() != nil {
		t.Fatal("Client() on a fresh Updater must be nil")
	}

	c := update.NewClient(update.Config{StageDir: u.Dir()})
	u.SetClient(c)
	if u.Client() != c {
		t.Error("Client() must return exactly what SetClient assigned")
	}
}

func TestUpdater_Now_FallsBackToRealClockWithoutClient(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)

	before := time.Now()
	got := u.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestUpdater_Now_UsesClientClockOnceOneExists(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	fixed := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	u.SetClient(update.NewClient(update.Config{
		StageDir: u.Dir(),
		Now:      func() time.Time { return fixed },
	}))

	if got := u.Now(); !got.Equal(fixed) {
		t.Errorf("Now() = %v, want %v", got, fixed)
	}
}

func TestUpdater_Done_StartsNotBegun(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	if u.Done().Begun() {
		t.Error("Done() on a fresh Updater must not report Begun")
	}
}

func TestDefaultDir_EndsInPicfetchUpdates(t *testing.T) {
	got := DefaultDir()
	want := filepath.Join("picfetch", "updates")
	if filepath.Base(filepath.Dir(got)) != "picfetch" || filepath.Base(got) != "updates" {
		t.Errorf("DefaultDir() = %q, want it to end in %q", got, want)
	}
}
