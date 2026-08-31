# 06: Enforce the transient-mode lifecycle

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Make Copy Selection mode own keyboard/pointer interaction
while active, cancel it before every non-viewport PicFetch command, preserve it
across application focus loss, and ignore cancellation or other commands while
copying is busy.

**Blocked by:** 05

**Status:** ready-for-agent

## Lifecycle contract

- Add one viewer-owned guard/helper that means "cancel region copy before this
  action, unless its copy worker is busy." Do not scatter slightly different
  busy rules across features.
- Modal overlays and existing feature-specific key owners retain priority over
  Copy Selection. When no higher modal owns input, Copy Selection handles
  `Escape`, `Return`, `Enter`, pointer gestures, and navigation suppression
  before normal viewer dispatch.
- Zoom/pan entry points are the only ordinary commands that remain active
  without cancelling.
- Window close/shutdown remains available while busy and must cancel or drain
  background work safely.

## Behavior checklist

- [ ] Start with a failing table-driven real-viewer test at the public action
      seams, not a test of helper call counts.
- [ ] `Escape` cancels without changing the clipboard; `Return` and `Enter`
      copy only with a valid rectangle.
- [ ] Suppress image navigation keys and typed runes while the mode is active.
- [ ] Cancel before navigation, open/drop, close files, rotate, Grid View,
      Picture Frame, secondary windows, and every other Actions command; then
      perform that command normally.
- [ ] Preserve the mode and rectangle through wheel/key zoom, `Shift+scroll`
      pan, window resize, and application focus loss.
- [ ] Repeated Copy selection activation remains a no-op.
- [ ] While busy, ignore editing, repeated copy, `Escape`, and application
      commands; allow normal close/shutdown.
- [ ] Restore the information overlay and animation exactly once on every
      successful or cancelled exit, with no stale completion affecting a later
      image or activation.
- [ ] Cover Grid selection terminology and ordinary `Cmd`/`Ctrl+C` behavior so
      the new image-region mode cannot regress batch copying.

## Files

- Modify focused action/dispatch files under `internal/ui/`, including
  `keys.go`, drop/open/close/navigation, rotation, Window actions, and Actions
  handlers only where the lifecycle guard is required
- Modify or add focused table-driven tests under `internal/ui/`
- Do not modify: `internal/ui/copyselection` interface, `internal/clipboard`,
  translations, manuals, or `ARCHITECTURE.md`

## Verification

```sh
go test -race ./internal/ui -run 'TestCopySelection(Keyboard|RepeatedActivation|FocusLoss|CancelsBeforeOtherCommands|BusyBlocksOtherCommands)$'
go test -race ./internal/ui -run 'TestCopy_(WhileGridVisibleCopiesTheSelectionAsFileReferences|OutsideTheGridStillCopiesTheImage)$'
```

Negatively verify the action-cancellation table, restore it, run the complete
`internal/ui` package once without race, set `Status: resolved`, and record the
result under `## Answer`. Do not commit.

## Answer

Pending.
