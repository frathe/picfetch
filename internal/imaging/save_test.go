package imaging

import (
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestCanEncode(t *testing.T) {
	cases := []struct {
		name string
		u    fakeURI
		want bool
	}{
		{"jpg", fakeURI{name: "a.jpg", ext: ".jpg"}, true},
		{"jpeg uppercase extension", fakeURI{name: "A.JPEG", ext: ".JPEG"}, true},
		{"png", fakeURI{name: "a.png", ext: ".png"}, true},
		{"gif", fakeURI{name: "a.gif", ext: ".gif"}, true},
		{"bmp", fakeURI{name: "a.bmp", ext: ".bmp"}, true},
		{"tif", fakeURI{name: "a.tif", ext: ".tif"}, true},
		{"tiff", fakeURI{name: "a.tiff", ext: ".tiff"}, true},
		{"avif", fakeURI{name: "a.avif", ext: ".avif"}, true},
		{"webp is decode-only, no encoder", fakeURI{name: "a.webp", ext: ".webp"}, false},
		{"heic is decode-only, no encoder", fakeURI{name: "a.heic", ext: ".heic"}, false},
		{"ico unsupported", fakeURI{name: "a.ico", ext: ".ico"}, false},
		{"xpm unsupported", fakeURI{name: "a.xpm", ext: ".xpm"}, false},
		{"no extension", fakeURI{name: "a", ext: ""}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanEncode(c.u); got != c.want {
				t.Errorf("CanEncode(%+v) = %v, want %v", c.u, got, c.want)
			}
		})
	}
}

// TestCanEncodeExt covers the extension check on its own, without
// CanEncode's symlink resolution: it is what the export path asks, since an
// export destination is a name the user just typed rather than a file
// already on disk.
func TestCanEncodeExt(t *testing.T) {
	cases := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".JPG", true},
		{".webp", false},
		{".heic", false},
		{"", false},
		{"png", false}, // no leading dot: not what filepath.Ext or URI.Extension ever produce
	}

	for _, c := range cases {
		t.Run(c.ext, func(t *testing.T) {
			if got := CanEncodeExt(c.ext); got != c.want {
				t.Errorf("CanEncodeExt(%q) = %v, want %v", c.ext, got, c.want)
			}
		})
	}
}

func TestSaveRotated(t *testing.T) {
	t.Run("overwrites the file with the given pixels, exactly for a lossless format", func(t *testing.T) {
		path := writeTempFile(t, "photo.png", []byte("placeholder, never read back"))
		u := storage.NewFileURI(path)

		const w, h = 3, 2
		if err := SaveRotated(u, markedImage(w, h)); err != nil {
			t.Fatalf("SaveRotated: %v", err)
		}

		got, err := LoadImage(u, DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("reload after save: %v", err)
		}

		b := got.Frames[0].Bounds()
		if b.Dx() != w || b.Dy() != h {
			t.Fatalf("bounds after save = %v, want %dx%d", b, w, h)
		}

		for y := range h {
			for x := range w {
				c := got.Frames[0].At(x, y).(color.RGBA)
				if int(c.R) != x || int(c.G) != y {
					t.Errorf("(%d,%d) = (%d,%d), want (%d,%d)", x, y, c.R, c.G, x, y)
				}
			}
		}
	})

	t.Run("JPEG keeps Exif and does not double-apply orientation on reload", func(t *testing.T) {
		orig := halfRedHalfBlueJPEG(t, 20, 10, 6)
		path := writeTempFile(t, "rotated.jpg", orig)
		u := storage.NewFileURI(path)

		loaded, err := LoadImage(u, DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("load original: %v", err)
		}
		oriented := loaded.Frames[0]
		if b := oriented.Bounds(); b.Dx() != 10 || b.Dy() != 20 {
			t.Fatalf("oriented bounds = %v, want 10x20", b)
		}

		if err := SaveRotated(u, oriented); err != nil {
			t.Fatalf("SaveRotated: %v", err)
		}

		saved, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if jpegEXIFOrientation(saved) != 1 {
			t.Errorf("saved orientation tag = %d, want 1", jpegEXIFOrientation(saved))
		}
		hasExif := slices.ContainsFunc(jpegMetadataSegments(saved), isExifAPP1)
		if !hasExif {
			t.Fatal("saved JPEG lost Exif APP1")
		}

		reloaded, err := LoadImage(u, DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		b := reloaded.Frames[0].Bounds()
		if b.Dx() != 10 || b.Dy() != 20 {
			t.Errorf("reloaded bounds = %v, want 10x20 (must not apply orientation 6 again)", b)
		}
	})

	t.Run("unsupported format returns an error and leaves the file untouched", func(t *testing.T) {
		original := []byte("not a real webp, but SaveRotated should never get far enough to care")
		path := writeTempFile(t, "photo.webp", original)
		u := storage.NewFileURI(path)

		if err := SaveRotated(u, markedImage(2, 2)); err == nil {
			t.Fatal("SaveRotated: want error for unsupported format, got nil")
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != string(original) {
			t.Error("SaveRotated modified the file despite returning an error")
		}
	})

	t.Run("preserves the original file permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not expose Unix permission bits")
		}

		path := writeTempFile(t, "private.png", []byte("placeholder"))
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatalf("chmod fixture: %v", err)
		}

		if err := SaveRotated(storage.NewFileURI(path), markedImage(2, 2)); err != nil {
			t.Fatalf("SaveRotated: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat saved file: %v", err)
		}
		if got, want := info.Mode().Perm(), os.FileMode(0o640); got != want {
			t.Errorf("saved file permissions = %o, want %o", got, want)
		}
	})

	t.Run("updates a symlink target without replacing the link", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.png")
		if err := os.WriteFile(target, []byte("placeholder"), 0o600); err != nil {
			t.Fatalf("write target: %v", err)
		}
		link := filepath.Join(dir, "photo.webp")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		u := storage.NewFileURI(link)
		if !CanEncode(u) {
			t.Fatal("CanEncode returned false for a link to an encodable PNG target")
		}
		if err := SaveRotated(u, markedImage(3, 2)); err != nil {
			t.Fatalf("SaveRotated: %v", err)
		}

		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("lstat link: %v", err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Error("SaveRotated replaced the symlink instead of updating its target")
		}

		loaded, err := LoadImage(storage.NewFileURI(target), DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("load saved target: %v", err)
		}
		if got := loaded.Frames[0].Bounds(); got.Dx() != 3 || got.Dy() != 2 {
			t.Errorf("saved target bounds = %v, want 3x2", got)
		}
	})

	t.Run("every broadened format round-trips to the new dimensions", func(t *testing.T) {
		for _, ext := range []string{".jpg", ".png", ".gif", ".bmp", ".tiff", ".avif"} {
			t.Run(ext, func(t *testing.T) {
				path := writeTempFile(t, "photo"+ext, nil)
				u := storage.NewFileURI(path)

				const w, h = 4, 3
				if err := SaveRotated(u, markedImage(w, h)); err != nil {
					t.Fatalf("SaveRotated: %v", err)
				}

				loaded, err := LoadImage(u, DefaultImgCacheBytes)
				if err != nil {
					t.Fatalf("reload after save: %v", err)
				}
				if b := loaded.Frames[0].Bounds(); b.Dx() != w || b.Dy() != h {
					t.Errorf("bounds after save = %v, want %dx%d", b, w, h)
				}
			})
		}
	})
}

