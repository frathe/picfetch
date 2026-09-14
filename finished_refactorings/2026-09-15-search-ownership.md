# PR 25: cache ownership and browsing transitions

Status: implementation and local verification complete. Final remote acceptance
is recorded on [PR 25](https://github.com/frathe/picfetch/pull/25) for its latest commit. Route: Deep (cross-package
refactoring). Evidence: [all 28 Codex findings](../docs/find-more-like-this/pr25-architecture-review.md).

Deliverable: put Favorite ownership, cache operation lifetime, and ranked browsing
decisions behind their owning modules while preserving the existing fixes.

## Decisions and limits

| Decision | Contract |
| --- | --- |
| One Favorite inventory | Retain directory handles and captured list versions; return healthy entries alongside errors. Unknown membership forbids general fallback, but remains inspectable/clearable. |
| Store owns routing | An internal write scope selects Favorite-only (Explorer) or all enabled stores (search); both consumers use store read/write methods. No dependency or wire-policy change. |
| Maintenance owns intent | Inspection is replaceable; eviction and policy retirement survive Settings close; cleanup and explicit retuning belong to their view. Coalesce automatic reserve requests. |
| Outcomes describe effects | Keep committed removals, cancellation, per-store observation completeness, and applied limit independent. A closed view cannot persist a late explicit retune. |
| Root owns browsing | Preserve separate live ranking, frozen image order, and saved Grid visits. Share per-operation context/capabilities at root; no global controller or event bus. |
| Existing safety boundaries | Preserve leases/epochs, staging-byte budgets, unavailable-source retention, source versions, cancellation/barrier joins and bounded transport/ranking. |

No new decoder, dependency, cache database, license resolution, release, or merge.
This refactor cannot guarantee absence of bugs or qualify outstanding shipped
dependency notices. Existing release qualification remains open.

## Tasks and acceptance

### 1. Favorite ownership and producer routing

Owner: T0 inline. Files: `internal/similarity/cache*.go`, `analyze.go`,
`search_worker.go` and existing cache tests.
Contract: `openFavoriteInventory(ctx, dir)` owns enumeration and root cleanup;
`openRepresentationStore(ctx, policy, scope)` owns every read/write decision.
Test: healthy/broken peers, empty/unknown membership, opt-outs, promotion,
Favorite-only misses, version changes, linear accounting and staging pressure.
Verify: `GOCACHE=/private/tmp/picfetch-h265-audit-buildcache go test -race -tags no_emoji -count=1 ./internal/similarity -run 'TestAnalysisCache|TestSearch'`.
Budget: no implementation spawns; one lead review plus necessary fixes.

### 2. Maintenance intent and transaction outcome

Owner: T0 inline. Depends: 1. Files: similarity maintenance and analysiscache UI
feature/work files, existing lifecycle/policy/feature tests.
Contract: explicit private operation intents own admission, lifetime and provider
dispatch; a maintenance transaction owns observations, committed removal accounting
and final report. Preserve public report/provider compatibility where sufficient.
Test: cancellations before/after lock, quiescence, removals and reconciliation;
close/reopen while cleanup, retune, policy retirement or eviction runs; unrelated
Favorite failures must not undo committed general policy.
Verify: `GOCACHE=/private/tmp/picfetch-h265-audit-buildcache go test -race -tags no_emoji -count=1 ./internal/similarity ./internal/ui/analysiscache -run TestAnalysisCache`.
Budget: no spawns; one lead review plus necessary fixes.

### 3. Browsing context and transitions

Owner: T0 inline. Independent of 1/2 until final gate. Files: root visualsearch,
navigation/action/menu adapters and existing root/feature/Grid tests as needed.
Contract: root captures current browsing order/capabilities once per operation;
search enter/back/exit/source replacement preserve complete surface state and
independent live progress. Search feature retains query/history ownership.
Test: cohort -> partial search -> opened image -> final publication -> overlay
dismissal -> Back -> Exit; duplicate identity, action targets and preload neighbors.
Verify: `GOCACHE=/private/tmp/picfetch-h265-audit-buildcache go test -race -tags no_emoji -count=1 ./internal/ui ./internal/ui/visualsearch ./internal/ui/grid ./internal/ui/menus -run 'TestFindMoreLikeThis|TestVisualSearch|TestRankedVisit|TestApply|TestActionsMenu|TestMosaicSources'`.
Budget: one read-only scout for command/menu/source ownership; implementation and
review remain lead-owned. Scout gate: short factual prompt, file/line oracle, zero
edits, independent cold consumer context, shell located but did not trace callers.

