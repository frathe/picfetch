# Implementation Plan: Mosaic source fidelity

Status: ready-for-human
Route: Standard
Spec: `.scratch/bildmosaik/spec.md` AC14
Issue: `.scratch/bildmosaik/issues/20-harden-source-fidelity-and-selection.md`

## Frame

Close three source-selection gaps at the existing module boundaries: decoded
raster orientation in `mosaic.Generate`, without-replacement URI sampling in
the generator, and hidden-duplicate representative resolution in the
Actions-to-mosaic snapshot. Layout, decode-format coverage, and Grid selection
semantics otherwise stay unchanged.

The route is Standard because the behavior crosses the viewer-independent
mosaic module and the cross-feature UI composition seam.

## Decisions - Do Not Relitigate

| Decision | Resolution |
| --- | --- |
| Orientation seam | Compare `mosaic.Generate` output for metadata-oriented input and its display-ready lossless equivalent |
| Sampling identity | Exact URI string, collapsed before shuffle |
| Duplicate identity | Existing highest-resolution representative from duplicate visibility |
| Explicit selection | Substitute a hidden representative only while duplicate filtering is active |

## Task Graph

```text
T1 Oriented raster geometry
  -> T2 Distinct-URI source pass
  -> T3 Hidden-duplicate snapshot
  -> T4 Final gate and ledger
```

## Tasks

### Task 1 - Preserve PNG/WebP EXIF orientation and decoded raster geometry

Owner: T0 inline
Files: `internal/imaging/exif.go`, `internal/imaging/loader.go`, `internal/imaging/loader_test.go`, `internal/mosaic/generator.go`, `internal/mosaic/generator_test.go`
Depends: none
Contract: PNG `eXIf` and WebP `EXIF` inputs render like the same pixels after their declared display transform
Test: deterministic public `Generate` differential test
Verify: `go test ./internal/mosaic -run 'TestGenerate_RespectsDecodedOrientation' -count=1`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete. The public differential guard first rendered PNG and WebP
orientation-6 fixtures differently from the same pixels transformed to
display-ready form. After extending the shared orientation reader and using
decoded raster bounds, both cases pass alongside the existing canonical format
suite and full imaging/mosaic package tests.

### Task 2 - Sample distinct URIs before reuse

Owner: T0 inline
Files: `internal/mosaic/generator.go`, `internal/mosaic/generator_test.go`
Depends: T1
Contract: duplicate request entries cannot repeat a successfully loaded URI while unused distinct entries remain
Test: record successful loader calls through `Generator.Generate`
Verify: `go test ./internal/mosaic -run 'TestGenerate_UsesDistinctSourceURIsBeforeReuse' -count=1`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete. With 128 distinct URIs repeated three times in the request,
the guard first loaded `source-31.png` twice while 118 choices remained unused.
Collapsing exact URI identities before the deterministic shuffle makes the same
public generation pass without changing lazy loading or post-exhaustion reuse.

### Task 3 - Resolve selected hidden duplicates

Owner: T0 inline
Files: `internal/ui/mosaic.go`, `internal/ui/mosaic_test.go`
Depends: T2
Contract: the mosaic window snapshots only the highest-resolution representative of a selected hidden duplicate group
Test: select a lower-resolution Grid duplicate, enable hiding, and inspect the Actions-created window snapshot
Verify: `go test ./internal/ui -run 'TestMosaicSources_HiddenDuplicatesUseHighestResolution' -count=1`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete. The Actions-created snapshot first retained `a-small.jpg`
instead of its larger representative, and a two-member selection retained both
files. The resolver now substitutes and collapses the group while the browse
exception remains unchanged; focused mosaic UI tests pass.

### Task 4 - Record and verify

Owner: T0 inline
Files: spec, issue 20, `ARCHITECTURE.md`, `todos.md`, `.github/testshards/internal-ui.tsv`, this plan
Depends: T1, T2, T3
Contract: the regression guarantees remain documented and repository verification stays green
Test: focused acceptance, package tests, negative guard evidence, and repository final gate
Verify: `make verify`
Budget: 0 implementation spawns; 1 review round; full suite: yes, once
Result: complete. The new top-level UI guard was assigned to the canonical
three-shard manifest, whose Linux/amd64 inventory reports all 582 runnable
tests assigned. `make verify` passes sync checks, vet, build, every non-UI race
package, and all three UI race partitions.

## Delegation Gate

Implementation stays T0 inline because all three tests define product behavior
and share the same source-snapshot contract. Two bounded read-only Scouts trace
the renderer/source pool and duplicate-visibility state; neither edits files.

## Cost Ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | ---: | --- | --- |
| Recon | 2 / 2 | - | no | bounded read-only source and duplicate-flow traces |
| T1 | 0 / 0 | 2 | no | red then green; imaging and mosaic packages pass |
| T2 | 0 / 0 | 2 | no | red then green; lazy and repeat behavior preserved |
| T3 | 0 / 0 | 2 | no | two red cases then green; browse exception pinned |
| T4 | 0 / 0 | 1 | yes | `make verify` passed after assigning the new UI guard to its race shard |
