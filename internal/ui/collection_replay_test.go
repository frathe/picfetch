package ui

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCollectionCapture(t *testing.T) {
	t.Run("after_removal", collectionCaptureAfterRemoval)
	t.Run("ranked_selected", func(t *testing.T) {
		for _, selected := range []bool{false, true} {
			t.Run(map[bool]string{false: "all", true: "selected"}[selected], func(t *testing.T) {
				v, publish := streamingSearch(t)
				v.favorites.SetDir(t.TempDir())
				publish(similarity.SearchPartial, 2, 1)
				want := []fyne.URI{v.FileAt(0), v.FileAt(2), v.FileAt(1)}
				if selected {
					v.grid.SimulateHover(1)
					v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
					v.grid.SimulateHover(2)
					v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
					want = want[1:]
				}
				before := v.state.Observe()
				v.saveSearchMatches()
				publish(similarity.SearchFinal, 3, 2)
				saveBrowsingFavorite(t, v, want)
				if v.Generation() != before.Generation() || v.state.Observe().Favorite() != before.Favorite() {
					t.Fatal("saving ranked matches rebound collection identity")
				}
			})
		}
	})
	t.Run("saving", func(t *testing.T) {
		v := newTestViewer(t)
		v.startHEICCheck(false)
		v.settleHEIC()
		files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
		unavailable := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
		source := []fyne.URI{files[1], unavailable, files[0], files[0]}
		favorite := storeFavorite(t, v, "Original", source...)
		v.favorites.Open(0)
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		before := v.state.Observe()
		v.favorites.AddCurrentList()
		saveBrowsingFavorite(t, v, []fyne.URI{files[0], files[0], files[1], unavailable})
		if v.state.Observe().Favorite() != favorite || v.Generation() != before.Generation() {
			t.Fatal("saving a Favorite rebound the loaded collection")
		}
		lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
		previous := lifecycle.OnStopped()
		registerShutdown(v.app, v)
		shutdown := lifecycle.OnStopped()
		v.app.Lifecycle().SetOnStopped(previous)
		shutdown()
		if got := session.Load(v.app); !slices.EqualFunc(got, source, sameURI) {
			t.Fatalf("shutdown did not save captured source order: %v", namesOfURIs(got))
		}
		t.Cleanup(func() { session.Save(v.app, nil) })
	})
	t.Run("orders", func(t *testing.T) {
		a, b := storage.NewFileURI("/a.jpg"), storage.NewFileURI("/b.jpg")
		u, gap := storage.NewFileURI("/u.heic"), storage.NewFileURI("/gap.heic")
		state := newAppState(filesort.ByName, false)
		state.Replace(collectionInput{source: []fyne.URI{b, a, a}, display: []fyne.URI{a, a, b}, retained: []collectionSource{{b, false}, {u, true}, {a, false}, {gap, true}, {a, false}, {u, true}}, favorite: "captured-favorite"})
		before := state.Observe()
		if got := before.Capture(collectionSourceOrder); !slices.Equal(got, []fyne.URI{b, u, a, gap, a, u}) {
			t.Fatalf("session source/gap capture = %v", got)
		}
		if got := before.Capture(collectionDisplayOrder); !slices.Equal(got, []fyne.URI{a, gap, a, u, b, u}) {
			t.Fatalf("Favorite display/gap capture = %v", got)
		}
		state.Clear()
		captured := before.Capture(collectionSourceOrder)
		captured[0] = u
		if before.Capture(collectionSourceOrder)[0] != b || before.Favorite() != "captured-favorite" {
			t.Fatal("capture or later clear changed a saved observation")
		}
	})
}

