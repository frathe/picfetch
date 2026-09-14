// Speculative image decoding shares the current navigation's cancellation.

package display

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

const preloadConcurrency = 2

func (f *Feature) preloadOne(token requestToken, u fyne.URI) {
	key := u.String()
	cacheWrite := f.config.Cache.Capture()

	// Contains, not Get: a presence test on a speculative path shouldn't
	// promote the neighbor to most-recently-used, which under a tight byte
	// budget could make it outlive the image actually on screen.
	if f.config.Cache.Contains(key) {
		return
	}
	if !f.preloads.Claim(key, struct{}{}) {
		return
	}

	// Bounded the same way the grid's thumbnail decodes are:
	// root only asks for two neighbors per settled image,
	// but rapid navigation could otherwise stack an unbounded number
	// of these full-size decode goroutines.
	f.preloads.Go(token.context(), func(acquired bool) {
		defer f.preloads.Release(key, struct{}{})

		// acquired is false when the token's context was cancelled while
		// this was still queued for a slot - the pool runs fn either way
		// precisely so the deferred Release above still clears the claim.
		if !acquired || !token.current() {
			return
		}

		data, bounds, err := imaging.ReadAndProbe(token.context(), u)
		if err != nil {
			return
		}

		// Read once: the settings window can change the budget between
		// these two uses, and a gate that passed under one value shouldn't
		// then decode under another.
		budget := f.config.Cache.Budget()

		// Preloading exists to make the *next* navigation instant. An
		// image big enough that caching it would evict what's on screen
		// turns that speculative win into a guaranteed re-decode of the
		// current image, so bail on the header alone rather than paying
		// for the decode first. Half the budget is where the current image
		// and one neighbor stop both fitting.
		if imaging.EstimateDecodedBytes(bounds) > budget/2 {
			return
		}

		loaded, err := imaging.DecodeRecord(token.context(), data, budget)
		if err != nil {
			return
		}

		b := loaded.Frames[0].Bounds()
		if b.Dx() == 0 || b.Dy() == 0 {
			return
		}

		if !token.current() {
			return
		}

		// Speculation must fit alongside existing content. The admission
		// checks remaining space and the captured generation atomically.
		_ = cacheWrite.AddIfRoom(key, loaded)
	})
}

// WaitPreloads observes speculative workers separately from load completion.
func (f *Feature) WaitPreloads() { f.preloads.Wait() }
