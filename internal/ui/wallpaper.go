// Making the image on screen the desktop wallpaper - the Actions
// "Set as Wallpaper" action.

package ui

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/wallpaper"
)

// wallpaperPrefix names the PNGs this writes into wallpaperDir, and is what
// sweepWallpapers matches on to clear the one it replaces. Everything else
// in that directory is left alone.
const wallpaperPrefix = "wallpaper-"

var errWallpaperBusy = wallpaper.ErrBusy

// canSetWallpaper reports whether the Actions "Set as Wallpaper"
// item should be enabled. Deliberately the same condition as canExport
// (export.go), and for the same reasons: this writes a PNG of the frame on
// screen, so neither the source format nor an animation nor a pending
// rotation stands in the way, while !v.loading.Load() still does - mid-load
// v.img.Image holds the outgoing file's pixels, and those are what would
// end up on the desktop.
func (v *viewer) canSetWallpaper() bool {
	return v.canExport()
}

// setAsWallpaper is the Actions "Set as Wallpaper" action: it writes the
// frame on screen into the app's own cache directory and points the OS at
// that copy. A no-op unless canSetWallpaper() is currently true.
//
// A copy rather than the file itself, because every platform in
// internal/wallpaper stores a *reference* to the path it is given: pointing
// the desktop at the user's own file would leave the wallpaper broken the
// moment they moved it, or trashed it with Shift+Delete one keystroke later.
// The copy also carries whatever is actually on screen - the current
// rotation, one frame of an animation - and is a PNG whatever the source
// was, so a WebP or HEIC this module can only decode still works.
//
// The file and frame are captured here, on the UI goroutine, before the
// goroutine below starts - mirroring exportAs, and for the same reason:
// v.img.Image belongs to the load path.
func (v *viewer) setAsWallpaper() {
	if v.comparisonActive() {
		return
	}
	if !v.canSetWallpaper() {
		return
	}
	if !v.wallpaperBusy.CompareAndSwap(false, true) {
		v.reportWallpaperError(errWallpaperBusy)
		return
	}

	src, _, _ := v.CurrentFile()
	img := v.img.Image

	// wallpaper is finished once this change has fully landed, toast
	// included, so a test can read widget state without racing the
	// goroutine that writes it.
	done := v.wallpaper.Begin()

	go func() {
		defer done()
		defer v.wallpaperBusy.Store(false)

		if err := v.applyWallpaper(context.Background(), img, src.Name(), ""); err != nil {
			v.reportWallpaperError(err)
			return
		}
		fyne.Do(func() {
			v.ShowToast(fmt.Sprintf(lang.L("set %q as the wallpaper"), src.Name()))
		})
	}()
}

// applyWallpaper is the single write-then-set lifecycle used by both viewer
// pixels and immutable mosaic-result pixels. The caller owns wallpaperBusy.
func (v *viewer) applyWallpaper(ctx context.Context, img image.Image, label string, target displays.ID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dest, err := v.writeWallpaperFile(img, target)
	if err != nil {
		return fmt.Errorf("write wallpaper copy for %q: %w", label, err)
	}

	if err := ctx.Err(); err != nil {
		_ = os.Remove(dest)
		return err
	}
	if err := wallpaper.Set(wallpaper.Request{Path: dest, Target: target}); err != nil {
		// The file just written is of no use to anyone now, and the previous
		// one may well still be the active wallpaper - so this removes only
		// what this call added, and sweeps nothing.
		_ = os.Remove(dest)
		return fmt.Errorf("set wallpaper for %q: %w", label, err)
	}

	sweepWallpapers(v.wallpaperDir, dest, target)

	return nil
}

// writeWallpaperFile encodes img as a PNG in the viewer's own cache
// directory and returns the path. The name carries a timestamp rather than
// being a fixed "wallpaper.png" because macOS caches the desktop picture by
// path: re-pointing it at the same path with different content can leave
// the old picture on screen. sweepWallpapers is what keeps that from
// accumulating a file per invocation.
func (v *viewer) writeWallpaperFile(img image.Image, target displays.ID) (string, error) {
	if err := os.MkdirAll(v.wallpaperDir, 0o755); err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s%s-%d.png", wallpaperPrefix, wallpaperScope(target), time.Now().UnixNano())
	dest := filepath.Join(v.wallpaperDir, name)
	if err := imaging.Export(storage.NewFileURI(dest), img, nil); err != nil {
		return "", err
	}

	return dest, nil
}

// sweepWallpapers removes every wallpaper this app wrote into dir except
// keep, which is the one the OS was just pointed at. Best-effort throughout:
// a file that can't be removed costs a few hundred KB of cache and nothing
// else, and there is nothing useful to tell the user about it - the wallpaper
// they asked for is already on screen by the time this runs.
func sweepWallpapers(dir, keep string, target displays.ID) {
	pattern := wallpaperPrefix + "*.png"
	if target != "" {
		pattern = wallpaperPrefix + wallpaperScope(target) + "-*.png"
	}
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	for _, m := range matches {
		if m == keep {
			continue
		}
		_ = os.Remove(m)
	}
}

// wallpaperScope turns an opaque platform ID into a stable filesystem-safe
// cache partition without persisting the raw identifier.
func wallpaperScope(target displays.ID) string {
	if target == "" {
		return "global"
	}
	digest := sha256.Sum256([]byte(target))
	return fmt.Sprintf("target-%x", digest[:8])
}

// SetMosaicWallpaper applies the immutable latest result to exactly the
// selected display. It deliberately never reads the main viewer image.
func (v *viewer) SetMosaicWallpaper(ctx context.Context, result mosaic.Result, target displays.ID) error {
	if target == "" {
		return errors.New("mosaic wallpaper requires a target display")
	}
	pixels := result.Image()
	if pixels == nil {
		return errors.New("mosaic wallpaper has no generated pixels")
	}
	if !v.wallpaperBusy.CompareAndSwap(false, true) {
		return errWallpaperBusy
	}
	defer v.wallpaperBusy.Store(false)

	return v.applyWallpaper(ctx, pixels, lang.L("Image Mosaic"), target)
}

// reportWallpaperError toasts a failed wallpaper change from the background
// goroutine. One message for both halves of the operation - writing the copy
// and handing it to the OS - since the error itself says which, and neither
// is anything the user can act on differently.
func (v *viewer) reportWallpaperError(err error) {
	fyne.LogError("failed to set the wallpaper", err)

	fyne.Do(func() {
		if errors.Is(err, wallpaper.ErrBusy) {
			v.ShowToast(lang.L("Another wallpaper change is already in progress."))
			return
		}
		v.ShowToast(fmt.Sprintf(lang.L("could not set the wallpaper: %v"), err))
	})
}

// defaultWallpaperDir is where the copies handed to the OS live: a
// subdirectory of the user's own cache directory, keyed by the same appID
// main.go gives Fyne. The cache directory is the honest home for them -
// they are regenerable, and losing them costs the user nothing but the
// current wallpaper's backing file, which the next Set rewrites. Falls back
// to the temp directory on the platforms os.UserCacheDir has no answer for.
func defaultWallpaperDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}

	return filepath.Join(base, "picfetch", "wallpapers")
}
