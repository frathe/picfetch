package imaging

import (
	"bytes"
	"errors"
	"image/color"
	"image/jpeg"
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
				if err := Export(storage.NewFileURI(dest), markedImage(w, h), nil, ExportOptions{}); err != nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(w, h), nil, ExportOptions{}); err != nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(w, h), nil, ExportOptions{}); err != nil {
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

		if err := Export(storage.NewFileURI(filepath.Join(dir, "copy.webp")), markedImage(2, 2), nil, ExportOptions{}); err == nil {
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

		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), nil, ExportOptions{}); err == nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), src, ExportOptions{}); err != nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), src, ExportOptions{}); err != nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), nil, ExportOptions{}); err != nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(4, 3), pngSrc, ExportOptions{}); err != nil {
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
		if err := Export(storage.NewFileURI(dest), markedImage(2, 2), missing, ExportOptions{}); err != nil {
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

// mustRead is os.ReadFile with the test failing on any error, so the
// StripJPEGMetadata tests below can inline a read-back without repeating
// the error check at every call site.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestStripJPEGMetadata_RemovesGPSWithoutTouchingPixelsWhenOrientation1(t *testing.T) {
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	})...))
	path := writeTempFile(t, "gps.jpg", spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif}))
	u := storage.NewFileURI(path)

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ReadMetadata(before).HasGPS {
		t.Fatal("setup: want GPS")
	}

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ReadMetadata(after).Empty() {
		t.Fatalf("metadata left: %+v", ReadMetadata(after))
	}
	pre, err := jpeg.Decode(bytes.NewReader(before))
	if err != nil {
		t.Fatal(err)
	}
	post, err := jpeg.Decode(bytes.NewReader(after))
	if err != nil {
		t.Fatal(err)
	}
	if pre.Bounds() != post.Bounds() {
		t.Fatalf("bounds %v vs %v", pre.Bounds(), post.Bounds())
	}
}

func TestStripJPEGMetadata_Orientation6StaysUpright(t *testing.T) {
	path := writeTempFile(t, "rotated.jpg", halfRedHalfBlueJPEG(t, 20, 10, 6))
	u := storage.NewFileURI(path)

	loadedBefore, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatal(err)
	}
	b := loadedBefore.Frames[0].Bounds()
	if b.Dx() != 10 || b.Dy() != 20 {
		t.Fatalf("setup size %dx%d, want 10x20", b.Dx(), b.Dy())
	}

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	if !ReadMetadata(mustRead(t, path)).Empty() {
		t.Fatal("want no Exif after strip")
	}
	if jpegEXIFOrientation(mustRead(t, path)) != 1 {
		t.Fatal("stripped file must not carry orientation 6")
	}

	loadedAfter, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatal(err)
	}
	img := loadedAfter.Frames[0]
	if img.Bounds() != b {
		t.Fatalf("size %v, want %v (upright)", img.Bounds(), b)
	}
	r, _, b2, _ := img.At(5, 5).RGBA()
	if r < b2 {
		t.Errorf("top: want red after strip+reload")
	}
	r, _, b2, _ = img.At(5, 15).RGBA()
	if b2 < r {
		t.Errorf("bottom: want blue after strip+reload")
	}
}

func TestStripJPEGMetadata_Orientation6KeepsICC(t *testing.T) {
	orig := halfRedHalfBlueJPEG(t, 20, 10, 6)
	icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
	withICC, err := injectJPEGMetadata(orig, [][]byte{icc})
	if err != nil {
		t.Fatal(err)
	}
	path := writeTempFile(t, "rotated-icc.jpg", withICC)
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if !bytes.Contains(got, []byte("ICC_PROFILE")) {
		t.Fatal("orientation 2–8 re-encode must keep the original ICC profile")
	}
	if !ReadMetadata(got).Empty() {
		t.Fatal("want no Exif after strip")
	}
	if jpegEXIFOrientation(got) != 1 {
		t.Fatal("stripped file must not carry orientation 6")
	}

	loaded, err := LoadImage(u, DefaultImgCacheBytes)
	if err != nil {
		t.Fatal(err)
	}
	img := loaded.Frames[0]
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 20 {
		t.Fatalf("size %v, want 10x20 upright", img.Bounds())
	}
}