func TestCollectionReplay(t *testing.T) {
	t.Run("unavailable_only", func(t *testing.T) {
		v := newTestViewer(t)
		v.startHEICCheck(false)
		v.settleHEIC()
		uri := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
		favorite := storeFavorite(t, v, "Unavailable", uri, uri)
		v.favorites.Open(0)
		waitForScan(t, v)
		observation := v.state.Observe()
		if observation.Count() != 0 || len(observation.Retained()) != 2 || observation.Favorite() != favorite {
			t.Fatal("saved unavailable-only replay lost repeated membership or association")
		}
	})
	t.Run("discovery_deduplicates_aliases", func(t *testing.T) {
		v := newTestViewer(t)
		photo := uitest.TempDirJPEGURIs(t, "photo.jpg")[0]
		alias := filepath.Join(t.TempDir(), "alias.jpg")
		if err := os.Symlink(photo.Path(), alias); err != nil {
			t.Skipf("symlink fixture unavailable: %v", err)
		}
		dropAndWait(t, v, photo, storage.NewFileURI(alias))
		if got := v.state.Observe(); got.Count() != 1 || len(got.Retained()) != 1 || got.Retained()[0].uri.String() != photo.String() {
			t.Fatal("discovery retained an occurrence its scanner deduplicated")
		}
	})
	t.Run("missing_entry", func(t *testing.T) {
		v := newTestViewer(t)
		photo := uitest.TempDirJPEGURIs(t, "a.jpg")[0]
		missing := storage.NewFileURI(filepath.Join(t.TempDir(), "0-missing.jpg"))
		storeFavorite(t, v, "Missing", missing, photo, photo)
		v.favorites.Open(0)
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		if got := v.state.Observe().SourceFiles(); !slices.EqualFunc(got, []fyne.URI{photo, photo}, sameURI) {
			t.Fatalf("ordinary missing-source recovery changed surviving replay occurrences: %v", got)
		}
	})
	t.Run("trial_truncation", func(t *testing.T) {
		v := newTestViewer(t)
		trial, err := explorertrial.New(filepath.Join(t.TempDir(), "trial"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = trial.Close() }()
		configureExplorer(v, func(options *explorerui.Options) { options.Trial = trial })
		limit := 1
		v.applyLaunchOptions(launch.Options{MaxFiles: &limit, PictureFrame: true})
		photo := uitest.TempDirJPEGURIs(t, "a.jpg")[0]
		storeFavorite(t, v, "Truncated", photo, photo)
		before := v.state.Observe()
		v.favorites.Open(0)
		waitForScan(t, v)
		if v.Generation() != before.Generation() || v.FileCount() != 0 || v.explorerInput.pendingLaunch || v.pendingPictureFrame || v.sortOp.done.Begun() {
			t.Fatal("trial replay bypassed truncation refusal")
		}
	})
	t.Run("truncated_browsable", func(t *testing.T) {
		v := newTestViewer(t)
		photo := uitest.TempDirJPEGURIs(t, "a.jpg")[0]
		v.SetMaxScan(2)
		storeFavorite(t, v, "Capped", photo, photo, photo)
		v.favorites.Open(0)
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		if v.FileCount() != 2 || !strings.Contains(v.toast.text.Text, "scan limit") {
			t.Fatal("repeated replay bypassed its cap or truncation feedback")
		}
	})
	t.Run("unavailable_entries_and_limits", func(t *testing.T) {
		for _, tc := range []struct {
			name                string
			limit               int
			retained, browsable []string
		}{
			{"gaps", 10, []string{"u.heic", "u.heic", "a.jpg", "u.heic", "a.jpg"}, []string{"a.jpg", "a.jpg"}},
			{"separate_caps", 2, []string{"u.heic", "u.heic", "a.jpg", "a.jpg"}, []string{"a.jpg", "a.jpg"}},
			{"floor", 0, []string{"u.heic", "a.jpg"}, []string{"a.jpg"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := newTestViewer(t)
				v.startHEICCheck(false)
				v.settleHEIC()
				v.SetMaxScan(tc.limit)
				unavailable := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
				photo := uitest.TempDirJPEGURIs(t, "a.jpg")[0]
				favorite := storeFavorite(t, v, "Bounded", unavailable, unavailable, photo, unavailable, photo)
				v.favorites.Open(0)
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				observation := v.state.Observe()
				var retained []fyne.URI
				for _, source := range observation.Retained() {
					retained = append(retained, source.uri)
				}
				if !slices.Equal(namesOfURIs(retained), tc.retained) || !slices.Equal(namesOfURIs(observation.SourceFiles()), tc.browsable) || observation.Favorite() != favorite {
					t.Fatalf("bounded saved replay: retained=%v browsable=%v", namesOfURIs(retained), namesOfURIs(observation.SourceFiles()))
				}
			})
		}
	})
	t.Run("pending_capability", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			available bool
			limit     int
			cancel    bool
			want      []string
		}{
			{"available", true, 5, false, []string{"u.heic", "u.heic", "a.jpg", "a.jpg"}},
			{"available_at_cap", true, 3, false, []string{"u.heic", "u.heic", "a.jpg"}},
			{"unavailable", false, 5, false, []string{"a.jpg", "a.jpg"}},
			{"unavailable_at_cap", false, 1, false, []string{"a.jpg"}},
			{"canceled", true, 5, true, nil},
			{"superseded", true, 5, true, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := newTestViewer(t)
				original := uitest.TempDirJPEGURIs(t, "original.jpg", "other.jpg")
				v.OpenFavorite("original-favorite", original)
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				v.settleHEIC()
				before := v.state.Observe()
				preferences.ClearHEICObservation(v.app)
				entered, release := make(chan struct{}), make(chan struct{})
				unblock := sync.OnceFunc(func() { close(release) })
				t.Cleanup(unblock)
				v.configureHEIC(testHEICBackend{
					check: func(ctx context.Context) error {
						close(entered)
						select {
						case <-release:
							if tc.available {
								return nil
							}
							return heic.ErrUnavailable
						case <-ctx.Done():
							return ctx.Err()
						}
					},
					read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
						result := heic.Result{Width: 2, Height: 1}
						if request.Pixels {
							result.Stride = 8
							result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 255}
						}
						return result, nil
					},
				})
				v.heic.ui = &uitest.UIQueue{}
				v.SetMaxScan(tc.limit)
				data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
				if err != nil {
					t.Fatal(err)
				}
				unavailable := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", data))
				photo := uitest.TempDirJPEGURIs(t, "a.jpg")[0]
				favorite := storeFavorite(t, v, "Pending", unavailable, unavailable, photo, photo)
				v.favorites.Open(0)
				select {
				case <-entered:
				case <-time.After(testTimeout):
					t.Fatal("replay did not start its captured capability check")
				}
				if tc.cancel {
					completion := v.scanOp.done.Current()
					if tc.name == "superseded" {
						v.OpenFavorite("newer-favorite", original)
						waitForScan(t, v)
						waitForSort(t, v)
						waitUntilLoaded(t, v)
						before = v.state.Observe()
					} else {
						v.cancelScan()
					}
					unblock()
					ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
					defer cancel()
					if err := completion.Wait(ctx); err != nil {
						t.Fatal("retired replay did not complete:", err)
					}
					if after := v.state.Observe(); after.Generation() != before.Generation() || after.Favorite() != before.Favorite() || !slices.Equal(after.Retained(), before.Retained()) {
						t.Fatal("canceled replay published partial collection facts")
					}
					return
				}
				unblock()
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				after := v.state.Observe()
				if !slices.Equal(namesOfURIs(after.SourceFiles()), tc.want) || after.Favorite() != favorite || after.Generation() != before.Generation()+1 {
					t.Fatalf("completed capability replay = %v", namesOfURIs(after.SourceFiles()))
				}
				if !tc.available && len(after.Retained()) != 2*min(2, tc.limit) {
					t.Fatal("pending unavailable replay lost repeated retained members")
				}
			})
		}
	})
	for _, entry := range []string{"favorite", "session"} {
		for _, shape := range []string{"singleton", "repeated"} {
			t.Run(entry+"/"+shape, func(t *testing.T) {
				files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "neighbor.jpg")
				recorded := []fyne.URI{files[1]}
				wantDisplay := []fyne.URI{files[1]}
				if shape == "repeated" {
					recorded = []fyne.URI{files[1], files[0], files[1]}
					wantDisplay = []fyne.URI{files[0], files[1], files[1]}
				}
				if entry == "session" {
					session.Save(testApp, recorded)
					t.Cleanup(func() { session.Save(testApp, nil) })
				}
				v := newTestViewer(t)
				v.state.SetSortMode(filesort.ByName)
				before := v.state.Observe()
				favorite := ""
				if entry == "favorite" {
					favorite = storeFavorite(t, v, "Recorded", recorded...)
					v.favorites.Open(0)
				} else {
					v.restoreSession()
				}
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				after := v.state.Observe()
				if !slices.EqualFunc(after.SourceFiles(), recorded, sameURI) || !slices.EqualFunc(after.DisplayFiles(), wantDisplay, sameURI) {
					t.Fatalf("recorded occurrences/order changed: source=%v display=%v", namesOfURIs(after.SourceFiles()), namesOfURIs(after.DisplayFiles()))
				}
				if after.Favorite() != favorite || after.Generation() != before.Generation()+1 {
					t.Fatal("replay did not bind a fresh complete collection")
				}
				if entry == "session" && (v.savedSession != nil || v.restoreLink.Visible()) {
					t.Fatal("replayed session left its restore offer active")
				}
				if entry == "favorite" {
					bookmark, _ := after.Bookmark(after.Count() - 1)
					v.favorites.Open(0)
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					if v.state.Observe().Resolve(bookmark) != -1 {
						t.Fatal("saved replay reused a previous collection binding")
					}
				}
			})
		}
	}
}
