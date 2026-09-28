package favorites

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFavoriteStorageReads(t *testing.T) {
	t.Run("held_refresh_coalesces", func(t *testing.T) {
		f := newFeature(t, &fakeHost{})
		saveFavorites(t, f, "Original")
		original := f.menu.Items[2]
		started, resume := make(chan struct{}), make(chan struct{})
		var release sync.Once
		unblock := func() { release.Do(func() { close(resume) }) }
		t.Cleanup(unblock)
		var calls atomic.Int32
		f.SetStorage(&controlledStorage{Store: &favstore.Store{}, list: func(ctx context.Context, dir string) ([]favstore.Entry, error) {
			if calls.Add(1) == 1 {
				close(started)
				<-resume
			}
			return (&favstore.Store{}).List(ctx, dir)
		}})
		returned := make(chan struct{})
		go func() { f.refreshMenu(); close(returned) }()
		waitStorage(t, started)
		waitStorage(t, returned)
		f.SetDir(t.TempDir())
		latest := t.TempDir()
		if err := favstore.Save(latest, "Latest", nil); err != nil {
			t.Fatal(err)
		}
		f.SetDir(latest)
		if f.menu.Items[2] != original || !slices.Equal(f.names, []string{"Original"}) {
			t.Fatal("refresh replaced the last complete menu before delivery")
		}
		unblock()
		f.Settle()
		if calls.Load() != 2 || !slices.Equal(f.names, []string{"Latest"}) {
			t.Fatalf("coalescing: calls=%d names=%v", calls.Load(), f.names)
		}
	})
	t.Run("held_replacement", func(t *testing.T) {
		host := &fakeHost{}
		f := newFeature(t, host)
		files := []fyne.URI{storage.NewFileURI("/offline.jpg")}
		if err := favstore.Save(f.dir, "Trip", files); err != nil {
			t.Fatal(err)
		}
		started, resume := make(chan struct{}), make(chan struct{})
		var release sync.Once
		unblock := func() { release.Do(func() { close(resume) }) }
		t.Cleanup(unblock)
		f.SetStorage(&controlledStorage{Store: &favstore.Store{}, open: func(ctx context.Context, dir string) (favstore.Definition, error) {
			definition, err := favstore.Open(ctx, dir)
			close(started)
			<-resume
			return definition, err
		}})
		returned := make(chan struct{})
		go func() { f.openFavorite("Trip"); close(returned) }()
		waitStorage(t, started)
		waitStorage(t, returned)
		if err := favstore.Save(f.dir, "Trip", files); err != nil {
			t.Fatal(err)
		}
		unblock()
		f.Settle()
		if len(host.opened) != 0 || len(host.syncedDirs) != 0 {
			t.Fatal("held opening adopted an identical replacement owner")
		}
	})
	t.Run("queued_refresh", func(t *testing.T) {
		f := newFeature(t, &fakeHost{})
		f.SetUIQueue(&uitest.UIQueue{})
		if err := favstore.Save(f.dir, "Trip", nil); err != nil {
			t.Fatal(err)
		}
		f.SetDir(f.dir)
		if len(f.names) != 0 {
			t.Fatal("refresh installed inline instead of through the owning queue")
		}
		f.Settle()
		if len(f.names) != 1 || f.names[0] != "Trip" {
			t.Fatal("complete queued menu was not installed")
		}
	})
	t.Run("queued_open_admission", func(t *testing.T) {
		host := &fakeHost{}
		f := newFeature(t, host)
		f.SetUIQueue(&uitest.UIQueue{})
		if err := favstore.Save(f.dir, "Trip", []fyne.URI{storage.NewFileURI("/offline.jpg")}); err != nil {
			t.Fatal(err)
		}
		f.openFavorite("Trip")
		if len(host.opened) != 0 {
			t.Fatal("Favorite opened inline before queued admission")
		}
		host.blockCommands = true
		f.Settle()
		if len(host.opened) != 0 || len(host.syncedDirs) != 0 {
			t.Fatal("late open ignored current command admission")
		}
	})
}

