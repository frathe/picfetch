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
	Tags       []string  `json:",omitempty"`
	Cohort     string    `json:",omitempty"`
	Position   []float32 `json:",omitempty"`
	Thumbnail  string    `json:",omitempty"`
	Preview    []byte    `json:",omitempty"`
}

// Event is an immutable analysis snapshot. Complete includes layout delivery.
type Event struct {
	OfflineVerified           bool
	Items                     []Item
	Merges                    []CohortMerge `json:",omitempty"`
	Total, Successful, Failed int
	Reused                    int
	CacheWarning              string `json:",omitempty"`
	Stage                     string
	Complete                  bool
	Measurements              Measurements
}

// Measurements are cumulative worker wall times in seconds, copied per event.
// Elapsed includes previous event delivery, but excludes delivery of this event
// and process startup/shutdown. Grouping includes the four named sub-stages;
// those sub-stages must not be added to it when totaling work.
type Measurements struct {
	ElapsedSeconds, SetupSeconds, ModelSeconds          float64
	DecodeSeconds, EncodeSeconds, PreviewSeconds        float64
	CacheSeconds, TagSeconds, GroupingSeconds           float64
	ReductionSeconds, HDBSCANSeconds, ProjectionSeconds float64
	HierarchySeconds                                    float64
	InferenceAttempts, Publications                     int
}

// CohortMerge joins base cohorts in increasing content distance. Cutting a
// prefix of this hierarchy produces nested broader groups without reanalysis.
type CohortMerge struct {
	Left, Right string
}

// Control carries the latest automatic-update setting and an optional rebuild
// request. Rebuilds use the representations already retained by the worker.
type Control struct {
	Automatic bool
	Update    bool
}

// Provider runs off the UI goroutine; callbacks are serialized on that worker.
type Provider func(context.Context, []string, <-chan Control, func(Event)) error
