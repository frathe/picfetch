// The duplicate-hash engine: the pool-driven pass that fills the model with
// dHashes and native pixel sizes, the accounting that tells the last job it
// is the last, and the throttle that keeps the UI goroutine answering input
// while a whole folder hashes.

package grid

import (
	"context"
	"image"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/decodepool"
	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/imaging"
)

// hideApplyMinInterval is how far apart the engine may schedule hide
// applies while jobs are still running. The last job always applies.
// Without a floor, an idle UI drains one install and the next hash
// immediately queues another, so the event loop never sees input until
// the pool is empty.
const hideApplyMinInterval = 250 * time.Millisecond

// hashEngine is the hashing pass behind hide-duplicates and browse: it
// walks the host's file set for files the model has no dHash or native
// size for yet, hashes them on the grid's decode pool, and throttles how
// often the results are installed on the UI goroutine.
//
// A type of its own rather than more methods on Overview because none of
// this is presentation, and everything it owns is concurrent: a sync.Map
// and three atomics that four decode workers write at once share nothing
// with the marquee's geometry or the search string sitting next to them
// on the overlay. What it needs from the grid it holds directly, so it
// keeps no pointer back to the Overview; the part of a completion that
// does have to touch the overlay comes back through Run's apply callback
// instead (Overview.applyHashSnapshot).
type hashEngine struct {
	// host, pool, thumbs, model and ui are the Overview's own, shared
	// rather than copied: the engine hashes onto the same decode pool the
	// cells decode on - which is what keeps Settle's decodes.Wait barrier
	// covering hash jobs too - fills the same thumbnail cache, and
	// installs into the same duplicate model the badges and the filter
	// read.
	host   Host
	pool   *decodepool.Pool[*fyne.Container, int]
	thumbs *imaging.ByteCache[image.Image]
	model  *dupes.Model

	// ui is how a finished job's install reaches the UI goroutine - see
	// uiqueue.go for why that is a field and not a direct fyne.Do.
	// SetUIQueue writes this one as well as the Overview's, because a
	// test that installed uitest.UIQueue on only one of them would still
	// have hashing completions running inline on the decode worker.
	ui UIQueue

	// hashing dedups in-flight Run jobs by URI string.
	hashing sync.Map

	// hashJobs counts those pool jobs so the last one can finishBrowse.
	hashJobs atomic.Int32

	// hideApply stays set until the in-flight UI install returns, so an
	// idle fyne.Do cannot re-arm mid-apply and queue one install per
	// file. hideApplyAt floors mid-window installs so the event loop
	// still sees input while hashing; beginPass clears it between passes
	// so a stale floor from a finished pass cannot swallow a new pass's
	// first mid-window apply.
	hideApply   atomic.Bool
	hideApplyAt atomic.Int64
}

