package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)

// The viewer and its pools must be created inside the synctest bubble. Three
// held hash reads leave a decode slot for the known groups before opening Grid.
type gridAnalysisFixture struct {
	v       *viewer
	queue   *uitest.UIQueue
	files   []fyne.URI
	release map[int]func()
}

func newGridAnalysisFixture(t *testing.T, hide bool) *gridAnalysisFixture {
	t.Helper()
	return newGridAnalysisFixtureWithPair(t, hide, nil)
}

func newGridAnalysisFixtureWithPair(t *testing.T, hide bool, pair []fyne.URI) *gridAnalysisFixture {
	t.Helper()
	v := newTestViewer(t)
	f := &gridAnalysisFixture{v: v, queue: &uitest.UIQueue{}, release: make(map[int]func())}
	v.grid.SetUIQueue(f.queue)
	seeds := []int{1, 1, 99, 99, 99, 777, 888, 44}
	for i, name := range []string{"a.jpg", "b.jpg", "c-source.jpg", "d-extra.jpg", "e-new.jpg", "f-unrelated.jpg", "g-held.jpg", "h-unique.jpg"} {
		w, h := 64, 48
		if i == 4 {
			w, h = 192, 144 // A later matching member becomes representative.
		}
		u := uitest.PatternedJPEGURISize(t, name, seeds[i], w, h)
		if pair != nil && (i == 2 || i == 3) {
			u = pair[i-2]
		}
		if i >= 4 && i <= 6 {
			data, err := os.ReadFile(u.Path())
			if err != nil {
				t.Fatal(err)
			}
			release := make(chan struct{})
			var once sync.Once
			f.release[i] = func() { once.Do(func() { close(release) }) }
			u = uitest.ReaderURI(u, func() (io.ReadCloser, error) {
				reader, first := bytes.NewReader(data), true
				return uitest.ReadCloser{ReadFunc: func(p []byte) (int, error) {
					if first {
						first = false
						<-release
					}
					return reader.Read(p)
				}, CloseFunc: func() error { return nil }}, nil
			})
		}
		f.files = append(f.files, u)
	}
	// Registered after newTestViewer: even Fatal releases reads before its drain.
	t.Cleanup(func() {
		for _, release := range f.release {
			release()
		}
		v.grid.Stop()
		v.grid.Settle()
	})
	dropAndWait(t, v, f.files...)
	v.grid.SetHideDuplicates(true)
	f.deliver()
	if v.dupes.GroupSize(0) != 2 || v.dupes.GroupSize(2) != 2 || v.dupes.GroupSize(7) != 1 {
		t.Fatal("fixture must publish two distinct pairs and a unique image")
	}
	for _, i := range []int{4, 5, 6} {
		if _, known := v.dupes.Hash(f.files[i].String()); known {
			t.Fatal("fixture published a held source before release")
		}
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	v.grid.SetHideDuplicates(hide)
	f.deliver()
	stubKeyModifiers(t, v, fyne.KeyModifierShift)
	return f
}

func (f *gridAnalysisFixture) deliver() {
	for {
		synctest.Wait()
		if !f.queue.Drain() {
			return
		}
	}
}

func (f *gridAnalysisFixture) releaseNext(i int) {
	// Advance synctest's fake clock past the hash notification throttle. This
	// models elapsed analysis time; Wait/Drain, not this timer, observes delivery.
	time.Sleep(time.Second)
	f.release[i]()
	synctest.Wait()
}

func (f *gridAnalysisFixture) browse(t *testing.T) {
	t.Helper()
	for display, host := range f.v.grid.ResultIndexes() {
		if f.v.FileAt(host).String() == f.files[2].String() {
			f.v.grid.SimulateHover(display)
			f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
			return
		}
	}
	t.Fatal("source is absent from ordinary grid")
}

func (f *gridAnalysisFixture) assertResult(t *testing.T, indexes ...int) {
	t.Helper()
	var got, want []string
	for _, i := range f.v.grid.ResultIndexes() {
		got = append(got, f.v.FileAt(i).String())
	}
	for _, i := range indexes {
		want = append(want, f.files[i].String())
	}
	if !slices.Equal(got, want) {
		t.Fatalf("grid result = %v, want source group %v", got, want)
	}
}

func TestGridBrowseDuringAnalysis(t *testing.T) {
	t.Run("known_group", func(t *testing.T) {
		for _, hide := range []bool{false, true} {
			t.Run(fmt.Sprintf("hide_%t", hide), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					f := newGridAnalysisFixture(t, hide)
					f.browse(t)
					f.assertResult(t, 2, 3)
					if f.v.FileAt(f.v.CurrentIndex()).String() != f.files[0].String() {
						t.Fatal("browsing changed the image underneath the grid")
					}
				})
			})
		}
	})
	t.Run("group_updates", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			f := newGridAnalysisFixture(t, true)
			f.browse(t)
			f.assertResult(t, 2, 3)
			f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
			f.releaseNext(4)
			if !f.queue.Drain() {
				t.Fatal("matching input produced no UI notification")
			}
			synctest.Wait()
			if f.queue.Len() == 0 {
				t.Fatal("matching input did not queue a grouping result")
			}
			f.assertResult(t, 2, 3)
			// A normal filter refresh during held group delivery must retain the pair.
			f.v.handleTypedRune('/')
			f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			f.assertResult(t, 2, 3)
			f.deliver()
			f.assertResult(t, 2, 3, 4)
			if f.v.dupes.RepresentativeOf(2) != 4 {
				t.Fatal("larger accepted variant did not become representative")
			}
			f.releaseNext(5)
			f.deliver()
			f.assertResult(t, 2, 3, 4)
			if _, known := f.v.dupes.Hash(f.files[6].String()); known {
				t.Fatal("analysis finished before the partial-update assertion")
			}
			f.release[6]()
			f.v.grid.Settle()
			f.assertResult(t, 2, 3, 4)
		})
	})
	t.Run("exit", func(t *testing.T) {
		for _, hide := range []bool{false, true} {
			for _, exit := range []string{"escape", "shift_d", "g", "close"} {
				t.Run(fmt.Sprintf("%s/hide_%t", exit, hide), func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						f := newGridAnalysisFixture(t, hide)
						f.browse(t)
						f.assertResult(t, 2, 3)
						if exit == "escape" {
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeySpace})
							f.v.handleTypedRune('/')
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
							if f.v.grid.SelectionCount() != 0 || !f.v.grid.Searching() || !f.v.grid.BrowsingDuplicates() {
								t.Fatal("first Escape must clear selection before search and browse")
							}
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
							if f.v.grid.Searching() || !f.v.grid.BrowsingDuplicates() {
								t.Fatal("second Escape must clear search before browse")
							}
						}
						f.releaseNext(4)
						if !f.queue.Drain() {
							t.Fatal("no partial completion to hold across exit")
						}
						synctest.Wait()
						switch exit {
						case "escape":
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
						case "shift_d":
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
						case "g":
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
						case "close":
							f.v.grid.Close()
						}
						for _, release := range f.release {
							release()
						}
						f.v.grid.Settle()
						if f.v.grid.BrowsingDuplicates() || f.v.dupes.HideDuplicates() != hide {
							t.Fatal("late delivery restored browse or changed hide")
						}
						closed := exit == "g" || exit == "close"
						if f.v.grid.Visible() == closed {
							t.Fatal("late delivery changed grid visibility")
						}
						if closed {
							f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
							f.v.grid.Settle()
						}
						if hide {
							// f-unrelated joins the other pair, so hide keeps a.jpg.
							f.assertResult(t, 0, 4, 6, 7)
						} else {
							f.assertResult(t, 0, 1, 2, 3, 4, 5, 6, 7)
						}
					})
				})
			}
		}
	})
	t.Run("source_identity", func(t *testing.T) {
		t.Run("reorder_remove", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newGridAnalysisFixture(t, true)
				f.browse(t)
				f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
				f.releaseNext(4)
				if !f.queue.Drain() {
					t.Fatal("no completion to hold across reorder")
				}
				synctest.Wait()
				ordered := slices.Clone(f.v.state.files)
				slices.Reverse(ordered)
				f.v.state.reorder(ordered)
				f.v.grid.FilesChanged()
				f.assertResult(t, 3, 2)
				f.deliver()
				f.assertResult(t, 4, 3, 2)
				// Remove the original source, not the moved ring or new representative.
				f.v.RemoveFiles([]int{5})
				if f.v.grid.BrowsingDuplicates() {
					t.Fatal("removing the original source did not end browse")
				}
				for _, release := range f.release {
					release()
				}
				f.v.grid.Settle()
				if f.v.grid.BrowsingDuplicates() {
					t.Fatal("stale delivery restored a removed source")
				}
				// The reordered f-unrelated is now the other group's first member.
				f.assertResult(t, 7, 6, 5, 4)
			})
		})
		t.Run("sensitivity", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var pair []fyne.URI
				for i, name := range []string{"c-source.png", "d-extra.png"} {
					pixels := image.NewGray(image.Rect(0, 0, 9, 8))
					for y := range 8 {
						for x := range 9 {
							pixels.SetGray(x, y, color.Gray{Y: uint8(x * 28)})
						}
					}
					if i == 1 {
						pixels.SetGray(1, 0, color.Gray{})
					}
					var data bytes.Buffer
					if err := png.Encode(&data, pixels); err != nil {
						t.Fatal(err)
					}
					pair = append(pair, storage.NewFileURI(uitest.WriteTempFile(t, name, data.Bytes())))
				}
				f := newGridAnalysisFixtureWithPair(t, true, pair)
				f.browse(t)
				f.assertResult(t, 2, 3)
				f.v.SetDuplicateDistance(0)
				synctest.Wait()
				f.assertResult(t, 2, 3) // Established pair survives pending delivery.
				f.deliver()
				if f.v.grid.BrowsingDuplicates() {
					t.Fatal("accepted sensitivity change did not dissolve partial browse")
				}
				f.assertResult(t, 0, 2, 3, 4, 5, 6, 7)
				for _, release := range f.release {
					release()
				}
				f.v.grid.Settle()
				if f.v.grid.BrowsingDuplicates() {
					t.Fatal("late analysis restored the dissolved group")
				}
			})
		})
	})
	t.Run("open_variant", func(t *testing.T) {
		for _, via := range []string{"return", "click"} {
			t.Run(via, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					f := newGridAnalysisFixture(t, true)
					// Visiting the source preloads its known copy through the normal
					// viewer path. The test driver's inline load completion must not
					// race GridWrap's deferred unselect after Return/click.
					f.v.ShowImage(2)
					waitUntilLoaded(t, f.v)
					f.v.ShowImage(0)
					waitUntilLoaded(t, f.v)
					wrap := comparisonGridWrap(t, f.v.grid.Overlay())
					if !gridAnalysisHasText(wrap, "2") {
						t.Fatal("fixture has no rendered duplicate badge before browse")
					}
					f.browse(t)
					f.assertResult(t, 2, 3)
					stubKeyModifiers(t, f.v, 0)
					f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
					wantTitle := fmt.Sprintf("(4/8) [64x48] %s", f.files[3].Path())
					if got := f.v.win.Title(); got != wantTitle {
						t.Fatalf("variant title = %q, want %q", got, wantTitle)
					}
					if gridAnalysisHasText(wrap, "2") || !gridAnalysisHasText(f.v.grid.Overlay(), "2 of 8") {
						t.Fatal("variants must hide badges and report the displayed pair's count")
					}
					if via == "return" {
						f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
					} else {
						wrap.Select(f.v.grid.Highlight())
					}
					// Full waitUntilLoaded includes the deliberately held neighbor preload.
					waitFor(t, "chosen variant", &f.v.load)
					synctest.Wait()
					if f.v.grid.Visible() || !f.v.dupes.Inspecting() || f.v.FileAt(f.v.CurrentIndex()).String() != f.files[3].String() {
						t.Fatal("opening a hidden extra did not keep the chosen file in inspect")
					}
					f.v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					f.deliver()
					f.assertResult(t, 2, 3)
					if !f.v.grid.Visible() || f.v.dupes.Inspecting() {
						t.Fatal("Escape did not return from inspect to the partial variants grid")
					}
				})
			})
		}
	})
}

