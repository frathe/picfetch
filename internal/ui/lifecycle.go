package ui

import "sync/atomic"

// revision is a monotonically increasing identity for state observed by
// background work. Its zero value is ready to use.
type revision struct {
	value atomic.Uint64
}

func (r *revision) advance() uint64 {
	return r.value.Add(1)
}

func (r *revision) current() uint64 {
	return r.value.Load()
}

func (r *revision) matches(value uint64) bool {
	return r.current() == value
}
