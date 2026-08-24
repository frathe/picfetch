package grid

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestWarm_RecordsDifferenceHash(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	if _, ok := g.hashOf(host.files[0]); !ok {
		t.Fatal("Warm should record a dHash for each decoded thumbnail")
	}
}

func TestRequestThumbnail_CacheHitFillsMissingHash(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.clearHashes()

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)
	g.requestThumbnail(cell, img, 0, host.gen)

	if _, ok := g.hashOf(host.files[0]); !ok {
		t.Fatal("a cache hit should fill a missing dHash from the cached thumbnail")
	}
}

func TestWipeHashesIfStale_DropsHashesOnGenerationChange(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	host.gen++
	g.wipeHashesIfStale()
	if _, ok := g.hashOf(host.files[0]); ok {
		t.Fatal("hashes must drop when host.Generation changes")
	}
}

func hostPatterned(t *testing.T, names []string, seeds []int) *fakeHost {
	t.Helper()
	if len(names) != len(seeds) {
		t.Fatalf("names/seeds length mismatch: %d vs %d", len(names), len(seeds))
	}
	uris := make([]fyne.URI, len(names))
	for i := range names {
		uris[i] = uitest.PatternedJPEGURI(t, names[i], seeds[i])
	}
	return &fakeHost{files: uris}
}

func openPatterned(t *testing.T, names []string, seeds []int) (*Overview, *fakeHost) {
	t.Helper()
	host := hostPatterned(t, names, seeds)
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.Toggle()
	return g, host
}

func pairAndUnique(t *testing.T) (*Overview, *fakeHost) {
	t.Helper()
	return openPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
}

func nearGrayPair() (*image.Gray, *image.Gray) {
	a := image.NewGray(image.Rect(0, 0, 9, 8))
	b := image.NewGray(image.Rect(0, 0, 9, 8))
	for y := range 8 {
		for x := range 9 {
			v := uint8(x * 28)
			a.SetGray(x, y, color.Gray{Y: v})
			b.SetGray(x, y, color.Gray{Y: v})
		}
	}
	b.SetGray(1, 0, color.Gray{Y: 0})
	return a, b
}

func TestSetHideDuplicates_HidesExtrasKeepsUniques(t *testing.T) {
	g, _ := pairAndUnique(t)

	g.SetHideDuplicates(true)

	if !g.HideDuplicates() {
		t.Fatal("HideDuplicates() = false after SetHideDuplicates(true)")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (one of the pair + the unique)", g.count())
	}
	if g.fileIndex(0) != 0 || g.fileIndex(1) != 2 {
		t.Fatalf("visible host indices = [%d, %d], want [0, 2]", g.fileIndex(0), g.fileIndex(1))
	}
	if !g.IsHiddenExtra(1) {
		t.Error("index 1 should be a hidden extra")
	}
	if g.IsHiddenExtra(0) || g.IsHiddenExtra(2) {
		t.Error("the representative and the unique must stay visible")
	}
	if g.RepresentativeOf(1) != 0 {
		t.Errorf("RepresentativeOf(1) = %d, want 0", g.RepresentativeOf(1))
	}
	if g.groupSize(0) != 2 || g.groupSize(2) != 1 {
		t.Errorf("groupSize(0,2) = (%d, %d), want (2, 1)", g.groupSize(0), g.groupSize(2))
	}
	if got, want := g.searchLabel.Text, lang.L("Hiding duplicates"); got != want {
		t.Errorf("search label = %q, want %q", got, want)
	}
	if got, want := g.countLabel.Text, fmt.Sprintf(lang.L("%d of %d"), 2, 3); got != want {
		t.Errorf("count label = %q, want %q", got, want)
	}
}

func TestSetHideDuplicates_IntersectsSearch(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)

	typeQuery(g, "sun")

	if g.count() != 1 {
		t.Fatalf("count() = %d, want 1 (the pair's extra is hidden)", g.count())
	}
	if g.fileIndex(0) != 0 {
		t.Errorf("fileIndex(0) = %d, want 0", g.fileIndex(0))
	}
	if got, want := g.searchLabel.Text, fmt.Sprintf(lang.L("Search: %s"), "sun"); got != want {
		t.Errorf("search label = %q, want %q (search wins over the hide chrome)", got, want)
	}
}

