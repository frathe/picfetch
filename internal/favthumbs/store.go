package favthumbs

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favstore"
)

// jpegQuality is the quality passed to image/jpeg when encoding a preview.
// 85 is comfortably above the encoder's own default of 75, which matters
// more here than usual since a preview is re-decoded and displayed at grid
// size on every open - visible blockiness would be far more noticeable than
// it is on a one-off export.
const jpegQuality = 85

// Only final cache mutations share this lock; codecs and temporary writes run
// independently. A failed reader cannot unlink a concurrently installed entry
// between checking its identity and removing it.
var previewCommitMu sync.Mutex

// Write stores thumb as the current preview for src under owner, replacing
// any preview already on disk for it. It fails when src cannot be
// identified with EntryName - typically because src no longer exists -
// since a preview's filename is how Read later recognizes which version of
// src it belongs to.
//
// The encode happens into a temp file inside the destination directory,
// which is then renamed into place, mirroring favstore.Save: a reader can
// never observe a partially written preview, and a failed encode never
// clobbers a good one that was there before.
func Write(owner *favstore.Owner, src fyne.URI, thumb image.Image) error {
	return WriteContext(context.Background(), owner, src, thumb)
}

// WriteContext checks cancellation before encoding, at encoder writes, and
// before committing the temporary file. In-flight sampling/filesystem calls
// must return before a boundary can observe cancellation.
func WriteContext(ctx context.Context, owner *favstore.Owner, src fyne.URI, thumb image.Image) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, ok := EntryName(src)
	if !ok {
		return fmt.Errorf("favthumbs: cannot determine entry name for %v", src)
	}
	return writeEntryContext(ctx, owner, name, thumb)
}

func acquirePreview(ctx context.Context, owner *favstore.Owner) (*favstore.Access, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if owner == nil {
		return nil, favstore.ErrRetired
	}
	return owner.Acquire(ctx)
}

// name is captured before producing pixels, so a later source replacement
// cannot label those pixels as the new file version.
func writeEntryContext(ctx context.Context, owner *favstore.Owner, name string, thumb image.Image) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if thumb == nil {
		return errors.New("favthumbs: nil thumbnail")
	}
	access, err := acquirePreview(ctx, owner)
	if err != nil {
		return err
	}
	defer func() { _ = access.Close() }()
	// Only the cache child may be created. Recreating a Favorite by pathname
	// would give a retired pass ownership of a replacement directory.
	if err := access.Root.Mkdir(SubDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	ext := ".jpg"
	if hasAlpha(thumb) {
		ext = ".png"
	}

	if err := access.Current(ctx); err != nil {
		return err
	}
	tmpPath := filepath.Join(SubDir, ".favthumb-"+rand.Text()+ext)
	tmp, err := access.Root.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	// A no-op once the rename below has already moved tmpPath into place;
	// left unchecked deliberately, since its only job left by then is
	// cleanup on an error return above.
	defer func() { _ = access.Root.Remove(tmpPath) }()

	var encErr error
	if ext == ".png" {
		encErr = png.Encode(contextWriter{ctx: ctx, w: tmp}, thumb)
	} else {
		encErr = jpeg.Encode(contextWriter{ctx: ctx, w: tmp}, thumb, &jpeg.Options{Quality: jpegQuality})
	}
	if encErr != nil {
		_ = tmp.Close()
		return encErr
	}

	if err := ctx.Err(); err != nil {
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

	if err := ctx.Err(); err != nil {
		return err
	}
	previewCommitMu.Lock()
	defer previewCommitMu.Unlock()
	if err := access.Current(ctx); err != nil {
		return err
	}
	if err := access.Root.Rename(tmpPath, filepath.Join(SubDir, name+ext)); err != nil {
		return err
	}

	// Both extensions hang off the same base name, so a thumbnail that
	// gained or lost transparency for an otherwise unchanged source would
	// leave the previous one beside it. Read probes .jpg first and Sweep
	// counts the base as expected whichever extension carries it, so that
	// leftover would win every lookup from here on with nothing to ever
	// clear it. Unlinked best-effort: failing to drop a superseded sibling
	// is not worth failing a preview that was written successfully.
	sibling := ".png"
	if ext == ".png" {
		sibling = ".jpg"
	}
	if access.Current(ctx) == nil {
		_ = access.Root.Remove(filepath.Join(SubDir, name+sibling))
	}

	return nil
}

// hasAlpha reports whether thumb has any pixel that is not fully opaque,
// which decides Write's choice between the lossy JPEG path (fine for a
// photo preview with no transparency to lose) and PNG (lossless, needed to
// keep an alpha channel at all).
//
// Every stdlib image type already implements Opaque() bool, and each one's
// implementation is both the fastest and the most correct way to answer
// this for that type: a constant true for formats that carry no alpha
// channel at all (*image.Gray, *image.YCbCr, *image.CMYK), a palette check
// for *image.Paletted, and a full pixel scan only for the two types where
// one is actually needed (*image.RGBA, *image.NRGBA). Re-deriving that
// per-type knowledge with a hand-rolled type switch would just be a worse
// duplicate of what the standard library already got right, so this asserts
// the interface and only falls back to a manual scan for an image.Image
// that doesn't implement it.
func hasAlpha(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return !o.Opaque()
	}

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff {
				return true
			}
		}
	}
	return false
}

