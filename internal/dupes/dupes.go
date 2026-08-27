// Package dupes owns which files in a file set are duplicates of which
// other files: dHashes and native pixel sizes keyed by a stable string
// per file, the Hamming distance threshold that turns those hashes into
// groups, and the groups themselves.
//
// It is viewer-independent - Fyne-free, no import of internal/ui or any
// feature package - so grouping, generation handling, and distance
// clamping can be unit-tested directly instead of only being reachable
// through a widget.GridWrap and a Fyne test app. The file set is reached
// through the FileSet interface below, with string keys rather than
// fyne.URI.
//
// Every fact stored here is scoped to a FileSet generation: WipeIfStale
// drops hashes, failures, and native sizes when the generation moves on
// (a fresh drop), while AdoptGeneration keeps them (an incremental
// shrink). See AdoptGeneration's doc comment for why that distinction
// matters.
package dupes

import (
	"image"
	"sync"
	"sync/atomic"

	"github.com/frathe/picfetch/internal/imaging"
)

// FileSet is the ordered file set the model groups over.
type FileSet interface {
	Count() int
	KeyAt(i int) string // stable per file; the app passes URI strings
	Generation() uint64 // file-set revision; a change invalidates stored facts
}

// MaxDistance is the largest Hamming threshold SetDistance accepts.
const MaxDistance = 32

// Model owns duplicate-detection facts and the current group snapshot
// for a FileSet. The zero value is not usable - see New.
type Model struct {
	set FileSet

	// mu guards every field below, including groups: Compute snapshots
	// dist under this lock because hashing workers call it off the UI
	// goroutine, and guarding the installed groups snapshot the same
	// way removes a class of race even though today's callers only
	// touch it from the UI goroutine.
	mu sync.Mutex

	// hashes maps a file key to its dHash. Not stored alongside
	// thumbnails: a hash is 8 bytes and must survive thumbnail
	// eviction. native maps a file key to its EXIF-oriented pixel size
	// (Dx, Dy) for the same generation; absent means unknown.
	// Thumbnails are capped, so size cannot be recovered from a thumb
	// cache. gen is the FileSet generation those entries belong to; a
	// newer drop wipes hashes, hashFailed, and native.
	hashes map[string]uint64
	native map[string]image.Point
	// hashFailed are keys whose thumbnail decode already failed this
	// generation. Callers must not retry them: mixed-format drops leave
	// unreadable files, and retrying on every hash pass re-raises
	// "analyzing" chrome with no CPU work left to do.
	hashFailed map[string]struct{}
	gen        uint64

	dist   int
	groups Groups
	// computes counts Compute calls so tests can prove a snapshot was
	// computed off the UI queue rather than inside it.
	computes atomic.Int32

	observers []func()
}

// New builds a Model over set. The Hamming distance defaults to
// imaging.DuplicateMaxDistance.
func New(set FileSet) *Model {
	return &Model{
		set:        set,
		hashes:     make(map[string]uint64),
		hashFailed: make(map[string]struct{}),
		native:     make(map[string]image.Point),
		dist:       imaging.DuplicateMaxDistance,
	}
}

// ensureMapsLocked allocates hashes, hashFailed, and native if any of
// them is nil, leaving any existing entries untouched. Callers must hold
// mu.
func (m *Model) ensureMapsLocked() {
	if m.hashes == nil {
		m.hashes = make(map[string]uint64)
	}
	if m.hashFailed == nil {
		m.hashFailed = make(map[string]struct{})
	}
	if m.native == nil {
		m.native = make(map[string]image.Point)
	}
}

// wipeIfStaleLocked is WipeIfStale's body for callers that already hold
// mu with gen already read.
func (m *Model) wipeIfStaleLocked(gen uint64) {
	if m.gen != gen {
		m.hashes = make(map[string]uint64)
		m.hashFailed = make(map[string]struct{})
		m.native = make(map[string]image.Point)
		m.gen = gen
	}
	m.ensureMapsLocked()
}

// WipeIfStale ensures the stored facts belong to set's current
// generation, wiping hashes, hashFailed, and native when they don't.
// Every read path in this file calls it first.
func (m *Model) WipeIfStale() {
	gen := m.set.Generation()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wipeIfStaleLocked(gen)
}

