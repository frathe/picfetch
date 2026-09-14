# MA-026: Explorer workflow ownership

Status: complete (MA-026), 2026-09-14
Implementation: recorded in commit `28d65ef`
Verification: local checks and full native CI passed for `28d65ef`
Acceptance: Ronin authorized completion when the full CI gate passes
Authorization: `/implement MA-026`
Baseline: `ed70639`
Route: Deep (cross-package extraction)
Owner: T0; design, implementation, review and fixes stay with the lead.

## Contract and scope

The existing `internal/ui/explorer` module will own analysis, setup, cohort and
preset dialogs, their request lifecycles, both worker groups and the UI queue.
Root UI retains collection/duplicate preparation, launch and window policy, and
Grid/image navigation. No appState, shared controller or registry crosses the
interface. Existing model/assets/cache policy and user-visible behavior stay
the same. MA-027/028 and Find more like this implementation are separate work.

The approved test seams are the Explorer feature's public operations and
observable widgets/state, with the existing similarity Provider and drainable
UIQueue as the native-worker/test adapters. Root integration keeps scan, menu,
duplicate filtering and map/Grid/image round trips. This implements MA-026's
explicit requirement to exercise behavior without constructing unrelated
viewer features; no new test-seam approval is needed.

## Interface

`NewFeature(WorkflowHost, Options) *Feature` composes the existing Map with workflow
ownership. Options supplies the existing client/provider, queue, preset store,
platform/setup state and settings. Options are configured on UI; each admitted
worker captures its dependencies. The feature exposes `State()` by value,
`Settings()`/`ApplySettings`, its surface for explicit composition, and controlled
configuration for the existing runtime/test adapters.

`EnsureReady(ready func()) bool` owns the first-use dialog and explicit asset
installation. It returns true when ready, otherwise invokes the captured
callback only after accepted current setup. Root's callback persists the
acknowledgment and repeats ordinary command admission.

`Open(OpenRequest)` copies the prepared sources and Favorite/cache directory
facts. `Preparing()` presents the duplicate-preparation state; the actual
duplicate request remains root-owned. `SourcesChanged`, `Close`, `Stop`,
`Wait` and `Settle` replace direct lifecycle and worker manipulation. Close
invalidates without waiting on UI; Stop also ends admission. Wait joins worker
exit off UI; Settle joins both groups, drains queued delivery and repeats,
returning whether delivery occurred so root can settle any resulting Grid work.
`SettlePresets` observes preset work without waiting for a streaming analysis.

The consumer-side Host supplies window access, input modifiers, repaint/menu
notification, cohort navigation, exit/return transitions, toast reporting and
the current presentation facts needed for trial records. Feature state keeps
frozen cohort membership; root maps those identities to collection indexes.
Neither root callers nor their tests access workflow fields or worker groups.

## Tasks and acceptance

Graph: lifecycle -> analysis -> dialogs -> root integration -> test migration
-> final gate. Analysis/dialogs share feature state, so edits run serially.

### 1. Feature lifecycle and stale delivery

Files: new `internal/ui/explorer/feature.go`, `workflow.go`, `lifecycle.go`,
`feature_test.go`; Qodana exact exclusion.
Test: opening captures sources; close invalidates a queued result; reopen
accepts a fresh result; Stop rejects new analysis and Wait observes worker exit.
Verify: `go test -race ./internal/ui/explorer -run '^TestFeature' -count=1`.
Budget: zero implementation spawns, at most two lead reviews, focused checks.

### 2. Setup, cohorts and presets

Files: move root `explorersetup.go`, `explorercohorts.go`,
`explorerpresets.go`, `explorerpresetrules.go` into the module with descriptive
filenames; adapt their calls to the feature/Host. Extend feature tests.
Test: first-use/download cancellation, frozen cohort navigation, named-cohort
save rollback and preset save/preview rollback through feature operations.
Verify: `go test -race ./internal/ui/explorer -count=1`.
Budget: zero implementation spawns, at most two lead reviews, focused checks.

### 3. Root composition and integration

Files: replace root `explorer.go` with the collection/navigation adapter; update
`viewer.go`, `features.go`, `build.go`, `harness_test.go`, `run.go`, `drop.go`,
`sort.go`, `launchoptions.go`, `autoupdate.go`, `memlimits.go`, `load.go`,
`visibility.go`, `menu.go`, `windowmenu.go`, `actionmenu.go`, `keys.go` and their
Explorer integration fixtures. Keep overlay and construction order explicit.
Test: existing map/Grid/image round trips, duplicate preparation, committed
file changes, trial launch/records and shutdown continue to pass. Runtime
settings and setup acknowledgment persist through the existing preference path.
Verify: `go test -race ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1`.
Budget: zero implementation spawns, at most two lead reviews, focused checks.

### 4. Move behavior tests and finish

