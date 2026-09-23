//go:build windows && (amd64 || arm64)

package heic

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// restrictWorker installs limits before reading input or activating native
// codecs. This bounds resources and forbids descendants; it is not a network
// or filesystem sandbox. The parent retains its cancellation/deadline kill.
func restrictWorker() error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	// Without KILL_ON_JOB_CLOSE, the assigned process keeps the job and its
	// limits alive after this handle closes, until the worker exits.
	defer func() { _ = windows.CloseHandle(job) }()
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY | windows.JOB_OBJECT_LIMIT_PROCESS_TIME | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	limits.BasicLimitInformation.PerProcessUserTimeLimit = 40 * 10_000_000
	limits.BasicLimitInformation.ActiveProcessLimit = 1
	limits.ProcessMemoryLimit = 4 * 1024 * 1024 * 1024
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return err
	}
	return windows.AssignProcessToJobObject(job, windows.CurrentProcess())
}
