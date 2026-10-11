// The grid's batch actions: what Shift+Delete and Cmd/Ctrl+C mean while the
// thumbnail overview is up.
//
// This file is the only thing in the module that knows both sides exist.
// internal/ui/grid owns a selection and will tell anyone who asks what is in
// it; internal/ui/deletion moves a set of files to the Trash; neither imports
// the other, and neither knows the grid's Targets are what the confirmation's
// Targets get built from. That is the same cross-feature composition rule the
// grid/slideshow guard follows - features expose state and actions, and this
// package decides how they compose.

package ui

import (
	"context"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/clipboard"
	"github.com/frathe/picfetch/internal/fileaccess"
	"github.com/frathe/picfetch/internal/ui/deletion"
)

// requestDelete is what Shift+Delete runs (see wireDeleteShortcut). It routes
// to the grid's selection while the overview is up and to the file on screen
// otherwise, so one shortcut means the same thing - "get rid of what I'm
// looking at" - in both places.
//
// It does nothing at all while the export-format prompt is up. Shift+Delete
// is a shortcut, so it arrives without passing handleKeyEvent's dispatch and
// could otherwise raise the delete card *underneath* a prompt that is still
// what the user is looking at - while handleKeyEvent, which checks deletion
// first, would hand their next Right/Return to the card they can't see. See
// promptExport (export.go) for the same guard in the other direction.
func (v *viewer) requestDelete() {
	if _, ok := v.admitCommand(commandRequest{command: commandTrash}); !ok {
		return
	}

	if v.grid.Visible() {
		v.deleteGridSelection()
		return
	}

	v.deletion.Request()
}

// deleteGridSelection opens the confirmation card for whatever the grid has
// picked - or, with nothing explicitly picked, the highlighted cell alone
// (grid.Targets). The card is raised over the grid rather than closing it:
// the whole point of a batch is working through a large set, and closing the
// overview after every one would throw away the user's place in it.
func (v *viewer) deleteGridSelection() {
	if _, ok := v.admitCommand(commandRequest{command: commandTrash}); !ok {
		return
	}
	targets := v.grid.Targets()
	if len(targets) == 0 {
		return
	}

	ts := make([]deletion.Target, 0, len(targets))
	collection := v.state.Observe()
	for _, i := range targets {
		if i < 0 || i >= collection.Count() {
			continue
		}
		ts = append(ts, deletion.Target{URI: collection.FileAt(i)})
	}

	v.deletion.RequestFiles(ts)
}

// copySelection is what Cmd/Ctrl+C runs (see wireClipboardShortcuts): the
// active image-region selection as cropped image data, the grid selection as
// file references while the overview is up, and the displayed frame as image
// data otherwise. Different things share one shortcut because they are the
// same intent applied to the subject the user is currently working with.
func (v *viewer) copySelection() {
	decision, ok := v.admitCommand(commandRequest{command: commandCopy})
	if !ok {
		return
	}
	if decision.target == targetRegion {
		v.regionCopy.HandleKey(fyne.KeyReturn)
		return
	}
	if decision.target == targetGridFiles {
		v.copyGridSelection()
		return
	}

	v.copyImageToClipboard()
}

// copyGridSelection puts the selected files on the clipboard as file
// references, so a paste in Finder/Explorer/a Linux file manager creates
// copies of the files themselves.
//
// Runs on its own goroutine and reports through v.clipboard for the same
// reasons copyImageToClipboard does: every backing command blocks on external
// I/O, and a test needs one thing to wait on rather than polling widgets the
// goroutine may still be writing.
func (v *viewer) copyGridSelection() {
	if _, ok := v.admitCommand(commandRequest{command: commandCopyFiles}); !ok {
		return
	}
	targets := v.grid.Targets()

	sources := make([]fyne.URI, 0, len(targets))
	collection := v.state.Observe()
	for _, i := range targets {
		if i >= 0 && i < collection.Count() {
			sources = append(sources, collection.FileAt(i))
		}
	}
	if len(sources) == 0 {
		return
	}

	token, done, ok := v.beginClipboardCopy(false)
	if !ok {
		return
	}
	v.clipboardWork.workers.Go(func() {
		if !token.Current() {
			v.completeClipboardCopy(token, done, nil)
			return
		}
		err := copyFilesWithAccess(token.Context(), sources)
		v.completeClipboardCopy(token, done, func() {
			if err != nil {
				v.reportFileCopyError(err)
				return
			}
			if len(sources) == 1 {
				v.ShowToast(lang.L("copied 1 file"))
				return
			}

			v.ShowToast(fmt.Sprintf(lang.L("copied %d files"), len(sources)))
		})
	})
}

// reportFileCopyError logs a failed file-reference copy and shows a toast.
func (v *viewer) reportFileCopyError(err error) {
	detail := chooserErrorDetail(err)
	fyne.LogError("clipboard file copy failed", errors.New(detail))

	v.ShowToast(fmt.Sprintf(lang.L("could not copy the files: %v"), detail))
}

// selectAllInGrid is Cmd/Ctrl+A, and does nothing outside the grid: there is
// nothing to select in the normal image view, and quietly building a
// selection there would make it appear out of nowhere the next time the
// overview opened.
func (v *viewer) selectAllInGrid() {
	if _, ok := v.admitCommand(commandRequest{command: commandSelectAll}); !ok {
		return
	}

	v.grid.SelectAll()
}

func copyFilesWithAccess(ctx context.Context, sources []fyne.URI) error {
	paths := make([]string, 0, len(sources))
	var releases []func()
	defer func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}()
	for _, uri := range sources {
		resolved, release, err := fileaccess.Acquire(ctx, uri)
		if err != nil {
			return err
		}
		releases = append(releases, release)
		paths = append(paths, resolved.Path())
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return clipboard.CopyFiles(paths)
}
