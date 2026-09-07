// Package deletion is the Shift+Delete confirmation flow: a dimmed scrim
// behind a centered card asking whether to move the file on screen to the
// Trash, and the trash.Move that follows if the user says yes.
//
// It owns which files a pending confirmation is about, and reaches back
// into the app only through Host - the first of the per-feature interfaces
// this app's package split is built on. Host is declared here, by the
// consumer, and lists exactly what this feature needs and nothing else; the
// viewer satisfies it incidentally, without knowing this package exists.
// The card itself - scrim, message, buttons, focus rings, the selection
// behind them, and the Left/Right/Return/Escape handling - is
// internal/ui/widgets.ChoiceCard, shared with the export-format prompt.
package deletion

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/trash"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

// Host is what the confirmation flow needs from the application: which file
// is on screen, and the handful of display actions that follow a delete.
type Host interface {
	// CurrentFile returns the displayed file and its index, ok=false when
	// nothing is loaded.
	CurrentFile() (u fyne.URI, index int, ok bool)

	// ReconcileDeletedFiles removes successful URI identities from the current
	// file set, even if it was reordered or replaced during the OS move.
	// It returns whether the current set contained any of those identities.
	ReconcileDeletedFiles(uris []fyne.URI) bool

	// ShowImage displays the file at index i, wrapping at both ends.
	ShowImage(i int)

	// ShowEmptyStateError clears to the empty drop zone with msg - used
	// when the deleted file was the last one.
	ShowEmptyStateError(msg string)

	// ShowToast raises a short, non-blocking notification.
	ShowToast(msg string)

	// ForceRepaint redraws the window after a visibility change.
	ForceRepaint()
}

// Target names one file a pending confirmation would move to the Trash.
// Its URI identity remains valid when the viewer reorders the file list.
type Target struct {
	URI fyne.URI
}

// cancelChoice and dangerChoice are the card's two button indices - Cancel
// first/left (selected by default), the red "Move to Trash" button
// second/right, so the Right arrow key - which moves selection toward the
// higher index - points toward where it actually sits.
const (
	cancelChoice = 0
	dangerChoice = 1
)

// Confirmer is the confirmation card and the state behind it.
type Confirmer struct {
	host   Host
	ui     UIQueue
	closed atomic.Bool
	ctx    context.Context
	cancel context.CancelFunc

	// targets is what confirming would move to the Trash: one file for the
	// Shift+Delete on the image being viewed, or the grid's whole selection
	// for a batch. Captured by Request/RequestFiles rather than read back
	// when the danger button is pressed, so a card that is already up asks
	// about exactly the files it named.
	targets []Target

	// Every completion is queued inside its owning worker, so waiting for
	// pending also guarantees that a test queue can drain all its effects.
	pending sync.WaitGroup

	// card is the prompt itself: scrim, message, the two buttons and their
	// focus rings, and the Left/Right/Return/Escape handling over them. It
	// owns the selection outright - this type keeps no copy of it, so a
	// click on a button (which reaches the card directly, never this
	// package) can't leave the two disagreeing.
	card *widgets.ChoiceCard
}

// New builds the confirmation card (hidden) around host.
//
// The card is a dimmed scrim behind a centered box, with Cancel first/left
// (selected by default) and the red "Move to Trash" button second/right -
// see cancelChoice/dangerChoice. The Cancel choice runs nothing of its own:
// widgets.ChoiceCard already hides itself before running either choice, and
// that is everything Cancel/Escape have ever needed to do here, so there is
// nothing left for SetOnCancel to add.
func New(host Host) *Confirmer {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Confirmer{host: host, ui: fyneQueue{}, ctx: ctx, cancel: cancel}

	c.card = widgets.NewChoiceCard(host.ForceRepaint,
		widgets.Choice{Label: lang.L("Cancel")},
		widgets.Choice{
			Label:      lang.L("Move to Trash"),
			Importance: widget.DangerImportance,
			OnChosen:   c.performDelete,
		},
	)

	return c
}

