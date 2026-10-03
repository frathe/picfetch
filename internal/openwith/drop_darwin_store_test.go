//go:build darwin && appleappstore

package openwith

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/frathe/picfetch/internal/fileaccess"
)

func TestNativeWindowDropRetainsSelectionWithoutChangingOtherViews(t *testing.T) {
	got := captureDelivered(t)
	path := filepath.Join(t.TempDir(), "cafe\u0301 # %20.png")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if result := testWindowDrop(path); result != 0 {
		t.Fatalf("native hook fixture failed: %d", result)
	}
	files := got()
	if len(files) != 1 || files[0].Path() != path || !fileaccess.NeedsCapture(files[0]) {
		t.Fatalf("drop lost native ownership: %v", files)
	}
	captured, err := fileaccess.CaptureSelected(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if len(captured) != 1 || !fileaccess.HasScope(captured[0]) {
		t.Fatal("drop did not capture persistent authority")
	}
	fileaccess.ReleaseSelected(files)
}
