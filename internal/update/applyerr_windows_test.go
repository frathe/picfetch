//go:build windows

package update

import (
	"os"
	"syscall"
	"testing"
)

func TestClassifyApplyError_WindowsErrno(t *testing.T) {
	cases := map[syscall.Errno]FailureReason{
		syscall.Errno(5):   ReasonAccessDenied,     // ERROR_ACCESS_DENIED
		syscall.Errno(225): ReasonVirusBlocked,     // ERROR_VIRUS_INFECTED
		syscall.Errno(226): ReasonVirusBlocked,     // ERROR_VIRUS_DELETED
		syscall.Errno(32):  ReasonSharingViolation, // ERROR_SHARING_VIOLATION
		syscall.Errno(33):  ReasonSharingViolation, // ERROR_LOCK_VIOLATION
	}
	for errno, want := range cases {
		err := &ApplyError{Op: "copy", Path: "p", Err: &os.LinkError{Op: "rename", Err: errno}}
		if got := ClassifyApplyError(err); got != want {
			t.Errorf("errno %d = %q, want %q", uintptr(errno), got, want)
		}
	}
}