func TestSetHideDuplicates_JumpsToRepresentative(t *testing.T) {
	g, host := pairAndUnique(t)
	host.index = 1

	g.SetHideDuplicates(true)

	if len(host.shown) == 0 || host.shown[len(host.shown)-1] != 0 {
		t.Errorf("ShowImage calls = %v, want a jump to representative 0", host.shown)
	}
}

func TestHandleKey_DTogglesHideDuplicates(t *testing.T) {
	g, _ := pairAndUnique(t)

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.HideDuplicates() || g.count() != 2 {
		t.Fatalf("after D: hide=%v count=%d, want hide=true count=2", g.HideDuplicates(), g.count())
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if g.HideDuplicates() || g.count() != 3 {
		t.Fatalf("after second D: hide=%v count=%d, want hide=false count=3", g.HideDuplicates(), g.count())
	}
}

func TestHandleRune_DIsAQueryCharacterWhileSearching(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	typeQuery(g, "x")

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.HideDuplicates() {
		t.Fatal("KeyD while searching must not toggle hide-duplicates")
	}
	if g.Query() != "x" {
		t.Errorf("Query() = %q, want %q (KeyD is not a typed rune)", g.Query(), "x")
	}

	g.HandleRune('d')
	if g.Query() != "xd" {
		t.Errorf("Query() = %q, want %q", g.Query(), "xd")
	}
	if !g.HideDuplicates() {
		t.Fatal("typing d into a search must leave hide-duplicates on")
	}
}

func TestHandleKey_EscapeTurnsOffHideDuplicatesBeforeClosing(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	typeQuery(g, "sun")
	click(g, host, 0, fyne.KeyModifierShortcutDefault)

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.SelectionCount() != 0 {
		t.Fatal("first Escape should clear the selection")
	}
	if !g.Searching() || !g.HideDuplicates() || !g.Visible() {
		t.Fatal("first Escape should leave search, hide, and the grid up")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.Searching() {
		t.Fatal("second Escape should clear the search")
	}
	if !g.HideDuplicates() || !g.Visible() {
		t.Fatal("second Escape should leave hide on and the grid up")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.HideDuplicates() {
		t.Fatal("third Escape should turn hide-duplicates off")
	}
	if !g.Visible() {
		t.Fatal("third Escape should not also close the grid")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.Visible() {
		t.Error("fourth Escape should close the grid")
	}
}

func TestClose_LeavesHideDuplicatesOn(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)

	g.Close()

	if g.Visible() {
		t.Fatal("Close should hide the grid")
	}
	if !g.HideDuplicates() {
		t.Error("Close must not clear hide-duplicates")
	}
}

func TestHandleKey_GLeavesHideDuplicatesOn(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyG})

	if g.Visible() {
		t.Fatal("G should close the grid")
	}
	if !g.HideDuplicates() {
		t.Error("G must not clear hide-duplicates")
	}
}

func TestApplyDupBadge_ShowsGroupSize(t *testing.T) {
	g, _ := pairAndUnique(t)
	badge := canvas.NewText("", color.White)

	g.applyDupBadge(badge, 0)
	if badge.Visible() {
		t.Fatal("badge should stay hidden while hide-duplicates is off")
	}

	g.SetHideDuplicates(true)
	g.applyDupBadge(badge, 0)
	if !badge.Visible() || badge.Text != "2" {
		t.Errorf("representative badge visible=%v text=%q, want visible text \"2\"", badge.Visible(), badge.Text)
	}

	g.applyDupBadge(badge, 2)
	if badge.Visible() {
		t.Error("a unique cell must hide the badge")
	}

	g.SetHideDuplicates(false)
	g.applyDupBadge(badge, 0)
	if badge.Visible() {
		t.Error("turning hide off must hide the badge")
	}
}

func TestSetHideDuplicates_HashesRemainingWithoutWarm(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)

	g.SetHideDuplicates(true)
	g.Settle()

	if g.count() != 2 {
		t.Fatalf("count() = %d after hashing remaining, want 2", g.count())
	}
	if _, ok := g.hashOf(host.files[0]); !ok {
		t.Fatal("hashRemaining should record a dHash for each file")
	}
	if !g.IsHiddenExtra(1) {
		t.Error("the pair's extra should be hidden once hashes land")
	}
}

