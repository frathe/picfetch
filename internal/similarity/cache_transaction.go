package similarity

import (
	"context"
	"errors"
)

// cacheTransaction owns the observation and effect ledger for one maintenance
// pass. Every exit goes through finish, including exits during lock acquisition
// or inventory. Errors and committed effects are independent outcomes.
type cacheTransaction struct {
	report             CacheReport
	retune             *CacheRetuneRequest
	observedUnderLease bool
	completed          bool
}

func (t *cacheTransaction) observe(usage CacheUsage, underLease bool) {
	t.report.Before, t.report.Remaining = usage, usage
	t.observedUnderLease = underLease
}

func (t *cacheTransaction) removed(record managedAnalysis) {
	size := &t.report.Remaining.Favorite
	if record.general {
		size = &t.report.Remaining.General
	}
	bytes := uint64(record.info.Size())
	t.report.RemovedBytes += bytes
	size.Bytes -= bytes
	if !record.temporary {
		t.report.RemovedRecords++
		size.Records--
	}
}

func (t *cacheTransaction) reconcile(ctx context.Context, usage CacheUsage) {
	// An interrupted reconciliation must not overwrite the committed ledger
	// with a partial traversal of the remaining files.
	if ctx.Err() == nil {
		t.report.Remaining = usage
	}
}

func (t *cacheTransaction) limit() uint64 {
	return t.retune.LimitBytes - min(t.retune.ReserveBytes, t.retune.LimitBytes)
}

func (t *cacheTransaction) finish(ctx context.Context, err error) (CacheReport, error) {
	if cancelErr := ctx.Err(); cancelErr != nil {
		err = errors.Join(err, cancelErr)
	}
	t.report.Canceled = errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
	if t.report.Canceled && !t.observedUnderLease {
		t.report.Remaining.Incomplete = true
	}
	if t.retune != nil && t.completed && !t.report.Canceled && t.report.Failures == 0 &&
		!t.report.Remaining.General.Incomplete && t.report.Remaining.General.Bytes <= t.limit() {
		t.report.AppliedLimit = t.retune.LimitBytes
	}
	return t.report, err
}
