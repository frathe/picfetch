// Package locationtrial records explicitly requested native Location Map trials.
// It contains no source paths/pixels and never measures paint latency.
package locationtrial

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

type Stage struct {
	Kind          string `json:"kind"`
	StartNS       int64  `json:"start_ns"`
	EndNS         int64  `json:"end_ns"`
	PreparationNS int64  `json:"preparation_ns"`
	ScanNS        int64  `json:"scan_ns"`
	Complete      bool   `json:"complete"`
}

type State struct {
	Sequence int            `json:"sequence"`
	Images   int            `json:"images"`
	Formats  map[string]int `json:"formats"`
	Stages   []Stage        `json:"stages"`
	Ready    bool           `json:"ready"`
	Active   bool           `json:"active"`
	Visible  bool           `json:"visible"`
}

// Recorder coalesces UI snapshots onto one tracked filesystem worker.
type Recorder struct {
	mu               sync.Mutex
	pending          *State
	dir              string
	wake, stop, done chan struct{}
	err              error
	closed           bool
}

func New(dir string) (*Recorder, error) {
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, err
	}
	r := &Recorder{dir: dir, wake: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	go r.run()
	return r, nil
}

func (r *Recorder) Publish(state State) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	state.Formats, state.Stages = maps.Clone(state.Formats), slices.Clone(state.Stages)
	r.pending = &state
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// Stop ends admission immediately; the worker flushes its last accepted state.
func (r *Recorder) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed {
		r.closed = true
		close(r.stop)
	}
}

func (r *Recorder) Wait() error {
	if r == nil {
		return nil
	}
	<-r.done
	return r.err
}

func (r *Recorder) flush() {
	r.mu.Lock()
	state := r.pending
	r.pending = nil
	r.mu.Unlock()
	if state != nil {
		err := r.write(*state)
		if r.err == nil {
			r.err = err
		}
	}
}

func (r *Recorder) run() {
	defer close(r.done)
	for {
		select {
		case <-r.wake:
			r.flush()
		case <-r.stop:
			r.flush()
			return
		}
	}
}

func (r *Recorder) write(state State) error {
	file, err := os.CreateTemp(r.dir, ".state-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	encodeErr := json.NewEncoder(file).Encode(state)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(encodeErr, syncErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), filepath.Join(r.dir, "state.json"))
}
