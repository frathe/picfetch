package display

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
)

// Request names one navigation, including its broken-source retries.
type Request struct {
	Source     fyne.URI
	Transition bool
}

func (f *Feature) Load(request Request) {
	if f.stopped || request.Source == nil {
		return
	}
	f.beginRequest(request.Source)
	token := f.loadLife.begin()
	done := f.load.Begin()
	f.loadFinish = done
	if f.config.Callbacks.Requested != nil {
		f.config.Callbacks.Requested(f.snapshot.Requested)
	}
	if !token.current() {
		done()
		return
	}
	f.attemptLoad(token, request, done)
}

func (f *Feature) cancelLoad() {
	f.loadLife.invalidate()
	if f.loadFinish != nil {
		f.loadFinish()
		f.loadFinish = nil
	}
}

func (f *Feature) attemptLoad(token requestToken, request Request, done func()) {
	if !token.current() {
		done()
		return
	}
	u := request.Source
	f.snapshot.Requested.Source = u
	writer := f.config.Cache.Capture()
	if loaded, ok := f.config.Cache.Get(u.String()); ok {
		if !writer.Current() {
			f.attemptLoad(token, request, done)
			return
		}
		f.finishLoad(token, request, loaded, done)
		return
	}
	f.loadWorkers.Go(func() {
		data, bounds, err := imaging.ReadAndProbe(token.context(), u)
		if err == nil && f.config.Callbacks.Probed != nil {
			f.config.Queue.Do(func() {
				if token.current() && writer.Current() {
					f.config.Callbacks.Probed(bounds)
				}
			})
		}
		var loaded *imaging.LoadedImage
		if err == nil {
			loaded, err = imaging.DecodeRecord(token.context(), data, f.config.Cache.Budget())
		}
		f.config.Queue.Do(func() {
			if !token.current() {
				done()
				return
			}
			if !writer.Current() {
				f.attemptLoad(token, request, done)
				return
			}
			if err != nil {
				f.failedLoad(token, request, err, done)
				return
			}
			if len(loaded.Frames) == 0 || loaded.Frames[0].Bounds().Empty() {
				f.failedLoad(token, request, &imaging.InvalidDimensionsError{}, done)
				return
			}
			if !writer.Add(u.String(), loaded) {
				f.attemptLoad(token, request, done)
				return
			}
			if loaded.AnimationTruncated && f.config.Callbacks.AnimationTruncated != nil {
				f.config.Callbacks.AnimationTruncated(u)
			}
			if !token.current() {
				done()
				return
			}
			f.finishLoad(token, request, loaded, done)
		})
	})
}

func (f *Feature) failedLoad(token requestToken, request Request, err error, done func()) {
	var replacement fyne.URI
	if f.config.Callbacks.Failed != nil {
		replacement = f.config.Callbacks.Failed(request.Source, err)
	}
	if !token.current() {
		done()
		return
	}
	if replacement == nil {
		f.snapshot.Loading = false
		token.cancelContext()
		done()
		return
	}
	request.Source = replacement
	f.attemptLoad(token, request, done)
}

func (f *Feature) finishLoad(token requestToken, request Request, loaded *imaging.LoadedImage, done func()) {
	if !token.current() {
		done()
		return
	}
	f.present(request.Source, loaded, request.Transition)
	var candidates []fyne.URI
	if f.config.Callbacks.Presented != nil {
		candidates = f.config.Callbacks.Presented(f.Snapshot())
	}
	if !token.current() {
		done()
		return
	}
	f.startAnimation()
	for _, source := range candidates {
		f.preloadOne(token, source)
	}
	done()
}

func (f *Feature) LoadDone() completion.Handle { return f.load.Current() }
func (f *Feature) LoadBegun() bool             { return f.load.Begun() }
func (f *Feature) RequestRevision() uint64     { return f.revision }
