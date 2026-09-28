package favorites

import (
	"context"
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
	list func(context.Context, string) ([]favstore.Entry, error)
	open func(context.Context, string) (favstore.Definition, error)
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
