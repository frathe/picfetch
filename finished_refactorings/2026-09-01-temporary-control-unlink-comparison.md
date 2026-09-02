# Temporary Control unlink in comparison

Status: complete

Route: Standard. This extends `internal/ui/compare`, the assembled viewer input
seam, translations, and both manuals. It adds no package, dependency,
preference, or external API.

Deliverable: a fresh physical Control hold temporarily restores two retained
pane-local views; release relinks from the hovered pane under the existing
shared no-blank clamp.

## Locked decisions

| Decision | Contract |
|---|---|
| Scope | The current comparison layout default is unchanged. Control means the physical Control key on every desktop platform. |
| Target | Pointer gestures and `0`, `1`, `+`, and `-` affect the hovered or last-hovered pane. With no target, local keyboard transforms do nothing. |
| Relink | Release always chooses the hovered pane, even when it was not edited, then applies the shared clamp. |
| Local bounds | Local centers are bounded to normalized `[0,1]`, allowing an image edge to reach pane center. |
| Retention | Pane-local views persist for the comparison session. Linked pan/zoom applies the same normalized delta/scale ratio to both caches; linked `0`/`1` changes their scale modes but retains their centers. |
| Transitions | Resize and layout toggles preserve both local views. Swap relinks, clears divergence, swaps, and suppresses the still-held Control until release. |
| Feedback | Show only `Unlinked`, `Unlinked: Left`, or `Unlinked: Right` in the toolbar while a fresh Control hold is active. |

## Tasks

### Task 1 - Pane-local transform state

Owner: T0 inline

Test first through `compare.Feature` and its overlay: local pointer/key input,
hover targeting, overscroll, release clamping, retained caches, relayout, Swap,
session reset, status, and vector rendering.

Verify: `go test ./internal/ui/compare -run 'Compare.*(Control|Unlink|Relink|Local|Cache)' -count=1`

### Task 2 - Assembled input wiring

Owner: T0 inline

Test first through the viewer and production shortcut/key-hook wiring: physical
Control lifecycle, Ctrl+D suppression, modified transform/divider keys,
Ctrl+0/Ctrl+1 favorite preservation, and covered-grid isolation.

Verify: `go test ./internal/ui -run 'Compare.*(Control|Unlink|Relink|Local|Cache)' -count=1`

### Task 3 - User documentation and landing

Owner: T0 inline

Localize the status strings, update both manuals and `ARCHITECTURE.md`, then
replace the open TODO with a verified release-note entry. Preserve the
historical finished comparison specification.

Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Budget and gate

Zero spawns; at most three review rounds; one full suite. Each behavior guard
must be observed red before implementation and fail once more under a deliberate
mutation before final acceptance. Final gate: Windows vet followed by `make
verify`.

## Outcome

Implemented inline with zero spawns. Comparison now owns retained pane-local
transforms, physical-Control lifecycle/status, target-based pointer and keyboard
input, release-time relinking/clamping, cache propagation during linked input,
layout/resize preservation, Swap suppression/reset, and pane-local SVG raster
targets. Viewer wiring chains desktop modifier hooks and registers modified-key
repeat shortcuts; Favorites temporarily releases its digit accelerators while
comparison is active so Windows/Linux Control+0/1 reaches comparison and is
restored afterward. Both manuals, locale catalogs, `ARCHITECTURE.md`, and the
release notes describe the landed behavior; side-by-side remains the default.

Verification completed 2026-09-01:

- Focused compare, viewer, Favorites, manual, translation, and Unicode-arrow
  tests passed.
- Deliberate mutations were detected by the local-render, cache propagation,
  status/relink/suppression, bounds/reset, shortcut/hook, and accelerator guards;
  the restored focused set passed.
- `GOOS=windows GOARCH=amd64 go vet ./internal/...` passed.
- `make verify` passed, including the Linux/amd64 race suite (`internal/ui` in
  639.593s).
