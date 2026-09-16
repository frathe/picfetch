// Package heicdecode defines PicFetch's fail-closed boundary to the isolated
// HEIC/HEIF decoder helper. It contains no codec or rich container parser.
package heicdecode

import (
	"errors"
	"fmt"
	"time"
)

const (
	hardMaxInputBytes      int64 = 64 * 1024 * 1024
	hardMaxPixels          int64 = 64_000_000
	hardMaxOutputBytes           = hardMaxPixels * 4
	hardMaxMetadataBytes         = 64 * 1024
	hardMaxDiagnosticBytes       = 4096
	hardMaxDuration              = time.Minute
)

var ErrInvalidLimits = errors.New("invalid HEIC decoder limits")

// Limits is the complete finite resource contract for one helper request.
// WASMMemoryBytes is an enforced linear-memory ceiling. OSProcessBytes is the
// requested native-process budget; Ready separately reports the hard OS memory
// control actually installed. macOS explicitly permits an absent hard native
// cap, while retaining bounded WASM and transport and verified OS sandboxing.
type Limits struct {
	Timeout            time.Duration
	WASMMemoryBytes    int64
	OSProcessBytes     int64
	MaxInputBytes      int64
	MaxPixels          int64
	MaxOutputBytes     int64
	MaxMetadataBytes   uint32
	MaxDiagnosticBytes uint32
	MaxThreads         uint32 // Decoder threads; the native Go host has its own finite thread ceiling.
	MaxLiveJobs        uint32
}

// DefaultLimits returns the initial engineering limits. A positive user input
// ceiling can only lower the hard HEIC ceiling, never raise it.
func DefaultLimits(userMaxInputBytes int64) Limits {
	input := hardMaxInputBytes
	if userMaxInputBytes > 0 && userMaxInputBytes < input {
		input = userMaxInputBytes
	}
	return Limits{
		Timeout:            hardMaxDuration,
		WASMMemoryBytes:    1 * 1024 * 1024 * 1024,
		OSProcessBytes:     2 * 1024 * 1024 * 1024,
		MaxInputBytes:      input,
		MaxPixels:          hardMaxPixels,
		MaxOutputBytes:     hardMaxOutputBytes,
		MaxMetadataBytes:   hardMaxMetadataBytes,
		MaxDiagnosticBytes: hardMaxDiagnosticBytes,
		MaxThreads:         1,
		MaxLiveJobs:        1,
	}
}

// Validate rejects relaxed, absent, or internally inconsistent limits.
func (l Limits) Validate() error {
	if l.Timeout <= 0 || l.Timeout > hardMaxDuration ||
		l.WASMMemoryBytes <= 0 || l.WASMMemoryBytes > 1*1024*1024*1024 || l.WASMMemoryBytes%(64*1024) != 0 ||
		l.OSProcessBytes <= 0 || l.OSProcessBytes > 2*1024*1024*1024 || l.WASMMemoryBytes > l.OSProcessBytes ||
		l.MaxInputBytes <= 0 || l.MaxInputBytes > hardMaxInputBytes ||
		l.MaxPixels <= 0 || l.MaxPixels > hardMaxPixels ||
		l.MaxOutputBytes <= 0 || l.MaxOutputBytes > hardMaxOutputBytes ||
		l.MaxMetadataBytes == 0 || l.MaxMetadataBytes > hardMaxMetadataBytes ||
		l.MaxDiagnosticBytes == 0 || l.MaxDiagnosticBytes > hardMaxDiagnosticBytes ||
		l.MaxThreads != 1 || l.MaxLiveJobs != 1 {
		return fmt.Errorf("%w: every resource must have a supported finite ceiling", ErrInvalidLimits)
	}
	return nil
}
