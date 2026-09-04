# Target the selected wallpaper display on macOS

Type: task
Status: resolved
Priority: P1
Blocked by: 02, 11

## Goal

Set a targeted mosaic on exactly the `NSScreen` identified by ticket 02 while
preserving the existing all-screen AppKit behavior for ordinary wallpaper.

## Existing code anchors

- `internal/wallpaper/darwin.go` already calls
  `NSWorkspace.setDesktopImageURL:forScreen:options:error:` in-process, loops
  `[NSScreen screens]`, and passes each screen's current desktop-image options.
- The current implementation deliberately avoids AppleScript/System Events and
  therefore avoids an Automation permission prompt.
- Ticket 02 exposes the same `CGDirectDisplayID` carried by
  `NSScreen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")]` (or an
  availability-safe equivalent); no name matching or geometry heuristic is
  needed at wallpaper time.

## Scope

- Extend the Darwin bridge to accept an optional display ID. Parse only the
  package's own stable decimal encoding of `CGDirectDisplayID`; callers still
  treat the value as opaque.
- Enumerate all `NSScreen` values and resolve the target before changing any
  screen. A malformed, missing, or stale ID must return an actionable error
  with zero partial mutation.
- For a selected screen, read that screen's existing
  `desktopImageOptionsForScreen:` and pass them unchanged to one
  `setDesktopImageURL` call.
- For a zero target, preserve the current preflight and all-screen loop. Do not
  change its options, error, persistence, or cache-path behavior.
- Keep AppKit in-process; add no `osascript`/Apple Events path.
- Make the Go request-routing logic testable without a real desktop mutation by
  passing an unexported native function into a package-private helper. Do not
  add another mutable package-level test seam.
- Add Darwin-selected tests and Qodana entries. Native two-display acceptance
  remains ticket 16.

## Acceptance Criteria

- Targeted routing resolves one exact `CGDirectDisplayID` and invokes the native
  setter once with that screen's unchanged options.
- Target resolution completes before mutation; unknown IDs invoke no setter.
- Zero target retains the existing all-attached-screen behavior.
- No code path invokes Apple Events or changes Fill/Fit/Stretch/background
  color options.
- Native macOS tests compile and pass without altering the test machine's real
  wallpaper. Internal/external display targeting and persistence after exit are
  recorded in ticket 16.

```sh
go test ./internal/wallpaper -run 'TestSetDarwin_(Target|NoTarget|Missing)' &&
make check-qodana-test-exclusions
```

## Non-Goals

- Persisting a mapping by display name
- Changing display arrangement or desktop-image options
- AppleScript fallback

## Comments

Native smoke needs a macOS host with an external display, but implementation and
side-effect-free adapter tests are agent work.

Implemented and verified on 2026-09-04: decimal display-ID validation, exact
`NSScreen` preflight, single-screen apply, and preserved all-screen behavior are green.
