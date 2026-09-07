// Writing the frame on screen out as a new file, in a format of the user's
// choosing rather than the source file's - the File > "Export image" prompt
// (promptExport, also Cmd/Ctrl+E) and the exportAs it runs once a format is
// chosen.

package ui

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filepicker"
	"github.com/frathe/picfetch/internal/imaging"
)

// exportPNGExt/exportJPEGExt are the two formats the File menu offers to
// export to: the universally readable lossless one and the universally
// readable lossy one. internal/imaging can encode four more (GIF, BMP,
// TIFF, AVIF) and honors any of them if the user types that extension into
// the save panel themselves - see imaging.Export - but offering all six
// as menu items would be a long list for a choice that is really only ever
// these two.
const (
	exportPNGExt  = ".png"
	exportJPEGExt = ".jpg"
)

// pngChoice and jpegChoice are the export prompt's two button indices - PNG
// first/left and the default selection (lossless, so the safer of the two
// formats to land on if Return is pressed without moving the ring), JPEG
// second/right. These place the buttons rather than merely describing them:
// registerFeatures (features.go) fills the card's choice slice through them.
const (
	pngChoice  = 0
	jpegChoice = 1
)

// canExport reports whether the File > "Export image" item (and its prompt's
// PNG/JPEG choices) should be enabled. Deliberately a much weaker condition
// than canSaveRotation
// (save.go), because an export answers a different question - "write these
// pixels somewhere new", not "write them back where they came from":
//
//   - The source format doesn't matter. imaging.CanEncode gates saving
//     because SaveRotated must re-encode the file in its own format; an
//     export picks the destination's format instead, which is exactly how a
//     WebP or HEIC (decode-only in this module's dependencies) gets out.
//   - An animation doesn't matter. Save Changes refuses one because it would
//     have to re-rotate and re-encode every frame; exporting the single
//     frame on screen as a still is well-defined.
//   - A pending rotation doesn't matter. There's nothing to persist - an
//     export writes whatever is on screen, rotated or not.
//
// What does still matter is !v.loading.Load(), for the same reason it does
// in canSaveRotation: mid-load, CurrentFile() already names the file being
// navigated to while v.img.Image still holds the previous one's pixels, so
// an export started then would offer the new file's name for the old file's
// image.
func (v *viewer) canExport() bool {
	_, _, ok := v.CurrentFile()

	return !v.fileWork.closed && !v.fileWork.exportPending && ok && !v.loading.Load() && v.img.Image != nil
}

// promptExport is the File menu's "Export image" action (also Cmd/Ctrl+E,
// see wireExportShortcuts in shortcuts.go): it raises v.exportPrompt, the
// widgets.ChoiceCard built in features.go, asking which format to export to
// before exportAs actually runs. A no-op unless canExport() is currently
// true - re-checked here rather than trusted from the menu item's Disabled
// state, since the shortcut bypasses the menu entirely, mirroring
// saveRotation's own check. Escape (or clicking neither button) leaves
// without opening the save panel at all, since exportAs is only ever reached
// through a choice's OnChosen.
//
// The other two guards are about the delete confirmation, and the dangerous
// one is deletion.Visible(). Cmd/Ctrl+E is a shortcut, so it reaches this
// function without passing handleKeyEvent's dispatch at all - it can fire
// while the delete card is up. This prompt paints above that card (see the
// window stack in build.go) while handleKeyEvent still routes every key to
// deletion, which it checks first: the user would be reading "Export as
// which format?", pressing Right expecting JPEG, and moving the *hidden*
// delete card's ring onto "Move to Trash" for Return to press. requestDelete
// (batch.go) carries the mirror-image guard for Shift+Delete arriving while
// this prompt is up.
//
// exportPrompt.Visible() is the milder one, and the same guard
// deletion.RequestFiles makes for itself: a second Cmd/Ctrl+E mid-decision
// would otherwise re-Show the card and reset the ring back to PNG under
// someone who had already moved it to JPEG and was reaching for Return.
func (v *viewer) promptExport() {
	if v.comparisonActive() {
		return
	}
	if !v.canExport() || v.deletion.Visible() || v.exportPrompt.Visible() {
		return
	}

	// The rows have no way to reach the frame themselves, and the Original
	// rung's label states its longest edge - so it is set here, from the
	// image the export is about to be run over, immediately before the card
	// resets everything else.
	b := v.img.Image.Bounds()
	v.exportOptions.SetSourceEdge(max(b.Dx(), b.Dy()))

	v.exportPrompt.Show(lang.L("Export as which format?"))
}

