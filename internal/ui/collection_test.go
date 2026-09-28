package ui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCollectionModel(t *testing.T) {
	t.Run("snapshots", func(t *testing.T) {
		a, b := storage.NewFileURI("/images/a.jpg"), storage.NewFileURI("/images/b.jpg")
		u := storage.NewFileURI("/images/unavailable.heic")
		source := []fyne.URI{b, a, a}
		display := []fyne.URI{a, a, b}
		retained := []collectionSource{{b, false}, {u, true}, {a, false}, {a, false}}
		state := newAppState(0, false)
		state.retainOrder(retained)
		state.replaceFiles(source, display)
		state.Select(1)
		before := state.Observe()
		bookmark, ok := before.Bookmark(1)
		if !ok || bookmark.occurrence != (fileidentity.Occurrence{Path: a.Path(), Ordinal: 1}) {
			t.Fatalf("second occurrence bookmark = %+v, %v", bookmark, ok)
		}
		source[0], display[0], retained[1].uri = u, u, a
		state.Select(2)
		state.reorder([]fyne.URI{b, a, a})
		after := state.Observe()
		if got := before.SourceFiles(); !slices.EqualFunc(got, []fyne.URI{b, a, a}, sameURI) {
			t.Fatalf("source input changed retained snapshot: %v", got)
		}
		if before.FileAt(0) != a || before.FileSet().KeyAt(0) != a.String() || before.Resolve(bookmark) != 1 {
			t.Fatal("old order, URI keys and occurrence lookup must stay together")
		}
		if got, index, present := before.Current(); got != a || index != 1 || !present {
			t.Fatalf("old requested occurrence = %v, %d, %v", got, index, present)
		}
		if after.Retained()[1].uri != u || before.Retained()[1].uri != u {
			t.Fatal("caller mutation changed retained unavailable membership")
		}
		if after.Generation() <= before.Generation() || after.FileSet().Generation() != after.Generation() {
			t.Fatal("published generation does not describe the complete order")
		}
		copySource, copyRetained := before.SourceFiles(), before.Retained()
		copySource[0], copyRetained[1].uri = u, a
		if before.SourceFiles()[0] != b || before.Retained()[1].uri != u {
			t.Fatal("observation exposed writable membership")
		}
	})
	t.Run("operations", func(t *testing.T) {
		t.Run("selection", func(t *testing.T) {
			a, b := storage.NewFileURI("/images/a.jpg"), storage.NewFileURI("/images/b.jpg")
			state := newAppState(0, false)
			if state.Select(0) {
				t.Fatal("empty collection admitted selection")
			}
			state.replaceFiles([]fyne.URI{a, b, a}, []fyne.URI{a, b, a})
			before := state.Observe()
			bookmark, _ := before.Bookmark(2)
			if !state.Select(-1) {
				t.Fatal("nonempty collection refused navigation")
			}
			chosen := state.Observe()
			if uri, index, ok := chosen.Current(); uri != a || index != 2 || !ok {
				t.Fatalf("wrapped requested occurrence = %v, %d, %v", uri, index, ok)
			}
			if chosen.Generation() != before.Generation() || chosen.Resolve(bookmark) != 2 {
				t.Fatal("navigation invalidated collection identity")
			}
			if _, index, _ := before.Current(); index != 0 {
				t.Fatal("navigation changed a retained observation")
			}
			state.replaceFiles([]fyne.URI{a, b, a}, []fyne.URI{a, b, a})
			replaced := state.Observe()
			if replaced.Resolve(bookmark) != -1 {
				t.Fatal("old bookmark resolved into unrelated replacement")
			}
			if _, index, ok := replaced.Current(); index != 0 || !ok {
				t.Fatalf("replacement selected %d, %v; want first occurrence", index, ok)
			}
		})
	})
}

func TestCollectionCompatibility(t *testing.T) {
	t.Run("navigation_snapshots", func(t *testing.T) {
		v := newTestViewer(t)
		files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
		dropAndWait(t, v, files...)
		v.SetMergeMode(true)
		dropAndWait(t, v, files[0])
		v.ShowImage(2)
		waitUntilLoaded(t, v)
		before := v.state.Observe()
		v.display.WaitPreloads()
		v.imgCache.Purge()
		v.ShowImage(1)
		chosen := v.state.Observe()
		bookmark, _ := chosen.Bookmark(1)
		if uri, index, ok := chosen.Current(); uri != files[0] || index != 1 || !ok {
			t.Fatalf("requested collection observation = %v, %d, %v", uri, index, ok)
		}
		if bookmark.occurrence.Ordinal != 1 || chosen.Generation() != before.Generation() {
			t.Fatal("navigation lost the repeated occurrence or changed collection generation")
		}
		if uri, index, ok := v.CurrentFile(); uri != files[0] || index != 1 || !ok {
			t.Fatalf("Host observation disagrees: %v, %d, %v", uri, index, ok)
		}
		if displayed, ok := v.displayedFile(); !ok || displayed.String() != files[1].String() {
			t.Fatal("requested observation was confused with outgoing displayed pixels")
		}
		waitUntilLoaded(t, v)
		v.StepImage(1)
		waitUntilLoaded(t, v)
		if uri, index, ok := v.CurrentFile(); uri.String() != files[1].String() || index != 2 || !ok {
			t.Fatalf("ordinary next after repeated occurrence = %v, %d, %v", uri, index, ok)
		}
	})
}