func TestSetDuplicateDistance_RegroupsLive(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	a, b := nearGrayPair()
	if d := imaging.Hamming(imaging.DifferenceHash(a), imaging.DifferenceHash(b)); d < 1 || d > imaging.DuplicateMaxDistance {
		t.Fatalf("setup Hamming = %d, want in 1..%d", d, imaging.DuplicateMaxDistance)
	}
	g.rememberHash(host.files[0], a)
	g.rememberHash(host.files[1], b)

	g.SetHideDuplicates(true)
	if g.count() != 1 {
		t.Fatalf("count() = %d at default distance, want 1 (near hashes grouped)", g.count())
	}

	g.SetDuplicateDistance(0)
	if g.count() != 2 {
		t.Fatalf("count() = %d at distance 0, want 2 (exact match only)", g.count())
	}
	if g.IsHiddenExtra(1) {
		t.Error("distance 0 should split the Hamming-1 pair")
	}

	g.SetDuplicateDistance(imaging.DuplicateMaxDistance)
	if g.count() != 1 {
		t.Fatalf("count() = %d after restoring default distance, want 1", g.count())
	}
}

func TestSetBrowsingDuplicates_ShowsOnlyTheGroup(t *testing.T) {
	g, host := pairAndUnique(t) // host 0,1 pair; host 2 unique

	g.SetBrowsingDuplicates(true)

	if !g.BrowsingDuplicates() {
		t.Fatal("BrowsingDuplicates() = false after SetBrowsingDuplicates(true)")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (pair only)", g.count())
	}
	if g.fileIndex(0) != 0 || g.fileIndex(1) != 1 {
		t.Fatalf("visible = [%d, %d], want [0, 1]", g.fileIndex(0), g.fileIndex(1))
	}
	if g.Highlight() != 0 {
		t.Errorf("Highlight() = %d, want 0 (source file)", g.Highlight())
	}
	if len(host.toasts) != 0 {
		t.Errorf("toasts = %v, want none (already hashed)", host.toasts)
	}
}

func TestSetBrowsingDuplicates_NoopOnUnique(t *testing.T) {
	g, host := pairAndUnique(t)
	g.setHighlight(2) // moon.jpg

	g.SetBrowsingDuplicates(true)

	if g.BrowsingDuplicates() {
		t.Fatal("unique file must not enter browse")
	}
	if g.count() != 3 {
		t.Fatalf("count() = %d, want 3 (unfiltered)", g.count())
	}
	if len(host.toasts) != 0 {
		t.Errorf("toasts = %v, want none (hashes already ready)", host.toasts)
	}
}

func TestSetBrowsingDuplicates_ShowsExtrasEvenWhenHideOn(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("setup hide count() = %d, want 2", g.count())
	}
	// Display 0 is host 0 (representative). Extra host 1 is hidden.
	g.setHighlight(0)

	g.SetBrowsingDuplicates(true)

	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (both pair members, unique excluded)", g.count())
	}
	seen := map[int]bool{g.fileIndex(0): true, g.fileIndex(1): true}
	if !seen[0] || !seen[1] {
		t.Fatalf("visible hosts = %v, want 0 and 1 (extra must be shown)", seen)
	}
	if !g.HideDuplicates() {
		t.Fatal("browse must not clear the hide flag")
	}
}

func TestSetBrowsingDuplicates_IntersectsSearch(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.HandleRune('/')
	for _, r := range "sunset" {
		g.HandleRune(r)
	}
	g.SetBrowsingDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (sunset pair)", g.count())
	}
	g.clearSearch()
	if !g.BrowsingDuplicates() || g.count() != 2 {
		t.Fatal("clearing search must leave browse on")
	}
}