### 4. Verification and GitHub review loop

Owner: T0. Depends: 1/2/3. Files: architecture, todos, this evidence record,
Qodana exclusions/UI shards only if tests added. Run focused regressions and
negative verification of new guards, `make verify-build`, `make check-test-shards`,
and GoLand inspections including weak warnings on changed code. Push authorized
fix commits; inspect full native CI, post-suppression Qodana, CodeQL, security and
a fresh clean Codex code review of the final commit. No broad local race duplicate.

Task graph: `1 -> 2 -> 4`, `3 -> 4`.

## Evidence and ledger

Budget: one scout, zero delegated implementations/reviews; focused local suites
and complete native GitHub CI. Actual: one read-only command ownership scout
completed; all design, implementation and review lead-owned. Local verification results are recorded below; remote results are linked from PR 25.

### Local implementation evidence

- Baseline focused cache/maintenance/search tests passed.
- Producer-scope matrix failed before enforcing Favorite-only write scope, then
  passed with all enabled/disabled combinations. Shared inventory preserves
  healthy peers beside damaged definitions; failed leases permit reads but deny writes.
- Inspection/mutation guard failed for cleanup and explicit retune under the old
  generic cancellation rule; explicit intent scheduling makes both pass.
- Direct ranked duplicate command and repeated preload snapshot guards failed
  before the shared browsing restriction/order changes, then passed.
- Combined cohort/search/opened-image/final/overlay/Back/Exit regression passes;
  existing identity, source replacement, action capture and live-progress tests remain.
- Public cache reports remain compatible; an internal transaction now owns every
  observation, committed removal and final cancellation/applied-limit outcome.
- One scout completed factual command/source mapping. No delegated fixes or review.

### Local gate results

- Cache/search/maintenance/Grid/menu focused race suites passed (similarity 9.5 s;
  analysiscache 2.4 s; visualsearch 1.7 s; Grid 3.0 s; menus 2.3 s).
- Expanded root regressions including Explorer passed in 129.3 s. After tightening
  the shared keyboard command entry, Find-more-like-this/Actions passed again in
  24.4 s. The keyboard regression was observed failing before that final change.
- Real production worker test `TestVisualSimilarityExplorerLocal/new_search_favorite_reuses_general_analysis`
  passed: both records promoted/reused; Favorite-only reopen made zero inferences.
- Negative overlays removed the lease guard and restored Back beneath a modal;
  each dedicated regression failed for its intended effect. Repository files
  were never replaced by those deliberately broken overlay variants.
- `make verify-build` passed; `make check-test-shards` passed for 686 runnable
  root tests over three shards; Qodana test exclusions remain complete. No new
  root test function or test file was needed (new cases extend existing files).
- All 28 changed Go files inspected in GoLand, including weak warnings. Struct
  packing warnings fixed. One narrowly scoped `GoDfaErrorMayBeNotNil` suppression
  documents the inventory's always-owned partial-result contract; reinspection clear.
- Restricted tool-sandbox attempts could not bind an unrelated update-test server,
  access Docker or launch nested macOS sandbox workers. Appropriate native checks
  were rerun with those host capabilities, without weakening worker policy.
- No new dependency or shipped closure change. Full Linux race/native Windows and
  Intel/ARM macOS CI, Qodana/CodeQL and fresh bot reviews remain the remote gate.

- Final timing guard exposed stale progress when preparation completed after Back
  was deferred behind a popup. Presentation now carries only the frozen visit and
  reads live progress on application; the new regression was observed failing first.
