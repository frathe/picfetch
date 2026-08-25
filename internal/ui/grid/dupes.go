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

func (g *Overview) rememberHash(u fyne.URI, img image.Image) {
	if u == nil || img == nil {
		return
	}
	h := imaging.DifferenceHash(img)
	gen := g.host.Generation()

	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	if g.hashGen != gen {
		g.hashes = make(map[string]uint64)
		g.hashFailed = make(map[string]struct{})
		g.hashGen = gen
	}
	if g.hashes == nil {
		g.hashes = make(map[string]uint64)
	}
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
	if g.hashGen != gen {
		g.hashes = make(map[string]uint64)
		g.hashFailed = make(map[string]struct{})
		g.hashGen = gen
	}
	if g.hashFailed == nil {
		g.hashFailed = make(map[string]struct{})
	}
	g.hashFailed[u.String()] = struct{}{}
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

func (g *Overview) wipeHashesIfStale() {
	gen := g.host.Generation()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	if g.hashGen != gen {
		g.hashes = make(map[string]uint64)
		g.hashFailed = make(map[string]struct{})
		g.hashGen = gen
	}
}

func (g *Overview) clearHashes() {
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.hashes = make(map[string]uint64)
	g.hashFailed = make(map[string]struct{})
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
		id := g.displayIndexOf(g.browseHost)
		g.setHighlight(id)
		g.wrap.ScrollTo(id)
	}
	g.fireDupeState()
}

// groupMembers returns host indices in the same duplicate group as
// hostIndex, in host-index order, or nil when the group has fewer than two
// members.
func (g *Overview) groupMembers(hostIndex int) []int {
	g.rebuildGroups()
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

func (g *Overview) jumpIfHiddenExtra() {
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
	if g.dupeDist == n {
		return
	}
	g.dupeDist = n
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

// RepresentativeOf is the lowest host index in hostIndex's duplicate group,
// or hostIndex itself when it is unique, unhashed, or out of range.
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
	g.groupSizes, g.groupReps = g.computeDuplicateGroups()
}

func (g *Overview) computeDuplicateGroups() (sizes, reps []int) {
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
	dist := g.dupeDist
	for i := range n {
		u := g.host.FileAt(i)
		if h, ok := g.hashes[u.String()]; ok {
			idx = append(idx, i)
			hs = append(hs, h)
			hashed[i] = true
		}
	}
	g.hashMu.Unlock()

	groups := imaging.DuplicateGroups(hs, dist)
	for _, grp := range groups {
		rep := idx[grp[0]]
		for _, gi := range grp {
			if idx[gi] < rep {
				rep = idx[gi]
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
	return sizes, reps
}

func (g *Overview) displayIndexOf(hostIdx int) int {
	if g.matches == nil {
		return hostIdx
	}
	for i, h := range g.matches {
		if h == hostIdx {
			return i
		}
	}
	return 0
}

// hashRemaining hashes every file that does not already have a dHash.
// Cache hits join the thumbnail pool the same way misses do — dHashing
// them on the D-key goroutine froze the UI for any folder that already
// fit in the thumb cache. Jobs have no per-cell Claim so Settle still
// waits, and they do not Add to a full thumbnail cache.
//
// DuplicateGroups runs on the worker before g.ui.Do. The callback only
// installs that snapshot and filters; hideApply stays set until it
// returns so an idle UI cannot re-arm mid-apply. Mid-window applies are
// also floored by hideApplyMinInterval; the last job always applies.
// Browse still waits for the last job (finishBrowse) so a partial group
// is never shown. g.ui.Do stays inside this Go body: Settle's barrier is
// decodes.Wait, which only covers completions the pool spawned.
func (g *Overview) hashRemaining() int {
	gen := g.host.Generation()
	g.wipeHashesIfStale()

	type hashJob struct {
		file  fyne.URI
		key   string
		thumb image.Image
	}
	var jobs []hashJob
	for i := 0; i < g.host.FileCount(); i++ {
		u := g.host.FileAt(i)
		if _, ok := g.hashOf(u); ok {
			continue
		}
		if g.hashFailedOf(u) {
			continue
		}
		key := u.String()
		if _, loaded := g.hashing.LoadOrStore(key, true); loaded {
			continue
		}
		job := hashJob{file: u, key: key}
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
		file, key, cached := j.file, j.key, j.thumb
		g.decodes.Go(context.Background(), func(acquired bool) {
			defer func() {
				g.hashing.Delete(key)
				remaining := g.hashJobs.Add(-1)
				if !g.shouldScheduleHideApply(remaining) {
					return
				}
				sizes, reps := g.computeDuplicateGroups()
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
						g.groupSizes, g.groupReps = sizes, reps
						g.applyVisibleFilter(false, keepHost)
						g.jumpIfHiddenExtra()
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
			if thumb == nil {
				var err error
				thumb, err = imaging.LoadThumbnail(file)
				if err != nil || thumb == nil {
					g.rememberHashFail(file)
					return
				}
				if !g.ThumbCacheFull() {
					g.thumbs.AddIfFits(file.String(), thumb)
				}
			}
			g.rememberHash(file, thumb)
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
