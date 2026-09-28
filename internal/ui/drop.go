// Drop handling: the recursive folder scan itself lives in internal/filescan.

package ui

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/filescan"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
)

// cancelScan aborts a scan in progress (Escape while v.scanOp.active is true).
// It invalidates the scan's own lifecycle, so the background goroutine in
// handleDrop stops touching the filesystem without interrupting navigation,
// preloading, or animation for an already-loaded merge-mode file set.
//
// Unlike reset, it never changes the collection: a merge-mode scan can be
// cancelled mid-way through without losing images that were
// already loaded before it started. Only a scan that had nothing loaded yet
// (the first-ever drop) needs the drop zone put back the way handleDrop
// found it.
func (v *viewer) cancelScan() {
	if !v.scanOp.cancel() {
		return
	}
	v.pendingPictureFrame = false
	v.explorerInput.pendingLaunch = false

	if v.state.Observe().Count() == 0 {
		v.showWelcomeState()
		v.dropzone.Show()
	}

	v.syncMenus()
	v.ForceRepaint()
	v.ShowToast(lang.L("cancelled scanning"))
}

// MaxScan is the current recursive-folder-scan cap - the settings window's
// getter for SetMaxScan below.
func (v *viewer) MaxScan() int {
	return v.settings.maxScan
}

// SetMaxScan sets the recursive-folder-scan cap directly - the settings
// window's binding. Floored at 1 rather than 0, since a 0 cap would stop a
// scan before it gathered anything at all - not a "no limit" filescan.Images
// is written to understand (it floors again itself, defence in depth rather
// than a replacement for this floor). Applies to the next scan; one already
// in flight keeps running under whatever cap it started with - handleDrop
// snapshots v.settings.maxScan once, before starting the scan, so this
// setter can't retroactively change a scan that's already running.
func (v *viewer) SetMaxScan(n int) {
	if n < 1 {
		n = 1
	}

	v.settings.maxScan = n
}

// handleDrop starts an asynchronous scan for images, recursing into dropped
// folders and updating a spinner + counter while gathering. A replace-mode
// drop of one supported image file expands to that file's parent-directory
// siblings (filescan.Siblings) and keeps the opened file on screen after
// sort; otherwise the first scanned image is shown once the scan finishes.
// A plain drop replaces the current set, same as always; with mergeMode on
// (toggled by M) the newly scanned images are merged into it instead,
// keeping the sort order applied and jumping to the first image just added.
func (v *viewer) handleDrop(uris []fyne.URI) {
	v.openCollection(uris, "", discoverCollection)
}

type collectionInputKind uint8

const (
	discoverCollection collectionInputKind = iota
	replayCollection
)

