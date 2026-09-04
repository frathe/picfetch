# Clean up mosaic dead branches and misleading names

Type: task
Status: resolved
Priority: P3
Blocked by: 06

## Goal

Remove small pieces of mosaic code that mislead a reader about what the program
does.

## Evidence

- `internal/mosaic/generator.go:96-102`: both arms of the cancellation check
  return the same thing.

  ```go
  if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
      return Result{}, err
  }

  return Result{}, err
  ```

  The branch implies cancellation is handled specially. It is not.
- `internal/ui/mosaicwin/window.go:642`: `_, target := w.selectedDisplay()` binds
  a `bool` to `target`, while the field `w.target` in the same type is a
  `displays.ID`. One name, two meanings, in one scope.
- `internal/ui/mosaicwin/window.go:293`: `settingSlider` forwards its arguments
  verbatim to `newNamedSlider` and adds nothing.
- `internal/wallpaper/target_windows.go:122`: `applyWindowsTarget` only calls two
  injected closures in order.

## Decisions

- Either delete the dead `errors.Is` branch or give cancellation the distinct
  handling the branch implies; do not leave the fork in place.
- Rename the boolean to say what it means, for example `ok` or `found`.
- Inline `settingSlider` and `applyWindowsTarget` at their call sites unless a
  test seam depends on them, in which case say so in a comment.

## Acceptance Criteria

- No branch in `internal/mosaic` forks to identical outcomes.
- No identifier in `mosaicwin` names a bool the same as a `displays.ID` field.
- Mosaic and Windows wallpaper behaviour is unchanged.

```sh
go test ./internal/mosaic ./internal/ui/mosaicwin -count=1 &&
GOOS=windows GOARCH=amd64 go build ./internal/...
```

## Non-Goals

- Splitting `window.go` (721 lines, several responsibilities); that is its own
  refactor if it is wanted
- Deduplicating the repeated `FrameStyle` and `ExportFormat` switches
- Changing the layout algorithm

## Comments

Found by a standards-axis review of the branch on 2026-09-04.

## Answer

Removed the generator's identical-outcome branch, renamed the display-presence
boolean to `attached`, and inlined the no-op `settingSlider` wrapper. Retained
`applyWindowsTarget` because its injected closures are the Windows-only test
seam that proves stale-display validation completes before any mutation; its
comment now records that purpose.

The focused mosaic suites and the Windows internal-package cross-build pass.
