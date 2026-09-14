package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
)

// The existing production client supplies a separate cache-reuse baseline. Its
// map work is included in these timings, never reported as search latency.
func profileSearchFavorites(ctx context.Context, config configuration, output io.Writer) error {
	corpus, err := readSearchCorpus(config.Out)
	if err != nil {
		return err
	}
	cache, err := os.MkdirTemp(config.Out, "favorite-baseline-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cache) }()
	files := make([]fyne.URI, len(corpus.Sources))
	paths := make([]string, len(corpus.Sources))
	for i, source := range corpus.Sources {
		paths[i] = filepath.Join(config.Library, filepath.FromSlash(source.Path))
		files[i] = storage.NewFileURI(paths[i])
	}
	if err := favstore.Save(cache, "Search evaluation", files); err != nil {
		return err
	}
	trace, err := os.OpenFile(filepath.Join(config.Out, "favorite-events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = trace.Close() }()
	client := similarity.Client{Assets: config.Assets, FavoritesDir: cache}
	report := struct {
		Scope, ModelRevision, InputSHA256 string
		CacheBytes                        int64
		Passes                            []profilePass
	}{Scope: "Existing Explorer analysis, including map publication. These timings are a Favorite-reuse baseline, not search latency.", ModelRevision: similarity.ModelRevision}
	for _, name := range []string{"cold", "warm"} {
		pass, digest, err := profileAnalysis(ctx, client, paths, false, name, trace, output)
		if err != nil {
			return err
		}
		if name == "cold" {
			report.InputSHA256 = digest
		} else if digest != report.InputSHA256 || pass.Successful != report.Passes[0].Successful || pass.Reused != pass.Successful || pass.Measurements.InferenceAttempts != 0 {
			return fmt.Errorf("favorite baseline sources changed or warm analysis did not reuse all successful sources")
		}
		report.Passes = append(report.Passes, pass)
	}
	if err := filepath.WalkDir(cache, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		report.CacheBytes += info.Size()
		return nil
	}); err != nil {
		return err
	}
	if err := trace.Close(); err != nil {
		return err
	}
	if err := os.RemoveAll(cache); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeJSON(filepath.Join(config.Out, "favorite-profile.json"), report)
}