func TestExport(t *testing.T) {
	t.Run("creates a new file in the destination's own format", func(t *testing.T) {
		for _, ext := range []string{".png", ".jpg"} {
			t.Run(ext, func(t *testing.T) {
				dest := filepath.Join(t.TempDir(), "copy"+ext)

				const w, h = 4, 3
				if err := Export(storage.NewFileURI(dest), markedImage(w, h), nil); err != nil {
					t.Fatalf("Export: %v", err)
				}

				loaded, err := LoadImage(storage.NewFileURI(dest), DefaultImgCacheBytes)
				if err != nil {
					t.Fatalf("reload the exported file: %v", err)
				}
				if b := loaded.Frames[0].Bounds(); b.Dx() != w || b.Dy() != h {
					t.Errorf("exported bounds = %v, want %dx%d", b, w, h)
				}
			})
		}
	})

	// The point of Export over SaveRotated: the destination's format is
	// chosen by where the pixels are going, with no relationship at all to
	// where they came from - which is what makes exporting a WebP or HEIC
	// (formats this module can decode but never encode) possible.
	t.Run("encodes by the destination extension, exactly for a lossless format", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "from-a-webp.png")

		const w, h = 3, 2
		if err := Export(storage.NewFileURI(dest), markedImage(w, h), nil); err != nil {
			t.Fatalf("Export: %v", err)
		}

		loaded, err := LoadImage(storage.NewFileURI(dest), DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("reload the exported file: %v", err)
		}
		for y := range h {
			for x := range w {
				c := loaded.Frames[0].At(x, y).(color.RGBA)
				if int(c.R) != x || int(c.G) != y {
					t.Errorf("(%d,%d) = (%d,%d), want (%d,%d)", x, y, c.R, c.G, x, y)
				}
			}
		}
	})

	t.Run("overwrites an existing destination", func(t *testing.T) {
		dest := writeTempFile(t, "existing.png", []byte("placeholder, never read back"))

		const w, h = 5, 2
		if err := Export(storage.NewFileURI(dest), markedImage(w, h), nil); err != nil {
			t.Fatalf("Export: %v", err)
		}

		loaded, err := LoadImage(storage.NewFileURI(dest), DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("reload the exported file: %v", err)
		}
		if b := loaded.Frames[0].Bounds(); b.Dx() != w || b.Dy() != h {
			t.Errorf("exported bounds = %v, want %dx%d", b, w, h)
		}
	})

	// A destination this module has no encoder for must fail before any
	// file exists - including the temp file the atomic write goes through,
	// which would otherwise be left behind in the user's chosen folder.
	t.Run("unsupported destination format writes nothing at all", func(t *testing.T) {
		dir := t.TempDir()

		if err := Export(storage.NewFileURI(filepath.Join(dir, "copy.webp")), markedImage(2, 2), nil); err == nil {
			t.Fatal("Export: want error for a format with no encoder, got nil")
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("destination directory holds %d entries, want none left behind", len(entries))
		}
	})

	t.Run("leaves an existing destination untouched when the encode fails", func(t *testing.T) {
		original := []byte("the previous copy, which a failed export must not damage")
		dest := writeTempFile(t, "copy.webp", original)

		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), nil); err == nil {
			t.Fatal("Export: want error for a format with no encoder, got nil")
		}

		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != string(original) {
			t.Error("Export modified the destination despite returning an error")
		}
	})
}

