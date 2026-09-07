package grid

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/imaging"
)

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
func rememberHash(facts dupes.FactWriter, u fyne.URI, img image.Image) bool {
	if u == nil || img == nil {
		return false
	}
	return facts.PutHash(u.String(), imaging.DifferenceHash(img))
}

// rememberNative records u's native pixel size. The conversion from a
// probe's rectangle to a plain size happens here because the model stores
// sizes, not rectangles with an origin; it applies its own clamp for a
// negative edge.
func rememberNative(facts dupes.FactWriter, u fyne.URI, native image.Rectangle) bool {
	if u == nil {
		return false
	}
	return facts.PutNativeSize(u.String(), image.Pt(native.Dx(), native.Dy()))
}

func (g *Overview) hashOf(u fyne.URI) (uint64, bool) {
	if u == nil {
		return 0, false
	}

	return g.dupes.Hash(u.String())
}

func (g *Overview) pixelCountOf(u fyne.URI) (int, bool) {
	if u == nil {
		return 0, false
	}

	return g.dupes.PixelCount(u.String())
}

func (g *Overview) wipeHashesIfStale() {
	g.dupes.WipeIfStale()
}

func (g *Overview) clearHashes() {
	g.dupes.Clear()
}

// SetOnDupeStateChanged registers f to run after hide, browse, completed
// grouping, or duplicate-distance changes. The field is read at fire
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
	return g.dupes.GroupSize(src)
}

// SetHideDuplicates turns extra-hiding on or off. Turning it on hashes any
// files that have not been hashed yet (cache hits and misses both join the
// decode pool so D never dHashes on the key-handler goroutine) and
// jumps the host to the group's representative if the current file is an
// extra. Close does not clear this flag: the viewer still skips extras
// after the grid is dismissed.
//
// This is the entry point for callers that hold only the grid: the
// overview's own D key and its escape ladder (nav.go). The app does not
// come through here - it owns the model, so it moves the flag there and
// calls HideDuplicatesChanged for the grid half (viewer.pushHideDuplicates
// in internal/ui/visibility.go). Same front-half/back-half split the
// duplicate distance uses, except that one has no grid-side front half
// left at all - nothing inside the grid changes the threshold, so only
// DuplicateDistanceChanged survives down here.
func (g *Overview) SetHideDuplicates(on bool) {
	if !g.dupes.SetHideDuplicates(on) {
		return
	}
	g.HideDuplicatesChanged(on)
}

// HideDuplicatesChanged re-applies the grid's own view of a hide flag the
// model has already accepted: the second half of SetHideDuplicates, split
// out so the app can set the flag on the model it owns and still get the
// sequence the grid always ran - hash first when hide just turned on, then
// re-filter, and only then the model's observers (the jump off a
// now-hidden extra).
//
// on is a parameter rather than a re-read of the model because the two
// directions do different work: turning hide on has to hash whatever is
// not hashed yet, turning it off only has to re-filter. A parameterless
// OnChange observer cannot express that difference, which is why this is
// an explicit call from the app and not a subscription.
//
// Only the caller that actually moved the flag should call this;
// dupes.Model.SetHideDuplicates reports whether it did.
func (g *Overview) HideDuplicatesChanged(on bool) {
	if on {
		_ = g.hashRemaining()
	}
	g.applyFilter()
	if _, current := g.dupes.CurrentGroups(); on && current {
		// The model's observers, not a call of the grid's own: the jump
		// off a now-hidden extra runs ShowImage, which belongs to the app
		// (internal/ui's jumpIfHiddenExtra). Fired after applyFilter, so
		// the observer sees the group snapshot the filter pass installed.
		g.dupes.Notify()
	}
	g.fireDupeState()
}

// BrowsingDuplicates reports whether the grid is showing a single duplicate
// group.
func (g *Overview) BrowsingDuplicates() bool {
	return g.browseHost >= 0
}

// BrowseReady reports a completed group that can be presented by the host.
func (g *Overview) BrowseReady() bool {
	snapshot, current := g.dupes.CurrentGroups()
	return g.browseHost >= 0 && g.hashes.hashJobs.Load() == 0 && current && snapshot.Size(g.browseHost) >= 2
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
	// Capture the browse source before admitting its background hash pass.
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
	// usual rebuild site, and the group-size check below must see the
	// rebuilt sizes first.
	if !g.rebuildGroups() {
		return
	}
	if g.dupes.GroupSize(g.browseHost) < 2 {
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
// members. It reads the model's installed group snapshot (same as the
// model's IsHiddenExtra / RepresentativeOf) and does not rebuild.
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

func (g *Overview) inspectSource() int {
	return g.dupes.InspectSource()
}

// DuplicateDistanceChanged re-applies the grid's own view of a Hamming
// threshold the model has already accepted - the model is what clamps it
// to 0–32 - and rebuilds groups. Live: if browsing, the group is
// re-checked and browse exits when it drops below two members. If hide is
// on and not browsing, extras are recomputed in the pool and the host
// jumps if the current file became an extra.
//
// There is no grid-side setter to pair with it any more: the app owns the
// model and sets the threshold on it directly (settingswin drives the
// slider through viewer.SetDuplicateDistance), then calls this for the
// grid half, so the live regroup still happens in the order the grid
// always did it - the grid re-filters first, and only then do the model's
// observers (the jump off a now-hidden extra) run.
//
// Only the caller that actually moved the value should call this;
// dupes.Model.SetDistance reports whether it did.
func (g *Overview) DuplicateDistanceChanged() {
	// Startup restores the threshold before any files or test UI queue exist.
	if g.host.FileCount() == 0 {
		g.fireDupeState()
		return
	}
	g.resumeWork()
	if g.browseHost >= 0 {
		g.finishBrowse()
	} else {
		g.applyFilter()
	}
	g.fireDupeState()
}

func (g *Overview) duplicateDistance() int {
	return g.dupes.Distance()
}

// hashRemaining fills missing facts. Its throttled UI notification requests
// grouping through the same cancellable owner as search and distance changes.
func (g *Overview) hashRemaining() int {
	ctx := g.resumeWork()
	return g.hashes.Run(ctx, g.hashFactsReady)
}

func (g *Overview) hashFactsReady(remaining int32, gen uint64) {
	if gen != g.host.Generation() {
		return
	}
	if !g.dupes.HideDuplicates() && g.browseHost < 0 {
		if remaining == 0 {
			g.fireDupeState()
		}
		return
	}
	if g.browseHost >= 0 && remaining != 0 {
		return
	}
	if g.rebuildGroups() {
		if snapshot, current := g.dupes.CurrentGroups(); current {
			g.groupsReady(snapshot)
		}
	}
}