func (v *viewer) openCollection(uris []fyne.URI, favoriteDir string, kind collectionInputKind) {
	if len(uris) == 0 {
		return
	}
	if _, ok := v.admitCommand(commandRequest{command: commandOpen}); !ok {
		return
	}

	// Admission precedes collection replacement and all cancellation effects.
	v.favorites.CancelOpen()
	// A replacement request must not inherit --slideshow from the launch
	// scan or its still-pending reorder.
	if v.scanOp.active || v.sortOp.active {
		v.pendingPictureFrame = false
		v.explorerInput.pendingLaunch = false
	}

	v.closeExplorer()
	v.closeLocationMap()
	v.openChooserLifecycle.invalidate()
	v.deletion.Cancel()
	v.grid.Close()

	// Snapshotted now, not read back inside the completion closure below:
	// a folder scan can take seconds, and toggling M while one is still
	// running shouldn't retroactively change how this already-in-flight
	// drop gets applied.
	merging := v.state.MergeMode() && v.state.Observe().HasMembers()

	v.invalidateSort()
	v.invalidateLoad()
	token, scanDone := v.scanOp.begin()
	v.syncMenus()

	v.scanOp.label.SetText(lang.L("Scanning... 0 images"))
	v.scanOp.show()
	v.dropzone.Hide()
	v.welcomeArt.Hide()
	v.restoreLink.Hide()
	v.emptyStateArt.Hide()
	v.ForceRepaint()

	// Snapshotted once, here, rather than read live from v.settings.maxScan
	// by whichever path runs below: the settings window can write that field
	// while a scan is in flight, and both handleDrop's goroutine and its
	// synchronous fast path need a stable cap for the lifetime of this one
	// scan. See SetMaxScan's doc comment.
	maxScan := v.settings.maxScan

	hasDirs := false
	for _, u := range uris {
		if canList, err := storage.CanList(u); err == nil && canList {
			hasDirs = true
			break
		}
	}

	expandSiblings := kind == discoverCollection && !merging && !hasDirs && len(uris) == 1 && (imaging.IsSupportedImage(uris[0]) || heic.IsExtension(uris[0].Extension()))
	gather := filescan.ImagesWithAdmission
	if kind == replayCollection {
		gather = filescan.ReplayWithAdmission
	}
	capability := v.heic.capability
	snapshot := capability.Snapshot()
	state := capability.State()
	var check <-chan struct{}
	if !state.Known {
		v.startHEICCheck(false)
		check = capability.Ensure(token.context())
	}
	waitForCapability := func() bool {
		if check == nil {
			return true
		}
		select {
		case <-check:
			snapshot = capability.Snapshot()
			check = nil
			return true
		case <-token.context().Done():
			return false
		}
	}
	var skipped, sourceOrder []fyne.URI
	retentionTruncated := false
	seenOrder := make(map[string]bool)
	record := func(uri fyne.URI) {
		if kind == replayCollection || !seenOrder[uri.String()] {
			sourceOrder = append(sourceOrder, uri)
			seenOrder[uri.String()] = true
		}
	}
	accepts := func(uri fyne.URI) bool {
		if !heic.IsExtension(uri.Extension()) {
			if !imaging.IsSupportedImage(uri) {
				return false
			}
			record(uri)
			return true
		}
		if !snapshot.Available {
			if kind == discoverCollection && seenOrder[uri.String()] {
				return false
			}
			if len(skipped) >= maxScan {
				retentionTruncated = true
				return false
			}
			record(uri)
			skipped = append(skipped, uri)
			return false
		}
		record(uri)
		return true
	}

	scan := func(progress func(int)) (images []fyne.URI, truncated bool) {
		if expandSiblings && heic.IsExtension(uris[0].Extension()) && (!waitForCapability() || !accepts(uris[0])) {
			// An unavailable explicit source belongs at its own guide/error
			// state; opening it must not display an unrelated neighbor.
			return nil, false
		}
		if expandSiblings {
			images, truncated = filescan.SiblingsWithAdmission(token.context(), uris[0], maxScan, progress, accepts)
		} else {
			images, truncated = gather(token.context(), uris, maxScan, progress, accepts)
		}
		if expandSiblings && len(sourceOrder) > 1 {
			// Sibling discovery preserves the opened source first and sorts
			// its bounded directory result by name, including retained gaps.
			slices.SortStableFunc(sourceOrder[1:], func(a, b fyne.URI) int {
				return strings.Compare(a.Name(), b.Name())
			})
		}
		// Admission is consulted before discovery's real-path deduplication.
		// Retain only entries the scanner actually admitted, plus unavailable
		// entries under their separate budget; aliases must not become members.
		sourceOrder = admittedSourceOrder(sourceOrder, images, skipped)
		if check != nil && len(skipped) > 0 {
			// Pending HEICs use the separate retention budget while other
			// formats keep traversal and progress moving. Resolve admission
			// only after traversal, preserving the input kind's occurrence
			// semantics and image cap if support becomes available.
			if !waitForCapability() {
				return nil, false
			}
			if snapshot.Available {
				var admissionTruncated bool
				images, admissionTruncated = gather(token.context(), sourceOrder, maxScan, nil, func(uri fyne.URI) bool {
					return heic.IsExtension(uri.Extension()) || imaging.IsSupportedImage(uri)
				})
				truncated = truncated || admissionTruncated
				skipped = nil
				sourceOrder = images
			}
		}
		return images, truncated || retentionTruncated
	}

	explicitHEIC := false
	for _, uri := range uris {
		explicitHEIC = explicitHEIC || heic.IsExtension(uri.Extension())
	}
	if !hasDirs && !expandSiblings && (check == nil || !explicitHEIC) {
		// nil progress: this path is synchronous and instantaneous, so
		// there's nothing to show, and it avoids calling fyne.Do from the
		// UI goroutine. Multi-file loose drops and merge-mode single
		// files stay here; sibling expansion of a possibly large
		// directory does not — that listing belongs on the goroutine
		// below, same as a folder drop.
		images, truncated := scan(nil)
		fyne.Do(func() {
			v.applyScanResult(token, merging, uris, images, truncated, maxScan, scanDone, favoriteDir, skipped, sourceOrder)
		})
		return
	}

	go func() {
		// token.context() is what lets a superseded scan (a newer drop, or
		// an explicit cancel - see cancelScan) stop walking the tree instead
		// of racing storage.List calls to completion for a result nobody
		// will see; the trailing fyne.Do below re-checks the token and would
		// discard the result anyway. The same context cancels a sibling
		// listing (Siblings) as a recursive Images walk.
		images, truncated := scan(func(n int) {
			fyne.Do(func() {
				if !token.current() {
					return
				}
				v.scanOp.label.SetText(fmt.Sprintf(lang.L("Scanning... %d images"), n))
			})
		})

		fyne.Do(func() {
			v.applyScanResult(token, merging, uris, images, truncated, maxScan, scanDone, favoriteDir, skipped, sourceOrder)
		})
	}()
}