// Follow the rendered tree and its ancestors' visibility; a detached or hidden
// badge's own text object alone is not evidence of an on-screen badge.
func gridAnalysisHasText(root fyne.CanvasObject, want string) bool {
	if !root.Visible() {
		return false
	}
	if text, ok := root.(*canvas.Text); ok && text.Text == want {
		return true
	}
	var children []fyne.CanvasObject
	switch root := root.(type) {
	case *fyne.Container:
		children = root.Objects
	case fyne.Widget:
		children = test.WidgetRenderer(root).Objects()
	}
	for _, child := range children {
		if gridAnalysisHasText(child, want) {
			return true
		}
	}
	return false
}

// The overview's own behaviour - opening, closing, the highlight, key
// handling, and the thumbnail cache - is covered in internal/ui/grid
// against a fake host. What stays here is the wiring: that G reaches it,
// that it takes over the keyboard while it's up, and that it composes
// correctly with the app's other full-window mode and with a fresh drop.
//
// Those last two are the interesting ones: neither package knows the other
// exists, so the guards that keep the grid and the slideshow from
// overlapping live in this package's dispatcher, and these tests are what
// hold them in place.

func TestHandleKeyEvent_GTogglesGrid(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	warmThumbs(t, v)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	if !v.grid.Visible() {
		t.Fatal("G should open the grid")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	if v.grid.Visible() {
		t.Error("a second G should close the grid")
	}
}

// TestHandleTypedRune_OnlyReachesTheGridWhileItIsUp is the other half of
// the search wiring: typed characters are a grid-only language, so outside
// it they must be dropped rather than quietly accumulating into a query
// that appears the next time the grid opens.
func TestHandleTypedRune_OnlyReachesTheGridWhileItIsUp(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	warmThumbs(t, v)

	v.handleTypedRune('/')
	if v.grid.Searching() {
		t.Fatal("a / typed in the normal image view must not open the grid's search")
	}

	v.grid.Toggle()
	v.handleTypedRune('/')
	v.handleTypedRune('a')

	if !v.grid.Searching() {
		t.Error("a / typed while the grid is up should open its search")
	}
	if v.grid.Query() != "a" {
		t.Errorf("Query() = %q, want %q", v.grid.Query(), "a")
	}
}

// TestHandleTypedRune_GridVisible_SwallowedByDeleteConfirmation: the delete
// card owns the keyboard whole while it is up, the same as in
// handleKeyEvent - a typed character must not edit a search behind it.
func TestHandleTypedRune_GridVisible_SwallowedByDeleteConfirmation(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)
	warmThumbs(t, v)

	v.grid.Toggle()
	v.handleTypedRune('/')
	v.deletion.Request()

	v.handleTypedRune('a')

	if v.grid.Query() != "" {
		t.Errorf("Query() = %q, want it untouched while the delete confirmation is up", v.grid.Query())
	}
}

