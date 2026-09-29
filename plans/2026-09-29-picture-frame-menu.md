# Picture-frame menu bar

## Frame

On Linux and Windows, picture-frame mode hides the in-window menu bar until the pointer has stayed along the top edge, then slides the bar in and out. An open menu holds the bar fully shown. macOS keeps the system menu bar.

Route: Standard. One feature, two packages (`internal/ui/framemenu`, `internal/ui`). No delegation (hot context).

## Decisions

| Decision | Choice |
|---|---|
| Where | Full-width top edge, not a literal corner. The bar spans the window. |
| Near | 24px band while hidden. Once shown, the whole bar counts. |
| Debounce | 500ms both ways. A shorter visit does not start a slide. |
| Slide | 200ms ease-in-out. |
| Open menu | Snap fully shown and do not slide out until the menu closes. |
| macOS | No change. Native menu stays. |
| Mechanism | Detach Fyne's in-window bar (`SetMainMenu(nil)`) and draw our own. Fyne's driver menu cannot be translated. |

## Acceptance

```
AC1  Hidden until a 500ms dwell, then eased in; a shorter visit does nothing.
     go test -tags no_emoji,nodynamic ./internal/ui/framemenu/ -count=1 -run 'TestReveal_'

AC2  After it is shown, leaving waits 500ms, then slides out.
     (same command)

AC3  An open menu stays fully shown; after it closes off the bar, the hide dwell runs.
     go test -tags no_emoji,nodynamic ./internal/ui/framemenu/ -count=1 -run 'TestChrome_OpenMenu'

AC4  Picture-frame mode with the sliding flag detaches the window menu and restores it on exit.
     With the flag off, the window menu stays.
     go test -tags no_emoji,nodynamic ./internal/ui/ -count=1 -run 'TestPictureFrameMenu_|TestInWindowMenuBar'

AC5  Manual mentions the Linux/Windows behavior, without a Unicode arrow.
     go test -tags no_emoji,nodynamic ./internal/ui/help/ -count=1 -run 'TestManualHasNoUnicodeArrows'
```

## Non-goals

- Animating the macOS system menu bar.
- Keyboard Alt to summon the bar.
- Changing picture-frame advance, shuffle, or fullscreen.

## Honest limit

The Fyne test driver does not paint the glfw menu, so "the image uses the pixels the old bar occupied" is inferred from `SetMainMenu(nil)`, which is what removes that bar in the real driver. Slide smoothness is the eased fraction plus a Fyne animation tick; pixel golden shots are not part of this change.

## Tasks

### Task 1 — Reveal
Owner: T0
Files: `internal/ui/framemenu/reveal.go`, `reveal_test.go`
Test: dwell, cancel, slide in, slide out, pin while a menu is open
Verify: AC1 AC2

### Task 2 — Chrome widget
Owner: T0
Files: `chrome.go`, `chrome_test.go`
Test: hidden bar is above the top edge; full width counts; open menu pins
Verify: AC3

### Task 3 — Viewer wiring
Owner: T0
Files: `pictureframe_menu.go`, `build.go`, `menu.go`, `viewer.go`, manuals, ARCHITECTURE, shards, qodana
Test: detach/restore; flag off keeps the menu; default follows OS
Verify: AC4 AC5

## Ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| T1–T3 | 0 / 0 | 1 | no | inline. Race on framemenu and the picture-frame UI tests. Full `./...` and Docker suite not run on this darwin host. |
| Review | 0 / 0 | 2 | no | Shutdown calls `slides.Close`, which does not notify the menu, so the dwell timer is stopped in `registerShutdown` and joined from the harness. |
| Review 2 | 0 / 0 | 3 | no | Deactivate holds the enqueue lock until a started callback has submitted or skipped `fyne.Do`. Leaving mid-slide already repaints every 16ms because the in-progress slide stays active; that delay is now tested. |
| Linux edge | 0 / 0 | 3 | no | glfw does not move window content up when the in-window menu is removed, so the hot strip started below the old menu. Picture-frame mode now sizes the content to the canvas. |
