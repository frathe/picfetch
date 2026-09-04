# Target the selected wallpaper display on Windows

Type: task
Status: resolved
Priority: P1
Blocked by: 02, 11

## Goal

Use Windows' monitor-aware desktop wallpaper interface for targeted mosaics
while preserving the current PowerShell/SystemParametersInfo no-target path.

## Existing code anchors

- Current `setWindows` is platform-neutral script construction in
  `internal/wallpaper/wallpaper.go`; only `hideConsoleWindow` is build-tagged.
  Its escaping and no-console tests run on non-Windows hosts and must remain.
- Ticket 02's Windows display ID is already the monitor device path returned by
  `IDesktopWallpaper.GetMonitorDevicePathAt`; no parser or HMONITOR translation
  layer is needed.
- `IDesktopWallpaper.SetWallpaper(NULL, path)` is not required for the legacy
  action. Keeping SystemParametersInfo for a zero target minimizes behavior
  change.

## Scope

- Keep the existing escaped PowerShell/SystemParametersInfo implementation for
  `Request.Target == ""`, including `CREATE_NO_WINDOW` and persistence flags.
- For a non-empty target, add a Windows-selected native adapter around
  `IDesktopWallpaper`. Initialize COM on a locked OS thread, release every COM
  and allocated-string resource, and pass the full path as UTF-16.
- Before mutation, call `GetMonitorRECT` (or an equivalent attached-monitor
  validation) for the exact device-path ID. Detached, stale, and unknown IDs
  must fail before `SetWallpaper` and change no monitor.
- Call `SetWallpaper(targetDevicePath, path)` for exactly one display. Do not
  shell out for the targeted path and do not apply globally on COM failure.
- Convert HRESULT failures into ordinary wrapped errors while preserving ticket
  11's typed unsupported outcome only for a genuinely unavailable target
  capability.
- Add Windows-selected tests for interface/vtable helpers and cross-platform
  tests for dispatch branching. Update Qodana for each new test file.

## Acceptance Criteria

- Zero-target calls produce the same escaped PowerShell script and flags as
  current tests.
- Targeted calls pass the opaque device path directly to
  `IDesktopWallpaper.SetWallpaper`; paths containing spaces and non-ASCII
  characters survive UTF-16 conversion.
- Unknown/detached target validation performs no SetWallpaper call.
- COM/HRESULT failure returns an actionable error and performs no global
  fallback.
- Windows-selected code vets and its test binary cross-compiles. One/two
  display behavior and persistence after exit are recorded in ticket 16.

```sh
go test ./internal/wallpaper -run 'TestSetWindows_(Legacy|Dispatch|Escapes)' &&
GOOS=windows GOARCH=amd64 go vet ./internal/wallpaper/... &&
GOOS=windows GOARCH=amd64 go test -c -o /tmp/picfetch-wallpaper-windows.test.exe ./internal/wallpaper &&
make check-qodana-test-exclusions
```

## Non-Goals

- Replacing the proven no-target SystemParametersInfo path
- Treating a stale target as permission to change every monitor

## Comments

Native smoke remains human-assisted in ticket 16, but the implementation itself
is agent work. Microsoft documents `IDesktopWallpaper` for desktop apps from
Windows 8 onward.

Implemented and verified on 2026-09-04: `IDesktopWallpaper` validation/set,
Unicode paths, legacy global dispatch, Windows test compilation, and cross-vet are green.
