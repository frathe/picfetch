package grid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/frathe/picfetch/internal/decodepool"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

// --- thumbnails ------------------------------------------------------------

// newCell returns a cell of the shape the grid's own CreateItem builds -
// the image plus its highlight ring - to hand to requestThumbnail
// directly.
func newCell() (*fyne.Container, *canvas.Image) {
	img := canvas.NewImageFromImage(nil)
	ring := canvas.NewRectangle(color.Transparent)

	return container.NewStack(img, ring), img
}

func TestRequestThumbnail_CacheHitAppliesSynchronously(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)

	g.requestThumbnail(cell, img, 0, host.gen)

	if img.Image == nil {
		t.Error("a cache hit should paint the cell synchronously, without waiting for a goroutine")
	}
}

func TestRequestThumbnail_DecodesInBackgroundAndCaches(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)

	g.requestThumbnail(cell, img, 0, host.gen)

	// Settle, not a poll of the cache: the cache write and the paint
	// happen in the same completion callback, so waiting on the decode
	// itself is what gives this goroutine a happens-before edge on both.
	g.Settle()

	if !g.Cached(host.files[0]) {
		t.Error("the decoded thumbnail should have been cached")
	}
	if img.Image == nil {
		t.Error("img.Image should be set once the background decode finishes")
	}
}

// TestSetCacheBytes_RetunesTheThumbnailBudget covers the one setter this
// package exposes - the settings window's route to the thumbnail cache, via
// internal/ui's SetMaxThumbCacheMB. An 8x8 JPEG stays 8x8 through scaleToFit
// (already inside ThumbnailSize) and decodes to a 4:2:0 *image.YCbCr, so each
// one weighs well under 200 bytes; a 100-byte budget therefore fits exactly
// one of them and Warm's second file has to evict its first.
func TestSetCacheBytes_RetunesTheThumbnailBudget(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)

	g.SetCacheBytes(100)

	if err := g.Warm(); err != nil {
		t.Fatalf("Warm() returned error: %v", err)
	}

	if g.thumbs.Len() != 1 {
		t.Errorf("cached thumbnails = %d, want 1 under a 100-byte budget", g.thumbs.Len())
	}
	if g.Cached(host.files[0]) {
		t.Error("the first thumbnail should have been evicted by the second")
	}
	if !g.Cached(host.files[1]) {
		t.Error("the most recently warmed thumbnail should still be cached")
	}

	// Raising the budget doesn't resurrect anything, but it does stop the
	// eviction: warming again now holds both.
	g.SetCacheBytes(imaging.DefaultThumbCacheBytes)

	if err := g.Warm(); err != nil {
		t.Fatalf("Warm() returned error: %v", err)
	}

	if g.thumbs.Len() != 2 {
		t.Errorf("cached thumbnails = %d after raising the budget, want 2", g.thumbs.Len())
	}
}

// --- CachedThumb / StoreThumb / ThumbCacheFull -----------------------------

func TestCachedThumb_MissesForUnstoredURI(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)

	if _, ok := g.CachedThumb(host.files[0]); ok {
		t.Error("CachedThumb should miss for a URI that was never stored")
	}
}

func TestStoreThumb_ThenCachedThumb_ReturnsWhatWasStored(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)

	thumb := image.NewRGBA(image.Rect(0, 0, 4, 4))

	if ok := g.StoreThumb(host.files[0], thumb); !ok {
		t.Fatal("StoreThumb should report true for a thumbnail well within budget")
	}

	got, ok := g.CachedThumb(host.files[0])
	if !ok {
		t.Fatal("CachedThumb should hit after StoreThumb")
	}
	if got != image.Image(thumb) {
		t.Error("CachedThumb should return the same image that was stored")
	}
}

