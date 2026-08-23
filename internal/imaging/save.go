package imaging

import (
	"context"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"

	"github.com/gen2brain/avif"
)

// jpegSaveQuality is used instead of image/jpeg's own default (75, quite
// lossy) since SaveRotated is re-encoding a photo that was very likely
// already JPEG-compressed once; keeping it high limits how much additional
// generation loss one rotate-and-save round trip adds.
const jpegSaveQuality = 95

// encoders maps a lowercased file extension to the function that encodes an
// image.Image in that format. Every entry's decoder is already linked into
// the binary for IsSupportedImage's sake (see the package doc's import
// block), so adding the matching Encode call here costs nothing extra.
// WebP and HEIC are decode-only in the libraries this module depends on
// (golang.org/x/image/webp and github.com/gen2brain/heic expose no Encode),
// and ICO/XPM aren't meaningful save targets for a rotated photo, so none of
// the four appear here - CanEncode reports false for them, and SaveRotated
// refuses before touching the file.
var encoders = map[string]func(io.Writer, image.Image) error{
	".jpg":  encodeJPEGForSave,
	".jpeg": encodeJPEGForSave,
	".jpe":  encodeJPEGForSave,
	".jfif": encodeJPEGForSave,
	".png":  png.Encode,
	".gif":  func(w io.Writer, img image.Image) error { return gif.Encode(w, img, nil) },
	".bmp":  bmp.Encode,
	".tif":  func(w io.Writer, img image.Image) error { return tiff.Encode(w, img, nil) },
	".tiff": func(w io.Writer, img image.Image) error { return tiff.Encode(w, img, nil) },
	".avif": func(w io.Writer, img image.Image) error { return avif.Encode(w, img) },
}

func encodeJPEGForSave(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: jpegSaveQuality})
}

func isJPEGExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".jpe", ".jfif":
		return true
	default:
		return false
	}
}

// CanEncode reports whether SaveRotated has an encoder for u's format, so a
// caller (internal/ui's canSaveRotation) can decide whether to offer saving
// at all instead of finding out only after attempting it. It resolves a
// symlink first, matching SaveRotated's own behavior: what governs there is
// the format of the file that will actually be written.
func CanEncode(u fyne.URI) bool {
	ext := u.Extension()
	if path, err := filepath.EvalSymlinks(u.Path()); err == nil {
		ext = filepath.Ext(path)
	}
	return CanEncodeExt(ext)
}

// CanEncodeExt reports whether ext (a leading-dot file extension, as
// filepath.Ext and fyne.URI.Extension both produce, in any case) has an
// encoder. It is the check the export path wants - internal/ui asks it
// about a destination the user just named, which may not exist yet and so
// has no symlink for CanEncode above to resolve.
func CanEncodeExt(ext string) bool {
	_, ok := encoders[strings.ToLower(ext)]
	return ok
}

// SaveRotated writes img - a caller's already-rotated, already-oriented
// frame, typically internal/ui's v.img.Image - back to u, re-encoded in the
// target file's format, replacing the file's previous contents.
// For JPEG, SaveRotated copies the original metadata segments onto the
// re-encoded file with Exif Orientation reset to 1. Other formats still
// do not carry metadata.
//
// It resolves a symlink before writing, so saving an image opened through a
// link updates the target instead of replacing the link itself, and the
// replacement keeps the original file's permission bits. See writeEncoded
// for the atomic write both this and Export go through.
func SaveRotated(u fyne.URI, img image.Image) error {
	path, err := filepath.EvalSymlinks(u.Path())
	if err != nil {
		return err
	}

	ext := filepath.Ext(path)
	encode, ok := encoders[strings.ToLower(ext)]
	if !ok {
		return &UnsupportedSaveFormatError{ext: ext}
	}

	if isJPEGExt(ext) {
		orig, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		encode = func(w io.Writer, img image.Image) error {
			return encodeJPEGPreservingMetadata(w, img, orig)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	return writeEncoded(path, info.Mode().Perm(), encode, img)
}

// defaultExportPerm is what a file Export creates from scratch gets, the
// same 0644-before-umask a plain os.WriteFile would produce - os.CreateTemp
// opens at 0600, so writeEncoded has to be told the mode either way.
const defaultExportPerm = 0o644

// Export writes img to dest, encoded in dest's format. src is the file
// the pixels came from and may be nil. When dest is JPEG and src is a
// readable JPEG, dest receives a normalized copy of src's metadata
// segments (same rules as SaveRotated). A read failure on src does not
// fail the export: pixels are written without metadata.
//
// The destination's extension alone picks the encoder: unlike SaveRotated,
// no symlink is resolved first, since dest is a destination the user just
// named rather than a file already open in the viewer, and the format they
// typed is the format they asked for. An existing destination is replaced
// (keeping its own permission bits), atomically, by the same
// temp-file-then-rename writeEncoded gives SaveRotated - so an export over
// a previous copy cannot damage it if the encode fails partway.
func Export(dest fyne.URI, img image.Image, src fyne.URI) error {
	ext := dest.Extension()
	encode, ok := encoders[strings.ToLower(ext)]
	if !ok {
		return &UnsupportedSaveFormatError{ext: ext}
	}
	path := dest.Path()
	perm := os.FileMode(defaultExportPerm)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	if isJPEGExt(ext) && src != nil && src.Path() != "" {
		if orig, err := jpegFileBytes(src.Path()); err == nil && orig != nil {
			encode = func(w io.Writer, img image.Image) error {
				return encodeJPEGPreservingMetadata(w, img, orig)
			}
		}
	}

	return writeEncoded(path, perm, encode, img)
}

// jpegFileBytes returns path's full contents if it starts with the JPEG SOI
// marker (FF D8), or (nil, nil) if it doesn't - so Export can decide whether
// a source is worth reading for metadata without an extension check (a
// JPEG with a wrong or missing extension must still work) and without
// reading the rest of a large non-JPEG source it has no use for. A file
// that can't be opened or read returns the error; Export treats that the
// same as (nil, nil), since an unreadable source must not fail the export.
func jpegFileBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, nil
		}
		return nil, err
	}
	if magic[0] != 0xFF || magic[1] != 0xD8 {
		return nil, nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}