func TestFavoriteStorageMutations(t *testing.T) {
	t.Run("queued_save", func(t *testing.T) {
		host := &fakeHost{files: []fyne.URI{storage.NewFileURI("/captured.jpg")}}
		f := newFeature(t, host)
		saved := 0
		f.SetOnSaved(func() { saved++ })
		f.writeFavorite("Trip")
		if saved != 0 || len(host.syncedDirs) != 0 {
			t.Fatal("save effects ran inline on UI")
		}
		f.Settle()
		if saved != 1 || len(host.syncedDirs) != 1 {
			t.Fatal("committed save did not deliver effects")
		}
	})
	for _, change := range []string{"occupied", "identical_list", "removed"} {
		t.Run(change, func(t *testing.T) {
			host := &fakeHost{files: []fyne.URI{storage.NewFileURI("/captured.jpg")}}
			f := newFeature(t, host)
			old := []fyne.URI{storage.NewFileURI("/old.jpg")}
			if change != "occupied" {
				if err := favstore.Save(f.dir, "Trip", old); err != nil {
					t.Fatal(err)
				}
			}
			started, resume, release := storageGate(t)
			var captures atomic.Int32
			f.SetStorage(&controlledStorage{Store: &favstore.Store{}, capture: func(ctx context.Context, dir, name string) (*favstore.Target, error) {
				target, err := (&favstore.Store{}).Capture(ctx, dir, name)
				if change == "occupied" && captures.Add(1) == 1 {
					close(started)
					<-resume
				}
				return target, err
			}})
			f.AddCurrentList()
			f.addPanel.entry.SetText("Trip")
			f.addPanel.entry.OnSubmitted("Trip")
			if change == "occupied" {
				waitStorage(t, started)
			} else {
				f.Settle()
			}
			if change == "removed" {
				if err := os.Rename(favstore.Dir(f.dir, "Trip"), favstore.Dir(f.dir, "Moved")); err != nil {
					t.Fatal(err)
				}
			} else if err := favstore.Save(f.dir, "Trip", old); err != nil {
				t.Fatal(err)
			}
			host.files = []fyne.URI{storage.NewFileURI("/later.jpg")}
			if change == "occupied" {
				release()
			} else {
				confirmStorageAction(t, f)
			}
			f.Settle()
			if len(host.syncedDirs) != 0 {
				t.Fatal("conflict retried without fresh consent")
			}
			if change == "removed" {
				if f.addPanel == nil || f.addPanel.entry.Text != "Trip" {
					t.Fatal("vacated name did not return to naming")
				}
				f.addPanel.entry.OnSubmitted("Trip")
			} else {
				if f.confirmDialog == nil {
					t.Fatal("changed target did not require a fresh confirmation")
				}
				panel := f.win.Canvas().Focused().(*widgets.ChoicePanel)
				if panel.Selected() != cancelChoice {
					t.Fatal("fresh confirmation did not default to Cancel")
				}
				confirmStorageAction(t, f)
			}
			f.Settle()
			files, err := favstore.Load(f.dir, "Trip")
			if err != nil || len(files) != 1 || files[0].Path() != "/captured.jpg" {
				t.Fatalf("fresh consent lost original capture: %v, %v", files, err)
			}
		})
	}
	t.Run("serialized_cancelled_middle", func(t *testing.T) {
		host := &fakeHost{files: []fyne.URI{storage.NewFileURI("/captured.jpg")}}
		f := newFeature(t, host)
		saveFavorites(t, f, "Old")
		started, resume, release := storageGate(t)
		nativeDone := make(chan struct{})
		uitest.StubTrashMove(t, func(path string) error {
			close(started)
			<-resume
			err := os.RemoveAll(path)
			close(nativeDone)
			return err
		})
		var captures atomic.Int32
		var overlapped atomic.Bool
		f.SetStorage(&controlledStorage{Store: &favstore.Store{}, capture: func(ctx context.Context, dir, name string) (*favstore.Target, error) {
			select {
			case <-nativeDone:
			default:
				overlapped.Store(true)
			}
			captures.Add(1)
			return (&favstore.Store{}).Capture(ctx, dir, name)
		}})
		f.performRemove("Old")
		waitStorage(t, started)
		f.writeFavorite("Cancelled")
		f.Close()
		f.writeFavorite("Fresh")
		if captures.Load() != 0 {
			t.Fatal("save bypassed the active native mutation")
		}
		release()
		f.Settle()
		if overlapped.Load() || captures.Load() != 1 || favstore.Exists(f.dir, "Cancelled") || !favstore.Exists(f.dir, "Fresh") {
			t.Fatal("cancelled queued mutation ran or fresh mutation was lost")
		}
	})
}

