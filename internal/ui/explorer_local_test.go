//go:build explorertrial

package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"encoding/json"
	"image/color"
	"strings"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

// The real worker enforces outbound denial before it reads image pixels. This
// explicit suite fails on absent local assets; it never silently substitutes a
// provider or skips the native integration.
func TestVisualSimilarityExplorerLocal(t *testing.T) {
	t.Run("create_cohort", func(t *testing.T) {
		v := openGridWith(t, "cat-a.png", "cat-b.png")
		pixels, err := os.ReadFile("testdata/explorer/chelsea.png")
		if err != nil {
			t.Fatal(err)
		}
		for i := range v.FileCount() {
			if err := os.WriteFile(v.FileAt(i).Path(), pixels, 0600); err != nil {
				t.Fatal(err)
			}
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		v.explorerAnalyze = (similarity.Client{Assets: assets}).Analyze
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Unassigned (2)"))
		v.grid.SelectAll()
		fynetest.Tap(explorerButton(t, v, "Analyze"))
		top := v.win.Canvas().Overlays().Top()
		if top == nil {
			t.Fatal("actual local tags did not produce a cohort review")
		}
		var create *widget.Button
		foundCat := false
		explorerWalk(top, func(o fyne.CanvasObject) {
			switch control := o.(type) {
			case *widget.Entry:
				control.SetText("Local cats")
			case *widget.Check:
				if control.Text == lang.L("Cat") {
					foundCat = control.Checked
				}
			case *widget.Button:
				if control.Text == lang.L("Create cohort") {
					create = control
				}
			}
		})
		if !foundCat || create == nil || create.Disabled() {
			t.Fatal("real offline Cat tags were not available for cohort creation")
		}
		fynetest.Tap(create)
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatalf("actual Unassigned sources did not form one user cohort: %d piles", len(piles))
		}
		fynetest.Tap(piles[0])
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path()}
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("created native cohort lost an actual source")
		}
	})

	t.Run("granularity", func(t *testing.T) {
		root := t.TempDir()
		var paths []string
		for _, name := range []string{"chelsea.png", "astronaut.png", "coffee.png"} {
			data, err := os.ReadFile(filepath.Join("testdata", "explorer", name))
			if err != nil {
				t.Fatal(err)
			}
			for i := range 8 {
				path := filepath.Join(root, fmt.Sprintf("%s-%d.png", name, i))
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
				paths = append(paths, path)
			}
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		var final similarity.Event
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		if err := (similarity.Client{Assets: assets}).Analyze(ctx, paths, nil, func(e similarity.Event) {
			if e.Complete {
				final = e
			}
		}); err != nil {
			t.Fatal(err)
		}
		groups := map[string]bool{}
		for _, item := range final.Items {
			if item.Cohort != "unassigned" {
				groups[item.Cohort] = true
			}
		}
		if !final.OfflineVerified || final.Successful != len(paths) || len(groups) < 2 {
			t.Fatalf("known-content corpus did not produce multiple complete real cohorts: ready=%d groups=%d", final.Successful, len(groups))
		}
		if len(final.Merges) != len(groups)-1 {
			t.Fatalf("actual worker delivered %d hierarchy edges for %d groups", len(final.Merges), len(groups))
		}
		parents := map[string]string{}
		var find func(string) string
		find = func(s string) string {
			if p, ok := parents[s]; ok {
				return find(p)
			}
			return s
		}
		for _, edge := range final.Merges {
			if !groups[edge.Left] || !groups[edge.Right] || find(edge.Left) == find(edge.Right) {
				t.Fatal("real hierarchy has an unknown, noise or redundant merge")
			}
			parents[find(edge.Right)] = find(edge.Left)
		}
	})

	t.Run("recovery", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		v.explorerAnalyze = (similarity.Client{Assets: t.TempDir()}).Analyze
		explorerMenu(t, v).Action()
		v.settleExplorer()
		failed := false
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Analysis failed. Open the explorer to retry.") {
				failed = true
			}
		})
		if !failed || len(explorerPiles(v)) != 0 {
			t.Fatal("missing native assets did not produce a recoverable visible failure")
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		v.explorerAnalyze = (similarity.Client{Assets: assets}).Analyze
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if piles := explorerPiles(v); len(piles) > 0 {
			fynetest.Tap(piles[0])
		} else {
			fynetest.Tap(explorerButton(t, v, fmt.Sprintf(lang.L("Unassigned (%d)"), 2)))
		}
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path()}
		if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("native retry did not expose the complete fresh cohort")
		}
	})

	t.Run("semantic_tags", func(t *testing.T) {
		v := openGridWith(t, "a-cat.png", "b-blank.jpg", "c-portrait.png", "d-coffee.png", "e-black.jpg", "f-gray.jpg")
		cat, err := os.ReadFile("testdata/explorer/chelsea.png")
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, v.FileCount())
		files := make([]fyne.URI, v.FileCount())
		for i := range paths {
			files[i] = v.FileAt(i)
			paths[i] = files[i].Path()
		}
		if err := os.WriteFile(paths[0], cat, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(paths[1], uitest.EncodeJPEG(t, 128, 128, color.White), 0600); err != nil {
			t.Fatal(err)
		}
		for i, name := range []string{"astronaut.png", "coffee.png"} {
			data, err := os.ReadFile("testdata/explorer/" + name)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(paths[i+2], data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		for i, c := range []color.Color{color.Black, color.Gray{Y: 128}} {
			if err := os.WriteFile(paths[i+4], uitest.EncodeJPEG(t, 128, 128, c), 0600); err != nil {
				t.Fatal(err)
			}
		}
		favorites := t.TempDir()
		if err := favstore.Save(favorites, "Tags", files); err != nil {
			t.Fatal(err)
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		client := similarity.Client{Assets: assets, FavoritesDir: favorites}
		run := func(reused int) {
			t.Helper()
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			var final similarity.Event
			if err := client.Analyze(ctx, paths, nil, func(e similarity.Event) {
				if e.Complete {
					final = e
				}
			}); err != nil {
				t.Fatal(err)
			}
			if !final.OfflineVerified || final.Successful != 6 || final.Failed != 0 || final.Reused != reused {
				t.Fatalf("offline labeled analysis: %+v", final)
			}
			m := final.Measurements
			if m.InferenceAttempts != 6-reused || m.Publications != 1 || m.ElapsedSeconds <= 0 || m.SetupSeconds <= 0 || m.CacheSeconds <= 0 || m.TagSeconds <= 0 || m.GroupingSeconds <= 0 {
				t.Fatalf("production throughput omitted actual work (reused=%d): %+v", reused, m)
			}
			if reused == 0 {
				if m.ModelSeconds <= 0 || m.DecodeSeconds <= 0 || m.EncodeSeconds <= 0 || m.PreviewSeconds <= 0 {
					t.Fatalf("cold analysis omitted source/inference stages: %+v", m)
				}
			} else if m.ModelSeconds != 0 || m.DecodeSeconds != 0 || m.EncodeSeconds != 0 || m.PreviewSeconds != 0 {
				t.Fatalf("warm analysis attributed skipped work to inference: %+v", m)
			}
			if m.ReductionSeconds <= 0 || m.HDBSCANSeconds <= 0 || m.ProjectionSeconds <= 0 || m.HierarchySeconds <= 0 || m.GroupingSeconds < m.ReductionSeconds+m.HDBSCANSeconds+m.ProjectionSeconds+m.HierarchySeconds {
				t.Fatalf("map publication omitted or double-counted grouping stages: %+v", m)
			}
			if m.ElapsedSeconds < m.SetupSeconds+m.ModelSeconds+m.DecodeSeconds+m.EncodeSeconds+m.PreviewSeconds+m.CacheSeconds+m.TagSeconds+m.GroupingSeconds {
				t.Fatalf("stage measurements exceed worker elapsed time: %+v", m)
			}
			if tags := final.Items[0].Tags; !slices.Contains(tags, "cat") || slices.Contains(tags, "dog") {
				t.Fatalf("known cat must have Cat, without Dog: %v", tags)
			}
			if tags := final.Items[2].Tags; !slices.Contains(tags, "person") || !slices.Contains(tags, "portrait") {
				t.Fatalf("known portrait must have Person and Portrait: %v", tags)
			}
			if tags := final.Items[3].Tags; !slices.Contains(tags, "food") {
				t.Fatalf("known coffee must have Food: %v", tags)
			}
			for _, i := range []int{1, 4, 5} {
				if tags := final.Items[i].Tags; len(tags) != 0 {
					t.Fatalf("blank input %d must remain untagged: %v", i, tags)
				}
			}
		}
		run(0)
		if err := os.Chmod(paths[0], 0); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(paths[0], 0600) }()
		run(6)
		v.explorerAnalyze = client.Analyze
		explorerMenu(t, v).Action()
		v.settleExplorer()
		explorerTag(t, v, "Cat", 1)
		explorerTag(t, v, "Untagged", 3)
	})

	t.Run("favorite_cache", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "ordinary.jpg")
		root := t.TempDir()
		if err := favstore.Save(root, "Trip", []fyne.URI{v.FileAt(0), v.FileAt(1)}); err != nil {
			t.Fatal(err)
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		client := similarity.Client{Assets: assets, FavoritesDir: root}
		paths := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
		run := func(reused int) similarity.Event {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			var final similarity.Event
			if err := client.Analyze(ctx, paths, nil, func(e similarity.Event) {
				if e.Complete {
					final = e
				}
			}); err != nil {
				t.Fatal(err)
			}
			if final.Successful != 3 || final.Failed != 0 || final.Reused != reused {
				t.Fatalf("ready=%d failed=%d reused=%d, want 3/0/%d", final.Successful, final.Failed, final.Reused, reused)
			}
			return final
		}
		first := run(0)
		// Cached content remains usable without pixel-read permission. A stat
		// still validates source version; the ordinary source must be scanned.
		if err := os.Chmod(paths[0], 0); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(paths[0], 0600) }()
		second := run(2)
		if second.Items[0].SHA256 != first.Items[0].SHA256 {
			t.Fatal("cache changed source identity")
		}
		if err := os.Chmod(paths[0], 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(paths[1], uitest.EncodeJPEG(t, 96, 80, color.Black), 0600); err != nil {
			t.Fatal(err)
		}
		changed := run(1)
		if changed.Items[1].SHA256 == first.Items[1].SHA256 {
			t.Fatal("changed source reused stale analysis")
		}
		entries, err := os.ReadDir(filepath.Join(root, "Trip", "analysis"))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("analysis cache includes non-members or obsolete versions: %d entries", len(entries))
		}
		for _, entry := range entries {
			if err := os.WriteFile(filepath.Join(root, "Trip", "analysis", entry.Name()), []byte("corrupt"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		run(0)
		if err := favstore.Save(root, "Trip", []fyne.URI{v.FileAt(0)}); err != nil {
			t.Fatal(err)
		}
		run(1)
		client.FavoritesDir = ""
		run(0)
	})
	t.Run("favorite_cache_lifecycle", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg", "g.jpg", "h.jpg")
		paths := make([]string, v.FileCount())
		files := make([]fyne.URI, v.FileCount())
		for i := range paths {
			files[i] = v.FileAt(i)
			paths[i] = files[i].Path()
		}
		root := t.TempDir()
		if err := favstore.Save(root, "Trip", files); err != nil {
			t.Fatal(err)
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		client := similarity.Client{Assets: assets, FavoritesDir: root}
		ctx, cancel := context.WithCancel(context.Background())
		err = client.Analyze(ctx, paths, nil, func(e similarity.Event) {
			if e.Successful >= 2 {
				cancel()
			}
		})
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation: %v", err)
		}
		var final similarity.Event
		renamed := false
		err = client.Analyze(context.Background(), paths, nil, func(e similarity.Event) {
			if !renamed && e.Successful >= 2 {
				renamed = true
				if err := os.Rename(filepath.Join(root, "Trip"), filepath.Join(root, "Moved")); err != nil {
					t.Fatal(err)
				}
			}
			if e.Complete {
				final = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if final.Reused < 2 || final.Successful != 8 {
			t.Fatal("cancellation discarded completed favorite representations")
		}
		if _, err := os.Stat(filepath.Join(root, "Trip")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("analysis recreated a moved favorite")
		}
		entries, err := os.ReadDir(filepath.Join(root, "Moved", "analysis"))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 8 {
			t.Fatal("incremental persistence lost representations during favorite move")
		}
		for _, entry := range entries {
			path := filepath.Join(root, "Moved", "analysis", entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var record map[string]any
			decoder := json.NewDecoder(strings.NewReader(string(data)))
			decoder.UseNumber()
			if err := decoder.Decode(&record); err != nil {
				t.Fatal(err)
			}
			record["Version"] = "different-model-or-preprocessing"
			data, err = json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		err = client.Analyze(context.Background(), paths, nil, func(e similarity.Event) {
			if e.Complete {
				final = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if final.Reused != 0 || final.Successful != 8 {
			t.Fatal("incompatible model cache was reused")
		}
	})
	t.Run("favorite_cache_ui", func(t *testing.T) {
		before := preferences.Load(testApp)
		t.Cleanup(func() { preferences.Save(testApp, before) })
		testApp.Preferences().SetBool("similarityFavoriteCache", true)
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("PICFETCH_SIMILARITY_ASSETS", assets)
		v := openGridWith(t, "a.jpg", "b.jpg")
		root := t.TempDir()
		if err := favstore.Save(root, "Trip", []fyne.URI{v.FileAt(0), v.FileAt(1)}); err != nil {
			t.Fatal(err)
		}
		v.favorites.SetDir(root)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		v.LeaveSimilarityMap()
		v.favorites.Menu().Items[2].Action()
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		reused := false
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if label, ok := o.(*widget.Label); ok && strings.Contains(label.Text, fmt.Sprintf(lang.L("%d reused"), 2)) {
				reused = true
			}
		})
		if !reused {
			t.Fatal("opening a favorite did not reuse and report its saved analysis")
		}
	})
	t.Run("manual_updates", func(t *testing.T) {
		names := make([]string, 80)
		for i := range names {
			names[i] = fmt.Sprintf("%02d.jpg", i)
		}
		v := openGridWith(t, names...)
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, v.FileCount())
		for i := range paths {
			paths[i] = v.FileAt(i).Path()
		}
		controls := make(chan similarity.Control, 1)
		var first, final similarity.Event
		requested := false
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		err = (similarity.Client{Assets: assets}).Analyze(ctx, paths, controls, func(e similarity.Event) {
			if e.Successful == 35 && !requested {
				requested = true
				if err := os.Remove(paths[0]); err != nil {
					t.Fatal(err)
				}
				controls <- similarity.Control{Update: true}
			}
			if len(e.Items) > 0 && !e.Complete && len(first.Items) == 0 {
				first = e
			}
			if e.Complete {
				final = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(first.Items) < 35 || first.Successful >= 80 {
			t.Fatalf("manual mode rebuilt without a request or failed to rebuild during scanning: %d items", len(first.Items))
		}
		if first.Items[0].Path != paths[0] || first.Items[0].Error != "" || len(first.Items[0].Embedding) != 0 {
			t.Fatal("manual update did not reuse the already represented source")
		}
		if final.Successful != 80 || final.Failed != 0 || len(final.Items) != 80 {
			t.Fatal("manual update restarted scanning or lost final accounting")
		}
	})
	t.Run("repeated_source_cohorts", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg", "g.jpg", "h.jpg")
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		var paths []string
		for range 5 {
			for i := range v.FileCount() {
				paths = append(paths, v.FileAt(i).Path())
			}
		}
		var final similarity.Event
		err = (similarity.Client{Assets: assets}).Analyze(context.Background(), paths, nil, func(e similarity.Event) {
			if e.Complete {
				final = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		assignments := map[string]string{}
		for _, item := range final.Items {
			if group, exists := assignments[item.Path]; exists && group != item.Cohort {
				t.Fatal("a repeated opened source belongs to different current cohorts")
			}
			assignments[item.Path] = item.Cohort
		}
		if len(final.Items) != 40 || len(assignments) != 8 {
			t.Fatal("grouping lost source accounting")
		}
		one := v.FileAt(0).Path()
		err = (similarity.Client{Assets: assets}).Analyze(context.Background(), []string{one, one, one, one}, nil, func(e similarity.Event) {
			if e.Complete {
				final = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range final.Items {
			if item.Cohort != "unassigned" {
				t.Fatal("merging one image repeatedly manufactured a density cohort")
			}
		}
	})
	t.Run("small_cohort", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		v.explorerAnalyze = (similarity.Client{Assets: assets}).Analyze
		explorerMenu(t, v).Action()
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatalf("four matching images should form a useful small cohort; got %d piles", len(piles))
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 4 {
			t.Fatal("small real cohort lost its members")
		}
	})
	t.Run("partial_map", func(t *testing.T) {
		names := make([]string, 80)
		for i := range names {
			names[i] = fmt.Sprintf("%02d.jpg", i)
		}
		v := openGridWith(t, names...)
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, v.FileCount())
		for i := range paths {
			paths[i] = v.FileAt(i).Path()
		}
		controls := make(chan similarity.Control, 1)
		controls <- similarity.Control{Automatic: true}
		var partial, final similarity.Event
		partials := 0
		err = (similarity.Client{Assets: assets}).Analyze(context.Background(), paths, controls, func(e similarity.Event) {
			if len(e.Items) > 0 && !e.Complete {
				partials++
				if len(partial.Items) == 0 {
					partial = e
					controls <- similarity.Control{Automatic: false}
				}
			}
			if e.Complete {
				final = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if partials != 1 {
			t.Fatalf("turning off automatic updates allowed %d partial maps", partials)
		}
		if partial.Successful != 30 || len(partial.Items) != 30 || !partial.OfflineVerified {
			t.Fatalf("no real partial map after 30 images: ready=%d items=%d", partial.Successful, len(partial.Items))
		}
		if !final.Complete || final.Successful != 80 || len(final.Items) != 80 {
			t.Fatal("partial publication lost final source accounting")
		}
		if partial.Measurements.Publications != 1 || final.Measurements.Publications != 2 || partial.Measurements.InferenceAttempts != 30 || final.Measurements.InferenceAttempts != 80 || final.Measurements.GroupingSeconds <= partial.Measurements.GroupingSeconds || final.Measurements.EncodeSeconds <= partial.Measurements.EncodeSeconds {
			t.Fatalf("progressive throughput lost cumulative work: partial=%+v final=%+v", partial.Measurements, final.Measurements)
		}
		for _, item := range partial.Items {
			if item.Cohort == "" || len(item.Position) != 2 || len(item.Embedding) != 0 {
				t.Fatal("partial display map must omit vectors and retain grouping/layout")
			}
		}
	})
	t.Run("offline", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		client := similarity.Client{Assets: assets}
		var result similarity.Event
		v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			return client.Analyze(ctx, paths, nil, func(event similarity.Event) {
				if event.Complete {
					result = event
				}
				emit(event)
			})
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !result.Complete || !result.OfflineVerified || result.Successful != 6 || len(result.Items) != 6 {
			t.Fatalf("real offline engine did not deliver all six inputs: complete=%v denied=%v ready=%d", result.Complete, result.OfflineVerified, result.Successful)
		}
		for _, item := range result.Items {
			if len(item.Embedding) != 0 || len(item.Position) != 2 || item.Cohort == "" {
				t.Fatal("real display result must omit inference vectors and retain map assignments")
			}
		}
		piles := explorerPiles(v)
		if len(piles) > 0 {
			fynetest.Tap(piles[0])
		} else {
			explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
				if b, ok := o.(*widget.Button); ok && b.Text == fmt.Sprintf(lang.L("Unassigned (%d)"), 6) {
					fynetest.Tap(b)
				}
			})
		}
		if !v.grid.Visible() || len(explorerGridPaths(v)) == 0 {
			t.Fatal("real completed map cannot open a cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() {
			t.Fatal("real cohort browsing did not return to Grid View")
		}
	})
	t.Run("failed_source", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		missing := v.FileAt(2).Path()
		if err := os.Remove(missing); err != nil {
			t.Fatal(err)
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		var result similarity.Event
		err = (similarity.Client{Assets: assets}).Analyze(context.Background(), []string{v.FileAt(0).Path(), missing}, nil, func(e similarity.Event) {
			if e.Complete {
				result = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Successful != 1 || result.Failed != 1 || len(result.Items) != 2 || result.Items[1].Error == "" || len(result.Items[1].Embedding) != 0 {
			t.Fatal("missing source was not accounted separately")
		}
	})
	t.Run("cancel_worker", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		paths := make([]string, 0, 64)
		for range 64 {
			paths = append(paths, v.FileAt(0).Path())
		}
		completed := false
		progress := false
		err = (similarity.Client{Assets: assets}).Analyze(ctx, paths, nil, func(e similarity.Event) {
			completed = completed || e.Complete
			if e.Successful > 0 {
				progress = true
				cancel()
			}
		})
		if !progress || completed || !errors.Is(err, context.Canceled) {
			t.Fatalf("real worker did not stop after progress: progress=%v complete=%v err=%v", progress, completed, err)
		}
		var retried similarity.Event
		err = (similarity.Client{Assets: assets}).Analyze(context.Background(), []string{v.FileAt(1).Path(), v.FileAt(2).Path()}, nil, func(e similarity.Event) {
			if e.Complete {
				retried = e
			}
		})
		if err != nil || !retried.OfflineVerified || retried.Successful != 2 || retried.Failed != 0 || len(retried.Items) != 2 {
			t.Fatalf("native restart after canceled worker: ready=%d failed=%d items=%d err=%v", retried.Successful, retried.Failed, len(retried.Items), err)
		}
	})
}
