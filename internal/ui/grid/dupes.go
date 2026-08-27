package grid

import (
	"context"
	"image"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/imaging"
)

const maxDuplicateDistance = 32

// hideApplyMinInterval is how far apart hashRemaining may schedule hide
// applies while jobs are still running. The last job always applies.
// Without a floor, an idle UI drains one install and the next hash
// immediately queues another, so the event loop never sees input until
// the pool is empty.
const hideApplyMinInterval = 250 * time.Millisecond

// Hash storage lives on Overview (hashMu / hashes / hashGen), not in the
// thumbnail ByteCache: a hash is 8 bytes and must survive thumbnail eviction.

// ensureMapsLocked allocates hashes, hashFailed, and native if any of them
// is nil, leaving any existing entries untouched. Callers must hold hashMu.
func (g *Overview) ensureMapsLocked() {
	if g.hashes == nil {
		g.hashes = make(map[string]uint64)
	}
	if g.hashFailed == nil {
		g.hashFailed = make(map[string]struct{})
	}
	if g.native == nil {
		g.native = make(map[string]image.Point)
	}
}

func (g *Overview) ensureHashGenLocked(gen uint64) {
	if g.hashGen != gen {
		g.hashes = make(map[string]uint64)
		g.hashFailed = make(map[string]struct{})
		g.native = make(map[string]image.Point)
		g.hashGen = gen
	}
	g.ensureMapsLocked()
}

// adoptHashGen records the host's current generation without dropping
// URI-keyed hashes, hashFailed, or native. Incremental shrink
// (RemoveFiles → FilesChanged) is not a new drop: surviving files keep
// their hashes so hide-duplicates grouping and inspect retarget still
// work. Orphan keys for deleted URIs linger until the next full-set
// change, which is harmless. Do not call ensureHashGenLocked: that
// wipes on mismatch.
func (g *Overview) adoptHashGen() {
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.hashGen = g.host.Generation()
	g.ensureMapsLocked()
}

func (g *Overview) rememberHash(u fyne.URI, img image.Image) {
	if u == nil || img == nil {
		return
	}
	h := imaging.DifferenceHash(img)
	gen := g.host.Generation()

	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.ensureHashGenLocked(gen)
	key := u.String()
	g.hashes[key] = h
	delete(g.hashFailed, key)
}

func (g *Overview) rememberHashFail(u fyne.URI) {
	if u == nil {
		return
	}
	gen := g.host.Generation()

	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.ensureHashGenLocked(gen)
	g.hashFailed[u.String()] = struct{}{}
}

func (g *Overview) rememberNative(u fyne.URI, native image.Rectangle) {
	if u == nil {
		return
	}
	sz := image.Pt(max(native.Dx(), 0), max(native.Dy(), 0))
	gen := g.host.Generation()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.ensureHashGenLocked(gen)
	g.native[u.String()] = sz
}

func (g *Overview) hashOf(u fyne.URI) (uint64, bool) {
	if u == nil {
		return 0, false
	}
	g.wipeHashesIfStale()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	h, ok := g.hashes[u.String()]
	return h, ok
}

func (g *Overview) hashFailedOf(u fyne.URI) bool {
	if u == nil {
		return false
	}
	g.wipeHashesIfStale()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	_, ok := g.hashFailed[u.String()]
	return ok
}

func (g *Overview) nativeSizeOf(u fyne.URI) (image.Point, bool) {
	if u == nil {
		return image.Point{}, false
	}
	g.wipeHashesIfStale()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	sz, ok := g.native[u.String()]
	return sz, ok
}

func (g *Overview) pixelCountOf(u fyne.URI) (int, bool) {
	sz, ok := g.nativeSizeOf(u)
	if !ok {
		return 0, false
	}
	return sz.X * sz.Y, true
}

// NativeSize is the EXIF-oriented pixel size of the file at hostIndex.
// ok is false when the index is out of range or no probe has been stored,
// or when a stored size has a non-positive edge (failed/empty probe).
func (g *Overview) NativeSize(hostIndex int) (w, h int, ok bool) {
	if hostIndex < 0 || hostIndex >= g.host.FileCount() {
		return 0, 0, false
	}
	sz, ok := g.nativeSizeOf(g.host.FileAt(hostIndex))
	if !ok || sz.X <= 0 || sz.Y <= 0 {
		return 0, 0, false
	}
	return sz.X, sz.Y, true
}

