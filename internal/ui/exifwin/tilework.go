package exifwin

import (
	"context"
	"time"
)

// URL jobs share one queue and worker count across foreground and warm work.
// A warm caller waits on job completion; it never creates a worker per URL.
type tileJob struct {
	url        string
	ctx        context.Context
	done       chan struct{}
	foreground bool // guarded by the fetcher mutex; a paint can join a warm job
}

// session captures the lifetime before handing a warm pass to a worker.
func (f *tileFetcher) session() context.Context {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ctx
}

// Restart cancels obsolete work and opens a session over the retained cache.
// Cancelled active requests retain their worker slots until the HTTP call ends.
func (f *tileFetcher) Restart() context.Context {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelLocked()
	if !f.stopped {
		f.ctx, f.cancel = context.WithCancel(context.Background())
	}
	return f.ctx
}

func (f *tileFetcher) Cancel() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelLocked()
}

func (f *tileFetcher) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
	f.cancelLocked()
}

func (f *tileFetcher) cancelLocked() {
	f.cancel()
	for _, job := range f.queue {
		if f.inflight[job.url] == job {
			delete(f.inflight, job.url)
		}
		close(job.done)
	}
	f.queue = nil
	f.changedLocked()
}

// Wait includes each worker's onChange call. It is for off-UI teardown or a
// harness that has finished submitting warm work, never for a UI cancel action.
func (f *tileFetcher) Wait() { f.work.Wait() }

func (f *tileFetcher) changedLocked() {
	close(f.changed)
	f.changed = make(chan struct{})
}

// submit returns the claimed job (also for a duplicate), or a capacity signal
// for warm callers. A nil job and nil signal means cached, backed off or stopped.
func (f *tileFetcher) submit(ctx context.Context, url string, foreground bool) (*tileJob, <-chan struct{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopped || ctx.Err() != nil || f.cache.Contains(url) {
		return nil, nil
	}
	if job := f.inflight[url]; job != nil && job.ctx.Err() == nil {
		job.foreground = job.foreground || foreground
		return job, nil
	}
	f.expireFailuresLocked()
	if _, failed := f.failed[url]; failed {
		return nil, nil
	}
	if len(f.queue) >= tileQueueCapacity {
		return nil, f.changed
	}
	job := &tileJob{url: url, ctx: ctx, done: make(chan struct{}), foreground: foreground}
	f.inflight[url] = job
	f.queue = append(f.queue, job)
	if f.workers < tileWorkers {
		f.workers++
		f.work.Go(f.runWorker)
	}
	return job, nil
}

func (f *tileFetcher) runWorker() {
	for {
		f.mu.Lock()
		if len(f.queue) == 0 {
			f.workers--
			f.mu.Unlock()
			return
		}
		job := f.queue[0]
		f.queue[0] = nil
		f.queue = f.queue[1:]
		f.changedLocked()
		f.mu.Unlock()

		var data []byte
		err := job.ctx.Err()
		if err == nil {
			data, err = f.get(job.ctx, job.url)
		}
		f.releaseJob(job, data, err)
	}
}

func (f *tileFetcher) releaseJob(job *tileJob, data []byte, err error) {
	f.mu.Lock()
	current := job.ctx.Err() == nil && f.inflight[job.url] == job
	if f.inflight[job.url] == job {
		delete(f.inflight, job.url)
	}
	if current {
		if err == nil {
			f.cache.Add(job.url, data)
			delete(f.failed, job.url)
		} else {
			f.expireFailuresLocked()
			if len(f.failed) >= tileFailureCapacity {
				var oldestURL string
				var oldest time.Time
				for url, at := range f.failed {
					if oldestURL == "" || at.Before(oldest) {
						oldestURL, oldest = url, at
					}
				}
				delete(f.failed, oldestURL)
			}
			f.failed[job.url] = f.now()
		}
	}
	pending, onChange := f.currentPendingLocked(), f.onChange
	if !current || !job.foreground {
		onChange = nil
	}
	f.changedLocked()
	f.mu.Unlock()

	if onChange != nil {
		onChange(pending)
	}
	close(job.done)
}

func (f *tileFetcher) currentPendingLocked() int {
	n := 0
	for _, job := range f.inflight {
		if job.ctx.Err() == nil {
			n++
		}
	}
	return n
}

func (f *tileFetcher) expireFailuresLocked() {
	now := f.now()
	for url, at := range f.failed {
		if now.Sub(at) >= tileRetryAfter {
			delete(f.failed, url)
		}
	}
}

// Warm retains the synchronous convenience API for a caller that has no
// intervening session handoff. Window captures its context before launching.
func (f *tileFetcher) Warm(lat, lon float64, zoom int) {
	f.WarmContext(f.session(), lat, lon, zoom)
}

func (f *tileFetcher) WarmContext(ctx context.Context, lat, lon float64, zoom int) {
	var jobs []*tileJob
	for _, url := range f.neighborhood(lat, lon, zoom) {
		for {
			job, capacity := f.submit(ctx, url, false)
			if capacity == nil {
				if job != nil {
					jobs = append(jobs, job)
				}
				break
			}
			select {
			case <-ctx.Done():
				return
			case <-capacity:
			}
		}
	}
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return
		case <-job.done:
		}
	}
}