// TestStoreThumb_TooBigForBudgetIsRefused covers AddIfFits's
// never-evict-for-a-speculative-write rule (see its own comment in
// internal/imaging/bytecache.go): an entry that alone outweighs the whole
// budget is refused outright rather than stored and left to evict
// everything else in the cache.
func TestStoreThumb_TooBigForBudgetIsRefused(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	g.SetCacheBytes(100)

	// A 100x100 RGBA thumbnail weighs 100*100*4 = 40000 bytes, comfortably
	// over the 100-byte budget.
	big := image.NewRGBA(image.Rect(0, 0, 100, 100))

	if ok := g.StoreThumb(host.files[0], big); ok {
		t.Error("StoreThumb should refuse a thumbnail that alone exceeds the whole budget")
	}
	if _, ok := g.CachedThumb(host.files[0]); ok {
		t.Error("CachedThumb should still miss after a refused StoreThumb")
	}
}

func TestThumbCacheFull_FalseUntilBudgetReached(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	g.SetCacheBytes(100)

	if g.ThumbCacheFull() {
		t.Error("a fresh overview's thumbnail cache should not report full")
	}

	// A 5x5 RGBA thumbnail weighs 5*5*4 = 100 bytes: exactly the budget.
	full := image.NewRGBA(image.Rect(0, 0, 5, 5))
	if ok := g.StoreThumb(host.files[0], full); !ok {
		t.Fatal("StoreThumb should accept a thumbnail exactly at budget")
	}

	if !g.ThumbCacheFull() {
		t.Error("the cache should report full once stored bytes reach the budget")
	}
}

// TestStoreThumb_AloneDoesNotProtectTheHeadOfTheList is the eviction-churn
// behavior ThumbCacheFull's doc comment warns about: a pre-warm pass that
// only ever calls StoreThumb, with no ThumbCacheFull check between offers,
// will happily evict the entries it stored first to make room for the
// ones it stores last. A caller pre-warming a favorite's disk previews in
// file-list order needs the *first* files warm when the grid opens at
// index 0, not the last ones - which is exactly what this test shows
// StoreThumb alone does not guarantee.
func TestStoreThumb_AloneDoesNotProtectTheHeadOfTheList(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)

	// A 5x5 RGBA thumbnail weighs 5*5*4 = 100 bytes, so a 100-byte budget
	// fits exactly one.
	g.SetCacheBytes(100)

	first := image.NewRGBA(image.Rect(0, 0, 5, 5))
	second := image.NewRGBA(image.Rect(0, 0, 5, 5))

	if ok := g.StoreThumb(host.files[0], first); !ok {
		t.Fatal("StoreThumb should accept the first thumbnail, which alone fits the budget")
	}
	if ok := g.StoreThumb(host.files[1], second); !ok {
		t.Fatal("StoreThumb should accept the second thumbnail, which alone fits the budget")
	}

	if _, ok := g.CachedThumb(host.files[0]); ok {
		t.Error("the first thumbnail should have been evicted by the second - StoreThumb alone does not protect the head of the list")
	}
	if _, ok := g.CachedThumb(host.files[1]); !ok {
		t.Error("the second (most recently stored) thumbnail should still be cached")
	}
}

func TestRequestThumbnail_OutOfRangeIDIsNoop(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)

	cell, img := newCell()

	g.requestThumbnail(cell, img, 5, host.gen) // only index 0 exists
	g.Settle()

	if img.Image != nil {
		t.Error("an out-of-range id should paint nothing")
	}
}

// TestClaimRelease drives the in-flight bookkeeping directly, the same way
// TestStillWanted drives the staleness predicate: these decisions guard
// against duplicate decode goroutines, which no amount of waiting on real
// ones could assert on.
func TestClaimRelease(t *testing.T) {
	g := newOverview(t, hostWith(t, "a.jpg"))
	cell, _ := newCell()

	if !g.decodes.Claim(cell, thumbClaim{id: 0, revision: g.work.revision}) {
		t.Fatal("the first claim for a cell should allow a spawn")
	}
	if g.decodes.Claim(cell, thumbClaim{id: 0, revision: g.work.revision}) {
		t.Error("an identical claim while one is in flight must not spawn a second decode")
	}
	if !g.decodes.Claim(cell, thumbClaim{id: 1, revision: g.work.revision}) {
		t.Error("a claim for a different id should supersede the old one - the cell scrolled on")
	}

	g.decodes.Release(cell, thumbClaim{id: 0, revision: g.work.revision}) // the superseded decode finishing late
	if g.decodes.Claim(cell, thumbClaim{id: 1, revision: g.work.revision}) {
		t.Error("a stale release must not drop the newer claim")
	}

	g.decodes.Release(cell, thumbClaim{id: 1, revision: g.work.revision})
	if !g.decodes.Claim(cell, thumbClaim{id: 1, revision: g.work.revision}) {
		t.Error("after its own release, a cell should be claimable again")
	}
}

