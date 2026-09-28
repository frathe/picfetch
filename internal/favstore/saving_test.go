package favstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestFavoriteConflicts(t *testing.T) {
	testFavoriteRemovalConflicts(t)
	t.Run("publication", func(t *testing.T) {
		ctx := context.Background()
		dir, name := putDefinition(t, `{"0":"/original"}`)
		store := &Store{beforePublish: func() {
			if err := Save(dir, name, []fyne.URI{storage.NewFileURI("/replacement")}); err != nil {
				t.Fatal(err)
			}
		}}
		target, err := store.Capture(ctx, dir, name)
		if err != nil {
			t.Fatal(err)
		}
		result, err := store.Save(ctx, target, []fyne.URI{storage.NewFileURI("/obsolete")})
		if !errors.Is(err, ErrConflict) || result.Committed {
			t.Fatalf("publication skipped final identity check: %+v, %v", result, err)
		}
		definition, err := Open(ctx, Dir(dir, name))
		if err != nil || definition.Paths[0] != "/replacement" {
			t.Fatalf("replacement overwritten: %+v, %v", definition, err)
		}
	})
	t.Run("retirement", func(t *testing.T) {
		ctx := context.Background()
		store := &Store{}
		dir := t.TempDir()
		target, err := store.Capture(ctx, dir, "Trip")
		if err != nil {
			t.Fatal(err)
		}
		files := []fyne.URI{storage.NewFileURI("/original")}
		if err := Save(dir, "Trip", files); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Save(ctx, target, files); !errors.Is(err, ErrConflict) {
			t.Fatalf("occupancy change missed: %v", err)
		}
		if err := os.Rename(Dir(dir, "Trip"), Dir(dir, "Moved")); err != nil {
			t.Fatal(err)
		}
		if result, err := store.Save(ctx, target, files); !errors.Is(err, ErrConflict) || result.Committed {
			t.Fatalf("observed conflict revived: %+v, %v", result, err)
		}
	})
	for _, change := range []string{"occupied", "identical_list", "directory", "base", "missing_list"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			store := &Store{}
			dir := filepath.Join(t.TempDir(), "favorites")
			files := []fyne.URI{storage.NewFileURI("/original")}
			if change != "occupied" {
				if err := Save(dir, "Trip", files); err != nil {
					t.Fatal(err)
				}
			}
			if change == "missing_list" {
				if err := os.Remove(filepath.Join(Dir(dir, "Trip"), fileListName)); err != nil {
					t.Fatal(err)
				}
			}
			target, err := store.Capture(ctx, dir, "Trip")
			if err != nil {
				t.Fatal(err)
			}
			if target.Occupied() != (change != "occupied") {
				t.Fatal("wrong captured occupancy")
			}
			switch change {
			case "directory":
				if err := os.Rename(Dir(dir, "Trip"), Dir(dir, "Old")); err != nil {
					t.Fatal(err)
				}
			case "base":
				if err := os.Rename(dir, dir+"-old"); err != nil {
					t.Fatal(err)
				}
			}
			if err := Save(dir, "Trip", files); err != nil {
				t.Fatal(err)
			}
			result, err := store.Save(ctx, target, []fyne.URI{storage.NewFileURI("/new")})
			if !errors.Is(err, ErrConflict) || result.Committed {
				t.Fatalf("stale target committed: %+v, %v", result, err)
			}
			definition, err := store.Open(ctx, Dir(dir, "Trip"))
			if err != nil || len(definition.Paths) != 1 || definition.Paths[0] != "/original" {
				t.Fatalf("replacement damaged: %+v, %v", definition, err)
			}
			fresh, err := store.Capture(ctx, dir, "Trip")
			if err != nil {
				t.Fatal(err)
			}
			result, err = store.Save(ctx, fresh, files)
			if err != nil || !result.Committed || result.Definition.Owner == nil {
				t.Fatalf("fresh target failed: %+v, %v", result, err)
			}
			access, err := result.Definition.Owner.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_ = access.Close()
		})
	}
}

