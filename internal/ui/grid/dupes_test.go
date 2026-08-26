package grid

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

// serialUIQueue is fyne.Do on an idle UI goroutine: the callback runs
// before Do returns to the worker, serialized with other callbacks, and
// Drain is a no-op because that work already happened. uitest.UIQueue
// cannot catch hideApply re-arming — it never runs f, so Store(false)
// never happens and every later job sees the flag still set.
type serialUIQueue struct {
	mu sync.Mutex
	n  atomic.Int32
}

func (q *serialUIQueue) Do(f func()) {
	q.n.Add(1)
	q.mu.Lock()
	defer q.mu.Unlock()
	f()
}

func (q *serialUIQueue) Drain() bool { return false }

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
	cell := newGridCell()
	_, _, _, badge := unpackGridCell(cell)

	g.applyDupBadge(badge, 0, fyne.NewSize(cellSize, cellSize))
	if badge.chip.Visible() {
		t.Fatal("badge should stay hidden while hide-duplicates is off")
	}

	g.SetHideDuplicates(true)
	g.applyDupBadge(badge, 0, fyne.NewSize(cellSize, cellSize))
	if !badge.chip.Visible() || badge.label.Text != "2" {
		t.Errorf("representative badge visible=%v text=%q, want visible text \"2\"", badge.chip.Visible(), badge.label.Text)
	}

	g.applyDupBadge(badge, 2, fyne.NewSize(cellSize, cellSize))
	if badge.chip.Visible() {
		t.Error("a unique cell must hide the badge")
	}

	g.SetHideDuplicates(false)
	g.applyDupBadge(badge, 0, fyne.NewSize(cellSize, cellSize))
	if badge.chip.Visible() {
		t.Error("turning hide off must hide the badge")
	}
}

