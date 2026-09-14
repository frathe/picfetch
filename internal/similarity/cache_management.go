package similarity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type CacheSize struct {
	Bytes      uint64
	Records    int
	Incomplete bool
}
type CacheUsage struct {
	General, Favorite CacheSize
	Incomplete        bool
}
type CacheProgress struct {
	Phase   string
	Records int
	Bytes   uint64
}
type CacheCleanMode int

const (
	ClearAll CacheCleanMode = iota
	RemoveStale
)

type CacheCleanRequest struct {
	Roots CacheRoots
	Mode  CacheCleanMode
}
type CacheRetuneRequest struct {
	Roots         CacheRoots
	LimitBytes    uint64
	ReserveBytes  uint64
	RetireWriters bool
}
type CacheReport struct {
	Before, Remaining                              CacheUsage
	RemovedBytes                                   uint64
	RemovedRecords, Skipped, Unavailable, Failures int
	Canceled                                       bool
	// AppliedLimit is nonzero after successful general maintenance, even when
	// Favorite inspection accompanies that result with an error.
	AppliedLimit uint64
}
type CacheMaintenanceProvider interface {
	Inspect(context.Context, CacheRoots, func(CacheProgress)) (CacheUsage, error)
	Clean(context.Context, CacheCleanRequest, func(CacheProgress)) (CacheReport, error)
	Retune(context.Context, CacheRetuneRequest, func(CacheProgress)) (CacheReport, error)
}

// CacheManager coordinates managed record changes with the owning UI producers.
type CacheManager struct {
	Quiesce func(context.Context, CacheRoots) error
}

