# Mosaic rotation and visible overlap

Status: rotation implemented locally; overlap design decision pending.
Route: Standard, following the existing ready-for-agent specification despite
its supporting control, persistence and manual checks spanning four packages.

Deliverable: complete both [mosaic tickets](../.scratch/mosaic-rotation-overlap/README.md),
extending rotation to 90 degrees and making overlap visibly affect a completely
covered composition in which every retained photo occurrence remains visible.

## Contract and scope

The parent [specification](../.scratch/mosaic-rotation-overlap/spec.md) supplies
AC1–AC8, their exact commands, decisions and non-goals. Those remain authoritative.
Use its pre-agreed generator-to-result, renderer, control-to-request and
preference save/load seams. Keep defaults, approximate-inset semantics and
existing interfaces. No public attribution API, dependencies or commits.
Outer clipping remains allowed; visibility does not guarantee subject visibility.
The user's existing `todos.md` change supplies the implementation target.

## Tasks and graph

T1 -> T2 -> T3 -> gate. Scout S runs alongside T1 and informs T2 test helpers.

### T1 — Full rotation path
Owner: T0 inline.
Files: internal/mosaic/{mosaic.go,mosaic_test.go,generator_test.go,quality_test.go},
internal/ui/mosaicwin/{window.go,mosaicwin_test.go},
internal/preferences/preferences_test.go, internal/ui/help/{manual.md,manual_de.md,manual_test.go}.
Depends: none.
Contract: existing Settings, Request, Generate, renderer and Save/Load interfaces.
Test: AC1, AC2 rotation, AC5 and manual range/inset guard, one vertical slice at a time.
Verify: parent AC1, AC2, AC5, AC6, AC7 documentation commands.
Budget: 0 implementation spawns; 1 review round; no full suite.

### S — Existing final-pixel test instrumentation
Owner: read-only Scout.
Files: read internal/mosaic/{generator_test.go,layout_test.go,render.go,preparation.go,generator.go} only.
Depends: none.
Contract: return existing helpers and source/cache/placement observation points
with file:line; no design decisions, changes or review.
Test: reconnaissance only.
Verify: lead checks returned identifiers with rg and targeted reads.
Budget: 1 spawn; no full suite.
Delegation gate: G1 bounded question; G2 file/identifier lookup verifies findings;
G3 no writes; G4 only relevant test machinery; G5 lead has not read those tests.
Rule S: comprehension across rendering and cache tests, not a text transform.
Rule W: no implementation prescribed. The configured harness has no Explore or
Sonnet/Haiku model, so this is one inherited-model read-only scout; no review
or implementation is delegated.

### T2 — Reproduce and fix overlap and visibility together
Owner: T0 inline.
Files: internal/mosaic/{layout.go,generator.go,generator_test.go,layout_test.go},
internal/ui/mosaicwin/mosaicwin_test.go; a separate registered mosaic test file if needed.
Depends: T1; S informs fixture reuse only.
Contract: existing request/result and internal layout/render lifecycle.
Test: AC3 and AC4 final-pixel fixtures, repeat occurrence accounting and calibrated
edge tolerance; observable cancellation and coverage remain mandatory.
Verify: parent AC3, AC4 and AC6 commands, then AC5 joint regression.
Budget: 0 spawns; 2 review rounds; no full suite.

### T3 — Visual evidence and documentation
Owner: T0 inline.
Files: existing manuals/tests as needed, evidence under .scratch/mosaic-rotation-overlap,
tickets, todos.md, this plan, ARCHITECTURE.md if the package map changes.
Depends: T2.
Contract: retained paired artifacts report settings, seed, metrics and full/preview inspection.
Test: AC7, negative visibility/rotation guards and exact wallpaper delivery.
Verify: parent AC7 and ticket 02 wallpaper command.
Budget: 0 spawns; 1 review round; no full suite.

### Gate — Lead review and repository verification
Owner: T0 inline.
Files: all changed files and verification evidence.
Depends: T3.
Contract: match spec against observed commands, close findings inline, no commit.
Test: all acceptance commands execute real scenarios; make verify runs once.
Verify: make verify.
Budget: 0 spawns; 1 final full suite, repeated only for failures or new changes.

## Checklist and evidence

- [x] Frame and source specification identified.
- [x] Existing test boundaries and commands recorded.
- [x] Reconnaissance and task ownership recorded.
- [x] Rotation red/green slices and focused regression checks.
- [ ] Final-image symptom reproduction and overlap/visibility red/green slices.
- [ ] Lead review, negative guards and visual inspection.
- [ ] Full verification and open-work updates.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
|------|----------------------|---------------|------------|----------|
| S | 1/1 | 0 | no | Existing occurrence/pixel helpers located and checked inline |
| T1 | 0/0 | 1 | no | AC1, AC2, AC5 and manual/root checks pass; preference and angle-clamp guards observed red |
| T2 | 0/0 | 1 | no | Reproductions fail; bounded placement/reflow experiments retained outside production |
| T3 | 0/0 | 1 | no | Prototype and baseline inspected; final visual evidence pending |
| gate | 0/0 | 0 | pending | verify-build and shard checks pass; full race gate awaits integrated implementation |

## Current evidence and decision

See [the evidence record](../.scratch/mosaic-rotation-overlap/evidence/README.md)
for commands, measured failures, artifacts and limits. The baseline generator
and layout are unchanged. Only rotation validation/normalization and the slider
range have production changes. The two new overlap test groups intentionally
remain red; no integrated or full-suite success is claimed.

Search that protects all retained photos cannot simply substitute for unrestricted
repairs: representative fixtures leave gaps. Bounded backtracking and enlargement
within the old size envelope also exhausted their search bounds. A prototype
allowing growth up to three times the ordinary upper short-edge bound completed
one fixture with 43 occurrences and a sampled minimum visible fraction of 47.2%.
This is evidence about these algorithms, not proof that the original constraints
are mathematically impossible. The prototype changes the size contract and
defers rendering until planning finishes, so its source-cache/lifecycle effects
also need review before any integration.

The user has been asked whether gap repair may enlarge photos beyond the existing
size range. The question remains pending; elapsed time is not approval. No such
enlargement has been applied to production. Work can resume from the retained
experiments after the decision, or continue searching within the original range.
