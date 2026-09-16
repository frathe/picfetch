package display

import (
	"image"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/decodepool"
	"github.com/frathe/picfetch/internal/imaging"
)

// Identity identifies a particular visit to a source, including repeated opens.
type Identity struct {
	Source   fyne.URI
	Revision uint64
}

// Snapshot describes requested and displayed content without exposing frames.
type Snapshot struct {
	Requested, Displayed Identity
	Loading              bool
	Size                 fyne.Size
	Rotation             int
	Vector, Animated     bool
	Duration             time.Duration
	FileSize             int64
	HasEXIF, Preview     bool
}

// Config supplies instance dependencies before presentation starts.
type Config struct {
	Reader         imaging.Reader
	PreloadReader  imaging.Reader
	Queue          UIQueue
	AnimationAfter func(time.Duration) <-chan time.Time
	Callbacks      Callbacks
	Cache          *imaging.ByteCache[*imaging.LoadedImage]
}

// Callbacks carry root-owned composition effects.
type Callbacks struct {
	Repaint            func()
	Requested          func(Identity)
	Probed             func(image.Rectangle)
	Presented          func(Snapshot) []fyne.URI
	Failed             func(fyne.URI, error) fyne.URI
	AnimationTruncated func(fyne.URI)
}

// UIQueue serializes worker results with input and layout.
type UIQueue interface {
	Do(func())
	Drain() bool
}
type fyneQueue struct{}

func (fyneQueue) Do(fn func()) { fyne.Do(fn) }
func (fyneQueue) Drain() bool  { return false }

// Feature owns single-image presentation. UI methods run on the UI goroutine;
// only worker wait/completion observations are used off UI.
type Feature struct {
	frames           []image.Image
	index            int
	rotation         int
	fade             *fyne.Animation
	surface          *canvas.Image
	snapshot         Snapshot
	stopped          bool
	revision         uint64
	logical          fyne.Size
	config           Config
	delays           []time.Duration
	animationLife    requestLifecycle
	animation        completion.Signal
	animationWorkers sync.WaitGroup
	pause            animationPause
	applied          atomic.Uint64
	vector           vectorView
	loadLife         requestLifecycle
	load             completion.Signal
	loadFinish       func()
	loadWorkers      sync.WaitGroup
	preloads         *decodepool.Pool[string, struct{}]
}

// New constructs an owner using the caller's non-nil shared image cache.
func New(config Config) *Feature {
	if config.Queue == nil {
		config.Queue = fyneQueue{}
	}
	if config.AnimationAfter == nil {
		config.AnimationAfter = time.After
	}
	surface := canvas.NewImageFromImage(nil)
	surface.FillMode = canvas.ImageFillContain
	surface.ScaleMode = canvas.ImageScaleSmooth
	surface.Hide()
	f := &Feature{surface: surface, config: config, preloads: decodepool.New[string, struct{}](preloadConcurrency)}
	f.SetVectorOptions(VectorOptions{Debounce: defaultVectorDebounce})
	return f
}

// SetUIQueue configures delivery before the first request.
func (f *Feature) SetUIQueue(queue UIQueue) { f.config.Queue = queue }

// SetAnimationClock configures timing before playback begins.
func (f *Feature) SetAnimationClock(after func(time.Duration) <-chan time.Time) {
	f.config.AnimationAfter = after
}

// Surface is shared with zoom for size/position only. Display publishes pixels.
func (f *Feature) Surface() *canvas.Image { return f.surface }

// beginRequest retires the preceding navigation while retaining outgoing pixels.
func (f *Feature) beginRequest(source fyne.URI) {
	if f.stopped {
		return
	}
	f.cancelAnimation()
	f.cancelLoad()
	f.revision++
	f.snapshot.Requested = Identity{Source: source, Revision: f.revision}
	f.snapshot.Loading = true
}