func (g *Overview) wipeHashesIfStale() {
	gen := g.host.Generation()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.ensureHashGenLocked(gen)
}

func (g *Overview) clearHashes() {
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.hashes = make(map[string]uint64)
	g.hashFailed = make(map[string]struct{})
	g.native = make(map[string]image.Point)
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
	return g.hideDupes
}

// SetHideDuplicates turns extra-hiding on or off. Turning it on hashes any
// files that have not been hashed yet (cache hits and misses both join the
// decode pool so D never dHashes on the key-handler goroutine) and
// jumps the host to the group's representative if the current file is an
// extra. Close does not clear this flag: the viewer still skips extras
// after the grid is dismissed.
func (g *Overview) SetHideDuplicates(on bool) {
	if g.hideDupes == on {
		return
	}
	g.hideDupes = on
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
// members. It reads the installed groupSizes/groupReps snapshot (same as
// IsHiddenExtra / RepresentativeOf) and does not rebuild.
func (g *Overview) groupMembers(hostIndex int) []int {
	if g.groupSize(hostIndex) < 2 {
		return nil
	}
	rep := g.RepresentativeOf(hostIndex)
	var members []int
	for i := range g.host.FileCount() {
		if g.RepresentativeOf(i) == rep {
			members = append(members, i)
		}
	}
	return members
}

// BeginInspect starts an inspect session on hostIndex so a hidden extra
// can stay on screen. Out-of-range or missing files clear inspect instead.
func (g *Overview) BeginInspect(hostIndex int) {
	if hostIndex < 0 || hostIndex >= g.host.FileCount() {
		g.inspectKey = ""
		return
	}
	u := g.host.FileAt(hostIndex)
	if u == nil {
		g.inspectKey = ""
		return
	}
	g.inspectKey = u.String()
}

// ClearInspect ends the inspect session.
func (g *Overview) ClearInspect() {
	g.inspectKey = ""
}

// InspectingDuplicates reports whether an inspect session is active.
func (g *Overview) InspectingDuplicates() bool {
	return g.inspectKey != ""
}

func (g *Overview) inspectSource() int {
	if g.inspectKey == "" {
		return -1
	}
	for i := 0; i < g.host.FileCount(); i++ {
		if g.host.FileAt(i).String() == g.inspectKey {
			return i
		}
	}
	return -1
}

// InspectMembers returns host indices of the inspected file's duplicate
// group in host-index order, or nil when inspect is off.
func (g *Overview) InspectMembers() []int {
	src := g.inspectSource()
	if src < 0 {
		return nil
	}
	return g.groupMembers(src)
}

func (g *Overview) jumpIfHiddenExtra() {
	if g.InspectingDuplicates() {
		return
	}
	if i := g.host.CurrentIndex(); g.IsHiddenExtra(i) {
		g.host.ShowImage(g.RepresentativeOf(i))
	}
}

// SetDuplicateDistance clamps n to 0–32 and rebuilds groups. Live: if
// browsing, the group is re-checked and browse exits when it drops below
// two members. If hide is on and not browsing, extras are recomputed
// immediately and the host jumps if the current file became an extra.
func (g *Overview) SetDuplicateDistance(n int) {
	if n < 0 {
		n = 0
	}
	if n > maxDuplicateDistance {
		n = maxDuplicateDistance
	}
	g.hashMu.Lock()
	same := g.dupeDist == n
	if !same {
		g.dupeDist = n
	}
	g.hashMu.Unlock()
	if same {
		return
	}
	if g.browseHost >= 0 {
		g.finishBrowse()
	} else if g.hideDupes {
		g.applyFilter()
		g.jumpIfHiddenExtra()
	} else {
		g.rebuildGroups()
	}
	g.fireDupeState()
}

func (g *Overview) duplicateDistance() int {
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	return g.dupeDist
}

// IsHiddenExtra reports whether hostIndex is a non-representative member of
// a duplicate group while hide is on. Unhashed files are never extras.
func (g *Overview) IsHiddenExtra(hostIndex int) bool {
	if !g.hideDupes || hostIndex < 0 || hostIndex >= len(g.groupSizes) {
		return false
	}
	if g.groupSizes[hostIndex] < 2 {
		return false
	}
	return hostIndex != g.groupReps[hostIndex]
}

// RepresentativeOf is the highest native pixel count in the group, lowest
// host index on a tie; itself when unique, unhashed, or out of range.
func (g *Overview) RepresentativeOf(hostIndex int) int {
	if hostIndex < 0 || hostIndex >= len(g.groupReps) {
		return hostIndex
	}
	return g.groupReps[hostIndex]
}

// groupSize is 0 if hostIndex is unhashed, 1 if it is a unique hashed file,
// and ≥2 if it belongs to a duplicate group.
func (g *Overview) groupSize(hostIndex int) int {
	if hostIndex < 0 || hostIndex >= len(g.groupSizes) {
		return 0
	}
	return g.groupSizes[hostIndex]
}

func (g *Overview) rebuildGroups() {
	g.groupSizes, g.groupReps, _ = g.computeDuplicateGroups()
}

func (g *Overview) computeDuplicateGroups() (sizes, reps []int, dist int) {
	g.groupComputes.Add(1)
	n := g.host.FileCount()
	sizes = make([]int, n)
	reps = make([]int, n)
	for i := range n {
		reps[i] = i
	}
	g.wipeHashesIfStale()

	g.hashMu.Lock()
	idx := make([]int, 0, n)
	hs := make([]uint64, 0, n)
	hashed := make([]bool, n)
	px := make([]int, n)
	dist = g.dupeDist
	for i := range n {
		u := g.host.FileAt(i)
		if h, ok := g.hashes[u.String()]; ok {
			idx = append(idx, i)
			hs = append(hs, h)
			hashed[i] = true
		}
		if sz, ok := g.native[u.String()]; ok {
			px[i] = sz.X * sz.Y
		}
	}
	g.hashMu.Unlock()

	groups := imaging.DuplicateGroups(hs, dist)
	for _, grp := range groups {
		rep := idx[grp[0]]
		repPx := px[rep]
		for _, gi := range grp {
			hi := idx[gi]
			if px[hi] > repPx || (px[hi] == repPx && hi < rep) {
				rep, repPx = hi, px[hi]
			}
		}
		for _, gi := range grp {
			hi := idx[gi]
			sizes[hi] = len(grp)
			reps[hi] = rep
		}
	}
	for i := range n {
		if hashed[i] && sizes[i] == 0 {
			sizes[i] = 1
		}
	}
	return sizes, reps, dist
}

// hashRemaining hashes every file that does not already have a dHash,
// and records native pixel counts for files that have a hash but no
// size. Cache hits join the thumbnail pool the same way misses do —
// dHashing them on the D-key goroutine froze the UI for any folder that
// already fit in the thumb cache. Jobs have no per-cell Claim so Settle
// still waits, and they do not Add to a full thumbnail cache.
//
// DuplicateGroups runs on the worker before g.ui.Do. The callback only
// installs that snapshot and filters, unless dupeDist changed since the
// snapshot (settings slider while hashing): then it recomputes so the
// install cannot undo the live regroup. hideApply stays set until the
// callback returns so an idle UI cannot re-arm mid-apply. Mid-window
// applies are also floored by hideApplyMinInterval; the last job always
// applies. Browse still waits for the last job (finishBrowse) so a
// partial group is never shown. g.ui.Do stays inside this Go body:
// Settle's barrier is decodes.Wait, which only covers completions the
// pool spawned.
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
				sizes, reps, snapDist := g.computeDuplicateGroups()
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
					if g.hideDupes {
						keepHost := g.fileIndex(g.highlight)
						if g.duplicateDistance() != snapDist {
							sizes, reps, _ = g.computeDuplicateGroups()
						}
						g.groupSizes, g.groupReps = sizes, reps
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