// exportAs is the export prompt's PNG/JPEG choice action (export.go's
// promptExport, or a menu/shortcut bypassed test calling it directly) for
// the format ext: it opens the OS's own save panel and writes the frame on
// screen to whatever the user names there. A no-op unless canExport() is
// currently true.
//
// The current file and frame are captured here, on the UI goroutine, before
// the panel opens - mirroring copyImageToClipboard's own capture, and for
// the same reason: the goroutine below outlives this call by however long
// the user spends in a modal dialog, and v.img.Image belongs to the load
// path.
func (v *viewer) exportAs(ext string) {
	if v.comparisonActive() {
		return
	}
	if !v.canExport() {
		return
	}

	src, _, _ := v.CurrentFile()
	req := exportRequest{ext: ext, opts: v.exportOptions.Options(), source: src, pixels: v.img.Image, choose: filepicker.ChooseSave, write: v.fileWork.export}

	// chooser is shared with openFileDialog's own goroutine rather than
	// given a twin of its own: it means "the native file dialog
	// goroutine", and these two are never in flight at once - both panels
	// are app-modal, so neither can be reached while the other is up.
	done := v.chooser.Begin()
	token := v.fileWork.exportLifecycle.begin()
	v.fileWork.exportPending = true
	v.syncMenus()

	v.fileWork.workers.Go(func() {
		result := req.run(token.context())
		if result.write.Committed {
			v.imgCache.Purge()
		}
		if !token.current() && !result.write.Committed {
			done()
			return
		}
		v.fileWork.ui.Do(func() {
			defer token.cancelContext()
			defer v.afterFileWrite(result.write, true, true, done)
			if !token.current() {
				return
			}
			v.fileWork.exportPending = false
			v.syncMenus()
			switch {
			case result.chooserFailure:
				v.showChooserError(result.err)
			case result.err != nil:
				fyne.LogError("failed to export image", result.err)
				v.ShowToast(fmt.Sprintf(lang.L("could not export %q: %v"), result.destination.Name(), result.err))
			case result.destination != nil:
				v.ShowToast(exportedToast(result.destination.Name(), result.edge, req.opts.OmitMetadata))
			}
		})
	})
}

// exportRequest captures pixels, source, options and external operations on UI.
// Its worker never reads the changing viewer or export prompt.
type exportRequest struct {
	ext    string
	opts   imaging.ExportOptions
	source fyne.URI
	pixels image.Image
	choose func(string) (fyne.URI, error)
	write  func(context.Context, fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) (imaging.WriteResult, error)
}

type exportResult struct {
	destination    fyne.URI
	write          imaging.WriteResult
	edge           int
	err            error
	chooserFailure bool
}

// run waits for the native destination and executes the captured transaction.
// The caller owns result delivery and checks the request again on UI.
func (req exportRequest) run(ctx context.Context) exportResult {
	// The applied edge, not the requested one: a 2400 limit on an 1800px
	// photo changes nothing, so it must not name the file or appear in the
	// toast either. Everything below reports what was written rather than
	// what was asked for.
	result := exportResult{edge: appliedExportEdge(req.pixels, req.opts.MaxEdge)}
	if err := ctx.Err(); err != nil {
		result.err = err
		return result
	}

	result.destination, result.err = req.choose(suggestedExportPath(req.source, req.ext, result.edge))
	if result.err != nil {
		result.chooserFailure = true
		return result
	}

	if result.destination == nil {
		return result // cancelled
	}
	req.opts.FallbackExt = req.ext
	if err := ctx.Err(); err != nil {
		result.err = err
		return result
	}
	result.write, result.err = req.write(ctx, result.destination, req.pixels, req.source, req.opts)
	return result
}

// appliedExportEdge is the size limit that actually changed img's pixels,
// or 0 for an export that came out at the frame's own size - whether
// because no limit was chosen or because the photo was already inside the
// one that was. The question goes to imaging rather than being answered by
// comparing the limit against the longest edge here, so what the name and
// the toast report can never drift from what the encoder actually did.
func appliedExportEdge(img image.Image, maxEdge int) int {
	if imaging.SizeLimitApplies(img.Bounds(), maxEdge) {
		return maxEdge
	}

	return 0
}

// exportedToast is the message a finished export reports, and it reports
// the outcome rather than the request: an export at the defaults keeps the
// short confirmation it has always shown (the file's own name already
// carries the format actually written, so an extension the user typed over
// the format they picked shows up there without a word about it), and only
// a copy that differs from those defaults spells out how.
func exportedToast(name string, edge int, omitted bool) string {
	var details []string
	if edge > 0 {
		details = append(details, fmt.Sprintf(lang.L("%d px"), edge))
	}
	if omitted {
		details = append(details, lang.L("no camera metadata"))
	}
	if len(details) == 0 {
		return fmt.Sprintf(lang.L("Exported %q"), name)
	}

	return fmt.Sprintf(lang.L("Exported %q (%s)"), name, strings.Join(details, ", "))
}

// suggestedExportPath is what the save panel opens pre-filled with: the
// source file's own name carrying the export format's extension, in the
// source file's own folder. A full path rather than a bare name, since that
// is what every panel in internal/filepicker needs to open somewhere more
// useful than the working directory.
//
// edge is the size limit that actually applied (0 for a copy at the frame's
// own size), and when there is one the name carries it - a resized copy
// exported into the source's own folder would otherwise open pre-filled
// with a name that collides with the very file it came from. At Original
// size the suggestion is exactly what it has always been.
func suggestedExportPath(src fyne.URI, ext string, edge int) string {
	name := src.Name()
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		// A name that is nothing but an extension (".jpg"). Rare enough to
		// be worth one line rather than a suggestion of "" that the panel
		// would show as an empty file-name field.
		base = "image"
	}
	if edge > 0 {
		base = fmt.Sprintf("%s-%d", base, edge)
	}

	return filepath.Join(filepath.Dir(src.Path()), base+ext)
}
