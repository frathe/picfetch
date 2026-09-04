# Log mosaic UI-boundary failures

Type: task
Status: resolved
Priority: P2
Blocked by: 06

## Goal

Report mosaic window failures through `fyne.LogError` as the rest of the app
does, so a user-visible status line is not the only trace of an error.

## Evidence

- `AGENTS.md`, Project Conventions: "Report UI-boundary failures with
  `fyne.LogError`; viewer-independent packages return errors."
- Three failure paths in `internal/ui/mosaicwin/window.go` set status text only:
  `:402` (generation failure), `:497` (display refresh failure), `:596`
  (wallpaper failure).
- The same package already follows the rule at
  `internal/ui/mosaicwin/export.go:58`, so the omission is inconsistent within
  one package rather than a deliberate exception.

## Decisions

- Call `fyne.LogError` alongside the existing localized status text in all three
  paths, matching the call shape used in `export.go`.
- Keep the user-facing status text unchanged; this adds logging, it does not
  change presentation.
- Do not log cancellation as an error: a user-initiated cancel is not a failure.

## Acceptance Criteria

- Generation, display-refresh, and wallpaper failures each log through
  `fyne.LogError` in addition to setting status.
- Cancelling a generation logs nothing.
- Status text and localization are unchanged.

```sh
go test ./internal/ui/mosaicwin -count=1
```

## Non-Goals

- Adding a log destination or log level configuration
- Changing error text or translations
- Auditing logging outside `internal/ui/mosaicwin`

## Comments

Found by a standards-axis review of the branch on 2026-09-04.

## Answer

Generation, display-refresh, and wallpaper errors now call `fyne.LogError`
alongside their unchanged localized status updates. Tests capture Fyne's real
standard-logger output and assert both reason and cause for all three paths.
The cancellation test also captures the same logger and confirms it remains
silent. The full mosaic window package passes.