// TestRequestThumbnail_RecycledBeforeDecodeBailsAndReleases pins the
// worker's pre-decode bail: a request whose cell is recycled while the
// request waits for a slot must neither paint the cell nor keep its claim.
// The workers are parked (see parkDecodes), so the recycle deterministically
// wins the race against the decode.
func TestRequestThumbnail_RecycledBeforeDecodeBailsAndReleases(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)

	unpark := parkDecodes(t, g)

	g.requestThumbnail(cell, img, 0, host.gen)
	g.cellIDs.Store(cell, 1) // the cell scrolls on before a worker picks this up

	unpark()
	g.Settle()

	if img.Image != nil {
		t.Error("a decode whose cell scrolled away must not paint it")
	}
	if !g.decodes.Claim(cell, thumbClaim{id: 0, revision: g.work.revision}) {
		t.Error("the bailed decode should have released its claim")
	}
}

// TestRequestThumbnail_QueryChangeDiscardsInFlightDecode covers the
// staleness filtering adds on top of the two guards already here: the file
// set and the cell's own id can both still be current while the query
// underneath has renumbered the cells, so display cell 0 means a different
// file than the one this decode was started for. Same parking technique as
// the recycling test above - park the pool so the change deterministically
// beats the decode.
func TestRequestThumbnail_QueryChangeDiscardsInFlightDecode(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)

	unpark := parkDecodes(t, g)

	g.requestThumbnail(cell, img, 0, host.gen)

	// Display cell 0 now means b.jpg; the decode in flight is for a.jpg.
	typeQuery(g, "b")

	unpark()
	g.Settle()

	if img.Image != nil {
		t.Error("a decode started under a different query must not paint a.jpg into a cell now showing b.jpg")
	}
}

func TestStillWanted(t *testing.T) {
	host := hostWith(t, "a.jpg")
	host.gen = 7
	g := newOverview(t, host)

	cell, _ := newCell()
	g.cellIDs.Store(cell, 3)

	fgen := g.filterGen.Load()

	if !g.stillWanted(cell, 3, 7, fgen) {
		t.Error("a decode for the cell's current id at the current generation is still wanted")
	}
	if g.stillWanted(cell, 4, 7, fgen) {
		t.Error("a decode for an id this cell has since been recycled away from is stale")
	}
	if g.stillWanted(cell, 3, 6, fgen) {
		t.Error("a decode from a superseded generation is stale")
	}
	if g.stillWanted(cell, 3, 7, fgen+1) {
		t.Error("a decode resolved under a superseded query is stale")
	}

	other, _ := newCell()
	if g.stillWanted(other, 3, 7, fgen) {
		t.Error("a cell the grid has never tracked is stale")
	}
}

// heldGridRead pauses the first read before returning a chunk. The test must
// release that already-blocked call; context cancellation stops the next one.
type heldGridRead struct {
	entered     chan struct{}
	release     chan struct{}
	once        sync.Once
	enteredOnce sync.Once
	reads       atomic.Int32
	opens       atomic.Int32
	data        []byte
	err         error
}

func heldGridURI(t *testing.T, src fyne.URI, readErr error) (fyne.URI, *heldGridRead) {
	t.Helper()
	data, err := os.ReadFile(src.Path())
	if err != nil {
		t.Fatal(err)
	}
	h := &heldGridRead{entered: make(chan struct{}), release: make(chan struct{}), data: data, err: readErr}
	t.Cleanup(h.unblock)
	return uitest.ReaderURI(src, func() (io.ReadCloser, error) {
		h.opens.Add(1)
		r := bytes.NewReader(h.data)
		first := true
		return uitest.ReadCloser{
			ReadFunc: func(p []byte) (int, error) {
				h.reads.Add(1)
				if first {
					first = false
					h.enteredOnce.Do(func() { close(h.entered) })
					<-h.release
				}
				if h.err != nil {
					return 0, h.err
				}
				return r.Read(p)
			},
			CloseFunc: func() error { return nil },
		}, nil
	}), h
}
func (h *heldGridRead) unblock() { h.once.Do(func() { close(h.release) }) }
func (h *heldGridRead) wait(t *testing.T) {
	t.Helper()
	select {
	case <-h.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("grid read did not start")
	}
}

