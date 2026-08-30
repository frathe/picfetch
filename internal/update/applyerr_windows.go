//go:build windows

package update

import (
	"errors"
	"syscall"
)

// Only ERROR_ACCESS_DENIED is exported by stdlib syscall on Windows
// (types_windows.go). The rest are declared here, unexported, naming their
// Win32 symbol; ERROR_SHARING_VIOLATION and ERROR_LOCK_VIOLATION exist in
// the standard library only under the non-importable internal/syscall/windows.
const (
	errorSharingViolation = 32  // ERROR_SHARING_VIOLATION
	errorLockViolation    = 33  // ERROR_LOCK_VIOLATION
	errorVirusInfected    = 225 // ERROR_VIRUS_INFECTED
	errorVirusDeleted     = 226 // ERROR_VIRUS_DELETED
)

// classifyPlatform inspects err for a Windows errno, extracted with
// errors.As so it works through the *os.LinkError and *fs.PathError
// wrappers os and os/exec attach to syscall failures.
func classifyPlatform(err error) (FailureReason, bool) {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return "", false
	}
	switch errno {
	case syscall.ERROR_ACCESS_DENIED:
		return ReasonAccessDenied, true
	case errorVirusInfected, errorVirusDeleted:
		return ReasonVirusBlocked, true
	case errorSharingViolation, errorLockViolation:
		return ReasonSharingViolation, true
	default:
		return "", false
	}
}
