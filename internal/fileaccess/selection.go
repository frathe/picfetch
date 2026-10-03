package fileaccess

import (
	"context"
	"errors"
	"sync"

	"fyne.io/fyne/v2"
)

type selectionURI struct {
	fyne.URI
	capture        func(context.Context) (Record, error)
	release        func()
	mu             sync.Mutex
	active, closed bool
}

// NewSelection transfers ownership of a native selected URL to an input batch.
// Its capture callback runs only on the consuming worker; release retires the URL.
func NewSelection(uri fyne.URI, capture func(context.Context) (Record, error), release func()) fyne.URI {
	return &selectionURI{URI: uri, capture: capture, release: release}
}

// NeedsCapture identifies selected URLs whose metadata must be captured off UI.
func NeedsCapture(uri fyne.URI) bool { _, ok := uri.(*selectionURI); return ok }

// CaptureSelected consumes selected input and refreshes restored source metadata
// on the opening worker. Unavailable sources retain their previous record for
// the scanner's per-item/offline handling; cancellation aborts the opening.
func CaptureSelected(ctx context.Context, files []fyne.URI) ([]fyne.URI, error) {
	defer ReleaseSelected(files)
	result := make([]fyne.URI, len(files))
	captured := make(map[*selectionURI]fyne.URI)
	refreshed := make(map[*sourceURI]fyne.URI)
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if file == nil {
			return nil, errors.New("selected input contains nil URI")
		}
		selected, ok := file.(*selectionURI)
		if !ok {
			result[i] = file
			if saved, scoped := file.(*sourceURI); scoped {
				current, seen := refreshed[saved]
				if !seen {
					current = file
					resolved, release, err := Acquire(ctx, saved)
					if err == nil {
						current = resolved
						release()
					}
					refreshed[saved] = current
				}
				result[i] = current
			}
			continue
		}
		source, ok := captured[selected]
		if !ok {
			var err error
			source, err = selected.consume(ctx)
			if err != nil {
				return nil, err
			}
			captured[selected] = source
		}
		result[i] = source
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

var errSelectionClosed = errors.New("selected input is already consumed or closed")

func (s *selectionURI) consume(ctx context.Context) (fyne.URI, error) {
	s.mu.Lock()
	if s.closed || s.active {
		s.mu.Unlock()
		return nil, errSelectionClosed
	}
	s.active = true
	s.mu.Unlock()
	record, err := s.capture(ctx)
	s.mu.Lock()
	discarded := s.closed
	s.active, s.closed = false, true
	s.mu.Unlock()
	s.release()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if discarded {
		return nil, errSelectionClosed
	}
	if len(record.Bookmark) == 0 {
		return nil, errors.New("selected input has no captured authority")
	}
	return FromRecord(record)
}

// ReleaseSelected discards unstarted input without waiting on an active capture.
// An active capture retains native ownership until its callback has returned.
func ReleaseSelected(files []fyne.URI) {
	for _, file := range files {
		s, ok := file.(*selectionURI)
		if !ok {
			continue
		}
		s.mu.Lock()
		release := !s.closed && !s.active
		s.closed = true
		s.mu.Unlock()
		if release {
			s.release()
		}
	}
}