func testFavoriteRemovalConflicts(t *testing.T) {
	for _, change := range []string{"identical_list", "directory", "move", "missing_list", "malformed", "cancelled", "committed_after_cancel"} {
		t.Run("removal_"+change, func(t *testing.T) {
			dir, name := putDefinition(t, `{"0":"/original"}`)
			if change == "missing_list" {
				if err := os.Remove(filepath.Join(Dir(dir, name), fileListName)); err != nil {
					t.Fatal(err)
				}
			}
			if change == "malformed" {
				if err := os.WriteFile(filepath.Join(Dir(dir, name), fileListName), []byte(`{"0":""}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := &Store{}
			target, err := store.Capture(ctx, dir, name)
			if err != nil {
				t.Fatal(err)
			}
			moves := 0
			uitest.StubTrashMove(t, func(path string) error {
				moves++
				if path != Dir(dir, name) {
					t.Errorf("retargeted move: %s", path)
				}
				if change == "committed_after_cancel" {
					cancel()
				}
				return os.Rename(path, Dir(dir, "Trashed"))
			})
			conflict := change == "identical_list" || change == "directory" || change == "move"
			if change == "directory" || change == "move" {
				if err := os.Rename(Dir(dir, name), Dir(dir, "Moved")); err != nil {
					t.Fatal(err)
				}
			}
			if change == "directory" || change == "identical_list" {
				if err := Save(dir, name, []fyne.URI{storage.NewFileURI("/original")}); err != nil {
					t.Fatal(err)
				}
			}
			if change == "cancelled" {
				cancel()
			}
			result, err := store.Remove(ctx, target)
			switch {
			case conflict:
				if !errors.Is(err, ErrConflict) || result.Committed || moves != 0 {
					t.Fatalf("stale target reached Trash: %+v %v moves=%d", result, err, moves)
				}
			case change == "cancelled":
				if !errors.Is(err, context.Canceled) || result.Committed || moves != 0 {
					t.Fatalf("unstarted removal ignored cancellation: %+v %v", result, err)
				}
			default:
				if err != nil || !result.Committed || moves != 1 {
					t.Fatalf("captured removal failed: %+v %v moves=%d", result, err, moves)
				}
				if _, err := os.Stat(Dir(dir, name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("committed move did not remove captured directory")
				}
			}
		})
	}
}

func testFavoriteSaveCancellation(t *testing.T) {
	for _, boundary := range []string{"before", "after", "after_replacement"} {
		t.Run("save_"+boundary, func(t *testing.T) {
			dir, name := putDefinition(t, `{"0":"/original"}`)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := &Store{}
			target, err := store.Capture(ctx, dir, name)
			if err != nil {
				t.Fatal(err)
			}
			if boundary == "before" {
				store.beforePublish = cancel
			} else {
				store.afterPublish = func() {
					cancel()
					if boundary == "after_replacement" {
						if err := Save(dir, name, []fyne.URI{storage.NewFileURI("/replacement")}); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			result, err := store.Save(ctx, target, []fyne.URI{storage.NewFileURI("/new")})
			if boundary == "before" {
				if !errors.Is(err, context.Canceled) || result.Committed {
					t.Fatalf("cancelled save published: %+v, %v", result, err)
				}
			} else {
				if err != nil || !result.Committed || result.Definition.Owner == nil || result.Definition.Paths[0] != "/new" {
					t.Fatalf("committed result lost: %+v, %v", result, err)
				}
				access, ownerErr := result.Definition.Owner.Acquire(context.Background())
				if access != nil {
					_ = access.Close()
				}
				if (ownerErr != nil) != (boundary == "after_replacement") {
					t.Fatalf("result adopted a replacement: %v", ownerErr)
				}
			}
			definition, err := Open(context.Background(), Dir(dir, name))
			want := map[string]string{"before": "/original", "after": "/new", "after_replacement": "/replacement"}[boundary]
			if err != nil || definition.Paths[0] != want {
				t.Fatalf("publication result = %+v, %v", definition, err)
			}
		})
	}
}
