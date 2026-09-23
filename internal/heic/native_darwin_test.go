//go:build darwin && cgo && (amd64 || arm64)

package heic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// This is a required native runner inventory, not a substitute for the wider
// fixture qualification recorded in the implementation evidence.
func TestHEICDarwinNativeQualification(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit native ImageIO qualification")
	}
	client := NewClient("")
	t.Cleanup(func() { client.Stop(); client.Wait() })
	t.Run("representative_pixels", func(t *testing.T) {
		if err := client.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
	for _, tc := range []struct {
		file string
		want [4]byte
	}{
		{"probe8.heic", [4]byte{0, 255, 0, 255}},
		{"probe10.heic", [4]byte{0, 255, 0, 255}},
		{"container-rotate.heic", [4]byte{255, 255, 0, 0}},
		{"exif-rotate.heic", [4]byte{0, 0, 255, 255}},
		{"container-and-exif.heic", [4]byte{255, 255, 0, 0}},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			request := Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 64 * 64}
			result, err := client.Read(context.Background(), data, request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Width != 64 || result.Height != 64 || result.Stride != 256 || len(result.Pixels) != 64*64*4 {
				t.Fatalf("full primary dimensions/payload = %dx%d stride=%d bytes=%d", result.Width, result.Height, result.Stride, len(result.Pixels))
			}
			if !strings.HasPrefix(result.Provider, "Apple ImageIO ") {
				t.Fatalf("unexpected native provider identity: %q", result.Provider)
			}
			t.Logf("provider=%q os=%+v", result.Provider, SystemIdentity())
			for index, point := range [][2]int{{8, 8}, {48, 8}, {8, 48}, {48, 48}} {
				pixel := result.Pixels[point[1]*result.Stride+point[0]*4:][:4]
				for _, value := range pixel[:3] {
					if delta := int(value) - int(tc.want[index]); delta < -2 || delta > 2 {
						t.Fatalf("pixel %v=%v, want grayscale %d", point, pixel, tc.want[index])
					}
				}
				if pixel[3] != 255 {
					t.Fatalf("opaque fixture lost alpha at %v: %v", point, pixel)
				}
			}
			request.Pixels = false
			metadata, err := client.Read(context.Background(), data, request)
			if err != nil || metadata.Width != result.Width || metadata.Height != result.Height || len(metadata.Pixels) != 0 {
				t.Fatalf("metadata dimensions disagree with pixels: %+v, %v", metadata, err)
			}
		})
	}
	t.Run("HDR_native_rendering", func(t *testing.T) {
		data := []byte(probe10)
		profile := bytes.Index(data, []byte("nclx"))
		if profile < 0 {
			t.Fatal("fixture has no explicit nclx profile")
		}
		binary.BigEndian.PutUint16(data[profile+6:], 16)
		_, err := client.Read(context.Background(), data, Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 64 * 64})
		if err != nil {
			t.Fatalf("native PQ rendition was refused: %v", err)
		}
	})
	t.Run("pixel_budget_refusal", func(t *testing.T) {
		_, err := client.Read(context.Background(), []byte(probe8), Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 64*64 - 1})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("oversized primary image = %v, want invalid", err)
		}
	})
	t.Run("primary_metadata", func(t *testing.T) {
		data, err := os.ReadFile("testdata/exif-rotate.heic")
		if err != nil {
			t.Fatal(err)
		}
		tiff := bytes.Index(data, []byte{'I', 'I', 42, 0})
		if tiff < 0 {
			t.Fatal("fixture lost EXIF data")
		}
		for _, pixels := range []bool{false, true} {
			request := Request{Pixels: pixels, MaxEncodedBytes: 64 * 1024, MaxPixels: 4096}
			result, err := client.Read(t.Context(), data, request)
			if err != nil || !bytes.Equal(result.EXIF, data[tiff:]) {
				t.Fatalf("pixels=%v primary metadata = %x, error=%v", pixels, result.EXIF, err)
			}
			unassociated := bytes.Clone(data)
			copy(unassociated[bytes.Index(unassociated, []byte("cdsc")):], "free")
			result, err = client.Read(t.Context(), unassociated, request)
			if err != nil || len(result.EXIF) != 0 {
				t.Fatalf("pixels=%v unassociated metadata = %x, error=%v", pixels, result.EXIF, err)
			}
		}
	})
}

func TestHEICDarwinInheritedSandbox(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires native sandbox qualification")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	stage := os.Getenv("PICFETCH_HEIC_SANDBOX_STAGE")
	if stage == "child" {
		for _, network := range []string{"tcp4", "udp4"} {
			dialer := net.Dialer{Timeout: time.Second}
			connection, err := dialer.DialContext(ctx, network, "192.0.2.1:443")
			if err == nil {
				if network == "udp4" {
					_, err = connection.Write(nil)
				}
				_ = connection.Close()
			}
			if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
				t.Fatalf("inherited %s network denial = %v", network, err)
			}
		}
		return
	}
	client := NewClient("")
	next := "parent"
	if stage == "parent" {
		client = NewInheritedSandboxClient("")
		next = "child"
	}
	t.Cleanup(func() { client.Stop(); client.Wait() })
	cmd := client.command(ctx, os.Args[0])
	cmd.Args = append(cmd.Args, "-test.run=^TestHEICDarwinInheritedSandbox$", "-test.v")
	cmd.Env = append(os.Environ(), "PICFETCH_HEIC_SANDBOX_STAGE="+next)
	prepareWorker(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sandbox stage %s: %v\n%s", next, err, output)
	}
	if stage == "parent" {
		if err := client.Check(ctx); err != nil {
			t.Fatalf("native decoding inside inherited sandbox: %v", err)
		}
	}
}