func TestRequestThumbnail_CloseAndStopCancelHeldRead(t *testing.T) {
	for _, tc := range []struct {
		name string
		stop func(*Overview)
	}{
		{"close", (*Overview).Close}, {"stop", (*Overview).Stop},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := hostWith(t, "source.jpg")
			u, held := heldGridURI(t, host.files[0], nil)
			host.files[0] = u
			g := newOverview(t, host)
			defer func() { held.unblock(); g.Settle() }()
			cell, img := newCell()
			g.cellIDs.Store(cell, 0)
			g.requestThumbnail(cell, img, 0, host.gen)
			held.wait(t)
			tc.stop(g)
			held.unblock()
			g.Settle()
			if held.reads.Load() != 1 {
				t.Errorf("cancelled source reads=%d, want 1", held.reads.Load())
			}
			if g.Cached(u) || img.Image != nil {
				t.Error("cancelled thumbnail reached cache or cell")
			}
			if _, ok := g.hashOf(u); ok {
				t.Error("cancelled thumbnail published a hash")
			}
			if _, ok := g.pixelCountOf(u); ok {
				t.Error("cancelled thumbnail published native size")
			}
		})
	}
}

func TestRequestThumbnail_CancelledSlotWaiterReleasesClaim(t *testing.T) {
	host := hostWith(t, "source.jpg")
	u, held := heldGridURI(t, host.files[0], nil)
	host.files[0] = u
	g := newOverview(t, host)
	synctest.Test(t, func(t *testing.T) {
		// Slot waits must belong to this bubble to be durably blocked.
		g.decodes = decodepool.New[*fyne.Container, thumbClaim](thumbConcurrency)
		g.hashes.pool = g.decodes
		unpark := parkDecodes(t, g)
		cell, img := newCell()
		g.cellIDs.Store(cell, 0)
		g.requestThumbnail(cell, img, 0, host.gen)
		synctest.Wait()
		g.Close()
		synctest.Wait()
		if !g.decodes.Claim(cell, thumbClaim{id: 0, revision: g.work.revision}) {
			t.Error("cancelled slot waiter retained its claim while all slots remained occupied")
		}
		g.decodes.Release(cell, thumbClaim{id: 0, revision: g.work.revision})
		held.unblock()
		unpark()
		g.Settle()
		if held.opens.Load() != 0 {
			t.Errorf("cancelled slot waiter opened %d sources", held.opens.Load())
		}
	})
}

func TestHashRemaining_CloseCancelsReadAndNativeBackfill(t *testing.T) {
	for _, kind := range []string{"decode", "failure", "native-backfill"} {
		t.Run(kind, func(t *testing.T) {
			host := hostWith(t, "source.jpg")
			var readErr error
			if kind == "failure" {
				readErr = errors.New("broken source")
			}
			u, held := heldGridURI(t, host.files[0], readErr)
			host.files[0] = u
			g := newOverview(t, host)
			defer func() { held.unblock(); g.Settle() }()
			if kind == "native-backfill" {
				g.dupes.PutHash(u.String(), 42)
				g.StoreThumb(u, image.NewRGBA(image.Rect(0, 0, 4, 4)))
			}
			callbacks := 0
			g.SetOnDupeStateChanged(func() { callbacks++ })
			if n := g.hashRemaining(); n != 1 {
				t.Fatalf("hash jobs=%d, want 1", n)
			}
			held.wait(t)
			g.Close()
			held.unblock()
			g.Settle()
			if held.reads.Load() != 1 {
				t.Errorf("cancelled hash source reads=%d, want 1", held.reads.Load())
			}
			if g.dupes.Failed(u.String()) {
				t.Error("cancelled hash read became a source failure")
			}
			if _, ok := g.pixelCountOf(u); ok {
				t.Error("cancelled hash read published native size")
			}
			if kind != "native-backfill" {
				if _, ok := g.hashOf(u); ok {
					t.Error("cancelled hash read published a hash")
				}
				if g.Cached(u) {
					t.Error("cancelled hash read cached its thumbnail")
				}
			}
			if callbacks != 0 {
				t.Errorf("cancelled hash work delivered %d callbacks", callbacks)
			}
		})
	}
}

