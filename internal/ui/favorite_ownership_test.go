package ui

import (
	"context"
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