func admittedSourceOrder(order, images, unavailable []fyne.URI) []fyne.URI {
	remaining := make(map[string]int, len(images)+len(unavailable))
	for _, group := range [][]fyne.URI{images, unavailable} {
		for _, uri := range group {
			remaining[uri.String()]++
		}
	}
	result := make([]fyne.URI, 0, len(order))
	for _, uri := range order {
		key := uri.String()
		if remaining[key] > 0 {
			remaining[key]--
			result = append(result, uri)
		}
	}
	return result
}

// applyScanResult is the shared completion step for both of handleDrop's
// paths - the synchronous no-directories fast path and the background
// goroutine (recursive folder walk or single-file sibling listing). It must
// run on the UI goroutine (both callers wrap it in fyne.Do) and always
// finishes scanDone, honoring that generation's contract even when a newer
// generation has made this result stale. maxScan is the cap the scan
// actually ran under (handleDrop's snapshot), so the truncation toast below
// reports it accurately even if the settings window has since changed
// v.settings.maxScan.
func (v *viewer) applyScanResult(token requestToken, merging bool, uris, images []fyne.URI, truncated bool, maxScan int, scanDone func(), favoriteDir string, skipped, sourceOrder []fyne.URI) {
	defer scanDone()
	defer token.cancelContext()

	if !token.current() {
		return
	}
	v.scanOp.finish()
	v.syncMenus()

	if len(images) == 0 {
		v.explorerInput.pendingLaunch = false
		msg := fmt.Sprintf(lang.L("none of the %d dropped files is a supported image"), len(uris))
		if len(uris) == 1 {
			msg = fmt.Sprintf(lang.L("%q is not a supported image file"), uris[0].Name())
		}
		input := collectionInput{retained: retainedSources(skipped, sourceOrder), favorite: favoriteDir}
		if !merging || len(input.retained) > 0 {
			v.commitOpenedCollection(input, merging, func() {
				if v.FileCount() > 0 {
					v.ShowToast(msg)
					return
				}
				v.pendingPictureFrame = false
				v.slides.Exit()
				v.resetFade()
				v.presentDropzone()
				v.showEmptyCollectionError(msg)
			})
		} else {
			// No admitted additions: keep all committed facts and bindings.
			v.ShowToast(msg)
			if v.FileCount() == 0 {
				v.dropzone.Show()
				v.emptyStateArt.Show()
			}
		}

		v.explainUnavailableHEIC(skipped, true)
		if truncated {
			v.ShowToast(fmt.Sprintf(lang.L("scan limit of %d reached - some files may not have been included"), maxScan))
		}

		// A --slideshow launch whose paths held no image is spent here
		// rather than left armed for whatever the user drops next.
		v.startPendingPictureFrame()

		return
	}

	v.ForceRepaint()

	// Both a Cmd/Ctrl+O pick and an OS-level drag-and-drop can land while
	// the window itself isn't focused (the file dialog owned focus; a drop
	// from Finder/Explorer never gives it in the first place). Without this
	// the freshly loaded image sits there unresponsive to keyboard input
	// until the user clicks the window once just to focus it.
	v.win.RequestFocus()

	if truncated {
		v.ShowToast(fmt.Sprintf(lang.L("scan limit of %d reached - some files may not have been included"), maxScan))
		if v.explorer.Trial() != nil {
			v.explorerInput.pendingLaunch = false
			v.pendingPictureFrame = false
			v.explorer.Trial().Reject("scan-truncated", len(images))
			if v.FileCount() == 0 {
				v.showWelcomeState()
				v.dropzone.Show()
				v.ForceRepaint()
			}
			return
		}
	}

	// Deliberately last: applyScannedFiles hands the reorder to a background
	// goroutine that goes on to call ShowImage, which itself kicks off an
	// async decode chain - and under the fyne test driver both that
	// goroutine and the decode's completion work (finishLoad/resizeToImage)
	// run inline rather than being marshaled onto one UI goroutine, so
	// nothing here may touch the UI once it starts. The truncation toast
	// above raced exactly that way before this ordering was fixed. Under the
	// real driver the fyne.Do queue serializes both orders identically.
	explicit := len(uris) == 1 && heic.IsExtension(uris[0].Extension())
	v.explainUnavailableHEIC(skipped, explicit)
	v.applyScannedCollection(merging, images, uris, favoriteDir, skipped, sourceOrder)
}