func TestFavoriteStorageCommittedEffects(t *testing.T) {
	for _, change := range []string{"close", "root", "stop", "replacement", "refresh_failure"} {
		t.Run(change, func(t *testing.T) {
			host := &fakeHost{files: []fyne.URI{storage.NewFileURI("/captured.jpg")}}
			f := newFeature(t, host)
			dir := f.dir
			saved := 0
			f.SetOnSaved(func() { saved++ })
			started, resume, release := storageGate(t)
			provider := &controlledStorage{Store: &favstore.Store{}, save: func(ctx context.Context, target *favstore.Target, files []fyne.URI) (favstore.SaveResult, error) {
				result, err := (&favstore.Store{}).Save(ctx, target, files)
				close(started)
				<-resume
				return result, err
			}}
			if change == "refresh_failure" {
				provider.list = func(_ context.Context, _ string) ([]favstore.Entry, error) {
					return nil, errors.New("held refresh failure")
				}
			}
			f.SetStorage(provider)
			f.writeFavorite("Trip")
			waitStorage(t, started)
			switch change {
			case "close":
				f.Close()
			case "root":
				f.SetDir(t.TempDir())
			case "stop":
				f.Stop()
			case "replacement":
				if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI("/replacement.jpg")}); err != nil {
					t.Fatal(err)
				}
			}
			joined := make(chan struct{})
			go func() { f.Wait(); close(joined) }()
			select {
			case <-joined:
				t.Fatal("Wait missed the held publication result")
			default:
			}
			release()
			waitStorage(t, joined)
			f.Settle()
			if !favstore.Exists(dir, "Trip") {
				t.Fatal("completed save lost its disk effect")
			}
			want := 1
			if change == "stop" {
				want = 0
			}
			if saved != want || len(host.syncedDirs) != want {
				t.Fatalf("committed effects: saved=%d previews=%d", saved, len(host.syncedDirs))
			}
			if want != 0 && host.syncedDirs[0] != favstore.Dir(dir, "Trip") {
				t.Fatal("committed save retargeted the current root")
			}
			if change == "replacement" {
				access, err := host.syncedOwners[0].Acquire(context.Background())
				if access != nil {
					_ = access.Close()
				}
				if !errors.Is(err, favstore.ErrRetired) {
					t.Fatalf("save handoff adopted a replacement: %v", err)
				}
			}
			if change == "close" || change == "root" || change == "stop" {
				if len(host.toasts) != 0 || f.addDialog != nil || f.confirmDialog != nil || len(host.opened) != 0 {
					t.Fatal("retired save revived presentation")
				}
			}
			if change == "root" && len(f.names) != 0 {
				t.Fatal("old save installed names in a new root")
			}
		})
	}
}

func confirmStorageAction(t *testing.T, f *Feature) {
	t.Helper()
	panel, ok := f.win.Canvas().Focused().(*widgets.ChoicePanel)
	if !ok {
		t.Fatalf("confirmation not focused: %T", f.win.Canvas().Focused())
	}
	panel.Select(confirmChoice)
	panel.Confirm()
}

func storageGate(t *testing.T) (chan struct{}, <-chan struct{}, func()) {
	t.Helper()
	started, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(resume) }) }
	t.Cleanup(release)
	return started, resume, release
}

