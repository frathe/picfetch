//go:build darwin

package wallpaper

import "testing"

func TestSetDarwin_TargetParsesStableDisplayID(t *testing.T) {
	request := Request{Path: "/tmp/mosaic.png", Target: "4294967295"}
	called := 0
	err := setDarwinRequestWith(request, func(path string, target uint32, targeted bool) error {
		called++
		if path != request.Path || target != ^uint32(0) || !targeted {
			t.Fatalf("native args = %q, %d, %v", path, target, targeted)
		}
		return nil
	})
	if err != nil || called != 1 {
		t.Fatalf("setDarwinRequestWith() = %v, calls=%d", err, called)
	}
}

func TestSetDarwin_NoTargetPreservesAllScreenMode(t *testing.T) {
	called := 0
	err := setDarwinRequestWith(Request{Path: "/tmp/photo.png"}, func(path string, target uint32, targeted bool) error {
		called++
		if path != "/tmp/photo.png" || target != 0 || targeted {
			t.Fatalf("native args = %q, %d, %v", path, target, targeted)
		}
		return nil
	})
	if err != nil || called != 1 {
		t.Fatalf("setDarwinRequestWith() = %v, calls=%d", err, called)
	}
}

func TestSetDarwin_MissingOrMalformedIDDoesNotInvokeNative(t *testing.T) {
	called := 0
	err := setDarwinRequestWith(Request{Path: "/tmp/mosaic.png", Target: "not-a-display"}, func(string, uint32, bool) error {
		called++
		return nil
	})
	if err == nil || called != 0 {
		t.Fatalf("setDarwinRequestWith() = %v, calls=%d; want preflight failure", err, called)
	}
}
