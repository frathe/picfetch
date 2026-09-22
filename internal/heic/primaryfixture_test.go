package heic

import (
	"context"
	"os"
	"testing"
)

func TestHEICNativePrimarySelection(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit native provider qualification")
	}
	data, err := os.ReadFile("testdata/nonfirst-primary.heic")
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient("")
	t.Cleanup(func() { client.Stop(); client.Wait() })
	result, err := client.Read(context.Background(), data, Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 64 || result.Height != 64 {
		t.Fatalf("primary dimensions = %dx%d", result.Width, result.Height)
	}
	assertGrayPixel(t, result, 8, 8, 255)
	assertGrayPixel(t, result, 48, 8, 0)
	t.Log(result.Provider)
}
