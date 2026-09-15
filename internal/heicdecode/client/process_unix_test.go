//go:build darwin || linux

package client

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func TestOwnedPeerCancellationTerminatesDescendant(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	if err = listener.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	client := ownedPeer(t, "family", 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, decodeErr := client.Do(ctx, heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
			return []byte(listener.Addr().String()), nil
		})
		done <- decodeErr
	}()
	connection, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = connection.Close() }()
	if err = connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var pid uint32
	if err = binary.Read(connection, binary.LittleEndian, &pid); err != nil {
		t.Fatal(err)
	}
	// If the group-kill guard regresses, clean up the still-connected owned
	// descendant after the bounded EOF assertion. Never poll PID existence.
	exited := false
	defer func() {
		if !exited {
			if process, findErr := os.FindProcess(int(pid)); findErr == nil {
				_ = process.Kill()
				_ = process.Release()
			}
		}
	}()
	cancel()
	if err = <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("family cancellation: %v", err)
	}
	var unexpected [1]byte
	_, err = connection.Read(unexpected[:])
	exited = errors.Is(err, io.EOF)
	if !exited {
		t.Fatalf("descendant kept its connection after client joined: %v", err)
	}
	client.Stop()
	client.Wait()
}
