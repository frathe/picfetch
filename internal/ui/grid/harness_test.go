package grid

import (
	"context"
	"image/color"
	"os"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMain(m *testing.M) {
	// The overlay is built from real widgets, so these need an app for the
	// theme and driver. Each test still gets its own window (see
	// newOverview) - New takes one to maximize on open - but the fyne test
	// driver's windows are not driver.NativeWindow, so winpos.Maximize
	// degrades to a no-op there, the same as every other winpos call this
	// app makes under test.
	test.NewApp()
	os.Exit(m.Run())
}

// fakeHost stands in for the viewer. It records the display actions the
// grid asks for, so a selection or a close can be observed without a real
// window behind it.
type fakeHost struct {
	files []fyne.URI
	index int
	gen   uint64

	// mods is what Modifiers reports - the keyboard state a tap is read
	// against, which a Fyne tap event carries none of. Set around a
	// wrap.Select call to stand in for holding a key while clicking (see
	// the click helper in selection_test.go).
	mods fyne.KeyModifier

	shown     []int
	repaints  int
	unfocused int

	// highlighted records every file index the grid reported for the ring
	// (-1 for "none"), so the window-title notification can be asserted in
	// order without a real title bar.
	highlighted []int

	toasts []string
}

func (f *fakeHost) FileCount() int         { return len(f.files) }
func (f *fakeHost) FileAt(i int) fyne.URI  { return f.files[i] }
func (f *fakeHost) CurrentIndex() int      { return f.index }
func (f *fakeHost) Generation() uint64     { return f.gen }
func (f *fakeHost) ShowImage(i int)        { f.shown = append(f.shown, i) }
func (f *fakeHost) ForceRepaint()          { f.repaints++ }
func (f *fakeHost) Unfocus()               { f.unfocused++ }
func (f *fakeHost) HighlightChanged(i int) { f.highlighted = append(f.highlighted, i) }

func (f *fakeHost) Modifiers() fyne.KeyModifier { return f.mods }

func (f *fakeHost) ShowToast(msg string) { f.toasts = append(f.toasts, msg) }

// hostWith returns a host holding n small real JPEGs - real files because
// the decode path under test actually reads them.
func hostWith(t *testing.T, names ...string) *fakeHost {
	t.Helper()

	uris := make([]fyne.URI, 0, len(names))
	for _, name := range names {
		uris = append(uris, uitest.TempJPEGURI(t, name, 8, 8, color.White))
	}

	return &fakeHost{files: uris}
}

// newOverview builds an Overview over host and a real (test-driver)
// window, closing the window when the test ends - the fixture behind every
// New call in this file, now that New needs a window to maximize on open.
//
// Every overview built here defers its background completions instead of
// letting Fyne's test driver run them inline on the decode worker (see
// uitest.UIQueue and uiqueue.go). Settle is what runs them, on the test
// goroutine, so a test that asserts on the effect of a decode - a painted
// cell, a rebuilt filter, a group that has finished hashing - has to
// Settle first.
func newOverview(t *testing.T, host Host) *Overview {
	t.Helper()

	win := test.NewWindow(nil)
	t.Cleanup(win.Close)

	g := New(host, win, dupes.New(hostSet{host: host}))
	g.SetUIQueue(&uitest.UIQueue{})
	registerJumpObserver(g, host)

	return g
}

// registerJumpObserver stands in for the observer internal/ui registers on
// the model it owns (viewer.jumpIfHiddenExtra, visibility.go): the grid no
// longer jumps the host off a hidden extra itself, it fires the model's
// observers once it has re-filtered, and the app is what turns that into a
// ShowImage. Registering the equivalent here is what lets this package go
// on asserting that a grid transition ends with the host on the group's
// representative - the grid's half of that contract is *when* it notifies.
//
// The real jump - including the inspect guard that makes committing an
// extra out of the variants grid stick - is production code in
// internal/ui and is tested there.
func registerJumpObserver(g *Overview, host Host) {
	g.dupes.OnChange(func() {
		if g.dupes.Inspecting() {
			return
		}
		if i := host.CurrentIndex(); g.dupes.IsHiddenExtra(i) {
			host.ShowImage(g.dupes.RepresentativeOf(i))
		}
	})
}

// hostSet adapts Host to dupes.FileSet, standing in for the adapter the
// app builds the real model from (internal/ui's dupeFileSet): production
// hands grid.New a model the viewer owns, so this is the grid's own way of
// getting an equivalent one over a fakeHost.
//
// KeyAt has to stay a plain lookup, for the same reason the production
// adapter's does. dupes.Model.Compute calls it while holding the model's
// own mutex - faithfully to the code this replaced, which read
// g.host.FileAt(i) under hashMu - so anything here that took a lock, or
// reached back into the model, would deadlock a hashing worker.
type hostSet struct {
	host Host
}

func (s hostSet) Count() int { return s.host.FileCount() }

// KeyAt is the URI string of the file at i, or "" when the host has no
// URI there: the same nil-URI guard every helper in dupes.go applies
// before it touches a fyne.URI, in the one place the model reaches
// through.
func (s hostSet) KeyAt(i int) string {
	u := s.host.FileAt(i)
	if u == nil {
		return ""
	}

	return u.String()
}

func (s hostSet) Generation() uint64 { return s.host.Generation() }

// parkDecodes fills the decode pool with jobs that block until the
// returned unpark runs, so a test can drive the grid - including opening it
// on a cold cache - with nothing actually decoding underneath. That is what
// makes a "hashes are still pending" window deterministic instead of a race
// against four workers that may or may not have hashed everything already.
//
// Unparks and Settles on cleanup, so a test that Fatals mid-window cannot
// leave a parked goroutine behind for the next one. Registered after
// newOverview's window close, and cleanups run last-registered-first, so
// the drain still happens while the widgets are alive. unpark is a OnceFunc:
// calling it yourself, which is the normal shape, is fine.
//
// Two preconditions this doesn't check for you: call it before anything
// else spawns a decode, since it fills every slot in the pool and a
// non-idle pool leaves it blocked waiting for one that will never free up;
// and never call g.Settle while still parked, since Settle's own
// decodes.Wait blocks on the parked jobs returning.
func parkDecodes(t *testing.T, g *Overview) (unpark func()) {
	t.Helper()

	// The pool waits for its slot on the spawned goroutine, so each parker
	// has to report that it really holds one before the caller can be sure
	// its own requests queue behind them.
	holding := make(chan struct{}, thumbConcurrency)
	parked := make(chan struct{})
	unpark = sync.OnceFunc(func() { close(parked) })

	for range thumbConcurrency {
		g.decodes.Go(context.Background(), func(bool) {
			holding <- struct{}{}
			<-parked
		})
	}
	for range thumbConcurrency {
		<-holding
	}

	t.Cleanup(func() {
		unpark()
		g.Settle()
	})

	return unpark
}

// last is the most recent index the grid reported for the ring, or -2 when
// it never reported anything - distinct from the -1 that means "nothing
// highlighted".
func (f *fakeHost) last() int {
	if len(f.highlighted) == 0 {
		return -2
	}

	return f.highlighted[len(f.highlighted)-1]
}

// openGrid builds an overview over the named files, warms every thumbnail
// (so no cell spawns a background decode, see Warm) and opens it - the
// starting state for every search test below.
func openGrid(t *testing.T, names ...string) (*Overview, *fakeHost) {
	t.Helper()

	host := hostWith(t, names...)
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.Toggle()

	return g, host
}

// typeQuery opens search and types q into it, one rune at a time - the way
// the canvas's typed-rune callback delivers them.
func typeQuery(g *Overview, q string) {
	g.HandleRune('/')
	for _, r := range q {
		g.HandleRune(r)
	}
}