func TestExport_JPEGSourceKeepsMetadataOnJPEGDest(t *testing.T) {
	srcPath := writeTempFile(t, "geo.jpg", uitest.GPSJPEG(t, 8, 4, 48.858, 2.294))
	src := storage.NewFileURI(srcPath)

	t.Run("jpeg dest", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "copy.jpg")
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), src); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !ReadMetadata(got).HasGPS {
			t.Fatal("JPEG→JPEG export dropped GPS")
		}
	})

	t.Run("png dest stays a PNG without JPEG APPn", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "copy.png")
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), src); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) < 4 || string(got[:4]) != "\x89PNG" {
			t.Fatal("PNG export must still be a PNG")
		}
		if jpegMetadataSegments(got) != nil {
			t.Fatal("PNG export must not carry JPEG segments")
		}
	})

	t.Run("nil src still encodes", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "bare.jpg")
		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), nil); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(got); len(segs) != 0 {
			t.Errorf("nil src invented %d metadata segments", len(segs))
		}
	})

	t.Run("non-JPEG src to JPEG dest still encodes without splicing", func(t *testing.T) {
		pngSrc := storage.NewFileURI(writeTempFile(t, "source.png", uitest.EncodePNG(t, 4, 3, color.White)))
		dest := filepath.Join(t.TempDir(), "copy.jpg")
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), pngSrc); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) < 2 || got[0] != 0xFF || got[1] != 0xD8 {
			t.Fatal("JPEG dest must still start with SOI")
		}
		if segs := jpegMetadataSegments(got); len(segs) != 0 {
			t.Errorf("non-JPEG src spliced %d segments into JPEG dest", len(segs))
		}
	})

	t.Run("unreadable src still encodes", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "copy.jpg")
		missing := storage.NewFileURI(filepath.Join(t.TempDir(), "gone.jpg"))
		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), missing); err != nil {
			t.Fatalf("unreadable src must not fail export: %v", err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(got); len(segs) != 0 {
			t.Errorf("unreadable src invented %d metadata segments", len(segs))
		}
	})
}

// TestJpegFileBytes covers jpegFileBytes on its own: the magic-byte peek
// Export relies on to decide whether a source is worth reading in full,
// independent of the source's file extension.
func TestJpegFileBytes(t *testing.T) {
	t.Run("JPEG magic returns the full contents", func(t *testing.T) {
		data := uitest.EncodeJPEG(t, 4, 3, color.White)
		path := writeTempFile(t, "photo.jpg", data)

		got, err := jpegFileBytes(path)
		if err != nil {
			t.Fatalf("jpegFileBytes: %v", err)
		}
		if string(got) != string(data) {
			t.Error("jpegFileBytes did not return the full file contents")
		}
	})

	t.Run("non-JPEG magic returns nil, nil without an error", func(t *testing.T) {
		path := writeTempFile(t, "photo.png", uitest.EncodePNG(t, 4, 3, color.White))

		got, err := jpegFileBytes(path)
		if err != nil {
			t.Fatalf("jpegFileBytes: %v", err)
		}
		if got != nil {
			t.Errorf("jpegFileBytes(png) = %d bytes, want nil", len(got))
		}
	})

	t.Run("JPEG content wins over a non-JPEG extension", func(t *testing.T) {
		path := writeTempFile(t, "mislabeled.png", uitest.EncodeJPEG(t, 4, 3, color.White))

		got, err := jpegFileBytes(path)
		if err != nil {
			t.Fatalf("jpegFileBytes: %v", err)
		}
		if got == nil {
			t.Error("jpegFileBytes(mislabeled JPEG) = nil, want contents")
		}
	})

	t.Run("too short to hold a magic number", func(t *testing.T) {
		path := writeTempFile(t, "truncated.jpg", []byte{0xFF})

		got, err := jpegFileBytes(path)
		if err != nil {
			t.Fatalf("jpegFileBytes: %v", err)
		}
		if got != nil {
			t.Errorf("jpegFileBytes(1 byte) = %d bytes, want nil", len(got))
		}
	})

	t.Run("missing file returns an error", func(t *testing.T) {
		if _, err := jpegFileBytes(filepath.Join(t.TempDir(), "gone.jpg")); err == nil {
			t.Fatal("jpegFileBytes: want error for a missing file, got nil")
		}
	})
}
