package analysiscache

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/similarity"
)

type featureQueue struct{}

func (featureQueue) Do(fn func()) { fyne.Do(fn) }
func (featureQueue) Drain() bool  { return false }

type work struct {
	cancel    context.CancelFunc
	done      chan struct{}
	view      uint64
	intent    maintenanceIntent
	delivered bool
}
type result struct {
	usage      similarity.CacheUsage
	report     similarity.CacheReport
	err        error
	incomplete bool
}

func (f *Feature) start(op operation, after ...<-chan struct{}) {
	if f.stopped {
		return
	}
	f.pruneWorkers()
	if f.current != nil {
		f.current.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &work{cancel: cancel, done: make(chan struct{}), view: f.view, intent: op.intent}
	var predecessors []<-chan struct{}
	for _, barrier := range after {
		if barrier != nil {
			predecessors = append(predecessors, barrier)
		}
	}
	for _, prior := range f.workers {
		predecessors = append(predecessors, prior.done)
	}
	f.workers = append(f.workers, w)
	f.current = w
	if f.open {
		f.status.SetText(lang.L("Working..."))
		f.progress.SetText("")
	}
	queue, notice, host := f.ui, f.notice, f.host
	notify := func() {
		select {
		case notice <- struct{}{}:
		default:
		}
	}
	provider := f.options.Provider
	if provider == nil {
		provider = similarity.CacheManager{Quiesce: func(ctx context.Context, _ similarity.CacheRoots) error {
			ready := make(chan []<-chan struct{}, 1)
			queue.Do(func() {
				if ctx.Err() != nil || f.stopped || f.current != w {
					ready <- nil
					return
				}
				ready <- host.Quiesce()
			})
			notify()
			var barriers []<-chan struct{}
			select {
			case barriers = <-ready:
			case <-ctx.Done():
				select {
				case barriers = <-ready:
				default:
					return ctx.Err()
				}
			}
			for _, barrier := range barriers {
				if barrier != nil {
					<-barrier
				}
			}
			return ctx.Err()
		}}
	}
	var progressMu sync.Mutex
	var latest similarity.CacheProgress
	progressQueued := false
	f.changed()
	go func() {
		defer close(w.done)
		for _, predecessor := range predecessors {
			<-predecessor
		}
		r := result{err: ctx.Err()}
		if r.err == nil {
			r = op.run(ctx, provider, func(progress similarity.CacheProgress) {
				progressMu.Lock()
				latest = progress
				if progressQueued {
					progressMu.Unlock()
					return
				}
				progressQueued = true
				progressMu.Unlock()
				queue.Do(func() {
					progressMu.Lock()
					progress := latest
					progressQueued = false
					progressMu.Unlock()
					if f.current != w || !f.open || f.view != w.view {
						return
					}
					f.progress.SetText(fmt.Sprintf(lang.L("Processed %d records (%.1f MB)."), progress.Records, float64(progress.Bytes)/mebibyte))
				})
				notify()
			})
		}
		r.report.Canceled = r.report.Canceled || ctx.Err() != nil || errors.Is(r.err, context.Canceled)
		r.incomplete = r.err != nil && !errors.Is(r.err, context.Canceled) || r.report.Failures > 0 || r.report.Remaining.Incomplete
		queue.Do(func() {
			w.delivered = true
			if f.current == w {
				f.current = nil
				if !f.stopped && (!w.intent.viewBound() || f.open && f.view == w.view) {
					if r.err != nil && !errors.Is(r.err, context.Canceled) {
						fyne.LogError("analysis cache maintenance", r.err)
					}
					f.apply(op, r)
				}
			}
			f.pruneWorkers()
			f.startPendingRoom()
			f.startPendingInspect()
			f.changed()
		})
		notify()
	}()
}

// Busy includes retirement and final UI delivery, keeping analysis admission
// closed until maintenance no longer owns managed records.
func (f *Feature) Busy() bool {
	return f.pendingRoom || f.pendingInspect || f.hasWorkers()
}

func (f *Feature) hasWorkers() bool {
	for _, w := range f.workers {
		if !w.delivered {
			return true
		}
	}
	return false
}

// Close cancels operations belonging to this view without waiting on UI.
func (f *Feature) Close() {
	f.open = false
	f.pendingInspect = false
	f.view++
	if f.current != nil && f.current.intent.viewBound() {
		f.current.cancel()
		f.current = nil
	}
	f.changed()
}
func (f *Feature) Stop() {
	f.stopped = true
	f.pendingRoom, f.pendingReserve = false, 0
	f.Close()
	for _, w := range f.workers {
		w.cancel()
	}
	f.current = nil
}
func (f *Feature) Wait() {
	for _, w := range f.workers {
		<-w.done
	}
}

// Settle drains while waiting: a maintenance worker can itself be waiting for
// the queued Host.Quiesce call before it can finish.
func (f *Feature) Settle() {
	for {
		delivered := f.ui.Drain()
		f.pruneWorkers()
		if len(f.workers) == 0 {
			if !delivered {
				return
			}
			continue
		}
		select {
		case <-f.notice:
		case <-f.workers[0].done:
		}
	}
}

func (f *Feature) pruneWorkers() {
	remaining := f.workers[:0]
	for _, w := range f.workers {
		select {
		case <-w.done:
			if !w.delivered {
				remaining = append(remaining, w)
			}
		default:
			remaining = append(remaining, w)
		}
	}
	clear(f.workers[len(remaining):])
	f.workers = remaining
}