func TestSyncTopBar_ShowingDuplicates(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetBrowsingDuplicates(true)
	if got, want := g.searchLabel.Text, lang.L("Showing duplicates"); got != want {
		t.Errorf("searchLabel = %q, want %q", got, want)
	}
	if got, want := g.countLabel.Text, fmt.Sprintf(lang.L("%d of %d"), 2, 3); got != want {
		t.Errorf("countLabel = %q, want %q", got, want)
	}
}

func TestGroupMembers_Pair(t *testing.T) {
	g, _ := pairAndUnique(t)
	got := g.groupMembers(1)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("groupMembers(1) = %v, want [0 1]", got)
	}
	if g.groupMembers(2) != nil {
		t.Fatalf("groupMembers(unique) = %v, want nil", g.groupMembers(2))
	}
}

func TestHandleKey_ShiftDTogglesBrowseDuplicates(t *testing.T) {
	g, host := pairAndUnique(t)
	host.mods = fyne.KeyModifierShift

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.BrowsingDuplicates() || g.count() != 2 {
		t.Fatalf("after Shift+D: browse=%v count=%d, want browse=true count=2", g.BrowsingDuplicates(), g.count())
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if g.BrowsingDuplicates() || g.count() != 3 {
		t.Fatalf("second Shift+D: browse=%v count=%d, want browse=false count=3", g.BrowsingDuplicates(), g.count())
	}
}

func TestHandleKey_PlainDStillTogglesHideWhileNotSearching(t *testing.T) {
	g, host := pairAndUnique(t)
	host.mods = 0
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.HideDuplicates() || g.BrowsingDuplicates() {
		t.Fatal("plain D must toggle hide, not browse")
	}
}

func TestHandleKey_ShiftDWhileSearchingDoesNotBrowse(t *testing.T) {
	g, host := pairAndUnique(t)
	g.HandleRune('/')
	g.HandleRune('x')
	host.mods = fyne.KeyModifierShift
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if g.BrowsingDuplicates() {
		t.Fatal("Shift+D while searching must not browse")
	}
	if g.Query() != "x" {
		t.Errorf("Query() = %q, want %q (KeyD is not a typed rune)", g.Query(), "x")
	}
}

func TestHandleKey_EscapeTurnsOffBrowseBeforeHide(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	host.mods = 0

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.BrowsingDuplicates() {
		t.Fatal("first Escape should leave browse")
	}
	if !g.HideDuplicates() || !g.Visible() {
		t.Fatal("first Escape should leave hide on and the grid up")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.HideDuplicates() {
		t.Fatal("second Escape should turn hide off")
	}
	if !g.Visible() {
		t.Fatal("second Escape should not close the grid")
	}
}

func TestClose_ClearsBrowseLeavesHide(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)

	g.Close()

	if g.Visible() {
		t.Fatal("Close should hide the grid")
	}
	if g.BrowsingDuplicates() {
		t.Error("Close must clear browse")
	}
	if !g.HideDuplicates() {
		t.Error("Close must not clear hide-duplicates")
	}
}

func TestHandleKey_GClearsBrowse(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetBrowsingDuplicates(true)
	host.mods = 0
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyG})
	if g.Visible() {
		t.Fatal("G should close")
	}
	if g.BrowsingDuplicates() {
		t.Error("G/Close must clear browse")
	}
}

func TestSetBrowsingDuplicates_HashesRemainingWithoutWarm(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.Toggle()
	host.index = 0

	g.SetBrowsingDuplicates(true)
	if len(host.toasts) != 1 || host.toasts[0] != lang.L("The images are currently being analyzed") {
		t.Fatalf("toasts = %v, want [%q] while hashing", host.toasts, lang.L("The images are currently being analyzed"))
	}
	g.Settle()

	if !g.BrowsingDuplicates() {
		t.Fatal("browse should turn on after remaining files hash")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d after hashing remaining, want 2", g.count())
	}
	if len(host.toasts) != 1 {
		t.Errorf("toasts after Settle = %v, want still one (no second toast)", host.toasts)
	}
}

