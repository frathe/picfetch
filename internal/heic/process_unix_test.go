//go:build linux || darwin

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

func TestHEICWorkerRetiresDescendants(t *testing.T) {
	client := fakeClient(t, "spawn_descendant")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
	command := client.command
	client.command = func(ctx context.Context, executable string) *exec.Cmd {
		cmd := command(ctx, executable)
		cmd.ExtraFiles = []*os.File{writer}
		return cmd
	}
	finished := make(chan error, 1)
	go func() { finished <- client.Check(context.Background()) }()
	if err := reader.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := binary.Read(reader, binary.BigEndian, &pid); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(int(pid), syscall.SIGKILL) })
	_ = writer.Close()
	client.Stop()
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped check = %v", err)
	}
	client.Wait()
	if err := reader.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var extra [1]byte
	if _, err := reader.Read(extra[:]); !errors.Is(err, io.EOF) {
		t.Fatalf("decoder descendant retained its pipe after Stop/Wait: %v", err)
	}
}
