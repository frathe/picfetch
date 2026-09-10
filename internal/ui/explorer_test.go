package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
)

func explorerMenu(t *testing.T, v *viewer) *fyne.MenuItem {
	t.Helper()
	for _, menu := range v.win.MainMenu().Items {
		for _, item := range menu.Items {
			if item.Label == lang.L("Visual Similarity Explorer") {
				return item
			}
		}
	}
	t.Fatal("Visual Similarity Explorer is missing from the main menu")
	return nil
}

func explorerFixture(t *testing.T) *viewer { return explorerFixtureScale(t, 1) }

func explorerLargeFixture(t *testing.T) (*viewer, func(int, bool)) {
	t.Helper()
	names := make([]string, 144)
	for i := range names {
		names[i] = fmt.Sprintf("%03d.jpg", i)
	}
	v, publish := streamingExplorerEvents(t, names...)
	v.win.Resize(fyne.NewSize(1100, 700))
	preview := uitest.EncodeJPEG(t, 128, 96, color.White)
	return v, func(count int, complete bool) {
		items := make([]similarity.Item, count)
		var merges []similarity.CohortMerge
		for i := range items {
			group := max(0, i-15)
			items[i] = similarity.Item{Path: v.FileAt(i).Path(), Cohort: fmt.Sprintf("%03d", group), Position: []float32{float32(group % 12), float32(group / 12)}, Preview: preview}
			if group > 0 {
				merges = append(merges, similarity.CohortMerge{Left: "000", Right: items[i].Cohort})
			}
		}
		publish(similarity.Event{Total: v.FileCount(), Successful: count, Complete: complete, Items: items, Merges: merges})
	}
}

func explorerSamples(pile *explorerui.Pile) []string {
	var names []string
	explorerWalk(pile, func(o fyne.CanvasObject) {
		if img, ok := o.(*canvas.Image); ok && img.Resource != nil {
			names = append(names, img.Resource.Name())
		}
	})
	return names
}

func explorerFixtureScale(t *testing.T, factor float32) *viewer {
	t.Helper()
	names := make([]string, 18)
	for i := range names {
		names[i] = fmt.Sprintf("%02d.jpg", i)
	}
	v := openGridWith(t, names...)
	previews := make([][]byte, 18)
	for i := range previews {
		previews[i] = uitest.EncodeJPEG(t, 128, 96, color.NRGBA{R: uint8(50 + i*9), G: uint8(70 + (i*19)%160), B: uint8(90 + (i*31)%150), A: 255})
	}
	v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		var items []similarity.Item
		for i, path := range paths {
			group := "a"
			x := float32(-1)
			if i == 16 {
				group = "b"
				x = 1
			}
			if i == 17 {
				group = "unassigned"
			}
			items = append(items, similarity.Item{Path: path, Cohort: group, Position: []float32{x * factor, 0}, Preview: previews[i]})
		}
		emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
		return nil
	}
	item := explorerMenu(t, v)
	if item.Disabled {
		t.Fatal("explorer unavailable with opened images")
	}
	item.Action()
	v.settleExplorer()
	return v
}

func explorerWalk(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if !o.Visible() {
		return
	}
	visit(o)
	switch o := o.(type) {
	case *fyne.Container:
		for _, child := range o.Objects {
			explorerWalk(child, visit)
		}
	case fyne.Widget:
		for _, child := range fynetest.WidgetRenderer(o).Objects() {
			explorerWalk(child, visit)
		}
	}
}
func explorerPiles(v *viewer) []*explorerui.Pile {
	var piles []*explorerui.Pile
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		if p, ok := o.(*explorerui.Pile); ok {
			piles = append(piles, p)
		}
	})
	return piles
}
func explorerGridPaths(v *viewer) []string {
	var paths []string
	for _, i := range v.grid.ResultIndexes() {
		paths = append(paths, v.FileAt(i).Path())
	}
	return paths
}

func explorerButton(t *testing.T, v *viewer, label string) *widget.Button {
	t.Helper()
	var found *widget.Button
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		if button, ok := o.(*widget.Button); ok && button.Text == lang.L(label) {
			found = button
		}
	})
	if found == nil {
		t.Fatalf("missing visible button %q", label)
	}
	return found
}

func explorerDialogButton(t *testing.T, v *viewer, label string) *widget.Button {
	t.Helper()
	var found *widget.Button
	if top := v.win.Canvas().Overlays().Top(); top != nil {
		explorerWalk(top, func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L(label) {
				found = b
			}
		})
	}
	if found == nil {
		t.Fatalf("missing dialog button %q", label)
	}
	return found
}

func explorerDialogEntry(t *testing.T, v *viewer, placeholder, value string) {
	t.Helper()
	var found *widget.Entry
	if top := v.win.Canvas().Overlays().Top(); top != nil {
		explorerWalk(top, func(o fyne.CanvasObject) {
			if e, ok := o.(*widget.Entry); ok && e.PlaceHolder == lang.L(placeholder) {
				found = e
			}
		})
	}
	if found == nil {
		t.Fatalf("missing dialog field %q", placeholder)
	}
	found.SetText(value)
}

func explorerDialogSelect(t *testing.T, v *viewer, placeholder, value string) {
	t.Helper()
	var found *widget.Select
	explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
		if s, ok := o.(*widget.Select); ok && s.PlaceHolder == lang.L(placeholder) {
			found = s
		}
	})
	if found == nil {
		t.Fatalf("missing dialog choice %q", placeholder)
	}
	found.SetSelected(lang.L(value))
}

func settlePresetUI(v *viewer) {
	for {
		v.explorer.presetWorkers.Wait()
		if !v.explorer.ui.Drain() {
			return
		}
	}
}

func explorerTag(t *testing.T, v *viewer, label string, count int) (*widget.Check, *widget.Hyperlink) {
	t.Helper()
	var found *widget.Check
	var number *widget.Hyperlink
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		row, ok := o.(*fyne.Container)
		if !ok {
			return
		}
		var check *widget.Check
		var link *widget.Hyperlink
		for _, child := range row.Objects {
			switch child := child.(type) {
			case *widget.Check:
				check = child
			case *widget.Hyperlink:
				link = child
			}
		}
		if check != nil && check.Text == lang.L(label) && link != nil && link.Text == fmt.Sprintf("(%d)", count) {
			found, number = check, link
		}
	})
	if found == nil {
		t.Fatalf("missing visible tag %q with clickable count %d", label, count)
	}
	return found, number
}

// A publication is observed only after its provider callback has queued it and
// the owning UI queue has delivered it. The worker remains held between maps.
func streamingExplorer(t *testing.T) (*viewer, func([]string, bool)) {
	t.Helper()
	v, publish := streamingExplorerEvents(t)
	preview := uitest.EncodeJPEG(t, 128, 96, color.White)
	return v, func(groups []string, complete bool) {
		var items []similarity.Item
		for i, group := range groups {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: group, Position: []float32{float32(i * 100), 0}, Preview: preview})
		}
		publish(similarity.Event{Items: items, Successful: len(items), Total: v.FileCount(), Complete: complete, Stage: "encoding"})
	}
}

func streamingExplorerEvents(t *testing.T, names ...string) (*viewer, func(similarity.Event)) {
	t.Helper()
	if len(names) == 0 {
		names = []string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg", "g.jpg", "h.jpg"}
	}
	v := openGridWith(t, names...)
	events := make(chan similarity.Event)
	published := make(chan struct{})
	v.explorerAnalyze = func(ctx context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event := <-events:
				emit(event)
				published <- struct{}{}
				if event.Complete {
					return nil
				}
			}
		}
	}
	explorerMenu(t, v).Action()
	return v, func(event similarity.Event) {
		events <- event
		<-published
		v.explorer.ui.Drain()
	}
}