func TestApplyFilter_BrowsePendingDoesNotCollapseGrid(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.Toggle()
	host.index = 0
	// Toggle's GridWrap decode fills the thumb cache; hashRemaining would
	// then remember those hits on this goroutine and the pair would already
	// be a group. Drain and drop them so hashes are still pending, matching
	// SetBrowsingDuplicates without Warm.
	g.Settle()
	g.clearHashes()
	g.thumbs.Purge()

	g.SetBrowsingDuplicates(true)
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("count() = %d while hashes pending, want 3 (not collapsed to the source cell)", g.count())
	}
	if !g.BrowsingDuplicates() {
		t.Fatal("BrowsingDuplicates() = false while hashes pending, want true")
	}

	g.Settle()

	if !g.BrowsingDuplicates() {
		t.Fatal("browse should stay on after remaining files hash")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d after hashing remaining, want 2", g.count())
	}
}

func TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	a, b := nearGrayPair()
	g.rememberHash(host.files[0], a)
	g.rememberHash(host.files[1], b)
	g.Toggle()
	g.SetBrowsingDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("setup count() = %d, want 2", g.count())
	}

	g.SetDuplicateDistance(0)
	if g.BrowsingDuplicates() {
		t.Fatal("distance 0 should exit browse when the pair splits")
	}
}

func injectHashes(t *testing.T, g *Overview, host *fakeHost, hs []uint64) {
	t.Helper()
	if len(hs) != len(host.files) {
		t.Fatalf("injectHashes: %d hashes for %d files", len(hs), len(host.files))
	}
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	if g.hashes == nil {
		g.hashes = make(map[string]uint64)
	}
	g.hashGen = host.Generation()
	for i, h := range hs {
		g.hashes[host.files[i].String()] = h
	}
}

func TestSetHideDuplicates_ChainDoesNotHideUnrelated(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	g := newOverview(t, host)
	injectHashes(t, g, host, []uint64{1 << 63, 1<<63 | 0x3FF, 1<<63 | 0xFFFFF})
	// A literal 10: this fixture is built at Hamming 10/10/20 to exercise
	// linkage, so it pins the threshold it was written for rather than
	// tracking the shipped default.
	g.SetDuplicateDistance(10)

	g.SetHideDuplicates(true)

	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (A visible, B hidden extra, C unique)", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Error("B is within distance 10 of A and must be an extra")
	}
	if g.IsHiddenExtra(2) {
		t.Error("C is Hamming 20 from A and must not be hidden as A's extra")
	}
	if g.RepresentativeOf(2) != 2 {
		t.Errorf("RepresentativeOf(2) = %d, want 2", g.RepresentativeOf(2))
	}
	if g.groupSize(2) != 1 {
		t.Errorf("groupSize(2) = %d, want 1 (C is hashed-and-unique, not unhashed)", g.groupSize(2))
	}
	if g.groupSize(0) != 2 {
		t.Errorf("groupSize(0) = %d, want 2 (not the whole set)", g.groupSize(0))
	}
}

func TestSetBrowsingDuplicates_ChainDoesNotListUnrelated(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	host.index = 0
	g := newOverview(t, host)
	injectHashes(t, g, host, []uint64{1 << 63, 1<<63 | 0x3FF, 1<<63 | 0xFFFFF})
	g.SetDuplicateDistance(10)

	g.SetBrowsingDuplicates(true)

	if !g.BrowsingDuplicates() {
		t.Fatal("A has a duplicate (B); browse must turn on")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (A and B, not C)", g.count())
	}
	seen := map[int]bool{g.fileIndex(0): true, g.fileIndex(1): true}
	if !seen[0] || !seen[1] || seen[2] {
		t.Fatalf("visible hosts = %v, want 0 and 1 only", seen)
	}
}

func TestSetHideDuplicates_HubSpokesDoNotHideUnrelated(t *testing.T) {
	host := hostWith(t, "hub.jpg", "spoke-a.jpg", "spoke-b.jpg")
	g := newOverview(t, host)
	const hub uint64 = 0xFFFF000000000000
	injectHashes(t, g, host, []uint64{hub, hub ^ 0x3FF, hub ^ (0x3FF << 10)})
	g.SetDuplicateDistance(10)

	g.SetHideDuplicates(true)

	if g.groupSize(0) != 2 {
		t.Fatalf("groupSize(0) = %d, want 2 (one spoke, not both)", g.groupSize(0))
	}
	if g.IsHiddenExtra(2) {
		t.Error("spoke B is 20 from spoke A and must not hide as hub's extra")
	}
	if g.RepresentativeOf(2) == 0 {
		t.Error("spoke B must not list the hub as representative")
	}
}

