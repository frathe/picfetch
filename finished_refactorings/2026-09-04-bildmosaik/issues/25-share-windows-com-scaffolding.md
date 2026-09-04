# Share the Windows COM scaffolding and mark ignored syscall results

Type: task
Status: resolved
Priority: P2
Blocked by: 02, 12

## Goal

Keep one copy of the `IDesktopWallpaper` COM plumbing used by display detection
and wallpaper targeting, and make discarded syscall results explicit as the
repository requires.

## Evidence

- `internal/displays/windows.go:15-55` and
  `internal/wallpaper/target_windows.go:12-51` each declare their own copy of the
  same scaffolding: identical `desktopWallpaperVtbl` layouts, the same CLSID and
  IID literals, the same `coinit*` and `clsctx*` constants, and
  `failedHRESULT` / `failedWallpaperHRESULT` differing only in name. A vtable
  layout duplicated across packages is a correctness hazard, not only a
  tidiness one: the two copies can drift and only one will be wrong.
- `AGENTS.md`, Project Conventions: "Mark intentionally ignored errors explicitly
  (`_ =` or `_, _ =`) so IDE/`errcheck` inspections see intent." Bare calls that
  discard results: `internal/displays/windows.go:69` (`procCoUninitialize.Call()`),
  `:93` (`procCoTaskMemFree.Call(...)`), and
  `internal/wallpaper/target_windows.go:78`.

## Decisions

- Move the shared vtable, CLSID/IID literals, COM constants, and the HRESULT
  predicate into one internal, Windows-tagged home, and have both call sites use
  it. `internal/displays` must not import `internal/wallpaper` or vice versa; if
  no existing package is the right owner, add a small one and record it in
  `ARCHITECTURE.md`.
- Assign the discarded results of the three bare `.Call()` sites to `_`.
- Keep the two features' behaviour identical; this is a consolidation, not a
  redesign.

## Acceptance Criteria

- The `IDesktopWallpaper` vtable layout, CLSID, IID, and COM constants are
  declared exactly once in the repository.
- No bare `.Call()` discards a result without an explicit `_` assignment.
- Windows display detection and target wallpaper application behave as before.
- `ARCHITECTURE.md` reflects any new package.

```sh
GOOS=windows GOARCH=amd64 go build ./internal/... &&
go test ./internal/displays ./internal/wallpaper -count=1
```

## Non-Goals

- Replacing the syscall approach with a third-party COM binding
- Changing display detection or wallpaper behaviour
- Touching the macOS or Linux backends

## Comments

Found by a standards-axis review of the branch on 2026-09-04. The behavioural
part needs a native Windows check; CI cross-compiles but does not run it.

## Answer

Added `internal/wincom` as the single owner of the `IDesktopWallpaper` GUIDs,
vtable layout, COM activation constants, and HRESULT predicate. Its interface
hides the vtable fields behind the exact procedure accessors the two adapters
need. Display inspection and targeted wallpaper retain their feature-specific
COM lifetime and error behavior, and all three ignored `.Call` results are now
explicit.

The duplicate/bare-call guard was negatively verified. Windows `go vet` and the
CI-supported `GOOS=windows GOARCH=amd64 go build ./internal/...` pass. The old
whole-repository cross-build command was corrected because `.github/workflows/ci.yml`
documents that Fyne's main package cannot be built that way. Native Windows
execution remains part of the supervised smoke matrix.
