package grid

import (
	"context"
	"errors"

	"github.com/frathe/picfetch/internal/dupes"
)

// workSession is UI-owned. Workers capture ctx and revision at admission;
// they never read the mutable session. Hash accounting belongs to one session,
// while the decode pool, thumbnail cache and duplicate model outlive it.
type workSession struct {
	ctx        context.Context
	cancel     context.CancelFunc
	generation uint64
	revision   uint64
	stopped    bool
	facts      dupes.FactWriter
}

// A reopened cell can have the same id while its old decode still unwinds.
// Including the session revision keeps that old Release from erasing new work.
type thumbClaim struct {
	id       int
	revision uint64
}

func (g *Overview) restartWork() {
	if g.work.stopped {
		return
	}
	if g.work.cancel != nil {
		g.work.cancel()
	}
	g.work.ctx, g.work.cancel = context.WithCancel(context.Background())
	g.work.generation = g.host.Generation()
	g.work.revision++
	g.work.facts = g.dupes.CaptureFacts()
	g.hashes = &hashEngine{host: g.host, pool: g.decodes, thumbs: g.thumbs, model: g.dupes, facts: g.work.facts, ui: g.ui}
}

func (g *Overview) workContext() context.Context {
	if !g.work.stopped && (g.work.generation != g.host.Generation() || !g.work.facts.Current()) {
		g.restartWork()
	}
	return g.work.ctx
}

// Explicit open/hash/warm commands can resume an ordinary closed session.
// Cell refreshes cannot: close itself can refresh hidden/recycled widgets.
func (g *Overview) resumeWork() context.Context {
	ctx := g.workContext()
	if ctx.Err() != nil && !g.work.stopped {
		g.restartWork()
	}
	return g.work.ctx
}

// Stop permanently ends admission and cancels the overview's background work.
// It does not wait on a blocked external read or require a UI callback to run.
// Settle observes eventual completion through the shared pool and UI queue.
func (g *Overview) Stop() {
	g.work.stopped = true
	if g.work.cancel != nil {
		g.work.cancel()
	}
}

func workCancelled(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