func TestDupBadge_TopRightClearsTheHighlightRing(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)

	cell := newGridCell()
	_, _, ring, badge := unpackGridCell(cell)
	if cell.Objects[2] != ring {
		t.Fatal("highlight ring must sit under the badge layer")
	}
	if cell.Objects[3] == ring {
		t.Fatal("badge layer must stack above the highlight ring")
	}

	cell.Resize(fyne.NewSize(cellSize, cellSize))
	g.applyDupBadge(badge, 0, cell.Size())

	r, _, _, a := badge.bg.FillColor.RGBA()
	if r != 0 || a != 0xffff {
		t.Errorf("backdrop RGBA = %d,_,_,%d, want opaque black", r, a)
	}
	if !badge.chip.Visible() {
		t.Fatal("representative badge should be visible")
	}

	pos := badge.chip.Position()
	sz := badge.chip.Size()
	if pos.Y < dupBadgeMargin-0.5 {
		t.Errorf("badge top = %v, want ≥ %v (clear of the highlight ring)", pos.Y, dupBadgeMargin)
	}
	rightGap := cellSize - (pos.X + sz.Width)
	if rightGap < dupBadgeMargin-0.5 {
		t.Errorf("badge right gap = %v, want ≥ %v (clear of the highlight ring)", rightGap, dupBadgeMargin)
	}
	if pos.X+sz.Width/2 < cellSize/2 {
		t.Errorf("badge centre X = %v, want the right half of the cell", pos.X+sz.Width/2)
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

func TestSetHideDuplicates_PendingShowsChromeAndLeavesUnhashedVisible(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.rememberHash(host.files[0], mustThumb(t, host.files[0]))
	g.rememberNative(host.files[0], image.Rect(0, 0, 64, 48)) // PatternedJPEGURI native

	unpark := parkDecodes(t, g)
	g.Toggle()
	g.SetHideDuplicates(true)

	if !g.HideDuplicates() {
		t.Fatal("HideDuplicates() = false after SetHideDuplicates(true)")
	}
	if got, want := g.searchLabel.Text, lang.L("Hiding duplicates"); got != want {
		t.Errorf("searchLabel = %q, want %q while hashes are still pending", got, want)
	}
	if g.count() != 3 {
		t.Fatalf("count() = %d while the extra is still unhashed, want 3", g.count())
	}
	if got := g.hashJobs.Load(); got != 2 {
		t.Fatalf("hashJobs = %d, want 2 (file 0 already hashed; 1 and 2 still pending)", got)
	}

	unpark()
	g.Settle()

	if g.count() != 2 {
		t.Fatalf("count() = %d after remaining hashes land, want 2", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Error("the pair's extra should hide once its hash lands")
	}
}

func queuedCompletions(t *testing.T, g *Overview) int {
	t.Helper()
	q, ok := g.ui.(*uitest.UIQueue)
	if !ok {
		t.Fatalf("ui type = %T, want *uitest.UIQueue", g.ui)
	}
	return q.Len()
}

// TestHashRemaining_CoalescesHideAppliesOntoTheUIQueue pins the lock-up:
// each hash job used to fyne.Do a full rebuildFilter (DuplicateGroups plus
// wrap.Refresh), so D on a large cold folder saturated the UI goroutine
// until every remaining thumbnail had hashed. Completions must share one
// in-flight apply; the last job always queues so the final state cannot
// be skipped. Grid stays closed so thumbnail paints do not inflate Len.
func TestHashRemaining_CoalescesHideAppliesOntoTheUIQueue(t *testing.T) {
	host := hostPatterned(t,
		[]string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg"},
		[]int{1, 2, 3, 4, 5},
	)
	g := newOverview(t, host)
	unpark := parkDecodes(t, g)
	g.SetHideDuplicates(true)
	if got := g.hashJobs.Load(); got != 5 {
		t.Fatalf("hashJobs = %d, want 5", got)
	}

	unpark()
	g.decodes.Wait()

	if got := queuedCompletions(t, g); got > 2 {
		t.Fatalf("queued UI completions = %d, want at most 2 (coalesced apply + last job)", got)
	}

	g.Settle()
}

// TestHashRemaining_CachedThumbsJoinThePool pins D on a warm cache: those
// thumbnails used to DifferenceHash on the key-handler goroutine, so a
// folder that already fit in the 256MB thumb cache froze the UI until
// hiding was done, with no pool jobs to return to.
func TestHashRemaining_CachedThumbsJoinThePool(t *testing.T) {
	host := hostPatterned(t,
		[]string{"a.jpg", "b.jpg", "c.jpg"},
		[]int{1, 2, 3},
	)
	g := newOverview(t, host)
	for _, u := range host.files {
		if !g.StoreThumb(u, mustThumb(t, u)) {
			t.Fatalf("StoreThumb(%s) refused", u.Name())
		}
	}

	unpark := parkDecodes(t, g)
	g.SetHideDuplicates(true)

	if got := g.hashJobs.Load(); got != 3 {
		t.Fatalf("hashJobs = %d, want 3 (cache hits must join the pool, not dHash on D)", got)
	}
	for i, u := range host.files {
		if _, ok := g.hashOf(u); ok {
			t.Errorf("file %d (%s) hashed on the caller", i, u.Name())
		}
	}

	unpark()
	g.Settle()
}

// TestHashRemaining_HideApplyStaysArmedUntilCallbackReturns pins the
// production lock coalescing missed: fyne.Do is async but an idle UI
// runs the callback before the next hash lands, and the callback used to
// Store(false) before DuplicateGroups. Later jobs then each queued
// another rebuild, and the UI goroutine drained that queue before input.
func TestHashRemaining_HideApplyStaysArmedUntilCallbackReturns(t *testing.T) {
	host := hostPatterned(t,
		[]string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg"},
		[]int{1, 2, 3, 4, 5},
	)
	g := newOverview(t, host)
	q := &serialUIQueue{}
	g.SetUIQueue(q)
	unpark := parkDecodes(t, g)
	g.SetHideDuplicates(true)

	unpark()
	g.decodes.Wait()

	if got := q.n.Load(); got > 2 {
		t.Fatalf("UI applies = %d, want at most 2 (in-flight apply must stay armed until it returns)", got)
	}
}

// TestHashRemaining_ComputesGroupsBeforeTheUIQueue pins the remaining
// hitch: even one apply per coalesce window ran DuplicateGroups on the
// UI goroutine, so a 13k-file folder spent the whole hashing window
// inside O(n²) complete linkage and never saw input. Workers compute;
// Drain only installs.
func TestHashRemaining_ComputesGroupsBeforeTheUIQueue(t *testing.T) {
	host := hostPatterned(t,
		[]string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg"},
		[]int{1, 2, 3, 4, 5},
	)
	g := newOverview(t, host)
	unpark := parkDecodes(t, g)
	g.SetHideDuplicates(true)
	if got := g.groupComputes.Load(); got != 1 {
		t.Fatalf("groupComputes after SetHideDuplicates = %d, want 1 (chrome apply)", got)
	}

	unpark()
	g.decodes.Wait()

	got := g.groupComputes.Load()
	if got < 2 {
		t.Fatalf("groupComputes after Wait = %d, want ≥2 (workers compute before g.ui.Do)", got)
	}

	g.Settle()
	if still := g.groupComputes.Load(); still != got {
		t.Fatalf("groupComputes after Drain = %d, want %d (UI must not DuplicateGroups again)", still, got)
	}
}

func TestSetHideDuplicates_OnePendingJobHidesExtraWithoutWaitingForAPeer(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.rememberHash(host.files[0], mustThumb(t, host.files[0]))
	g.rememberHash(host.files[2], mustThumb(t, host.files[2]))
	g.rememberNative(host.files[0], image.Rect(0, 0, 64, 48)) // PatternedJPEGURI native
	g.rememberNative(host.files[2], image.Rect(0, 0, 64, 48)) // PatternedJPEGURI native

	unpark := parkDecodes(t, g)
	g.Toggle()
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("count() = %d with the extra still unhashed, want 3", g.count())
	}
	if got := g.hashJobs.Load(); got != 1 {
		t.Fatalf("hashJobs = %d, want 1", got)
	}

	unpark()
	g.Settle()

	if g.count() != 2 {
		t.Fatalf("count() = %d after the one remaining job, want 2", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Error("the extra should hide when its own job completes")
	}
}

func mustThumb(t *testing.T, u fyne.URI) image.Image {
	t.Helper()
	thumb, err := imaging.LoadThumbnail(u)
	if err != nil {
		t.Fatalf("LoadThumbnail(%s): %v", u.Name(), err)
	}
	return thumb
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
	g.rebuildGroups()
	got := g.groupMembers(1)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("groupMembers(1) = %v, want [0 1]", got)
	}
	if g.groupMembers(2) != nil {
		t.Fatalf("groupMembers(unique) = %v, want nil", g.groupMembers(2))
	}
}

func TestInspectMembers_ReadsSnapshotWithoutRebuild(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.rebuildGroups()
	g.BeginInspect(1)
	got := g.InspectMembers()
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("InspectMembers() = %v, want [0 1]", got)
	}
	before := g.groupComputes.Load()
	_ = g.InspectMembers()
	if n := g.groupComputes.Load(); n != before {
		t.Fatalf("InspectMembers incremented groupComputes from %d to %d", before, n)
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
	// Parked before opening: an unwarmed Toggle spawns a decode per visible
	// cell, and each one that finishes remembers a hash. Whether they beat
	// SetBrowsingDuplicates below is what decides whether there is anything
	// left for it to hash - that is, whether the toast appears at all.
	// Parked, there provably is.
	unpark := parkDecodes(t, g)
	g.Toggle()

	g.SetBrowsingDuplicates(true)
	if len(host.toasts) != 1 || host.toasts[0] != lang.L("The images are currently being analyzed") {
		t.Fatalf("toasts = %v, want [%q] while hashing", host.toasts, lang.L("The images are currently being analyzed"))
	}

	unpark()
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
	unpark := parkDecodes(t, g)
	g.Toggle()

	g.SetBrowsingDuplicates(true)
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("count() = %d while hashes pending, want 3 (not collapsed to the source cell)", g.count())
	}
	if !g.BrowsingDuplicates() {
		t.Fatal("BrowsingDuplicates() = false while hashes pending, want true")
	}

	unpark()
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
	// Parked before opening, and left parked: hostWith's JPEGs are solid
	// white, so a decode that landed would rememberHash 0 over the
	// near-gray pair injected above, leaving the two files exact duplicates
	// at every distance - the split this test is named for could not
	// happen. parkDecodes unparks and Settles on cleanup.
	parkDecodes(t, g)
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
	if g.pixels == nil {
		g.pixels = make(map[string]int)
	}
	g.hashGen = host.Generation()
	for i, h := range hs {
		g.hashes[host.files[i].String()] = h
		g.pixels[host.files[i].String()] = 1
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

func TestRebuildFilter_KeepsHighlightedHostWhenAnExtraDisappears(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.rememberHash(host.files[0], mustThumb(t, host.files[0]))
	g.rememberHash(host.files[2], mustThumb(t, host.files[2]))

	unpark := parkDecodes(t, g)
	g.Toggle()
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("setup count() = %d, want 3 (extra still unhashed)", g.count())
	}
	g.setHighlight(2)
	if g.fileIndex(g.Highlight()) != 2 {
		t.Fatalf("setup highlight host = %d, want 2", g.fileIndex(g.Highlight()))
	}

	g.rememberHash(host.files[1], mustThumb(t, host.files[1]))
	g.rebuildFilter(false)

	if g.count() != 2 {
		t.Fatalf("count() = %d after extra hashes, want 2", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Fatal("index 1 should now be a hidden extra")
	}
	if g.fileIndex(g.Highlight()) != 2 {
		t.Fatalf("Highlight host = %d, want 2 (moon.jpg stayed under the ring; display index may have moved)", g.fileIndex(g.Highlight()))
	}
	if g.Highlight() != 1 {
		t.Fatalf("Highlight() = %d, want 1 (host 2 is now display index 1)", g.Highlight())
	}

	unpark()
	g.Settle()
}

func TestSourceDuplicateGroupSize_UnknownUntilGroupsBuilt(t *testing.T) {
	g, host := pairAndUnique(t)
	host.index = 0
	if got := g.SourceDuplicateGroupSize(); got != 0 {
		t.Fatalf("before hide/browse rebuild: size = %d, want 0 (unknown)", got)
	}

	g.SetHideDuplicates(true)
	if got := g.SourceDuplicateGroupSize(); got != 2 {
		t.Fatalf("pair representative size = %d, want 2", got)
	}

	host.index = 2
	// grid is visible: source is the highlight, not host.index.
	// Move the ring to the unique cell (display index of host 2).
	g.setHighlight(g.displayIndexOf(2))
	if got := g.SourceDuplicateGroupSize(); got != 1 {
		t.Fatalf("unique cell size = %d, want 1", got)
	}
}

func TestSourceDuplicateGroupSize_ClosedGridUsesCurrentIndex(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.Close()
	host.index = 2
	if g.Visible() {
		t.Fatal("premises: grid closed")
	}
	if got := g.SourceDuplicateGroupSize(); got != 1 {
		t.Fatalf("closed grid unique current = %d, want 1", got)
	}
}

func TestSetOnDupeStateChanged_HideAndBrowse(t *testing.T) {
	g, _ := pairAndUnique(t)
	var n int
	g.SetOnDupeStateChanged(func() { n++ })

	g.SetHideDuplicates(true)
	if n != 1 {
		t.Fatalf("after hide on: n=%d, want 1", n)
	}
	g.SetHideDuplicates(true)
	if n != 1 {
		t.Fatalf("idempotent hide fired: n=%d", n)
	}
	g.SetHideDuplicates(false)
	if n != 2 {
		t.Fatalf("after hide off: n=%d, want 2", n)
	}

	n = 0
	g.SetOnDupeStateChanged(func() { n++ }) // still read at fire time
	g.SetBrowsingDuplicates(true)
	if !g.BrowsingDuplicates() {
		t.Fatal("premises: pair should browse")
	}
	if n < 1 {
		t.Fatalf("browse on did not fire: n=%d", n)
	}

	was := n
	g.SetBrowsingDuplicates(false)
	if g.BrowsingDuplicates() {
		t.Fatal("browse should be off")
	}
	if n <= was {
		t.Fatalf("browse off did not fire: n=%d was=%d", n, was)
	}
}

func TestSetOnDupeStateChanged_SetAfterHideStillFiresOnNextChange(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	var n int
	g.SetOnDupeStateChanged(func() { n++ })
	g.SetHideDuplicates(false)
	if n != 1 {
		t.Fatalf("hook registered after hide-on must still run on hide-off: n=%d", n)
	}
}

func TestSetOnDupeStateChanged_NilIsNoop(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetOnDupeStateChanged(nil)
	g.SetHideDuplicates(true) // must not panic
}

func TestSetOnDupeStateChanged_SetDuplicateDistanceWhileHide(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	var n int
	g.SetOnDupeStateChanged(func() { n++ })

	g.SetDuplicateDistance(0)
	if n != 1 {
		t.Fatalf("distance change while hide on: n=%d, want 1", n)
	}
	g.SetDuplicateDistance(0)
	if n != 1 {
		t.Fatalf("idempotent distance fired: n=%d", n)
	}
}

func TestSetOnDupeStateChanged_SetDuplicateDistanceWhileIdleRebuilds(t *testing.T) {
	g, host := pairAndUnique(t)
	host.index = 0
	if got := g.SourceDuplicateGroupSize(); got != 0 {
		t.Fatalf("setup size = %d, want 0 (groups not built)", got)
	}
	var n int
	g.SetOnDupeStateChanged(func() { n++ })

	g.SetDuplicateDistance(0)
	if n != 1 {
		t.Fatalf("idle distance change: n=%d, want 1", n)
	}
	if got := g.SourceDuplicateGroupSize(); got != 2 {
		t.Fatalf("idle rebuild pair size = %d, want 2", got)
	}
}

func TestSetOnDupeStateChanged_LastHashJobFiresOnce(t *testing.T) {
	host := hostPatterned(t,
		[]string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg"},
		[]int{1, 2, 3, 4, 5},
	)
	g := newOverview(t, host)
	unpark := parkDecodes(t, g)
	var n int
	g.SetOnDupeStateChanged(func() { n++ })

	g.SetHideDuplicates(true)
	if n != 1 {
		t.Fatalf("hide on while pending: n=%d, want 1", n)
	}

	unpark()
	g.Settle()
	if n != 2 {
		t.Fatalf("after last hash job: n=%d, want 2 (mid-interval applies must not fire)", n)
	}
}

func TestSetOnDupeStateChanged_PendingBrowseFiresOnceThenFinish(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	unpark := parkDecodes(t, g)
	g.Toggle()
	var n int
	g.SetOnDupeStateChanged(func() { n++ })

	g.SetBrowsingDuplicates(true)
	if n != 1 {
		t.Fatalf("pending browse: n=%d, want 1", n)
	}

	unpark()
	g.Settle()
	if n < 2 {
		t.Fatalf("last-job finishBrowse did not fire: n=%d", n)
	}
}

func TestWarm_RecordsNativePixelCountNotThumbnailSize(t *testing.T) {
	// Larger than ThumbnailSize (200) so thumb.Bounds() cannot equal native.
	u := uitest.TempJPEGURI(t, "big.jpg", 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255})
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	px, ok := g.pixelCountOf(u)
	if !ok {
		t.Fatal("Warm should record native pixels")
	}
	if px != 800*400 {
		t.Errorf("pixels = %d, want %d (not the thumbnail)", px, 800*400)
	}
	if thumb, ok := g.thumbs.Get(u.String()); ok {
		tb := thumb.Bounds()
		if tb.Dx()*tb.Dy() == px {
			t.Fatal("native pixel count must not equal thumbnail bounds")
		}
	}
}

func TestWipeHashesIfStale_DropsPixelsOnGenerationChange(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	host.gen++
	g.wipeHashesIfStale()
	if _, ok := g.pixelCountOf(host.files[0]); ok {
		t.Fatal("pixels must drop when host.Generation changes")
	}
}

func TestHashRemaining_BackfillsPixelsForAlreadyHashedFiles(t *testing.T) {
	u := uitest.TempJPEGURI(t, "big.jpg", 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255})
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	thumb := mustThumb(t, u)
	g.thumbs.Add(u.String(), thumb)
	g.rememberHash(u, thumb)
	if _, ok := g.pixelCountOf(u); ok {
		t.Fatal("setup: pixels should be missing")
	}

	g.SetHideDuplicates(true)
	g.Settle()

	px, ok := g.pixelCountOf(u)
	if !ok {
		t.Fatal("hashRemaining should backfill pixels for a hashed file")
	}
	if px != 800*400 {
		t.Errorf("pixels = %d, want %d (not the thumbnail)", px, 800*400)
	}
	if tb := thumb.Bounds(); tb.Dx()*tb.Dy() == px {
		t.Fatal("backfill must not use thumbnail bounds as native size")
	}
}