func TestSetBrowsingDuplicates_HubSpokesDoNotListUnrelated(t *testing.T) {
	host := hostWith(t, "hub.jpg", "spoke-a.jpg", "spoke-b.jpg")
	host.index = 0
	g := newOverview(t, host)
	const hub uint64 = 0xFFFF000000000000
	injectHashes(t, g, host, []uint64{hub, hub ^ 0x3FF, hub ^ (0x3FF << 10)})
	g.SetDuplicateDistance(10)

	g.SetBrowsingDuplicates(true)

	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (hub + one spoke, not both)", g.count())
	}
	seen := map[int]bool{g.fileIndex(0): true, g.fileIndex(1): true}
	if seen[2] {
		t.Fatal("Shift+D on the hub must not list spoke B")
	}
}

// TestSetHideDuplicates_UnrelatedLineArtStaysVisible is the end-to-end lock
// for the reported bug, and the only duplicate test that goes through the
// real path - decode, thumbnail, dHash - instead of injectHashes, because
// the defect was in the hash rather than in the grouping.
//
// Three unrelated sketches on white. Reducing them to the dHash grid by
// sampling a few pixels per cell hit the white background nearly every
// time, so all three hashed to two or three bits and every one of them
// matched every other. Hiding extras then left a single cell, and Shift+D
// on the first file listed the other two as its duplicates.
func TestSetHideDuplicates_UnrelatedLineArtStaysVisible(t *testing.T) {
	host := &fakeHost{files: []fyne.URI{
		uitest.LineArtJPEGURI(t, "sketch-a.jpg", 1),
		uitest.LineArtJPEGURI(t, "sketch-b.jpg", 2),
		uitest.LineArtJPEGURI(t, "sketch-c.jpg", 3),
	}}
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	g.Toggle()

	g.SetHideDuplicates(true)
	g.Settle()

	if g.count() != 3 {
		t.Fatalf("count() = %d, want 3: unrelated sketches must not hide each other", g.count())
	}
	for i := range 3 {
		if g.groupSize(i) != 1 {
			t.Errorf("groupSize(%d) = %d, want 1 (hashed and unique)", i, g.groupSize(i))
		}
	}
}

func TestSetHideDuplicates_ZeroHashFirstFileIsUnique(t *testing.T) {
	host := hostWith(t, "flat.jpg", "sparse-a.jpg", "sparse-b.jpg")
	g := newOverview(t, host)
	injectHashes(t, g, host, []uint64{0, 1, 2})
	g.SetDuplicateDistance(imaging.DuplicateMaxDistance)

	g.SetHideDuplicates(true)

	if g.groupSize(0) != 1 {
		t.Fatalf("groupSize(0) = %d, want 1 (hash 0 must not absorb sparse hashes)", g.groupSize(0))
	}
	if g.RepresentativeOf(1) == 0 || g.RepresentativeOf(2) == 0 {
		t.Fatal("sparse hashes must not pick the hash-0 first file as representative")
	}
}

func garbageURI(t *testing.T, name string) fyne.URI {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	return storage.NewFileURI(p)
}

func TestSetBrowsingDuplicates_FailedDecodeDoesNotRetoast(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg"},
		[]int{1, 1},
	)
	host.files = append(host.files, garbageURI(t, "corrupt.dat"))
	g := newOverview(t, host)
	host.index = 0

	g.SetBrowsingDuplicates(true)
	if len(host.toasts) != 1 {
		t.Fatalf("toasts = %v, want one analyzing toast on first browse", host.toasts)
	}
	g.Settle()

	g.SetBrowsingDuplicates(false)
	host.toasts = nil
	g.SetBrowsingDuplicates(true)
	g.Settle()

	if len(host.toasts) != 0 {
		t.Fatalf("toasts = %v, want none (failed files must not be re-hashed)", host.toasts)
	}
}