func TestRequestThumbnail_ReopenKeepsNewClaimWhileOldReadStops(t *testing.T) {
	host := hostWith(t, "source.jpg")
	base := host.files[0]
	g := newOverview(t, host)
	synctest.Test(t, func(t *testing.T) {
		g.decodes = decodepool.New[*fyne.Container, thumbClaim](thumbConcurrency)
		g.hashes.pool = g.decodes
		oldURI, oldRead := heldGridURI(t, base, nil)
		newURI, newRead := heldGridURI(t, base, nil)
		defer func() { oldRead.unblock(); newRead.unblock(); g.Stop(); g.Settle() }()
		host.files[0] = oldURI
		cell, img := newCell()
		g.cellIDs.Store(cell, 0)
		g.requestThumbnail(cell, img, 0, host.gen)
		<-oldRead.entered
		g.Close()
		host.files[0] = newURI
		g.Toggle() // Actual reopening starts a new session over the same file set.
		g.requestThumbnail(cell, img, 0, host.gen)
		synctest.Wait()
		oldRead.unblock()
		synctest.Wait()
		claim := thumbClaim{id: 0, revision: g.work.revision}
		if g.decodes.Claim(cell, claim) {
			t.Error("old release erased the reopened cell's new claim")
		}
		newRead.unblock()
		g.Settle()
		if img.Image == nil {
			t.Error("reopened cell did not receive its current thumbnail")
		}
		if oldRead.reads.Load() != 1 {
			t.Errorf("old read continued for %d chunks", oldRead.reads.Load())
		}
	})
}

type notifiedGridQueue struct {
	uitest.UIQueue
	queued chan struct{}
}

func (q *notifiedGridQueue) Do(fn func()) { q.UIQueue.Do(fn); q.queued <- struct{}{} }

func TestHashRemaining_ReopenFinishesNewPassBeforeOldReadReturns(t *testing.T) {
	host := hostWith(t, "source.jpg")
	base := host.files[0]
	oldURI, oldRead := heldGridURI(t, base, nil)
	newURI, newRead := heldGridURI(t, base, nil)
	host.files[0] = oldURI
	g := newOverview(t, host)
	defer func() { oldRead.unblock(); newRead.unblock(); g.Stop(); g.Settle() }()
	queue := &notifiedGridQueue{queued: make(chan struct{}, 8)}
	g.SetUIQueue(queue)
	callbacks := 0
	g.SetOnDupeStateChanged(func() { callbacks++ })
	if n := g.hashRemaining(); n != 1 {
		t.Fatalf("old jobs=%d, want 1", n)
	}
	oldRead.wait(t)
	g.Close()
	host.files[0] = newURI
	if n := g.hashRemaining(); n != 1 {
		t.Fatalf("new jobs=%d, old URI claim suppressed the new pass", n)
	}
	newRead.wait(t)
	newRead.unblock()
	select {
	case <-queue.queued:
	case <-time.After(5 * time.Second):
		t.Fatal("new pass did not finish while old read remained held")
	}
	queue.Drain()
	if callbacks != 1 {
		t.Errorf("new pass completion callbacks=%d, want 1", callbacks)
	}
	oldRead.unblock()
	g.Settle()
	if callbacks != 1 {
		t.Errorf("old pass changed completion count to %d", callbacks)
	}
}

