package similarity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestAnalysisCacheClosesRoots(t *testing.T) {
	dir := t.TempDir()
	shared := storage.NewFileURI(filepath.Join(dir, "shared.jpg"))
	other := storage.NewFileURI(filepath.Join(dir, "other.jpg"))
	for _, name := range []string{"First", "Second"} {
		if err := favstore.Save(dir, name, []fyne.URI{shared, other}); err != nil {
			t.Fatal(err)
		}
	}
	cache, err := openAnalysisCache(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.close()
	roots := make(map[*os.Root]bool)
	for _, favorites := range cache.members {
		for _, favorite := range favorites {
			roots[favorite.root] = true
		}
	}
	if len(roots) != 2 {
		t.Fatalf("premise: cache retained %d directory roots, want 2", len(roots))
	}
	cache.close()
	for root := range roots {
		if _, err := root.Stat("file-list.json"); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("cache shutdown left a directory root open: %v", err)
		}
	}
}

func TestFavoriteAnalysisFollowsOpenedDirectory(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "moved"
		if replacement {
			name = "replaced"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, "original.jpg")
			other := filepath.Join(dir, "other.jpg")
			if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI(original)}); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(favstore.Dir(dir, "Trip"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = root.Close() }()
			moveErr := os.Rename(favstore.Dir(dir, "Trip"), filepath.Join(dir, "Moved"))
			if moveErr != nil {
				// Windows keeps os.Root's directory handle open without delete
				// sharing. The original root must remain valid until it closes.
				const sharingViolation = syscall.Errno(32)
				if runtime.GOOS != "windows" || !errors.Is(moveErr, sharingViolation) {
					t.Fatal(moveErr)
				}
			}
			if replacement && moveErr == nil {
				if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI(other)}); err != nil {
					t.Fatal(err)
				}
			}
			favorite, err := loadFavoriteAnalysis(root)
			if err != nil {
				t.Fatalf("moved favorite lost cache admission: %v", err)
			}
			if len(favorite.members) != 1 || !favorite.members[original] || favorite.members[other] || !favorite.current() {
				t.Fatalf("opened directory associated with replacement membership: %v", favorite.members)
			}
			if moveErr != nil {
				if err := root.Close(); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(favstore.Dir(dir, "Trip"), filepath.Join(dir, "Moved")); err != nil {
					t.Fatalf("closed directory still cannot move: %v", err)
				}
			}
		})
	}
}