func TestHashRemaining_DoesNotRequeueWhenHashAndPixelsExist(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	if n := g.hashRemaining(); n != 0 {
		t.Fatalf("hashRemaining() = %d, want 0 after Warm", n)
	}
}

func TestHashRemaining_FailedProbeRecordsZeroAndDoesNotRequeue(t *testing.T) {
	u := uitest.TempJPEGURI(t, "gone.jpg", 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255})
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	thumb := mustThumb(t, u)
	g.thumbs.Add(u.String(), thumb)
	g.rememberHash(u, thumb)
	if err := os.Remove(u.Path()); err != nil {
		t.Fatal(err)
	}

	g.SetHideDuplicates(true)
	g.Settle()

	px, ok := g.pixelCountOf(u)
	if !ok {
		t.Fatal("failed probe must record a size so hide does not requeue")
	}
	if px != 0 {
		t.Errorf("pixels = %d, want 0 after failed probe", px)
	}
	if n := g.hashRemaining(); n != 0 {
		t.Fatalf("hashRemaining() = %d after failed probe, want 0", n)
	}
}

func TestComputeDuplicateGroups_PicksHighestPixelCount(t *testing.T) {
	host := hostWith(t, "small.jpg", "large.jpg", "unique.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
		host.files[2].String(): 0x2222222222222222,
	}
	g.pixels = map[string]int{
		host.files[0].String(): 100,
		host.files[1].String(): 400,
		host.files[2].String(): 9999,
	}
	g.hashGen = host.gen
	g.rebuildGroups()

	if g.RepresentativeOf(0) != 1 || g.RepresentativeOf(1) != 1 {
		t.Errorf("rep of pair = %d/%d, want 1 (larger file)", g.RepresentativeOf(0), g.RepresentativeOf(1))
	}
	if g.RepresentativeOf(2) != 2 {
		t.Errorf("unique rep = %d, want 2", g.RepresentativeOf(2))
	}

	g.SetHideDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2", g.count())
	}
	if g.fileIndex(0) != 1 || g.fileIndex(1) != 2 {
		t.Fatalf("visible = [%d, %d], want [1, 2] (large + unique)", g.fileIndex(0), g.fileIndex(1))
	}
	if !g.IsHiddenExtra(0) || g.IsHiddenExtra(1) || g.IsHiddenExtra(2) {
		t.Error("only the smaller pair member is a hidden extra")
	}
}