func TestStripJPEGMetadata_NotJPEG(t *testing.T) {
	path := writeTempFile(t, "x.png", []byte("\x89PNG\r\n"))
	err := StripJPEGMetadata(storage.NewFileURI(path))
	if !errors.Is(err, errNotJPEG) {
		t.Fatalf("err = %v, want errNotJPEG", err)
	}
}

func TestStripJPEGMetadata_NoRemovableSegmentsIsNoop(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	path := writeTempFile(t, "plain.jpg", buf.Bytes())
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()

	if err := StripJPEGMetadata(storage.NewFileURI(path)); err != nil {
		t.Fatal(err)
	}

	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().Equal(mtime) {
		t.Fatal("noop strip must not rewrite the file")
	}
}

func TestStripJPEGMetadata_PreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}

	path := writeTempFile(t, "private.jpg", halfRedHalfBlueJPEG(t, 4, 4, 6))
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatalf("chmod fixture: %v", err)
	}

	if err := StripJPEGMetadata(storage.NewFileURI(path)); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat stripped file: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o640); got != want {
		t.Errorf("stripped file permissions = %o, want %o", got, want)
	}
}

func TestStripJPEGMetadata_UpdatesSymlinkTargetWithoutReplacingTheLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.jpg")
	if err := os.WriteFile(target, halfRedHalfBlueJPEG(t, 4, 4, 6), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "photo.jpg")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	u := storage.NewFileURI(link)
	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("StripJPEGMetadata replaced the symlink instead of updating its target")
	}

	if !ReadMetadata(mustRead(t, target)).Empty() {
		t.Fatal("want no Exif in the symlink target after strip")
	}
}

func TestStripJPEGMetadata_DropsGPSTrailerAfterPrimaryEOI(t *testing.T) {
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	})...))
	primary := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif})
	data := appendAfterEOI(t, primary, gpsTrailerJPEG(t))
	path := writeTempFile(t, "mpf-trailer.jpg", data)
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if !ReadMetadata(got).Empty() {
		t.Fatalf("metadata left: %+v", ReadMetadata(got))
	}
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("trailer Exif survived the file rewrite")
	}
	if n := jpegLength(got); n != len(got) {
		t.Fatalf("file still has a trailer: jpegLength=%d len=%d", n, len(got))
	}
}

func TestStripJPEGMetadata_TrailerOnlyRewritesTheFile(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	before := appendAfterEOI(t, plain, gpsTrailerJPEG(t))
	path := writeTempFile(t, "trailer-only.jpg", before)

	if err := StripJPEGMetadata(storage.NewFileURI(path)); err != nil {
		t.Fatal(err)
	}

	got := mustRead(t, path)
	if bytes.Equal(got, before) {
		t.Fatal("trailer-only strip must rewrite the file")
	}
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("trailer Exif survived")
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("header-clean primary must be unchanged aside from dropping the trailer")
	}
}

func TestStripJPEGMetadata_Orientation6DropsTrailer(t *testing.T) {
	primary := halfRedHalfBlueJPEG(t, 20, 10, 6)
	path := writeTempFile(t, "rotated-trailer.jpg", appendAfterEOI(t, primary, gpsTrailerJPEG(t)))
	u := storage.NewFileURI(path)

	if err := StripJPEGMetadata(u); err != nil {
		t.Fatalf("StripJPEGMetadata: %v", err)
	}

	got := mustRead(t, path)
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Fatal("re-encode path left trailer Exif")
	}
	if n := jpegLength(got); n != len(got) {
		t.Fatalf("re-encode path left a trailer: jpegLength=%d len=%d", n, len(got))
	}
}

