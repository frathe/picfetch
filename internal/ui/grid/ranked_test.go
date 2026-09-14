package grid

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/frathe/picfetch/internal/uitest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
)

type rankedCountingURI struct {
	fyne.URI
	reads *atomic.Int64
}

func (u rankedCountingURI) Path() string {
	u.reads.Add(1)
	return u.URI.Path()
}

func TestRankedVisitUpdatesUseCapturedSourceIndexes(t *testing.T) {
	h := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	const prefix, updates = 4096, 32
	files := make([]fyne.URI, prefix, prefix+len(h.files))
	for i := range files {
		files[i] = storage.NewFileURI("/unranked.jpg")
	}
	h.files = append(files, h.files...)
	var reads atomic.Int64
	for i, source := range h.files {
		h.files[i] = rankedCountingURI{URI: source, reads: &reads}
	}
	g := newOverview(t, h)
	reference := h.FileAt(prefix).Path()
	paths := []string{h.FileAt(prefix + 1).Path(), h.FileAt(prefix + 2).Path()}
	g.OpenRanked(RankedVisit{ReferencePath: reference, Paths: paths, Revision: 1})
	g.HandleRune('/')
	g.HandleRune('a')
	g.sel.Replace([]int{prefix})
	g.Settle()
	reads.Store(0)
	for revision := uint64(2); revision < updates+2; revision++ {
		g.OpenRanked(RankedVisit{ReferencePath: reference, Paths: paths, Revision: revision})
	}
	g.Settle()
	if got := reads.Load(); got > updates*200+1000 {
		t.Fatalf("rank revisions rescanned the collection: %d path reads for %d updates of three ranked sources", got, updates)
	} else {
		t.Logf("rank update path reads: %d", got)
	}
	if g.Query() != "a" || !slices.Equal(g.Selection(), []int{prefix}) || !slices.Equal(g.ResultIndexes(), []int{prefix}) {
		t.Fatalf("bounded updates lost filtering or selection: %q %v %v", g.Query(), g.Selection(), g.ResultIndexes())
	}
	slices.Reverse(h.files)
	h.gen++
	g.FilesChanged()
	g.OpenRanked(RankedVisit{ReferencePath: reference, Paths: paths, Revision: updates + 2})
	if !slices.Equal(g.ResultIndexes(), []int{2}) {
		t.Fatalf("new source generation reused old indexes: %v", g.ResultIndexes())
	}
}

func TestRankedVisitInitialOrderAndFilter(t *testing.T) {
	h := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	g := newOverview(t, h)
	g.OpenRanked(RankedVisit{Paths: []string{h.FileAt(2).Path(), h.FileAt(0).Path()}, Revision: 1, Progress: Progress{Processed: 1, Total: 3}})
	if !g.Visible() || !slices.Equal(g.ResultIndexes(), []int{2, 0}) {
		t.Fatalf("ranked result: visible=%v indexes=%v", g.Visible(), g.ResultIndexes())
	}
	var containsProgress func(fyne.CanvasObject) bool
	containsProgress = func(object fyne.CanvasObject) bool {
		if !object.Visible() {
			return false
		}
		if object == g.rankProgress {
			return true
		}
		if c, ok := object.(*fyne.Container); ok {
			for _, child := range c.Objects {
				if containsProgress(child) {
					return true
				}
			}
		}
		return false
	}
	if !containsProgress(g.overlay) {
		t.Fatal("progress bar is missing from the visible Grid tree")
	}
	g.SetRankedProgress(Progress{Processed: 3, Total: 3, Complete: true})
	if containsProgress(g.overlay) {
		t.Fatal("completed scan retained its loading bar")
	}
	if !g.Visible() || !slices.Equal(g.ResultIndexes(), []int{2, 0}) {
		t.Fatal("completing the scan hid its ranked matches")
	}
	g.SetRankedProgress(Progress{Total: 3})
	if !containsProgress(g.overlay) {
		t.Fatal("a new scan did not restore its loading bar")
	}
	g.HandleRune('/')
	g.HandleRune('a')
	if !slices.Equal(g.ResultIndexes(), []int{0}) {
		t.Fatalf("filtered rank: %v", g.ResultIndexes())
	}
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if !slices.Equal(g.ResultIndexes(), []int{2, 0}) {
		t.Fatalf("restored rank: %v", g.ResultIndexes())
	}
}