// Read returns the current preview for src stored under owner. It reports
// false when there is no preview, when the stored preview is for an older
// version of src, or when the stored preview cannot be decoded. A corrupt
// preview is removed best-effort so a later memory hit can repair it.
func Read(owner *favstore.Owner, src fyne.URI) (image.Image, bool) {
	img, ok, _ := ReadContext(context.Background(), owner, src)
	return img, ok
}

// ReadContext retains Read's tolerant cache-miss policy while reporting
// cancellation distinctly, so a stopped pass cannot decode the original next.
func ReadContext(ctx context.Context, owner *favstore.Owner, src fyne.URI) (image.Image, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	name, ok := EntryName(src)
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	access, err := acquirePreview(ctx, owner)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = access.Close() }()
	for _, ext := range [...]string{".jpg", ".png"} {
		if err := access.Current(ctx); err != nil {
			return nil, false, err
		}
		img, ok := decodeFile(ctx, access, filepath.Join(SubDir, name+ext))
		if err := access.Current(ctx); err != nil {
			return nil, false, err
		}
		if ok {
			return img, true, nil
		}
	}
	return nil, false, nil
}

// hasCurrentPreview reports whether a current preview for src is already
// stored under owner, without decoding it. Sync uses this instead of Read
// for the one case where the pixels are not wanted - the caller already
// holds the thumbnail in memory and only needs to know whether disk is
// behind - since decoding a JPEG purely to learn that a file exists is
// work nobody consumes.
func hasCurrentPreview(owner *favstore.Owner, src fyne.URI) bool {
	name, ok := EntryName(src)
	if !ok {
		return false
	}

	access, err := acquirePreview(context.Background(), owner)
	if err != nil {
		return false
	}
	defer func() { _ = access.Close() }()
	for _, ext := range [...]string{".jpg", ".png"} {
		if _, err := access.Root.Stat(filepath.Join(SubDir, name+ext)); err == nil {
			return access.Current(context.Background()) == nil
		}
	}
	return false
}

// decodeFile opens and decodes path, reporting false for any failure: the
// file does not exist (the common case, since Read tries both extensions
// for every call), or it exists but is not a decodable image (a previous
// write that never completed, or on-disk corruption) - either way that is
// a plain miss for Read, not an error.
func decodeFile(ctx context.Context, access *favstore.Access, path string) (image.Image, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	f, err := access.Root.Open(path)
	if err != nil {
		return nil, false
	}
	// Failed reads close earlier under the commit lock; this also covers a
	// decoder panic and the successful-read path.
	defer func() { _ = f.Close() }()
	img, err := decodePreview(ctx, f)
	if err != nil {
		// A cache-only miss cannot decode an original to overwrite this
		// file. Drop the failed entry so a later current memory thumbnail
		// is not blocked by hasCurrentPreview's inexpensive existence test.
		// Close first for Windows, then check for a replacement installed
		// while the failed file was being decoded.
		discardCorruptPreview(ctx, access, path, f)
		return nil, false
	}
	return img, true
}

func discardCorruptPreview(ctx context.Context, access *favstore.Access, path string, failed *os.File) {
	previewCommitMu.Lock()
	defer previewCommitMu.Unlock()
	// Retain the open handle until this critical section so a previously
	// unlinked inode cannot be recycled for a healthy replacement first.
	failedInfo, statErr := failed.Stat()
	_ = failed.Close()
	if access.Current(ctx) != nil || statErr != nil {
		return
	}
	if current, err := access.Root.Stat(path); err == nil && os.SameFile(failedInfo, current) {
		_ = access.Root.Remove(path)
	}
}

func decodePreview(ctx context.Context, r io.Reader) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(contextReader{ctx: ctx, r: r})
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	return img, err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type contextWriter struct {
	ctx context.Context
	w   io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.w.Write(p)
}
