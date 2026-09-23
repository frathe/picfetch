package heic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv("PICFETCH_HEIC_TEST_CHILD"); mode != "" {
		fakeWorker(mode)
		os.Exit(0)
	}
	if WorkerMain() {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestHEICNativeQualification(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit native provider qualification")
	}
	client := NewClient("")
	t.Cleanup(func() { client.Stop(); client.Wait() })
	t.Run("probe_8_and_10_bit_pixels", func(t *testing.T) {
		if err := client.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
	for _, bits := range []string{"8", "10"} {
		t.Run("full_primary_"+bits+"_bit", func(t *testing.T) {
			data, err := os.ReadFile("testdata/probe" + bits + ".heic")
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.Read(context.Background(), data, Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 4096})
			if err != nil {
				t.Fatal(err)
			}
			t.Log(result.Provider)
			if result.Width != 64 || result.Height != 64 || result.Stride != 256 {
				t.Fatalf("dimensions: %+v", result)
			}
			assertGrayPixel(t, result, 8, 8, 0)
			assertGrayPixel(t, result, 48, 8, 255)
		})
	}
	for _, fixture := range []struct {
		name        string
		top, bottom int
	}{
		{"container-rotate", 255, 0}, {"exif-rotate", 0, 255}, {"container-and-exif", 255, 0},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + fixture.name + ".heic")
			if err != nil {
				t.Fatal(err)
			}
			request := Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 4096}
			result, err := client.Read(context.Background(), data, request)
			if err != nil {
				t.Fatal(err)
			}
			assertGrayPixel(t, result, 8, 8, fixture.top)
			assertGrayPixel(t, result, 8, 48, fixture.bottom)
			request.Pixels = false
			config, err := client.Read(context.Background(), data, request)
			if err != nil {
				t.Fatal(err)
			}
			if config.Width != result.Width || config.Height != result.Height || len(config.Pixels) != 0 {
				t.Fatalf("config differs: %+v", config)
			}
		})
	}
	for _, input := range []struct {
		name   string
		change func([]byte) []byte
		want   error
	}{
		{"sequence", func(data []byte) []byte { copy(data[8:12], "hevc"); return data }, ErrUnsupported},
		{"AVIF_keeps_existing_decoder", func(data []byte) []byte { copy(data[8:12], "avif"); return data }, ErrUnsupported},
		{"HDR", func(data []byte) []byte {
			i := bytes.Index(data, []byte("nclx"))
			binary.BigEndian.PutUint16(data[i+6:], 16)
			return data
		}, nil},
		{"wide_gamut", func(data []byte) []byte {
			i := bytes.Index(data, []byte("nclx"))
			binary.BigEndian.PutUint16(data[i+4:], 12)
			return data
		}, nil},
		{"oversized_dimensions", func(data []byte) []byte {
			i := bytes.Index(data, []byte("ispe"))
			binary.BigEndian.PutUint32(data[i+8:], 200_000_000)
			return data
		}, ErrInvalid},
		{"malformed_box", func(data []byte) []byte { binary.BigEndian.PutUint32(data, uint32(len(data)+1)); return data }, ErrInvalid},
	} {
		t.Run(input.name, func(t *testing.T) {
			data := input.change([]byte(probe8))
			result, err := client.Read(context.Background(), data, Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 4096})
			if !errors.Is(err, input.want) {
				t.Fatalf("decode = %v, want %v", err, input.want)
			}
			if input.want == nil && (result.Width != 64 || result.Height != 64 || result.Stride != 256 || len(result.Pixels) != 64*64*4) {
				t.Fatalf("native rendition has invalid dimensions/payload: %dx%d, stride %d, bytes %d", result.Width, result.Height, result.Stride, len(result.Pixels))
			}
		})
	}
}

func assertGrayPixel(t *testing.T, result Result, x, y, want int) {
	t.Helper()
	if x >= result.Width || y >= result.Height || len(result.Pixels) != result.Stride*result.Height {
		t.Fatalf("invalid pixel plane: %+v", result)
	}
	pixel := result.Pixels[y*result.Stride+x*4:][:4]
	for _, value := range pixel[:3] {
		if int(value) < want-2 || int(value) > want+2 {
			t.Fatalf("pixel (%d,%d) = %v, want grayscale %d", x, y, pixel, want)
		}
	}
	if pixel[3] != 255 {
		t.Fatalf("pixel alpha = %d", pixel[3])
	}
}

