package displays

import (
	"errors"
	"image"
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
	var empty *EmptyError
	if !errors.As(err, &empty) {
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
			var invalid *InvalidTopologyError
			if !errors.As(err, &invalid) {
				t.Fatalf("newSnapshot() = %v, want InvalidTopologyError", err)
			}
		})
	}
}

func TestInspectUnsupported_IsTyped(t *testing.T) {
	_, err := inspectWindow(nil)
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("inspectWindow(nil) = %v, want UnsupportedError", err)
	}
}