func (CacheManager) Inspect(ctx context.Context, roots CacheRoots, progress func(CacheProgress)) (CacheUsage, error) {
	inventory := &analysisInventory{}
	defer inventory.close()
	err := inventory.scan(ctx, roots, progress)
	return inventory.usage, err
}
func (m CacheManager) Clean(ctx context.Context, request CacheCleanRequest, progress func(CacheProgress)) (CacheReport, error) {
	if request.Mode != ClearAll && request.Mode != RemoveStale {
		return CacheReport{}, fmt.Errorf("unknown cache cleanup mode")
	}
	return m.maintain(ctx, request.Roots, &request.Mode, nil, true, progress)
}
func (m CacheManager) Retune(ctx context.Context, request CacheRetuneRequest, progress func(CacheProgress)) (CacheReport, error) {
	if request.LimitBytes == 0 {
		return CacheReport{}, fmt.Errorf("analysis cache limit must be positive")
	}
	return m.maintain(ctx, request.Roots, nil, &request, request.RetireWriters, progress)
}
func (m CacheManager) maintain(ctx context.Context, roots CacheRoots, mode *CacheCleanMode, retune *CacheRetuneRequest, retire bool, progress func(CacheProgress)) (report CacheReport, err error) {
	observedUnderLease := false
	defer func() {
		report.Canceled = report.Canceled || ctx.Err() != nil
		if report.Canceled && !observedUnderLease {
			report.Remaining.Incomplete = true
		}
	}()
	before, inspectErr := m.Inspect(ctx, roots, progress)
	report.Before, report.Remaining = before, before
	// Inspection deliberately returns observed usage alongside traversal errors.
	report.Before.Incomplete = report.Before.Incomplete || inspectErr != nil
	report.Remaining = report.Before
	var limit uint64
	if retune != nil {
		limit = retune.LimitBytes - min(retune.ReserveBytes, retune.LimitBytes)
		if !retire && !report.Before.General.Incomplete && ctx.Err() == nil && report.Before.General.Bytes <= limit {
			report.AppliedLimit = retune.LimitBytes
			return report, inspectErr
		}
	}
	if ctx.Err() != nil {
		report.Canceled = true
		return report, ctx.Err()
	}
	// Empty stores still need shared locks and epochs: another process may
	// create its first record while this maintenance pass is active.
	lease, err := openCacheLease(ctx, roots, true)
	if err != nil {
		return report, errors.Join(inspectErr, err)
	}
	defer lease.close()
	release, err := lease.lock(ctx)
	if err != nil {
		return report, errors.Join(inspectErr, err)
	}
	defer release()
	if err := lease.invalidate(); err != nil {
		return report, err
	}
	// Epoch invalidation closes admission before producer cancellation. Writers
	// waiting for these locks observe cancellation without a UI-thread join.
	if m.Quiesce != nil {
		if err := m.Quiesce(ctx, roots); err != nil {
			report.Canceled = ctx.Err() != nil
			return report, err
		}
	}
	inventory := &analysisInventory{}
	defer inventory.close()
	scanErr := inventory.scan(ctx, roots, nil)
	report.Before, report.Remaining = inventory.usage, inventory.usage
	if ctx.Err() != nil {
		return report, ctx.Err()
	}
	observedUnderLease = true
	if retune != nil {
		slices.SortFunc(inventory.records, func(a, b managedAnalysis) int {
			if a.info.ModTime().Before(b.info.ModTime()) {
				return -1
			}
			if a.info.ModTime().After(b.info.ModTime()) {
				return 1
			}
			return strings.Compare(a.name, b.name)
		})
	}
	remaining := inventory.usage.General.Bytes
	var failures []error
	if scanErr != nil {
		failures = append(failures, scanErr)
	}
	for _, record := range inventory.records {
		if ctx.Err() != nil {
			report.Canceled = true
			failures = append(failures, ctx.Err())
			break
		}
		remove := true
		if retune != nil {
			remove = record.general && remaining > limit
		}
		if mode != nil && *mode == RemoveStale {
			var unavailable bool
			remove, unavailable = record.stale()
			if unavailable {
				report.Unavailable++
			}
		}
		if !remove {
			report.Skipped++
			continue
		}
		// Favorite saves do not take the analysis lease. Recheck its captured
		// membership immediately before an unlink based on stale-list analysis.
		if mode != nil && *mode == RemoveStale && record.favorite != nil && !record.favorite.current() {
			report.Unavailable++
			report.Skipped++
			continue
		}
		if err := record.root.Remove(record.name); err != nil {
			report.Failures++
			failures = append(failures, err)
			continue
		}
		report.RemovedBytes += uint64(record.info.Size())
		if !record.temporary {
			report.RemovedRecords++
		}
		if record.general {
			remaining -= uint64(record.info.Size())
		}
		size := &report.Remaining.Favorite
		if record.general {
			size = &report.Remaining.General
		}
		size.Bytes -= uint64(record.info.Size())
		if !record.temporary {
			size.Records--
		}

		if progress != nil {
			progress(CacheProgress{Phase: "remove", Records: report.RemovedRecords, Bytes: report.RemovedBytes})
		}
	}
	// Retain the measured remainder if cancellation interrupts reconciliation.
	// A canceled request never starts another full inventory.
	if ctx.Err() == nil {
		after, afterErr := m.Inspect(ctx, roots, nil)
		if ctx.Err() == nil {
			report.Remaining = after
		}
		if afterErr != nil {
			failures = append(failures, afterErr)
		}
	} else {
		failures = append(failures, ctx.Err())
	}
	if retune != nil && ctx.Err() == nil && report.Failures == 0 && !report.Remaining.General.Incomplete && report.Remaining.General.Bytes <= limit {
		report.AppliedLimit = retune.LimitBytes
	}
	return report, errors.Join(failures...)
}
func (r managedAnalysis) stale() (bool, bool) {
	if r.favorite != nil && !r.favorite.current() {
		return false, true
	}
	if r.temporary {
		return true, false
	}
	if r.info.Size() > 1024*1024 {
		return true, false
	}
	file, err := r.root.Open(r.name)
	if err != nil {
		return false, true
	}
	item, err := decodeRepresentation(file)
	_ = file.Close()
	if err != nil || filepath.Base(analysisName(item.Path)) != r.name {
		return true, false
	}

	if r.favorite != nil && !r.favorite.members[filepath.Clean(item.Path)] {
		return true, false
	}
	info, err := os.Stat(item.Path)
	if err == nil {
		return info.Size() != item.Size || info.ModTime().UnixNano() != item.ModifiedNS, false
	}
	// Missing files are ambiguous without stored volume identity: an unmounted
	// volume can leave an accessible, empty mount-point directory behind.
	// Retain these records until the source returns, general LRU eviction, or full cleanup.
	return false, true
}