func TestComputeDuplicateGroups_EqualPixelsKeepsLowestIndex(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
	}
	g.pixels = map[string]int{
		host.files[0].String(): 100,
		host.files[1].String(): 100,
	}
	g.hashGen = host.gen
	g.rebuildGroups()
	if g.RepresentativeOf(1) != 0 {
		t.Errorf("RepresentativeOf(1) = %d, want 0 (tie-break)", g.RepresentativeOf(1))
	}
}

func TestComputeDuplicateGroups_UnknownPixelsLoseToKnown(t *testing.T) {
	host := hostWith(t, "unknown.jpg", "known.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
	}
	g.pixels = map[string]int{
		host.files[1].String(): 50,
	}
	g.hashGen = host.gen
	g.rebuildGroups()
	if g.RepresentativeOf(0) != 1 {
		t.Errorf("RepresentativeOf(0) = %d, want 1 (known size wins)", g.RepresentativeOf(0))
	}
}

func TestSetHideDuplicates_JumpsToHighestResolution(t *testing.T) {
	host := hostWith(t, "small.jpg", "large.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
	}
	g.pixels = map[string]int{
		host.files[0].String(): 100,
		host.files[1].String(): 400,
	}
	g.hashGen = host.gen
	host.index = 0

	g.SetHideDuplicates(true)

	if len(host.shown) == 0 || host.shown[len(host.shown)-1] != 1 {
		t.Errorf("ShowImage calls = %v, want a jump to representative 1", host.shown)
	}
}

