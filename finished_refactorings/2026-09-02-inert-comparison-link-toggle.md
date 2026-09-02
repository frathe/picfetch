# Inert comparison link toggle

Status: complete

Route: Standard. This corrects one comparison interaction across
`internal/ui/compare`, its manuals, release notes, and architecture record. It
adds no package, dependency, preference, user-visible string, or external API.

Deliverable: locking or unlocking comparison changes only which transform owns
future input; neither transition changes either photo's visible size or
position.

## Locked decisions

| Decision | Contract |
|---|---|
| Photo state | Each photo keeps its own position, scale, and fit/actual mode for the comparison session. |
| Camera state | Linked controls operate one shared overhead camera composed over both photo poses. |
| Link toggle | Physical `Ctrl+L` changes input ownership only. Unlock and relock are exact geometry no-ops. |
| Linked resets | `0` frames both current poses with one camera move; `1` returns the camera to its 1x home without rewriting photo poses. |
| Unlinked resets | `0` fits and centers only the target photo in the current camera; `1` shows only that photo at decoded-pixel size. |
| Bounds | A local photo or the shared camera may expose the table, but cannot move a photo completely past its pane center. |
| Existing transitions | Resize and layout preserve photo and camera state. Swap retains its explicit relink/reset behavior before exchanging sources. |

## Tasks

### Task 1 - Separate persistent photo poses from the shared camera

Owner: T0 inline

Files: `internal/ui/compare/compare.go`, `internal/ui/compare/transform.go`,
`internal/ui/compare/input.go`, `internal/ui/compare/compare_test.go`

Test first through `compare.Feature`: unlock/relock geometry is exact, unlinked
input changes one photo, linked input moves both through one camera, camera fit
and home preserve divergent poses, movement remains bounded, and resize/layout
round trips retain the composed state.

Verify: `go test ./internal/ui/compare -count=1`

### Task 2 - Preserve assembled comparison behavior

Owner: T0 inline

Files: existing viewer and Favorites tests only; no production change expected.

Verify: `go test ./internal/ui -run 'Compare' -count=1 && go test
./internal/ui/favorites -run 'Compare' -count=1`. The complete UI suite remains
part of the Linux/amd64 final gate because native golden rendering is not
authoritative.

### Task 3 - Documentation and final gate

Owner: T0 inline

Files: `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`,
`internal/ui/help/manual_test.go`, `ARCHITECTURE.md`, `todos.md`, this plan.

Document the photo/table/camera model and the differing linked/unlinked `0` and
`1` meanings. Preserve the superseded Ctrl+L plan as historical evidence.

Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Budget and gate

Zero spawns; at most three review rounds; one full suite. Negatively verify the
exact relock guard before the final `make verify` run.

## Outcome

Comparison now composes two persistent photo transforms with one shared camera.
Unlocking and relocking change only input ownership, so both transitions retain
the exact rendered size and position of each photo. Unlinked input edits one
photo; linked zoom, pan, fit, and home move only the camera over the retained
arrangement. Camera-aware photo and camera bounds keep each image over its pane
center, while resize/layout round trips and the existing Swap reset semantics
remain intact.

The manuals, their behavior guard, architecture map, and release notes now use
the same photo/table/camera model.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 2 | no | Photo/camera split, inert toggles, composed controls, bounds, and component guards complete. |
| T2 | 0 / 0 | 1 | no | Assembled comparison and Favorites integration guards passed. |
| T3 | 0 / 0 | 1 | no | Manuals, manual guard, architecture, TODO, and plan records complete. |
| gate | - | - | - | yes | `make verify` passed. |

## Verification record

- `go test ./internal/ui/compare -count=1` passed.
- `go test ./internal/ui -run 'Compare' -count=1` and the matching Favorites
  command passed.
- `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1` passed.
- The camera-offset local-bound guard first failed with the target photo wholly
  beyond its pane center, then passed after the bound became camera-aware.
- A deliberate relock-time photo-transform overwrite made
  `TestCompareLinkToggle_RelockKeepsDivergentPhotoPoses` fail with the expected
  geometry change; the restored implementation passed the same exact guard.
- `make verify` passed formatting, TUF root validation, vet, build, and the
  Linux/amd64 race suite (`internal/ui` 679.263s;
  `internal/ui/compare` 17.186s).
