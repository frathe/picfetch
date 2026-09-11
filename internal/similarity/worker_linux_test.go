//go:build linux && (amd64 || arm64)

package similarity

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestLinuxWorkerIsolation(t *testing.T) {
	if os.Getenv("PICFETCH_TEST_ISOLATION") == "1" {
		// Hold a thread created before installation to prove TSYNC reaches it.
		ready, check, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			defer close(done)
			close(ready)
			<-check
			assertSocketsDenied(t)
		}()
		<-ready
		if err := isolateWorker(); err != nil {
			t.Fatal(err)
		}
		close(check)
		<-done
		assertSocketsDenied(t)
		if err := VerifyOffline(context.Background()); err != nil {
			t.Fatal(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestLinuxWorkerIsolation$", "-test.v")
	cmd.Env = append(os.Environ(), "PICFETCH_TEST_ISOLATION=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("isolated child: %v\n%s", err, output)
	}
	// Installation in the worker must not change the viewer's network access.
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_ = listener.Close()
}

func assertSocketsDenied(t *testing.T) {
	t.Helper()
	for _, domain := range []int{unix.AF_INET, unix.AF_INET6, unix.AF_UNIX} {
		for _, kind := range []int{unix.SOCK_STREAM, unix.SOCK_DGRAM} {
			fd, err := unix.Socket(domain, kind, 0)
			if err == nil {
				_ = unix.Close(fd)
			}
			if !errors.Is(err, unix.EPERM) {
				t.Errorf("socket(%d,%d): %v, want EPERM", domain, kind, err)
			}
		}
	}
	// Invalid arguments normally produce EINVAL/EFAULT; denial must happen first.
	for _, call := range []uintptr{unix.SYS_SOCKETPAIR, unix.SYS_IO_URING_SETUP, 0x40000000 | unix.SYS_SOCKET} {
		_, _, err := unix.Syscall(call, 0, 0, 0)
		if err != unix.EPERM {
			t.Errorf("syscall %d: %v, want EPERM", call, err)
		}
	}
}