func TestGridWork_SourceChangeCancelsOldReadAndAllowsRetry(t *testing.T) {
	host := hostWith(t, "source.jpg")
	base := host.files[0]
	oldURI, oldRead := heldGridURI(t, base, errors.New("old source failure"))
	newURI, newRead := heldGridURI(t, base, nil)
	host.files[0] = oldURI
	g := newOverview(t, host)
	defer func() { oldRead.unblock(); newRead.unblock(); g.Stop(); g.Settle() }()
	g.hashRemaining()
	oldRead.wait(t)
	host.gen++
	host.files[0] = newURI
	g.FilesChanged()
	oldRead.unblock()
	g.Settle()
	if g.dupes.Failed(newURI.String()) {
		t.Fatal("superseded read poisoned the replacement's retry")
	}
	newRead.unblock()
	if n := g.hashRemaining(); n != 1 {
		t.Fatalf("replacement jobs=%d, want 1", n)
	}
	g.Settle()
	if _, ok := g.hashOf(newURI); !ok {
		t.Error("replacement was not hashed")
	}
	if size, ok := g.pixelCountOf(newURI); !ok || size != 64 {
		t.Errorf("replacement size=%d, %v; want 64/true", size, ok)
	}
}

func TestGridWork_StopPreventsFreshAdmission(t *testing.T) {
	host := hostWith(t, "source.jpg")
	u, held := heldGridURI(t, host.files[0], nil)
	host.files[0] = u
	g := newOverview(t, host)
	defer func() { held.unblock(); g.Settle() }()
	held.unblock()
	g.Stop()
	g.Toggle()
	if g.visible {
		t.Error("stopped grid reopened")
	}
	if n := g.hashRemaining(); n != 0 {
		t.Errorf("stopped grid started %d hash jobs", n)
	}
	if err := g.Warm(); !errors.Is(err, context.Canceled) {
		t.Errorf("stopped warm = %v", err)
	}
	cell, img := newCell()
	g.cellIDs.Store(cell, 0)
	g.requestThumbnail(cell, img, 0, host.gen)
	held.unblock()
	g.Settle()
	if held.opens.Load() != 0 {
		t.Errorf("stopped grid opened %d sources", held.opens.Load())
	}
}

func TestGridWork_CloseDiscardsQueuedDelivery(t *testing.T) {
	for _, kind := range []string{"cell", "hash"} {
		t.Run(kind, func(t *testing.T) {
			host := hostWith(t, "source.jpg")
			g := newOverview(t, host)
			cell, img := newCell()
			callbacks := 0
			g.SetOnDupeStateChanged(func() { callbacks++ })
			if kind == "cell" {
				g.cellIDs.Store(cell, 0)
				g.requestThumbnail(cell, img, 0, host.gen)
			} else {
				g.hashRemaining()
			}
			g.decodes.Wait() // Submission is inside each owning worker.
			if g.ui.(*uitest.UIQueue).Len() == 0 {
				t.Fatal("worker did not enqueue its delivery")
			}
			g.Close()
			g.Settle()
			if img.Image != nil || callbacks != 0 {
				t.Errorf("closed delivery painted=%v callbacks=%d", img.Image != nil, callbacks)
			}
		})
	}
}

func TestToggle_ReopenResumesIncompleteHideAnalysis(t *testing.T) {
	// Uniform hashes are deliberately excluded from duplicate groups.
	host := hostPatterned(t, []string{"a.jpg", "b.jpg"}, []int{1, 1})
	base := host.files[0]
	oldURI, oldRead := heldGridURI(t, base, nil)
	newURI, newRead := heldGridURI(t, base, nil)
	host.files[0] = oldURI
	g := newOverview(t, host)
	defer func() { oldRead.unblock(); newRead.unblock(); g.Stop(); g.Settle() }()
	g.SetHideDuplicates(true)
	oldRead.wait(t)
	g.Close()
	oldRead.unblock()
	g.Settle()
	host.files[0] = newURI
	g.Toggle()
	if g.hashes.hashJobs.Load() == 0 {
		t.Error("reopening did not resume incomplete hide analysis")
	}
	newRead.unblock()
	g.Settle()
	if got := g.dupes.GroupSize(0); got != 2 {
		t.Errorf("reopened duplicate group size=%d, want 2", got)
	}
}

type heldFirstGridQueue struct {
	uitest.UIQueue
	calls        atomic.Int32
	first        chan struct{}
	releaseFirst chan struct{}
	second       chan struct{}
}

