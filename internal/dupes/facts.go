package dupes

import "image"

// FactWriter binds results to the file set and model reset that admitted the
// work. Capture it before reading a source; never recapture on completion.
// Its zero value rejects every mutation. Copies are safe across workers.
type FactWriter struct {
	model             *Model
	generation, reset uint64
}

// CaptureFacts starts or joins the current fact namespace. Call it at work
// admission, after AdoptGeneration when established facts should survive.
func (m *Model) CaptureFacts() FactWriter {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wipeIfStaleLocked(m.set.Snapshot().Generation())
	return FactWriter{model: m, generation: m.gen, reset: m.reset}
}

// currentLocked and each mutation share the model lock. A stale writer never
// wipes or adopts another namespace, including after a same-generation Clear.
func (w FactWriter) currentLocked() bool {
	return w.generation == w.model.gen && w.reset == w.model.reset &&
		w.generation == w.model.set.Snapshot().Generation()
}

// Current reports whether work or queued delivery still belongs to this model.
func (w FactWriter) Current() bool {
	if w.model == nil {
		return false
	}
	w.model.mu.Lock()
	defer w.model.mu.Unlock()
	return w.currentLocked()
}

// PutHash conditionally records a dHash and clears its previous failure.
func (w FactWriter) PutHash(key string, hash uint64) bool {
	if w.model == nil {
		return false
	}
	m := w.model
	m.mu.Lock()
	defer m.mu.Unlock()
	if !w.currentLocked() {
		return false
	}
	if old, ok := m.hashes[key]; !ok || old != hash {
		m.factRevision++
	}
	m.hashes[key] = hash
	delete(m.hashFailed, key)
	return true
}

// PutFailed conditionally records a source failure, allowing a replacement to
// retry even if the previous source's failed read finishes later.
func (w FactWriter) PutFailed(key string) bool {
	if w.model == nil {
		return false
	}
	m := w.model
	m.mu.Lock()
	defer m.mu.Unlock()
	if !w.currentLocked() {
		return false
	}
	m.hashFailed[key] = struct{}{}
	return true
}

// PutNativeSize conditionally records EXIF-oriented size, clamping negative
// edges to zero. Known-zero probes retain their existing no-retry semantics.
func (w FactWriter) PutNativeSize(key string, size image.Point) bool {
	if w.model == nil {
		return false
	}
	m := w.model
	m.mu.Lock()
	defer m.mu.Unlock()
	if !w.currentLocked() {
		return false
	}
	size = image.Pt(max(size.X, 0), max(size.Y, 0))
	if old, ok := m.native[key]; !ok || old != size {
		m.factRevision++
	}
	m.native[key] = size
	return true
}