// TestHandleTypedRune_GridVisible_SwallowedByExportPrompt is the export
// prompt's twin of TestHandleTypedRune_GridVisible_SwallowedByDeleteConfirmation
// above: it also owns the keyboard whole while it's up, so a typed character
// must not reach a search behind it either.
func TestHandleTypedRune_GridVisible_SwallowedByExportPrompt(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)
	warmThumbs(t, v)

	v.grid.Toggle()
	v.handleTypedRune('/')
	v.promptExport()
	// Asserted rather than assumed, unlike the delete twin's own
	// deletion.Request(): promptExport can silently no-op behind any of its
	// guards (canExport, a delete card already up, the prompt already up),
	// and a future one - grid.Visible() is plausible, given requestDelete
	// already special-cases the grid - would leave this test failing on the
	// query below as if the key dispatcher had broken.
	if !v.exportPrompt.Visible() {
		t.Fatal("setup: the export prompt should be up after promptExport")
	}

	v.handleTypedRune('a')

	if v.grid.Query() != "" {
		t.Errorf("Query() = %q, want it untouched while the export prompt is up", v.grid.Query())
	}
}

// TestHandleKeyEvent_GridVisible_SwallowsNavigation is the dispatcher's
// half of the contract: while the grid is up, ordinary navigation must not
// slip through and change what's on screen behind it.
func TestHandleKeyEvent_GridVisible_SwallowsNavigation(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	warmThumbs(t, v)
	v.grid.Toggle()
	before := v.state.index

	// Right is intercepted by the grid (it moves the highlight) rather
	// than falling through to normal next-image navigation.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})

	if v.state.index != before {
		t.Errorf("index changed to %d while the grid was up, want unchanged from %d", v.state.index, before)
	}
	if !v.grid.Visible() {
		t.Error("Right should not close the grid")
	}
}

