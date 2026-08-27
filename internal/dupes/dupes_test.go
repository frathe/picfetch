package dupes

import (
	"image"
	"slices"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

// fakeSet is a minimal FileSet for tests: a fixed list of keys plus a
// generation the test controls directly, so wipe-vs-adopt can be
// exercised without any real file scan.
type fakeSet struct {
	keys []string
	gen  uint64
}

func (f *fakeSet) Count() int         { return len(f.keys) }
func (f *fakeSet) KeyAt(i int) string { return f.keys[i] }
func (f *fakeSet) Generation() uint64 { return f.gen }

func newFakeSet(n int, gen uint64) *fakeSet {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = string(rune('a' + i))
	}
	return &fakeSet{keys: keys, gen: gen}
}

func TestNew_DefaultDistanceIsImagingDuplicateMaxDistance(t *testing.T) {
	m := New(newFakeSet(3, 1))

	if got := m.Distance(); got != imaging.DuplicateMaxDistance {
		t.Errorf("Distance() = %d, want %d", got, imaging.DuplicateMaxDistance)
	}
}

func TestPutHash_And_Hash_RoundTrip(t *testing.T) {
	m := New(newFakeSet(3, 1))

	m.PutHash("a", 42)

	h, ok := m.Hash("a")
	if !ok || h != 42 {
		t.Errorf("Hash(a) = (%d, %v), want (42, true)", h, ok)
	}
	if _, ok := m.Hash("b"); ok {
		t.Error("Hash(b) ok = true for a key never put, want false")
	}
}

func TestPutNativeSize_And_NativeSize_RoundTrip(t *testing.T) {
	m := New(newFakeSet(3, 1))

	m.PutNativeSize("a", image.Pt(100, 50))

	sz, ok := m.NativeSize("a")
	if !ok || sz != image.Pt(100, 50) {
		t.Errorf("NativeSize(a) = (%v, %v), want (%v, true)", sz, ok, image.Pt(100, 50))
	}
}

func TestPutNativeSize_ClampsNegativeEdgesToZero(t *testing.T) {
	m := New(newFakeSet(3, 1))

	m.PutNativeSize("a", image.Pt(-5, 10))
	m.PutNativeSize("b", image.Pt(5, -10))

	if sz, _ := m.NativeSize("a"); sz != image.Pt(0, 10) {
		t.Errorf("NativeSize(a) = %v, want (0, 10)", sz)
	}
	if sz, _ := m.NativeSize("b"); sz != image.Pt(5, 0) {
		t.Errorf("NativeSize(b) = %v, want (5, 0)", sz)
	}
}

func TestPixelCount_DerivedFromNativeSize(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.PutNativeSize("a", image.Pt(4, 5))

	px, ok := m.PixelCount("a")
	if !ok || px != 20 {
		t.Errorf("PixelCount(a) = (%d, %v), want (20, true)", px, ok)
	}

	if _, ok := m.PixelCount("z"); ok {
		t.Error("PixelCount(z) ok = true for an unprobed key, want false")
	}
}

func TestFailed_TrueOnlyAfterPutFailed(t *testing.T) {
	m := New(newFakeSet(3, 1))

	if m.Failed("a") {
		t.Error("Failed(a) = true before PutFailed, want false")
	}

	m.PutFailed("a")

	if !m.Failed("a") {
		t.Error("Failed(a) = false after PutFailed, want true")
	}
}

func TestPutHash_ClearsAPriorFailure(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.PutFailed("a")

	m.PutHash("a", 7)

	if m.Failed("a") {
		t.Error("Failed(a) = true after a later PutHash, want false: PutHash must clear the failed flag")
	}
}

// TestWipeIfStale_WipesFactsOnGenerationChange and
// TestAdoptGeneration_KeepsFactsAcrossGenerationChange are the two halves
// of the single most load-bearing invariant in this package: a full
// file-set change must drop stale hashes, but an incremental shrink must
// not. Keep both directions covered explicitly.
func TestWipeIfStale_WipesFactsOnGenerationChange(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.PutHash("a", 1)
	m.PutFailed("b")
	m.PutNativeSize("a", image.Pt(10, 10))

	set.gen = 2

	if _, ok := m.Hash("a"); ok {
		t.Error("Hash(a) ok = true after a generation change, want false (wiped)")
	}
	if m.Failed("b") {
		t.Error("Failed(b) = true after a generation change, want false (wiped)")
	}
	if _, ok := m.NativeSize("a"); ok {
		t.Error("NativeSize(a) ok = true after a generation change, want false (wiped)")
	}
}

func TestAdoptGeneration_KeepsFactsAcrossGenerationChange(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.PutHash("a", 1)
	m.PutFailed("b")
	m.PutNativeSize("a", image.Pt(10, 10))

	set.gen = 2
	m.AdoptGeneration()

	h, ok := m.Hash("a")
	if !ok || h != 1 {
		t.Errorf("Hash(a) = (%d, %v) after AdoptGeneration, want (1, true): adopt must keep entries", h, ok)
	}
	if !m.Failed("b") {
		t.Error("Failed(b) = false after AdoptGeneration, want true: adopt must keep entries")
	}
	if sz, ok := m.NativeSize("a"); !ok || sz != image.Pt(10, 10) {
		t.Errorf("NativeSize(a) = (%v, %v) after AdoptGeneration, want (%v, true)", sz, ok, image.Pt(10, 10))
	}
}

