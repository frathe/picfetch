package display

import "github.com/frathe/picfetch/internal/completion"

// startAnimation admits playback only after the root handoff.
func (f *Feature) startAnimation() {
	if f.stopped || !f.snapshot.Animated {
		return
	}
	token := f.animationLife.begin()
	done := f.animation.Begin()
	count, delays := f.Count(), f.delays
	f.animationWorkers.Go(func() {
		defer done()
		idx := 0
		for token.current() {
			if !f.pause.wait(token.context()) || !token.current() {
				return
			}
			acquisition, changed, running := f.pause.phase()
			if !running {
				continue
			}
			select {
			case <-f.config.AnimationAfter(delays[idx]):
			case <-changed:
				continue
			case <-token.context().Done():
				return
			}
			next := (idx + 1) % count
			applied := make(chan bool, 1)
			f.config.Queue.Do(func() {
				advanced := false
				f.pause.advance(acquisition, func() {
					if !token.current() {
						return
					}
					f.index = next
					f.publish()
					advanced = true
				})
				applied <- advanced
			})
			select {
			case <-token.context().Done():
				return
			case advanced := <-applied:
				if advanced {
					idx = next
				}
			}
		}
	})
}
func (f *Feature) cancelAnimation()                 { f.animationLife.invalidate(); f.pause.unpause() }
func (f *Feature) AnimationDone() completion.Handle { return f.animation.Current() }
func (f *Feature) AnimationBegun() bool             { return f.animation.Begun() }
func (f *Feature) AppliedFrames() uint64            { return f.applied.Load() }