func TestHandleKeyEvent_GridVisible_ReturnNavigatesAndCloses(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.RGBA{R: 255, A: 255})
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.RGBA{G: 255, A: 255})
	dropAndWait(t, v, a, b)

	warmThumbs(t, v)
	v.grid.Toggle()

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)

	if v.grid.Visible() {
		t.Error("committing a cell should close the grid")
	}
	if v.state.index != 1 {
		t.Errorf("index = %d, want 1 - the highlighted image should now be on screen", v.state.index)
	}
}

// --- composition with the app's other full-window mode --------------------

func TestHandleKeyEvent_GIsIgnoredDuringPictureFrameMode(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	v.togglePictureFrameMode()
	t.Cleanup(func() { settleSlideshow(t, v) })

	warmThumbs(t, v)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})

	if v.grid.Visible() {
		t.Error("the grid should not open while the slideshow owns the screen")
	}
}

func TestEnterPictureFrameMode_ClosesOpenGrid(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	warmThumbs(t, v)
	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("setup: the grid should be open before entering picture-frame mode")
	}

	v.togglePictureFrameMode()
	t.Cleanup(func() { settleSlideshow(t, v) })

	if v.grid.Visible() {
		t.Error("entering picture-frame mode should close the grid")
	}
	if !v.slides.Active() {
		t.Error("picture-frame mode should still turn on")
	}
}

