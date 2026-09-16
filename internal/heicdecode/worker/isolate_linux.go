//go:build linux && !cgo && (amd64 || arm64)

package worker

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// This policy is qualified for the pure Go helper, not arbitrary libc paths.
// cgo and unsupported Linux architectures select the refusing implementation.
func isolate(limits heicdecode.Limits) (int64, error) {
	if err := limits.Validate(); err != nil {
		return 0, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := linuxLimits(limits); err != nil {
		return 0, err
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return 0, fmt.Errorf("HEIC no_new_privs: %w", err)
	}
	raw, err := linuxFilter(nativeLinuxCalls(), uint32(os.Getpid()))
	if err != nil {
		return 0, err
	}
	// Keep this conversion local to the HEIC policy; sharing similarity's
	// looser network-only installation would join unrelated trust boundaries.
	//noinspection DuplicatedCode
	filter := make([]unix.SockFilter, len(raw))
	for i, instruction := range raw {
		filter[i] = unix.SockFilter{Code: instruction.Op, Jt: instruction.Jt, Jf: instruction.Jf, K: instruction.K}
	}
	program := unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}
	result, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, unix.SECCOMP_FILTER_FLAG_TSYNC, uintptr(unsafe.Pointer(&program)))
	runtime.KeepAlive(filter)
	if errno != 0 {
		return 0, fmt.Errorf("HEIC seccomp: %w", errno)
	}
	if result != 0 {
		return 0, errors.New("HEIC seccomp thread synchronization failed")
	}
	// Observe the installed limit with a no-access reservation, without
	// committing or touching RAM. Success is a failed boundary, never readiness.
	if err = verifyLinuxAddressLimit(limits.OSProcessBytes); err != nil {
		return 0, err
	}
	return limits.OSProcessBytes, nil
}

func nativeLinuxCalls() linuxCalls {
	calls := linuxCalls{
		architecture: unix.AUDIT_ARCH_X86_64, threadFlags: 0xd0f00,
		clone: unix.SYS_CLONE, mmap: unix.SYS_MMAP, mprotect: unix.SYS_MPROTECT,
		futex: unix.SYS_FUTEX, fcntl: unix.SYS_FCNTL, tgkill: unix.SYS_TGKILL,
		ordinary: []uint32{
			unix.SYS_READ, unix.SYS_WRITE, unix.SYS_CLOSE,
			unix.SYS_MUNMAP, unix.SYS_MADVISE,
			unix.SYS_RT_SIGACTION, unix.SYS_RT_SIGPROCMASK, unix.SYS_RT_SIGRETURN, unix.SYS_SIGALTSTACK,
			unix.SYS_GETPID, unix.SYS_GETTID, unix.SYS_SCHED_YIELD, unix.SYS_NANOSLEEP, unix.SYS_CLOCK_GETTIME,
			unix.SYS_EPOLL_CREATE1, unix.SYS_EPOLL_CTL, unix.SYS_EPOLL_PWAIT, unix.SYS_EVENTFD2,
			unix.SYS_EXIT, unix.SYS_EXIT_GROUP,
		},
	}
	if runtime.GOARCH == "arm64" {
		calls.architecture = unix.AUDIT_ARCH_AARCH64
		calls.threadFlags = 0x50f00
	}
	return calls
}

func linuxLimits(limits heicdecode.Limits) error {
	// RLIMIT_AS constrains virtual mappings, not physical RSS. A lowered limit
	// does not revoke existing mappings, so reject an already oversized helper.
	file, err := os.Open("/proc/self/statm")
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 4097))
	closeErr := file.Close()
	if err = errors.Join(readErr, closeErr); err != nil {
		return err
	}
	fields := strings.Fields(string(data))
	if len(data) > 4096 || len(fields) == 0 {
		return errors.New("HEIC address-space accounting unavailable")
	}
	pages, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || pages > uint64(limits.OSProcessBytes)/uint64(os.Getpagesize()) {
		return errors.New("HEIC initial address space exceeds native limit")
	}
	for _, ceiling := range []struct {
		resource int
		value    uint64
	}{
		{unix.RLIMIT_AS, uint64(limits.OSProcessBytes)},
		{unix.RLIMIT_CPU, uint64((limits.Timeout + time.Second - 1) / time.Second)},
		{unix.RLIMIT_CORE, 0}, {unix.RLIMIT_FSIZE, 0}, {unix.RLIMIT_NOFILE, 32},
	} {
		var previous unix.Rlimit
		if err = unix.Getrlimit(ceiling.resource, &previous); err != nil {
			return err
		}
		value := min(ceiling.value, previous.Cur, previous.Max)
		bound := unix.Rlimit{Cur: value, Max: value}
		if err = unix.Setrlimit(ceiling.resource, &bound); err != nil {
			return err
		}
		var installed unix.Rlimit
		if err = unix.Getrlimit(ceiling.resource, &installed); err != nil {
			return err
		}
		if installed != bound {
			return errors.New("HEIC native resource limit not installed")
		}
		// Readiness reports the requested AS ceiling; refuse inherited tighter
		// AS limits rather than reporting a different control as that value.
		if ceiling.resource == unix.RLIMIT_AS && value != ceiling.value {
			return errors.New("HEIC inherited address-space limit is too small")
		}
	}
	return nil
}

func verifyLinuxAddressLimit(maxBytes int64) error {
	mapping, err := unix.Mmap(-1, 0, int(maxBytes)+os.Getpagesize(), unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err == nil {
		_ = unix.Munmap(mapping)
		return errors.New("HEIC native address-space limit not enforced")
	}
	if !errors.Is(err, unix.ENOMEM) {
		return fmt.Errorf("HEIC native memory probe: %w", err)
	}
	return nil
}
