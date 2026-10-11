// Command macstorequalify is a qualification-only main executable. Packaging
// tests put it in a disposable signed app while preserving production workers.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/macbundle"
	"github.com/frathe/picfetch/internal/similarity"
)

func main() {
	denySource := flag.Bool("expect-source-denial", false, "require an ungranted private source to be refused")
	flag.Parse()
	if err := qualify(*denySource); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func qualify(denySource bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root, err := macbundle.CurrentRuntimeDirectory()
	if err != nil {
		return err
	}
	resources := filepath.Join(filepath.Dir(root), "Resources")
	executable := filepath.Join(filepath.Dir(root), "MacOS", "PicFetch")
	decoder := heic.NewClient(executable)
	defer func() { decoder.Stop(); decoder.Wait() }()
	if err := decoder.Check(ctx); err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cache, 0700); err != nil {
		return err
	}
	private, err := os.MkdirTemp(cache, "picfetch-worker-qualification-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(private) }()
	data, err := os.ReadFile(filepath.Join(resources, "fixture.png"))
	if err != nil {
		return err
	}
	path := filepath.Join(private, "cafe\u0301 # %20.png")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	client := similarity.Client{Assets: filepath.Join(resources, "models"), GeneralAnalysisDir: filepath.Join(private, "analysis"), GeneralAnalysisLimitBytes: 16 * 1024 * 1024, DisableFavoriteCache: true}
	var fresh, cached similarity.Event
	paths := []string{path}
	if denySource {
		// A granted bundle source proves worker/model setup still works.
		paths = append(paths, filepath.Join(resources, "fixture.png"))
	}
	if err := client.Analyze(ctx, paths, nil, func(e similarity.Event) { fresh = e }); err != nil {
		return err
	}
	if denySource {
		denied := false
		for _, item := range fresh.Items {
			if item.Path == path && (strings.Contains(item.Error, "operation not permitted") || strings.Contains(item.Error, "permission denied")) {
				denied = true
			}
		}
		if !denied || !fresh.Complete || fresh.Successful != 1 || fresh.Failed != 1 || !fresh.OfflineVerified || fresh.Measurements.InferenceAttempts != 1 {
			return fmt.Errorf("ungranted private source: complete=%t successful=%d failed=%d offline=%t inference=%d", fresh.Complete, fresh.Successful, fresh.Failed, fresh.OfflineVerified, fresh.Measurements.InferenceAttempts)
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"source_denied": true, "event": fresh})
	}
	if err := client.Analyze(ctx, []string{path}, nil, func(e similarity.Event) { cached = e }); err != nil {
		return err
	}
	if !fresh.Complete || fresh.Successful != 1 || fresh.Failed != 0 || !fresh.OfflineVerified || fresh.Measurements.InferenceAttempts != 1 || fresh.CacheWarning != "" {
		return fmt.Errorf("fresh analysis failed: %+v", fresh)
	}
	if !cached.Complete || cached.Successful != 1 || cached.Reused != 1 || !cached.OfflineVerified || cached.CacheWarning != "" {
		return fmt.Errorf("cache replay failed: %+v", cached)
	}
	queries := make(chan similarity.SearchQuery, 1)
	queries <- similarity.SearchQuery{ID: 1, ReferencePath: path}
	var search similarity.SearchEvent
	request := similarity.SearchRequest{SessionID: 1, Paths: []string{path}, Limit: 30, Cache: similarity.CachePolicy{Roots: similarity.CacheRoots{GeneralDir: client.GeneralAnalysisDir}, LooseEnabled: true, GeneralLimitBytes: client.GeneralAnalysisLimitBytes}}
	if err := client.Search(ctx, request, queries, func(event similarity.SearchEvent) {
		if event.Kind == similarity.SearchFinal {
			search = event
			close(queries)
		}
	}); err != nil {
		return err
	}
	if search.Kind != similarity.SearchFinal || search.Failed != 0 || !search.OfflineVerified || search.CacheWarning != "" {
		return fmt.Errorf("search failed: %+v", search)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"heic": true, "fresh": fresh, "cached": cached, "search": search})
}