// applyScannedFiles prepares the display order on a worker before committing
// replacement or merge. Capture-date/modified/size sorting can read every source;
// preparation leaves the previous complete collection authoritative throughout.
//
// On a non-merge drop of one file, the URI the user opened is shown after
// the reorder rather than index 0, so sibling expansion does not jump to
// the first name-sorted neighbour. A folder drop's dropped[0] is a
// directory, which is never in the image list, so that lookup fails and
// we still land on index 0.
//
// The current sort callback commits both projections, retained membership,
// association and selection together. Stale preparation cannot install one
// order ahead of the other or resurrect a source removed while sorting.
func (v *viewer) applyScannedFiles(merging bool, images, dropped []fyne.URI, favoriteDir string) {
	v.applyScannedCollection(merging, images, dropped, favoriteDir, nil, images)
}

func (v *viewer) applyScannedCollection(merging bool, images, dropped []fyne.URI, favoriteDir string, skipped, sourceOrder []fyne.URI) {
	v.closeVisualSearch()
	var unsorted []fyne.URI
	if merging {
		// SourceFiles returns an owned copy for background sort preparation.
		unsorted = append(v.state.Observe().SourceFiles(), images...)
	} else {
		unsorted = images
	}

	v.startSort(v.state.SortMode(), unsorted, func(ordered []fyne.URI) {
		input := collectionInput{source: images, display: ordered, retained: retainedSources(skipped, sourceOrder), favorite: favoriteDir}
		var target fyne.URI
		if merging {
			target = images[0]
		} else if len(dropped) == 1 {
			target = dropped[0]
		}
		if target != nil {
			for i, uri := range ordered {
				if uri.String() == target.String() {
					input.index = i
					break
				}
			}
		}
		v.commitOpenedCollection(input, merging, func() {

			// Here rather than anywhere earlier because this is the first point
			// at which the files a --slideshow launch asked to frame exist:
			// picture-frame mode no-ops at zero files. Before the ShowImage
			// calls below, so entering full-screen and showing the first image
			// are one repaint rather than two.
			if v.explorerInput.pendingLaunch {
				v.explorerInput.pendingLaunch = false
				v.pendingPictureFrame = false
				v.showExplorer()
				return
			}
			v.startPendingPictureFrame()

			v.loadImage(input.index)
		})
	})
}