func TestHEICWorkerLifecycle(t *testing.T) {
	t.Run("cancelled_before_admission", func(t *testing.T) {
		client := NewClient("")
		t.Cleanup(func() { client.Stop(); client.Wait() })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := client.Check(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled check = %v, want cancellation", err)
		}
	})
	t.Run("stop_cancels_submitted_and_queued_work", func(t *testing.T) {
		client := fakeClient(t, "hang")
		started := make(chan struct{}, 2)
		client.started = func() { started <- struct{}{} }
		command := client.command
		var launches atomic.Int32
		client.command = func(ctx context.Context, executable string) *exec.Cmd {
			launches.Add(1)
			cmd := command(ctx, executable)
			return cmd
		}
		completed := make(chan error, 3)
		for range 3 {
			go func() { completed <- client.Check(context.Background()) }()
		}
		for range 2 {
			<-started
		}
		client.Stop()
		client.Wait()
		for range 3 {
			if err := <-completed; !errors.Is(err, context.Canceled) {
				t.Fatalf("stopped check = %v", err)
			}
		}
		if got := launches.Load(); got != 2 {
			t.Fatalf("launched %d children, want bounded two", got)
		}
		if err := client.Check(context.Background()); !errors.Is(err, context.Canceled) {
			t.Fatalf("admission after Stop = %v", err)
		}
	})
	t.Run("deadline_retires_hung_process", func(t *testing.T) {
		client := fakeClient(t, "hang")
		client.deadline = 50 * time.Millisecond
		if err := client.Check(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("hung child = %v", err)
		}
		client.Wait()
	})
	t.Run("crash_is_operational_failure", func(t *testing.T) {
		client := fakeClient(t, "crash")
		err := client.Check(context.Background())
		if err == nil || errors.Is(err, ErrUnavailable) {
			t.Fatalf("crash = %v", err)
		}
	})
}

func TestHEICWorkerProtocol(t *testing.T) {
	t.Run("larger caller limit does not reject a small image", func(t *testing.T) {
		client := fakeClient(t, "valid")
		result, err := client.Read(context.Background(), []byte{1}, Request{Pixels: true, MaxEncodedBytes: 4 * 1024 * 1024 * 1024, MaxPixels: 100})
		if err != nil || len(result.Pixels) != 4 {
			t.Fatalf("small image under larger caller budget: %+v, %v", result, err)
		}
	})
	for _, mode := range []string{"negative_dimensions", "overflow_dimensions", "bad_stride", "truncated_pixels", "excess_pixels", "oversized_metadata", "oversized_header", "invalid_json"} {
		t.Run(mode, func(t *testing.T) {
			client := fakeClient(t, mode)
			_, err := client.Read(context.Background(), []byte{1}, Request{Pixels: true, MaxEncodedBytes: 1, MaxPixels: 100})
			if !errors.Is(err, ErrProtocol) {
				t.Fatalf("malformed response = %v, want protocol error", err)
			}
		})
	}
	t.Run("straight_RGBA_and_metadata", func(t *testing.T) {
		client := fakeClient(t, "valid")
		result, err := client.Read(context.Background(), []byte{1}, Request{Pixels: true, MaxEncodedBytes: 1, MaxPixels: 100})
		if err != nil {
			t.Fatal(err)
		}
		if result.Width != 1 || result.Height != 1 || result.Stride != 4 || !bytes.Equal(result.Pixels, []byte{12, 34, 56, 78}) || string(result.EXIF) != "exif" {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
	t.Run("absence_is_distinct", func(t *testing.T) {
		client := fakeClient(t, "unavailable")
		if err := client.Check(context.Background()); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("absence = %v", err)
		}
	})
}

func fakeClient(t *testing.T, mode string) *Client {
	t.Helper()
	client := NewClient("")
	client.command = func(ctx context.Context, executable string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, executable)
		cmd.Env = append(os.Environ(), "PICFETCH_HEIC_TEST_CHILD="+mode)
		return cmd
	}
	t.Cleanup(func() { client.Stop(); client.Wait() })
	return client
}

func fakeWorker(mode string) {
	if mode == "hold_descendant_pipe" {
		ready := os.NewFile(3, "descendant-ready")
		if ready == nil {
			os.Exit(25)
		}
		_ = binary.Write(ready, binary.BigEndian, int64(os.Getpid()))
		<-time.After(time.Hour)
		return
	}
	var request wireRequest
	if err := readHeader(os.Stdin, &request); err != nil {
		os.Exit(21)
	}
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		os.Exit(22)
	}
	switch mode {
	case "spawn_descendant":
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(), "PICFETCH_HEIC_TEST_CHILD=hold_descendant_pipe")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		ready := os.NewFile(3, "descendant-ready")
		cmd.ExtraFiles = []*os.File{ready}
		if err := cmd.Start(); err != nil {
			os.Exit(26)
		}
		return
	case "hang":
		<-time.After(time.Hour)
		return
	case "crash":
		os.Exit(13)
	case "unavailable":
		_ = writeResponse(os.Stdout, Result{}, ErrUnavailable)
		return
	case "oversized_header":
		_ = binary.Write(os.Stdout, binary.BigEndian, uint32(headerLimit+1))
		return
	case "invalid_json":
		_, _ = os.Stdout.Write([]byte{0, 0, 0, 1, '{'})
		return
	}
	response := wireResponse{Width: 1, Height: 1, Stride: 4, PixelBytes: 4}
	pixels, exif := []byte{12, 34, 56, 78}, []byte(nil)
	switch mode {
	case "negative_dimensions":
		response.Width = -1
	case "overflow_dimensions":
		response.Width, response.Height = 1<<62, 1<<62
	case "bad_stride":
		response.Stride = 3
	case "truncated_pixels":
		pixels = pixels[:3]
	case "excess_pixels":
		pixels = append(pixels, 0)
	case "oversized_metadata":
		response.EXIFBytes = metadataLimit + 1
	case "valid":
		exif, response.EXIFBytes = []byte("exif"), 4
	}
	_ = writeHeader(os.Stdout, response)
	_, _ = os.Stdout.Write(exif)
	_, _ = os.Stdout.Write(pixels)
}
