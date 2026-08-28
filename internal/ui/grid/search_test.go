package grid

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/uitest"
)

// --- search ----------------------------------------------------------------

// TestHandleRune_FilteringResetsTheKeyboardCursorToo: same reset as the
// ring's, since a cursor left past the end of the filtered set would send
// the first arrow key somewhere the user never was.
func TestHandleRune_FilteringResetsTheKeyboardCursorToo(t *testing.T) {
	host := hostWith(t, "moon.jpg", "a.jpg", "b.jpg", "c.jpg")
	host.index = 3
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.Toggle()

	typeQuery(g, "moon")
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})

	if g.Highlight() != 0 {
		t.Errorf("Highlight() = %d, want it to stay at 0 - the filtered grid has a single cell", g.Highlight())
	}
}

// TestHandleRune_NoMatchesLeavesTheKeyboardCursorAlone: an empty grid has
// no cell to put a cursor on, and widening the query again must not have
// left one pointing into the set that was filtered away.
func TestHandleRune_NoMatchesLeavesTheKeyboardCursorAlone(t *testing.T) {
	g := newOverview(t, hostWith(t, "a.jpg", "b.jpg", "c.jpg"))
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.Toggle()

	typeQuery(g, "zzz")
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if g.Highlight() != 0 {
		t.Errorf("Highlight() = %d, want 0 with nothing to highlight", g.Highlight())
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if g.Highlight() != 1 {
		t.Errorf("Highlight() = %d, want 1 - Right from the reset cursor once every cell is back", g.Highlight())
	}
}

func TestHandleRune_SlashOpensSearch(t *testing.T) {
	g, _ := openGrid(t, "a.jpg")

	g.HandleRune('/')

	if !g.Searching() {
		t.Error("a typed / should open search mode")
	}
	if g.Query() != "" {
		t.Errorf("Query() = %q, want empty - the activating / must not land in the query itself", g.Query())
	}
}

func TestHandleRune_QueryFiltersToMatchingNames(t *testing.T) {
	g, _ := openGrid(t, "sunset.jpg", "moon.jpg", "sunrise.jpg")

	typeQuery(g, "sun")

	if g.Query() != "sun" {
		t.Errorf("Query() = %q, want %q", g.Query(), "sun")
	}
	// Length is what GridWrap itself calls to size the grid, so this is the
	// cell count the user actually sees.
	if got := g.wrap.Length(); got != 2 {
		t.Errorf("grid length = %d, want 2 - only sunset.jpg and sunrise.jpg match %q", got, "sun")
	}
}

func TestHandleRune_MatchingIsCaseInsensitive(t *testing.T) {
	g, _ := openGrid(t, "Sunset.JPG", "moon.jpg")

	typeQuery(g, "sUnSeT")

	if got := g.wrap.Length(); got != 1 {
		t.Errorf("grid length = %d, want 1 - matching should ignore case on both sides", got)
	}
}

// TestHandleRune_FilteringResetsHighlightToFirstMatch: the highlight is a
// display index, so a filter that shortens the grid under it would leave it
// pointing past the last cell.
func TestHandleRune_FilteringResetsHighlightToFirstMatch(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg", "moon.jpg")
	host.index = 2
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.Toggle()

	if g.Highlight() != 2 {
		t.Fatalf("Highlight() = %d, want it to start on the current image (2)", g.Highlight())
	}

	typeQuery(g, "a")

	if g.Highlight() != 0 {
		t.Errorf("Highlight() = %d, want 0 - only one cell is left to highlight", g.Highlight())
	}
}

func TestHandleRune_NoMatchesShowsAnEmptyGrid(t *testing.T) {
	g, _ := openGrid(t, "a.jpg", "b.jpg")

	typeQuery(g, "zzz")

	if got := g.wrap.Length(); got != 0 {
		t.Errorf("grid length = %d, want 0 - nothing matches %q", got, "zzz")
	}
}

func TestSearchBar_HiddenUntilSearchOpens(t *testing.T) {
	g, _ := openGrid(t, "a.jpg")

	if g.searchBar.Visible() {
		t.Error("the search bar should stay hidden until / opens it")
	}

	g.HandleRune('/')
	if !g.searchBar.Visible() {
		t.Error("/ should show the search bar")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.searchBar.Visible() {
		t.Error("clearing the search should hide the bar again")
	}
}

func TestSearchBar_ShowsQueryAndMatchCount(t *testing.T) {
	g, _ := openGrid(t, "sunset.jpg", "moon.jpg", "sunrise.jpg")

	typeQuery(g, "sun")

	if want := fmt.Sprintf(lang.L("Search: %s"), "sun"); g.searchLabel.Text != want {
		t.Errorf("search label = %q, want %q", g.searchLabel.Text, want)
	}
	if want := fmt.Sprintf(lang.L("%d of %d"), 2, 3); g.countLabel.Text != want {
		t.Errorf("count label = %q, want %q", g.countLabel.Text, want)
	}
}

// TestSearchBar_EmptyNoticeOnlyWhenNothingMatches: an empty grid with no
// explanation reads as a bug, so the one state that draws no cells at all
// says why.
func TestSearchBar_EmptyNoticeOnlyWhenNothingMatches(t *testing.T) {
	g, _ := openGrid(t, "a.jpg", "b.jpg")

	typeQuery(g, "a")
	if g.empty.Visible() {
		t.Error("the empty notice should stay hidden while something still matches")
	}

	g.HandleRune('z')
	if !g.empty.Visible() {
		t.Error("the empty notice should appear once nothing matches")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if g.empty.Visible() {
		t.Error("the empty notice should go away again once a match comes back")
	}
}

// --- applyVisibleFilter's visibility-read budget ----------------------------

// bulkFakeHost builds a host over n files with no real image content.
// applyVisibleFilter never decodes a pixel - it only reads FileAt(i).Name()
// and the model's installed Visibility - so a FakeURI is enough, and it
// spares the perf-shaped test below from writing n real files to disk.
func bulkFakeHost(n int) *fakeHost {
	uris := make([]fyne.URI, n)
	for i := range uris {
		uris[i] = uitest.FakeURI{FileName: fmt.Sprintf("f%d.jpg", i), Ext: ".jpg", Mime: "image/jpeg"}
	}
	return &fakeHost{files: uris}
}

// installPairedGroups fabricates a duplicate-group snapshot pairing every
// two consecutive host indices - (0,1), (2,3), ... - directly through
// Install, so every index in an n-file host has something for hide to
// filter on without a single real hash being computed.
func installPairedGroups(g *Overview, n int) {
	sizes := make([]int, n)
	reps := make([]int, n)
	for i := range n {
		sizes[i] = 2
		reps[i] = i &^ 1
	}
	g.dupes.Install(dupes.Groups{Sizes: sizes, Reps: reps, Dist: 10})
}

// TestApplyVisibleFilter_TakesOneVisibilityReadPerPass is the regression
// guard for the defect this stage fixes: applyVisibleFilter used to take
// the model mutex twice per host index (RepresentativeOf, IsHiddenExtra)
// plus three more before the loop. The read is measured tightly around a
// single call - nothing else here touches the model - so a reappearance of
// the per-index pattern cannot hide behind unrelated harness activity.
// Running the same assertion at two very different file counts is what
// proves the delta is independent of n, not just equal to 1 by accident at
// one size.
func TestApplyVisibleFilter_TakesOneVisibilityReadPerPass(t *testing.T) {
	for _, n := range []int{8, 4000} {
		host := bulkFakeHost(n)
		g := newOverview(t, host)
		installPairedGroups(g, n)
		g.dupes.SetHideDuplicates(true)

		before := g.dupes.VisibilityReads()
		g.applyVisibleFilter(true, -1)
		delta := g.dupes.VisibilityReads() - before

		if delta != 1 {
			t.Errorf("n=%d: VisibilityReads delta = %d, want exactly 1", n, delta)
		}
	}
}

// --- applyVisibleFilter's filter-combination parity -------------------------

// fixedGroupHost builds a 4-file host with one duplicate group fabricated
// directly through Install, bypassing hashing entirely: host 0 is the
// pair's representative, host 1 its extra, hosts 2 and 3 are unrelated
// uniques. Every parity test below shares this exact shape, so the
// expected g.matches slices are easy to state and independent of dHash
// behavior. Callers set g.searching/g.query/g.browseHost/hide themselves
// and call applyVisibleFilter directly, rather than going through
// rebuildFilter/SetHideDuplicates/SetBrowsingDuplicates - those would call
// rebuildGroups or hashRemaining and overwrite this fabricated snapshot
// with one computed from the (blank) fixture images.
func fixedGroupHost(t *testing.T) (*Overview, *fakeHost) {
	t.Helper()

	host := hostWith(t, "sun-a.jpg", "sun-b.jpg", "moon.jpg", "star.jpg")
	g := newOverview(t, host)
	g.dupes.Install(dupes.Groups{
		Sizes: []int{2, 2, 1, 1},
		Reps:  []int{0, 0, 2, 3},
		Dist:  10,
	})

	return g, host
}

// assertMatches fails unless g.matches holds exactly want, in order.
func assertMatches(t *testing.T, g *Overview, want ...int) {
	t.Helper()

	if len(g.matches) != len(want) {
		t.Fatalf("matches = %v, want %v", g.matches, want)
	}
	for i, w := range want {
		if g.matches[i] != w {
			t.Fatalf("matches = %v, want %v", g.matches, want)
		}
	}
}

// TestApplyVisibleFilter_NoFilterLeavesMatchesNil guards the invariant
// count() and fileIndex() both depend on: nil means "no filter active", not
// "a filter that happens to match everything". A pass that produced an
// all-indices slice instead of nil here would be a silent behaviour change.
func TestApplyVisibleFilter_NoFilterLeavesMatchesNil(t *testing.T) {
	g, _ := fixedGroupHost(t)

	g.applyVisibleFilter(true, -1)

	if g.matches != nil {
		t.Fatalf("matches = %v, want nil with no search, hide, or browse active", g.matches)
	}
}

func TestApplyVisibleFilter_SearchOnly(t *testing.T) {
	g, _ := fixedGroupHost(t)
	g.searching = true
	g.query = "sun"

	g.applyVisibleFilter(true, -1)

	assertMatches(t, g, 0, 1)
}

func TestApplyVisibleFilter_HideOnly(t *testing.T) {
	g, _ := fixedGroupHost(t)
	g.dupes.SetHideDuplicates(true)

	g.applyVisibleFilter(true, -1)

	assertMatches(t, g, 0, 2, 3)
}

func TestApplyVisibleFilter_BrowseOnly(t *testing.T) {
	g, _ := fixedGroupHost(t)
	g.browseHost = 0

	g.applyVisibleFilter(true, -1)

	assertMatches(t, g, 0, 1)
}

func TestApplyVisibleFilter_SearchAndHide(t *testing.T) {
	g, _ := fixedGroupHost(t)
	g.searching = true
	g.query = "sun"
	g.dupes.SetHideDuplicates(true)

	g.applyVisibleFilter(true, -1)

	assertMatches(t, g, 0)
}
