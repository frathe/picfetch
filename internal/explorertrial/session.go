// Package explorertrial records metadata from an explicitly requested native
// explorer trial. Source paths, pixels and representations stay out of its trace.
package explorertrial

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/frathe/picfetch/internal/similarity"
)

// Record describes a causal boundary without copying the source-bearing Event.
type Record struct {
	Kind                              string
	Run                               int
	Event                             int
	Seconds                           float64
	Total, Successful, Failed, Reused int
	Stage                             string
	Complete, OfflineVerified         bool
	Measurements                      similarity.Measurements
	InputSHA256                       string `json:",omitempty"`
	QueueSeconds, ApplySeconds        float64
	Outcome                           string   `json:",omitempty"`
	Surface                           string   `json:",omitempty"`
	VisibleTotal                      int      `json:",omitempty"`
	VisibleSHA256                     string   `json:",omitempty"`
	Map                               *MapView `json:",omitempty"`
}

// MapView measures the foreground map in Fyne logical units, not physical
// pixels or paint latency. Piles includes cohorts hidden by tag filters.
type MapView struct {
	Piles                                              int
	Zoom, MinimumZoom, CenterX, CenterY, Width, Height float32
}

// Analysis distinguishes a delivered map from an observed worker exit.
type Analysis struct {
	Record
	MapApplied bool
	InputTotal int
}

// Summary describes technical collection only; it contains no quality verdict.
type Summary struct {
	Collected     bool
	Seconds       float64
	ModelRevision string
	Analyses      []Analysis
}

// Session owns one native trial's evidence.
// It starts no workers. Callers serialize UI work themselves and join all
// producers before Close; its mutex orders records from UI and analysis.
type Session struct {
	mu       sync.Mutex
	file     *os.File
	encoder  *json.Encoder
	start    time.Time
	run      int
	event    int
	err      error
	closed   bool
	dir      string
	analyses []Analysis
	rejected bool
}

// New reserves a new evidence directory.
func New(dir string) (*Session, error) {
	if err := os.Mkdir(dir, 0700); err != nil {
		return nil, fmt.Errorf("trial directory must be new: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Session{file: f, encoder: json.NewEncoder(f), start: time.Now(), dir: dir}, nil
}

// Begin captures the exact analysis input identities as a digest.
func (s *Session) Begin(paths []string) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.run++
	r := Record{Kind: "analysis-started", Run: s.run, Total: len(paths), InputSHA256: sourceDigest(paths)}
	s.analyses = append(s.analyses, Analysis{Record: r, InputTotal: len(paths)})
	s.write(r)
	return s.run
}

func sourceDigest(paths []string) string {
	paths = slices.Clone(paths)
	slices.Sort(paths)
	digest := sha256.New()
	_ = json.NewEncoder(digest).Encode(paths) // Strings and a hash writer cannot fail to encode.
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func eventRecord(kind string, run int, e similarity.Event) Record {
	return Record{Kind: kind, Run: run, Total: e.Total, Successful: e.Successful,
		Failed: e.Failed, Reused: e.Reused, Stage: e.Stage, Complete: e.Complete,
		OfflineVerified: e.OfflineVerified, Measurements: e.Measurements}
}

// Received records worker delivery, before admission to the UI queue, and
// returns its session-unique identity for subsequent application records.
func (s *Session) Received(run int, e similarity.Event) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.event++
	r := eventRecord("worker-event", run, e)
	r.Event = s.event
	s.write(r)
	return r.Event
}

// Applied records actual map construction on UI, separately from paint.
func (s *Session) Applied(run, event int, e similarity.Event, received, started time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := eventRecord("map-applied", run, e)
	r.Event = event
	r.QueueSeconds, r.ApplySeconds = started.Sub(received).Seconds(), time.Since(started).Seconds()
	a := &s.analyses[run-1]
	r.InputSHA256, r.Outcome = a.InputSHA256, a.Outcome
	if e.Complete {
		paths := make([]string, 0, len(e.Items))
		successful, failed := 0, 0
		for _, item := range e.Items {
			paths = append(paths, item.Path)
			if item.Error == "" {
				successful++
			} else {
				failed++
			}
		}
		if e.Total != a.InputTotal || e.Total == 0 || e.Successful != successful || e.Failed != failed || len(e.Items) != e.Total || sourceDigest(paths) != a.InputSHA256 {
			r.Complete = false
		}
	}
	a.Record, a.MapApplied = r, true
	s.write(r)
}

// Exited observes the provider returning after its child exits.
func (s *Session) Exited(run int, err error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	outcome := "completed"
	if err != nil {
		outcome = "failed"
	}
	if errors.Is(err, context.Canceled) {
		outcome = "canceled"
	}
	s.analyses[run-1].Outcome = outcome
	s.write(Record{Kind: "worker-exited", Run: run, Outcome: outcome})
}

// Action records a source-free viewer interaction, with an optional item count.
func (s *Session) Action(run int, kind string, count int) {
	if s == nil || run <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.write(Record{Kind: kind, Run: run, Total: count})
}

// Presented records the foreground browsing surface after UI state changes.
// Paths describe the actual grid results or current image target, not a saved
// cohort. Only their count and unordered identity digest enter the trace.
// This observes UI state, not framebuffer paint or image-load completion.
func (s *Session) Presented(run, event int, kind, surface string, paths []string, view *MapView) {
	if s == nil || run <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := Record{Kind: kind, Run: run, Event: event, Surface: surface, VisibleTotal: len(paths)}
	if surface == "map" {
		r.Map = view
	}
	if len(paths) > 0 {
		r.VisibleSHA256 = sourceDigest(paths)
	}
	s.write(r)
}

func (s *Session) write(r Record) {
	if s.closed || s.err != nil {
		return
	}
	r.Seconds = time.Since(s.start).Seconds()
	s.err = s.encoder.Encode(r)
}

// Close finalizes evidence after analysis workers and UI delivery have stopped.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		collected := len(s.analyses) > 0 && !s.rejected
		for _, a := range s.analyses {
			if a.Outcome == "" {
				collected = false
			}
		}
		if collected {
			a := s.analyses[len(s.analyses)-1]
			collected = a.MapApplied && a.Complete && a.OfflineVerified && a.Outcome == "completed"
		}
		s.write(Record{Kind: "session-closed", Complete: collected})
		s.closed = true
		s.err = errors.Join(s.err, s.file.Close())
		if s.err == nil {
			data, err := json.MarshalIndent(Summary{Collected: collected, Seconds: time.Since(s.start).Seconds(), ModelRevision: similarity.ModelRevision, Analyses: s.analyses}, "", "  ")
			if err == nil {
				err = os.WriteFile(filepath.Join(s.dir, "session.json.part"), data, 0600)
			}
			if err == nil {
				err = os.Rename(filepath.Join(s.dir, "session.json.part"), filepath.Join(s.dir, "session.json"))
			}
			s.err = err
		}
		if !collected {
			s.err = errors.Join(s.err, errors.New("native trial has no completed map with observed worker exit"))
		}
	}
	return s.err
}

// Reject permanently marks an incomplete collection, without retaining a path.
func (s *Session) Reject(kind string, count int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rejected = true
	s.write(Record{Kind: kind, Total: count})
}

// Identity isolates Fyne preferences and session storage from ordinary runs.
// New rejects reuse of the corresponding evidence directory.
func Identity(dir string) string {
	absolute, _ := filepath.Abs(dir)
	return fmt.Sprintf("io.picfetch.explorer-trial.%x", sha256.Sum256([]byte(absolute)))
}