// Overlay is the card, for the app to place in its window stack.
func (c *Confirmer) Overlay() fyne.CanvasObject {
	return c.card.Overlay()
}

// Visible reports whether the card is up - the app's key dispatcher checks
// this before its own handling.
func (c *Confirmer) Visible() bool {
	return c.card.Visible()
}

// Request opens the confirmation card for the file currently on screen - the
// plain Shift+Delete on the image being viewed. A no-op with nothing loaded.
func (c *Confirmer) Request() {
	u, _, ok := c.host.CurrentFile()
	if !ok {
		return
	}

	c.RequestFiles([]Target{{URI: u}})
}

// RequestFiles opens the confirmation card for a whole set of files - the
// grid's selection, via the app's batch glue. A no-op with no targets, or if
// the card is already up: re-triggering the shortcut mid-prompt shouldn't
// reset the selection out from under a user who's already moved it onto the
// danger button and is reaching for Return.
//
// A single target is worded exactly as the single-file prompt always was,
// naming the file; anything more names the count instead, since a card
// listing forty file names would be unreadable and unbounded in height.
func (c *Confirmer) RequestFiles(targets []Target) {
	if len(targets) == 0 || c.card.Visible() || c.closed.Load() {
		return
	}

	c.targets = make([]Target, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		key := target.URI.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		c.targets = append(c.targets, target)
	}
	targets = c.targets

	var msg string
	if len(targets) == 1 {
		msg = fmt.Sprintf(lang.L("Move %q to the Trash?"), targets[0].URI.Name())
	} else {
		msg = fmt.Sprintf(lang.L("Move %d files to the Trash?"), len(targets))
	}

	// Show resets the selection to cancelChoice, so a card never opens with
	// the destructive button already under Return.
	c.card.Show(msg)
}

// Cancel dismisses the card without touching any file - Escape while it's
// up, clicking Cancel, or a fresh drop arriving mid-prompt. A no-op when
// the card isn't showing, so callers that call it defensively (the app, on
// every drop) don't need to check Visible themselves first.
func (c *Confirmer) Cancel() {
	c.targets = nil
	if !c.card.Visible() {
		return
	}

	c.card.Hide()
}

// Close stops admission, unstarted batch moves and late UI publication.
// An OS move already submitted cannot be interrupted. Settle observes its
// completion separately, so shutdown never waits on the UI event loop.
func (c *Confirmer) Close() {
	c.closed.Store(true)
	c.cancel()
	c.Cancel()
}

// HandleKey handles a key press while the card is up: Left/Right move the
// selection, Return runs whichever action is selected, Escape cancels.
// Every other key is deliberately swallowed by the caller.
func (c *Confirmer) HandleKey(ev *fyne.KeyEvent) {
	c.card.HandleKey(ev)
}

