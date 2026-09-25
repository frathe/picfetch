package locationmap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/favthumbs"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFavoriteFactsRetainOnlyLocation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("source version fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := storage.NewFileURI(path)
	if err := favstore.Save(dir, "Saved", []fyne.URI{source}); err != nil {
		t.Fatal(err)
	}
	owners, err := openFavoriteFacts(context.Background(), dir, []fyne.URI{source})
	if err != nil || len(owners.owners) != 1 {
		t.Fatalf("open Favorite: %v", err)
	}
	defer owners.close()
	version, ok := favthumbs.EntryName(source)
	if !ok {
		t.Fatal("fixture has no source version")
	}
	metadata := imaging.Metadata{HasGPS: true, Latitude: 52.52, Longitude: 13.405, Make: strings.Repeat("x", 96*1024)}
	if err := owners.store(context.Background(), source, Fact{Version: version, Metadata: metadata}); err != nil {
		t.Fatalf("unrelated EXIF prevented GPS persistence: %v", err)
	}
	want := imaging.Metadata{HasGPS: true, Latitude: 52.52, Longitude: 13.405}
	if got, ok, err := owners.load(context.Background(), source, version); err != nil || !ok || got != want {
		t.Fatal("persisted GPS lost location or retained unrelated EXIF")
	}
	// Old or externally modified schema-1 records may still contain camera data.
	metadata.Make = strings.Repeat("x", 30*1024)
	data, err := json.Marshal(gpsRecord{Schema: 1, Path: path, Version: version, Metadata: metadata})
	if err != nil {
		t.Fatal(err)
	}
	if err := owners.owners[0].records.WriteFile(gpsRecordName(path), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := owners.load(context.Background(), source, version); err != nil || !ok || got != want {
		t.Fatal("legacy record retained non-location EXIF")
	}
}

