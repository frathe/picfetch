//go:build heicnative && (darwin || linux || windows) && (amd64 || arm64)

package similarity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/filescan"
	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
)

func TestNativeHEICAnalysisPixels(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if runtime.GOOS == "darwin" {
		root = filepath.Join(root, "PicFetch.app")
	}
	build := exec.Command("go", "run", "./scripts/heicpackage", "-os", runtime.GOOS, "-arch", runtime.GOARCH, "-out", root)
	build.Dir = repository
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("stage native analysis helper: %v: %s", buildErr, output)
	}
	marker := filepath.Join(root, "picfetch")
	if runtime.GOOS == "darwin" {
		marker = filepath.Join(root, "Contents", "MacOS", "picfetch")

	}
	if err = os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(marker, []byte("owned package marker"), 0700); err != nil {
		t.Fatal(err)
	}
	owner, err := heicclient.OpenInstalled(context.Background(), marker, filepath.Join(t.TempDir(), "cache"), heicdecode.DefaultLimits(0))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Stop(); owner.Wait() }()
	_ = RegisterLocalFiles()
	reader := imaging.NewReader(owner.Do)
	uri := storage.NewFileURI(filepath.Join(repository, "scripts", "heicbuild", "testdata", "tenbit.heic"))
	admitted, truncated := filescan.Images(context.Background(), []fyne.URI{uri}, 2, nil, filescan.WithAdmission(reader.IsSupportedImage))
	if truncated || len(admitted) != 1 {
		t.Fatal("native analysis source was not admitted")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, search := range []bool{false, true} {
		for _, cancelAfterDelivery := range []bool{false, true} {
			t.Run(fmt.Sprintf("search=%v/cancel=%v", search, cancelAfterDelivery), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, executable, "-test.run=^TestHEICAnalysisHelper$", "-test.timeout=25s")
				cmd.Env = append(os.Environ(), "PICFETCH_OWNED_HEIC_ANALYSIS=pixels")
				req := request{Paths: []string{admitted[0].Path()}}
				client := Client{HEIC: owner}
				seen := 0
				var runErr error
				if search {
					req.Search = &SearchRequest{SessionID: 17, Paths: req.Paths}
					queries := make(chan SearchQuery, 2)
					runErr = client.searchCommand(ctx, cmd, req, queries, func(event SearchEvent) {
						if event.Kind == SearchReady {
							queries <- SearchQuery{ID: 1, ReferencePath: req.Paths[0]}
						} else if event.Kind == SearchFinal {
							seen++
							if seen == 1 {
								queries <- SearchQuery{ID: 2, ReferencePath: req.Paths[0]}
							} else if cancelAfterDelivery {
								cancel()
							}
						}
					})
				} else {
					runErr = client.analyzeCommand(ctx, cmd, req, nil, func(event Event) {
						if event.Complete {
							seen++
							if cancelAfterDelivery {
								cancel()
							}
						}
					})
				}
				want := 1
				if search {
					want = 2
				}
				if seen != want || (!cancelAfterDelivery && runErr != nil) || (cancelAfterDelivery && !errors.Is(runErr, context.Canceled)) {
					t.Fatalf("native pixel deliveries=%d, want %d: %v", seen, want, runErr)
				}
				if cmd.ProcessState == nil {
					t.Fatal("native pixel consumer returned before process join")
				}
			})
		}
	}
}
