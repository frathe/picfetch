package grid

import (
	"context"

	"github.com/frathe/picfetch/internal/dupes"
)

// groupWork is UI-owned admission. One pool job runs at a time; changed
// requests cancel it and its completion admits only the latest inputs.
type groupWork struct {
	// compute is an instance override for held-worker tests; nil uses the model.
	compute   func(context.Context) (dupes.Groups, error)
	pending   bool
	key       dupes.GroupingKey
	revision  uint64
	cancel    context.CancelFunc
	retarget  bool
	readyKey  dupes.GroupingKey
	ready     bool
	readyIdle bool
}

// rebuildGroups returns true only when an accepted snapshot is current.
// Changed input never performs grouping on the calling UI goroutine.
func (g *Overview) rebuildGroups() bool {
	if _, current := g.dupes.CurrentGroups(); current {
		return true
	}
	ctx := g.workContext()
	if ctx.Err() != nil {
		return false
	}
	key := g.dupes.GroupingKey()
	if g.grouping.pending {
		if g.grouping.key != key || g.grouping.revision != g.work.revision {
			g.grouping.cancel()
		}
		return false
	}
	ctx, cancel := context.WithCancel(ctx)
	revision := g.work.revision
	g.grouping.pending, g.grouping.key, g.grouping.revision, g.grouping.cancel = true, key, revision, cancel
	compute := g.grouping.compute
	if compute == nil {
		compute = g.dupes.ComputeContext
	}
	g.decodes.Go(ctx, func(acquired bool) {
		var snapshot dupes.Groups
		var err error
		if acquired {
			snapshot, err = compute(ctx)
		}
		// Submission belongs to the pool job, so Settle covers both this
		// completion and a replacement it admits while draining.
		g.ui.Do(func() {
			defer cancel()
			g.grouping.pending = false
			g.grouping.cancel = nil
			if g.work.ctx.Err() != nil {
				return
			}
			if acquired && err == nil && ctx.Err() == nil && revision == g.work.revision && g.dupes.Install(snapshot) {
				g.groupsReady(snapshot)
			} else {
				g.rebuildGroups()
			}
		})
	})
	return false
}

func (g *Overview) groupsReady(snapshot dupes.Groups) {
	key := snapshot.Key()
	idle := g.hashes.hashJobs.Load() == 0
	if g.grouping.ready && g.grouping.readyKey == key && (g.grouping.readyIdle || !idle) {
		return
	}
	g.grouping.ready, g.grouping.readyKey, g.grouping.readyIdle = true, key, idle
	if g.grouping.retarget {
		g.grouping.retarget = false
		g.retargetInspect()
	}
	if g.browseHost >= 0 && g.hashes.hashJobs.Load() == 0 {
		g.finishBrowse()
		return
	}
	keepHost := g.fileIndex(g.highlight)
	g.applyVisibleFilter(false, keepHost)
	if g.dupes.HideDuplicates() && g.browseHost < 0 && !g.dupes.Inspecting() {
		g.dupes.Notify()
	}
	if idle {
		g.fireDupeState()
	}
}