// --- composition with a fresh drop ----------------------------------------

func TestHandleDrop_ClosesOpenGrid(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	warmThumbs(t, v)
	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("setup: the grid should be open before the second drop")
	}

	b := uitest.TempJPEGURI(t, "b.txt", 4, 4, color.White)
	dropAndWaitScan(t, v, b)

	if v.grid.Visible() {
		t.Error("a new drop should close the grid")
	}
}

// Shift+Delete while the grid is up used to be ignored outright. It now
// targets whatever the grid has picked instead - see batch_test.go, which
// owns that behaviour along with the rest of the batch composition.

// --- window title ----------------------------------------------------------

// TestGridHighlight_NamesTheHighlightedFileInTheTitle is this package's
// half of the notification internal/ui/grid emits: with the image view
// hidden behind the overlay, the title is where a thumbnail's file name is
// spelled out - and it must hand the title back on the way out.
func TestGridHighlight_NamesTheHighlightedFileInTheTitle(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "alpha.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "beta.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	warmThumbs(t, v)

	before := v.win.Title()

	v.grid.Toggle()
	if title := v.win.Title(); !strings.Contains(title, "alpha.jpg") {
		t.Fatalf("title = %q, want it to name alpha.jpg - the cell the grid opened on", title)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	title := v.win.Title()
	if !strings.Contains(title, "beta.jpg") {
		t.Errorf("title = %q, want it to name beta.jpg after moving the highlight", title)
	}
	if !strings.Contains(title, "(2/2)") {
		t.Errorf("title = %q, want the position counter alongside the name", title)
	}

	v.grid.Toggle()
	if got := v.win.Title(); got != before {
		t.Errorf("title = %q after closing the grid, want the image view's own title %q back", got, before)
	}
}

// TestGridHighlight_VariantsTitleUsesSizeAndPath: Show-variants replaces the
// grid's basename+counter title with `(index/count) [WxH] /absolute/path`,
// using the already-probed native size rather than a fresh decode, and
// applyTitle drops the [merge]/[shuffle]/sort prefixes for that form.
func TestGridHighlight_VariantsTitleUsesSizeAndPath(t *testing.T) {
	v := newTestViewer(t)
	small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
	large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
	unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
	dropAndWait(t, v, small, large, unique)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	v.SetMergeMode(true)
	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	v.grid.Toggle()
	v.grid.SetBrowsingDuplicates(true)
	v.grid.Settle()
	if !v.grid.BrowsingDuplicates() || !v.grid.Visible() {
		t.Fatal("premises: variants grid up")
	}

	title := v.win.Title()
	wantSmall := fmt.Sprintf("(1/3) [64x48] %s", small.Path())
	wantLarge := fmt.Sprintf("(2/3) [192x144] %s", large.Path())
	if title != wantSmall && title != wantLarge {
		t.Fatalf("variants title = %q, want %q or %q", title, wantSmall, wantLarge)
	}
	if strings.HasPrefix(title, "[merge]") {
		t.Errorf("variants title = %q, must not include [merge]", title)
	}
	if strings.Contains(title, "a.jpg  (") || strings.Contains(title, " — ") {
		t.Errorf("variants title = %q, must not use basename or image-view format", title)
	}

	v.grid.SetBrowsingDuplicates(false)
	v.grid.Settle()
	title = v.win.Title()
	if strings.Contains(title, "[64x48]") || strings.Contains(title, "[192x144]") {
		t.Errorf("after leaving variants, title = %q, want the basename grid title back", title)
	}
	if !strings.HasPrefix(title, "[merge] ") {
		t.Errorf("after leaving variants, title = %q, want [merge] restored", title)
	}
	if !strings.Contains(title, ".jpg") {
		t.Errorf("after leaving variants, title = %q, want a file name", title)
	}
}

func TestGridHighlight_VariantsHoverUpdatesTitleAndHidesMergePrefix(t *testing.T) {
	v := newTestViewer(t)
	small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
	large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
	unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
	dropAndWait(t, v, small, large, unique)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	v.SetMergeMode(true)
	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	v.grid.Toggle()
	v.grid.SetBrowsingDuplicates(true)
	v.grid.Settle()
	if !v.grid.Visible() || !v.grid.BrowsingDuplicates() {
		t.Fatal("premises: variants grid up")
	}

	startID := v.grid.Highlight()
	otherID := 1 - startID

	before := v.win.Title()
	if strings.HasPrefix(before, "[merge]") {
		t.Fatalf("variants title = %q, must not include [merge]", before)
	}

	v.grid.SimulateHover(otherID)
	after := v.win.Title()
	if after == before {
		t.Fatalf("title did not change after hovering the other variant, still %q", after)
	}
	if strings.HasPrefix(after, "[merge]") {
		t.Errorf("title = %q, must not include [merge]", after)
	}

	wantSmall := fmt.Sprintf("(1/3) [64x48] %s", small.Path())
	wantLarge := fmt.Sprintf("(2/3) [192x144] %s", large.Path())
	if after != wantSmall && after != wantLarge {
		t.Errorf("after hover, title = %q, want %q or %q", after, wantSmall, wantLarge)
	}
	if strings.Contains(before, small.Path()) && !strings.Contains(after, large.Path()) {
		t.Errorf("hovered off small, title = %q, want large path", after)
	}
	if strings.Contains(before, large.Path()) && !strings.Contains(after, small.Path()) {
		t.Errorf("hovered off large, title = %q, want small path", after)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	arrow := v.win.Title()
	if strings.HasPrefix(arrow, "[merge]") {
		t.Errorf("after Right, title = %q, must not include [merge]", arrow)
	}
	if arrow != wantSmall && arrow != wantLarge {
		t.Errorf("after Right, title = %q, want %q or %q", arrow, wantSmall, wantLarge)
	}

	v.grid.Toggle()
	if v.grid.Visible() {
		t.Fatal("grid should be closed")
	}
	closed := v.win.Title()
	if strings.Contains(closed, "[64x48]") || strings.Contains(closed, "[192x144]") {
		t.Errorf("closed-grid title = %q, want image-view title, not variants format", closed)
	}
	if !strings.HasPrefix(closed, "[merge] ") {
		t.Errorf("closed-grid title = %q, want [merge] restored on the image-view title", closed)
	}
}

// TestGridHighlight_TitleKeepsTheModePrefixes: the grid's file name takes
// the *base* title's place, not the whole title - the sort/merge/shuffle
// prefixes describe modes that are still on behind it.
func TestGridHighlight_TitleKeepsTheModePrefixes(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "alpha.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)
	warmThumbs(t, v)

	v.SetMergeMode(true)
	v.grid.Toggle()

	title := v.win.Title()
	if !strings.HasPrefix(title, "[merge] ") {
		t.Errorf("title = %q, want the [merge] prefix kept while the grid is up", title)
	}
	if !strings.Contains(title, "alpha.jpg") {
		t.Errorf("title = %q, want it to name the highlighted file", title)
	}
}

func TestShutdownStopsGridAdmission(t *testing.T) {
	application := test.NewApp()
	v, win := buildStartupViewer(application)
	v.grid.SetUIQueue(&uitest.UIQueue{})
	v.compare.SetUIQueue(&uitest.UIQueue{})
	v.mosaicWin.SetUIQueue(&uitest.UIQueue{})
	v.slides.SetUIQueue(&uitest.UIQueue{})
	v.deletion.SetUIQueue(&uitest.UIQueue{})
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })
	src := uitest.TempJPEGURI(t, "source.jpg", 4, 4, color.White)
	v.state.replaceFiles([]fyne.URI{src}, []fyne.URI{src})
	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test lifecycle has no stopped hook")
	}
	original := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(original)
	shutdown()
	if err := v.grid.Warm(); !errors.Is(err, context.Canceled) {
		t.Errorf("shutdown grid admitted warm: %v", err)
	}
	v.grid.Toggle()
	v.grid.Settle()
	if v.grid.Visible() || v.grid.Cached(src) {
		t.Error("shutdown grid reopened or populated its cache")
	}
}