func TestBeginInspect_ReportsMembersAndSkipsJump(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.index = 1 // extra; equal-size pair, representative is 0

	if g.InspectingDuplicates() {
		t.Fatal("inspect should start off")
	}
	g.BeginInspect(1)
	if !g.InspectingDuplicates() {
		t.Fatal("BeginInspect(1) should turn inspect on")
	}
	got := g.InspectMembers()
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("InspectMembers() = %v, want [0 1]", got)
	}

	host.shown = nil
	g.jumpIfHiddenExtra()
	if len(host.shown) != 0 {
		t.Errorf("ShowImage calls = %v, want none while inspecting", host.shown)
	}
}

func TestJumpIfHiddenExtra_StillJumpsWhenNotInspecting(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.index = 1
	host.shown = nil

	g.jumpIfHiddenExtra()
	if len(host.shown) != 1 || host.shown[0] != 0 {
		t.Errorf("ShowImage calls = %v, want jump to representative 0", host.shown)
	}
}

func TestClose_ClearsInspectEvenWhenAlreadyHidden(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.BeginInspect(1)
	if g.Visible() {
		g.Close()
	}
	if g.Visible() {
		t.Fatal("premises: grid closed")
	}
	g.BeginInspect(1)
	g.Close()
	if g.InspectingDuplicates() {
		t.Fatal("Close must clear inspect while the overlay is already hidden")
	}
}

