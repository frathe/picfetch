package grid

import (
	"context"
	"image"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/imaging"
)

// hideApplyMinInterval is how far apart hashRemaining may schedule hide
// applies while jobs are still running. The last job always applies.
// Without a floor, an idle UI drains one install and the next hash
// immediately queues another, so the event loop never sees input until
// the pool is empty.
const hideApplyMinInterval = 250 * time.Millisecond

// hostSet adapts Host to dupes.FileSet, so the model can group over the
// grid's file set while staying Fyne-free: every fact it stores is keyed
// by the URI string the grid already keys its own caches by.
//
// KeyAt has to stay a plain lookup. dupes.Model.Compute calls it while
// holding the model's own mutex - faithfully to the code this replaced,
// which read g.host.FileAt(i) under hashMu - so anything here that took a
// lock, or reached back into the model, would deadlock a hashing worker.
type hostSet struct {
	host Host
}

func (s hostSet) Count() int { return s.host.FileCount() }

// KeyAt is the URI string of the file at i, or "" when the host has no
// URI there: the same nil-URI guard every helper below applies before it
// touches a fyne.URI, moved to the one place the model reaches through.
func (s hostSet) KeyAt(i int) string {
	u := s.host.FileAt(i)
	if u == nil {
		return ""
	}

	return u.String()
}

func (s hostSet) Generation() uint64 { return s.host.Generation() }

// adoptHashGen records the host's current generation without dropping the
// model's hashes, hash failures, or native sizes. Incremental shrink
// (RemoveFiles → FilesChanged) is not a new drop: surviving files keep
// their hashes so hide-duplicates grouping and inspect retarget still
// work. Do not route this through wipeHashesIfStale - that wipes on a
// mismatch; dupes.Model.AdoptGeneration carries the full reasoning.
func (g *Overview) adoptHashGen() {
	g.dupes.AdoptGeneration()
}

// rememberHash dHashes img here, on whichever goroutine already holds the
// decoded thumbnail, and hands the model only the 8-byte result: the
// model knows nothing about images.
func (g *Overview) rememberHash(u fyne.URI, img image.Image) {
	if u == nil || img == nil {
		return
	}
	g.dupes.PutHash(u.String(), imaging.DifferenceHash(img))
}

func (g *Overview) rememberHashFail(u fyne.URI) {
	if u == nil {
		return
	}
	g.dupes.PutFailed(u.String())
}

// rememberNative records u's native pixel size. The conversion from a
// probe's rectangle to a plain size happens here because the model stores
// sizes, not rectangles with an origin; it applies its own clamp for a
// negative edge.
func (g *Overview) rememberNative(u fyne.URI, native image.Rectangle) {
	if u == nil {
		return
	}
	g.dupes.PutNativeSize(u.String(), image.Pt(native.Dx(), native.Dy()))
}

func (g *Overview) hashOf(u fyne.URI) (uint64, bool) {
	if u == nil {
		return 0, false
	}

	return g.dupes.Hash(u.String())
}

func (g *Overview) hashFailedOf(u fyne.URI) bool {
	if u == nil {
		return false
	}

	return g.dupes.Failed(u.String())
}

func (g *Overview) pixelCountOf(u fyne.URI) (int, bool) {
	if u == nil {
		return 0, false
	}

	return g.dupes.PixelCount(u.String())
}

// NativeSize is the EXIF-oriented pixel size of the file at hostIndex.
// ok is false when the index is out of range or no probe has been stored,
// or when a stored size has a non-positive edge (failed/empty probe).
func (g *Overview) NativeSize(hostIndex int) (w, h int, ok bool) {
	return g.dupes.NativeSizeAt(hostIndex)
}

func (g *Overview) wipeHashesIfStale() {
	g.dupes.WipeIfStale()
}

func (g *Overview) clearHashes() {
	g.dupes.Clear()
}

// SetOnDupeStateChanged registers f to run after hide, browse, last-job
// hash apply, or duplicate-distance changes. The field is read at fire
// time. nil is a no-op.
func (g *Overview) SetOnDupeStateChanged(f func()) { g.onDupeState = f }

func (g *Overview) fireDupeState() {
	if g.onDupeState != nil {
		g.onDupeState()
	}
}

// SourceDuplicateGroupSize is the duplicate-group size of the source
// file: the highlight while the grid is visible, otherwise the host's
// current index. 0 means groups have not been built yet.
func (g *Overview) SourceDuplicateGroupSize() int {
	src := g.host.CurrentIndex()
	if g.visible {
		src = g.fileIndex(g.highlight)
	}
	return g.groupSize(src)
}

// HideDuplicates reports whether extras are currently hidden.
func (g *Overview) HideDuplicates() bool {
	return g.dupes.HideDuplicates()
}

