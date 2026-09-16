//go:build heicnative && linux && !cgo && (amd64 || arm64)

package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 2 && (os.Args[1] == "--owned-linux-policy" || os.Args[1] == "--owned-no-address-limit") {
		if err := runOwnedLinuxControl(os.Args[1]); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runOwnedLinuxControl(mode string) error {
	limits := heicdecode.DefaultLimits(0)
	if mode == "--owned-no-address-limit" {
		return verifyLinuxAddressLimit(limits.OSProcessBytes)
	}
	runtime.GOMAXPROCS(1)
	debug.SetMaxThreads(32)
	threads, err := os.ReadDir("/proc/self/task")
	if err != nil {
		return err
	}
	if len(threads) > 24 {
		return errors.New("unexpected initial thread count")
	}
	// Hold one existing thread across TSYNC, then hold more threads than
	// existed beforehand. This exercises both synchronization and inheritance.
	start, release := make(chan struct{}), make(chan struct{})
	started := make(chan struct{}, 32)
	results := make(chan error, 32)
	var work sync.WaitGroup
	defer func() { close(release); work.Wait() }()
	launch := func() {
		work.Add(1)
		go func() {
			defer work.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			started <- struct{}{}
			select {
			case <-start:
			case <-release:
				return
			}
			fd, openErr := unix.Open("/proc/self/statm", unix.O_RDONLY, 0)
			if openErr == nil {
				_ = unix.Close(fd)
			}
			results <- openErr
			<-release
		}()
	}
	launch()
	<-started
	nativeMemory, err := isolate(limits)
	if err != nil {
		return err
	}
	if nativeMemory != limits.OSProcessBytes {
		return errors.New("wrong native limit")
	}
	close(start)
	for range len(threads) + 1 {
		launch()
	}
	for range len(threads) + 2 {
		if err = <-results; !errors.Is(err, unix.EPERM) {
			return fmt.Errorf("native thread file denial: %v", err)
		}
	}
	// No process or executable is contacted: a null pathname must be refused
	// by seccomp as EPERM before kernel pathname validation could return EFAULT.
	_, _, errno := unix.RawSyscall(unix.SYS_EXECVE, 0, 0, 0)
	if errno != unix.EPERM {
		return fmt.Errorf("exec denial: %v", errno)
	}
	// Runtime-private futexes remain usable; unknown clone3 and io_uring calls
	// are denied before reading their absent argument structures.
	for _, call := range []uintptr{unix.SYS_CLONE3, unix.SYS_IO_URING_SETUP, unix.SYS_IO_URING_ENTER, unix.SYS_IO_URING_REGISTER} {
		_, _, errno = unix.RawSyscall(call, 0, 0, 0)
		if errno != unix.EPERM {
			return fmt.Errorf("syscall %d denial: %v", call, errno)
		}
	}
	// A fixed PicFetch-owned loop verifies that the sandbox still lets wazero
	// compile code and cancel execution; it contains no decoder/image input.
	limits.Timeout = 50 * time.Millisecond
	loop := []byte{0x03, 0x40, 0x0c, 0, 0x0b, 0x0b}
	err = execute(context.Background(), ownedModule(1, loop), bytes.NewReader(nil), io.Discard, io.Discard, limits)
	if !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("sandboxed WASI cancellation: %v", err)
	}
	_, err = fmt.Fprintln(os.Stdout, "owned Linux policy controls passed")
	return err
}

func TestNativeLinuxRuntimePolicy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "--owned-linux-policy")
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("owned native controls: %v: %s", err, out)
	}
	// This negative control omits the native AS limit. The same no-access
	// reservation check must detect that omission without committing RAM.
	command = exec.CommandContext(ctx, os.Args[0], "--owned-no-address-limit")
	out, err := command.CombinedOutput()
	if err == nil || !bytes.Contains(out, []byte("address-space limit not enforced")) {
		t.Fatalf("missing-limit control: error=%v output=%s", err, out)
	}
}