func TestFavoriteStorageLifecycle(t *testing.T) {
	for _, action := range []string{"close", "source", "root", "modal", "stop"} {
		t.Run(action, func(t *testing.T) {
			host := &fakeHost{}
			f := newFeature(t, host)
			if err := favstore.Save(f.dir, "Trip", []fyne.URI{storage.NewFileURI("/offline.jpg")}); err != nil {
				t.Fatal(err)
			}
			f.openFavorite("Trip")
			f.Wait()
			switch action {
			case "close":
				f.Close()
			case "source":
				f.CancelOpen()
			case "root":
				f.SetDir(t.TempDir())
			case "modal":
				f.SetAvailability(Availability{})
				f.SetAvailability(Availability{Open: true, Add: true, Manage: true})
			case "stop":
				f.Stop()
			}
			f.Settle()
			if len(host.opened) != 0 || len(host.syncedDirs) != 0 {
				t.Fatal("retired storage callback reached the host")
			}
			if action == "close" {
				f.openFavorite("Trip")
				f.Settle()
				if len(host.opened) != 1 {
					t.Fatal("Close prevented fresh reopening")
				}
			}
			if action == "stop" {
				before := host.runCommands
				f.openFavorite("Trip")
				f.SetDir(t.TempDir())
				f.ShowManage()
				f.Settle()
				if host.runCommands != before || f.manageDialog != nil {
					t.Fatal("terminal Stop admitted new work")
				}
			}
		})
	}
	t.Run("active_native_call", func(t *testing.T) {
		host := &fakeHost{}
		f := newFeature(t, host)
		saveFavorites(t, f, "Trip")
		started, resume := make(chan struct{}), make(chan struct{})
		var release sync.Once
		unblock := func() { release.Do(func() { close(resume) }) }
		t.Cleanup(unblock)
		uitest.StubTrashMove(t, func(dir string) error {
			close(started)
			<-resume
			return os.RemoveAll(dir)
		})
		f.performRemove("Trip")
		waitStorage(t, started)
		f.Stop()
		joined := make(chan struct{})
		go func() { f.Wait(); close(joined) }()
		select {
		case <-joined:
			t.Fatal("Wait missed the active native Trash call")
		default:
		}
		unblock()
		waitStorage(t, joined)
		f.Settle()
		if len(host.toasts) != 0 || f.manageDialog != nil {
			t.Fatal("shutdown delivered stale mutation presentation")
		}
		if _, err := os.Stat(filepath.Join(f.dir, "Trip")); !os.IsNotExist(err) {
			t.Fatalf("committed native effect was lost: %v", err)
		}
	})
}

type controlledStorage struct {
	*favstore.Store
	list    func(context.Context, string) ([]favstore.Entry, error)
	open    func(context.Context, string) (favstore.Definition, error)
	capture func(context.Context, string, string) (*favstore.Target, error)
	save    func(context.Context, *favstore.Target, []fyne.URI) (favstore.SaveResult, error)
}

func (s *controlledStorage) Capture(ctx context.Context, dir, name string) (*favstore.Target, error) {
	if s.capture != nil {
		return s.capture(ctx, dir, name)
	}
	return s.Store.Capture(ctx, dir, name)
}

func (s *controlledStorage) Save(ctx context.Context, target *favstore.Target, files []fyne.URI) (favstore.SaveResult, error) {
	if s.save != nil {
		return s.save(ctx, target, files)
	}
	return s.Store.Save(ctx, target, files)
}

func (s *controlledStorage) List(ctx context.Context, dir string) ([]favstore.Entry, error) {
	if s.list != nil {
		return s.list(ctx, dir)
	}
	return s.Store.List(ctx, dir)
}

func (s *controlledStorage) Open(ctx context.Context, dir string) (favstore.Definition, error) {
	if s.open != nil {
		return s.open(ctx, dir)
	}
	return s.Store.Open(ctx, dir)
}

func waitStorage(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("storage operation did not reach its observable boundary")
	}
}
