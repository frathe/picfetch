//go:build linux && (amd64 || arm64)

package heic

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestHEICLinuxWorkerRestrictions(t *testing.T) {
	stage := os.Getenv("PICFETCH_HEIC_RESTRICTION_TEST")
	if stage == "" {
		if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
			t.Skip("requires explicit native restriction qualification")
		}
	} else {
		if stage == "install" {
			if err := restrictWorker(); err != nil {
				t.Fatal(err)
			}
		}
		for _, kind := range []int{unix.SOCK_STREAM, unix.SOCK_DGRAM} {
			fd, err := unix.Socket(unix.AF_INET, kind, 0)
			if err == nil {
				_ = unix.Close(fd)
			}
			if !errors.Is(err, unix.EPERM) {
				t.Fatalf("network socket denial = %v", err)
			}
		}
		for _, limit := range []struct {
			resource int
			value    uint64
		}{{unix.RLIMIT_AS, 4 * 1024 * 1024 * 1024}, {unix.RLIMIT_CPU, 40}, {unix.RLIMIT_CORE, 0}, {unix.RLIMIT_FSIZE, 0}} {
			var observed unix.Rlimit
			if err := unix.Getrlimit(limit.resource, &observed); err != nil {
				t.Fatal(err)
			}
			if observed.Cur != limit.value || observed.Max != limit.value {
				t.Fatalf("resource %d = %+v, want %d", limit.resource, observed, limit.value)
			}
		}
		if stage == "inherited" {
			return
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestHEICLinuxWorkerRestrictions$", "-test.v")
	next := "install"
	if stage == "install" {
		next = "inherited"
	}
	cmd.Env = append(os.Environ(), "PICFETCH_HEIC_RESTRICTION_TEST="+next)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restriction child: %v\n%s", err, output)
	}
	t.Log(string(output))
}
