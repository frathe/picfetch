package analysiscache

import (
	"context"

	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/similarity"
)

type maintenanceIntent uint8

const (
	inspectUsage maintenanceIntent = iota
	cleanRecords
	applyLimit
	evictRecords
	retirePolicy
)

func (i maintenanceIntent) viewBound() bool {
	return i == inspectUsage || i == cleanRecords || i == applyLimit
}

// An operation captures its inputs before a worker starts. Intent determines
// admission, provider dispatch, view lifetime and which effects may be applied.
type operation struct {
	roots    similarity.CacheRoots
	mode     similarity.CacheCleanMode
	limitMiB int
	reserve  uint64
	intent   maintenanceIntent
	retire   bool
	enabled  bool
}

func (f *Feature) submit(op operation) {
	if f.stopped || op.intent.viewBound() && !f.open {
		return
	}
	var barriers []<-chan struct{}
	switch op.intent {
	case inspectUsage:
		// A refresh can replace a refresh, never a mutation or retirement.
		if f.current != nil && f.current.intent != inspectUsage {
			f.pendingInspect = true
			return
		}
		f.pendingInspect = false
	case cleanRecords:
		if op.mode != similarity.ClearAll && op.mode != similarity.RemoveStale {
			return
		}
	case applyLimit:
		if op.limitMiB <= 0 || uint64(op.limitMiB) > ^uint64(0)/mebibyte {
			return
		}
		op.retire = op.limitMiB < f.limitMiB
	case evictRecords:
		if !f.enabled {
			return
		}
		if f.Busy() {
			f.pendingRoom = true
			f.pendingReserve = max(f.pendingReserve, op.reserve)
			return
		}
		op.limitMiB = f.limitMiB
	case retirePolicy:
		if f.enabled == op.enabled {
			return
		}
		f.enabled = op.enabled
		f.pendingRoom, f.pendingInspect, f.pendingReserve = false, false, 0
		barriers = f.host.Quiesce()
		f.host.ApplyPolicy(f.enabled, f.limitMiB)
	}
	op.roots = f.options.Roots
	f.start(op, barriers...)
}

func (op operation) run(ctx context.Context, provider similarity.CacheMaintenanceProvider, progress func(similarity.CacheProgress)) result {
	switch op.intent {
	case inspectUsage:
		usage, err := provider.Inspect(ctx, op.roots, progress)
		return result{usage: usage, err: err}
	case cleanRecords:
		report, err := provider.Clean(ctx, similarity.CacheCleanRequest{Roots: op.roots, Mode: op.mode}, progress)
		return result{report: report, err: err}
	case applyLimit, evictRecords:
		report, err := provider.Retune(ctx, similarity.CacheRetuneRequest{
			Roots: op.roots, LimitBytes: uint64(op.limitMiB) * mebibyte,
			ReserveBytes: op.reserve, RetireWriters: op.retire,
		}, progress)
		return result{report: report, err: err}
	default:
		// Policy was committed on UI; joining its predecessors is the work.
		return result{}
	}
}

func (f *Feature) apply(op operation, r result) {
	switch op.intent {
	case inspectUsage:
		f.showUsage(r.usage)
		if r.err != nil {
			f.status.SetText(lang.L("Usage is incomplete."))
		}
	case retirePolicy:
		f.Inspect()
	case applyLimit, evictRecords, cleanRecords:
		if op.intent == applyLimit && r.report.AppliedLimit == uint64(op.limitMiB)*mebibyte && !r.report.Canceled {
			f.limitMiB = op.limitMiB
			f.host.ApplyPolicy(f.enabled, f.limitMiB)
		}
		if f.open {
			f.syncing = true
			f.loose.SetChecked(f.enabled)
			f.syncing = false
			f.showReport(r)
		}
	}
}

func (f *Feature) startPendingRoom() {
	if !f.pendingRoom || f.current != nil || f.hasWorkers() {
		return
	}
	reserve := f.pendingReserve
	f.pendingRoom, f.pendingReserve = false, 0
	f.MakeRoom(reserve)
}

func (f *Feature) startPendingInspect() {
	if !f.pendingInspect || f.current != nil || f.hasWorkers() {
		return
	}
	f.pendingInspect = false
	f.Inspect()
}
