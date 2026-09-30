package similarity

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/fileaccess"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestWorkerAccessCapturesExactPathsUntilRetirement(t *testing.T) {
	root := storage.NewFileURI(t.TempDir()).Path()
	source, err := fileaccess.FromRecord(fileaccess.Record{URI: storage.NewFileURI(root + "/photos/one.png").String(), Bookmark: []byte("persistent")})
	if err != nil {
		t.Fatal(err)
	}
	ctx := fileaccess.WithSources(context.Background(), []fyne.URI{source})
	req := request{Assets: root + "/models", Paths: []string{source.Path(), source.Path()}, GeneralAnalysisDir: root + "/cache"}
	active := map[string]bool{}
	release, err := captureWorkerAccess(ctx, &req, root+"/PicFetch.app", func(_ context.Context, uri fyne.URI) (fileaccess.Transfer, func(), error) {
		path := uri.Path()
		if path == source.Path() && uri != source {
			t.Fatal("source authority flattened")
		}
		active[path] = true
		return fileaccess.Transfer{Path: path, Bookmark: []byte("implicit")}, func() { active[path] = false }, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Use absolute host paths: Windows roots need a volume as well as slashes.
	want := []string{root + "/PicFetch.app", root + "/models/vision_model.onnx", root + "/models/preprocessor_config.json", root + "/photos/one.png", root + "/cache"}
	if len(req.Access) != len(want) {
		t.Fatalf("unexpected grants: %+v", req.Access)
	}
	for _, path := range want {
		if !active[path] {
			t.Fatalf("access lost: %s", path)
		}
	}
	if active[root+"/photos"] {
		t.Fatal("source expanded to parent")
	}
	release()
	release()
	for path, on := range active {
		if on {
			t.Fatalf("access leaked: %s", path)
		}
	}
}

func TestWorkerAccessFailureRetiresPriorGrants(t *testing.T) {
	req := request{Assets: "/models"}
	retired := 0
	_, err := captureWorkerAccess(context.Background(), &req, "/App.app", func(_ context.Context, uri fyne.URI) (fileaccess.Transfer, func(), error) {
		if uri.Path() != "/App.app" {
			return fileaccess.Transfer{}, nil, errors.New("missing models")
		}
		return fileaccess.Transfer{Path: uri.Path()}, func() { retired++ }, nil
	})
	if err == nil || retired != 1 {
		t.Fatalf("err=%v retired=%d", err, retired)
	}
}

func TestWorkerAccessMovedSourcesKeepCollectionIdentity(t *testing.T) {
	checkScans := RegisterLocalFiles()
	t.Cleanup(func() {
		if err := checkScans(); err != nil {
			t.Error(err)
		}
	})
	for _, search := range []bool{false, true} {
		t.Run(fmt.Sprintf("search=%t", search), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			policy := cacheTestPolicy(t)
			items := []Item{cacheFixtureItem(t, "one.jpg"), cacheFixtureItem(t, "two.jpg")}
			store := cacheTestStore(t, policy)
			moved := map[string]string{}
			var paths []string
			for i := range items {
				item := &items[i]
				data := uitest.EncodeJPEG(t, 32, 24, color.White)
				if err := os.WriteFile(item.Path, data, 0600); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(item.Path)
				if err != nil {
					t.Fatal(err)
				}
				item.Path = storage.NewFileURI(item.Path).Path()
				item.Size, item.ModifiedNS = info.Size(), info.ModTime().UnixNano()
				item.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
				if err := store.write(ctx, *item); err != nil {
					t.Fatal(err)
				}
				paths = append(paths, item.Path)
				newPath := storage.NewFileURI(filepath.Join(t.TempDir(), "moved.jpg")).Path()
				if err := os.Rename(item.Path, newPath); err != nil {
					t.Fatal(err)
				}
				moved[item.Path] = newPath
			}
			req := request{Assets: t.TempDir(), Paths: slices.Clone(paths),
				GeneralAnalysisDir: policy.Roots.GeneralDir, GeneralAnalysisLimitBytes: policy.GeneralLimitBytes}
			if search {
				req.Search = &SearchRequest{SessionID: 7, Paths: slices.Clone(paths), Cache: policy}
			}
			active := 0
			release, err := captureWorkerAccess(ctx, &req, t.TempDir(), func(_ context.Context, uri fyne.URI) (fileaccess.Transfer, func(), error) {
				path := uri.Path()
				if resolved := moved[path]; resolved != "" {
					path = resolved
				}
				active++
				return fileaccess.Transfer{Path: path, Bookmark: []byte("test grant")}, func() { active-- }, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			if !slices.Equal(req.Paths, paths) || search && !slices.Equal(req.Search.Paths, paths) {
				t.Fatal("capture changed collection identities")
			}
			// Exercise the actual serialized boundary, not an in-process-only map.
			wire, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			var decoded request
			if err := json.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			for original, resolved := range moved {
				if decoded.SourcePaths.resolve(original) != resolved {
					t.Fatalf("worker still reads stale source %s", original)
				}
			}
			preparer := searchPreparer{cache: store, versions: map[string]os.FileInfo{}, sources: decoded.SourcePaths}
			for _, item := range items {
				got, reused, err := preparer.prepare(ctx, item.Path)
				if err != nil || !reused || got.Path != item.Path {
					t.Fatalf("moved preparation: %+v, reused=%t, %v", got, reused, err)
				}
				preview, err := preparer.completePreview(ctx, got)
				if err != nil || len(preview.Preview) == 0 || preview.Path != item.Path {
					t.Fatalf("moved Favorite preview: %v", err)
				}
			}
			if err := preparer.validate(ctx, items[0], []Match{{Path: items[1].Path}}, true); err != nil {
				t.Fatal(err)
			}
			queries := make(chan SearchQuery, 1)
			queries <- SearchQuery{ID: 1, ReferencePath: paths[0]}
			var matches []Match
			err = preparer.run(ctx, SearchRequest{SessionID: 7, Paths: paths, Cache: policy}, queries, func(event SearchEvent) error {
				if event.Kind == SearchFinal {
					matches = event.Matches
				}
				if event.Kind == SearchReady {
					close(queries)
				}
				return nil
			})
			if err != nil || len(matches) != 1 || matches[0].Path != paths[1] {
				t.Fatalf("original query/result identity: %v, %v", matches, err)
			}
			if active == 0 {
				t.Fatal("grants retired before worker work")
			}
			release()
			if active != 0 {
				t.Fatal("grants leaked")
			}
			if err := os.WriteFile(moved[paths[0]], []byte("changed"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := preparer.validate(ctx, items[0], nil, false); err == nil {
				t.Fatal("moved source change missed")
			}
		})
	}
}

func TestWorkerMovedSourceAnalyzer(t *testing.T) {
	// This fixture launches the Go test executable with development assets.
	// Store workers require an installed package's runtime and native broker.
	//goland:noinspection GoBoolExpressions
	if distribution.StoreManaged {
		t.Skip("requires an ordinary worker executable; Store workers need a packaged fixture")
	}
	assets := os.Getenv("PICFETCH_SIMILARITY_ASSETS")
	if assets == "" {
		t.Skip("requires pinned native analysis assets")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	item := cacheFixtureItem(t, "original.jpg")
	policy := cacheTestPolicy(t)
	store := cacheTestStore(t, policy)
	if err := store.write(ctx, item); err != nil {
		t.Fatal(err)
	}
	store.close()
	moved := filepath.Join(t.TempDir(), "moved.jpg")
	if err := os.Rename(item.Path, moved); err != nil {
		t.Fatal(err)
	}
	req := request{Assets: assets, Paths: []string{item.Path}, SourcePaths: workerSourcePaths{item.Path: moved},
		GeneralAnalysisDir: policy.Roots.GeneralDir, GeneralAnalysisLimitBytes: policy.GeneralLimitBytes}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := (Client{}).workerCommand(ctx, executable)
	cmd.Env = append(os.Environ(), workerEnvironment+"=1")
	var final Event
	err = analyzeCommand(ctx, cmd, req, nil, func(event Event) {
		if event.Complete {
			final = event
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if final.Successful != 1 || final.Reused != 1 || final.Failed != 0 || len(final.Items) != 1 || final.Items[0].Path != item.Path {
		t.Fatalf("moved analyzer source lost identity or reuse: %+v", final)
	}
}
