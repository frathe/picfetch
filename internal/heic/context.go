package heic

import "context"

type snapshotKey struct{}

// WithSnapshot carries immutable admission through existing imaging contexts.
func WithSnapshot(ctx context.Context, snapshot Snapshot) context.Context {
	return context.WithValue(ctx, snapshotKey{}, snapshot)
}

// FromContext returns the operation's captured policy; absent means unavailable.
func FromContext(ctx context.Context) Snapshot {
	snapshot, _ := ctx.Value(snapshotKey{}).(Snapshot)
	return snapshot
}

// ReportUnavailable carries a private producer's decisive backend-loss result
// back to the capability generation that admitted the producer.
func ReportUnavailable(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	if backend, ok := FromContext(ctx).Backend.(observedBackend); ok {
		backend.capability.Invalidate(backend.generation)
	}
}

// CaptureContext binds new work to the capability currently effective for this
// app. A nil capability keeps ordinary standalone imaging unchanged.
func (c *Capability) CaptureContext(ctx context.Context) context.Context {
	if c == nil {
		return ctx
	}
	return WithSnapshot(ctx, c.Snapshot())
}
