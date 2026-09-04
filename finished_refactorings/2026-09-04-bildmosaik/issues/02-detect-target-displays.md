# Detect target displays

Type: task
Status: resolved
Priority: P0
Blocked by: none

## Goal

Expose attached desktop displays and the display containing most of PicFetch's
main window through one native seam, using native pixel geometry throughout.

## Existing code anchors

- `internal/winpos` is the established pattern for reaching a Fyne desktop
  window through `driver.NativeWindow.RunNative` and build-tagged adapters.
- `internal/ui/spiral.getMonitorInfo` reports only a window canvas size and a
  synthetic `main`/`ext` label. It is not display enumeration and must not be
  promoted or reused for this feature.
- OS integrations use dispatcher vars and `internal/uitest` restorers; tests
  must never enumerate or change the developer's real desktop.

## Scope

- Add `internal/displays` with a small interface such as
  `Inspect(fyne.Window) (Snapshot, error)`. `Snapshot` contains ordered
  `Display` values and the default display ID; `Display` contains an opaque
  typed ID, localized OS name, and native-pixel bounds.
- Make `Inspect` the stubbable dispatcher. Add `uitest.StubDisplays` matching
  the existing clipboard/file-picker/trash/wallpaper stub style.
- Let each native adapter choose the default display using native window and
  display rectangles. Do not mix `fyne.Size` logical units with native pixels
  in `internal/ui`.
- On Windows, use the monitor device path and rectangle exposed by
  `IDesktopWallpaper`; detached monitor records are not attached displays.
- On macOS, obtain the `CGDirectDisplayID` from
  `NSScreen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")]` (or an
  availability-safe equivalent), and use `localizedName`, frame, and backing
  conversion. IDs must be directly usable by ticket 13.
- On Linux/X11, enumerate real outputs and native geometry. On Wayland or a
  backend where the compositor exposes no truthful global display topology,
  return a typed unsupported error rather than inventing displays from the
  current canvas size.
- Document the new package in `ARCHITECTURE.md` in this change and update the
  exact Qodana test exclusions.

## Acceptance Criteria

- Callers compare but never parse display IDs; IDs remain stable for the life
  of an attached display session.
- Pixel bounds are physical/native bounds, including HiDPI displays, and may
  have negative origins in a multi-display arrangement.
- The default ID is the display with the greatest intersection with the main
  window, with a deterministic fallback when the window rectangle is
  unavailable.
- An empty attached-display set and unsupported enumeration are explicit typed
  outcomes, not empty success.
- Unit tests run entirely through adapter fixtures/stubs. Native topology is
  deferred to ticket 16's smoke matrix.

```sh
go test ./internal/displays ./internal/uitest -run 'Test(Display|Inspect|StubDisplays)' -count=1 &&
GOOS=windows GOARCH=amd64 go vet ./internal/displays/... &&
make check-qodana-test-exclusions
```

## Non-Goals

- Persisting a selected display across app restarts
- Wallpaper mutation
- Pretending the existing Spiral canvas measurement is monitor detection

## Comments

Windows `IDesktopWallpaper.GetMonitorDevicePathAt/GetMonitorRECT` and macOS
`NSScreenNumber`/`CGDirectDisplayID` provide IDs that the later wallpaper
adapters can consume without a second mapping scheme.

Implemented and verified on 2026-09-04: native-pixel snapshots, typed failures,
stubs, Linux XRandR parsing, macOS inspection, and Windows cross-vet are green.
