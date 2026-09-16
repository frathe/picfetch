//go:build windows && (amd64 || arm64)

package winisolation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/heicdecode"
)

const (
	securityCapabilitiesAttribute = 0x00020009
	childProcessPolicyAttribute   = 0x0002000e
	childProcessRestricted        = 1
)

type securityCapabilities struct {
	AppContainerSID *windows.SID
	Capabilities    *windows.SIDAndAttributes
	CapabilityCount uint32
	Reserved        uint32
}

// Process owns one noninheritable job and process handle. Kill is nonblocking;
// Wait joins the process and closes the final job handle. Callers stop their
// pipe work separately, then Wait once admission can be released.
type Process struct {
	mu           sync.Mutex
	process, job windows.Handle
	wait         sync.Once
	waitErr      error
}

// Start creates the helper suspended inside a zero-capability AppContainer,
// assigns and verifies its private job, then resumes its sole initial thread.
// Only the three duplicated stdio handles are inherited. Every setup failure
// terminates and joins the still-owned process before returning.
func Start(executable string, args []string, stdio [3]*os.File, limits heicdecode.Limits) (_ *Process, resultErr error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(executable) {
		return nil, errors.New("HEIC helper path must be absolute")
	}
	sid, err := containerSID(true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = windows.FreeSid(sid) }()
	if err = verifyLoopbackIsolation(sid); err != nil {
		return nil, err
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			_ = windows.CloseHandle(job)
		}
	}()
	policy := jobLimits(limits)
	// SetInformationJobObject carries this address as uintptr through its
	// Go wrapper; pin it across lazy DLL resolution and the native call.
	var pinned runtime.Pinner
	pinned.Pin(&policy)
	defer pinned.Unpin()
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&policy)), uint32(unsafe.Sizeof(policy))); err != nil {
		return nil, fmt.Errorf("set HEIC job limits: %w", err)
	}
	if err = verifyJob(job, limits); err != nil {
		return nil, err
	}
	var handles [3]windows.Handle
	defer func() {
		for _, handle := range handles {
			if handle != 0 {
				_ = windows.CloseHandle(handle)
			}
		}
	}()
	for i, file := range stdio {
		if file == nil {
			return nil, errors.New("HEIC helper requires three explicit pipes")
		}
		if err = windows.DuplicateHandle(windows.CurrentProcess(), windows.Handle(file.Fd()), windows.CurrentProcess(), &handles[i], 0, true, windows.DUPLICATE_SAME_ACCESS); err != nil {
			return nil, err
		}
	}
	attributes, err := windows.NewProcThreadAttributeList(3)
	if err != nil {
		return nil, err
	}
	defer attributes.Delete()
	capability := securityCapabilities{AppContainerSID: sid}
	children := uint32(childProcessRestricted)
	if err = attributes.Update(securityCapabilitiesAttribute, unsafe.Pointer(&capability), unsafe.Sizeof(capability)); err != nil {
		return nil, err
	}
	if err = attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), unsafe.Sizeof(handles)); err != nil {
		return nil, err
	}
	if err = attributes.Update(childProcessPolicyAttribute, unsafe.Pointer(&children), unsafe.Sizeof(children)); err != nil {
		return nil, err
	}
	startup := windows.StartupInfoEx{ProcThreadAttributeList: attributes.List()}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.StdInput, startup.StdOutput, startup.StdErr = handles[0], handles[1], handles[2]
	path, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return nil, err
	}
	command, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{executable}, args...)))
	if err != nil {
		return nil, err
	}
	// AppContainer creation needs LOCALAPPDATA to locate its profile. Obtain
	// OS directory values directly, without inheriting the parent's environment
	// or DLL search path. Windows redirects LOCALAPPDATA for the child.
	environment, err := windows.UTF16FromString("GOMAXPROCS=1")
	if err != nil {
		return nil, err
	}
	profileDirectory, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		return nil, fmt.Errorf("locate local application data: %w", err)
	}
	localAppData, err := windows.UTF16FromString("LOCALAPPDATA=" + profileDirectory)
	if err != nil {
		return nil, err
	}
	environment = append(environment, localAppData...)
	windowsDirectory, err := windows.GetWindowsDirectory()
	if err != nil {
		return nil, fmt.Errorf("locate Windows directory: %w", err)
	}
	systemRoot, err := windows.UTF16FromString("SystemRoot=" + windowsDirectory)
	if err != nil {
		return nil, err
	}
	environment = append(environment, systemRoot...)
	environment = append(environment, 0)
	directory, err := windows.UTF16PtrFromString(filepath.Dir(executable))
	if err != nil {
		return nil, err
	}
	var information windows.ProcessInformation
	// The helper uses only inherited pipes. Avoid hidden-console initialization
	// and its console host inside this one-process AppContainer.
	flags := uint32(windows.CREATE_SUSPENDED | windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT | windows.DETACHED_PROCESS)
	if err = windows.CreateProcess(path, command, nil, nil, true, flags, &environment[0], directory, &startup.StartupInfo, &information); err != nil {
		return nil, fmt.Errorf("create AppContainer process: %w", err)
	}
	runtime.KeepAlive(capability)
	runtime.KeepAlive(handles)
	runtime.KeepAlive(children)
	defer func() { _ = windows.CloseHandle(information.Thread) }()
	defer func() {
		if resultErr != nil {
			_ = windows.TerminateProcess(information.Process, 1)
			_, _ = windows.WaitForSingleObject(information.Process, windows.INFINITE)
			_ = windows.CloseHandle(information.Process)
		}
	}()
	if err = windows.AssignProcessToJobObject(job, information.Process); err != nil {
		return nil, err
	}
	if err = verifyJob(job, limits); err != nil {
		return nil, err
	}
	count, err := windows.ResumeThread(information.Thread)
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, fmt.Errorf("HEIC initial thread had unexpected suspend count %d", count)
	}
	return &Process{process: information.Process, job: job}, nil
}

func (p *Process) Kill() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.job != 0 {
		_ = windows.TerminateJobObject(p.job, 1)
	}
}

func (p *Process) Wait() error {
	p.wait.Do(func() {
		state, err := windows.WaitForSingleObject(p.process, windows.INFINITE)
		if err != nil {
			p.waitErr = err
		} else if state != windows.WAIT_OBJECT_0 {
			p.waitErr = fmt.Errorf("HEIC process wait returned %d", state)
		} else {
			var code uint32
			if err = windows.GetExitCodeProcess(p.process, &code); err != nil {
				p.waitErr = err
			} else if code != 0 {
				p.waitErr = fmt.Errorf("HEIC process exited with code %d", code)
			}
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		// Kill-on-close is retained until the process has been observed to exit.
		_ = windows.CloseHandle(p.job)
		_ = windows.CloseHandle(p.process)
		p.job, p.process = 0, 0
	})
	return p.waitErr
}