// performDelete is the danger button's action (or Return with it
// selected): it moves the current file to the OS trash/recycle bin via
// trash.Move rather than removing it outright, so Shift+Delete is
// recoverable the same way a delete from Finder/Explorer/a Linux file
// manager already is. It runs as the danger choice's OnChosen, so
// widgets.ChoiceCard has already hidden the card by the time this starts.
//
// It runs on its own goroutine, mirroring openFileDialog/copyImageToClipboard
// - and here that's not just consistency with them, it's load-bearing:
// trash.Move's darwin implementation waits on NSWorkspace's completion
// handler via a semaphore, and that handler is delivered back through
// Cocoa's own machinery, which needs this app's UI goroutine free to keep
// running for the delivery to ever happen. Call trash.Move synchronously
// from the UI goroutine - as os.Remove safely was - and that wait
// deadlocks the whole app on every confirmed delete on macOS; a background
// goroutine (confirmed against a real NSWorkspace call, not just reasoned
// about) keeps the UI goroutine free the whole time.
//
// Each confirmation owns its captured targets and results. Successful moves
// are reconciled by URI on the UI goroutine: reordering, another deletion, or
// a fresh drop can change indices while the OS works, but cannot retarget
// either the operation or its bookkeeping.
//
// The moves run one after another on that single goroutine rather than in
// parallel: trash.Move's darwin implementation already blocks on a
// completion handler per call, and a selection is tens of files, not
// thousands. Failures are collected instead of aborting the batch - one file
// the OS refuses to move shouldn't cost the user the rest of it - and only
// what actually moved is removed from the file set, so anything left behind
// on disk is also still in the app.
func (c *Confirmer) performDelete() {
	targets := c.targets
	c.targets = nil
	if len(targets) == 0 || c.closed.Load() {
		return
	}

	c.pending.Go(func() {

		moved := make([]fyne.URI, 0, len(targets))
		var firstErr error
		var firstFailed string

		for _, t := range targets {
			if c.closed.Load() {
				return
			}
			// The claim includes Save/Strip/Export so their atomic replacement
			// cannot recreate a source after its successful move to Trash.
			// Pass the original path to Trash: a symlink is itself the target.
			err := imaging.WithFileMutation(c.ctx, t.URI.Path(), func() error {
				return trash.Move(t.URI.Path())
			})
			if err != nil {
				if firstErr == nil {
					firstErr, firstFailed = err, t.URI.Name()
				}

				continue
			}
			moved = append(moved, t.URI)
		}

		c.ui.Do(func() {
			if c.closed.Load() {
				return
			}
			if len(moved) == 0 {
				c.host.ShowToast(fmt.Sprintf(lang.L("could not move %q to the Trash: %v"), firstFailed, firstErr))
				return
			}

			msg := c.movedMessage(targets, moved, firstFailed, firstErr)
			if !c.host.ReconcileDeletedFiles(moved) {
				c.host.ShowToast(msg)
				return
			}

			if _, i, stillLoaded := c.host.CurrentFile(); stillLoaded {
				c.host.ShowToast(msg)
				c.host.ShowImage(i)
			} else {
				c.host.ShowEmptyStateError(msg)
			}
		})
	})
}

// movedMessage is what the toast (or the empty-state notice) says once the
// moves are done: the single-file wording when one file was asked for, a
// count when more were, and a count of both when some of them failed - a
// batch that silently reported success for files still sitting on disk would
// be the worst of the three.
func (c *Confirmer) movedMessage(targets []Target, moved []fyne.URI, failedName string, failedErr error) string {
	switch {
	case len(moved) < len(targets):
		return fmt.Sprintf(lang.L("moved %d of %d files to the Trash; %q failed: %v"),
			len(moved), len(targets), failedName, failedErr)
	case len(targets) == 1:
		return fmt.Sprintf(lang.L("moved %q to the Trash"), targets[0].URI.Name())
	default:
		return fmt.Sprintf(lang.L("moved %d files to the Trash"), len(moved))
	}
}

// Settle waits for all submitted OS moves and drains a configured test queue.
// Production Fyne queues drain themselves; this never waits on the UI thread
// from a worker or promises that the native event loop has applied a callback.
func (c *Confirmer) Settle() {
	for {
		c.pending.Wait()
		if !c.ui.Drain() {
			return
		}
	}
}

// ShortcutHandler is registered against &fyne.ShortcutCut{} rather than a
// desktop.CustomShortcut - see the app's wireDeleteShortcut for why a
// CustomShortcut for Shift+Delete can never actually fire. shortcut is
// whatever ShortcutName() == "Cut" event the driver produced; only the
// Secondary one (Shift+Delete) is ours, so a real Ctrl/Cmd+X - which this
// app has no cut action for - is correctly ignored rather than opening a
// delete prompt.
//
// request is what a real Shift+Delete runs. A callback rather than this
// package calling Request itself, because what the key should confirm
// depends on state this package deliberately knows nothing about - the grid
// overview's selection, when that is up. Telling the two apart is the app's
// job (see internal/ui's requestDelete); recognising the key is this one's.
func ShortcutHandler(request func()) func(fyne.Shortcut) {
	return func(shortcut fyne.Shortcut) {
		if cut, ok := shortcut.(*fyne.ShortcutCut); ok && cut.Secondary {
			request()
		}
	}
}
