package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFavoriteOwnershipIntegration(t *testing.T) {
	for _, change := range []string{"close", "root", "shutdown"} {
		t.Run("native_removal_"+change, func(t *testing.T) {
			v := newTestViewer(t)
			file := uitest.TempJPEGURI(t, "current.jpg", 4, 4, color.White)
			dropAndWait(t, v, file)
			dir := storeFavorite(t, v, "Trip", file)
			started, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			t.Cleanup(release)
			trashDir := filepath.Join(t.TempDir(), "Trashed")
			uitest.StubTrashMove(t, func(path string) error {
				if path != dir {
					return fmt.Errorf("unexpected captured target %q", path)
				}
				close(started)
				<-resume
				return os.Rename(path, trashDir)
			})
			v.favorites.ShowManage()
			v.favorites.Settle()
			panel := v.win.Canvas().Focused()
			if panel == nil {
				t.Fatal("Manage did not own input")
			}
			panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			v.favorites.Settle()
			panel = v.win.Canvas().Focused()
			if panel == nil || len(v.win.Canvas().Overlays().List()) != 2 {
				t.Fatal("removal did not raise its captured confirmation")
			}
			panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitFavoriteStorageBoundary(t, started)
			scan := v.scanOp.done.Current()
			switch change {
			case "close":
				v.favorites.Close()
			case "root":
				v.favorites.SetDir(t.TempDir())
			case "shutdown":
				lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
				previous := lifecycle.OnStopped()
				registerShutdown(v.app, v)
				shutdown := lifecycle.OnStopped()
				v.app.Lifecycle().SetOnStopped(previous)
				shutdown()
				joined := make(chan struct{})
				go func() { v.waitForShutdown(); close(joined) }()
				select {
				case <-joined:
					t.Fatal("root shutdown missed native Favorite removal")
				default:
				}
				release()
				waitFavoriteStorageBoundary(t, joined)
			}
			release()
			v.favorites.Settle()
			if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("committed native removal lost: %v", err)
			}
			if v.scanOp.done.Current() != scan || v.FileCount() != 1 || v.FileAt(0).String() != file.String() {
				t.Fatal("removal changed the current collection")
			}
			if len(v.win.Canvas().Overlays().List()) != 0 {
				t.Fatal("native completion revived a closed dialog")
			}
		})
	}
	for _, change := range []string{"close", "opt_out", "shutdown"} {
		t.Run("committed_save_"+change, func(t *testing.T) {
			v := newTestViewer(t)
			file := uitest.TempJPEGURI(t, "source.jpg", 4, 4, color.White)
			dropAndWait(t, v, file)
			v.favorites.SetDir(t.TempDir())
			v.favorites.Settle()
			started, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			t.Cleanup(release)
			v.favorites.SetStorage(&favoriteSaveGate{Store: &favstore.Store{}, started: started, resume: resume})
			v.favorites.AddCurrentList()
			entry, ok := v.win.Canvas().Focused().(interface {
				SetText(string)
				TypedKey(*fyne.KeyEvent)
			})
			if !ok {
				t.Fatal("Favorite naming was not admitted")
			}
			entry.SetText("Captured")
			entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitFavoriteStorageBoundary(t, started)
			scan := v.scanOp.done.Current()
			switch change {
			case "close":
				v.favorites.Close()
			case "opt_out":
				v.SetFavoritePreviewCache(false)
			case "shutdown":
				lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
				previous := lifecycle.OnStopped()
				registerShutdown(v.app, v)
				shutdown := lifecycle.OnStopped()
				v.app.Lifecycle().SetOnStopped(previous)
				shutdown()
				joined := make(chan struct{})
				go func() { v.waitForShutdown(); close(joined) }()
				select {
				case <-joined:
					t.Fatal("root shutdown missed a held save result")
				default:
				}
				release()
				waitFavoriteStorageBoundary(t, joined)
			}
			release()
			v.favorites.Settle()
			files, err := favstore.Load(v.favorites.Dir(), "Captured")
			if err != nil || len(files) != 1 || files[0].String() != file.String() {
				t.Fatalf("committed root save lost: %v, %v", files, err)
			}
			if v.scanOp.done.Current() != scan || v.state.Observe().Favorite() != "" {
				t.Fatal("save notification replayed the collection")
			}
			if v.favThumb.Begun() != (change == "close") {
				t.Fatal("committed preview handoff lost or revived opted-out/stopped work")
			}
		})
	}
	t.Run("queued_replay", func(t *testing.T) {
		v := newTestViewer(t)
		files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
		dropAndWait(t, v, files[0])
		saved := []fyne.URI{files[1], files[0], files[1]}
		dir := storeFavorite(t, v, "Trip", saved...)
		before := v.Generation()
		v.favorites.Open(0)
		v.favorites.Wait()
		if v.Generation() != before {
			t.Fatal("worker applied Favorite replay before UI delivery")
		}
		v.favorites.Settle()
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		if v.state.Observe().Favorite() != dir || !slices.EqualFunc(v.state.Observe().Capture(collectionSourceOrder), saved, sameURI) {
			t.Fatal("queued opening lost original occurrences or association")
		}
	})
	for _, change := range []string{"modal", "source", "malformed"} {
		t.Run(change, func(t *testing.T) {
			v := newTestViewer(t)
			files := []fyne.URI{
				uitest.TempJPEGURI(t, "current.jpg", 4, 4, color.White),
				uitest.TempJPEGURI(t, "favorite.jpg", 4, 4, color.White),
				uitest.TempJPEGURI(t, "new.jpg", 4, 4, color.White),
			}
			dropAndWait(t, v, files[0])
			dir := storeFavorite(t, v, "Trip", files[1])
			if change == "malformed" {
				if err := os.WriteFile(filepath.Join(dir, "file-list.json"), []byte(`{"0":"/first.jpg","9":""}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			queue := &uitest.UIQueue{}
			v.favorites.SetUIQueue(queue)
			v.favorites.Open(0)
			v.favorites.Wait()
			if queue.Len() != 1 {
				t.Fatal("fixture did not hold one completed Favorite open")
			}
			want := files[0]
			switch change {
			case "modal":
				v.requestDelete()
			case "source":
				dropAndWait(t, v, files[2])
				want = files[2]
			}
			scanBefore := v.scanOp.done.Current()
			v.favorites.Settle()
			if v.scanOp.done.Current() != scanBefore {
				t.Fatal("obsolete/invalid Favorite open admitted another scan")
			}
			if v.FileCount() != 1 || v.FileAt(0).String() != want.String() || v.state.Observe().Favorite() != "" {
				t.Fatal("obsolete/invalid Favorite open changed collection")
			}
		})
	}
	for _, change := range []string{"identical_replacement", "shutdown"} {
		t.Run(change, func(t *testing.T) {
			v := newTestViewer(t)
			files := []fyne.URI{
				uitest.TempJPEGURI(t, "current.jpg", 4, 4, color.White),
				uitest.TempJPEGURI(t, "favorite.jpg", 4, 4, color.White),
			}
			dropAndWait(t, v, files[0])
			dir := storeFavorite(t, v, "Trip", files[1])
			started, resume := make(chan struct{}), make(chan struct{})
			var release sync.Once
			unblock := func() { release.Do(func() { close(resume) }) }
			t.Cleanup(unblock)
			v.favorites.SetStorage(&favoriteReadGate{Store: &favstore.Store{}, started: started, resume: resume})
			v.favorites.Open(0)
			waitFavoriteStorageBoundary(t, started)
			if change == "identical_replacement" {
				if err := favstore.Save(filepath.Dir(dir), "Trip", files[1:]); err != nil {
					t.Fatal(err)
				}
			} else {
				lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
				previous := lifecycle.OnStopped()
				registerShutdown(v.app, v)
				shutdown := lifecycle.OnStopped()
				v.app.Lifecycle().SetOnStopped(previous)
				shutdown()
				joined := make(chan struct{})
				go func() { v.waitForShutdown(); close(joined) }()
				select {
				case <-joined:
					t.Fatal("root shutdown missed the held Favorite worker")
				default:
				}
				unblock()
				waitFavoriteStorageBoundary(t, joined)
			}
			unblock()
			v.favorites.Settle()
			if v.FileCount() != 1 || v.FileAt(0).String() != files[0].String() || v.state.Observe().Favorite() != "" {
				t.Fatal("held Favorite read survived owner/shutdown retirement")
			}
		})
	}
}

type favoriteReadGate struct {
	*favstore.Store
	started chan struct{}
	resume  <-chan struct{}
}

type favoriteSaveGate struct {
	*favstore.Store
	started chan struct{}
	resume  <-chan struct{}
}

func (s *favoriteSaveGate) Save(ctx context.Context, target *favstore.Target, files []fyne.URI) (favstore.SaveResult, error) {
	result, err := s.Store.Save(ctx, target, files)
	close(s.started)
	<-s.resume
	return result, err
}

func (s *favoriteReadGate) Open(ctx context.Context, dir string) (favstore.Definition, error) {
	definition, err := s.Store.Open(ctx, dir)
	close(s.started)
	<-s.resume
	return definition, err
}

func waitFavoriteStorageBoundary(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("Favorite worker did not reach its observable boundary")
	}
}