// CancelRequest retires pending work while keeping outgoing pixels.
func (f *Feature) CancelRequest() {
	f.cancelAnimation()
	f.cancelLoad()
	f.revision++
	f.snapshot.Loading = false
	f.snapshot.Requested = Identity{}
}
func (f *Feature) Snapshot() Snapshot {
	s := f.snapshot
	s.Rotation = f.Rotation()
	s.Size = f.logical
	if s.Rotation%2 != 0 {
		s.Size = fyne.NewSize(s.Size.Height, s.Size.Width)
	}
	return s
}

// present installs one coherent decoded record before the root handoff.
func (f *Feature) present(source fyne.URI, loaded *imaging.LoadedImage, transition bool) {
	if f.stopped || loaded == nil || len(loaded.Frames) == 0 {
		return
	}
	if !f.snapshot.Loading {
		f.beginRequest(source)
	}
	f.clearVector()
	id := Identity{Source: source, Revision: f.revision}
	f.snapshot = Snapshot{Requested: id, Displayed: id, Vector: loaded.Vector != nil,
		Animated: len(loaded.Frames) > 1, FileSize: loaded.FileSize, HasEXIF: loaded.HasEXIF, Preview: loaded.Preview}
	if f.snapshot.Animated {
		for _, delay := range loaded.Delays {
			f.snapshot.Duration += delay
		}
	}
	b := loaded.Frames[0].Bounds()
	f.logical = fyne.NewSize(float32(b.Dx()), float32(b.Dy()))
	f.vector.svg = loaded.Vector
	if loaded.Vector != nil {
		f.vector.raster = b.Size()
	}
	f.frames = slices.Clone(loaded.Frames)
	f.delays = slices.Clone(loaded.Delays)
	f.index = 0
	f.resetRotation()
	if transition {
		f.surface.Translucency = 1
	} else {
		f.ResetFade()
	}
	f.publish()
	f.surface.Show()
	if transition {
		f.FadeTo(1, 0, 400*time.Millisecond)
	}
}

// publish composes orientation with the immutable source frame.
func (f *Feature) publish() {
	if f.Count() == 0 {
		return
	}
	f.surface.Image = imaging.RotateSteps(f.frames[f.index], f.rotation)
	f.surface.Refresh()
	f.applied.Add(1)
}

func (f *Feature) RotateBy(steps int) {
	if f.Count() == 0 {
		return
	}
	f.rotation = normalizedRotation(f.rotation + steps)
	f.publish()
}

func (f *Feature) ResetRotation() bool {
	if !f.resetRotation() {
		return false
	}
	f.publish()
	return true
}

func (f *Feature) FadeTo(start, end float32, duration time.Duration) {
	if f.stopped {
		return
	}
	if f.fade != nil {
		f.fade.Stop()
	}
	f.fade = fyne.NewAnimation(duration, func(t float32) {
		f.surface.Translucency = float64(start + t*(end-start))
		f.surface.Refresh()
	})
	f.fade.Start()
}

func (f *Feature) ResetFade() {
	if f.fade != nil {
		f.fade.Stop()
		f.fade = nil
	}
	f.surface.Translucency = 0
	f.surface.Refresh()
}

func (f *Feature) Clear() {
	f.CancelRequest()
	f.clearVector()
	f.frames = nil
	f.index = 0
	f.rotation = 0
	f.delays = nil
	f.ResetFade()
	f.snapshot = Snapshot{}
	f.logical = fyne.Size{}
	f.surface.Image = nil
	f.surface.Hide()
	f.surface.Refresh()
}
func (f *Feature) Stop() { f.stopped = true; f.Clear() }

func (f *Feature) resetRotation() bool {
	if f.rotation == 0 {
		return false
	}
	f.rotation = 0
	return true
}

// Wait joins every worker generation after cancellation, without draining UI.
func (f *Feature) Wait() {
	f.loadWorkers.Wait()
	f.preloads.Wait()
	f.animationWorkers.Wait()
	f.vector.pending.Wait()
}
func (f *Feature) Settle() {
	for {
		f.loadWorkers.Wait()
		f.vector.pending.Wait()
		f.preloads.Wait()
		if !f.config.Queue.Drain() {
			return
		}
	}
}
