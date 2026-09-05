# Implementation Plan: Return to mosaic settings

Status: ready-for-human
Route: Standard
Spec: `.scratch/bildmosaik/spec.md` AC13
Issue: `.scratch/bildmosaik/issues/19-return-to-settings-from-preview.md`

## Frame

Add a top-left result-screen action that returns the existing mosaic window to
configuration while retaining the values needed to target another monitor.
Generation, wallpaper routing, and cross-window persistence are out of scope.

The route is Standard because this is one user-visible workflow change with
localization and manual updates, despite staying inside one feature package.

## Decisions - Do Not Relitigate

| Decision | Resolution |
| --- | --- |
| UI seam | Observe and operate the existing `mosaicwin.Window` canvas |
| Label and placement | Localized **Start Over**, top-left of the preview |
| Retained state | Sources, target, visual settings, and export format |
| Discarded state | Result pixels and result status |
| Focus | Target-display selector after returning |

## Task Graph

```text
T1 Result-to-configuration transition
  -> T2 Localization and documentation
  -> T3 Final gate
```

## Tasks

### Task 1 - Add the result transition

Owner: T0 inline
Files: `internal/ui/mosaicwin/window.go`, `internal/ui/mosaicwin/mosaicwin_test.go`
Depends: none
Contract: a finished preview exposes Start Over; activation restores the
configuration and permits generation for a newly selected display
Test: drive generation and Start Over through the window UI, then generate for
a second display and observe its result dimensions
Verify: `go test ./internal/ui/mosaicwin -run 'TestMosaic(StartOver|Keyboard|Accessibility)' -count=1`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete. The behavior guard first failed because the preview had no
accessible Start Over action, then passed with the in-place transition. The
full `internal/ui/mosaicwin` package also passes.

### Task 2 - Localize and document

Owner: T0 inline
Files: `translations/en.json`, `translations/de.json`, `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`, spec, issue, `todos.md`
Depends: T1
Contract: every displayed label is localized and both manuals describe the
multi-display continuation
Test: catalogue parity and manual guards
Verify: `go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1`
Budget: 0 implementation spawns; 1 review round; full suite: no
Result: complete. The bilingual manual guard first failed on all four missing
phrases, then passed after both catalogues and manuals were updated.

### Task 3 - Final gate

Owner: T0 inline
Files: none
Depends: T1, T2
Contract: repository verification remains green
Test: repository final gate
Verify: `make verify`
Budget: 0 implementation spawns; 1 review round; full suite: yes, once
Result: complete. The first concurrent race run lost one unchanged compare
test process to an OS-level `signal: killed`; that exact test passed alone, and
a warmed-cache `make verify` retry passed every check and partition. Final
review then tightened Start Over's busy-state coverage for regeneration,
export, and wallpaper work; the expanded acceptance set passes both natively
and under the canonical Linux/amd64 race detector.

## Delegation Gate

Implementation stays T0 inline: the test is design-bearing, the string and
translations may never be delegated, and T0 already owns the hot workflow
context (G4/G5). Two read-only Scouts were reused for independent layout/focus
and localization/documentation reconnaissance; neither edits files.

## Cost Ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | ---: | --- | --- |
| Recon | 2 / 2 | - | no | reused bounded read-only Scouts |
| T1 | 0 / 0 | 2 | no | red then green; full feature package passed; final review clean after busy-path hardening |
| T2 | 0 / 0 | 1 | no | bilingual guard red then green |
| T3 | 0 / 0 | 1 | yes | `make verify` passed; focused acceptance race set passed after final test hardening |
