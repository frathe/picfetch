//go:build windows

package wallpaper

import (
	"errors"
	"syscall"
	"testing"
	"unsafe"
)

func TestSetWindows_TargetPreservesOpaqueIDAndUnicodePath(t *testing.T) {
	request := Request{Path: `C:\Fotos\Mosaik ä.png`, Target: `\\?\DISPLAY#OPAQUE`}
	err := setWindowsTargetWith(request, func(target, path *uint16) error {
		if got := windowsString(target); got != string(request.Target) {
			t.Fatalf("target = %q", got)
		}
		if got := windowsString(path); got != request.Path {
			t.Fatalf("path = %q", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetWindows_TargetValidationFailsBeforeMutation(t *testing.T) {
	setCalls := 0
	err := applyWindowsTarget(nil, nil,
		func(*uint16) error { return errors.New("detached") },
		func(*uint16, *uint16) error { setCalls++; return nil },
	)
	if err == nil || setCalls != 0 {
		t.Fatalf("applyWindowsTarget() = %v, setCalls=%d", err, setCalls)
	}
}

func windowsString(value *uint16) string {
	units := unsafe.Slice(value, 32768)
	length := 0
	for units[length] != 0 {
		length++
	}
	return syscall.UTF16ToString(units[:length])
}
