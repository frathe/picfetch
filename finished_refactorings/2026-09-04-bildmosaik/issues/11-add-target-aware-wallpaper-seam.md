# Add a target-aware wallpaper seam

Type: task
Status: resolved
Priority: P1
Blocked by: 02, 08

## Goal

Extend the one existing wallpaper seam and persistent-copy lifecycle so both
ordinary viewer pixels and mosaic result pixels can request an optional display.

## Existing code anchors

- `wallpaper.Set` is a stubbable `func(path string) error` dispatcher. Platform
  behavior is selected in `internal/wallpaper/wallpaper.go` plus Darwin and
  Windows build-tag helpers.
- `viewer.setAsWallpaper -> applyWallpaper -> writeWallpaperFile` captures the
  displayed frame, writes a timestamped PNG under `viewer.wallpaperDir`, calls
  the seam, deletes a failed new copy, then `sweepWallpapers` keeps one file.
- `viewer.wallpaper completion.Signal` makes one operation observable, and
  `uitest.StubWallpaperSet` protects the real desktop.

## Scope

- Replace the path-only dispatcher interface once with
  `wallpaper.Request{Path, Target}` where `Target` is the opaque
  `displays.ID`; its zero value means the existing all-desktop/no-target action.
  Update `uitest.StubWallpaperSet` to capture the request.
- Add a typed `TargetUnsupportedError` (or equivalent `errors.As` contract)
  distinct from ordinary execution/path errors. The typed error must be
  preserved through UI reporting so `mosaicwin` can explain a limitation rather
  than call it a generic failure.
- Refactor `internal/ui/wallpaper.go` around one common write-then-set helper
  that accepts captured pixels, user-facing label, and optional target. Existing
  `setAsWallpaper` still captures `v.img.Image`; the mosaic Host adapter passes
  pixels captured from the latest `mosaic.Result` and its selected target.
- Keep all file writes under `viewer.wallpaperDir` and through
  `imaging.Export(..., nil)`. Never point the OS at a source file or let a
  wallpaper call mutate one.
- Serialize/reject overlapping viewer and mosaic wallpaper operations so
  `completion.Signal`, cache cleanup, and user feedback cannot race.
- Make cleanup target-aware. A targeted success may sweep only the previous
  copy for that same target; it must retain files that may still back other
  displays, including a preceding global file. A no-target success may sweep
  all older PicFetch wallpaper copies. Use a filesystem-safe derived scope key,
  never a raw monitor device path in a filename.
- On unsupported/failure, delete only the just-written copy, sweep nothing,
  retain the mosaic preview, and re-enable export/wallpaper actions after the
  completion callback reaches `mosaicwin` on its UI queue.
- Add/translate limitation and busy/failure text in both catalogues in this
  ticket; update Qodana for new test paths.

## Acceptance Criteria

- Every existing no-target single-image wallpaper test stays green with the
  new request shape and current on-screen rotation behavior.
- Mosaic wallpaper receives the exact latest result pixels and selected opaque
  ID without reading `viewer.img` or regenerating.
- Two target IDs can retain two active cache files; replacing one target cannot
  delete the other's file. Global replacement returns to the legacy one-file
  cleanup behavior.
- Concurrent main-window/mosaic requests cannot race files, sweeps, signals, or
  callbacks.
- `errors.As` distinguishes target unsupported from command/native failure;
  neither closes the preview or disables Save after completion.
- Tests stub the new dispatcher and never change the developer's desktop.

```sh
go test ./internal/wallpaper -run 'Test(Request|TargetUnsupported)' -count=1 &&
go test ./internal/ui -run 'Test.*Wallpaper' -count=1 &&
go test ./internal/ui/mosaicwin -run 'TestMosaicWallpaper' -count=1 &&
go test . -run 'TestTranslations_' -count=1 &&
make check-qodana-test-exclusions
```

## Non-Goals

- A second mosaic-specific wallpaper package or cache directory
- Persisting source URIs or raw display IDs in filenames
- Implementing native target behavior (tickets 12-14)

## Comments

The old `sweepWallpapers(dir, keep)` policy is unsafe once two monitors can
reference two different PicFetch files; target-scoped retention is required.

Implemented and verified on 2026-09-04: shared serialized apply, opaque target
routing, exact captured pixels, scoped retention, and failure cleanup are green.
