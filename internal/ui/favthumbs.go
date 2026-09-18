// favthumbs.go is the viewer-side wiring for the disk-backed favorite
// preview cache (internal/favthumbs): the "Cache favorite previews on disk"
// preference the settings window binds to, and the background pass the
// favorites feature asks for whenever a favorite's file list changes.

package ui

import (
	"context"
	"errors"
	"image"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favthumbs"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
)

// FavoritePreviewCache and SetFavoritePreviewCache are the settings
// window's getter/setter pair for the preference, the same shape
// memlimits.go uses for the three memory limits.
func (v *viewer) FavoritePreviewCache() bool { return v.settings.favPreviewCache }

// SetFavoritePreviewCache applies the preference, and switching it off also
// abandons any pass still in flight. Guarding only the *start* of a pass
// would leave the checkbox lying about itself: a favorite big enough to be
// worth caching is also big enough that its pass runs for a while, and
// during that while the app would keep decoding and keep writing preview
// files to disk after the user asked it to stop doing exactly that.
func (v *viewer) SetFavoritePreviewCache(on bool) {
	v.settings.favPreviewCache = on

	if !on {
		v.favThumbLifecycle.invalidate()
	}
}

// SetFavoritePreviewLimit retires a pass admitted under the previous limit.
// The next Favorite open/save captures the new limit before launching work.
func (v *viewer) SetFavoritePreviewLimit(n int) {
	if n <= 0 {
		n = preferences.DefaultFavoritePreviewLimit
	}
	if v.settings.favPreviewLimit != n {
		v.settings.favPreviewLimit = n
		v.favThumbLifecycle.invalidate()
	}
}

// SyncFavoritePreviews brings the previews stored under favDir in line with
// files, in the background - the favorites feature's report that a favorite
// now holds this list, arriving both when one is saved and when one is
// opened. This is where that report turns into thumbnail work: favorites
// itself has no idea previews exist, the same way batch.go rather than the
// grid decides what a selection means.
//
// Deliberately not skipped for an empty files slice: a favorite the user
// emptied should have its previews swept, and that is exactly what a Sync
// over no files does.
func (v *viewer) SyncFavoritePreviews(favDir string, files []fyne.URI) {
	// favDir is empty when favstore.Dir was handed a name it rejects, which
	// leaves nothing to write previews into or sweep.
	if v.favThumbClosed || !v.settings.favPreviewCache || favDir == "" {
		return
	}

	// A new pass supersedes and cancels any pass still running, so opening
	// favorite B while A is still being walked stops A rather than leaving
	// two passes competing for decodes - and for the thumbnail cache, where
	// the loser would be evicting the winner's entries.
	token := v.favThumbLifecycle.begin()

	done := v.favThumb.Begin()
	sink := gridSink{writer: v.grid.CaptureThumbs()}
	limit := v.settings.favPreviewLimit

	v.favThumbWorkers.Go(func() {
		defer done()
		defer token.cancelContext()

		if err := favthumbs.Sync(token.context(), favDir, files, limit, sink); err != nil {
			// A superseded pass returns context.Canceled, which is this
			// design working rather than anything failing.
			if errors.Is(err, context.Canceled) {
				return
			}
			fyne.LogError("failed to cache favorite previews", err)
		}
	})
}

// gridSink adapts the grid overview to favthumbs.Sink, so a preview pass
// can skip decoding what the grid already holds and hand back what it
// produced. This is the pre-warm: by the time the user presses G, the
// thumbnails for the favorite they just opened are already in memory.
//
// Both methods are called from several of Sync's worker goroutines at once,
// and neither wraps its work in fyne.Do - unlike almost everything else
// this package does off the UI goroutine. That is safe *because* of how
// little they reach: Peek and RefreshIfRoom use the grid's captured
// imaging.ByteCache writer, which guards itself with a mutex, and touch no widget,
// no canvas, and no viewer field. Anything added here that does touch a
// widget needs fyne.Do again.
type gridSink struct {
	writer imaging.CacheWriter[image.Image]
}

func (s gridSink) Cached(src fyne.URI) (image.Image, bool) {
	return s.writer.Peek(src.String())
}

func (s gridSink) Store(src fyne.URI, thumb image.Image) {
	// Background warming must not evict thumbnails already in use. A separate
	// "full" check misses a partially free cache and races other producers;
	// generation, remaining space and admission must share the cache lock.
	// Sync rejected an older source version before offering these pixels.
	// Replace that key if possible, or discard it if the new version cannot
	// fit; unrelated thumbnails keep their bytes and recency.
	_ = s.writer.RefreshIfRoom(src.String(), thumb)
}

// closeFavoritePreviews stops admission and cancels the pass without waiting
// on an already-blocked external read. favThumbWorkers tracks its eventual
// completion, including older passes superseded by the latest Signal.
func (v *viewer) closeFavoritePreviews() {
	v.favThumbClosed = true
	v.favThumbLifecycle.invalidate()
}
