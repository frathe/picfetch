# Browsing source transitions and occurrence identity

Status: implemented and verified locally; branch acceptance pending. Route: Deep (root, Grid, search and a shared
identity value). This is the first follow-up from the assessment of all 44 Codex
findings on PR 25. The [complete assessment and finding inventory](../docs/find-more-like-this/pr25-complete-architecture-review.md)
records resolved findings as well as the later review rounds. Ronin authorized
starting the work and requested SDD/TDD, then explicitly reauthorized commits.

## Problem and contract

Source removal, committed writes and terminal search recovery currently sequence
search exit, Grid reconciliation, comparison handling and loads in their callers.
The review history found partial fixes to the same ordering rule in successive
paths. Image and Grid visits also implement path occurrence capture separately.

Deliverable: one root-owned source transition applies the complete change before
restoring browsing; image and Grid bookmarks share one immutable occurrence
identity/index contract. Existing behavior remains, including frozen image order,
live search progress, Grid interaction restoration and conservative source recovery.

| Decision | Contract |
| --- | --- |
| Source changes are facts | Ordinary removal admission stays separate from completed Trash/write reconciliation. One transition owns mutation, derived Grid updates, analysis retirement and restoration. |
| Search yields its origin | Search can detach an owned origin snapshot and retire its session without calling Host.Restore. Root restores it after reconciliation. Normal Back/Exit still restore immediately. |
| One load owner | During display failure the transition selects/restores state but returns the successor to display's existing retry chain; it must not start a competing ShowImage. |
| Occurrences are list identity | A path plus zero-based occurrence identifies a repeated source. Shared lookup resolves an exact occurrence; image fallback and missing Grid selection remain explicit caller policies. Source file versions remain separate. |
| Preserve distinct snapshots | Live rank, saved interaction, frozen image navigation, original scope and current progress remain separate values. |
| Scope | This phase does not change cache-full policy, persistence/worker phases, dependencies, platform integrations or public UI strings. Local commits are authorized; publishing is outside this implementation handoff. |

The honest limit: this change reduces caller ordering obligations; it does not
model every application mode or establish an absence of all future interleaving
bugs. Cache lifecycle simplification remains a separate follow-up.

## Tasks and acceptance

### 1. Characterize the source transition boundary

Owner: lead. Files: existing `internal/ui/visualsearch_test.go`,
`internal/ui/load_test.go` and search feature tests as needed.
Test: removals, terminal recovery and committed source writes preserve restored
Grid/image targets; display failure has one load owner; source mutation cannot
publish a pending ranking during reconciliation. Reuse existing fixtures/queues.
Verify: `go test -race -tags no_emoji ./internal/ui -run 'TestFindMoreLikeThis|TestLoad' -count=1`.
Record red behavior before implementation and retain the existing multi-step
history/overlay/source-failure cases. New root top-level tests receive shards.

### 2. Centralize source changes and restoration

Owner: lead. Depends: 1. Files: new `internal/ui/sourcechange.go`; existing
`viewer.go`, `filework.go`, `load.go`, `explorer.go`, `memlimits.go`,
`visualsearch.go`, `visualsearch/feature.go` and applicable tests.
Contract: search detaches its original visit without presentation effects;
root applies typed source-change intent and restores only after the collection,
cohort, Grid and comparison agree. The display callback retains ownership of
retry loading. Analysis-policy retirement uses an explicitly named root entry.
Verify: `go test -race -tags no_emoji ./internal/ui ./internal/ui/visualsearch -run 'TestFindMoreLikeThis|TestVisualSearch|TestLoad|TestBatch|TestDelete|TestSave|TestExport|TestVisualSimilarityExplorer' -count=1`.

### 3. Share occurrence capture and resolution

Owner: lead. Independent of 2 until integration. Files: new
`internal/fileidentity/{occurrence.go,occurrence_test.go}`; existing
`internal/ui/grid/{grid.go,ranked.go,search.go}`, `internal/ui/visualsearch.go`,
`internal/ui/visualsearch/{feature.go,feature_test.go}`.
Contract: immutable `fileidentity.Occurrence` and `Index` capture exact list
occurrences and resolve them or report absence. Grid retains its generation-bound
index and bounded ranking updates; root preserves image fallback semantics.
Verify: `go test -race -tags no_emoji ./internal/fileidentity ./internal/ui/grid ./internal/ui/visualsearch ./internal/ui -run 'TestIndex|TestRankedVisit|TestVisualSearch|TestFindMoreLikeThis' -count=1`.

### 4. Verification and records

Owner: lead. Depends: 2, 3. Files: architecture, todos, this plan; Qodana/shard
metadata for any added tests. Preserve the user's existing AGENTS.md changes.
Use the installed Go binary directly if the host snap launcher is unavailable.
Run focused race regressions, negate new guards with temporary Go overlays,
inspect every changed Go file with GoLand including weak warnings, and run
`make verify` once using the native Linux/amd64 Docker daemon. Fix confirmed
issues inline and record unavailable checks accurately. No model/runtime/library
dependencies change; existing distribution qualification remains in
`docs/find-more-like-this/dependency-qualification.md`.