func TestToggleOpen_ClearsInspect(t *testing.T) {
	g, _ := pairAndUnique(t)
	if g.Visible() {
		g.Close()
	}
	g.BeginInspect(1)
	g.Toggle()
	if !g.Visible() {
		t.Fatal("Toggle should open")
	}
	if g.InspectingDuplicates() {
		t.Fatal("opening the grid with G must end inspect")
	}
}

func TestClearInspect_StopsSkippingJump(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.index = 1
	g.BeginInspect(1)
	g.ClearInspect()
	host.shown = nil
	g.jumpIfHiddenExtra()
	if len(host.shown) != 1 || host.shown[0] != 0 {
		t.Errorf("after ClearInspect, ShowImage calls = %v, want [0]", host.shown)
	}
}

func TestHandleKey_ReturnFromBrowseCommitsExtra(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	if g.displayIndexOf(1) < 0 {
		t.Fatal("extra should be visible while browsing")
	}
	g.setHighlight(g.displayIndexOf(1))
	host.shown = nil
	host.index = 1 // what the viewer would set after ShowImage; jump uses CurrentIndex

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if g.Visible() {
		t.Fatal("Return should close the grid")
	}
	if g.BrowsingDuplicates() {
		t.Fatal("Return should end browse")
	}
	if !g.InspectingDuplicates() {
		t.Fatal("Return from browse should begin inspect")
	}
	if len(host.shown) != 1 || host.shown[0] != 1 {
		t.Fatalf("ShowImage calls = %v, want [1] (the extra, not representative 0)", host.shown)
	}

	g.jumpIfHiddenExtra()
	if len(host.shown) != 1 || host.shown[0] != 1 {
		t.Fatalf("after jumpIfHiddenExtra ShowImage = %v, want still [1]", host.shown)
	}
}

