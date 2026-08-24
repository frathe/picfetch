package grid

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"

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
