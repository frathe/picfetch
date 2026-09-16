package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func TestRemoteRepeatedRequestsShareOwner(t *testing.T) {
	owner := ownedPeer(t, "success", 5*time.Second)
	parent, child := net.Pipe()
	remote, err := NewRemote(child, child, owner.config.Limits)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- owner.Serve(ctx, parent, parent) }()
	t.Cleanup(func() { remote.Stop(); remote.Wait(); cancel(); <-done })
	reads := 0
	for range 2 {
		result, decodeErr := remote.Do(ctx, heicdecode.Decode, func(_ context.Context, maxBytes int64) ([]byte, error) {
			reads++
			if maxBytes != owner.config.Limits.MaxInputBytes {
				t.Errorf("grant=%d", maxBytes)
			}
			return []byte("owned input"), nil
		})
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if _, ok := result.Image.(*image.NRGBA64); !ok {
			t.Fatalf("pixel layout changed: %T", result.Image)
		}
	}
	if reads != 2 {
		t.Fatalf("source reads=%d", reads)
	}
}

func TestRemotePreAdmissionRefusalDoesNotReadSource(t *testing.T) {
	peer := ownedPeer(t, "success", 5*time.Second)
	config := peer.config
	config.SHA256[0] ^= 1
	owner, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Stop(); owner.Wait() }()
	parent, child := net.Pipe()
	remote, err := NewRemote(child, child, config.Limits)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { remote.Stop(); remote.Wait() }()
	done := make(chan error, 1)
	go func() { done <- owner.Serve(context.Background(), parent, parent) }()
	read := false
	_, err = remote.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) { read = true; return []byte{1}, nil })
	var failure *heicdecode.Failure
	if read || !errors.As(err, &failure) || failure.Status != heicdecode.StatusUnavailable {
		t.Fatalf("read=%v refusal=%v", read, err)
	}
	if err = <-done; !errors.Is(err, io.EOF) {
		t.Fatalf("refused connection not retired: %v", err)
	}
}

func TestBrokerResultRejectsOversizedTruncatedAndTrailingFrames(t *testing.T) {
	limits := heicdecode.DefaultLimits(0)
	var encoded bytes.Buffer
	result := heicdecode.Response{Image: image.NewNRGBA64(image.Rect(0, 0, 1, 1))}
	if err := heicdecode.WriteResponse(&encoded, heicdecode.Decode, result, limits); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	if _, err := readBrokerResult(bytes.NewReader(data), uint64(len(data)), heicdecode.Decode, limits); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		size uint64
		data []byte
	}{
		{"oversized", ^uint64(0), nil},
		{"missing frame byte", uint64(len(data) + 1), data},
		{"short payload", uint64(len(data)), data[:len(data)-1]},
		{"trailing payload", uint64(len(data) + 1), append(bytes.Clone(data), 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if result, err := readBrokerResult(bytes.NewReader(tc.data), tc.size, heicdecode.Decode, limits); err == nil || result.Image != nil {
				t.Fatal("invalid frame published pixels")
			}
		})
	}
}

func TestRemoteRejectsGrantBeforeReadingSource(t *testing.T) {
	limits := heicdecode.DefaultLimits(0)
	for _, size := range []uint64{0, uint64(limits.MaxInputBytes) + 1, ^uint64(0)} {
		parent, child := net.Pipe()
		remote, err := NewRemote(child, child, limits)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() { _ = parent.Close() }()
			if _, err := readBrokerHeader(parent); err == nil {
				_ = writeBrokerHeader(parent, brokerHeader{kind: brokerGrant, operation: heicdecode.Decode, size: size})
			}
		}()
		read := false
		_, err = remote.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) { read = true; return []byte{1}, nil })
		remote.Stop()
		remote.Wait()
		<-done
		if read || !errors.Is(err, heicdecode.ErrInvalidResponse) {
			t.Fatalf("grant=%d read=%v error=%v", size, read, err)
		}
	}
}

// Hold output inside a Write that Close can interrupt, just like a full pipe.
// The result header arrives only after helper cleanup and pixel validation.
type heldResultWriter struct {
	io.WriteCloser
	blocked, closed chan struct{}
	once            sync.Once
}

func (w *heldResultWriter) Write(p []byte) (int, error) {
	if len(p) == 16 && string(p[:4]) == "PHB1" && binary.LittleEndian.Uint16(p[4:]) == brokerResult {
		close(w.blocked)
		<-w.closed
		return 0, io.ErrClosedPipe
	}
	return w.WriteCloser.Write(p)
}