func TestCanStripJPEGMetadata(t *testing.T) {
	var plainBuf bytes.Buffer
	if err := jpeg.Encode(&plainBuf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}

	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	})...))
	gpsJPEG := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif})

	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"stdlib JPEG has nothing removable", plainBuf.Bytes(), false},
		{"GPS splice is removable", gpsJPEG, true},
		{"GPS JPEG after EOI is removable", appendAfterEOI(t, plainBuf.Bytes(), gpsTrailerJPEG(t)), true},
		{"orientation 6 must re-encode", halfRedHalfBlueJPEG(t, 4, 4, 6), true},
		{"PNG magic is not a JPEG", []byte("\x89PNG\r\n"), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanStripJPEGMetadata(c.data); got != c.want {
				t.Errorf("CanStripJPEGMetadata(%s) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

// --- ExportOptions ---------------------------------------------------------

// TestExport_DefaultOptionsWriteTheUntouchedFile pins the promise the
// options value was added under: the zero value is today's behaviour, byte
// for byte. Comparing against encodeJPEGPreservingMetadata's own output
// rather than a golden file keeps that honest without pinning the bytes
// image/jpeg happens to produce in one Go release.
func TestExport_DefaultOptionsWriteTheUntouchedFile(t *testing.T) {
	srcPath := writeTempFile(t, "geo.jpg", uitest.GPSJPEG(t, 8, 4, 48.858, 2.294))
	src := storage.NewFileURI(srcPath)
	img := markedImage(8, 4)

	dest := filepath.Join(t.TempDir(), "copy.jpg")
	if err := Export(storage.NewFileURI(dest), img, src, ExportOptions{}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	var want bytes.Buffer
	if err := encodeJPEGPreservingMetadata(&want, img, mustRead(t, srcPath), false); err != nil {
		t.Fatalf("encodeJPEGPreservingMetadata: %v", err)
	}

	if got := mustRead(t, dest); !bytes.Equal(got, want.Bytes()) {
		t.Errorf("default-options export wrote %d bytes, want the %d the metadata-preserving encode produces",
			len(got), want.Len())
	}
}

// TestExport_SizeLimitCapsTheLongestEdge covers the export size limit end
// to end - what is on disk afterwards, at every rung the prompt offers,
// including the one that is not a limit at all.
func TestExport_SizeLimitCapsTheLongestEdge(t *testing.T) {
	for _, tc := range []struct {
		name         string
		w, h         int
		maxEdge      int
		wantW, wantH int
	}{
		{"original writes the frame's own size", 900, 600, 0, 900, 600},
		{"landscape capped on its width", 900, 600, 300, 300, 200},
		{"portrait capped on its height", 600, 900, 300, 200, 300},
		{"a photo already inside the limit is never enlarged", 200, 150, 300, 200, 150},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, ext := range []string{".png", ".jpg"} {
				t.Run(ext, func(t *testing.T) {
					dest := filepath.Join(t.TempDir(), "copy"+ext)

					err := Export(storage.NewFileURI(dest), markedImage(tc.w, tc.h), nil,
						ExportOptions{MaxEdge: tc.maxEdge})
					if err != nil {
						t.Fatalf("Export: %v", err)
					}

					loaded, err := LoadImage(storage.NewFileURI(dest), DefaultImgCacheBytes)
					if err != nil {
						t.Fatalf("reload the exported file: %v", err)
					}
					if b := loaded.Frames[0].Bounds(); b.Dx() != tc.wantW || b.Dy() != tc.wantH {
						t.Errorf("exported bounds = %v, want %dx%d", b, tc.wantW, tc.wantH)
					}
				})
			}
		})
	}
}

// --- metadata omission -----------------------------------------------------
//
// Omission is the export operation: the *copy* is written without the
// source's identifying tags and the source keeps everything it had - as
// opposed to StripJPEGMetadata's metadata removal, which rewrites the
// original in place and cannot be undone.

func TestExport_OmitMetadataWritesACleanCopyAndLeavesTheSourceAlone(t *testing.T) {
	srcBytes := uitest.GPSJPEG(t, 8, 4, 48.858, 2.294)
	srcPath := writeTempFile(t, "geo.jpg", srcBytes)
	src := storage.NewFileURI(srcPath)

	dest := filepath.Join(t.TempDir(), "copy.jpg")
	err := Export(storage.NewFileURI(dest), markedImage(8, 4), src, ExportOptions{OmitMetadata: true})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	got := mustRead(t, dest)
	if ReadMetadata(got).HasGPS {
		t.Error("an omitted-metadata export still carries the source's GPS position")
	}
	for _, seg := range jpegMetadataSegments(got) {
		if isExifAPP1(seg) {
			t.Error("an omitted-metadata export still carries an Exif APP1 segment")
		}
	}

	if !ReadMetadata(mustRead(t, srcPath)).HasGPS {
		t.Error("omitting metadata from a copy must never touch the source file")
	}
}

// TestExport_OmitMetadataKeepsTheColourProfile is why omission goes through
// the ICC-preserving encode rather than a bare jpeg.Encode: the recipient's
// colours must not shift because the sender ticked a privacy box.
func TestExport_OmitMetadataKeepsTheColourProfile(t *testing.T) {
	icc := wrapAPP2(append([]byte("ICC_PROFILE\x00"), 1, 1, 'p', 'r', 'o', 'f'))
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
	srcPath := writeTempFile(t, "profiled.jpg",
		spliceMetadataIntoJPEG(t, markedImage(8, 4), [][]byte{exif, icc, wrapAPP14()}))

	dest := filepath.Join(t.TempDir(), "copy.jpg")
	err := Export(storage.NewFileURI(dest), markedImage(8, 4), storage.NewFileURI(srcPath),
		ExportOptions{OmitMetadata: true})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	got := mustRead(t, dest)
	if len(jpegICCSegments(got)) != 1 {
		t.Errorf("omitted-metadata export carries %d ICC segments, want the source's one", len(jpegICCSegments(got)))
	}
	// APP14 must not come back: it describes the *original* entropy-coded
	// colour transform, and would misdeclare what image/jpeg.Encode just
	// wrote.
	for _, seg := range jpegMetadataSegments(got) {
		if len(seg) >= 2 && seg[1] == 0xEE {
			t.Error("omitted-metadata export spliced the Adobe APP14 segment back on")
		}
	}
}

// TestExport_OmitMetadataOnAPNGIsTheSameFileEitherWay covers the reason the
// checkbox's label states its JPEG-only scope permanently instead of
// greying itself out: for a PNG the answer never mattered.
func TestExport_OmitMetadataOnAPNGIsTheSameFileEitherWay(t *testing.T) {
	src := storage.NewFileURI(writeTempFile(t, "geo.jpg", uitest.GPSJPEG(t, 8, 4, 48.858, 2.294)))
	img := markedImage(8, 4)

	kept := filepath.Join(t.TempDir(), "kept.png")
	omitted := filepath.Join(t.TempDir(), "omitted.png")
	if err := Export(storage.NewFileURI(kept), img, src, ExportOptions{}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if err := Export(storage.NewFileURI(omitted), img, src, ExportOptions{OmitMetadata: true}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	if !bytes.Equal(mustRead(t, kept), mustRead(t, omitted)) {
		t.Error("the metadata choice changed a PNG export, which carries no metadata either way")
	}
}

// TestExport_OmitMetadataAndASizeLimitTogether is the combination the
// prompt makes reachable in one keystroke, and the one where an
// implementation that resized only on the metadata-preserving branch would
// silently write full-size pixels.
func TestExport_OmitMetadataAndASizeLimitTogether(t *testing.T) {
	src := storage.NewFileURI(writeTempFile(t, "geo.jpg", uitest.GPSJPEG(t, 900, 600, 48.858, 2.294)))

	dest := filepath.Join(t.TempDir(), "copy.jpg")
	err := Export(storage.NewFileURI(dest), markedImage(900, 600), src,
		ExportOptions{MaxEdge: 300, OmitMetadata: true})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	got := mustRead(t, dest)
	if ReadMetadata(got).HasGPS {
		t.Error("a resized omitted-metadata export still carries GPS")
	}
	loaded, err := LoadImage(storage.NewFileURI(dest), DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("reload the exported file: %v", err)
	}
	if b := loaded.Frames[0].Bounds(); b.Dx() != 300 || b.Dy() != 200 {
		t.Errorf("exported bounds = %v, want 300x200", b)
	}
}

// --- dimension tags a resize invalidated -----------------------------------

// exportedDimensionTags is the tag sets a written export exposes, per IFD -
// the same view a program reading the file would get.
func exportedDimensionTags(t *testing.T, opts ExportOptions, w, h int) map[int]map[uint16][]byte {
	t.Helper()

	src := storage.NewFileURI(writeTempFile(t, "camera.jpg", dimensionTagJPEG(t, w, h)))
	dest := filepath.Join(t.TempDir(), "copy.jpg")
	if err := Export(storage.NewFileURI(dest), markedImage(w, h), src, opts); err != nil {
		t.Fatalf("Export: %v", err)
	}

	return readIFDs(t, mustRead(t, dest))
}

// TestExport_ResizeDropsTheDimensionTagsItInvalidated is the whole point of
// the removal: after a real resize, no tag in the file still claims the
// original's pixel dimensions, so a reader falls back to the JPEG frame
// header - which carries the true size for free - instead of believing a
// stale one.
func TestExport_ResizeDropsTheDimensionTagsItInvalidated(t *testing.T) {
	ifds := exportedDimensionTags(t, ExportOptions{MaxEdge: 300}, 900, 600)

	for _, tc := range []struct {
		ifd  int
		name string
		tags []uint16
	}{
		{tiffIFD0, "IFD0", []uint16{0x0100, 0x0101}},
		{tiffExifIFD, "the Exif SubIFD", []uint16{0xA002, 0xA003, 0x9214, 0xA214}},
		{tiffInteropIFD, "the Interoperability IFD", []uint16{0x1001, 0x1002}},
	} {
		if ifds[tc.ifd] == nil {
			t.Fatalf("%s is missing from the written file entirely", tc.name)
		}
		for _, tag := range tc.tags {
			if _, present := ifds[tc.ifd][tag]; present {
				t.Errorf("%s still carries tag %#04x after a resize", tc.name, tag)
			}
		}
	}
}

// TestExport_ResizeKeepsEverythingItDidNotInvalidate is the other half, and
// the more important one: dropping the tags a resize made false must not
// cost the user the metadata they explicitly asked to keep. MakerNote stays
// because it cannot be audited, and the resolution/DPI trio because it
// states print density rather than a pixel count.
func TestExport_ResizeKeepsEverythingItDidNotInvalidate(t *testing.T) {
	ifds := exportedDimensionTags(t, ExportOptions{MaxEdge: 300}, 900, 600)

	for _, tc := range []struct {
		ifd  int
		name string
		tag  uint16
		what string
	}{
		{tiffIFD0, "IFD0", 0x010F, "Make"},
		{tiffIFD0, "IFD0", 0x0110, "Model"},
		{tiffIFD0, "IFD0", 0x011A, "XResolution"},
		{tiffIFD0, "IFD0", 0x011B, "YResolution"},
		{tiffIFD0, "IFD0", 0x0128, "ResolutionUnit"},
		{tiffIFD0, "IFD0", 0x8825, "the GPS pointer"},
		{tiffExifIFD, "the Exif SubIFD", 0x829A, "ExposureTime"},
		{tiffExifIFD, "the Exif SubIFD", 0x9003, "DateTimeOriginal"},
		{tiffExifIFD, "the Exif SubIFD", 0x927C, "MakerNote"},
		{tiffExifIFD, "the Exif SubIFD", 0xA434, "LensModel"},
		{tiffInteropIFD, "the Interoperability IFD", 0x0001, "InteropIndex"},
	} {
		if _, present := ifds[tc.ifd][tc.tag]; !present {
			t.Errorf("%s lost %s (%#04x), which a resize does not invalidate", tc.name, tc.what, tc.tag)
		}
	}
}

// TestExport_ResizeLeavesSurvivingValuesReadable is what proves the entries
// were moved rather than the file scrambled: every value below lives in the
// TIFF's trailing value area at an absolute offset, so a removal that
// shifted those offsets - or corrupted an entry while compacting - would
// read back as garbage rather than as an error.
func TestExport_ResizeLeavesSurvivingValuesReadable(t *testing.T) {
	src := storage.NewFileURI(writeTempFile(t, "camera.jpg", dimensionTagJPEG(t, 900, 600)))
	dest := filepath.Join(t.TempDir(), "copy.jpg")
	if err := Export(storage.NewFileURI(dest), markedImage(900, 600), src, ExportOptions{MaxEdge: 300}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	got := mustRead(t, dest)

	m := ReadMetadata(got)
	if m.Make != "Canon" || m.Model != "EOS 90D" {
		t.Errorf("camera = %q / %q, want Canon / EOS 90D", m.Make, m.Model)
	}
	if m.LensModel != "EF50mm f/1.8" {
		t.Errorf("LensModel = %q, want EF50mm f/1.8", m.LensModel)
	}
	if !m.HasGPS {
		t.Error("the GPS position did not survive the removal")
	}
	if m.ExposureTime == "" {
		t.Error("ExposureTime did not survive the removal")
	}

	ifds := readIFDs(t, got)
	if note := string(ifds[tiffExifIFD][0x927C]); note != "MAKERNOTE-8" {
		t.Errorf("MakerNote reads %q, want it byte-for-byte intact", note)
	}

	// Orientation is still normalized to 1 on the way out, exactly as it
	// was before any of this: the pixels written are already upright.
	if orient := ifds[tiffIFD0][0x0112]; len(orient) < 2 || orient[0] != 1 {
		t.Errorf("Orientation reads % x, want the normalized 1", orient)
	}
}

// TestExport_KeepsDimensionTagsWhenThePixelsDidNotChange pins the trigger:
// it is that the pixels actually changed, not that a rung was selected.
func TestExport_KeepsDimensionTagsWhenThePixelsDidNotChange(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts ExportOptions
	}{
		{"original size", ExportOptions{}},
		{"a limit larger than the photo", ExportOptions{MaxEdge: 2400}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ifds := exportedDimensionTags(t, tc.opts, 900, 600)

			for _, tag := range []uint16{0x0100, 0x0101} {
				if _, present := ifds[tiffIFD0][tag]; !present {
					t.Errorf("IFD0 lost tag %#04x, which nothing invalidated", tag)
				}
			}
			for _, tag := range []uint16{0xA002, 0xA003, 0x9214, 0xA214} {
				if _, present := ifds[tiffExifIFD][tag]; !present {
					t.Errorf("the Exif SubIFD lost tag %#04x, which nothing invalidated", tag)
				}
			}
			for _, tag := range []uint16{0x1001, 0x1002} {
				if _, present := ifds[tiffInteropIFD][tag]; !present {
					t.Errorf("the Interoperability IFD lost tag %#04x, which nothing invalidated", tag)
				}
			}
		})
	}
}

// TestSaveRotated_LeavesDimensionTagsAlone guards the path this feature
// must not touch: Save Changes resizes nothing, so its metadata
// normalization has to keep behaving exactly as it did.
func TestSaveRotated_LeavesDimensionTagsAlone(t *testing.T) {
	path := writeTempFile(t, "camera.jpg", dimensionTagJPEG(t, 90, 60))

	if err := SaveRotated(storage.NewFileURI(path), markedImage(60, 90)); err != nil {
		t.Fatalf("SaveRotated: %v", err)
	}

	ifds := readIFDs(t, mustRead(t, path))
	for _, tag := range []uint16{0x0100, 0x0101} {
		if _, present := ifds[tiffIFD0][tag]; !present {
			t.Errorf("a rotate-and-save dropped IFD0 tag %#04x", tag)
		}
	}
	for _, tag := range []uint16{0xA002, 0xA003} {
		if _, present := ifds[tiffExifIFD][tag]; !present {
			t.Errorf("a rotate-and-save dropped Exif SubIFD tag %#04x", tag)
		}
	}
}
