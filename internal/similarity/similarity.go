// Package similarity analyzes local images in an isolated native worker.
package similarity

import "context"

// Item retains the source identity across representations, layout and browsing.
type Item struct {
	Path       string
	Size       int64
	ModifiedNS int64
	SHA256     string
	Error      string    `json:",omitempty"`
	Embedding  []float32 `json:",omitempty"`
	Cohort     string    `json:",omitempty"`
	Position   []float32 `json:",omitempty"`
	Thumbnail  string    `json:",omitempty"`
	Preview    []byte    `json:",omitempty"`
}

// Event is an immutable analysis snapshot. Complete includes layout delivery.
type Event struct {
	OfflineVerified           bool
	Items                     []Item
	Total, Successful, Failed int
	Stage                     string
	Complete                  bool
}

// Provider runs off the UI goroutine; callbacks are serialized on that worker.
type Provider func(context.Context, []string, func(Event)) error