// AdoptGeneration records set's current generation without dropping
// hashes, hashFailed, or native. An incremental shrink (RemoveFiles ->
// FilesChanged) is not a new drop: surviving files keep their hashes so
// grouping and inspect retarget still work. Orphan keys for deleted
// files linger until the next full-set change, which is harmless. Do
// not call WipeIfStale here: that wipes on a mismatch.
func (m *Model) AdoptGeneration() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gen = m.set.Generation()
	m.ensureMapsLocked()
}

// Clear drops all stored facts regardless of generation.
func (m *Model) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hashes = make(map[string]uint64)
	m.hashFailed = make(map[string]struct{})
	m.native = make(map[string]image.Point)
}

// PutHash records key's dHash, adopting set's current generation if it
// has moved on, and clears any failure previously recorded for key.
func (m *Model) PutHash(key string, h uint64) {
	gen := m.set.Generation()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wipeIfStaleLocked(gen)
	m.hashes[key] = h
	delete(m.hashFailed, key)
}

// PutFailed marks key's thumbnail decode as having already failed this
// generation.
func (m *Model) PutFailed(key string) {
	gen := m.set.Generation()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wipeIfStaleLocked(gen)
	m.hashFailed[key] = struct{}{}
}

// PutNativeSize records key's native pixel size, clamping either edge to
// 0 if it is negative. Callers with an image.Rectangle convert it to a
// size (Dx, Dy) themselves - this package works with plain sizes, not
// rectangles with an origin.
func (m *Model) PutNativeSize(key string, sz image.Point) {
	sz = image.Pt(max(sz.X, 0), max(sz.Y, 0))
	gen := m.set.Generation()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wipeIfStaleLocked(gen)
	m.native[key] = sz
}

// Hash returns key's stored dHash.
func (m *Model) Hash(key string) (uint64, bool) {
	m.WipeIfStale()
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.hashes[key]
	return h, ok
}

// Failed reports whether key's thumbnail decode already failed this
// generation.
func (m *Model) Failed(key string) bool {
	m.WipeIfStale()
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.hashFailed[key]
	return ok
}

// NativeSize returns key's stored native pixel size.
func (m *Model) NativeSize(key string) (image.Point, bool) {
	m.WipeIfStale()
	m.mu.Lock()
	defer m.mu.Unlock()
	sz, ok := m.native[key]
	return sz, ok
}

// PixelCount is key's native size reduced to a single pixel count.
func (m *Model) PixelCount(key string) (int, bool) {
	sz, ok := m.NativeSize(key)
	if !ok {
		return 0, false
	}
	return sz.X * sz.Y, true
}

// NativeSizeAt is the EXIF-oriented pixel size of the file at index i in
// set. ok is false when i is out of range, no probe has been stored for
// it, or a stored size has a non-positive edge (failed/empty probe).
func (m *Model) NativeSizeAt(i int) (w, h int, ok bool) {
	if i < 0 || i >= m.set.Count() {
		return 0, 0, false
	}
	sz, ok := m.NativeSize(m.set.KeyAt(i))
	if !ok || sz.X <= 0 || sz.Y <= 0 {
		return 0, 0, false
	}
	return sz.X, sz.Y, true
}

// Distance is the current Hamming threshold Compute groups at.
func (m *Model) Distance() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dist
}

// SetDistance clamps n to [0, MaxDistance] and reports whether the
// stored value actually changed.
func (m *Model) SetDistance(n int) bool {
	if n < 0 {
		n = 0
	}
	if n > MaxDistance {
		n = MaxDistance
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dist == n {
		return false
	}
	m.dist = n
	return true
}

// OnChange registers f to run when Notify is called. Registration order
// is fire order.
func (m *Model) OnChange(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers = append(m.observers, f)
}

// Notify fires every registered observer, in registration order. Nothing
// in this package calls it yet; a later stage wires it to group
// installs, mode changes, and distance changes.
func (m *Model) Notify() {
	m.mu.Lock()
	observers := m.observers
	m.mu.Unlock()
	for _, f := range observers {
		f()
	}
}