func (q *heldFirstGridQueue) Do(fn func()) {
	n := q.calls.Add(1)
	if n == 1 {
		close(q.first)
		<-q.releaseFirst
	}
	q.UIQueue.Do(fn)
	if n == 2 {
		close(q.second)
	}
}

func TestHashRemaining_LatePartialDeliveryCannotReplaceFinalGroups(t *testing.T) {
	// Uniform hashes are deliberately excluded from duplicate groups.
	host := hostPatterned(t, []string{"a.jpg", "b.jpg"}, []int{1, 1})
	u, held := heldGridURI(t, host.files[0], nil)
	host.files[0] = u
	g := newOverview(t, host)
	queue := &heldFirstGridQueue{first: make(chan struct{}), releaseFirst: make(chan struct{}), second: make(chan struct{})}
	var once sync.Once
	releaseFirst := func() { once.Do(func() { close(queue.releaseFirst) }) }
	defer func() { held.unblock(); releaseFirst(); g.Stop(); g.Settle() }()
	g.SetUIQueue(queue)
	g.SetHideDuplicates(true)
	// B computes a partial snapshot while A is still reading, then is held
	// immediately before enqueueing. A's final snapshot reaches the queue first.
	select {
	case <-queue.first:
	case <-time.After(5 * time.Second):
		t.Fatal("partial snapshot did not reach submission")
	}
	held.unblock()
	select {
	case <-queue.second:
	case <-time.After(5 * time.Second):
		t.Fatal("final snapshot did not enqueue")
	}
	releaseFirst()
	g.Settle()
	if got := g.dupes.GroupSize(0); got != 2 {
		t.Errorf("late partial snapshot replaced final group: size=%d, want 2", got)
	}
}

// Keep the work context alive: generation admission must protect facts even
// when cancellation has not yet reached a worker finishing an old same-URI read.
func TestGridFacts_RejectOldReadsAcrossReplacementAndReset(t *testing.T) {
	for _, reset := range []bool{false, true} {
		for _, path := range []string{"hash", "hash-failure", "cell", "cell-failure", "native", "native-failure"} {
			name := fmt.Sprintf("reset=%v/%s", reset, path)
			t.Run(name, func(t *testing.T) {
				host := hostWith(t, "source.jpg")
				var readErr error
				if strings.HasSuffix(path, "failure") {
					readErr = errors.New("old read failed")
				}
				u, held := heldGridURI(t, host.files[0], readErr)
				host.files[0] = u
				g := newOverview(t, host)
				defer func() { held.unblock(); g.Stop(); g.Settle() }()
				ctx := g.work.ctx
				backfill := strings.HasPrefix(path, "native")
				if backfill {
					g.thumbs.Add(u.String(), image.NewNRGBA(image.Rect(0, 0, 2, 2)))
					g.dupes.PutHash(u.String(), 7)
				}
				cell := container.NewStack()
				img := canvas.NewImageFromImage(nil)
				callbacks := 0
				g.SetOnDupeStateChanged(func() { callbacks++ })
				if strings.HasPrefix(path, "cell") {
					g.cellIDs.Store(cell, 0)
					g.requestThumbnail(cell, img, 0, host.gen)
				} else {
					g.hashRemaining()
				}
				held.wait(t)
				if reset {
					g.dupes.Clear()
				} else {
					host.gen++
					g.dupes.WipeIfStale()
				}
				// These represent facts established by a replacement worker.
				g.dupes.PutHash(u.String(), 99)
				g.dupes.PutNativeSize(u.String(), image.Pt(100, 200))
				held.unblock()
				g.Settle()
				if ctx.Err() != nil {
					t.Fatal("fixture cancelled old work; admission was not exercised")
				}
				if h, ok := g.dupes.Hash(u.String()); !ok || h != 99 {
					t.Errorf("old hash replaced current: %d/%v", h, ok)
				}
				if sz, ok := g.dupes.NativeSize(u.String()); !ok || sz != image.Pt(100, 200) {
					t.Errorf("old native size replaced current: %v/%v", sz, ok)
				}
				if g.dupes.Failed(u.String()) {
					t.Error("old failure poisoned current source")
				}
				if !backfill && g.Cached(u) {
					t.Error("old pixels entered cache")
				}
				if img.Image != nil || callbacks != 0 {
					t.Errorf("old delivery: image=%v callbacks=%d", img.Image != nil, callbacks)
				}
			})
		}
	}
}