func TestFavoriteRetiredCleanup(t *testing.T) {
	for _, scenario := range []string{"cancelled", "cancelled_during", "bounded", "unexpected_tree"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			sources := []fyne.URI{storage.NewFileURI(filepath.Join(dir, "photo.jpg"))}
			if err := favstore.Save(dir, "Saved", sources); err != nil {
				t.Fatal(err)
			}
			owners, err := openFavoriteFacts(context.Background(), dir, sources)
			if err != nil || len(owners.owners) != 1 {
				t.Fatalf("open initial Favorite: %v", err)
			}
			owners.close()
			retired := filepath.Join(favstore.Dir(dir, "Saved"), favoriteGPSDirectory, strings.Repeat("a", 64))
			if err := os.Mkdir(retired, 0o700); err != nil {
				t.Fatal(err)
			}
			count := 4
			if scenario == "bounded" {
				count = 1100
			} else if scenario == "cancelled_during" {
				count = 200
			}
			for i := range count {
				if err := os.WriteFile(filepath.Join(retired, fmt.Sprintf("%064x.json", i)), []byte("old"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "unexpected_tree" {
				if err := os.Mkdir(filepath.Join(retired, "unexpected"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(retired, "unexpected", "keep"), []byte("not a cache record"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var scanContext context.Context = ctx
			if scenario == "cancelled" {
				scanContext = &resolutionCancelContext{Context: ctx, cancel: cancel, checks: 2}
			} else if scenario == "cancelled_during" {
				scanContext = &resolutionCancelContext{Context: ctx, cancel: cancel, checks: 12}
			}
			owners, err = openFavoriteFacts(scanContext, dir, sources)
			defer owners.close()
			remaining, readErr := os.ReadDir(retired)
			switch scenario {
			case "cancelled":
				if !errors.Is(err, context.Canceled) || ctx.Err() == nil || readErr != nil || len(remaining) != count {
					t.Fatalf("cleanup ignored cancellation: error=%v remaining=%d read=%v", err, len(remaining), readErr)
				}
			case "bounded":
				if err != nil || readErr != nil || len(remaining) == 0 || len(remaining) >= count {
					t.Fatalf("cleanup did not make bounded progress: error=%v remaining=%d read=%v", err, len(remaining), readErr)
				}
			case "cancelled_during":
				if !errors.Is(err, context.Canceled) || readErr != nil || len(remaining) == 0 || len(remaining) >= count {
					t.Fatalf("cleanup did not stop after cancellation during removal: error=%v remaining=%d read=%v", err, len(remaining), readErr)
				}
			case "unexpected_tree":
				if _, err := os.Stat(filepath.Join(retired, "unexpected", "keep")); err != nil {
					t.Fatal("cleanup recursively removed an unexpected directory tree")
				}
			}
		})
	}
}

func TestFavoriteFactsLiveScope(t *testing.T) {
	dir := t.TempDir()
	live := storage.NewFileURI(filepath.Join(dir, "live.jpg"))
	other := storage.NewFileURI(filepath.Join(dir, "other.jpg"))
	for name, sources := range map[string][]fyne.URI{
		"Mixed": {live, other}, "Shared": {live}, "Unrelated": {other},
	} {
		if err := favstore.Save(dir, name, sources); err != nil {
			t.Fatal(err)
		}
	}
	owners, err := openFavoriteFacts(context.Background(), dir, []fyne.URI{live, live})
	if err != nil {
		t.Fatal(err)
	}
	defer owners.close()
	if len(owners.owners) != 2 || len(owners.members) != 1 || len(owners.members[live.Path()]) != 2 {
		t.Fatalf("inventory retained unrelated membership: owners=%d members=%d live owners=%d", len(owners.owners), len(owners.members), len(owners.members[live.Path()]))
	}
	for _, owner := range owners.owners {
		if len(owner.members) != 1 || !owner.members[live.Path()] {
			t.Fatal("owner retained unrelated paths")
		}
	}
	if _, err := os.Stat(filepath.Join(favstore.Dir(dir, "Unrelated"), favoriteGPSDirectory)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unrelated Favorite acquired a location cache: %v", err)
	}
	empty, err := openFavoriteFacts(context.Background(), dir, nil)
	defer empty.close()
	if err != nil || len(empty.owners) != 0 || len(empty.members) != 0 {
		t.Fatal("empty source scope retained Favorite inventory")
	}
}

func TestFavoriteInvalidationKeepsNewerFact(t *testing.T) {
	for _, publication := range []string{"pending", "published", "invalidated"} {
		t.Run(publication, func(t *testing.T) {
			dir := t.TempDir()
			source := storage.NewFileURI(filepath.Join(dir, "edited.jpg"))
			if err := os.WriteFile(source.Path(), []byte("source version fixture"), 0o600); err != nil {
				t.Fatal(err)
			}
			sources := []fyne.URI{source}
			if err := favstore.Save(dir, "Saved", sources); err != nil {
				t.Fatal(err)
			}
			owners, err := openFavoriteFacts(context.Background(), dir, sources)
			if err != nil {
				t.Fatal(err)
			}
			defer owners.close()
			version, ok := favthumbs.EntryName(source)
			if !ok {
				t.Fatal("fixture has no source version")
			}
			old := Fact{Version: version, Metadata: imaging.Metadata{HasGPS: true, Latitude: 52.52, Longitude: 13.405}}
			if err := owners.store(context.Background(), source, old); err != nil {
				t.Fatal(err)
			}
			f := &Feature{facts: NewFactCache(), lifetime: context.Background()}
			f.facts.Keep([]string{source.String()})
			fresh := Fact{Version: version, Metadata: imaging.Metadata{HasGPS: true, Latitude: 40.7, Longitude: -74}}
			if publication != "invalidated" {
				if !f.facts.Capture(source.String(), version).Store(context.Background(), fresh.Metadata) {
					t.Fatal("fresh raw fact was refused")
				}
			}
			if publication == "published" {
				if err := f.persistFact(context.Background(), owners, source, fresh); err != nil {
					t.Fatal(err)
				}
			}
			// Exercise the legal worker ordering in which a fresh scan has already
			// published its raw result before the queued invalidation acquires I/O.
			queue := &uitest.UIQueue{}
			f.invalidatePersistentFacts(queue, dir, sources)
			got, hit, err := owners.load(context.Background(), source, version)
			if err != nil {
				t.Fatal(err)
			}
			if publication == "invalidated" {
				if hit {
					t.Fatal("invalidated disk fact survived without new raw authority")
				}
			} else if !hit || got != fresh.Metadata {
				t.Fatal("delayed invalidation discarded the newly valid Favorite fact")
			}
		})
	}
}
