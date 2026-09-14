//go:build explorertrial && darwin

package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestProductionSearchSessionGeneralReuse(t *testing.T) {
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	library := t.TempDir()
	paths := make([]string, 5)
	for i := range paths {
		paths[i] = filepath.Join(library, fmt.Sprintf("image-%d.png", i))
		if err := os.WriteFile(paths[i], uitest.EncodePNG(t, 80, 60, color.NRGBA{R: uint8(i * 40), G: 150, B: 70, A: 255}), 0600); err != nil {
			t.Fatal(err)
		}
	}
	client := similarity.Client{Assets: assets}
	cache := similarity.CachePolicy{Roots: similarity.CacheRoots{GeneralDir: t.TempDir()}, LooseEnabled: true, GeneralLimitBytes: 2 * 1024 * 1024}
	for pass := range 2 {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		queries := make(chan similarity.SearchQuery, 1)
		queries <- similarity.SearchQuery{ID: 1, ReferencePath: paths[4]}
		finals, reused := 0, 0
		started := time.Now()
		err := client.Search(ctx, similarity.SearchRequest{SessionID: uint64(pass + 1), Paths: paths, Cache: cache}, queries, func(event similarity.SearchEvent) {
			if event.Kind != similarity.SearchFinal {
				return
			}
			finals++
			reused = event.Reused
			if !event.OfflineVerified || event.Processed != 5 || event.Failed != 0 || len(event.Matches) != 4 {
				t.Errorf("native result = %+v", event)
			}
			for _, match := range event.Matches {
				reference := paths[4]
				if event.QueryID == 2 {
					reference = paths[0]
				}
				if match.Path == reference {
					t.Error("reference returned itself")
				}
			}
			if event.QueryID == 1 {
				queries <- similarity.SearchQuery{ID: 2, ReferencePath: paths[0]}
			} else {
				close(queries)
			}
		})
		cancel()
		if err != nil {
			t.Fatalf("native pass %d: %v", pass, err)
		}
		if finals != 2 || reused != pass*5 {
			t.Fatalf("pass %d finals=%d reused=%d", pass, finals, reused)
		}
		t.Logf("native pass %d: %s, reused %d/5, two references", pass, time.Since(started), reused)
	}
}
