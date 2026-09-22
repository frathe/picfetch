package similarity

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
)

// Only immutable policy crosses the producer boundary. The desktop's Backend
// owns a different process lifetime and cannot be reused inside this worker.
type workerHEIC struct {
	Available  bool
	Generation uint64
}

func (r *request) captureHEIC(snapshot heic.Snapshot) {
	r.HEIC = workerHEIC{Available: snapshot.Available && snapshot.Backend != nil, Generation: snapshot.Generation}
	r.MaxPixels = imaging.MaxImagePixels()
}

func workerHEICContext(ctx context.Context, req request) (context.Context, func()) {
	snapshot := heic.Snapshot{Available: req.HEIC.Available, Generation: req.HEIC.Generation}
	if !snapshot.Available {
		return heic.WithSnapshot(ctx, snapshot), func() {}
	}
	client := heic.NewClient("")
	maxPixels := req.MaxPixels
	if maxPixels <= 0 || maxPixels > imaging.MaxImagePixels() {
		maxPixels = imaging.MaxImagePixels()
	}
	snapshot.Backend = &boundedHEIC{Backend: client, maxPixels: maxPixels, maxEncodedBytes: imaging.MaxEncodedBytes()}
	return heic.WithSnapshot(ctx, snapshot), func() { client.Stop(); client.Wait() }
}

type boundedHEIC struct {
	heic.Backend
	maxEncodedBytes, maxPixels int64
	unavailable                atomic.Bool
}

func (b *boundedHEIC) Read(ctx context.Context, data []byte, req heic.Request) (heic.Result, error) {
	req.MaxEncodedBytes = min(req.MaxEncodedBytes, b.maxEncodedBytes)
	req.MaxPixels = min(req.MaxPixels, b.maxPixels)
	result, err := b.Backend.Read(ctx, data, req)
	if errors.Is(err, heic.ErrUnavailable) {
		b.unavailable.Store(true)
	}
	return result, err
}

func workerHEICUnavailable(ctx context.Context) bool {
	backend, ok := heic.FromContext(ctx).Backend.(*boundedHEIC)
	return ok && backend.unavailable.Load()
}