Move representative setup, stale-result and editing tests to the module seam;
retain root coverage of cross-feature transitions. Refresh exact Qodana paths
and UI shard counts only where the top-level inventory changes. Update
ARCHITECTURE, the existing MA-026 backlog record, todos and this evidence.
Verify: module/root focused race commands above; native
`make explorer-ui-test`; `make verify`; GoLand inspection of all changed code
including weak warnings. Negatively verify guards with temporary overlays.
If the Docker daemon remains ARM, report the native Linux/amd64 gate unverified
and run the available build/native checks without weakening worker isolation.
No commits unless separately authorized.

## Recon and cost ledger

The lead retains the previous Explorer ownership sweep. One bounded read-only
Scout task inventories representative existing tests and helper dependencies;
it makes no interface or review decisions. This survives amnesia, has exact
file/line outputs and does not duplicate the lead's current implementation work.
No new agent is spawned. Recon budget: at most two Scout tasks this phase.

| Work | Spawns budget/actual | Lead review rounds | Full suite |
| --- | --- | --- | --- |
| Recon | 0 / 0 (one reused Scout task) | n/a | no |
| Feature lifecycle/analysis | 0 / 0 | 2 | no |
| Dialogs | 0 / 0 | 2 | no |
| Root integration/test migration | 0 / 0 | 2 | no |
| Final gate | 0 / 0 | 2 shared lead rounds | one attempt; platform preflight refused ARM daemon |

## Evidence

### TDD and review fixes

| Behavior | Observed red | Green implementation |
| --- | --- | --- |
| Prepared-source admission | `TestFeatureOpenCopiesSourcesAndRejectsClosedDelivery` reported that opening did not admit analysis against a compilable stub. | Feature captures sources and owns analysis/token/UI delivery. |
| First-use acceptance | `TestFeatureSetupRequiresAcceptance` reported skipped acceptance against a compilable readiness stub. | Feature owns the dialog and releases its captured continuation only after acceptance. |
| Preset editing | `TestFeaturePresetSavePreviewAndApply` reported a missing New preset action. | Real feature-owned browser/editor/store/preview/application pass. |
| Source reconciliation | Root `source_changes/remove_after_reorder` panicked in Grid thumbnail lookup when menu notification also repainted. | Separate `Changed` and `Repaint`; retain repaint at original delivery boundaries, after collection indexes are valid. |
| Completed-map reuse with duplicate hiding | Root `duplicate_representatives` reported two analyses when returning from a cohort. | Present Preparing only when duplicate preparation is actually pending; retain the completed map when facts are ready. |
| Source replacement and preset delivery | `TestFeatureSourceReplacementRetiresPresetDialog` retained the old dialog after a replacement Open. | New-source Open retires setup/analysis/preset lifecycles before admitting replacement work; completed same-source reuse remains intact. |

Moved representative setup cases out of the root test and added independent
feature tests for admission, stale results, stop/reopen, frozen cohort paths,
and actual temporary Favorite/preset storage rollback. Root retains all
cross-feature and native trial scenarios. Provider and UIQueue remain the
native/UI adapters; filesystem and editing behavior are real in these fixtures.

Twelve deliberate violations were run through temporary Go overlays, without
editing working source. Each test failed for the intended reason, including
the strengthened assertion that reopened map contents retain the new paths:

| Deliberate violation | Guard and observed failure |
| --- | --- |
| Share Open's caller slice | `TestFeatureOpenCopiesSourcesAndRejectsClosedDelivery`: source snapshot changed. |
| Deliver an old queued map | `TestFeatureCloseReopenAndStop`: reopened map contents replaced. |
| Admit Open after Stop | `TestFeatureCloseReopenAndStop/stop`: wrong admission. |
| Omit cancellation at Close | `TestFeatureCloseReopenAndStop/close`: timed out in Wait on the uncancelled provider. |
| Share captured cohort input | `TestFeatureCohortRetainsCapturedMembership`: membership changed. |
| Return mutable cohort storage | Same guard: membership changed. |
| Skip Favorite cohort rollback | `TestFeatureFavoriteSaveRollback/cohort`: proposed memberships retained. |
| Skip Favorite preset rollback | `TestFeatureFavoriteSaveRollback/preset`: proposed memberships retained. |
| Admit unsupported setup | `TestFeatureSetupUnsupported`: continuation offered. |
| Omit download cancellation | `TestFeatureSetupDownloadRetryCancel`: timed out joining the uncancelled HTTP request. |
| Implicitly acknowledge setup | `TestFeatureSetupRequiresAcceptance`: showing setup acknowledged it. |
| Bypass actual preset application | `TestFeaturePresetSavePreviewAndApply`: no named cohort created. |

Each overlay ran its named guard with `go test ./internal/ui/explorer -overlay=<temporary-json>
-run '<guard>' -count=1 -timeout=3s`. Compiler failures were rejected as evidence.
The ordinary module race suite passed again after removing the overlays and
after the final source-replacement fix.