func TestRankedVisitIdentityAndFrozenOrigin(t *testing.T) {
	h := hostWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
	g := newOverview(t, h)
	backs := 0
	g.OpenSubset([]string{h.FileAt(0).Path(), h.FileAt(1).Path()}, func() { backs++ })
	origin := g.CaptureVisit()
	g.OpenRanked(RankedVisit{Paths: []string{h.FileAt(2).Path(), h.FileAt(0).Path()}, Revision: 1})
	g.sel.Replace([]int{2, 0})
	g.HandleRune('/')
	g.HandleRune('a')
	g.OpenRanked(RankedVisit{Paths: []string{h.FileAt(0).Path(), h.FileAt(3).Path(), h.FileAt(2).Path()}, Revision: 2})
	if !slices.Equal(g.ResultIndexes(), []int{0}) || !slices.Equal(g.Selection(), []int{0, 2}) {
		t.Fatalf("filter lost hidden selected identities: %v / %v", g.ResultIndexes(), g.Selection())
	}
	g.OpenRanked(RankedVisit{Paths: []string{h.FileAt(3).Path(), h.FileAt(0).Path()}, Revision: 3})
	if !slices.Equal(g.Selection(), []int{0}) {
		t.Fatalf("evicted selection survived: %v", g.Selection())
	}
	g.RestoreVisit(origin)
	g.subsetBack.OnTapped()
	if backs != 1 {
		t.Fatal("restored cohort lost Back to map command")
	}
}

func TestRankedVisitPixelsAfterEmptyPreparation(t *testing.T) {
	h := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	for i, name := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		h.files[i] = uitest.TempJPEGURI(t, name, 16, 16, color.NRGBA{R: 210, G: 45, B: 70, A: 255})
	}
	g := newOverview(t, h)
	g.win.SetContent(g.overlay)
	g.win.Resize(fyne.NewSize(800, 600))
	if err := g.Warm(); err != nil {
		t.Fatal(err)
	}
	g.Toggle()
	ref := h.FileAt(0).Path()
	g.OpenRanked(RankedVisit{ReferencePath: ref, Revision: 1})
	g.OpenRanked(RankedVisit{ReferencePath: ref, Paths: []string{h.FileAt(2).Path(), h.FileAt(1).Path()}, Revision: 2})
	g.Settle()
	capture := g.win.Canvas().Capture()
	redPixels := 0
	for y := capture.Bounds().Min.Y; y < capture.Bounds().Max.Y; y++ {
		for x := capture.Bounds().Min.X; x < capture.Bounds().Max.X; x++ {
			r, green, b, _ := capture.At(x, y).RGBA()
			if r > 180*257 && green < 100*257 && b < 100*257 {
				redPixels++
			}
		}
	}
	if redPixels < 1000 {
		t.Fatalf("ranked thumbnails painted only %d red pixels", redPixels)
	}
}

func TestRankedVisitReferenceLeadsFilteringAndActions(t *testing.T) {
	h := hostWith(t, "match-a.jpg", "match-b.jpg", "reference.jpg")
	g := newOverview(t, h)
	paths := []string{h.FileAt(1).Path(), h.FileAt(2).Path(), h.FileAt(0).Path(), h.FileAt(2).Path()}
	g.OpenRanked(RankedVisit{ReferencePath: h.FileAt(2).Path(), Paths: paths, Revision: 1})
	paths[0] = "/changed-after-admission"
	if got := g.ResultIndexes(); !slices.Equal(got, []int{2, 1, 0}) {
		t.Fatalf("reference must lead the deduplicated displayed list: %v", got)
	}
	typeQuery(g, "reference")
	if got := g.ResultIndexes(); !slices.Equal(got, []int{2}) {
		t.Fatalf("filename filtering excluded the reference: %v", got)
	}
	var opened Visit
	g.SetOnRankedOpen(func(visit Visit) { opened = visit })
	g.wrap.Select(0)
	if !slices.Equal(h.shown, []int{2}) || !slices.Equal(opened.Results, []string{h.FileAt(2).Path()}) {
		t.Fatalf("opening reference did not use the displayed source: shown=%v visit=%+v", h.shown, opened)
	}
	g.OpenRanked(RankedVisit{ReferencePath: h.FileAt(0).Path(), Paths: []string{h.FileAt(1).Path(), h.FileAt(2).Path()}, Revision: 2})
	if got := g.ResultIndexes(); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("new reference retained the prior reference order/filter: %v", got)
	}
}

