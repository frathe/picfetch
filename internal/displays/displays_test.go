package displays

import (
	"errors"
	"image"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDisplaySnapshot_PreservesNativeBoundsAndIDs(t *testing.T) {
	displays := []Display{
		{ID: "internal-1", Name: "Built-in Display", Bounds: image.Rect(0, 0, 2560, 1600)},
		{ID: "external-7", Name: "Studio Display", Bounds: image.Rect(-5120, -400, 0, 2480)},
	}
	snapshot, err := newSnapshot(displays, "external-7")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Default != "external-7" || snapshot.Displays[1].Bounds != image.Rect(-5120, -400, 0, 2480) {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	displays[0].ID = "changed"
	if snapshot.Displays[0].ID != "internal-1" {
		t.Fatal("Snapshot retained the caller's display slice")
	}
}

func TestDisplayDefault_UsesGreatestWindowIntersection(t *testing.T) {
	displays := []Display{
		{ID: "left", Name: "Left", Bounds: image.Rect(-1920, 0, 0, 1080)},
		{ID: "right", Name: "Right", Bounds: image.Rect(0, 0, 2560, 1440)},
	}
	if got := defaultForWindow(displays, image.Rect(-300, 100, 700, 900), true); got != "right" {
		t.Fatalf("defaultForWindow() = %q, want right", got)
	}
	if got := defaultForWindow(displays, image.Rectangle{}, false); got != "left" {
		t.Fatalf("fallback default = %q, want deterministic first display", got)
	}
}

func TestDisplaySnapshot_RejectsEmptyAndInvalidTopology(t *testing.T) {
	_, err := newSnapshot(nil, "")
	if _, ok := errors.AsType[*EmptyError](err); !ok {
		t.Fatalf("empty snapshot = %v, want EmptyError", err)
	}

	tests := []struct {
		name     string
		displays []Display
		fallback ID
	}{
		{name: "empty ID", displays: []Display{{Name: "Display", Bounds: image.Rect(0, 0, 1, 1)}}},
		{name: "empty bounds", displays: []Display{{ID: "one", Name: "Display"}}},
		{name: "duplicate ID", displays: []Display{{ID: "one", Bounds: image.Rect(0, 0, 1, 1)}, {ID: "one", Bounds: image.Rect(1, 0, 2, 1)}}},
		{name: "missing default", displays: []Display{{ID: "one", Bounds: image.Rect(0, 0, 1, 1)}}, fallback: "missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSnapshot(tt.displays, tt.fallback)
			if _, ok := errors.AsType[*InvalidTopologyError](err); !ok {
				t.Fatalf("newSnapshot() = %v, want InvalidTopologyError", err)
			}
		})
	}
}

func TestInspectUnsupported_IsTyped(t *testing.T) {
	_, err := inspectWindow(nil)
	if _, ok := errors.AsType[*UnsupportedError](err); !ok {
		t.Fatalf("inspectWindow(nil) = %v, want UnsupportedError", err)
	}
}

func TestInspectDarwinSourceUsesNativeModePixelsAndPointSpaceSelection(t *testing.T) {
	source, err := os.ReadFile("darwin.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(source)
	for _, required := range []string{
		"CGDisplayCopyDisplayMode(ident)",
		"CGDisplayModeGetPixelWidth(mode)",
		"CGDisplayModeGetPixelHeight(mode)",
		"NSIntersectionRect(frame, screen.frame)",
		"Bounds: image.Rect(0, 0, int(width), int(height))",
	} {
		if !strings.Contains(code, required) {
			t.Errorf("darwin display inspection does not contain %q", required)
		}
	}
	if strings.Contains(code, "*width = (int)bounds.size.width") || strings.Contains(code, "*height = (int)bounds.size.height") {
		t.Error("darwin display inspection still reports point-space dimensions")
	}
	if strings.Contains(code, "CGDisplayBounds(ident)") {
		t.Error("darwin display inspection mixes a point-space origin into native-pixel bounds")
	}
}

func TestWindowsDesktopWallpaperScaffoldingIsShared(t *testing.T) {
	paths := []string{"windows.go", "../wallpaper/target_windows.go", "../wincom/desktopwallpaper_windows.go", "../wincom/monitor.go"}
	var combined strings.Builder
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(source)
	}
	for _, unique := range []string{
		"type desktopWallpaperVtbl struct",
		"0xc2cf3110",
		"0xb92b56a9",
		"return int32(uint32(value)) < 0",
	} {
		if count := strings.Count(combined.String(), unique); count != 1 {
			t.Errorf("Windows COM scaffold %q occurs %d times, want exactly once", unique, count)
		}
	}

	bareCall := regexp.MustCompile(`(?m)^\s*(?:defer\s+)?[A-Za-z0-9_]+\.Call\(`)
	for _, path := range paths[:2] {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if call := bareCall.Find(source); call != nil {
			t.Errorf("%s discards syscall results with bare call %q", path, strings.TrimSpace(string(call)))
		}
	}
}
