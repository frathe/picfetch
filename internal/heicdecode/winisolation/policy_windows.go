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
	if err := VerifyAppContainer(); err != nil {
		return err
	}
	// A null handle queries the caller's immediate job even in a nested job.
	return verifyJob(0, limits)
}

// VerifyAppContainer checks the exact helper identity and zero capabilities.
// The launcher separately checks privileged loopback configuration before
// creation; the worker can recheck this token after its traffic probes.
func VerifyAppContainer() error {
	token := windows.GetCurrentProcessToken()
	var isContainer uint32
	var length uint32
	if err := windows.GetTokenInformation(token, tokenIsAppContainer, (*byte)(unsafe.Pointer(&isContainer)), uint32(unsafe.Sizeof(isContainer)), &length); err != nil {
		return fmt.Errorf("query HEIC AppContainer token: %w", err)
	}
	if isContainer != 1 {
		return errors.New("HEIC helper is not an AppContainer process")
	}
	// TOKEN_GROUPS and TOKEN_APPCONTAINER_INFORMATION contain native pointers.
	// A byte array on the Go stack has no pointer-alignment guarantee.
	var buffer [512]uintptr
	data := (*byte)(unsafe.Pointer(&buffer[0]))
	if err := windows.GetTokenInformation(token, tokenCapabilities, data, uint32(unsafe.Sizeof(buffer)), &length); err != nil {
		return fmt.Errorf("query HEIC token capabilities: %w", err)
	}
	if length < 4 || (*windows.Tokengroups)(unsafe.Pointer(&buffer[0])).GroupCount != 0 {
		return errors.New("HEIC helper has unexpected AppContainer capabilities")
	}
	if err := windows.GetTokenInformation(token, tokenAppContainerSID, data, uint32(unsafe.Sizeof(buffer)), &length); err != nil {
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
	return nil
}

// The parent checks the native loopback exemption list on every launch. Windows
// denies this query to the restricted helper; no capability is added to it.
func verifyLoopbackIsolation(expected *windows.SID) (resultErr error) {
	query := windows.NewLazySystemDLL("Firewallapi.dll").NewProc("NetworkIsolationGetAppContainerConfig")
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	getHeap, free := kernel.NewProc("GetProcessHeap"), kernel.NewProc("HeapFree")
	for _, proc := range []*windows.LazyProc{query, getHeap, free} {
		if err := proc.Find(); err != nil {
			return err
		}
	}
	heap, _, heapErr := getHeap.Call()
	if heap == 0 {
		return fmt.Errorf("get HEIC isolation query heap: %w", heapErr)
	}
	var count uint32
	var entries *windows.SIDAndAttributes
	code, _, _ := query.Call(uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&entries)))
	if err := validateLoopbackQuery(code, count, entries); err != nil {
		return err
	}
	items := unsafe.Slice(entries, count)
	// The API allocates both the array and each SID on the process heap.
	// https://learn.microsoft.com/windows/win32/api/networkisolation/nf-networkisolation-networkisolationgetappcontainerconfig
	defer func() {
		for _, item := range items {
			if item.Sid != nil {
				if ok, _, _ := free.Call(heap, 0, uintptr(unsafe.Pointer(item.Sid))); ok == 0 {
					resultErr = errors.Join(resultErr, errors.New("free HEIC loopback exemption SID"))
				}
			}
		}
		if entries != nil {
			if ok, _, _ := free.Call(heap, 0, uintptr(unsafe.Pointer(entries))); ok == 0 {
				resultErr = errors.Join(resultErr, errors.New("free HEIC loopback exemption array"))
			}
		}
	}()
	return validateLoopbackExemptions(expected, items)
}

func validateLoopbackExemptions(expected *windows.SID, items []windows.SIDAndAttributes) error {
	for _, item := range items {
		if item.Sid == nil {
			return errors.New("HEIC loopback exemption query returned an invalid SID")
		}
		if expected.Equals(item.Sid) {
			return errors.New("HEIC AppContainer has a loopback exemption")
		}
	}
	return nil
}

func validateLoopbackQuery(code uintptr, count uint32, entries *windows.SIDAndAttributes) error {
	if code != 0 {
		return fmt.Errorf("query HEIC loopback exemptions: %w", windows.Errno(code))
	}
	if count != 0 && entries == nil {
		return errors.New("HEIC loopback exemption query returned no entries")
	}
	return nil
}