func TestOnSelected_ClickFromBrowseCommitsExtra(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	host.shown = nil
	host.index = 1
	click(g, host, g.displayIndexOf(1), 0)

	if !g.InspectingDuplicates() || g.Visible() || g.BrowsingDuplicates() {
		t.Fatalf("inspect=%v visible=%v browse=%v", g.InspectingDuplicates(), g.Visible(), g.BrowsingDuplicates())
	}
	if len(host.shown) != 1 || host.shown[0] != 1 {
		t.Fatalf("ShowImage = %v, want [1]", host.shown)
	}
}

func TestHandleKey_ReturnFromHideGridDoesNotInspect(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	if g.BrowsingDuplicates() {
		t.Fatal("premises: not browsing")
	}
	g.setHighlight(0)
	host.shown = nil
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if g.InspectingDuplicates() {
		t.Fatal("Return from hide-duplicates (not browse) must not begin inspect")
	}
}

func TestHandleKey_DNoopWhileBrowsing(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.HideDuplicates() || !g.BrowsingDuplicates() {
		t.Fatal("plain D while browsing must not toggle hide or leave browse")
	}
}

func TestFilesChanged_ClearsInspectWhenGroupDissolves(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.BeginInspect(1)
	host.files = host.files[2:] // only moon.jpg left
	host.index = 0
	host.gen++
	g.FilesChanged()
	if g.InspectingDuplicates() {
		t.Fatal("inspect must end when the group no longer has two members")
	}
}

func TestFilesChanged_RetargetsInspectWhenCurrentStillGrouped(t *testing.T) {
	g, host := openPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon-a.jpg", "moon-b.jpg"},
		[]int{1, 1, 99, 99},
	)
	g.SetHideDuplicates(true)
	g.BeginInspect(1)
	host.files = host.files[2:]
	host.index = 0
	host.gen++
	g.FilesChanged()
	if !g.InspectingDuplicates() {
		t.Fatal("inspect should retarget onto the remaining group")
	}
	members := g.InspectMembers()
	if len(members) < 2 {
		t.Fatalf("InspectMembers = %v, want a group of at least 2", members)
	}
}

func TestFilesChanged_HideDuplicatesGroupingSurvivesDelete(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.files = host.files[:2]
	host.gen++
	g.FilesChanged()
	if !g.IsHiddenExtra(1) {
		t.Fatal("the extra must stay hidden after deleting a unique")
	}
	if g.count() != 1 {
		t.Fatalf("count() = %d, want 1", g.count())
	}
}
