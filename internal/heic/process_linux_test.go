package heic

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestHEICWorkerDiesWithProducer(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_PARENT_TEST") == "1" {
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(), "PICFETCH_HEIC_TEST_CHILD=hold_descendant_pipe")
		cmd.ExtraFiles = []*os.File{os.NewFile(3, "ready")}
		prepareWorker(cmd)
		if err := cmd.Start(); err != nil {
			os.Exit(31)
		}
		var release [1]byte
		_, _ = os.Stdin.Read(release[:])
		os.Exit(17)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ready, signal, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ready.Close(); _ = signal.Close() }()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHEICWorkerDiesWithProducer$")
	cmd.Env = append(os.Environ(), "PICFETCH_HEIC_PARENT_TEST=1")
	cmd.ExtraFiles = []*os.File{signal}
	stop, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	_ = signal.Close()
	if err := ready.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := binary.Read(ready, binary.BigEndian, &pid); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(int(pid), syscall.SIGKILL) })
	_ = stop.Close()
	if err := cmd.Wait(); err == nil {
		t.Fatal("producer did not crash")
	}
	if err := ready.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var extra [1]byte
	if _, err := ready.Read(extra[:]); !errors.Is(err, io.EOF) {
		t.Fatalf("native child survived producer crash: %v", err)
	}
}