func TestRankedVisitReferenceLimitAndRenderedFirstPosition(t *testing.T) {
	var names []string
	for i := range 33 {
		names = append(names, fmt.Sprintf("%02d.jpg", i))
	}
	h := hostWith(t, names...)
	h.files[32] = uitest.TempJPEGURI(t, "reference.jpg", 16, 16, color.NRGBA{R: 225, G: 30, B: 30, A: 255})
	for i := range 32 {
		h.files[i] = uitest.TempJPEGURI(t, names[i], 16, 16, color.NRGBA{R: 30, G: 30, B: 225, A: 255})
	}
	g := newOverview(t, h)
	g.win.SetContent(g.overlay)
	g.win.Resize(fyne.NewSize(800, 600))
	paths := make([]string, 32)
	for i := range paths {
		paths[i] = h.FileAt(i).Path()
	}
	g.OpenRanked(RankedVisit{ReferencePath: h.FileAt(32).Path(), Paths: paths, Revision: 1})
	g.Settle()
	indexes := g.ResultIndexes()
	if len(indexes) != 31 || indexes[0] != 32 || indexes[30] != 29 {
		t.Fatalf("reference displaced or exceeded thirty other matches: %v", indexes)
	}
	// Root refreshes the complete surface after applying a ranked visit.
	g.win.Content().Refresh()
	capture := g.win.Canvas().Capture()
	redX, blueX := capture.Bounds().Max.X, capture.Bounds().Max.X
	redY := capture.Bounds().Max.Y
	for y := capture.Bounds().Min.Y; y < capture.Bounds().Max.Y; y++ {
		for x := capture.Bounds().Min.X; x < capture.Bounds().Max.X; x++ {
			r, green, b, _ := capture.At(x, y).RGBA()
			if r > 180*257 && green < 100*257 && b < 100*257 {
				redX, redY = min(redX, x), min(redY, y)
			}
		}
	}
	if redX == capture.Bounds().Max.X {
		t.Fatal("reference thumbnail is missing from the rendered Grid")
	}
	for x := capture.Bounds().Min.X; x < capture.Bounds().Max.X; x++ {
		r, green, b, _ := capture.At(x, redY+cellSize/2).RGBA()
		if b > 180*257 && r < 100*257 && green < 100*257 {
			blueX = min(blueX, x)
		}
	}
	if blueX == capture.Bounds().Max.X || redX >= blueX {
		t.Fatalf("rendered reference is not before its first match: red x=%d blue x=%d", redX, blueX)
	}
}

func TestRankedVisitReferenceOutlineLayersAndRecycledCells(t *testing.T) {
	h := hostWith(t, "match.jpg", "reference.jpg")
	g := newOverview(t, h)
	g.OpenRanked(RankedVisit{ReferencePath: h.FileAt(1).Path(), Paths: []string{h.FileAt(0).Path()}, Revision: 1})
	cell := g.wrap.CreateItem().(*fyne.Container)
	g.win.SetContent(cell)
	g.win.Resize(fyne.NewSize(cellSize, cellSize))
	g.highlight = 1
	g.wrap.UpdateItem(0, cell)
	g.Settle()
	_, tint, ring, _ := unpackGridCell(cell)
	var reference *canvas.Rectangle
	referenceIndex, tintIndex, ringIndex := -1, -1, -1
	for i, object := range cell.Objects {
		switch object {
		case tint:
			tintIndex = i
		case ring:
			ringIndex = i
		default:
			if rectangle, ok := object.(*canvas.Rectangle); ok && rectangle.StrokeWidth > 0 {
				reference, referenceIndex = rectangle, i
			}
		}
	}
	if reference == nil || !reference.Visible() {
		t.Fatal("reference outline is absent from the displayed cell tree")
	}
	if referenceIndex >= tintIndex || referenceIndex >= ringIndex {
		t.Fatal("reference outline must be below ordinary selection and highlight layers")
	}
	if purplePixels(g.win.Canvas().Capture()) < 100 {
		t.Fatal("reference outline did not paint purple pixels")
	}
	g.highlight = 0
	g.wrap.UpdateItem(0, cell)
	g.Settle()
	if !ring.Visible() || purplePixels(g.win.Canvas().Capture()) != 0 {
		t.Fatal("reference purple painted over the ordinary highlight ring")
	}
	g.wrap.UpdateItem(1, cell)
	if reference.Visible() {
		t.Fatal("recycled match cell retained the reference outline")
	}
	g.wrap.UpdateItem(0, cell)
	if !reference.Visible() {
		t.Fatal("reference outline did not return when the cell was reused for it")
	}
	g.OpenRanked(RankedVisit{ReferencePath: h.FileAt(0).Path(), Paths: []string{h.FileAt(1).Path()}, Revision: 2})
	g.wrap.UpdateItem(1, cell)
	if reference.Visible() {
		t.Fatal("former reference retained its outline after a reference change")
	}
	g.Close()
	g.Toggle()
	g.wrap.UpdateItem(0, cell)
	if reference.Visible() {
		t.Fatal("ordinary Grid cell retained a ranked reference outline")
	}
	g.Settle()
}

func purplePixels(img image.Image) int {
	count := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 130*257 && r < 210*257 && g < 120*257 && b > 210*257 {
				count++
			}
		}
	}
	return count
}

func TestRankedVisitRestoresDuplicateOccurrences(t *testing.T) {
	for _, selected := range []int{0, 2} {
		t.Run(fmt.Sprint(selected), func(t *testing.T) {
			h := hostWith(t, "a.jpg", "b.jpg", "a.jpg")
			h.files[2] = h.files[0]
			g := newOverview(t, h)
			g.Toggle()
			g.sel.Replace([]int{selected})
			g.setHighlight(selected)
			origin := g.CaptureVisit()
			g.OpenRanked(RankedVisit{ReferencePath: h.FileAt(1).Path(), Revision: 1})
			g.RestoreVisit(origin)
			if !slices.Equal(g.Selection(), []int{selected}) || g.Highlight() != selected {
				t.Fatalf("restored selection=%v highlight=%d, want occurrence %d", g.Selection(), g.Highlight(), selected)
			}
		})
	}
}