func TestAdoptGeneration_DoesNotWipeWhenGenerationDidNotChange(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.PutHash("a", 1)

	m.AdoptGeneration()

	if _, ok := m.Hash("a"); !ok {
		t.Error("Hash(a) ok = false after a no-op AdoptGeneration, want true")
	}
}

func TestClear_DropsFactsRegardlessOfGeneration(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.PutHash("a", 1)
	m.PutFailed("b")
	m.PutNativeSize("a", image.Pt(10, 10))

	m.Clear()

	if _, ok := m.Hash("a"); ok {
		t.Error("Hash(a) ok = true after Clear, want false")
	}
	if m.Failed("b") {
		t.Error("Failed(b) = true after Clear, want false")
	}
	if _, ok := m.NativeSize("a"); ok {
		t.Error("NativeSize(a) ok = true after Clear, want false")
	}
}

func TestSetDistance_ClampsAndReportsChange(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		want    int
		changed bool
	}{
		{"below zero clamps to zero", -1, 0, true},
		{"far below zero still clamps to zero", -1000, 0, true},
		{"above max clamps to MaxDistance", MaxDistance + 1, MaxDistance, true},
		{"far above max still clamps to MaxDistance", 1000, MaxDistance, true},
		{"at the max is unclamped", MaxDistance, MaxDistance, true},
		{"at zero is unclamped", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(newFakeSet(1, 1))
			// Force a distance the test case is guaranteed to differ
			// from (or match), independent of the package default.
			m.dist = imaging.DuplicateMaxDistance

			changed := m.SetDistance(tt.n)

			if got := m.Distance(); got != tt.want {
				t.Errorf("Distance() = %d after SetDistance(%d), want %d", got, tt.n, tt.want)
			}
			if changed != tt.changed {
				t.Errorf("SetDistance(%d) = %v, want %v", tt.n, changed, tt.changed)
			}
		})
	}
}

func TestSetDistance_ReturnsFalseWhenUnchanged(t *testing.T) {
	m := New(newFakeSet(1, 1))
	m.dist = 6

	if changed := m.SetDistance(6); changed {
		t.Error("SetDistance(6) on a model already at 6 = true, want false")
	}
	if changed := m.SetDistance(-5); !changed {
		t.Error("SetDistance(-5) on a model at 6 = false, want true (clamps to 0)")
	}
	if changed := m.SetDistance(0); changed {
		t.Error("SetDistance(0) on a model already at 0 = true, want false")
	}
}

func TestNativeSizeAt_OutOfRange(t *testing.T) {
	m := New(newFakeSet(2, 1))

	if _, _, ok := m.NativeSizeAt(-1); ok {
		t.Error("NativeSizeAt(-1) ok = true, want false")
	}
	if _, _, ok := m.NativeSizeAt(2); ok {
		t.Error("NativeSizeAt(2) ok = true for an index == Count(), want false")
	}
}

func TestNativeSizeAt_Unprobed(t *testing.T) {
	m := New(newFakeSet(2, 1))

	if _, _, ok := m.NativeSizeAt(0); ok {
		t.Error("NativeSizeAt(0) ok = true before any PutNativeSize, want false")
	}
}

func TestNativeSizeAt_NonPositiveEdgeIsNotOK(t *testing.T) {
	m := New(newFakeSet(2, 1))
	m.PutNativeSize("a", image.Pt(0, 10))
	m.PutNativeSize("b", image.Pt(10, 0))

	if _, _, ok := m.NativeSizeAt(0); ok {
		t.Error("NativeSizeAt(0) ok = true for a zero-width stored size, want false")
	}
	if _, _, ok := m.NativeSizeAt(1); ok {
		t.Error("NativeSizeAt(1) ok = true for a zero-height stored size, want false")
	}
}

func TestNativeSizeAt_ReturnsWidthAndHeight(t *testing.T) {
	m := New(newFakeSet(2, 1))
	m.PutNativeSize("a", image.Pt(1920, 1080))

	w, h, ok := m.NativeSizeAt(0)
	if !ok || w != 1920 || h != 1080 {
		t.Errorf("NativeSizeAt(0) = (%d, %d, %v), want (1920, 1080, true)", w, h, ok)
	}
}

// TestEnsureMapsLocked_RepopulatesNilMaps exercises the defensive
// nil-map branches AdoptGeneration relies on: New always allocates the
// three maps, so nothing in a normal run leaves them nil, but the guard
// must still hold for any future caller that skips New's zeroing.
func TestEnsureMapsLocked_RepopulatesNilMaps(t *testing.T) {
	m := New(newFakeSet(1, 1))
	m.hashes = nil
	m.hashFailed = nil
	m.native = nil

	m.AdoptGeneration()

	if m.hashes == nil || m.hashFailed == nil || m.native == nil {
		t.Error("AdoptGeneration left a nil map after ensureMapsLocked, want all three allocated")
	}
}

func TestOnChange_FiresObserversInRegistrationOrder(t *testing.T) {
	m := New(newFakeSet(1, 1))
	var order []int
	m.OnChange(func() { order = append(order, 1) })
	m.OnChange(func() { order = append(order, 2) })
	m.OnChange(func() { order = append(order, 3) })

	m.Notify()

	if want := []int{1, 2, 3}; !slices.Equal(order, want) {
		t.Errorf("fire order = %v, want %v", order, want)
	}
}

func TestNotify_WithNoObserversIsANoOp(t *testing.T) {
	m := New(newFakeSet(1, 1))
	m.Notify() // must not panic
}