// Run hashes every file that does not already have a dHash, and records
// native pixel counts for files that have a hash but no size, returning
// how many jobs it queued. Cache hits join the thumbnail pool the same
// way misses do — dHashing them on the D-key goroutine froze the UI for
// any folder that already fit in the thumb cache. Jobs have no per-cell
// Claim so Settle still waits, and they do not Add to a full thumbnail
// cache.
//
// DuplicateGroups runs on the worker before e.ui.Do. apply only installs
// that snapshot and filters, unless the duplicate distance changed since
// the snapshot (settings slider while hashing): then it recomputes so the
// install cannot undo the live regroup. hideApply stays set until apply
// returns so an idle UI cannot re-arm mid-apply. Mid-window applies are
// also floored by hideApplyMinInterval; the last job always applies.
// Browse still waits for the last job (finishBrowse) so a partial group
// is never shown. e.ui.Do stays inside this Go body: Settle's barrier is
// decodes.Wait, which only covers completions the pool spawned.
//
// apply is Overview.applyHashSnapshot - the half of a completion that has
// to touch the overlay, and so has to run on the UI goroutine. It is
// handed the snapshot the worker computed, how many jobs were still
// outstanding when this one finished, and the generation this pass
// started at.
func (e *hashEngine) Run(apply func(snap dupes.Groups, remaining int32, gen uint64)) int {
	gen := e.host.Generation()
	e.model.WipeIfStale()

	type hashJob struct {
		file   fyne.URI
		key    string
		thumb  image.Image
		hashed bool
		sized  bool
	}
	var jobs []hashJob
	for i := 0; i < e.host.FileCount(); i++ {
		u := e.host.FileAt(i)
		// An index with no URI has nothing this pass can dedup, cache or
		// hash by, so skip it rather than dereferencing it below. Every
		// neighbouring helper (rememberHash, hashOf, ... in
		// grid/dupes.go) guards the same way.
		if u == nil {
			continue
		}
		// The URI string is the model's key throughout - it stores facts
		// about files, not fyne.URIs. Read straight off the model here
		// rather than through Overview's hashOf/pixelCountOf/
		// hashFailedOf wrappers, which exist to nil-guard the fyne.URI
		// the cell and Warm paths hand them; the guard above is this
		// loop's equivalent.
		key := u.String()
		_, hashed := e.model.Hash(key)
		_, sized := e.model.PixelCount(key)
		if hashed && sized {
			continue
		}
		if e.model.Failed(key) {
			continue
		}
		if _, loaded := e.hashing.LoadOrStore(key, true); loaded {
			continue
		}
		job := hashJob{file: u, key: key, hashed: hashed, sized: sized}
		if thumb, ok := e.thumbs.Get(key); ok {
			job.thumb = thumb
		}
		jobs = append(jobs, job)
	}
	n := len(jobs)
	if n == 0 {
		return 0
	}
	e.beginPass(n)
	for _, j := range jobs {
		file, key, cached, hashed, sized := j.file, j.key, j.thumb, j.hashed, j.sized
		e.pool.Go(context.Background(), func(acquired bool) {
			defer func() {
				e.hashing.Delete(key)
				remaining := e.hashJobs.Add(-1)
				if !e.shouldScheduleHideApply(remaining) {
					return
				}
				snap := e.model.Compute()
				e.ui.Do(func() {
					defer e.hideApply.Store(false)
					apply(snap, remaining, gen)
				})
			}()
			if !acquired || gen != e.host.Generation() {
				return
			}
			thumb := cached
			var native image.Rectangle
			haveNative := false
			if thumb == nil {
				var err error
				thumb, native, err = imaging.LoadThumbnailAndBounds(file)
				if err != nil || thumb == nil {
					e.model.PutFailed(key)
					return
				}
				haveNative = true
				if !thumbCacheFull(e.thumbs) {
					e.thumbs.AddIfFits(file.String(), thumb)
				}
			}
			if !hashed {
				// dHashed here, on the goroutine that already holds the
				// decoded thumbnail, so the model is only ever handed the
				// 8-byte result: it knows nothing about images.
				e.model.PutHash(key, imaging.DifferenceHash(thumb))
			}
			if !sized {
				// Rectangle to plain size at the call, because the model
				// stores sizes rather than rectangles with an origin; it
				// applies its own clamp for a negative edge.
				if haveNative {
					e.model.PutNativeSize(key, image.Pt(native.Dx(), native.Dy()))
				} else {
					_, b, err := imaging.ReadAndProbe(context.Background(), file)
					if err != nil {
						// Known-zero so hide/browse stop re-queueing an
						// unprobeable file (same trade-off as hashFailed).
						b = image.Rectangle{}
					}
					e.model.PutNativeSize(key, image.Pt(b.Dx(), b.Dy()))
				}
			}
		})
	}
	return n
}

// beginPass books n new jobs onto the counter and, when this is the first
// work in flight, clears the mid-window throttle floor.
//
// shouldScheduleHideApply leaves hideApplyAt at the previous pass's last
// mid-window timestamp when that pass ended, so without this a pass
// starting within hideApplyMinInterval of the old one's last apply would
// skip its own first mid-window apply. The last job always applies, so
// that was latency, not lost work - this removes the latency.
//
// The Load/Add pair is deliberately not atomic as a unit: a worker
// finishing between the two can only make this look like a continuing
// pass and keep a floor that is about to expire anyway. The cost of
// losing that race is one throttled apply, never a wrong result.
func (e *hashEngine) beginPass(n int) {
	if e.hashJobs.Load() == 0 {
		e.hideApplyAt.Store(0)
	}
	e.hashJobs.Add(int32(n))
}

func (e *hashEngine) shouldScheduleHideApply(remaining int32) bool {
	if remaining == 0 {
		e.hideApply.Store(true)
		return true
	}
	now := time.Now().UnixMilli()
	last := e.hideApplyAt.Load()
	if last != 0 && now-last < hideApplyMinInterval.Milliseconds() {
		return false
	}
	if !e.hideApply.CompareAndSwap(false, true) {
		return false
	}
	e.hideApplyAt.Store(now)
	return true
}