Task graph: `1 -> 2 -> 4`, `3 -> 4`.

## Routing and evidence

One read-only scout traced load-failure/deletion/settings caller obligations while
the lead inspected source recovery and designed the transition. G1: bounded
factual prompt; G2: named source/test locations checked by the lead; G3: no writes;
G4/G5: caller details outside the lead's filework focus. Shell searches located
the callers first; cross-package retry ownership needed tracing. No implementation,
design, review or fixes are delegated. No additional scouting is planned.

| Task | Spawns budget/actual | Review | Full suite |
| --- | --- | --- | --- |
| Recon | 1 / 1 | lead verified caller facts | no |
| 1-3 | 0 / 0 | lead owns implementation and review | no |
| 4 | 0 / 0 | final lead gate | once |

Baseline focused race run (before edits): root UI 48.576 s, visualsearch 1.042 s,
Grid 9.513 s; all passed. Log: `/tmp/picfetch-browsing-baseline.log`.
Native Linux/amd64 Docker platform check passed. The shell's snap Go launcher
cannot start inside confinement; `/snap/go/current/bin/go` works normally.

## Implementation and regression evidence

The root transition now detaches search before callbacks, applies the complete
source change and derived Grid updates, then restores browsing. Load-failure
restoration returns its target through display's existing retry chain. Committed
writes preserve comparison for Grid origins and close it for image origins.
Grid and image visits share exact occurrence identity; their fallback policies
remain explicit. Grid reuses its generation-bound index for bounded ranked updates.

| Guard | Observed failure | Passing implementation |
| --- | --- | --- |
| Failed ranked image, image/Grid origin | Before the transition, current source was the next file while displayed pixels belonged to the origin. A negative overlay that reintroduced `ShowImage` during failure also failed the one-request-revision assertion. | Both origin variants pass through one display request and agree on the displayed/action source. |
| Committed write during comparison, image/Grid origin | A temporary overlay of the old write/exit ordering failed image-origin comparison restoration. | Both origins return to the intended surface; image origin displays the committed pixels. |
| Detached origin with late queued publication | A negative overlay restoring through Host during detach failed the callback-free boundary assertion. | Detach owns its bookmark, retires late delivery, cannot restore twice, and permits a fresh search. |
| Exact repeated-path identity | Empty API implementation failed occurrence and snapshot tests. An overlay resolving ordinal zero failed repeated-path and reconciliation assertions. | Exact occurrences survive reorder; absent occurrences do not silently select a different one. |

The committed-write fixture performs a real disk write and mirrors production's
cache invalidation at commit. It also settles comparison work started by refresh.
The first fixture omitted those two barriers; its stale-pixel/harness failures
were corrected before using it to validate the old-order negative overlay.

Evidence logs (local, temporary):

- `/tmp/picfetch-browsing-red.log`, `/tmp/picfetch-identity-red.log`.
- `/tmp/picfetch-browsing-negative-{retry,write,detach,identity}.log`: all four
  deliberate violations failed for the specified behavioral reason. Go overlays
  in `/tmp/picfetch-browsing-overlays/` left production files intact.
- `/tmp/picfetch-browsing-focused.log`: final focused race regression command
  below passed root UI (315.307 s), visualsearch (1.034 s), Grid (9.615 s),
  fileidentity (1.010 s).

```sh
/snap/go/current/bin/go test -race -tags no_emoji -count=1 \
  ./internal/ui ./internal/ui/visualsearch ./internal/ui/grid ./internal/fileidentity \
  -run 'TestIndex|TestRankedVisit|TestFindMoreLikeThis|TestVisualSearch|TestLoad|TestBatch|TestDelete|TestSave|TestExport|TestVisualSimilarityExplorer|TestRemoveFile|TestGeneration|TestMaxFile|TestDuplicateDistance'
```

GoLand inspected all 15 changed Go files with `errorsOnly:false`; final results
are empty, including weak warnings. Two new-package inspections initially timed
out and were repeated successfully. The transition's missing explicit enum
switch cases were corrected before the final inspection.

No root top-level runnable was added; the new scenarios are subtests of
`TestFindMoreLikeThisSourceAndSortRetirement`. Shard validation confirms all
686 root runnables are assigned. The new identity test file has an exact Qodana
duplication exclusion. Architecture and open-work records are updated.

Full gate: `env PATH=/snap/go/current/bin:$PATH make verify` passed (exit 0).
Formatting, generated assets/TUF checks, Qodana exclusion validation, vet and
build passed; the native Linux/amd64 Docker race run passed all 68 non-root-UI
packages and all three root UI shards. Log: `/tmp/picfetch-browsing-verify.log`.
Structured artifacts: `.scratch/race-runs/20260915T084833Z-FHNGj1/`.

Final lead review checked source mutation/restoration order, display retry
ownership, comparison callbacks, occurrence fallback and generation-bound
ranked lookup against the passing regression commands. `git diff --check` is
clean. No implementation/review/fix delegation was added; one full suite was
run, within the declared budget. The existing user edit in AGENTS.md is excluded
from the local commit. Cache lifecycle/policy follow-up remains in `todos.md`.