// writeEncoded encodes img into a temp file in path's own directory and
// renames it over path only once the encode has fully succeeded, so a
// failed or interrupted encode can never leave the destination truncated or
// corrupted - and, since the rename is within one directory, never leaves a
// half-written file where the caller asked for a whole one either. Shared
// by SaveRotated (overwriting the file on screen) and Export (writing a
// copy elsewhere), which differ only in how they arrive at path, perm, and
// encode.
func writeEncoded(path string, perm os.FileMode, encode func(io.Writer, image.Image) error, img image.Image) error {
	return writeFile(path, perm, func(w io.Writer) error { return encode(w, img) })
}

// writeFile is writeEncoded's underlying atomic write, generalized to any
// write func rather than an (encode, img) pair: temp file in path's own
// directory, Chmod(perm), write, Sync, Close, Rename - so a failed or
// interrupted write can never leave the destination truncated or
// corrupted, and never leaves a half-written file behind either, since the
// rename stays within one directory. StripJPEGMetadata uses this directly
// (writing already-encoded bytes rather than encoding an image.Image).
func writeFile(path string, perm os.FileMode, write func(io.Writer) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".picfetch-save-*"+filepath.Ext(path))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// A no-op once the rename below has already moved tmpPath to path; left
	// unchecked deliberately, since its only job left by then is to clean up
	// on an error return above.
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// StripJPEGMetadata removes identifying metadata from the JPEG at u
// (Exif, XMP, IPTC, COM, MPF) in place, keeping JFIF APP0, Adobe APP14,
// and ICC. Bytes after the primary EOI (a concatenated second JPEG,
// motion-photo video) are discarded. CanStripJPEGMetadata is true when
// those bytes are the only thing left to remove. When the file's Exif
// Orientation is 2–8, the pixels are
// decoded with that orientation applied and re-encoded at jpegSaveQuality
// so the photo does not appear sideways after the tag is gone; ICC APP2
// from the original is spliced back (Adobe APP14 is not: it would
// misdeclare image/jpeg.Encode's color transform). On orientation 1 the
// lossless header walk keeps APP14 as well.
//
// A non-JPEG returns errNotJPEG and does not write. A JPEG with nothing
// removable returns nil without rewriting the file. The write is the
// same temp-file-then-rename as SaveRotated, through a symlink to the
// target, preserving permission bits.
func StripJPEGMetadata(u fyne.URI) error {
	path, err := filepath.EvalSymlinks(u.Path())
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return errNotJPEG
	}
	orient := jpegEXIFOrientation(data)
	if !jpegHasRemovableMetadata(data) && orient == 1 {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if orient == 1 {
		stripped, err := stripJPEGSegments(data)
		if err != nil {
			return err
		}
		return writeFile(path, info.Mode().Perm(), func(w io.Writer) error {
			_, err := w.Write(stripped)
			return err
		})
	}

	loaded, err := DecodeLoaded(context.Background(), data, 0)
	if err != nil {
		return err
	}
	if len(loaded.Frames) == 0 {
		return errNotJPEG
	}
	return writeFile(path, info.Mode().Perm(), func(w io.Writer) error {
		return encodeJPEGKeepingICC(w, loaded.Frames[0], data)
	})
}

// CanStripJPEGMetadata reports whether StripJPEGMetadata would rewrite
// data. False for non-JPEG. True when there is a removable COM/APPn
// segment, bytes after the primary EOI (a concatenated second JPEG or
// motion-photo video), or when Exif Orientation is 2–8 (those files must
// be re-encoded so they stay upright).
func CanStripJPEGMetadata(data []byte) bool {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return false
	}
	return jpegHasRemovableMetadata(data) || jpegEXIFOrientation(data) != 1
}

// UnsupportedSaveFormatError reports that SaveRotated has no encoder for a
// file's extension.
type UnsupportedSaveFormatError struct {
	ext string
}

func (e *UnsupportedSaveFormatError) Error() string {
	return "saving " + e.ext + " images isn't supported"
}