// SetHideDuplicates turns extra-hiding on or off. Turning it on hashes any
// files that have not been hashed yet (cache hits and misses both join the
// decode pool so D never dHashes on the key-handler goroutine) and
// jumps the host to the group's representative if the current file is an
// extra. Close does not clear this flag: the viewer still skips extras
// after the grid is dismissed.
func (g *Overview) SetHideDuplicates(on bool) {
	if !g.dupes.SetHideDuplicates(on) {
		return
	}
	if on {
		_ = g.hashRemaining()
	}
	g.applyFilter()
	if on {
		g.jumpIfHiddenExtra()
	}
	g.fireDupeState()
}

// BrowsingDuplicates reports whether the grid is showing a single duplicate
// group.
func (g *Overview) BrowsingDuplicates() bool {
	return g.browseHost >= 0
}

// SetBrowsingDuplicates turns group-browsing on or off. Turning it on hashes
// any files that have not been hashed yet and filters the grid to the source
// file's duplicate group. A unique source is a silent no-op.
func (g *Overview) SetBrowsingDuplicates(on bool) {
	if !on {
		if g.browseHost < 0 {
			return
		}
		g.browseHost = -1
		g.applyFilter()
		g.fireDupeState()
		return
	}

	src := g.host.CurrentIndex()
	if g.visible {
		src = g.fileIndex(g.highlight)
	}
	if src < 0 {
		return
	}
	// Set before hashRemaining so an inline last-job fyne.Do can finishBrowse.
	g.browseHost = src
	pending := g.hashRemaining()
	if pending > 0 {
		g.host.ShowToast(lang.L("The images are currently being analyzed"))
		g.fireDupeState()
	}
	if pending == 0 {
		g.finishBrowse()
	}
}

// ToggleBrowseDuplicates turns group-browsing off if it is on, and on if it
// is off.
func (g *Overview) ToggleBrowseDuplicates() {
	g.SetBrowsingDuplicates(!g.BrowsingDuplicates())
}

// finishBrowse applies the group filter once hashes are ready. A unique
// source leaves browse off with no toast.
func (g *Overview) finishBrowse() {
	if g.browseHost < 0 {
		return
	}
	// Warm records hashes but does not rebuild groups; applyFilter is the
	// usual rebuild site, and groupSize must see the rebuilt sizes first.
	g.rebuildGroups()
	if g.groupSize(g.browseHost) < 2 {
		g.browseHost = -1
		g.applyFilter()
		g.fireDupeState()
		return
	}
	g.applyFilter()
	if g.visible {
		id := 0
		if d := g.displayIndexOfHost(g.browseHost); d >= 0 {
			id = d
		}
		g.setHighlight(id)
		g.wrap.ScrollTo(id)
	}
	g.fireDupeState()
}

// groupMembers returns host indices in the same duplicate group as
// hostIndex, in host-index order, or nil when the group has fewer than two
// members. It reads the model's installed group snapshot (same as
// IsHiddenExtra / RepresentativeOf) and does not rebuild.
func (g *Overview) groupMembers(hostIndex int) []int {
	return g.dupes.Members(hostIndex)
}

// BeginInspect starts an inspect session on hostIndex so a hidden extra
// can stay on screen. Out-of-range or missing files clear inspect instead.
func (g *Overview) BeginInspect(hostIndex int) {
	g.dupes.BeginInspect(hostIndex)
}

// ClearInspect ends the inspect session.
func (g *Overview) ClearInspect() {
	g.dupes.ClearInspect()
}

// InspectingDuplicates reports whether an inspect session is active.
func (g *Overview) InspectingDuplicates() bool {
	return g.dupes.Inspecting()
}

func (g *Overview) inspectSource() int {
	return g.dupes.InspectSource()
}

// InspectMembers returns host indices of the inspected file's duplicate
// group in host-index order, or nil when inspect is off.
func (g *Overview) InspectMembers() []int {
	return g.dupes.InspectMembers()
}

func (g *Overview) jumpIfHiddenExtra() {
	if g.InspectingDuplicates() {
		return
	}
	if i := g.host.CurrentIndex(); g.IsHiddenExtra(i) {
		g.host.ShowImage(g.RepresentativeOf(i))
	}
}

// SetDuplicateDistance sets the Hamming threshold - the model clamps it
// to 0–32 - and rebuilds groups. Live: if browsing, the group is
// re-checked and browse exits when it drops below two members. If hide is
// on and not browsing, extras are recomputed immediately and the host
// jumps if the current file became an extra.
func (g *Overview) SetDuplicateDistance(n int) {
	if !g.dupes.SetDistance(n) {
		return
	}
	if g.browseHost >= 0 {
		g.finishBrowse()
	} else if g.HideDuplicates() {
		g.applyFilter()
		g.jumpIfHiddenExtra()
	} else {
		g.rebuildGroups()
	}
	g.fireDupeState()
}

func (g *Overview) duplicateDistance() int {
	return g.dupes.Distance()
}

// IsHiddenExtra reports whether hostIndex is a non-representative member of
// a duplicate group while hide is on. Unhashed files are never extras.
func (g *Overview) IsHiddenExtra(hostIndex int) bool {
	return g.dupes.IsHiddenExtra(hostIndex)
}