### Final checks

- `go test -race ./internal/ui/explorer -count=1`: passed, 4.258s on final code.
- `go test -race ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1`:
  passed. The expanded root race run also covered launch options, memory limits,
  Actions/Window menus and settings, passing in 123.844s.
- After the final replacement fix, focused root race checks for source changes,
  pending duplicate preparation, completed-map reuse, streaming presets and
  trial cancellation passed in 10.905s.
- `make explorer-ui-test`: passed on final code, 32.033s. The real local-model
  worker tests passed in 23.60s; the command also reran the ordinary Explorer
  suite with native trial wiring. Expected recovery/rollback errors were logged
  by their intentional failure fixtures.
- `make verify-build`: passed on final code, including format/generated/TUF/
  Qodana-exclusion checks, vet and build. No dependency or model changes.
- `make check-test-shards`: passed, 682 root runnables across three shards.
  No root top-level tests were added; the shard manifest remains unchanged.
  All three new module test paths have exact Qodana exclusions.
- GoLand inspected all 31 changed Go files with `errorsOnly:false`, then
  re-inspected files changed by fixes. No remaining confirmed issues.
  Four existing duplicate-fragment warnings in `explorer_local_test.go` and
  `menu_test.go` are intentional fixture repetition, already covered by the
  exact test-file exclusions in `qodana.yaml`. Narrow source suppressions cover
  Map Fit's bounds calculation (distinct from selection-aware centering) and
  settings' deliberate grouping by purpose instead of saving 16 bytes per viewer.
  Their re-inspections are clean, including weak warnings.
- `git diff --check`: passed. Root no longer accesses Explorer workflow fields,
  worker groups or private tokens. The moved dialog/field code preserves strings
  and storage operations; only feature/Host adaptation and ownership changed.
- `make verify`: attempted once; the platform check refused the local
  `linux/aarch64` Docker daemon. The complete native Linux/amd64 race suite
  remains unverified. Worker isolation was not weakened or skipped. Other-platform
  runtime qualification is not claimed by the native macOS checks.

The lead performed standards and spec review against `ed70639` plus all new
files, following the repository's root-owned review rule. Standards: no remaining
confirmed findings after the documented mitigations. Spec: MA-026 ownership and
independent feature testing are implemented; complete qualification remains open.
The implementation phase performed no commits, pushes or release actions. The separate CONTEXT.md
edit present during review was left intact and is outside this refactoring.
The passing CI evidence below satisfies the accepted completion condition; this
plan is archived in `finished_refactorings/`.

## PR #24 qualification and review

Ronin authorized pushing the implementation and completing MA-026 after CI,
then invoked the GitHub Codex fix/review loop. This authorizes focused fixes,
fix commits, pushes and review replies under AGENTS.md; no merge or release.
The existing committed branch was pushed at `28d65ef` and
[draft PR #24](https://github.com/frathe/picfetch/pull/24) opened because this
repository's CI runs on PRs or main pushes, not ordinary branch pushes.

The full [native CI run](https://github.com/frathe/picfetch/actions/runs/34829485376)
passed all eight jobs for `28d65ef`: validation, all four Linux/amd64 race
partitions, Windows guards, and macOS ARM64/amd64 guards. This supplies the
complete qualification that the local ARM64 Docker daemon could not provide.
[Qodana](https://github.com/frathe/picfetch/actions/runs/34829485254) passed;
the downloaded `qodana-report` artifact's post-suppression `qodana.sarif.json`
contains zero results. [CodeQL](https://github.com/frathe/picfetch/actions/runs/34829485279)
passed and reports no open alerts for `refs/pull/24/merge`. Codex security review
completed with no findings for `28d65ef`. Code review raised two FML-001 findings,
outside the Explorer extraction: the report discarded existing relevance labels
outside the displayed results, and the canonical specification's delivery status
was stale. Both are confirmed and corrected in the PR follow-up.

The export regression executes the shipped inline JavaScript with synthetic
browser objects using the existing Node runtime (local v26.8.2; CI logs its
runner version). No npm packages or application runtime dependencies were added;
Node and its libraries are not shipped in PicFetch. The first test reproduced
the loss of `s-31` and `s-32`, then passed after retaining undisplayed IDs while
applying edits to displayed inputs. Guards for unreviewed/zero-match and failed
references were also negatively verified with temporary copies. The three tests
run in CI validation. Focused Go evaluator race tests passed in 9.629s; GoLand
inspected the HTML, JavaScript test and workflow with no findings.

MA-026 is complete. The broader PR follow-up still requires a fresh clean Codex
code/security review and passing CI for its latest commit, as tracked by
[PR #24](https://github.com/frathe/picfetch/pull/24) and the
[FML implementation evidence](../docs/find-more-like-this/implementation.md).
The lead owns all assessment and fixes. The unrelated Spiral specification
wording already edited in `todos.md` stays outside any review-fix commit.
