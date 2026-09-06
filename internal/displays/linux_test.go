//go:build linux

package displays

import (
	"errors"
	"image"
	"testing"
)

func TestDisplayLinux_ParseXRandRPreservesNegativeNativeGeometry(t *testing.T) {
	output := `DP-1 connected 1920x1080-1920+200 (normal left inverted right x axis y axis)
eDP-1 connected primary 2560x1600+0+0 (normal left inverted right x axis y axis)
HDMI-1 disconnected (normal left inverted right x axis y axis)`
	found, err := parseXRandR(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("displays = %+v", found)
	}
	if found[0].ID != "DP-1" || found[0].Bounds != image.Rect(-1920, 200, 0, 1280) {
		t.Fatalf("negative-origin display = %+v", found[0])
	}
	if found[1].ID != "eDP-1" || found[1].Bounds != image.Rect(0, 0, 2560, 1600) {
		t.Fatalf("HiDPI-sized display = %+v", found[1])
	}
}

func TestDisplayLinux_ParseXRandREmptyIsTyped(t *testing.T) {
	_, err := parseXRandR("HDMI-1 disconnected\n")
	if _, ok := errors.AsType[*EmptyError](err); !ok {
		t.Fatalf("parseXRandR() = %v, want EmptyError", err)
	}
}