// RepresentativeOf is the highest native pixel count in the group, lowest
// host index on a tie; itself when unique, unhashed, or out of range.
func (g *Overview) RepresentativeOf(hostIndex int) int {
	return g.dupes.RepresentativeOf(hostIndex)
}

// groupSize is 0 if hostIndex is unhashed, 1 if it is a unique hashed file,
// and ≥2 if it belongs to a duplicate group.
func (g *Overview) groupSize(hostIndex int) int {
	return g.dupes.GroupSize(hostIndex)
}

func (g *Overview) rebuildGroups() {
	g.dupes.Rebuild()
}

// hashRemaining hashes every file that does not already have a dHash,
// and records native pixel counts for files that have a hash but no
// size. Cache hits join the thumbnail pool the same way misses do —
// dHashing them on the D-key goroutine froze the UI for any folder that
// already fit in the thumb cache. Jobs have no per-cell Claim so Settle
// still waits, and they do not Add to a full thumbnail cache.
//
// DuplicateGroups runs on the worker before g.ui.Do. The callback only
// installs that snapshot and filters, unless the duplicate distance
// changed since the snapshot (settings slider while hashing): then it
// recomputes so the install cannot undo the live regroup. hideApply
// stays set until the callback returns so an idle UI cannot re-arm
// mid-apply. Mid-window applies are also floored by
// hideApplyMinInterval; the last job always applies. Browse still waits
// for the last job (finishBrowse) so a partial group is never shown.
// g.ui.Do stays inside this Go body: Settle's barrier is decodes.Wait,
// which only covers completions the pool spawned.
func (g *Overview) hashRemaining() int {
	gen := g.host.Generation()
	g.wipeHashesIfStale()

	type hashJob struct {
		file   fyne.URI
		key    string
		thumb  image.Image
		hashed bool
		sized  bool
	}
	var jobs []hashJob
	for i := 0; i < g.host.FileCount(); i++ {
		u := g.host.FileAt(i)
		_, hashed := g.hashOf(u)
		_, sized := g.pixelCountOf(u)
		if hashed && sized {
			continue
		}
		if g.hashFailedOf(u) {
			continue
		}
		key := u.String()
		if _, loaded := g.hashing.LoadOrStore(key, true); loaded {
			continue
		}
		job := hashJob{file: u, key: key, hashed: hashed, sized: sized}
		if thumb, ok := g.thumbs.Get(key); ok {
			job.thumb = thumb
		}
		jobs = append(jobs, job)
	}
	n := len(jobs)
	if n == 0 {
		return 0
	}
	g.hashJobs.Add(int32(n))
	for _, j := range jobs {
		file, key, cached, hashed, sized := j.file, j.key, j.thumb, j.hashed, j.sized
		g.decodes.Go(context.Background(), func(acquired bool) {
			defer func() {
				g.hashing.Delete(key)
				remaining := g.hashJobs.Add(-1)
				if !g.shouldScheduleHideApply(remaining) {
					return
				}
				snap := g.dupes.Compute()
				g.ui.Do(func() {
					defer g.hideApply.Store(false)
					if gen != g.host.Generation() {
						return
					}
					if g.browseHost >= 0 {
						if remaining == 0 {
							g.finishBrowse()
							g.fireDupeState()
						}
						return
					}
					if g.HideDuplicates() {
						keepHost := g.fileIndex(g.highlight)
						if g.duplicateDistance() != snap.Dist {
							snap = g.dupes.Compute()
						}
						g.dupes.Install(snap)
						g.applyVisibleFilter(false, keepHost)
						if !g.InspectingDuplicates() {
							g.jumpIfHiddenExtra()
						}
					}
					if remaining == 0 {
						g.fireDupeState()
					}
				})
			}()
			if !acquired || gen != g.host.Generation() {
				return
			}
			thumb := cached
			var native image.Rectangle
			haveNative := false
			if thumb == nil {
				var err error
				thumb, native, err = imaging.LoadThumbnailAndBounds(file)
				if err != nil || thumb == nil {
					g.rememberHashFail(file)
					return
				}
				haveNative = true
				if !g.ThumbCacheFull() {
					g.thumbs.AddIfFits(file.String(), thumb)
				}
			}
			if !hashed {
				g.rememberHash(file, thumb)
			}
			if !sized {
				if haveNative {
					g.rememberNative(file, native)
				} else {
					_, b, err := imaging.ReadAndProbe(context.Background(), file)
					if err != nil {
						// Known-zero so hide/browse stop re-queueing an
						// unprobeable file (same trade-off as hashFailed).
						b = image.Rectangle{}
					}
					g.rememberNative(file, b)
				}
			}
		})
	}
	return n
}

func (g *Overview) shouldScheduleHideApply(remaining int32) bool {
	if remaining == 0 {
		g.hideApply.Store(true)
		return true
	}
	now := time.Now().UnixMilli()
	last := g.hideApplyAt.Load()
	if last != 0 && now-last < hideApplyMinInterval.Milliseconds() {
		return false
	}
	if !g.hideApply.CompareAndSwap(false, true) {
		return false
	}
	g.hideApplyAt.Store(now)
	return true
}
