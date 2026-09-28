package favthumbs

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestSyncFavoriteOwnership(t *testing.T) {
	t.Run("held_read", func(t *testing.T) {
		dir := t.TempDir()
		owner := testFavoriteOwner(t, dir)
		source := uitest.TempJPEGURI(t, "source.jpg", 4, 4, color.White)
		data, err := os.ReadFile(source.Path())
		if err != nil {
			t.Fatal(err)
		}
		started, resume := make(chan struct{}), make(chan struct{})
		release := sync.OnceFunc(func() { close(resume) })
		t.Cleanup(release)
		reader := bytes.NewReader(data)
		pause := sync.OnceFunc(func() { close(started); <-resume })
		held := uitest.ReaderURI(source, func() (io.ReadCloser, error) {
			return uitest.ReadCloser{
				ReadFunc:  func(p []byte) (int, error) { pause(); return reader.Read(p) },
				CloseFunc: func() error { return nil },
			}, nil
		})
		done := make(chan error, 1)
		go func() { done <- Sync(context.Background(), owner, []fyne.URI{held}, 1, nil) }()
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("original read was not admitted")
		}
		if err := favstore.Save(filepath.Dir(dir), filepath.Base(dir), nil); err != nil {
			t.Fatal(err)
		}
		release()
		select {
		case err := <-done:
			if !errors.Is(err, favstore.ErrRetired) {
				t.Fatalf("held original result = %v, want retirement", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("held original did not finish")
		}
		if _, err := os.Stat(Dir(dir)); !os.IsNotExist(err) {
			t.Fatalf("held original published into replacement: %v", err)
		}
	})
	t.Run("publication", func(t *testing.T) {
		dir := t.TempDir()
		owner := testFavoriteOwner(t, dir)
		source := newSourceFile(t, t.TempDir(), "source.jpg")
		if err := Write(owner, source, newOpaqueThumb(4, 4, color.RGBA{G: 255, A: 255})); err != nil {
			t.Fatal(err)
		}
		path := previewPath(t, dir, source, ".jpg")
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		replace := sync.OnceFunc(func() {
			if err := favstore.Save(filepath.Dir(dir), filepath.Base(dir), nil); err != nil {
				t.Fatal(err)
			}
		})
		pixels := cancelOnEncodeImage{Image: newOpaqueThumb(4, 4, color.RGBA{R: 255, A: 255}), cancel: replace}
		if err := Write(owner, source, pixels); !errors.Is(err, favstore.ErrRetired) {
			t.Fatalf("publish after identical replacement = %v", err)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("retired publication changed the current preview: %v", err)
		}
		entries, err := os.ReadDir(Dir(dir))
		if err != nil || len(entries) != 1 {
			t.Fatalf("retired publication left temporary output: %v, %v", entries, err)
		}
		if _, ok, err := ReadContext(context.Background(), owner, source); ok || !errors.Is(err, favstore.ErrRetired) {
			t.Fatalf("retired read accepted existing pixels: hit=%v err=%v", ok, err)
		}
		fresh := testFavoriteOwner(t, dir)
		if err := Write(fresh, source, newOpaqueThumb(8, 4, color.RGBA{B: 255, A: 255})); err != nil {
			t.Fatal(err)
		}
		if pixels, ok := Read(fresh, source); !ok || pixels.Bounds().Dx() != 8 {
			t.Fatal("fresh ownership did not read/write current previews")
		}
	})
	t.Run("cleanup", func(t *testing.T) {
		dir := t.TempDir()
		owner := testFavoriteOwner(t, dir)
		if err := os.Mkdir(Dir(dir), 0755); err != nil {
			t.Fatal(err)
		}
		relative := filepath.Join(SubDir, "corrupt.jpg")
		if err := os.WriteFile(filepath.Join(dir, relative), []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
		access, err := owner.Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = access.Close() }()
		failed, err := access.Root.Open(relative)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = failed.Close() }()
		if err := favstore.Save(filepath.Dir(dir), filepath.Base(dir), nil); err != nil {
			t.Fatal(err)
		}
		discardCorruptPreview(context.Background(), access, relative, failed)
		if _, err := os.Stat(filepath.Join(dir, relative)); err != nil {
			t.Fatalf("retired corrupt cleanup unlinked current storage: %v", err)
		}
	})
	t.Run("sweep", func(t *testing.T) {
		dir := t.TempDir()
		owner := testFavoriteOwner(t, dir)
		source := newSourceFile(t, t.TempDir(), "source.jpg")
		if err := Write(owner, source, newOpaqueThumb(4, 4, color.RGBA{A: 255})); err != nil {
			t.Fatal(err)
		}
		member := ownerChangingURI{URI: newSourceFile(t, t.TempDir(), "member.jpg"), change: sync.OnceFunc(func() {
			if err := favstore.Save(filepath.Dir(dir), filepath.Base(dir), nil); err != nil {
				t.Fatal(err)
			}
		})}
		if err := Sweep(owner, []fyne.URI{member}); !errors.Is(err, favstore.ErrRetired) {
			t.Fatalf("sweep after identical replacement = %v", err)
		}
		if _, err := os.Stat(previewPath(t, dir, source, ".jpg")); err != nil {
			t.Fatalf("retired sweep unlinked current preview: %v", err)
		}
	})
	t.Run("fresh_record", func(t *testing.T) {
		dir := t.TempDir()
		owner := testFavoriteOwner(t, dir)
		source := newSourceFile(t, t.TempDir(), "source.jpg")
		if err := Write(owner, source, newOpaqueThumb(4, 4, color.RGBA{A: 255})); err != nil {
			t.Fatal(err)
		}
		path := previewPath(t, dir, source, ".jpg")
		observed, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := Write(owner, source, newOpaqueThumb(8, 4, color.RGBA{G: 255, A: 255})); err != nil {
			t.Fatal(err)
		}
		access, err := owner.Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = access.Close() }()
		if err := removeStalePreview(context.Background(), access, filepath.Join(SubDir, filepath.Base(path)), observed); err != nil {
			t.Fatal(err)
		}
		if pixels, ok := Read(owner, source); !ok || pixels.Bounds().Dx() != 8 {
			t.Fatal("old sweep inventory removed a newly published preview")
		}
	})
	for _, change := range []string{"move", "remove", "identical_list", "directory"} {
		t.Run(change, func(t *testing.T) {
			base := t.TempDir()
			source := uitest.TempJPEGURI(t, "source.jpg", 4, 4, color.White)
			files := []fyne.URI{source}
			if err := favstore.Save(base, "Trip", files); err != nil {
				t.Fatal(err)
			}
			dir := favstore.Dir(base, "Trip")
			definition, err := favstore.Open(context.Background(), dir)
			if err != nil {
				t.Fatal(err)
			}
			started, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			release := func() { once.Do(func() { close(resume) }) }
			t.Cleanup(release)
			version, ok := EntryName(source)
			if !ok {
				t.Fatal("fixture source has no version")
			}
			sink := heldOwnerSink{started: started, resume: resume, preview: &Preview{Image: image.NewRGBA(image.Rect(0, 0, 4, 4)), SourceVersion: version}}
			done := make(chan error, 1)
			go func() { done <- Sync(context.Background(), definition.Owner, files, 1, sink) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("preview did not reach held memory lookup")
			}
			switch change {
			case "move", "directory":
				if err := os.Rename(dir, filepath.Join(base, "Moved")); err != nil {
					t.Fatal(err)
				}
			case "remove":
				if err := os.RemoveAll(dir); err != nil {
					t.Fatal(err)
				}
			}
			if change == "identical_list" || change == "directory" {
				if err := favstore.Save(base, "Trip", files); err != nil {
					t.Fatal(err)
				}
			}
			release()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("retired preview did not finish")
			}
			if _, err := os.Stat(Dir(dir)); !os.IsNotExist(err) {
				t.Fatalf("retired preview created/wrote current storage: %v", err)
			}
			if change == "move" || change == "directory" {
				if _, err := os.Stat(Dir(filepath.Join(base, "Moved"))); !os.IsNotExist(err) {
					t.Fatal("retired preview followed moved storage")
				}
			}
		})
	}
}

type ownerChangingURI struct {
	fyne.URI
	change func()
}

func (u ownerChangingURI) Path() string { u.change(); return u.URI.Path() }

type heldOwnerSink struct {
	started chan struct{}
	resume  <-chan struct{}
	preview image.Image
}

func (s heldOwnerSink) Cached(_ fyne.URI) (image.Image, bool) {
	close(s.started)
	<-s.resume
	return s.preview, true
}
func (_ heldOwnerSink) Store(_ fyne.URI, _ image.Image) {}
