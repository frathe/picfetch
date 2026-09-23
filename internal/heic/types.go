// Package heic owns the optional system decoder boundary and its capability
// observations. Native parsing is confined to a cancellable child process.
package heic

import (
	"context"
	"errors"
	"time"
)

// Revision changes when the adapter or its representative pixel checks change.
const Revision = 3

var (
	ErrUnavailable = errors.New("system HEIC decoder is unavailable")
	ErrUnsupported = errors.New("unsupported HEIC image")
	ErrInvalid     = errors.New("invalid HEIC image")
	ErrProtocol    = errors.New("invalid HEIC worker response")
)

// Request supplies the caller's captured resource policy. Pixels false requests
// oriented dimensions and metadata without returning the pixel plane.
type Request struct {
	Pixels          bool
	MaxEncodedBytes int64
	MaxPixels       int64
}

// Result contains a designated primary still, with all transforms applied once.
// Pixels are the native decoder's 8-bit straight-alpha RGBA rendition. EXIF is
// optional bounded metadata; consumers must not apply its orientation again.
type Result struct {
	Width, Height int
	Stride        int
	Pixels        []byte
	EXIF          []byte
	Provider      string
}

// Backend is the sole instance-owned substitution boundary for system work.
// Check must verify real 8-bit and 10-bit pixels, not just decoder registration.
type Backend interface {
	Check(context.Context) error
	Read(context.Context, []byte, Request) (Result, error)
}

// Identity identifies the environment in which an observation is reusable.
// OSVersion is an OS identity, not PicFetch's unrelated application version.
type Identity struct {
	OS, Architecture, OSVersion string
	Revision                    int
}

// Observation records only a completed available or unavailable check.
type Observation struct {
	Identity  Identity
	CheckedAt time.Time
	Available bool
}

// Matches rejects incomplete or incompatible persisted records.
func (o Observation) Matches(identity Identity) bool {
	return o.Identity == identity && identity.OS != "" && identity.Architecture != "" &&
		identity.OSVersion != "" && identity.Revision > 0 && !o.CheckedAt.IsZero()
}

// Snapshot binds an operation to the capability generation captured at admission.
// A later support check never mutates that operation's policy.
type Snapshot struct {
	Backend    Backend
	Available  bool
	Generation uint64
}