func TestVisualSimilarityExplorer(t *testing.T) {
	before := preferences.Load(testApp)
	t.Cleanup(func() { preferences.Save(testApp, before) })
	for _, key := range []string{"similarityFavoriteCache", "similarityAutoFit", "similarityAutoUpdate"} {
		testApp.Preferences().RemoveValue(key)
	}
	t.Run("setup_first_use", func(t *testing.T) {
		v := openGridWith(t, "first.jpg")
		v.explorer.introSeen = false
		v.explorer.assetsReady = true
		analyzed := false
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			analyzed = true
			emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true})
			return nil
		}
		v.showExplorer()
		v.settleExplorer()
		if analyzed || v.explorerMapActive() {
			t.Fatal("first use started analysis before the explanation was accepted")
		}
		art := false
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if img, ok := o.(*canvas.Image); ok && img.Resource != nil && img.Resource.Name() == "explorer-intro.png" {
				art = true
			}
		})
		if !art {
			t.Fatal("first-use page does not contain Trane's illustration")
		}
		fynetest.Tap(explorerDialogButton(t, v, "Continue"))
		v.settleExplorer()
		if !analyzed || !v.explorerMapActive() || !preferences.Load(testApp).SimilarityIntroSeen {
			t.Fatal("Continue did not persist acknowledgment and start Explorer")
		}
	})
	t.Run("setup_download_retry_cancel", func(t *testing.T) {
		v := openGridWith(t, "first.jpg")
		v.explorer.introSeen, v.explorer.assetsReady = false, false
		v.explorer.supported = true
		requests := 0
		started := make(chan struct{})
		v.explorer.client = similarity.Client{Assets: filepath.Join(t.TempDir(), "assets"), HTTPClient: &http.Client{Transport: explorerAssetTransport(func(r *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				return nil, fmt.Errorf("connection unavailable")
			}
			close(started)
			<-r.Context().Done()
			return nil, r.Context().Err()
		})}}
		v.showExplorer()
		v.settleExplorer()
		if requests != 0 {
			t.Fatal("reading the explanation made a network request")
		}
		fynetest.Tap(explorerDialogButton(t, v, "Download"))
		v.settleExplorer()
		if requests != 1 || v.explorerMapActive() {
			t.Fatal("failed setup did not remain on the setup page")
		}
		fynetest.Tap(explorerDialogButton(t, v, "Retry"))
		<-started
		fynetest.Tap(explorerDialogButton(t, v, "Cancel"))
		v.settleExplorer()
		if v.explorer.setup != nil || v.explorer.introSeen || v.explorerMapActive() {
			t.Fatal("cancelled setup continued into analysis or left its page open")
		}
		entries, err := os.ReadDir(filepath.Dir(v.explorer.client.Assets))
		if err != nil || len(entries) != 0 {
			t.Fatalf("cancelled download left staged assets: %v, %v", entries, err)
		}
	})
	t.Run("setup_source_change", func(t *testing.T) {
		v := openGridWith(t, "first.jpg")
		v.explorer.introSeen, v.explorer.assetsReady = false, false
		v.explorer.supported = true
		v.explorer.client.Assets = filepath.Join(t.TempDir(), "assets")
		v.showExplorer()
		// Replacement invalidates even an asset check whose UI delivery is pending.
		dropAndWait(t, v, uitest.TempJPEGURI(t, "replacement.jpg", 4, 4, color.White))
		v.settleExplorer()
		if v.explorer.setup != nil || v.win.Canvas().Overlays().Top() != nil || v.explorer.introSeen {
			t.Fatal("source replacement left first-use setup active")
		}
	})
	t.Run("setup_window_size", func(t *testing.T) {
		v := openGridWith(t, "small.jpg")
		v.win.Resize(fyne.NewSize(520, 360))
		v.explorer.introSeen = false
		v.showExplorer()
		v.settleExplorer()
		if size := v.win.Canvas().Size(); size.Width < 700 || size.Height < 620 {
			t.Fatalf("first-use page stayed constrained by the small image window: %v", size)
		}
		for _, size := range []fyne.Size{fyne.NewSize(720, 660), fyne.NewSize(960, 800)} {
			v.win.Resize(size)
			var title *widget.Label
			explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
				if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Visual Similarity Explorer") {
					title = label
				}
			})
			if title == nil {
				t.Fatal("setup page title is missing")
			}
			if position := testApp.Driver().AbsolutePositionForObject(title); position.X > 16 || position.Y > 16 {
				t.Fatalf("setup remains inset behind a dark frame at %v: title at %v", size, position)
			}
		}
		v.win.Resize(fyne.NewSize(720, 660))
		capture := func(name string) {
			if directory := os.Getenv("PICFETCH_EXPLORER_SETUP_QA"); directory != "" {
				var output bytes.Buffer
				if err := png.Encode(&output, v.win.Canvas().Capture()); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, name), output.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
		capture("setup.png")
		v.win.Resize(fyne.NewSize(520, 400))
		button := explorerDialogButton(t, v, "Continue")
		position := testApp.Driver().AbsolutePositionForObject(button)
		if position.Y < 0 || position.Y+button.Size().Height > v.win.Canvas().Size().Height {
			t.Fatalf("resizing hides the action button: position=%v size=%v canvas=%v", position, button.Size(), v.win.Canvas().Size())
		}
		capture("setup-small.png")
	})

	t.Run("keyboard_entry", func(t *testing.T) {
		for _, gridVisible := range []bool{false, true} {
			t.Run(fmt.Sprintf("grid_%t", gridVisible), func(t *testing.T) {
				v := explorerFixture(t)
				v.LeaveSimilarityMap()
				v.settleExplorer()
				if gridVisible {
					v.grid.Toggle()
				}
				beforeSort := v.state.SortMode()
				stubKeyModifiers(t, v, fyne.KeyModifierShift)
				if gridVisible {
					v.grid.HandleRune('/')
					v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyS})
					if v.explorerMapActive() || !v.grid.Searching() {
						t.Fatal("Shift+S interrupted Grid search")
					}
					v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
				}
				modal := dialog.NewInformation("Test", "Keyboard belongs to this dialog", v.win)
				modal.Show()
				v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyS})
				if v.explorerMapActive() {
					t.Fatal("Shift+S opened Explorer behind a dialog")
				}
				modal.Hide()
				v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyS})
				v.settleExplorer()
				if !v.explorerMapActive() || len(explorerPiles(v)) != 2 {
					t.Fatal("Shift+S did not open the Visual Similarity Explorer")
				}
				if v.state.SortMode() != beforeSort {
					t.Fatal("Shift+S changed the ordinary sort order")
				}
				v.LeaveSimilarityMap()
				v.settleExplorer()
				stubKeyModifiers(t, v, 0)
				v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyS})
				waitForSort(t, v)
				if v.state.SortMode() == beforeSort || v.explorerMapActive() {
					t.Fatal("plain S no longer cycles sort order")
				}
			})
		}
	})
	t.Run("trial_launch", func(t *testing.T) {
		v := newTestViewer(t)
		root := filepath.Join(t.TempDir(), "trial")
		session, err := explorertrial.New(root)
		if err != nil {
			t.Fatal(err)
		}
		v.explorer.trial = session
		defer func() { _ = session.Close() }()
		v.applyLaunchOptions(launch.Options{ExplorerTrial: root})
		v.SetCheckForUpdates(true)
		if v.CheckForUpdates() || v.updater.Dir() != filepath.Join(root, "updates") || v.explorer.presets.Dir != filepath.Join(root, "presets") {
			t.Fatal("trial did not isolate storage and disable updates")
		}

		library := t.TempDir()
		pixels := uitest.EncodeJPEG(t, 16, 16, color.White)
		for _, name := range []string{"one.jpg", "two.jpg"} {
			if err := os.WriteFile(filepath.Join(library, name), pixels, 0600); err != nil {
				t.Fatal(err)
			}
		}
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for _, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: "a", Preview: pixels})
			}
			emit(similarity.Event{OfflineVerified: true, Complete: true, Total: len(paths), Successful: len(paths), Items: items})
			return nil
		}
		v.pendingInitial = []fyne.URI{storage.NewFileURI(library)}
		v.openFilesFromOS(nil)
		waitForScan(t, v)
		waitForSort(t, v)
		v.settleExplorer()
		if !v.explorerMapActive() || len(explorerPiles(v)) != 1 {
			t.Fatal("native trial launch did not reach the map through normal scanning")
		}
		v.LeaveSimilarityMap()
		v.settleExplorer()
		if err := session.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("trial_truncated", func(t *testing.T) {
		v := newTestViewer(t)
		root := filepath.Join(t.TempDir(), "trial")
		session, err := explorertrial.New(root)
		if err != nil {
			t.Fatal(err)
		}
		v.explorer.trial = session
		defer func() { _ = session.Close() }()
		limit := 1
		v.applyLaunchOptions(launch.Options{ExplorerTrial: root, MaxFiles: &limit})
		library := t.TempDir()
		pixels := uitest.EncodeJPEG(t, 16, 16, color.White)
		for _, name := range []string{"one.jpg", "two.jpg"} {
			if err := os.WriteFile(filepath.Join(library, name), pixels, 0600); err != nil {
				t.Fatal(err)
			}
		}
		var called atomic.Bool
		v.explorerAnalyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
			called.Store(true)
			return nil
		}
		v.handleDrop([]fyne.URI{storage.NewFileURI(library)})
		waitForScan(t, v)
		waitFor(t, "sort", &v.sortOp.done)
		v.settleExplorer()
		if called.Load() {
			t.Fatal("truncated scan entered native analysis")
		}
		data, err := os.ReadFile(filepath.Join(root, "events.jsonl"))
		if err != nil || !bytes.Contains(data, []byte("scan-truncated")) {
			t.Fatalf("truncation missing from evidence: %s %v", data, err)
		}
	})
	t.Run("trial_recording", func(t *testing.T) {
		v := openGridWith(t, "private-cat.jpg", "private-dog.jpg")
		out := filepath.Join(t.TempDir(), "trial")
		var err error
		v.explorer.trial, err = explorertrial.New(out)
		if err != nil {
			t.Fatal(err)
		}
		preview := uitest.EncodeJPEG(t, 16, 16, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{OfflineVerified: true, Total: 2, Successful: 2, Complete: true,
				Items: []similarity.Item{{Path: paths[0], Cohort: "a", Preview: preview, Facts: similarity.ImageFacts{Make: "Private Camera"}}, {Path: paths[1], Cohort: "a", Preview: preview}}})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerPiles(v)[0])
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.LeaveSimilarityMap()
		if err := v.explorer.trial.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(out, "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		var kinds []string
		var receivedEvent int
		decoder := json.NewDecoder(bytes.NewReader(data))
		for decoder.More() {
			var event struct {
				Kind                      string
				Event                     int
				Total, Successful, Failed int
				Complete, OfflineVerified bool
				Map                       *struct {
					Piles                                              int
					Zoom, MinimumZoom, CenterX, CenterY, Width, Height float32
				}
			}
			if err := decoder.Decode(&event); err != nil {
				t.Fatal(err)
			}
			kinds = append(kinds, event.Kind)
			if event.Kind == "view-observed" || event.Kind == "map-return" {
				if event.Map == nil || event.Map.Piles != 1 || event.Map.MinimumZoom != .03 || event.Map.Zoom <= 0 || event.Map.Width <= 0 || event.Map.Height <= 0 {
					t.Fatalf("trial lacks the displayed map's pile count, zoom and viewport: %+v", event.Map)
				}
			}
			if event.Kind == "cohort-open" && event.Map != nil {
				t.Fatal("trial claimed the covered map was presented")
			}
			if event.Kind == "worker-event" {
				receivedEvent = event.Event
			}
			if event.Kind == "map-applied" && (event.Event == 0 || event.Event != receivedEvent) {
				t.Fatalf("map application is not linked to its received event: received=%d applied=%d", receivedEvent, event.Event)
			}
			if event.Kind == "map-applied" && (event.Total != 2 || event.Successful != 2 || event.Failed != 0 || !event.Complete || !event.OfflineVerified) {
				t.Fatalf("applied map accounting: %+v", event)
			}
		}
		for _, want := range []string{"analysis-started", "worker-event", "map-applied", "worker-exited", "cohort-open", "map-return", "explorer-exit", "session-closed"} {
			if !slices.Contains(kinds, want) {
				t.Fatalf("trial missing %s", want)
			}
		}
		for _, forbidden := range []string{"private-cat", "private-dog", "Private Camera", v.FileAt(0).Path(), `"Preview"`, `"Embedding"`, `"Items"`, `"Facts"`} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Fatalf("trial retained source content %q", forbidden)
			}
		}
	})

	t.Run("trial_recording_frozen_browse", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "private-a.jpg", "private-b.jpg", "private-c.jpg", "private-d.jpg")
		v.LeaveSimilarityMap()
		v.settleExplorer()
		out := filepath.Join(t.TempDir(), "trial")
		var err error
		v.explorer.trial, err = explorertrial.New(out)
		if err != nil {
			t.Fatal(err)
		}
		explorerMenu(t, v).Action()
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		emit := func(n int, complete bool) {
			var items []similarity.Item
			for i := range n {
				items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: "a", Preview: preview})
			}
			publish(similarity.Event{Total: 4, Successful: n, Complete: complete, OfflineVerified: true, Items: items})
		}
		emit(2, false)
		fynetest.Tap(explorerPiles(v)[0])
		frozen := explorerGridPaths(v)
		emit(3, false)
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("publication changed the browsed grid")
		}
		v.grid.SelectAll()
		fireCompareShortcut(v)
		waitForCompare(t, v)
		emit(3, false)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		for _, r := range "/private-a" {
			v.handleTypedRune(r)
		}
		emit(3, false)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		emit(4, true)
		v.settleExplorer()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.LeaveSimilarityMap()
		if err := v.explorer.trial.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(out, "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		var views []struct {
			Kind, Surface, VisibleSHA256 string
			Event, VisibleTotal          int
			Map                          *explorertrial.MapView
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		applied := 0
		for decoder.More() {
			var record struct {
				Kind, Surface, VisibleSHA256 string
				Event, VisibleTotal          int
				Map                          *explorertrial.MapView
			}
			if err := decoder.Decode(&record); err != nil {
				t.Fatal(err)
			}
			if record.Kind == "map-applied" {
				if record.Event <= applied {
					t.Fatal("publications lack distinct event identities")
				}
				applied = record.Event
			}
			if record.Kind == "view-observed" || record.Kind == "cohort-open" || record.Kind == "map-return" {
				if record.Event != applied {
					t.Fatal("view observation is not linked to the applied map")
				}
				views = append(views, record)
			}
		}
		if len(views) != 7 {
			t.Fatalf("expected map/grid/update/comparison/filter/image/return evidence, got %+v", views)
		}
		for i, surface := range []string{"map", "grid", "grid", "comparison", "grid", "image", "map"} {
			if views[i].Surface != surface {
				t.Fatalf("observation %d: surface=%q, want %q", i, views[i].Surface, surface)
			}
			if (views[i].Map != nil) != (surface == "map") {
				t.Fatalf("observation %d confused foreground and covered map geometry", i)
			}
		}
		if views[1].VisibleTotal != 2 || views[1].VisibleSHA256 == "" || views[1].VisibleSHA256 != views[2].VisibleSHA256 || views[2].VisibleTotal != 2 {
			t.Fatalf("frozen grid identity missing from evidence: %+v", views)
		}
		if views[3].VisibleTotal != 0 || views[3].VisibleSHA256 != "" {
			t.Fatal("comparison observation claimed its covered grid was visible")
		}
		if views[4].VisibleTotal != 1 || views[4].VisibleSHA256 == "" || views[4].VisibleSHA256 == views[1].VisibleSHA256 {
			t.Fatal("evidence recorded the saved cohort instead of the filtered grid")
		}
		if views[5].VisibleTotal != 1 || views[5].VisibleSHA256 == "" || views[5].VisibleSHA256 == views[1].VisibleSHA256 {
			t.Fatal("image identity was confused with the whole cohort")
		}
		if views[0].VisibleTotal != 0 || views[6].VisibleSHA256 != "" {
			t.Fatal("map observation claimed a visible grid")
		}
		if bytes.Contains(data, []byte("private-")) {
			t.Fatal("view evidence retained source names")
		}
	})

	t.Run("trial_recording_large_camera", func(t *testing.T) {
		v, publish := explorerLargeFixture(t)
		v.LeaveSimilarityMap()
		v.settleExplorer()
		out := filepath.Join(t.TempDir(), "trial")
		var err error
		v.explorer.trial, err = explorertrial.New(out)
		if err != nil {
			t.Fatal(err)
		}
		explorerMenu(t, v).Action()
		publish(115, false) // 100 piles: one has 16 members.
		publish(116, false) // 101 piles must raise the floor.
		pile := explorerPiles(v)[0]
		viewport := v.explorer.surface.Size()
		pan := fyne.Delta{DX: viewport.Width/2 - pile.Position().X - pile.Size().Width/2 + 21, DY: viewport.Height/2 - pile.Position().Y - pile.Size().Height/2 - 17}
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: pan})
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		position, size := pile.Position(), pile.Size()
		fynetest.Tap(pile)
		frozen := explorerGridPaths(v)
		publish(144, false)
		if len(frozen) != 16 || !slices.Equal(frozen, explorerGridPaths(v)) {
			t.Fatal("large-map publication changed the frozen cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		pile = explorerPiles(v)[0]
		if pile.Position() != position || pile.Size() != size || size.Width != 190 || len(explorerSamples(pile)) != 15 {
			t.Fatal("large-map return lost the camera, floor or full sample set")
		}
		v.LeaveSimilarityMap()
		v.settleExplorer()
		if err := v.explorer.trial.Close(); err == nil {
			t.Fatal("synthetic canceled provider must not qualify as a complete offline trial")
		}
		data, err := os.ReadFile(filepath.Join(out, "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		var maps []explorertrial.Record
		decoder := json.NewDecoder(bytes.NewReader(data))
		for decoder.More() {
			var record explorertrial.Record
			if err := decoder.Decode(&record); err != nil {
				t.Fatal(err)
			}
			if record.Surface == "map" {
				maps = append(maps, record)
			} else if record.Map != nil {
				t.Fatalf("covered map claimed foreground geometry: %+v", record)
			}
		}
		if len(maps) != 4 || maps[2].Kind != "map-departure" || maps[3].Kind != "map-return" {
			t.Fatalf("missing publication/departure/return geometry: %+v", maps)
		}
		for i, count := range []int{100, 101, 101, 129} {
			floor := float32(.5)
			if i == 0 {
				floor = .03
			}
			if maps[i].Map == nil || maps[i].Map.Piles != count || maps[i].Map.MinimumZoom != floor || (i > 0 && maps[i].Map.Zoom != .5) {
				t.Fatalf("map %d has incorrect pile count or zoom geometry: %+v", i, maps[i].Map)
			}
		}
		departure, returned := *maps[2].Map, *maps[3].Map
		if math.Abs(float64(departure.CenterX-(maps[1].Map.CenterX-pan.DX/.5))) > .01 || math.Abs(float64(departure.CenterY-(maps[1].Map.CenterY-pan.DY/.5))) > .01 || departure.Width != viewport.Width || departure.Height != viewport.Height {
			t.Fatal("trial geometry did not measure the actual pan and viewport")
		}
		departure.Piles = returned.Piles // The live map grew behind the frozen grid.
		if departure != returned || maps[2].Event == maps[3].Event {
			t.Fatal("trace lost the camera across distinct map revisions")
		}
	})

	t.Run("trial_recording_cancellation", func(t *testing.T) {
		v := openGridWith(t, "private.jpg")
		out := filepath.Join(t.TempDir(), "trial")
		var err error
		v.explorer.trial, err = explorertrial.New(out)
		if err != nil {
			t.Fatal(err)
		}
		published := make(chan struct{})
		preview := uitest.EncodeJPEG(t, 16, 16, color.White)
		v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{OfflineVerified: true, Complete: true, Total: 1, Successful: 1, Items: []similarity.Item{{Path: paths[0], Cohort: "a", Preview: preview}}})
			close(published)
			<-ctx.Done()
			return ctx.Err()
		}
		explorerMenu(t, v).Action()
		<-published
		before, err := os.ReadFile(filepath.Join(out, "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(before, []byte("worker-event")) || bytes.Contains(before, []byte("map-applied")) {
			t.Fatal("worker receipt was confused with UI application")
		}
		v.LeaveSimilarityMap()
		v.settleExplorer()
		if err := v.explorer.trial.Close(); err == nil {
			t.Fatal("canceled trial reported successful collection")
		}
		after, err := os.ReadFile(filepath.Join(out, "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(after, []byte("map-applied")) || !bytes.Contains(after, []byte(`"Outcome":"canceled"`)) {
			t.Fatal("stale map applied or worker cancellation was lost")
		}
	})

	t.Run("trial_recording_wrong_sources", func(t *testing.T) {
		v := openGridWith(t, "expected.jpg")
		var err error
		v.explorer.trial, err = explorertrial.New(filepath.Join(t.TempDir(), "trial"))
		if err != nil {
			t.Fatal(err)
		}
		preview := uitest.EncodeJPEG(t, 16, 16, color.White)
		v.explorerAnalyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{OfflineVerified: true, Complete: true, Total: 1, Successful: 1, Items: []similarity.Item{{Path: "/wrong-input.jpg", Cohort: "a", Preview: preview}}})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if err := v.explorer.trial.Close(); err == nil {
			t.Fatal("a different source set was reported as complete collection")
		}
	})

	t.Run("presets_create_apply", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "protected.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for i, path := range paths {
				makeName, cohort := "Canon", "unassigned"
				if i == 2 {
					makeName = "Nikon"
				}
				if i == 3 {
					cohort = "protected"
				}
				items = append(items, similarity.Item{Path: path, Cohort: cohort, Preview: preview, Facts: similarity.ImageFacts{Version: 1, Width: 80, Height: 120, Format: "jpg", Make: makeName}})
			}
			emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Canon portraits")
		explorerDialogEntry(t, v, "Camera make", "canon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Canon portraits"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		v.settleExplorer()
		if got := len(explorerPiles(v)); got != 1 {
			t.Fatalf("preview changed current grouping: %d piles", got)
		}
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatalf("preset did not create a cohort: %d", len(piles))
		}
		var named *explorerui.Pile
		for _, pile := range piles {
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if label, ok := o.(*canvas.Text); ok && label.Text == "Canon portraits (2)" {
					named = pile
				}
			})
		}
		if named == nil {
			t.Fatal("named preset cohort missing from surface")
		}
		fynetest.Tap(named)
		if got, want := explorerGridPaths(v), []string{v.FileAt(0).Path(), v.FileAt(1).Path()}; !slices.Equal(got, want) {
			t.Fatalf("preset moved wrong sources: %v", got)
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Canon portraits"))
		fynetest.Tap(explorerDialogButton(t, v, "Delete preset"))
		fynetest.Tap(explorerDialogButton(t, v, "Delete preset"))
		v.settleExplorer()
		found := false
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == "Canon portraits" {
				found = true
			}
		})
		if found {
			t.Fatal("deleted rule remains in the browser")
		}
		fynetest.Tap(explorerDialogButton(t, v, "Close"))
		if len(explorerPiles(v)) != 2 {
			t.Fatal("deleting a preset removed its existing cohort")
		}
	})

	t.Run("presets_image_properties", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "wide.jpg", "small.jpg", "portrait.png")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for i, path := range paths {
				facts := similarity.ImageFacts{Version: 1, Width: 80, Height: 120, Format: "jpg"}
				if i == 2 {
					facts.Width, facts.Height = 120, 80
				}
				if i == 3 {
					facts.Width = 79
				}
				if i == 4 {
					facts.Format = "png"
				}
				items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Preview: preview, Facts: facts})
			}
			emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Exact portraits")
		explorerDialogSelect(t, v, "File type", "jpg")
		explorerDialogSelect(t, v, "Image orientation", "Portrait")
		for _, field := range []struct{ name, value string }{{"Minimum width", "80"}, {"Maximum width", "80"}, {"Minimum height", "120"}, {"Maximum height", "120"}} {
			explorerDialogEntry(t, v, field.name, field.value)
		}
		if explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("valid property-only rule cannot be saved")
		}
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Exact portraits"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatalf("property-only cohort missing: %d", len(piles))
		}
		fynetest.Tap(piles[0])
		if got, want := explorerGridPaths(v), []string{v.FileAt(0).Path(), v.FileAt(1).Path()}; !slices.Equal(got, want) {
			t.Fatalf("property boundaries selected wrong sources: %v", got)
		}
	})

	t.Run("presets_camera_dates_tags", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for i, path := range paths {
				facts := similarity.ImageFacts{Version: 1, Width: 80, Height: 120, Model: "EOS Test", CaptureDate: "2026-09-09"}
				tags := []string{"cat", "animal"}
				if i == 1 {
					facts.CaptureDate = "2026-09-10"
				}
				if i == 2 {
					facts.CaptureDate = "2026-09-08"
				}
				if i == 3 {
					facts.CaptureDate = ""
				}
				if i == 4 {
					facts.Model = "Other"
				}
				if i == 5 {
					tags = []string{"dog"}
				}
				items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Preview: preview, Facts: facts, Tags: tags})
			}
			emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Trip cats")
		explorerDialogEntry(t, v, "Camera model", "eos test")
		explorerDialogEntry(t, v, "Capture date from (YYYY-MM-DD)", "2026-09-09")
		explorerDialogEntry(t, v, "Capture date through (YYYY-MM-DD)", "2026-09-10")
		var cat *widget.Check
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if c, ok := o.(*widget.Check); ok && c.Text == lang.L("Cat") {
				cat = c
			}
		})
		if cat == nil {
			t.Fatal("visual tag rule unavailable")
		}
		cat.SetChecked(true)
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Trip cats"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatalf("combined rule cohort missing: %d", len(piles))
		}
		fynetest.Tap(piles[0])
		if got, want := explorerGridPaths(v), []string{v.FileAt(0).Path(), v.FileAt(1).Path()}; !slices.Equal(got, want) {
			t.Fatalf("camera/date/tag conjunction selected wrong sources: %v", got)
		}
	})

	t.Run("presets_edit_link", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for i, path := range paths {
				model := "EOS2"
				if i == 0 {
					model = "EOS1"
				}
				items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Preview: preview, Facts: similarity.ImageFacts{Version: 1, Make: "Canon", Model: model}})
			}
			emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "My cameras")
		explorerDialogEntry(t, v, "Camera make", "Canon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "My cameras"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "My cameras"))
		explorerDialogEntry(t, v, "Preset name", "My second camera")
		explorerDialogEntry(t, v, "Camera model", "EOS2")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		apply := explorerDialogButton(t, v, "Apply preset")
		if apply.Disabled() {
			t.Fatal("linked cohort edit was not offered for review")
		}
		fynetest.Tap(apply)
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatalf("edit duplicated the linked cohort: %d", len(piles))
		}
		fynetest.Tap(piles[0])
		if got, want := explorerGridPaths(v), []string{v.FileAt(1).Path(), v.FileAt(2).Path()}; !slices.Equal(got, want) {
			t.Fatalf("linked edit retained wrong members: %v", got)
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		fynetest.Tap(explorerButton(t, v, "Unassigned (1)"))
		if got := explorerGridPaths(v); len(got) != 1 || got[0] != v.FileAt(0).Path() {
			t.Fatalf("removed member did not return to Unassigned: %v", got)
		}
	})

	t.Run("presets_analyze_metadata", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for _, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Preview: preview, Facts: similarity.ImageFacts{Version: 1, Make: "Canon"}})
			}
			emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Unassigned (2)"))
		v.grid.SelectAll()
		fynetest.Tap(explorerButton(t, v, "Analyze"))
		fynetest.Tap(explorerDialogButton(t, v, "Save as preset"))
		explorerDialogEntry(t, v, "Preset name", "Metadata group")
		explorerDialogEntry(t, v, "Camera make", "Canon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Metadata group"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		if v.grid.Visible() || len(explorerPiles(v)) != 1 {
			t.Fatal("metadata-only Analyze flow did not return to its created map cohort")
		}
	})

	t.Run("presets_streaming", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		event := func(n int, cohort string, complete bool) similarity.Event {
			var items []similarity.Item
			for i := range n {
				model := "EOS2"
				if i == 0 || i == 3 {
					model = "EOS1"
				}
				items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: cohort, Preview: preview, Facts: similarity.ImageFacts{Version: 1, Make: "Canon", Model: model}})
			}
			return similarity.Event{Total: 4, Successful: n, Items: items, Complete: complete}
		}
		publish(event(3, "unassigned", false))
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Streaming cameras")
		explorerDialogEntry(t, v, "Camera make", "Canon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Streaming cameras"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		settlePresetUI(v)
		publish(event(4, "unassigned", false))
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		settlePresetUI(v)
		_ = explorerButton(t, v, "Unassigned (1)")
		fynetest.Tap(explorerPiles(v)[0])
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
		publish(event(4, "automatic", false))
		if got := explorerGridPaths(v); !slices.Equal(got, want) {
			t.Fatalf("publication changed frozen reviewed cohort: %v", got)
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Streaming cameras"))
		explorerDialogEntry(t, v, "Camera model", "EOS2")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		settlePresetUI(v)
		publish(event(4, "automatic", true))
		fynetest.Tap(explorerButton(t, v, "Unassigned (1)"))
		if got := explorerGridPaths(v); len(got) != 1 || got[0] != v.FileAt(0).Path() {
			t.Fatalf("regrouping undid reviewed removal: %v", got)
		}
	})

	t.Run("presets_stale_preview", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "a.jpg", "b.jpg", "c.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i := range 3 {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: "unassigned", Preview: preview, Facts: similarity.ImageFacts{Version: 1, Make: "Canon"}})
		}
		publish(similarity.Event{Total: 3, Successful: 3, Items: slices.Clone(items)})
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Stale cameras")
		explorerDialogEntry(t, v, "Camera make", "Canon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Stale cameras"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		settlePresetUI(v)
		items[1].Cohort = "automatic"
		publish(similarity.Event{Total: 3, Successful: 3, Items: items, Complete: true})
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		settlePresetUI(v)
		found := false
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if l, ok := o.(*widget.Label); ok && l.Text == lang.L("The images or cohort name changed. Preview the preset again.") {
				found = true
			}
		})
		if !found || len(explorerPiles(v)) != 1 {
			t.Fatal("stale preview changed current cohorts")
		}
	})

	t.Run("presets_pending_members", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		files := slices.Clone(v.state.files)
		dir := t.TempDir()
		if err := favstore.Save(dir, "Pending", files); err != nil {
			t.Fatal(err)
		}
		p, err := v.explorer.presets.Save(context.Background(), explorerpresets.Preset{Name: "Pending cameras", Rule: explorerpresets.Rule{Make: "Canon"}})
		if err != nil {
			t.Fatal(err)
		}
		store, _, err := favstore.OpenCohorts(context.Background(), favstore.Dir(dir, "Pending"))
		if err != nil {
			t.Fatal(err)
		}
		members := []string{files[0].Path(), files[1].Path(), files[2].Path()}
		if err := store.Save(context.Background(), favstore.CohortState{Groups: []favstore.Cohort{{Name: p.Name, PresetID: p.ID, Paths: members}}}); err != nil {
			t.Fatal(err)
		}
		first, release := make(chan struct{}), make(chan struct{})
		pixels := uitest.EncodeJPEG(t, 16, 16, color.White)
		v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for _, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: "automatic", Preview: pixels, Facts: similarity.ImageFacts{Version: 1, Make: "Canon"}})
			}
			emit(similarity.Event{Total: 3, Successful: 1, Items: items[:1]})
			close(first)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-release:
			}
			emit(similarity.Event{Total: 3, Successful: 3, Complete: true, Items: items})
			return nil
		}
		v.favorites.SetDir(dir)
		v.favorites.Open(0)
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		explorerMenu(t, v).Action()
		<-first
		v.explorer.ui.Drain()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Pending cameras"))
		explorerDialogEntry(t, v, "Camera make", "Nikon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		settlePresetUI(v)
		_, saved, err := favstore.OpenCohorts(context.Background(), favstore.Dir(dir, "Pending"))
		if err != nil || len(saved.Groups) != 1 || !slices.Equal(saved.Groups[0].Paths, members[1:]) {
			t.Fatalf("pending members lost on partial review: %+v %v", saved, err)
		}
		close(release)
		v.settleExplorer()
		fynetest.Tap(explorerPiles(v)[0])
		if got := explorerGridPaths(v); !slices.Equal(got, members[1:]) {
			t.Fatalf("late results undid saved pending membership: %v", got)
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Pending cameras"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		settlePresetUI(v)
		if len(explorerPiles(v)) != 0 {
			t.Fatal("empty linked cohort was not dissolved")
		}
		_ = explorerButton(t, v, "Unassigned (3)")
	})
	t.Run("presets_incompatible", func(t *testing.T) {
		v := explorerFixture(t)
		p := explorerpresets.Preset{ID: "older", Name: "Older rule", Rule: explorerpresets.Rule{Make: "Canon", Tags: []string{"retired-tag"}}, Versions: explorerpresets.CurrentVersions()}
		p.Versions.Model = "older"
		data, err := json.Marshal(struct {
			Version int
			Presets []explorerpresets.Preset
		}{1, []explorerpresets.Preset{p}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(v.explorer.presets.Dir, "presets.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Older rule"))
		var retired *widget.Check
		warning := false
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if l, ok := o.(*widget.Label); ok && l.Text == lang.L("This preset uses an older analysis version. Review its rules and save it before applying.") {
				warning = true
			}
			if c, ok := o.(*widget.Check); ok && c.Text == fmt.Sprintf(lang.L("Unavailable tag: %s"), "retired-tag") {
				retired = c
			}
		})
		if !warning || retired == nil || !retired.Checked || !explorerDialogButton(t, v, "Preview preset").Disabled() {
			t.Fatal("incompatible definition was not readable with review guidance")
		}
		explorerDialogEntry(t, v, "Preset name", "Reviewed rule")
		if !explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("edit silently discarded an unsupported condition")
		}
		fynetest.Tap(retired)
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "Reviewed rule"))
		if explorerDialogButton(t, v, "Preview preset").Disabled() {
			t.Fatal("explicitly reviewed rule did not become compatible")
		}
	})
	t.Run("presets_invalid_rules", func(t *testing.T) {
		v := explorerFixture(t)
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Validation")
		if !explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("empty rule accepted")
		}
		explorerDialogEntry(t, v, "Minimum width", "100")
		explorerDialogEntry(t, v, "Maximum width", "99")
		if !explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("inverted range accepted")
		}
		explorerDialogEntry(t, v, "Maximum width", "100")
		if explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("inclusive single-size range rejected")
		}
		explorerDialogEntry(t, v, "Capture date from (YYYY-MM-DD)", "2026-02-29")
		if !explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("invalid date accepted")
		}
		explorerDialogEntry(t, v, "Capture date from (YYYY-MM-DD)", "2026-09-10")
		explorerDialogEntry(t, v, "Capture date through (YYYY-MM-DD)", "2026-09-09")
		if !explorerDialogButton(t, v, "Save preset").Disabled() {
			t.Fatal("inverted capture dates accepted")
		}
		fynetest.Tap(explorerDialogButton(t, v, "Cancel"))
		if err := os.WriteFile(filepath.Join(v.explorer.presets.Dir, "presets.json"), []byte("broken"), 0600); err != nil {
			t.Fatal(err)
		}
		fynetest.Tap(explorerButton(t, v, "Presets"))
		settlePresetUI(v)
		found := false
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if l, ok := o.(*widget.Label); ok && l.Text == lang.L("Could not load presets. The saved library was left unchanged.") {
				found = true
			}
		})
		if !found {
			t.Fatal("corrupt library did not produce visible recovery guidance")
		}
	})

	t.Run("presets_favorite", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		dir := t.TempDir()
		if err := favstore.Save(dir, "Cameras", slices.Clone(v.state.files)); err != nil {
			t.Fatal(err)
		}
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		provider := func(group string) similarity.Provider {
			return func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				var items []similarity.Item
				for i, path := range paths {
					model := "EOS2"
					if i == 0 {
						model = "EOS1"
					}
					items = append(items, similarity.Item{Path: path, Cohort: group, Preview: preview, Facts: similarity.ImageFacts{Version: 1, Make: "Canon", Model: model}})
				}
				emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
				return nil
			}
		}
		open := func(v *viewer, group string) {
			v.explorer.cacheFavorites = false
			v.explorerAnalyze = provider(group)
			v.favorites.SetDir(dir)
			v.favorites.Open(0)
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			explorerMenu(t, v).Action()
			v.settleExplorer()
		}
		open(v, "unassigned")
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "New preset"))
		explorerDialogEntry(t, v, "Preset name", "Saved cameras")
		explorerDialogEntry(t, v, "Camera make", "Canon")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Saved cameras"))
		fynetest.Tap(explorerDialogButton(t, v, "Preview preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		fynetest.Tap(explorerButton(t, v, "Presets"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Saved cameras"))
		explorerDialogEntry(t, v, "Camera model", "EOS2")
		fynetest.Tap(explorerDialogButton(t, v, "Save preset"))
		v.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, v, "Apply preset"))
		v.settleExplorer()
		v.LeaveSimilarityMap()
		v.settleExplorer()
		reopened := newTestViewer(t)
		reopened.explorer.presets = &explorerpresets.Store{Dir: v.explorer.presets.Dir}
		open(reopened, "automatic")
		piles := explorerPiles(reopened)
		var named *explorerui.Pile
		for _, pile := range piles {
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if label, ok := o.(*canvas.Text); ok && label.Text == "Saved cameras (2)" {
					named = pile
				}
			})
		}
		if named == nil {
			t.Fatal("favorite reopen lost the preset cohort and reviewed membership")
		}
		fynetest.Tap(named)
		if got, want := explorerGridPaths(reopened), []string{v.FileAt(1).Path(), v.FileAt(2).Path()}; !slices.Equal(got, want) {
			t.Fatalf("restored preset membership: %v", got)
		}
		fynetest.Tap(explorerButton(t, reopened, "Back to map"))
		fynetest.Tap(explorerButton(t, reopened, "Unassigned (1)"))
		if got := explorerGridPaths(reopened); len(got) != 1 || got[0] != v.FileAt(0).Path() {
			t.Fatal("removed member was reassigned automatically on reopen")
		}
		fynetest.Tap(explorerButton(t, reopened, "Back to map"))
		fynetest.Tap(explorerButton(t, reopened, "Presets"))
		reopened.settleExplorer()
		fynetest.Tap(explorerDialogButton(t, reopened, "Saved cameras"))
		explorerDialogEntry(t, reopened, "Camera model", "")
		fynetest.Tap(explorerDialogButton(t, reopened, "Save preset"))
		reopened.settleExplorer()
		if explorerDialogButton(t, reopened, "Apply preset").Disabled() {
			t.Fatal("reopened preset lost its stable cohort link")
		}

		if err := os.RemoveAll(favstore.Dir(dir, "Cameras")); err != nil {
			t.Fatal(err)
		}
		fynetest.Tap(explorerDialogButton(t, reopened, "Apply preset"))
		settlePresetUI(reopened)
		found := false
		explorerWalk(reopened.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if l, ok := o.(*widget.Label); ok && l.Text == lang.L("Could not save the cohort. Reopen the favorite and try again.") {
				found = true
			}
		})
		if !found {
			t.Fatal("failed preset membership save did not show recovery guidance")
		}
		fynetest.Tap(explorerDialogButton(t, reopened, "Cancel"))
		_ = explorerButton(t, reopened, "Unassigned (1)")
		fynetest.Tap(explorerPiles(reopened)[0])
		if got := explorerGridPaths(reopened); len(got) != 2 {
			t.Fatalf("failed preset save did not restore prior membership: %v", got)
		}
		stored, err := reopened.explorer.presets.Load(context.Background())
		if err != nil || len(stored) != 1 || stored[0].Rule.Model != "" {
			t.Fatal("cohort failure rolled back the independent global rule")
		}
	})

	t.Run("create_cohort_favorite", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		files := slices.Clone(v.state.files)
		dir := t.TempDir()
		for _, name := range []string{"Cats", "Other"} {
			if err := favstore.Save(dir, name, files); err != nil {
				t.Fatal(err)
			}
		}
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		configure := func(v *viewer, group string) {
			v.explorer.cacheFavorites = false
			v.favorites.SetDir(dir)
			v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				var items []similarity.Item
				for _, path := range paths {
					items = append(items, similarity.Item{Path: path, Cohort: group, Tags: []string{"cat"}, Preview: preview})
				}
				emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
				return nil
			}
		}
		open := func(v *viewer, index int) {
			v.favorites.Open(index)
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			explorerMenu(t, v).Action()
			v.settleExplorer()
		}
		configure(v, "unassigned")
		open(v, 0)
		fynetest.Tap(explorerButton(t, v, "Unassigned (3)"))
		v.grid.SelectAll()
		fynetest.Tap(explorerButton(t, v, "Analyze"))
		var create *widget.Button
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if name, ok := o.(*widget.Entry); ok {
				name.SetText("My saved cats")
			}
			if button, ok := o.(*widget.Button); ok && button.Text == lang.L("Create cohort") {
				create = button
			}
		})
		if create == nil {
			t.Fatal("missing creation action")
		}
		fynetest.Tap(create)
		v.settleExplorer()
		v.LeaveSimilarityMap()

		// A new viewer proves persistence rather than reuse of the old map.
		reopened, _, _ := newTestUI(t)
		configure(reopened, "automatic")
		open(reopened, 0)
		var saved *explorerui.Pile
		for _, pile := range explorerPiles(reopened) {
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if label, ok := o.(*canvas.Text); ok && strings.Contains(label.Text, "My saved cats") {
					saved = pile
				}
			})
		}
		if saved == nil {
			t.Fatal("reopening the favorite lost its named cohort")
		}
		fynetest.Tap(saved)
		want := []string{files[0].Path(), files[1].Path(), files[2].Path()}
		slices.Sort(want)
		if !slices.Equal(explorerGridPaths(reopened), want) {
			t.Fatal("restored cohort lost its reviewed source membership")
		}
		open(reopened, 1)
		for _, pile := range explorerPiles(reopened) {
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if label, ok := o.(*canvas.Text); ok && strings.Contains(label.Text, "My saved cats") {
					t.Fatal("cohort leaked to another favorite with identical files")
				}
			})
		}
		configure(reopened, "unassigned")
		open(reopened, 1)
		fynetest.Tap(explorerButton(t, reopened, "Unassigned (3)"))
		reopened.grid.SelectAll()
		fynetest.Tap(explorerButton(t, reopened, "Analyze"))
		var cancel *widget.Button
		explorerWalk(reopened.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if name, ok := o.(*widget.Entry); ok {
				name.SetText("Unsaved cats")
			}
			if button, ok := o.(*widget.Button); ok {
				switch button.Text {
				case lang.L("Create cohort"):
					create = button
				case lang.L("Cancel"):
					cancel = button
				}
			}
		})
		if err := os.RemoveAll(favstore.Dir(dir, "Other")); err != nil {
			t.Fatal(err)
		}
		fynetest.Tap(create)
		reopened.settleExplorer()
		if !reopened.grid.Visible() || reopened.win.Canvas().Overlays().Top() == nil || len(explorerPiles(reopened)) != 0 {
			t.Fatal("failed favorite save appeared to create a cohort")
		}
		explained := false
		explorerWalk(reopened.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Could not save the cohort. Reopen the favorite and try again.") {
				explained = true
			}
		})
		if !explained || cancel == nil {
			t.Fatal("failed favorite save lacks a visible explanation and cancel action")
		}
		fynetest.Tap(cancel)
		fynetest.Tap(explorerButton(t, reopened, "Back to map"))
		_ = explorerButton(t, reopened, "Unassigned (3)")
	})

	t.Run("create_cohort_selection", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "a.jpg", "b.jpg", "c.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i := range v.FileCount() {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview})
		}
		publish(similarity.Event{Total: 3, Successful: 3, Complete: true, Items: items})
		fynetest.Tap(explorerButton(t, v, "Unassigned (3)"))
		analyze := explorerButton(t, v, "Analyze")
		if !analyze.Disabled() {
			t.Fatal("Analyze enabled without an explicit selection")
		}
		var wrap *widget.GridWrap
		explorerWalk(v.grid.Overlay(), func(o fyne.CanvasObject) {
			if w, ok := o.(*widget.GridWrap); ok {
				wrap = w
			}
		})
		if wrap == nil {
			t.Fatal("missing visible Unassigned grid")
		}
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShortcutDefault }
		wrap.Select(0)
		if !analyze.Disabled() {
			t.Fatal("Analyze enabled for a single selected image")
		}
		wrap.Select(1)
		if analyze.Disabled() {
			t.Fatal("Analyze disabled with two selected images")
		}
		v.grid.ClearSelection()
		if !analyze.Disabled() {
			t.Fatal("Analyze stayed enabled after clearing selection")
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		_, tag := explorerTag(t, v, "Cat", 3)
		fynetest.Tap(tag)
		explorerWalk(v.grid.Overlay(), func(o fyne.CanvasObject) {
			if button, ok := o.(*widget.Button); ok && button.Text == lang.L("Analyze") {
				t.Fatal("Analyze leaked into a tag cohort grid")
			}
		})
	})

	t.Run("create_cohort_review", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "a.jpg", "b.jpg", "c.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		items := []similarity.Item{
			{Path: v.FileAt(0).Path(), Cohort: "unassigned", Tags: []string{"cat", "animal"}, Preview: preview},
			{Path: v.FileAt(1).Path(), Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview},
			{Path: v.FileAt(2).Path(), Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview},
		}
		publish(similarity.Event{Total: 3, Successful: 3, Complete: true, Items: items})
		fynetest.Tap(explorerButton(t, v, "Unassigned (3)"))
		wrap := comparisonGridWrap(t, v.grid.Overlay())
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShortcutDefault }
		wrap.Select(0)
		wrap.Select(1)
		v.keyModifiers = func() fyne.KeyModifier { return 0 }
		fynetest.Tap(explorerButton(t, v, "Analyze"))
		top := v.win.Canvas().Overlays().Top()
		if top == nil {
			t.Fatal("Analyze did not open the shared-trait review")
		}
		var name *widget.Entry
		var cat *widget.Check
		var create, cancel *widget.Button
		explorerWalk(top, func(o fyne.CanvasObject) {
			switch c := o.(type) {
			case *widget.Entry:
				name = c
			case *widget.Check:
				if c.Text == lang.L("Include other matching Unassigned images") {
					return
				}
				if c.Text != lang.L("Cat") {
					t.Fatalf("review offered a trait not shared by the selection: %s", c.Text)
				}
				cat = c
			case *widget.Button:
				if c.Text == lang.L("Create cohort") {
					create = c
				}
				if c.Text == lang.L("Cancel") {
					cancel = c
				}
			}
		})
		if name == nil || cat == nil || create == nil || cancel == nil {
			t.Fatal("review lacks shared traits, name or actions")
		}
		if !cat.Checked || !create.Disabled() {
			t.Fatal("review did not select the common trait and require a name")
		}
		name.SetText("My cats")
		if create.Disabled() {
			t.Fatal("named shared-trait cohort cannot be created")
		}
		fynetest.Tap(cat)
		if !create.Disabled() {
			t.Fatal("cohort creation enabled without any shared trait")
		}
		fynetest.Tap(cancel)
		if v.win.Canvas().Overlays().Top() != nil || len(explorerGridPaths(v)) != 3 {
			t.Fatal("cancel changed the Unassigned cohort or left the review open")
		}
	})

	t.Run("create_cohort_membership", func(t *testing.T) {
		for _, selectedOnly := range []bool{false, true} {
			t.Run(fmt.Sprintf("selected_only_%t", selectedOnly), func(t *testing.T) {
				v, publish := streamingExplorerEvents(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				var items []similarity.Item
				for i, tag := range []string{"cat", "cat", "cat", "dog", "cat"} {
					cohort := "unassigned"
					if i == 4 {
						cohort = "existing"
					}
					items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: cohort, Tags: []string{tag}, Preview: preview, Position: []float32{float32(i), 0}})
				}
				publish(similarity.Event{Total: 5, Successful: 5, Items: items})
				catFilter, _ := explorerTag(t, v, "Cat", 4)
				fynetest.Tap(catFilter)
				fynetest.Tap(explorerButton(t, v, "Unassigned (4)"))
				wrap := comparisonGridWrap(t, v.grid.Overlay())
				v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShortcutDefault }
				wrap.Select(0)
				wrap.Select(1)
				v.keyModifiers = func() fyne.KeyModifier { return 0 }
				fynetest.Tap(explorerButton(t, v, "Analyze"))
				top := v.win.Canvas().Overlays().Top()
				if top == nil {
					t.Fatal("missing review")
				}
				var name *widget.Entry
				var create *widget.Button
				var include *widget.Check
				explorerWalk(top, func(o fyne.CanvasObject) {
					switch c := o.(type) {
					case *widget.Entry:
						name = c
					case *widget.Button:
						if c.Text == lang.L("Create cohort") {
							create = c
						}
					case *widget.Check:
						if c.Text == lang.L("Include other matching Unassigned images") {
							include = c
						}
					}
				})
				if name == nil || create == nil || include == nil || !include.Checked {
					t.Fatal("review does not offer all matching Unassigned images")
				}
				if selectedOnly {
					fynetest.Tap(include)
				}
				name.SetText("My cats")
				fynetest.Tap(create)
				if v.win.Canvas().Overlays().Top() != nil || v.grid.Visible() {
					t.Fatal("creation did not return to the map")
				}
				if len(explorerPiles(v)) != 2 {
					t.Fatal("creation lost the existing cohort or failed to add a new one")
				}
				remaining := []string{v.FileAt(3).Path()}
				if selectedOnly {
					remaining = []string{v.FileAt(2).Path(), v.FileAt(3).Path()}
				}
				unassignedLabel := fmt.Sprintf("Unassigned (%d)", len(remaining))
				fynetest.Tap(explorerButton(t, v, unassignedLabel))
				if !slices.Equal(explorerGridPaths(v), remaining) {
					t.Fatal("creation moved unrelated Unassigned images")
				}
				fynetest.Tap(explorerButton(t, v, "Back to map"))
				var created *explorerui.Pile
				for _, pile := range explorerPiles(v) {
					explorerWalk(pile, func(o fyne.CanvasObject) {
						if label, ok := o.(*canvas.Text); ok && strings.Contains(label.Text, "My cats") {
							created = pile
						}
					})
				}
				if created == nil {
					t.Fatal("named cohort is not present on the map")
				}
				fynetest.Tap(created)
				want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
				if selectedOnly {
					want = want[:2]
				}
				if !slices.Equal(explorerGridPaths(v), want) {
					t.Fatal("created cohort did not capture exactly the reviewed matching Unassigned images")
				}
				publish(similarity.Event{Total: 5, Successful: 5, Complete: true, Items: items})
				if !slices.Equal(explorerGridPaths(v), want) {
					t.Fatal("later analysis changed the opened custom cohort")
				}
				fynetest.Tap(explorerButton(t, v, "Back to map"))
				if len(explorerPiles(v)) != 2 {
					t.Fatal("final publication discarded the user-created cohort")
				}
				_ = explorerButton(t, v, unassignedLabel)
			})
		}
	})

	t.Run("create_cohort_no_shared", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "cat.jpg", "dog.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		publish(similarity.Event{Total: 2, Successful: 2, Complete: true, Items: []similarity.Item{
			{Path: v.FileAt(0).Path(), Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview},
			{Path: v.FileAt(1).Path(), Cohort: "unassigned", Tags: []string{"dog"}, Preview: preview},
		}})
		fynetest.Tap(explorerButton(t, v, "Unassigned (2)"))
		v.grid.SelectAll()
		fynetest.Tap(explorerButton(t, v, "Analyze"))
		top := v.win.Canvas().Overlays().Top()
		if top == nil {
			t.Fatal("missing no-shared-traits explanation")
		}
		found := false
		explorerWalk(top, func(o fyne.CanvasObject) {
			if _, ok := o.(*widget.Entry); ok {
				t.Fatal("no-shared-traits review asks for an unusable cohort name")
			}
			if label, ok := o.(*widget.Label); ok && label.Text == lang.L("No shared visual tags found. Select different images.") {
				found = true
			}
			if button, ok := o.(*widget.Button); ok && button.Text == lang.L("Create cohort") && !button.Disabled() {
				t.Fatal("unrelated images can create a shared-trait cohort")
			}
		})
		if !found {
			t.Fatal("no-shared-traits explanation is absent from the dialog")
		}
	})

	t.Run("create_cohort_stale", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t, "a.jpg", "b.jpg")
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		event := similarity.Event{Total: 2, Successful: 2, Complete: true, Items: []similarity.Item{
			{Path: v.FileAt(0).Path(), Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview},
			{Path: v.FileAt(1).Path(), Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview},
		}}
		publish(event)
		fynetest.Tap(explorerButton(t, v, "Unassigned (2)"))
		v.grid.SelectAll()
		fynetest.Tap(explorerButton(t, v, "Analyze"))
		var create *widget.Button
		explorerWalk(v.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if name, ok := o.(*widget.Entry); ok {
				name.SetText("Old proposal")
			}
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Create cohort") {
				create = b
			}
		})
		if create == nil {
			t.Fatal("missing creation action")
		}
		v.LeaveSimilarityMap()
		if v.win.Canvas().Overlays().Top() != nil {
			t.Fatal("leaving Explorer retained its cohort review")
		}
		explorerMenu(t, v).Action()
		publish(event)
		fynetest.Tap(explorerButton(t, v, "Unassigned (2)"))
		create.OnTapped()
		if !v.grid.Visible() || len(explorerPiles(v)) != 0 {
			t.Fatal("stale creation changed a reopened map")
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		_ = explorerButton(t, v, "Unassigned (2)")
	})

	t.Run("source_changes", func(t *testing.T) {
		t.Run("missing_cohort_member", func(t *testing.T) {
			v := explorerFixture(t)
			fynetest.Tap(explorerPiles(v)[0])
			members := explorerGridPaths(v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.preloads.Wait()
			if err := os.Remove(members[len(members)-1]); err != nil {
				t.Fatal(err)
			}
			v.imgCache.Purge()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
			waitUntilLoaded(t, v)
			remaining := members[:len(members)-1]
			if current, _, _ := v.CurrentFile(); !slices.Contains(remaining, current.Path()) {
				t.Fatalf("missing member sent navigation outside its cohort: %s", current.Path())
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), remaining) {
				t.Fatal("missing file remained actionable in the frozen cohort")
			}
		})

		t.Run("remove_after_reorder", func(t *testing.T) {
			v := openGridWith(t, "c.jpg", "a.jpg", "b.jpg")
			want := []string{v.FileAt(2).Path(), v.FileAt(1).Path()}
			preview := uitest.EncodeJPEG(t, 32, 24, color.White)
			queued, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			t.Cleanup(unblock)
			v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				items := []similarity.Item{
					{Path: paths[0], Cohort: "a", Preview: preview},
					{Path: paths[1], Cohort: "a", Preview: preview},
					{Path: paths[2], Cohort: "b", Preview: preview},
				}
				emit(similarity.Event{Items: items, Successful: 3, Total: 3})
				close(queued)
				<-release
				emit(similarity.Event{Items: items, Successful: 3, Total: 3, Complete: true})
				return nil
			}
			explorerMenu(t, v).Action()
			<-queued
			v.explorer.ui.Drain()
			fynetest.Tap(explorerPiles(v)[0])
			members := explorerGridPaths(v)
			moving, moveRelease := make(chan struct{}), make(chan struct{})
			finishMove := sync.OnceFunc(func() { close(moveRelease) })
			t.Cleanup(finishMove)
			uitest.StubTrashMove(t, func(path string) error {
				close(moving)
				<-moveRelease
				return os.Remove(path)
			})
			v.requestDelete()
			v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			<-moving
			v.menus.Actions().Sort()[filesort.ByDropOrder].Action()
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			if got := explorerGridPaths(v); !slices.Equal(got, members) {
				t.Fatalf("reorder changed cohort identities: %v", got)
			}
			finishMove()
			v.deletion.Settle()
			if got := explorerGridPaths(v); !slices.Equal(got, members[1:]) {
				t.Fatalf("deletion after reorder targeted the wrong member: %v", got)
			}
			unblock()
			v.settleExplorer()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			waitUntilLoaded(t, v)
			if current, _, _ := v.CurrentFile(); current.Path() != members[1] {
				t.Fatal("navigation escaped the surviving frozen cohort")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			fynetest.Tap(explorerButton(t, v, "Back to map"))
			if len(explorerPiles(v)) != 0 {
				t.Fatal("removed source survived in the map through a late publication")
			}
			changed := false
			explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
				if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Source files changed. Open the explorer to analyze again.") {
					changed = true
				}
			})
			if !changed || !explorerButton(t, v, "Update map").Disabled() {
				t.Fatal("retired source analysis left misleading feedback or an active update control")
			}
			v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				items := make([]similarity.Item, len(paths))
				for i, path := range paths {
					items[i] = similarity.Item{Path: path, Cohort: "fresh", Preview: preview}
				}
				emit(similarity.Event{Items: items, Successful: len(paths), Total: len(paths), Complete: true})
				return nil
			}
			explorerMenu(t, v).Action()
			v.settleExplorer()
			piles := explorerPiles(v)
			if len(piles) != 1 {
				t.Fatalf("explicit source-change retry produced %d piles", len(piles))
			}
			fynetest.Tap(piles[0])
			if got := explorerGridPaths(v); !slices.Equal(got, want) {
				t.Fatalf("source-change retry did not rebuild surviving original identities: got %v, want %v", got, want)
			}
		})

		for _, destination := range []string{"source", "alias", "unrelated"} {
			t.Run(destination, func(t *testing.T) {
				v := explorerFixture(t)
				fynetest.Tap(explorerPiles(v)[0])
				members := explorerGridPaths(v)
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
				waitUntilLoaded(t, v)
				source, _, _ := v.CurrentFile()
				dest := source
				if destination != "source" {
					dest = storage.NewFileURI(filepath.Join(t.TempDir(), "export.png"))
				}
				if destination == "alias" {
					if err := os.Rename(source.Path(), dest.Path()); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(dest.Path(), source.Path()); err != nil {
						t.Fatal(err)
					}
				}
				entered, release := make(chan struct{}), make(chan struct{})
				unblock := sync.OnceFunc(func() { close(release) })
				t.Cleanup(unblock)
				v.fileWork.export = func(ctx context.Context, dest fyne.URI, pixels image.Image, src fyne.URI, opts imaging.ExportOptions) (imaging.WriteResult, error) {
					result, err := imaging.ExportContext(ctx, dest, pixels, src, opts)
					close(entered)
					<-release
					return result, err
				}
				uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return dest, nil })
				v.rotateBy(1)
				v.exportAs(".png")
				<-entered
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
				waitUntilLoaded(t, v)
				current, _, _ := v.CurrentFile()
				unblock()
				settleChooser(t, v)
				if got, _, _ := v.CurrentFile(); got.String() != current.String() {
					t.Fatal("stale export changed the currently viewed file")
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), members) {
					t.Fatal("committed export changed the frozen browsing cohort")
				}
				fynetest.Tap(explorerButton(t, v, "Back to map"))
				if destination == "unrelated" {
					if len(explorerPiles(v)) != 2 {
						t.Fatal("unrelated export invalidated source analysis")
					}
				} else if len(explorerPiles(v)) != 0 {
					t.Fatal("committed source write retained stale map results")
				}
			})
		}
	})

	t.Run("lifecycle", func(t *testing.T) {
		for _, action := range []string{"exit", "restart", "shutdown"} {
			t.Run(action, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				queued := make(chan struct{})
				v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					emit(similarity.Event{Complete: true, Successful: 1, Total: 2, Items: []similarity.Item{
						{Path: paths[0], Cohort: "old", Preview: preview},
					}})
					close(queued)
					<-ctx.Done()
					return ctx.Err()
				}
				explorerMenu(t, v).Action()
				<-queued
				oldQueue := v.explorer.ui
				if action == "shutdown" {
					lifecycle, ok := testApp.Lifecycle().(interface{ OnStopped() func() })
					if !ok {
						t.Fatal("test app has no stopped hook")
					}
					previous := lifecycle.OnStopped()
					registerShutdown(testApp, v)
					shutdown := lifecycle.OnStopped()
					testApp.Lifecycle().SetOnStopped(previous)
					shutdown()
				} else {
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				}
				v.explorer.workers.Wait()
				v.explorer.ui = &uitest.UIQueue{}
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					emit(similarity.Event{Complete: true, Successful: 1, Total: 2, Items: []similarity.Item{
						{Path: paths[1], Cohort: "new", Preview: preview},
					}})
					return nil
				}
				if action != "exit" {
					explorerMenu(t, v).Action()
				}
				v.settleExplorer()
				oldQueue.Drain()
				piles := explorerPiles(v)
				if action != "restart" {
					if len(piles) != 0 || v.explorer.surface.Visible() || v.FileCount() != 2 {
						t.Fatal("retired analysis mutated or reopened the viewer")
					}
					return
				}
				if len(piles) != 1 {
					t.Fatalf("restarted map has %d piles", len(piles))
				}
				fynetest.Tap(piles[0])
				if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(1).Path()}) {
					t.Fatalf("old queued result replaced the new cohort: %v", got)
				}
			})
		}
	})

	t.Run("recovery", func(t *testing.T) {
		for _, failure := range []string{"setup", "after_partial", "after_final", "incomplete"} {
			t.Run(failure, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					if failure != "setup" {
						emit(similarity.Event{Complete: failure == "after_final", Successful: 1, Total: 2, Items: []similarity.Item{
							{Path: paths[0], Cohort: "failed", Preview: preview},
						}})
					}
					if failure == "incomplete" {
						return nil
					}
					return io.ErrUnexpectedEOF
				}
				explorerMenu(t, v).Action()
				v.settleExplorer()
				failed := false
				explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
					if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Analysis failed. Open the explorer to retry.") {
						failed = true
					}
				})
				if !failed {
					t.Fatal("analysis failure has no visible recovery feedback")
				}
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					emit(similarity.Event{Complete: true, Successful: 1, Failed: 1, Total: 2, Items: []similarity.Item{
						{Path: paths[1], Cohort: "retried", Preview: preview},
						{Path: paths[0], Error: "unreadable"},
					}})
					return nil
				}
				explorerMenu(t, v).Action()
				v.settleExplorer()
				piles := explorerPiles(v)
				if len(piles) != 1 {
					t.Fatalf("retry produced %d piles, want one readable cohort", len(piles))
				}
				fynetest.Tap(piles[0])
				if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(1).Path()}) {
					t.Fatalf("retry retained failed analysis: %v", got)
				}
			})
		}
	})

	t.Run("granularity", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i, group := range []string{"a", "a", "b", "b", "c", "unassigned"} {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: group, Tags: []string{"bird"}, Preview: preview, Position: []float32{float32(i), 0}})
		}
		merges := []similarity.CohortMerge{{Left: "a", Right: "b"}, {Left: "b", Right: "c"}}
		publish(similarity.Event{Items: items, Merges: merges, Successful: 6, Total: 8})
		var slider *widget.Slider
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if s, ok := o.(*widget.Slider); ok {
				slider = s
			}
		})
		if slider == nil {
			t.Fatal("map is missing its granularity slider")
		}
		if len(explorerPiles(v)) != 3 {
			t.Fatal("default granularity changed the original cohorts")
		}
		for _, x := range []float32{0, slider.Size().Width, slider.Size().Width / 2} {
			slider.Dragged(&fyne.DragEvent{Position: fyne.NewPos(x, slider.Size().Height/2)})
			if len(explorerPiles(v)) != 3 {
				t.Fatal("dragging granularity rebuilt the map before release")
			}
		}
		if slider.Value != 50 {
			t.Fatal("granularity thumb did not follow the drag")
		}
		publish(similarity.Event{Items: items, Merges: merges, Successful: 6, Total: 8})
		if len(explorerPiles(v)) != 3 || slider.Value != 50 {
			t.Fatal("analysis publication applied unfinished granularity or moved the thumb")
		}
		slider.DragEnd()
		if len(explorerPiles(v)) != 2 || v.win.Canvas().Focused() != nil {
			t.Fatal("broader granularity did not combine the nearest cohorts or release focus")
		}
		fynetest.Tap(explorerPiles(v)[0])
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path(), v.FileAt(3).Path()}
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatalf("broader cohort membership: %v", explorerGridPaths(v))
		}
		items = append(items, similarity.Item{Path: v.FileAt(6).Path(), Cohort: "a", Tags: []string{"bird"}, Preview: preview})
		publish(similarity.Event{Items: items, Merges: merges, Successful: 7, Total: 8, Complete: true})
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("new hierarchy changed the open grid")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if len(explorerPiles(v)) != 2 || slider.Value != 50 {
			t.Fatal("publication or grid return reset granularity")
		}
		explorerTag(t, v, "Bird", 7)
		fynetest.TapAt(slider, fyne.NewPos(0, slider.Size().Height/2))
		if len(explorerPiles(v)) != 1 {
			t.Fatal("broadest granularity did not join the assigned groups")
		}
		fynetest.Tap(explorerButton(t, v, "Unassigned (1)"))
		if !slices.Equal(explorerGridPaths(v), []string{v.FileAt(5).Path()}) {
			t.Fatal("granularity moved unassigned images into a cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		slider.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		if len(explorerPiles(v)) != 2 {
			t.Fatal("keyboard granularity did not apply its completed change")
		}
		fynetest.TapAt(slider, fyne.NewPos(slider.Size().Width, slider.Size().Height/2))
		if len(explorerPiles(v)) != 3 {
			t.Fatal("fine granularity did not restore the original groups")
		}
		fynetest.TapAt(slider, fyne.NewPos(0, slider.Size().Height/2))
		v.LeaveSimilarityMap()
		v.explorerAnalyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{Items: items, Merges: merges, Successful: 7, Total: 8, Complete: true})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if len(explorerPiles(v)) != 3 || slider.Value != 100 {
			t.Fatal("reopening the explorer retained the previous applied granularity")
		}
	})

	t.Run("granularity_rearranges", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		previous := v.settingsState()
		next := previous
		next.SimilarityAutoFit = false
		v.ApplySettings(previous, next)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i, position := range [][]float32{{-2, 0}, {-1, 1}, {1, -1}, {2, 0}} {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: fmt.Sprint(i), Position: position, Preview: preview})
		}
		event := similarity.Event{Items: items, Successful: 4, Total: 8, Merges: []similarity.CohortMerge{
			{Left: "0", Right: "1"}, {Left: "1", Right: "2"}, {Left: "2", Right: "3"},
		}}
		publish(event)
		var slider *widget.Slider
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if s, ok := o.(*widget.Slider); ok {
				slider = s
			}
		})
		if slider == nil {
			t.Fatal("missing granularity control")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 75, DY: -35}})
		piles := explorerPiles(v)
		width := piles[0].Size().Width
		offsets := make([]fyne.Position, len(piles))
		for i, pile := range piles {
			offsets[i] = pile.Position().Subtract(piles[0].Position())
		}
		slider.SetValue(0)
		piles = explorerPiles(v)
		if len(piles) != 1 || piles[0].Size().Width != width {
			t.Fatal("coarsening must retain the zoom and merge the four groups")
		}
		center := piles[0].Position().Add(fyne.NewPos(piles[0].Size().Width/2, piles[0].Size().Height/2))
		viewport := v.explorer.surface.Size()
		if math.Abs(float64(center.X-viewport.Width/2)) > .01 || math.Abs(float64(center.Y-viewport.Height/2)) > .01 {
			t.Fatalf("single merged cohort retained an old corner instead of centering: %v", center)
		}
		slider.SetValue(100)
		piles = explorerPiles(v)
		if len(piles) != len(offsets) || piles[0].Size().Width != width {
			t.Fatal("refining must restore all groups at the user's zoom")
		}
		positions := make([]fyne.Position, len(piles))
		for i, pile := range piles {
			positions[i] = pile.Position()
			got := positions[i].Subtract(piles[0].Position())
			if math.Abs(float64(got.X-offsets[i].X)) > .01 || math.Abs(float64(got.Y-offsets[i].Y)) > .01 {
				t.Fatal("restored fine layout depends on the intervening merged layout")
			}
		}
		publish(event)
		for i, pile := range explorerPiles(v) {
			if pile.Position() != positions[i] || pile.Size().Width != width {
				t.Fatal("background publication rearranged the freshly chosen layout")
			}
		}
	})

	t.Run("keyboard", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i, position := range [][]float32{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: fmt.Sprint(i), Position: position, Tags: []string{"bird"}, Preview: preview})
		}
		publish(similarity.Event{Items: items, Successful: 4, Total: 8})
		piles := explorerPiles(v)
		for _, label := range []string{"+", "-", "Fit map"} {
			fynetest.Tap(explorerButton(t, v, label))
			if v.win.Canvas().Focused() != nil {
				t.Fatalf("%s captured keyboard focus instead of returning it to the map", label)
			}
		}
		width := piles[0].Size().Width
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		if piles[0].Size().Width <= width {
			t.Fatal("keyboard + did not zoom the map")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		if math.Abs(float64(piles[0].Size().Width-width)) > .01 {
			t.Fatal("keyboard - did not undo zoom")
		}
		highlight := func(want *explorerui.Pile) {
			t.Helper()
			for _, pile := range explorerPiles(v) {
				visible := false
				explorerWalk(pile, func(o fyne.CanvasObject) {
					if frame, ok := o.(*canvas.Rectangle); ok && frame.StrokeWidth >= 2 && frame.Size().Width > 0 {
						visible = true
					}
				})
				if visible != (pile == want) {
					t.Fatal("arrow navigation did not visibly highlight exactly the active stack")
				}
			}
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		highlight(piles[0])
		for _, step := range []struct {
			key   fyne.KeyName
			index int
		}{{fyne.KeyRight, 1}, {fyne.KeyDown, 3}, {fyne.KeyLeft, 2}, {fyne.KeyUp, 0}} {
			v.handleKeyEvent(&fyne.KeyEvent{Name: step.key})
			highlight(piles[step.index])
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), []string{v.FileAt(0).Path()}) {
			t.Fatal("Enter did not open the highlighted stack")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		highlight(piles[0])
		items[0].Cohort = "renamed"
		publish(similarity.Event{Items: items, Successful: 4, Total: 8, Complete: true})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if !slices.Equal(explorerGridPaths(v), []string{v.FileAt(0).Path()}) {
			t.Fatal("publication lost the selected source identity")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		fynetest.Tap(explorerButton(t, v, "Clear tags"))
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if v.grid.Visible() {
			t.Fatal("keyboard opened a filtered-out stack")
		}
	})

	t.Run("navigation_preview_reuse", func(t *testing.T) {
		names := make([]string, 30)
		for i := range names {
			names[i] = fmt.Sprintf("%02d.jpg", i)
		}
		v, publish := streamingExplorerEvents(t, names...)
		v.win.Resize(fyne.NewSize(1100, 700))
		items := make([]similarity.Item, len(names))
		for i := range items {
			items[i] = similarity.Item{
				Path: v.FileAt(i).Path(), Cohort: fmt.Sprint(i / 15), Position: []float32{float32(i / 15), 0},
				Preview: uitest.EncodeJPEG(t, 160, 120, color.NRGBA{R: uint8(40 + i*6), G: 120, B: 180, A: 255}),
			}
		}
		publish(similarity.Event{Items: items, Successful: len(items), Total: len(items), Complete: true})
		v.settleExplorer()
		for _, key := range []fyne.KeyName{fyne.KeyRight, fyne.KeyRight, fyne.KeyLeft} {
			v.handleKeyEvent(&fyne.KeyEvent{Name: key})
		}
		piles := explorerPiles(v)
		if len(piles) != 2 || len(explorerSamples(piles[0])) != 15 || len(explorerSamples(piles[1])) != 15 {
			t.Fatal("navigation fixture must display two complete 15-sample piles")
		}
		snapshot := func() []byte {
			var b bytes.Buffer
			if err := png.Encode(&b, v.win.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			return b.Bytes()
		}
		before := snapshot()
		var start, end runtime.MemStats
		runtime.ReadMemStats(&start)
		for range 20 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
		}
		runtime.ReadMemStats(&end)
		allocated := end.TotalAlloc - start.TotalAlloc
		t.Logf("40 selections allocated %.2f MiB", float64(allocated)/(1<<20))
		if allocated > 8<<20 {
			t.Errorf("selection allocated %.2f MiB for unchanged previews; want under 8 MiB", float64(allocated)/(1<<20))
		}
		if !bytes.Equal(before, snapshot()) {
			t.Fatal("returning selection to its starting pile changed the rendered map")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if len(explorerGridPaths(v)) != 15 {
			t.Fatal("keyboard selection lost the complete cohort")
		}
	})

	t.Run("viewport_previews", func(t *testing.T) {
		names := make([]string, 300)
		for i := range names {
			names[i] = fmt.Sprintf("%03d.jpg", i)
		}
		v, publish := streamingExplorerEvents(t, names...)
		v.win.Resize(fyne.NewSize(1100, 700))
		items := make([]similarity.Item, len(names))
		for i := range items {
			group := i / 15
			items[i] = similarity.Item{
				Path: v.FileAt(i).Path(), Cohort: fmt.Sprintf("%02d", group), Position: []float32{float32(group % 5), float32(group / 5)},
				Preview: uitest.EncodeJPEG(t, 160, 120, color.NRGBA{R: uint8(i), G: 120, B: 180, A: 255}),
			}
		}
		publish(similarity.Event{Items: items, Successful: len(items), Total: len(items), Complete: true})
		v.settleExplorer()
		snapshot := func() []byte {
			var b bytes.Buffer
			if err := png.Encode(&b, v.win.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			return b.Bytes()
		}
		before := snapshot()
		allocated := func() uint64 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return m.HeapAlloc
		}
		loaded := allocated()
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 10000, DY: 10000}})
		after := allocated()
		t.Logf("loaded heap %.2f MiB; away %.2f MiB", float64(loaded)/(1<<20), float64(after)/(1<<20))
		if after+4<<20 > loaded {
			t.Errorf("panning away did not release at least 4 MiB of decoded previews")
		}
		fynetest.Tap(explorerButton(t, v, "Fit map"))
		piles := explorerPiles(v)
		if len(piles) != 20 {
			t.Fatalf("viewport eviction lost cohorts: got %d", len(piles))
		}
		for _, pile := range piles {
			if len(explorerSamples(pile)) != 15 {
				t.Fatal("returning to the map did not restore every sample")
			}
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Image == nil {
					t.Fatal("returning to the viewport left a preview waiting for pixels")
				}
			})
		}
		if !bytes.Equal(before, snapshot()) {
			t.Fatal("viewport round trip changed the map's rendered appearance")
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 15 {
			t.Fatal("viewport eviction lost full cohort membership")
		}
	})

	t.Run("viewport_margin", func(t *testing.T) {
		v := explorerFixture(t)
		pile := explorerPiles(v)[0]
		// Start with a decoded pile one pile-width beyond the left viewport
		// edge, then cross that preparation boundary repeatedly.
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: -2*pile.Size().Width - pile.Position().X}})
		var start, end runtime.MemStats
		runtime.ReadMemStats(&start)
		for range 20 {
			v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: -1}})
			v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 1}})
		}
		runtime.ReadMemStats(&end)
		allocated := end.TotalAlloc - start.TotalAlloc
		if allocated > 1<<20 {
			t.Fatalf("small reversals at the viewport margin churned %.2f MiB of previews", float64(allocated)/(1<<20))
		}
	})

	t.Run("tags_collapse", func(t *testing.T) {
		v := explorerFixture(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		v.ForceRepaint()
		before := v.explorer.surface.Size()
		piles := explorerPiles(v)
		fynetest.Tap(explorerButton(t, v, "Hide tags"))
		v.ForceRepaint()
		if v.explorer.surface.Size().Width <= before.Width {
			t.Fatal("collapsing tags did not give their width back to the map")
		}
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Clear tags") {
				t.Fatal("collapsed tags still expose sidebar controls")
			}
		})
		fynetest.Tap(explorerButton(t, v, "Show tags"))
		v.ForceRepaint()
		if v.explorer.surface.Size() != before || !slices.Equal(explorerPiles(v), piles) || v.win.Canvas().Focused() != nil {
			t.Fatal("restoring tags changed map contents, geometry or input focus")
		}
	})

	t.Run("expanded_tag_catalogue", func(t *testing.T) {
		tags := []struct{ id, label string }{
			{"costume", "Costume"},
			{"traditional_clothing", "Traditional dress"},
			{"clothing", "Clothing"},
			{"jewelry", "Jewelry"},
			{"train", "Train"},
			{"bus", "Bus"},
			{"truck", "Truck"},
			{"tram", "Tram"},
			{"fish", "Fish"},
			{"reptile", "Reptile"},
			{"castle", "Castle"},
			{"church", "Church"},
			{"temple", "Temple"},
			{"tower", "Tower"},
			{"ruins", "Ruins"},
			{"street", "Street"},
			{"park", "Park"},
			{"garden", "Garden"},
			{"waterfall", "Waterfall"},
			{"desert", "Desert"},
			{"countryside", "Countryside"},
			{"cave", "Cave"},
			{"furniture", "Furniture"},
			{"musical_instrument", "Musical instrument"},
			{"toy", "Toy"},
			{"sculpture", "Sculpture"},
			{"painting", "Painting"},
			{"drawing", "Drawing"},
			{"camera", "Camera"},
			{"book", "Book"},
			{"sign", "Sign"},
			{"computer", "Computer"},
			{"sports", "Sports"},
			{"concert", "Concert"},
			{"festival", "Festival"},
			{"wedding", "Wedding"},
			{"dance", "Dance"},
			{"hiking", "Hiking"},
			{"camping", "Camping"},
			{"swimming", "Swimming"},
			{"skiing", "Skiing"},
			{"drink", "Drink"},
			{"fruit", "Fruit"},
			{"dessert", "Dessert"},
		}
		names := make([]string, len(tags))
		for i := range names {
			names[i] = fmt.Sprintf("%02d.jpg", i)
		}
		v, publish := streamingExplorerEvents(t, names...)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		items := make([]similarity.Item, len(tags))
		for i, tag := range tags {
			items[i] = similarity.Item{Path: v.FileAt(i).Path(), Cohort: "subjects", Tags: []string{tag.id}, Preview: preview}
		}
		publish(similarity.Event{Items: items, Total: len(items), Successful: len(items), Complete: true})
		for i, tag := range tags {
			if !slices.Contains(similarity.TagIDs(), tag.id) {
				t.Fatalf("new subject %q is missing from the model catalogue", tag.id)
			}
			check, number := explorerTag(t, v, tag.label, 1)
			if !check.Checked {
				t.Fatalf("new subject %q starts excluded", tag.label)
			}
			fynetest.TapAt(number, fyne.NewPos(number.MinSize().Width/2, number.MinSize().Height/2))
			if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(i).Path()}) {
				t.Fatalf("subject %q opened the wrong sources", tag.label)
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
	})

	t.Run("cohort_subject_titles", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		tags := [][]string{{"forest", "mountain"}, {"forest", "mountain"}, {"forest", "cat"}, {"bird", "bird", "bird"}, nil, nil, {"unknown"}, {"forest", "mountain"}}
		var items []similarity.Item
		for i, group := range []string{"a", "a", "a", "b", "b", "b", "c", "d"} {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: group, Tags: tags[i], Preview: preview})
		}
		publish(similarity.Event{Items: items, Successful: 8, Total: 8, Merges: []similarity.CohortMerge{
			{Left: "a", Right: "b"}, {Left: "b", Right: "c"}, {Left: "c", Right: "d"},
		}})
		title := func(pile *explorerui.Pile) string {
			var text string
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if label, ok := o.(*canvas.Text); ok {
					text = label.Text
				}
			})
			return text
		}
		for i, want := range []string{"Forest / Mountain (3)", "3 images", "1 image", "Forest / Mountain (1)"} {
			if got := title(explorerPiles(v)[i]); got != want {
				t.Fatalf("cohort %d title = %q, want %q", i, got, want)
			}
		}
		fynetest.Tap(explorerPiles(v)[0])
		if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}) {
			t.Fatal("subject title changed the cohort's membership")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if slider, ok := o.(*widget.Slider); ok {
				slider.SetValue(0)
			}
		})
		if piles := explorerPiles(v); len(piles) != 1 || title(piles[0]) != "Forest (8)" {
			t.Fatal("merged cohort title did not reflect its new shared subjects")
		}
	})

	t.Run("tags_browse", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		items := []similarity.Item{
			{Path: v.FileAt(0).Path(), Cohort: "a", Tags: []string{"bird", "bird"}, Preview: preview},
			{Path: v.FileAt(1).Path(), Cohort: "a", Tags: []string{"dog"}, Preview: preview},
			{Path: v.FileAt(2).Path(), Cohort: "b", Tags: []string{"bird"}, Preview: preview},
			{Path: v.FileAt(3).Path(), Cohort: "unassigned", Tags: []string{"bird"}, Preview: preview},
		}
		items = append(items, items[0])
		publish(similarity.Event{Items: items, Successful: 4, Total: 8})
		check, number := explorerTag(t, v, "Bird", 3)
		fynetest.Tap(check)
		before := explorerPiles(v)[0].Position()
		fynetest.TapAt(number, fyne.NewPos(number.MinSize().Width/2, number.MinSize().Height/2))
		want := []string{v.FileAt(0).Path(), v.FileAt(2).Path(), v.FileAt(3).Path()}
		if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), want) {
			t.Fatalf("tag link did not open exactly the matching distinct images: %v", explorerGridPaths(v))
		}
		items = append(items, similarity.Item{Path: v.FileAt(4).Path(), Cohort: "b", Tags: []string{"bird"}, Preview: preview})
		publish(similarity.Event{Items: items, Successful: 5, Total: 8, Complete: true})
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("publication changed the tag grid being browsed")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		check, number = explorerTag(t, v, "Bird", 4)
		if check.Checked || explorerPiles(v)[0].Position() != before {
			t.Fatal("tag grid return lost filtering or map position")
		}
		fynetest.TapAt(number, fyne.NewPos(number.MinSize().Width/2, number.MinSize().Height/2))
		want = append(want, v.FileAt(4).Path())
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("reopened tag grid did not include new matches")
		}
	})

	t.Run("tags_filter", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg")
		paths := make([]string, v.FileCount())
		for i := range paths {
			paths[i] = v.FileAt(i).Path()
		}
		preview := uitest.EncodeJPEG(t, 128, 96, color.White)
		v.explorerAnalyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			items := []similarity.Item{
				{Path: paths[0], Cohort: "a", Tags: []string{"bird", "dog", "bird"}},
				{Path: paths[1], Cohort: "a", Tags: []string{"bird"}},
				{Path: paths[2], Cohort: "b", Tags: []string{"dog"}},
				{Path: paths[3], Cohort: "c"},
				{Path: paths[4], Cohort: "unassigned", Tags: []string{"bird"}},
			}
			for i := range items {
				items[i].Preview = preview
			}
			// Repeated source identities and repeated labels count only once.
			items = append(items, items[0])
			emit(similarity.Event{Items: items, Total: 5, Successful: 5, Complete: true})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		check := func(label string, count int) *widget.Check {
			t.Helper()
			check, _ := explorerTag(t, v, label, count)
			return check
		}
		bird, dog, untagged := check("Bird", 3), check("Dog", 2), check("Untagged", 1)
		if !bird.Checked || !dog.Checked || !untagged.Checked || len(explorerPiles(v)) != 3 {
			t.Fatal("tags must start checked with every cohort reachable")
		}
		if path := os.Getenv("PICFETCH_EXPLORER_TAG_QA"); path != "" {
			v.win.Resize(fyne.NewSize(1100, 700))
			v.ForceRepaint()
			v.explorer.surface.Fit()
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, v.win.Canvas().Capture())
			closeErr := f.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("tag render: %v/%v", err, closeErr)
			}
		}
		first := explorerPiles(v)[0]
		position, size := first.Position(), first.Size()
		fynetest.Tap(untagged)
		fynetest.Tap(dog)
		if v.win.Canvas().Focused() != nil {
			t.Fatal("tag checkbox captured keyboard focus and blocked map Escape")
		}
		if len(explorerPiles(v)) != 1 || explorerPiles(v)[0] != first {
			t.Fatal("Bird must show only its assigned cohort; filtering must preserve piles")
		}
		var clearTags *widget.Button
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Clear tags") {
				clearTags = b
			}
		})
		if clearTags == nil {
			t.Fatal("tag list needs a clear action to avoid toggling every tag individually")
		}
		fynetest.Tap(clearTags)
		if v.win.Canvas().Focused() != nil {
			t.Fatal("clear-tags button captured keyboard focus and blocked map Escape")
		}
		if len(explorerPiles(v)) != 0 {
			t.Fatal("all tags off must hide every pile")
		}
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == fmt.Sprintf(lang.L("Unassigned (%d)"), 1) {
				t.Fatal("all tags off left Unassigned reachable")
			}
		})
		fynetest.Tap(dog)
		if len(explorerPiles(v)) != 2 || first.Position() != position || first.Size() != size {
			t.Fatal("OR filtering changed the map camera or cohort positions")
		}
		check("Bird", 3)
		check("Dog", 2)
		check("Untagged", 1)
		fynetest.Tap(first)
		if got := explorerGridPaths(v); !slices.Equal(got, paths[:2]) {
			t.Fatalf("filter changed cohort membership: %v, want %v", got, paths[:2])
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if check("Bird", 3).Checked || !check("Dog", 2).Checked || len(explorerPiles(v)) != 2 {
			t.Fatal("returning from the cohort lost filter choices")
		}
		var allTags *widget.Button
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("All tags") {
				allTags = b
			}
		})
		if allTags == nil {
			t.Fatal("tag list needs an action to restore every tag")
		}
		fynetest.Tap(allTags)
		if !check("Bird", 3).Checked || !check("Untagged", 1).Checked || len(explorerPiles(v)) != 3 {
			t.Fatal("All tags did not restore every cohort")
		}
		fynetest.Tap(clearTags)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !check("Bird", 3).Checked || !check("Untagged", 1).Checked || len(explorerPiles(v)) != 3 {
			t.Fatal("a fresh explorer session must reset all tags to checked")
		}
	})

	t.Run("tags_progressive", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 128, 96, color.White)
		item := func(i int, cohort string, tags ...string) similarity.Item {
			return similarity.Item{Path: v.FileAt(i).Path(), Cohort: cohort, Tags: tags, Preview: preview}
		}
		check := func(label string) *widget.Check {
			t.Helper()
			check, _ := explorerTag(t, v, label, 1)
			return check
		}
		publish(similarity.Event{Items: []similarity.Item{item(0, "a", "bird"), item(1, "b", "dog"), item(2, "c")}, Successful: 3, Total: 8})
		fynetest.Tap(check("Bird"))
		fynetest.Tap(explorerPiles(v)[0])
		frozen := []string{v.FileAt(1).Path()}
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("did not open the dog cohort")
		}
		publish(similarity.Event{Items: []similarity.Item{item(1, "b", "dog"), item(2, "b", "cat")}, Successful: 2, Total: 8})
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("tagged publication changed an open cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !check("Cat").Checked {
			t.Fatal("a newly discovered tag must start checked")
		}
		failed := item(4, "failed", "bird")
		failed.Error = "unreadable"
		publish(similarity.Event{Items: []similarity.Item{item(0, "a", "bird"), item(1, "b", "dog"), item(2, "b", "cat"), item(3, "c", "unknown-label"), failed}, Successful: 4, Failed: 1, Total: 8, Complete: true})
		if check("Bird").Checked || !check("Cat").Checked || !check("Untagged").Checked || len(explorerPiles(v)) != 2 {
			t.Fatal("publication lost choices, counted failed sources, or hid unknown content")
		}
		fynetest.Tap(check("Dog"))
		fynetest.Tap(check("Untagged"))
		if len(explorerPiles(v)) != 1 {
			t.Fatal("cohort must stay visible while any member tag is active")
		}
		fynetest.Tap(explorerPiles(v)[0])
		if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(1).Path(), v.FileAt(2).Path()}) {
			t.Fatalf("reopened tagged cohort did not use current complete membership: %v", got)
		}
	})

	t.Run("release_on_exit", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		allocated := func() uint64 { runtime.GC(); var m runtime.MemStats; runtime.ReadMemStats(&m); return m.HeapAlloc }
		baseline := allocated()
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			for pass := range 2 {
				var items []similarity.Item
				for i, path := range paths {
					preview := make([]byte, 8<<20)
					copy(preview, uitest.EncodeJPEG(t, 16, 16, color.White))
					items = append(items, similarity.Item{Path: path, Cohort: fmt.Sprint(i), Preview: preview})
				}
				emit(similarity.Event{Complete: pass == 1, Successful: len(items), Total: len(items), Items: items})
			}
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		loaded := allocated()
		if loaded < baseline+40<<20 {
			t.Fatal("fixture did not retain its map resources")
		}
		if loaded > baseline+64<<20 {
			t.Fatalf("map rebuild retained superseded resources: baseline %.1f MiB, mapped %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20))
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.settleExplorer()
		after := allocated()
		t.Logf("heap baseline %.1f MiB, current map %.1f MiB, after exit %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20), float64(after)/(1<<20))
		if after > baseline+16<<20 {
			t.Fatalf("leaving Explorer retained map memory: baseline %.1f MiB, mapped %.1f MiB, exited %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20), float64(after)/(1<<20))
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if len(explorerPiles(v)) != 6 {
			t.Fatal("reopening the released explorer did not rebuild a map")
		}
	})
	t.Run("close_files_memory", func(t *testing.T) {
		v, _, closed := newTestUI(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		uris := make([]fyne.URI, 512)
		jpeg := uitest.EncodeJPEG(t, 256, 256, color.White)
		for i := range uris {
			uris[i] = storage.NewFileURI(uitest.WriteTempFile(t, fmt.Sprintf("%03d.jpg", i), jpeg))
		}
		allocated := func() uint64 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return m.HeapAlloc
		}
		baseline := allocated()
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			items := make([]similarity.Item, len(paths))
			for i, path := range paths {
				preview := make([]byte, 128<<10)
				copy(preview, jpeg)
				items[i] = similarity.Item{Path: path, Cohort: "all", Preview: preview}
			}
			emit(similarity.Event{Total: len(items), Successful: len(items), Complete: true, Items: items})
			return nil
		}
		dropAndWait(t, v, uris...)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerPiles(v)[0])
		warmThumbs(t, v)
		v.grid.Settle()
		if len(explorerGridPaths(v)) != len(uris) {
			t.Fatal("cohort did not expose the complete file set")
		}
		for v.grid.Highlight() < v.FileCount()-1 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPageDown})
			v.grid.Settle()
			// Long browsing sessions collect recycled grid cells too.
			runtime.GC()
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.settleExplorer()
		if len(explorerPiles(v)) != 1 {
			t.Fatal("browsing did not return to the map")
		}
		loaded := allocated()
		if loaded < baseline+48<<20 {
			t.Fatal("fixture did not retain its map and browsed thumbnails")
		}
		buildMainMenu(v).Items[0].Items[3].Action()
		v.settleExplorer()
		v.preloads.Wait()
		after := allocated()
		t.Logf("heap baseline %.1f MiB, browsed %.1f MiB, closed %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20), float64(after)/(1<<20))
		if after > baseline+8<<20 {
			t.Errorf("Close Files retained source-derived memory after browsing")
		}
		if closed() || v.FileCount() != 0 || !v.dropzone.Visible() || v.grid.Visible() {
			t.Fatal("Close Files did not leave the running app at its empty dropzone")
		}
		dropAndWait(t, v, uris[:2]...)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		fynetest.Tap(explorerPiles(v)[0])
		if len(explorerGridPaths(v)) != 2 {
			t.Fatal("reopening after Close Files did not admit the new cohort")
		}
		buildMainMenu(v).Items[0].Items[3].Action()
		v.settleExplorer()
		if closed() || v.FileCount() != 0 || !v.dropzone.Visible() || v.grid.Visible() {
			t.Fatal("Close Files from the cohort left the grid over the empty dropzone")
		}
	})
	t.Run("settings", func(t *testing.T) {
		before := preferences.Load(testApp)
		t.Cleanup(func() { preferences.Save(testApp, before) })
		for _, key := range []string{"similarityFavoriteCache", "similarityAutoUpdate", "similarityAutoFit"} {
			testApp.Preferences().RemoveValue(key)
		}
		labels := []string{"Save analysis for favorites", "Auto-update every 30 images", "Fit new stacks into view"}
		findChecks := func(v *viewer) (fyne.Window, map[string]*widget.Check) {
			v.settingsWin.Show(v.settingsState(), false)
			checks := map[string]*widget.Check{}
			for _, win := range v.app.Driver().AllWindows() {
				if win.Title() != lang.L("Settings") {
					continue
				}
				explorerWalk(win.Content(), func(o fyne.CanvasObject) {
					if c, ok := o.(*widget.Check); ok {
						checks[c.Text] = c
					}
				})
				return win, checks
			}
			t.Fatal("settings window did not open")
			return nil, nil
		}
		v := newTestViewer(t)
		win, checks := findChecks(v)
		defaults := []bool{true, false, true}
		for i, label := range labels {
			c := checks[lang.L(label)]
			if c == nil {
				t.Fatalf("settings is missing %s", label)
			}
			if c.Checked != defaults[i] {
				t.Fatalf("wrong default for %s", label)
			}
			fynetest.Tap(c)
		}
		win.Close()
		preferences.Save(testApp, v.currentPreferences())
		reopened := newTestViewer(t)
		win, checks = findChecks(reopened)
		defer win.Close()
		for i, label := range labels {
			if checks[lang.L(label)].Checked == defaults[i] {
				t.Fatalf("%s was not persisted", label)
			}
		}
	})
	t.Run("manual_controls", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		events := make(chan similarity.Event)
		published := make(chan struct{})
		seen := make(chan similarity.Control, 8)
		v.explorerAnalyze = func(ctx context.Context, _ []string, controls <-chan similarity.Control, emit func(similarity.Event)) error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case control := <-controls:
					seen <- control
				case event := <-events:
					emit(event)
					published <- struct{}{}
					if event.Complete {
						return nil
					}
				}
			}
		}
		explorerMenu(t, v).Action()
		var update *widget.Button
		var automatic *widget.Check
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Update map") {
				update = b
			}
			if c, ok := o.(*widget.Check); ok && c.Text == lang.L("Auto-update every 30 images") {
				automatic = c
			}
		})
		if update == nil || automatic == nil {
			t.Fatal("map is missing manual update controls")
		}
		if !update.Disabled() || automatic.Checked {
			t.Fatal("map must begin in manual mode with no data to rebuild")
		}
		receive := func() similarity.Control {
			select {
			case c := <-seen:
				return c
			case <-time.After(testTimeout):
				t.Fatal("map control did not reach analysis")
				return similarity.Control{}
			}
		}
		if c := receive(); c.Automatic || c.Update {
			t.Fatal("analysis did not start in manual mode")
		}
		publish := func(event similarity.Event) { events <- event; <-published; v.explorer.ui.Drain() }
		publish(similarity.Event{Successful: 1, Total: 2, Stage: "encoding"})
		if update.Disabled() {
			t.Fatal("available data cannot be rebuilt")
		}
		fynetest.Tap(update)
		if c := receive(); !c.Update || c.Automatic {
			t.Fatal("button did not request a manual rebuild")
		}
		if !update.Disabled() {
			t.Fatal("pending rebuild allows duplicate requests")
		}
		fynetest.Tap(automatic)
		if c := receive(); !c.Automatic || c.Update {
			t.Fatal("automatic option was not delivered")
		}
		publish(similarity.Event{Successful: 1, Total: 2, Items: []similarity.Item{{Path: v.FileAt(0).Path(), Cohort: "a"}}})
		if !update.Disabled() {
			t.Fatal("up-to-date map can be rebuilt without new data")
		}
		publish(similarity.Event{Successful: 2, Total: 2, Stage: "encoding"})
		if update.Disabled() {
			t.Fatal("new data did not re-enable manual update")
		}
		fynetest.Tap(automatic)
		if c := receive(); c.Automatic || c.Update {
			t.Fatal("automatic updates cannot be disabled")
		}
		publish(similarity.Event{Successful: 2, Total: 2, Complete: true, Items: []similarity.Item{{Path: v.FileAt(0).Path(), Cohort: "a"}, {Path: v.FileAt(1).Path(), Cohort: "a"}}})
		v.settleExplorer()
		if !update.Disabled() {
			t.Fatal("completed map still offers a rebuild")
		}
	})
	t.Run("pending_duplicates", func(t *testing.T) {
		for _, action := range []string{"finish", "cancel", "replace"} {
			t.Run(action, func(t *testing.T) {
				v := newTestViewer(t)
				small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
				large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
				unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
				data, err := os.ReadFile(large.Path())
				if err != nil {
					t.Fatal(err)
				}
				var armed atomic.Bool
				var once sync.Once
				release := make(chan struct{})
				defer once.Do(func() { close(release) })
				held := uitest.ReaderURI(large, func() (io.ReadCloser, error) {
					r := bytes.NewReader(data)
					return uitest.ReadCloser{ReadFunc: func(p []byte) (int, error) {
						if armed.Load() {
							<-release
						}
						return r.Read(p)
					}, CloseFunc: func() error { return nil }}, nil
				})
				dropAndWait(t, v, small, held, unique)
				started := make(chan []string, 1)
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					started <- paths
					emit(similarity.Event{Complete: true, Total: len(paths)})
					return nil
				}
				func() {
					armed.Store(true)
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
					explorerMenu(t, v).Action()
					v.explorer.workers.Wait()
					v.explorer.ui.Drain()
					select {
					case <-started:
						t.Fatal("analysis started before duplicate facts were ready")
					default:
					}
					if !v.explorerMapActive() {
						t.Fatal("waiting for duplicate facts hid the map")
					}
					if action == "cancel" {
						v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					}
					if action == "replace" {
						v.handleDrop([]fyne.URI{small, unique})
					}
					once.Do(func() { close(release) })
					v.settleExplorer()
				}()
				if action == "replace" {
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					v.settleExplorer()
				}
				select {
				case paths := <-started:
					if action != "finish" {
						t.Fatal("cancelled duplicate preparation started analysis")
					}
					if !slices.Equal(paths, []string{large.Path(), unique.Path()}) {
						t.Fatalf("pending hashes did not elect highest quality: %v", paths)
					}
				default:
					if action == "finish" {
						t.Fatal("finished duplicate preparation did not start analysis")
					}
				}
			})
		}
	})
	t.Run("duplicate_representatives", func(t *testing.T) {
		v := newTestViewer(t)
		small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
		large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
		unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
		dropAndWait(t, v, small, large, unique)
		warmThumbs(t, v)
		v.grid.Toggle()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
		v.grid.Settle()
		waitUntilLoaded(t, v)
		if !slices.Equal(explorerGridPaths(v), []string{large.Path(), unique.Path()}) {
			t.Fatal("premise: D did not filter the smaller duplicate")
		}
		v.grid.HandleRune('/')
		v.grid.HandleRune('c')
		v.grid.SelectAll()
		var got []string
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			got = append([]string(nil), paths...)
			emit(similarity.Event{Complete: true, Total: len(paths)})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, []string{large.Path(), unique.Path()}) {
			t.Fatalf("filtered analysis must use the highest-resolution representative plus unique sources: %v", got)
		}
		if v.FileCount() != 3 {
			t.Fatal("representative analysis changed the opened file set")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, []string{small.Path(), large.Path(), unique.Path()}) {
			t.Fatal("disabling duplicate filtering did not restore all analysis inputs")
		}
	})
	t.Run("minimum_zoom_inputs", func(t *testing.T) {
		v, publish := explorerLargeFixture(t)
		publish(144, true)
		v.settleExplorer()
		pile := explorerPiles(v)[0]
		// Sampling is observed when the target is on screen; distant piles
		// may release their decoded pixels while retaining their identities.
		viewport := v.explorer.surface.Size()
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: viewport.Width/2 - pile.Position().X - pile.Size().Width/2, DY: viewport.Height/2 - pile.Position().Y - pile.Size().Height/2}})
		samples := explorerSamples(pile)
		if len(samples) != 15 {
			t.Fatalf("large map must retain all 15 sampled thumbnails, got %d", len(samples))
		}
		for range 12 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		}
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := pile.Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatalf("large-map zoom-out must stop at a 190px pile, got %.2fpx", got)
		}
		pos, size := pile.Position(), pile.Size()
		fynetest.Tap(explorerButton(t, v, "-"))
		v.explorer.surface.Scrolled(&fyne.ScrollEvent{Position: fyne.NewPos(81, 93), Scrolled: fyne.Delta{DY: -1000}})
		if pile.Position() != pos || pile.Size() != size || !slices.Equal(explorerSamples(pile), samples) {
			t.Fatal("zoom-out at the floor changed the camera or sampled thumbnails")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		if pile.Size().Width <= size.Width {
			t.Fatal("zoom-in stopped working at the floor")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		// Bring the sampled cohort onto the screen using the ordinary pan input.
		viewport = v.explorer.surface.Size()
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: viewport.Width/2 - pile.Position().X - pile.Size().Width/2, DY: viewport.Height/2 - pile.Position().Y - pile.Size().Height/2}})
		pos, size = pile.Position(), pile.Size()
		fynetest.Tap(pile)
		if len(explorerGridPaths(v)) != 16 {
			t.Fatal("large-map pile did not open its complete cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if pile.Position() != pos || pile.Size() != size || !slices.Equal(explorerSamples(pile), samples) {
			t.Fatal("cohort return lost the camera or full sample set")
		}

		small := explorerFixture(t)
		for range 40 {
			small.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := explorerPiles(small)[0].Size().Width; math.Abs(float64(got-11.4)) > .01 {
			t.Fatalf("small map lost its original 0.03x zoom floor: pile %.2fpx", got)
		}
	})

	t.Run("minimum_zoom_fit", func(t *testing.T) {
		v, publish := explorerLargeFixture(t)
		publish(120, false)
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatalf("initial large-map fit bypassed the floor: %.2fpx", got)
		}
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 23, DY: -17}})
		before := explorerPiles(v)[0].Position()
		publish(144, false)
		pile := explorerPiles(v)[0]
		if math.Abs(float64(pile.Size().Width-190)) > .01 || pile.Position() != before {
			t.Fatalf("discovery at the floor moved the camera: %v -> %v, width %.2f", before, pile.Position(), pile.Size().Width)
		}
		fynetest.Tap(explorerButton(t, v, "Fit map"))
		if got := pile.Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatalf("manual large-map fit bypassed the floor: %.2fpx", got)
		}
		// Keyboard navigation must still bring distant piles into view.
		for range 20 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		}
		var selected *explorerui.Pile
		for _, p := range explorerPiles(v) {
			explorerWalk(p, func(o fyne.CanvasObject) {
				if frame, ok := o.(*canvas.Rectangle); ok && frame.StrokeWidth >= 2 {
					selected = p
				}
			})
		}
		if selected == nil {
			t.Fatal("large-map navigation lost selection")
		}
		pos, size, viewport := selected.Position(), selected.Size(), v.explorer.surface.Size()
		if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > viewport.Width || pos.Y+size.Height > viewport.Height {
			t.Fatal("keyboard navigation left the selected large-map pile off screen")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if !v.grid.Visible() || len(explorerGridPaths(v)) == 0 {
			t.Fatal("keyboard could not open the distant cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})

		var slider *widget.Slider
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if s, ok := o.(*widget.Slider); ok {
				slider = s
			}
		})
		if slider == nil {
			t.Fatal("missing granularity control")
		}
		slider.SetValue(0)
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-11.4)) > .01 {
			t.Fatal("reducing a large map to one pile did not release its zoom limit")
		}
		slider.SetValue(100)
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatal("restoring a large map retained the small-map zoom")
		}
		v.win.Resize(fyne.NewSize(700, 400))
		v.ForceRepaint()
		if explorerPiles(v)[0].Size().Width < 190 {
			t.Fatal("window resize bypassed the large-map zoom floor")
		}
		publish(115, true) // Exactly 100 cohorts, including the 16-image pile.
		v.settleExplorer()
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-11.4)) > .01 {
			t.Fatalf("100-pile map incorrectly received the large-map zoom floor: %.2fpx", got)
		}
	})

	t.Run("auto_fit_off", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		prev := v.settingsState()
		next := prev
		next.SimilarityAutoFit = false
		v.ApplySettings(prev, next)
		publish([]string{"a", "a"}, false)
		before := explorerPiles(v)[0].Size()
		publish([]string{"a", "a", "b", "c", "d", "e", "f", "g"}, true)
		if explorerPiles(v)[0].Size() != before {
			t.Fatal("disabled automatic fitting still changed zoom")
		}
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Fit map") {
				fynetest.Tap(b)
			}
		})
		if explorerPiles(v)[0].Size() == before {
			t.Fatal("manual Fit map no longer works with automatic fitting disabled")
		}
		viewport := v.explorer.surface.Size()
		for _, p := range explorerPiles(v) {
			pos, size := p.Position(), p.Size()
			if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > viewport.Width || pos.Y+size.Height > viewport.Height {
				t.Fatal("manual fitting left a stack outside the map")
			}
		}
	})
	t.Run("discovery_expands_view", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		publish([]string{"a", "a"}, false)
		before := explorerPiles(v)[0].Size()
		publish([]string{"a", "a", "b", "c", "d", "e", "f", "g"}, false)
		piles := explorerPiles(v)
		if len(piles) != 7 {
			t.Fatal("discovery lost newly created piles")
		}
		viewport := v.explorer.surface.Size()
		for _, pile := range piles {
			pos, size := pile.Position(), pile.Size()
			if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > viewport.Width || pos.Y+size.Height > viewport.Height {
				t.Fatalf("new discovery left a pile outside the viewport: %v/%v in %v", pos, size, viewport)
			}
		}
		zoomedOut := piles[0].Size()
		if zoomedOut.Width >= before.Width {
			t.Fatal("map did not zoom out to include new piles")
		}
		publish([]string{"a", "a", "a", "a", "a", "a", "a", "a"}, true)
		if got := explorerPiles(v)[0].Size(); got != zoomedOut {
			t.Fatal("a later grouping automatically zoomed back in")
		}
		v.settleExplorer()
	})
	t.Run("progressive_browse_camera", func(t *testing.T) {
		for _, surface := range []string{"grid", "image"} {
			t.Run(surface, func(t *testing.T) {
				v, publish := streamingExplorer(t)
				v.win.Resize(fyne.NewSize(1100, 700))
				prev := v.settingsState()
				next := prev
				next.SimilarityAutoFit = true
				v.ApplySettings(prev, next)
				publish([]string{"a", "a", "a", "a"}, false)
				center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
				fynetest.Drag(v.win.Canvas(), center, 50, 35)
				fynetest.Scroll(v.win.Canvas(), center, 0, 80)
				departure := v.explorer.surface.View()
				fynetest.Tap(explorerPiles(v)[0])
				frozen := explorerGridPaths(v)
				if surface == "image" {
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
					waitUntilLoaded(t, v)
				}
				publish([]string{"a", "a", "b", "c", "d", "e", "f", "g"}, false)
				if surface == "image" {
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
					waitUntilLoaded(t, v)
					current, _, _ := v.CurrentFile()
					if current.Path() != frozen[len(frozen)-1] {
						t.Fatal("progressive discovery changed frozen image navigation")
					}
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				}
				if !slices.Equal(explorerGridPaths(v), frozen) {
					t.Fatal("progressive discovery changed the open cohort")
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				returned := v.explorer.surface.View()
				if returned.Center != departure.Center || returned.Zoom != departure.Zoom {
					t.Fatalf("automatic fitting moved the browsing camera: departure=%+v return=%+v", departure, returned)
				}
				if len(explorerPiles(v)) != 7 {
					t.Fatal("camera preservation discarded the new map revision")
				}
				fynetest.Tap(explorerPiles(v)[0])
				if !slices.Equal(explorerGridPaths(v), frozen[:2]) {
					t.Fatal("reopening did not use current cohort membership")
				}
				publish([]string{"a", "b", "c", "d", "e", "f", "g", "h"}, true)
				v.settleExplorer()
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				completed := v.explorer.surface.View()
				if completed.Center != departure.Center || completed.Zoom != departure.Zoom || completed.Piles != 8 {
					t.Fatalf("completion did not preserve the browsing camera and latest map: %+v", completed)
				}
			})
		}
	})
	t.Run("progressive_exploration", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "a", "a"}, false)
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatal("partial map is unavailable while analysis remains pending")
		}
		progress := false
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if label, ok := o.(*widget.Label); ok && label.Text == fmt.Sprintf(lang.L("%d ready, %d failed, %d total"), 4, 0, 8) {
				progress = true
			}
		})
		if !progress {
			t.Fatal("partial analysis counts are missing from the visible surface")
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 4 {
			t.Fatal("partial cohort is not browsable")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		publish([]string{"a", "a", "a", "a", "b", "b", "b", "b"}, true)
		v.settleExplorer()
		if len(explorerPiles(v)) != 2 {
			t.Fatal("final discovery did not extend the partial map")
		}
	})
	t.Run("pile_spacing", func(t *testing.T) {
		for name, positions := range map[string][][]float32{
			"coincident":      {{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},
			"cell_boundaries": {{-1, -1}, {-.99, 1}, {.5, -.1}, {1, 1}, {-.5, -1}, {.7, .75}},
		} {
			t.Run(name, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					var items []similarity.Item
					for i, path := range paths {
						items = append(items, similarity.Item{Path: path, Cohort: fmt.Sprint(i), Position: positions[i], Preview: preview})
					}
					emit(similarity.Event{Complete: true, Items: items})
					return nil
				}
				explorerMenu(t, v).Action()
				v.settleExplorer()
				piles := explorerPiles(v)
				if len(piles) != 6 {
					t.Fatal("missing close/coincident cohorts")
				}
				for i, a := range piles {
					for _, b := range piles[i+1:] {
						dx := float32(math.Abs(float64(a.Position().X-b.Position().X))) - a.Size().Width
						dy := float32(math.Abs(float64(a.Position().Y-b.Position().Y))) - a.Size().Height
						if max(dx, dy) < a.Size().Width*.08 {
							t.Fatalf("piles overlap or lack breathing room: horizontal gap %.1f, vertical gap %.1f", dx, dy)
						}
					}
				}
			})
		}
	})
	t.Run("navigation", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "b", "b"}, false)
		center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
		fynetest.Drag(v.win.Canvas(), center, 50, 35)
		fynetest.Scroll(v.win.Canvas(), center, 0, 80)
		before := explorerPiles(v)
		delta := before[1].Position().Subtract(before[0].Position())
		delta.X /= before[0].Size().Width
		delta.Y /= before[0].Size().Height
		publish([]string{"new-a", "new-a", "new-b", "new-b", "new-c", "new-c", "new-c", "new-c"}, false)
		piles := explorerPiles(v)
		if len(piles) != 3 {
			t.Fatal("discovery lost a continuing pile")
		}
		after := piles[1].Position().Subtract(piles[0].Position())
		if math.Abs(float64(after.X/piles[0].Size().Width-delta.X)) > .001 || math.Abs(float64(after.Y/piles[0].Size().Height-delta.Y)) > .001 {
			t.Fatal("discovery moved continuing piles relative to each other")
		}
		position := piles[0].Position()
		fynetest.Drag(v.win.Canvas(), center, 30, 20)
		if piles[0].Position() == position {
			t.Fatal("map stopped responding while further analysis remained held")
		}
	})
	t.Run("fit_uses_window", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		v.win.Resize(fyne.NewSize(1100, 700))
		preview := uitest.EncodeJPEG(t, 128, 96, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for i, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: fmt.Sprint(i), Position: []float32{0, float32(i * 100)}, Preview: preview})
			}
			emit(similarity.Event{Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		for _, pile := range explorerPiles(v) {
			if pile.Size().Width < v.win.Canvas().Size().Width*.2 {
				t.Fatalf("a tall projection wastes the wide window: pile width %.1f", pile.Size().Width)
			}
		}
	})
	t.Run("open_cohort", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "a", "a", "b", "b"}, false)
		fynetest.Tap(explorerPiles(v)[0])
		frozen := explorerGridPaths(v)
		publish([]string{"new-a", "new-a", "new-b", "new-b", "new-b", "new-b", "new-c", "new-c"}, true)
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("regrouping changed the already open cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		current, _, _ := v.CurrentFile()
		if current.Path() != frozen[len(frozen)-1] {
			t.Fatal("regrouping changed image navigation in the frozen cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		fynetest.Tap(explorerPiles(v)[0])
		if got := explorerGridPaths(v); !slices.Equal(got, frozen[:2]) {
			t.Fatal("reopening a pile did not use its current membership")
		}
		v.settleExplorer()
	})
	t.Run("non_overlapping_cohorts", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "a", "a", "b", "b"}, false)
		fynetest.Tap(explorerPiles(v)[0])
		publish([]string{"new-a", "new-a", "new-b", "new-b", "new-b", "new-b", "new-c", "new-c"}, true)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		owners := map[string]bool{}
		for _, pile := range explorerPiles(v) {
			fynetest.Tap(pile)
			for _, path := range explorerGridPaths(v) {
				if owners[path] {
					t.Fatal("current map assigns one source to multiple cohorts")
				}
				owners[path] = true
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
		if len(owners) != 8 {
			t.Fatal("current map lost members during regrouping")
		}
		v.settleExplorer()
	})
	t.Run("cohort_piles", func(t *testing.T) {
		v := explorerFixture(t)
		if path := os.Getenv("PICFETCH_EXPLORER_QA"); path != "" {
			v.win.Resize(fyne.NewSize(1100, 700))
			v.ForceRepaint()
			v.explorer.surface.Fit()
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, v.win.Canvas().Capture())
			closeErr := f.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("render: %v/%v", err, closeErr)
			}
		}
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatalf("want two cohort piles in the surface, got %d", len(piles))
		}
		for i, pile := range piles {
			samples := map[string]bool{}
			count := 0
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Resource != nil {
					samples[img.Resource.Name()] = true
					count++
				}
			})
			want := 15
			if i == 1 {
				want = 1
			}
			if count != want || len(samples) != want {
				t.Fatalf("pile %d has %d images/%d distinct sources; want %d", i, count, len(samples), want)
			}
			fynetest.Tap(pile)
			members := explorerGridPaths(v)
			expected := 16
			if i == 1 {
				expected = 1
			}
			if len(members) != expected {
				t.Fatalf("opening pile has %d members; want %d including unsampled", len(members), expected)
			}
			for sample := range samples {
				if !slices.Contains(members, sample) {
					t.Fatal("pile sample is outside cohort")
				}
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
		found := false
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if button, ok := o.(*widget.Button); ok && button.Text == fmt.Sprintf(lang.L("Unassigned (%d)"), 1) {
				found = true
				fynetest.Tap(button)
			}
		})
		if !found || len(explorerGridPaths(v)) != 1 {
			t.Fatal("Unassigned must open its own collection")
		}
	})
	t.Run("merged_source_samples", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		v.SetMergeMode(true)
		dropAndWait(t, v, v.FileAt(0))
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for _, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: "a", Position: []float32{0, 0}, Preview: preview})
			}
			emit(similarity.Event{Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatal("missing merged cohort")
		}
		samples := map[string]bool{}
		count := 0
		explorerWalk(piles[0], func(o fyne.CanvasObject) {
			if img, ok := o.(*canvas.Image); ok && img.Resource != nil {
				samples[img.Resource.Name()] = true
				count++
			}
		})
		if count != 3 || len(samples) != 3 {
			t.Fatalf("merged source repeated in preview: %d samples / %d distinct", count, len(samples))
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 4 || v.FileCount() != 4 {
			t.Fatal("distinct previews must preserve all opened occurrences in Grid View")
		}
	})
	t.Run("scattered_previews", func(t *testing.T) {
		v := explorerFixture(t)
		pile := explorerPiles(v)[0]
		var pictures []*canvas.Image
		explorerWalk(pile, func(o fyne.CanvasObject) {
			if img, ok := o.(*canvas.Image); ok {
				pictures = append(pictures, img)
			}
		})
		for i, a := range pictures {
			if ratio := a.Size().Width / a.Size().Height; math.Abs(float64(ratio-4.0/3)) > .01 {
				t.Fatalf("preview frame reserves unused letterbox space: ratio %.3f, want 4:3", ratio)
			}
			for _, b := range pictures[i+1:] {
				w := max(float32(0), min(a.Position().X+a.Size().Width, b.Position().X+b.Size().Width)-max(a.Position().X, b.Position().X))
				h := max(float32(0), min(a.Position().Y+a.Size().Height, b.Position().Y+b.Size().Height)-max(a.Position().Y, b.Position().Y))
				if w*h > a.Size().Width*a.Size().Height*.7 {
					t.Fatal("two sampled images overlap by more than 70 percent")
				}
			}
		}
		objects := fynetest.WidgetRenderer(pile).Objects()
		for _, img := range pictures {
			for _, obj := range objects {
				border, ok := obj.(*canvas.Rectangle)
				if !ok || border.Position().Y > img.Position().Y || border.Position().X > img.Position().X {
					continue
				}
				if abs := img.Position().X - border.Position().X; abs < img.Size().Width*.04 && img.Position().Y-border.Position().Y < img.Size().Height*.04 {
					if border.Size().Width-img.Size().Width > img.Size().Width*.025 {
						t.Fatal("preview border is too thick")
					}
				}
			}
		}
	})
	t.Run("projection_scale", func(t *testing.T) {
		v := explorerFixtureScale(t, 1000000)
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatal("missing cohorts")
		}
		for _, pile := range piles {
			if pile.Size().Width < 100 {
				t.Fatalf("projection units made a pile unreadable: %v", pile.Size())
			}
		}
	})
	t.Run("completed_map", func(t *testing.T) {
		v := explorerFixture(t)
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatal("map not delivered")
		}
		fynetest.Tap(piles[0])
		want := explorerGridPaths(v)
		if len(want) != 16 {
			t.Fatal("cohort missing members")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		current, _, _ := v.CurrentFile()
		if current.Path() != want[len(want)-1] {
			t.Fatal("End escaped the cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		waitUntilLoaded(t, v)
		current, _, _ = v.CurrentFile()
		if current.Path() != want[0] {
			t.Fatal("next image did not wrap within the cohort")
		}
		if v.FileCount() != 18 {
			t.Fatal("cohort browsing replaced the opened file set")
		}
	})
	t.Run("cold_cohort_size", func(t *testing.T) {
		v := explorerFixture(t)
		v.settings.staticWindowSize = false
		v.win.Resize(fyne.NewSize(1100, 700))
		want := v.win.Canvas().Size()
		fynetest.Tap(explorerPiles(v)[0])
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.imgCache.Purge()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		if got := v.win.Canvas().Size(); got != want {
			t.Fatalf("uncached cohort image resized the explorer window: %v -> %v", want, got)
		}
	})
	t.Run("return_to_map", func(t *testing.T) {
		v := explorerFixture(t)
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatal("map missing")
		}
		before, size := piles[0].Position(), piles[0].Size()
		center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
		fynetest.Drag(v.win.Canvas(), center, 50, 35)
		fynetest.Scroll(v.win.Canvas(), center, 0, 80)
		moved, scaled := piles[0].Position(), piles[0].Size()
		if moved == before || scaled == size {
			t.Fatalf("canvas drag/scroll did not move and zoom the map: canvas=%v map=%v/%v pile %v/%v -> %v/%v", v.win.Canvas().Size(), v.explorer.surface.Position(), v.explorer.surface.Size(), before, size, moved, scaled)
		}
		fynetest.Tap(piles[0])
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() || len(explorerGridPaths(v)) != 16 {
			t.Fatal("Escape did not reopen cohort Grid View")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		returned := explorerPiles(v)
		if v.grid.Visible() || len(returned) != 2 || returned[0].Position() != moved || returned[0].Size() != scaled {
			t.Fatal("return changed the map camera")
		}
	})
	t.Run("shift_pan", func(t *testing.T) {
		v := explorerFixture(t)
		pile := explorerPiles(v)[0]
		before, size := pile.Position(), pile.Size()
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
		center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
		fynetest.Scroll(v.win.Canvas(), center, 40, 60)
		if pile.Position() == before || pile.Size() != size {
			t.Fatal("Shift-scroll must pan the map without zooming")
		}
		v.keyModifiers = func() fyne.KeyModifier { return 0 }
		fynetest.Scroll(v.win.Canvas(), center, 0, 60)
		if pile.Size() == size {
			t.Fatal("unmodified scrolling must still zoom")
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		started := make(chan struct{})
		stopped := make(chan struct{})
		v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			close(started)
			<-ctx.Done()
			emit(similarity.Event{Complete: true, Total: len(paths), Successful: len(paths)})
			close(stopped)
			return ctx.Err()
		}
		explorerMenu(t, v).Action()
		<-started
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		select {
		case <-stopped:
		case <-time.After(testTimeout):
			t.Fatal("leaving map did not cancel analysis")
		}
		v.settleExplorer()
		if len(explorerPiles(v)) != 0 || v.FileCount() != 2 {
			t.Fatal("canceled work changed the opened viewer")
		}
	})
	t.Run("map_commands", func(t *testing.T) {
		v := explorerFixture(t)
		if !v.menus.Actions().Trash().Disabled || !v.menus.Actions().Copy().Disabled {
			t.Fatal("image actions remain enabled behind the map")
		}
		v.menus.Actions().Trash().Action()
		if v.deletion.Visible() {
			t.Fatal("map command targeted a covered image")
		}
		v.menus.Window().Viewer().Action()
		if len(explorerPiles(v)) != 0 {
			t.Fatal("Window -> Viewer did not leave the map")
		}
	})

	t.Run("opened_files", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		var got []string
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			got = append([]string(nil), paths...)
			emit(similarity.Event{Complete: true, Total: len(paths)})
			return nil
		}
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
		v.grid.HandleRune('/')
		v.grid.HandleRune('a')
		v.grid.SelectAll()
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, want) {
			t.Fatal("analysis used Grid search/selection instead of all opened sources")
		}
		v.SetMergeMode(true)
		extra := uitest.TempJPEGURI(t, "d.jpg", 4, 4, color.White)
		dropAndWait(t, v, extra)
		want = append(want, extra.Path())
		slices.Sort(want)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Fatal("merged sources not included in analysis")
		}
		v.SetMergeMode(false)
		dir := t.TempDir()
		favorite := uitest.TempJPEGURI(t, "favorite.jpg", 4, 4, color.White)
		if err := favstore.Save(dir, "Trip", []fyne.URI{favorite}); err != nil {
			t.Fatal(err)
		}
		v.favorites.SetDir(dir)
		v.favorites.Menu().Items[2].Action()
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, []string{favorite.Path()}) {
			t.Fatal("Favorite did not replace the analysis input")
		}
	})
	t.Run("replacement_discards_late_map", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		started := make(chan struct{})
		release := make(chan struct{})
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			close(started)
			<-release
			emit(similarity.Event{Complete: true, Items: []similarity.Item{{Path: paths[0], Cohort: "old", Position: []float32{0, 0}, Preview: preview}}})
			return nil
		}
		explorerMenu(t, v).Action()
		<-started
		replacement := uitest.TempJPEGURI(t, "new.jpg", 4, 4, color.White)
		dropAndWait(t, v, replacement)

		delivered := make(chan struct{})
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{Complete: true, Items: []similarity.Item{{Path: paths[0], Cohort: "new", Position: []float32{0, 0}, Preview: preview}}})
			close(delivered)
			return nil
		}
		explorerMenu(t, v).Action()
		<-delivered
		v.explorer.ui.Drain()
		close(release)
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatal("new map disappeared")
		}
		fynetest.Tap(piles[0])
		if !slices.Equal(explorerGridPaths(v), []string{replacement.Path()}) {
			t.Fatal("late analysis overwrote the new map")
		}

	})

}

type explorerAssetTransport func(*http.Request) (*http.Response, error)

func (f explorerAssetTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