func (w *heldResultWriter) Close() error {
	w.once.Do(func() { close(w.closed) })
	return w.WriteCloser.Close()
}

func TestRemoteOutputRetainsAdmissionAndStopJoins(t *testing.T) {
	owner := ownedPeer(t, "success", 5*time.Second)
	parent, child := net.Pipe()
	held := &heldResultWriter{WriteCloser: parent, blocked: make(chan struct{}), closed: make(chan struct{})}
	remote, err := NewRemote(child, child, owner.config.Limits)
	if err != nil {
		t.Fatal(err)
	}
	serveDone, remoteDone := make(chan error, 1), make(chan error, 1)
	go func() { serveDone <- owner.Serve(context.Background(), parent, held) }()
	go func() {
		_, err := remote.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) { return []byte{1}, nil })
		remoteDone <- err
	}()
	defer func() {
		owner.Stop()
		owner.Wait()
		remote.Stop()
		remote.Wait()
		<-serveDone
		<-remoteDone
	}()
	select {
	case <-held.blocked:
	case <-time.After(5 * time.Second):
		t.Fatal("result did not reach output")
	}
	// This enters the same lane used by foreground Do, synchronously; it
	// cannot acquire a grant while the remote output is blocked.
	next := owner.lane.enter(context.Background(), Foreground)
	defer next.release()
	select {
	case <-next.ready:
		t.Fatal("output escaped the shared admission bound")
	default:
	}
	owner.Stop()
	owner.Wait()
	select {
	case <-next.ready:
	default:
		t.Fatal("stopped output retained its grant")
	}
}

type readObserver struct {
	io.ReadCloser
	once    sync.Once
	started chan struct{}
}

func (r *readObserver) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.ReadCloser.Read(p)
}

func TestBrokerBoundsIdleConnectionsAndJoinsStop(t *testing.T) {
	owner := ownedPeer(t, "success", 5*time.Second)
	var completions []chan error
	var peers []net.Conn
	defer func() {
		for _, peer := range peers {
			_ = peer.Close()
		}
	}()
	for range maxServices {
		parent, child := net.Pipe()
		peers = append(peers, child)
		reader := &readObserver{ReadCloser: parent, started: make(chan struct{})}
		done := make(chan error, 1)
		completions = append(completions, done)
		go func() { done <- owner.Serve(context.Background(), reader, parent) }()
		<-reader.started
	}
	parent, child := net.Pipe()
	defer func() { _ = child.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := owner.Serve(ctx, parent, parent); !errors.Is(err, ErrBusy) {
		t.Fatalf("excess connection: %v", err)
	}
	owner.Stop()
	owner.Wait()
	for _, done := range completions {
		<-done
	}
}

type intentObserver struct {
	io.WriteCloser
	once    sync.Once
	written chan struct{}
}

func (w *intentObserver) Write(p []byte) (int, error) {
	n, err := w.WriteCloser.Write(p)
	w.once.Do(func() { close(w.written) })
	return n, err
}

func TestRemoteQueuedCancellationReleasesService(t *testing.T) {
	owner := ownedPeer(t, "success", 5*time.Second)
	localCtx, cancelLocal := context.WithCancel(context.Background())
	entered, localDone := make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := owner.Do(localCtx, heicdecode.Decode, func(ctx context.Context, _ int64) ([]byte, error) {
			close(entered)
			<-ctx.Done()
			return nil, ctx.Err()
		})
		localDone <- err
	}()
	t.Cleanup(func() { cancelLocal(); <-localDone })
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("local request not admitted")
	}
	parent, child := net.Pipe()
	observer := &intentObserver{WriteCloser: child, written: make(chan struct{})}
	remote, err := NewRemote(child, observer, owner.config.Limits)
	if err != nil {
		t.Fatal(err)
	}
	serveCtx, stopServe := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- owner.Serve(serveCtx, parent, parent) }()
	var serveJoined bool
	t.Cleanup(func() {
		remote.Stop()
		remote.Wait()
		stopServe()
		if !serveJoined {
			<-serveDone
		}
	})
	remoteCtx, cancelRemote := context.WithCancel(context.Background())
	remoteDone := make(chan error, 1)
	go func() {
		_, err := remote.Do(remoteCtx, heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
			t.Error("queued source was read before owner released its grant")
			return []byte("owned input"), nil
		})
		remoteDone <- err
	}()
	<-observer.written
	cancelRemote()
	if err = <-remoteDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("remote cancellation: %v", err)
	}
	select {
	case <-serveDone:
		serveJoined = true
	case <-time.After(time.Second):
		t.Fatal("disconnected queued service still waits for the foreground grant")
	}
}