func TestGridFacts_ResetDiscardsQueuedDeliveryAndAllowsRetry(t *testing.T) {
	for _, cellPath := range []bool{false, true} {
		t.Run(fmt.Sprintf("cell=%v", cellPath), func(t *testing.T) {
			host := hostWith(t, "source.jpg")
			g := newOverview(t, host)
			cell := container.NewStack()
			img := canvas.NewImageFromImage(nil)
			callbacks := 0
			g.SetOnDupeStateChanged(func() { callbacks++ })
			if cellPath {
				g.cellIDs.Store(cell, 0)
				g.requestThumbnail(cell, img, 0, host.gen)
			} else {
				g.hashRemaining()
			}
			g.decodes.Wait()
			g.dupes.Clear()
			g.Settle()
			if callbacks != 0 || img.Image != nil {
				t.Error("reset accepted already-queued delivery")
			}
			if n := g.hashRemaining(); n != 1 {
				t.Fatalf("reset retry jobs=%d, want 1", n)
			}
			g.Settle()
			if _, ok := g.hashOf(host.files[0]); !ok {
				t.Error("reset prevented a new hash")
			}
			if size, ok := g.pixelCountOf(host.files[0]); !ok || size != 64 {
				t.Errorf("retry native size=%d/%v", size, ok)
			}
		})
	}
}

func TestGridFacts_OldFailureCannotSuppressReplacementRetry(t *testing.T) {
	host := hostWith(t, "source.jpg")
	base := host.files[0]
	oldURI, old := heldGridURI(t, base, errors.New("obsolete failure"))
	newURI, current := heldGridURI(t, base, nil)
	host.files[0] = oldURI
	g := newOverview(t, host)
	defer func() { old.unblock(); current.unblock(); g.Stop(); g.Settle() }()
	g.hashRemaining()
	old.wait(t)
	host.gen++
	host.files[0] = newURI
	g.dupes.WipeIfStale()
	old.unblock()
	g.Settle()
	current.unblock()
	if n := g.hashRemaining(); n != 1 {
		t.Fatalf("replacement retry jobs=%d, want 1", n)
	}
	g.Settle()
	if g.dupes.Failed(newURI.String()) {
		t.Error("replacement retained old failure")
	}
	if _, ok := g.hashOf(newURI); !ok {
		t.Error("replacement not hashed")
	}
}

func TestInvalidateContentDropsPixelsAndFactsWithoutResettingSelection(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	unpark := parkDecodes(t, g)
	defer unpark()
	if err := g.Warm(); err != nil {
		t.Fatal(err)
	}
	g.searching = true
	g.SelectAll()
	g.highlight = 1
	g.dupes.SetHideDuplicates(true)
	ctx, facts, writer := g.work.ctx, g.work.facts, g.CaptureThumbs()
	g.InvalidateContent()
	if ctx.Err() == nil {
		t.Error("old readers were not cancelled")
	}
	if facts.PutHash(host.files[0].String(), 99) {
		t.Error("old facts were still accepted")
	}
	old := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if writer.AddIfFits(host.files[0].String(), old) {
		t.Error("old preview writer repopulated invalidated pixels")
	}
	for _, u := range host.files {
		if g.Cached(u) {
			t.Error("cached pixels survived content invalidation")
		}
		if _, ok := g.dupes.Hash(u.String()); ok {
			t.Error("derived hash survived content invalidation")
		}
	}
	if g.SelectionCount() != 2 || g.fileIndex(g.highlight) != 1 || !g.Searching() || !g.dupes.HideDuplicates() {
		t.Errorf("view reset: selection=%v highlight=%d search=%v hide=%v", g.Selection(), g.fileIndex(g.highlight), g.Searching(), g.dupes.HideDuplicates())
	}
}
