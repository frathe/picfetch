package macbundle

import (
	"path/filepath"
	"testing"
)

func TestRuntimeDirectoryUsesKnownAppLayouts(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Renamed PicFetch.app")
	want := filepath.Join(app, "Contents", "Frameworks")
	for _, relative := range []string{"Contents/MacOS/PicFetch", "Contents/XPCServices/io.github.frathe.picfetch.worker.xpc/Contents/MacOS/picfetch-image-worker"} {
		got, err := RuntimeDirectory(filepath.Join(app, filepath.FromSlash(relative)))
		if err != nil || got != want {
			t.Fatalf("layout %s = %q, %v", relative, got, err)
		}
	}
}

func TestRuntimeDirectoryRejectsUnrelatedExecutables(t *testing.T) {
	for _, path := range []string{"/tmp/PicFetch", "/tmp/PicFetch.app/Contents/Resources/tool", "/tmp/PicFetch.app/Contents/XPCServices/unrelated.xpc/Contents/MacOS/picfetch-image-worker", "/tmp/PicFetch.app/Contents/XPCServices/io.github.frathe.picfetch.worker.xpc/Contents/MacOS/other", "/tmp/not-an-app/Contents/MacOS/PicFetch", "relative.app/Contents/MacOS/PicFetch"} {
		if root, err := RuntimeDirectory(filepath.FromSlash(path)); err == nil {
			t.Fatalf("unrelated %q selected %q", path, root)
		}
	}
}
