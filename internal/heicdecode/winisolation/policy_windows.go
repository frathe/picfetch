//go:build windows && (amd64 || arm64)

package winisolation

import (
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// TOKEN_INFORMATION_CLASS values from the Windows SDK; x/sys does not expose
// these three declarations. https://learn.microsoft.com/windows/win32/api/winnt/ne-winnt-token_information_class
const (
	tokenIsAppContainer  = 29
	tokenCapabilities    = 30
	tokenAppContainerSID = 31
)

func jobLimits(limits heicdecode.Limits) windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION {
	return windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			PerJobUserTimeLimit: int64((limits.Timeout + 100*time.Nanosecond - 1) / (100 * time.Nanosecond)), // Windows counts 100-nanosecond units.
			LimitFlags: windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY |
				windows.JOB_OBJECT_LIMIT_JOB_MEMORY | windows.JOB_OBJECT_LIMIT_JOB_TIME | windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			ActiveProcessLimit: 1,
		},
		ProcessMemoryLimit: uintptr(limits.OSProcessBytes), JobMemoryLimit: uintptr(limits.OSProcessBytes),
	}
}

func verifyJob(job windows.Handle, limits heicdecode.Limits) error {
	var actual windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	// x/sys accepts a uintptr here, so its lazy DLL resolution cannot keep a
	// Go stack address valid if the stack grows before the native call.
	var pinned runtime.Pinner
	pinned.Pin(&actual)
	defer pinned.Unpin()
	if err := windows.QueryInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&actual)), uint32(unsafe.Sizeof(actual)), nil); err != nil {
		return fmt.Errorf("query HEIC job limits: %w", err)
	}
	wanted := jobLimits(limits)
	if actual.BasicLimitInformation.LimitFlags != wanted.BasicLimitInformation.LimitFlags ||
		actual.BasicLimitInformation.ActiveProcessLimit != 1 ||
		actual.BasicLimitInformation.PerJobUserTimeLimit != wanted.BasicLimitInformation.PerJobUserTimeLimit ||
		actual.ProcessMemoryLimit != wanted.ProcessMemoryLimit || actual.JobMemoryLimit != wanted.JobMemoryLimit {
		return errors.New("HEIC Job Object limits do not match the requested boundary")
	}
	return nil
}

// Verify checks the current process's kernel token and immediate Job Object.
// The worker calls this before performing denial probes or reading any input.
func Verify(limits heicdecode.Limits) error {
	if err := limits.Validate(); err != nil {
		return err
	}
	token := windows.GetCurrentProcessToken()
	var isContainer uint32
	var length uint32
	if err := windows.GetTokenInformation(token, tokenIsAppContainer, (*byte)(unsafe.Pointer(&isContainer)), uint32(unsafe.Sizeof(isContainer)), &length); err != nil {
		return fmt.Errorf("query HEIC AppContainer token: %w", err)
	}
	if isContainer != 1 {
		return errors.New("HEIC helper is not an AppContainer process")
	}
	var buffer [4096]byte
	if err := windows.GetTokenInformation(token, tokenCapabilities, &buffer[0], uint32(len(buffer)), &length); err != nil {
		return fmt.Errorf("query HEIC token capabilities: %w", err)
	}
	if length < 4 || (*windows.Tokengroups)(unsafe.Pointer(&buffer[0])).GroupCount != 0 {
		return errors.New("HEIC helper has unexpected AppContainer capabilities")
	}
	if err := windows.GetTokenInformation(token, tokenAppContainerSID, &buffer[0], uint32(len(buffer)), &length); err != nil {
		return fmt.Errorf("query HEIC AppContainer identity: %w", err)
	}
	if length < uint32(unsafe.Sizeof(uintptr(0))) {
		return errors.New("HEIC helper has no AppContainer SID")
	}
	actual := *(**windows.SID)(unsafe.Pointer(&buffer[0]))
	expected, err := containerSID(false)
	if err != nil {
		return err
	}
	defer func() { _ = windows.FreeSid(expected) }()
	matches := actual != nil && actual.Equals(expected)
	runtime.KeepAlive(buffer)
	if !matches {
		return errors.New("HEIC helper has an unexpected AppContainer identity")
	}
	// A null handle queries the caller's immediate job even in a nested job.
	return verifyJob(0, limits)
}
