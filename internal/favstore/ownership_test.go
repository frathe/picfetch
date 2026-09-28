package favstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFavoriteOwnership(t *testing.T) {
	t.Run("active_release", func(t *testing.T) {
		dir, name := putDefinition(t, `{"0":"/offline.jpg"}`)
		path := Dir(dir, name)
		definition, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		access, err := definition.Owner.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = access.Root.Close() }()
		cancel()
		if err := access.Current(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("active access ignored cancellation: %v", err)
		}
		if err := access.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := access.Root.Stat("."); err == nil {
			t.Fatal("Close retained active directory access")
		}
		if err := os.Rename(path, filepath.Join(dir, "Released")); err != nil {
			t.Fatalf("cancelled active operation pinned its directory after Close: %v", err)
		}
		if stale, err := definition.Owner.Acquire(context.Background()); !errors.Is(err, ErrRetired) {
			if stale != nil {
				_ = stale.Close()
			}
			t.Fatalf("released owner followed a moved directory: %v", err)
		}
	})
	for _, change := range []string{"move", "directory_replacement", "list_replacement", "list_change", "idle_removal"} {
		t.Run(change, func(t *testing.T) {
			dir, name := putDefinition(t, `{"0":"/offline.jpg"}`)
			path := Dir(dir, name)
			definition, err := Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "move", "directory_replacement", "idle_removal":
				if err := os.Rename(path, filepath.Join(dir, "Retired")); err != nil {
					t.Fatalf("idle owner pinned its directory: %v", err)
				}
				if change == "directory_replacement" {
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, fileListName), []byte(`{"0":"/offline.jpg"}`), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "list_replacement":
				if err := os.Rename(filepath.Join(path, fileListName), filepath.Join(path, "old.json")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, fileListName), []byte(`{"0":"/offline.jpg"}`), 0o600); err != nil {
					t.Fatal(err)
				}
			case "list_change":
				if err := os.WriteFile(filepath.Join(path, fileListName), []byte(`{"0":"/changed-offline.jpg"}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			access, err := definition.Owner.Acquire(context.Background())
			if access != nil {
				_ = access.Close()
			}
			if !errors.Is(err, ErrRetired) {
				t.Fatalf("obsolete owner acquired access: %v", err)
			}
			if change == "move" || change == "idle_removal" {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("acquisition recreated removed path: %v", err)
				}
				if err := os.Rename(filepath.Join(dir, "Retired"), path); err != nil {
					t.Fatal(err)
				}
				if access, err := definition.Owner.Acquire(context.Background()); err == nil {
					_ = access.Close()
					t.Fatal("retired owner revived after path returned")
				}
			}
			fresh, err := Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			access, err = fresh.Owner.Acquire(context.Background())
			if err != nil {
				t.Fatalf("fresh owner refused: %v", err)
			}
			if err := access.Root.WriteFile("fresh-record", []byte("current"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := access.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFavoriteCancellation(t *testing.T) {
	testFavoriteSaveCancellation(t)
	for _, change := range []string{"cancelled", "changed_while_reading", "growth"} {
		t.Run(change, func(t *testing.T) {
			dir, name := putDefinition(t, `{}`)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := Store{readFile: func(ctx context.Context, file *os.File) ([]byte, error) {
				switch change {
				case "cancelled":
					cancel()
				case "changed_while_reading":
					if err := os.WriteFile(file.Name(), []byte(`{"0":"/late.jpg"}`), 0o600); err != nil {
						t.Fatal(err)
					}
				case "growth":
					if err := os.WriteFile(file.Name(), []byte(`{}`+strings.Repeat(" ", MaxDefinitionBytes)), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				return readDefinition(ctx, file)
			}}
			definition, err := store.Open(ctx, Dir(dir, name))
			if err == nil || definition.Owner != nil || definition.Paths != nil {
				t.Fatalf("read admitted partial/unstable definition: %+v, %v", definition, err)
			}
			if change == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if change == "growth" && !errors.Is(err, ErrDefinitionTooLarge) {
				t.Fatalf("growing input bypassed encoded limit: %v", err)
			}
			if err := os.Rename(Dir(dir, name), filepath.Join(dir, "Released")); err != nil {
				t.Fatalf("failed read leaked directory handle: %v", err)
			}
		})
	}
}
