//go:build windows && (amd64 || arm64)

// Package winisolation owns the Windows HEIC helper's native privilege and
// process-family boundary. It contains no decoder or user-image handling.
package winisolation

import (
	"fmt"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const containerName = "PicFetch.HEICWorker.v2"

// Profile APIs return HRESULT, not the thread's last-error value. Each SID is
// allocated by Windows and belongs to the caller until FreeSid.
func containerSID(create bool) (*windows.SID, error) {
	name, err := windows.UTF16PtrFromString(containerName)
	if err != nil {
		return nil, err
	}
	library := windows.NewLazySystemDLL("userenv.dll")
	var sid *windows.SID
	if create {
		procedure := library.NewProc("CreateAppContainerProfile")
		if err = procedure.Find(); err != nil {
			return nil, err
		}
		result, _, _ := procedure.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(name)), 0, 0, uintptr(unsafe.Pointer(&sid)))
		runtime.KeepAlive(name)
		if int32(result) >= 0 {
			if sid == nil {
				return nil, fmt.Errorf("HEIC AppContainer returned no SID")
			}
			return sid, nil
		}
		if sid != nil {
			_ = windows.FreeSid(sid)
			sid = nil
		}
		if uint32(result) != 0x800700b7 {
			return nil, fmt.Errorf("create HEIC AppContainer: HRESULT %#x", uint32(result))
		}
	}
	procedure := library.NewProc("DeriveAppContainerSidFromAppContainerName")
	if err = procedure.Find(); err != nil {
		return nil, err
	}
	result, _, _ := procedure.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&sid)))
	runtime.KeepAlive(name)
	if int32(result) < 0 {
		if sid != nil {
			_ = windows.FreeSid(sid)
		}
		return nil, fmt.Errorf("derive HEIC AppContainer: HRESULT %#x", uint32(result))
	}
	if sid == nil {
		return nil, fmt.Errorf("HEIC AppContainer returned no SID")
	}
	return sid, nil
}

// PrepareExecutable grants only read/execute of a dedicated helper file and
// its containing directory to this AppContainer. Callers own this installation
// directory; no permission is inherited by other files or child directories.
// Launch does not change ACLs, so immutable packaged locations must already
// provide the necessary access or be staged into an owned helper directory.
func PrepareExecutable(executable string) error {
	if !filepath.IsAbs(executable) {
		return fmt.Errorf("HEIC helper path must be absolute")
	}
	sid, err := containerSID(true)
	if err != nil {
		return err
	}
	defer func() { _ = windows.FreeSid(sid) }()
	var pinned runtime.Pinner
	pinned.Pin(sid)
	defer pinned.Unpin()
	for _, path := range []string{filepath.Dir(executable), executable} {
		descriptor, getErr := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if getErr != nil {
			return getErr
		}
		existing, _, getErr := descriptor.DACL()
		if getErr != nil {
			return getErr
		}
		access := windows.EXPLICIT_ACCESS{
			AccessPermissions: windows.FILE_GENERIC_READ | windows.FILE_GENERIC_EXECUTE,
			AccessMode:        windows.GRANT_ACCESS, Inheritance: windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_UNKNOWN, TrusteeValue: windows.TrusteeValueFromSID(sid)},
		}
		acl, aclErr := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{access}, existing)
		if aclErr != nil {
			return aclErr
		}
		if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
			return err
		}
	}
	return nil
}
