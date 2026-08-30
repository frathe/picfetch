package update

import (
	"errors"
	"fmt"
	"io/fs"
)

// FailureReason categorizes why Apply failed, for the next-launch failure
// report. It is stored as a plain string (its value is written verbatim
// into a JSON cache file), so the constants below are the only valid values.
type FailureReason string

const (
	ReasonAccessDenied     FailureReason = "access-denied"
	ReasonVirusBlocked     FailureReason = "virus-blocked"
	ReasonSharingViolation FailureReason = "sharing-violation"
	ReasonUnknown          FailureReason = "unknown"
)

// ApplyError wraps a failure from one step of Apply with the step name and
// the path it was operating on, so the next-launch failure report can name
// both without re-deriving them from the wrapped error's own message.
type ApplyError struct {
	Op   string // "rename", "copy", "verify", "restore", "relaunch"
	Path string
	Err  error
}

// Error implements the error interface.
func (e *ApplyError) Error() string {
	return fmt.Sprintf("update apply: %s %s: %v", e.Op, e.Path, e.Err)
}

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *ApplyError) Unwrap() error {
	return e.Err
}

// ClassifyApplyError maps a failed Apply to the reason PicFetch reports on
// the next launch. Windows errno values win over the portable
// fs.ErrPermission check because Defender's virus and sharing denials are
// not permission errors.
func ClassifyApplyError(err error) FailureReason {
	if err == nil {
		return ReasonUnknown
	}
	if reason, ok := classifyPlatform(err); ok {
		return reason
	}
	if errors.Is(err, fs.ErrPermission) {
		return ReasonAccessDenied
	}
	return ReasonUnknown
}
