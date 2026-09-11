# PR 18 review: limits, cache identity and evidence

Deliverable: close validated PR 18 findings with regression evidence and finish
with fresh clean Codex/security reviews and passing CI on the latest commit.
Route: Deep, because the independent findings span three packages, native trial
evidence, vendored code inspections and archive extraction. Lead owns all review
and fixes. Existing user edits to AGENTS.md stay outside commits. No merge/release.

## Decisions and acceptance criteria

- Keep worker request limits immutable; retire affected live UI analysis when the
  limit changes. Cached sources must meet the same cap as freshly decoded sources.
- Favorite membership and version checks must use one directory object.
- Trial image observations describe successfully applied images, including after
  a completed map. Evaluation timings include failed probes; methodology names
  the actual pinned HDBSCAN implementation.
- Validate scanner reports against code before changing or dismissing them.
- Local focused regressions and build checks; GitHub runs the complete race suite.

AC1: Oversized cached sources are rejected; moved favorite admission stays bound
to its opened directory. Verify: focused `go test ./internal/similarity` cache
guards and the actual-worker cache/limit regression under `explorertrial`.

AC2: Failed-map Shift+S retries, live size-limit changes retire analysis, and
completed-map image loads emit accurate trial observations. Verify:
`go test ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1` (targeted
subtests while iterating; exact new selectors recorded in evidence).

AC3: Failed probes contribute decode time and reports name the active engine.
Verify: focused `go test -tags explorertrial ./scripts/explorereval` cases and
ordinary report tests, with exact selectors recorded below.

AC4: Archive allowlisting remains safe; actionable Qodana results are fixed.
Verify: `go test ./internal/similarity ./internal/hdbscan`, fresh post-suppression
Qodana SARIF and CodeQL PR alerts.

AC5: All threads have commit-linked dispositions; fresh no-findings code review,
completed clean security review, actionable SARIF clear and required CI passing
on current PR head. Verify: `gh pr view`, GraphQL review threads/summary,
`gh pr checks`, code-scanning API and downloaded Qodana SARIF.

## Tasks and routing

1. Lead: validate/fix similarity cache and archive findings. Files: existing
   analyze/cache/assets implementation and tests. Tests cover each behavior above.
   Budget: 0 implementer spawns; focused tests; no local full suite.
2. Lead: validate/fix UI retry, settings invalidation and trial observations.
   Files: keys/settings/load/explorer and existing Explorer tests as needed.
   Budget: 0 implementer spawns; focused tests; no local full suite.
3. Lead: evaluator timing/provenance and Qodana corrections. Files: evaluator
   implementation/report/docs/tests and exact SARIF locations after inspection.
   Budget: 0 implementer spawns; focused tests; no local full suite.
4. Lead: final gate, commits/push, replies/resolution and fresh review cycles.
   Budget: fresh review rounds until clean; complete suite in GitHub CI.

Tasks 1–3 are independent; task 4 depends on them. No interfaces are delegated.

Scout delegation: locate existing evaluator failure/timing/report harnesses and
real-worker cache-limit fixtures (read-only conclusions with file:line).
G1 yes: bounded search prompt; G2 yes: returned symbols/locations checked with rg;
G3 yes: zero writes; G4 yes: broad test inventory stays outside Lead context;
G5 yes: harnesses not yet understood. Shell inventory above found candidates but
did not establish harness behavior. Rule S: semantic harness mapping needs reads;
rule W: no implementation supplied. Budget: one Scout, no review or fixes.

## Evidence and ledger

- Starting head: 7891ec30c10781fa4d34621434c5e1a35fb91272.
- Eight unresolved threads; CodeQL alert 6; Qodana reports ten items.
- Code and security reviews already running at start; no duplicate request sent.
- Scout spawns: budget 1 / actual 1. Local broad race runs: 0.
- Exact red/green commands, dispositions and final checks pending.

## Review expansion at 14:10 UTC

Latest review on 7891ec3 adds Intel macOS minimum-version admission, modal
shortcut protection, and build-selected runtime reporting. Lead keeps ownership.
AC6: Intel macOS below 13.4 (or unreadable version) cannot start setup/analysis;
supported versions remain admitted. Verify: pure boundary cases plus native
Intel CI and cross-compilation. Native execution remains CI evidence.
AC7: actual global shortcuts cannot dismiss an Explorer modal; they work again
after dismissal. Verify: targeted Explorer modal/favorite-shortcut subtests.
AC8: emitted reports identify the selected binding/native runtime pair.
Verify: report text guard and both platform version-selection cases.

Additional files: similarity platform build-tag pair/test, existing assets tests;
UI shortcuts and existing Explorer tests; evaluator report/tests; architecture.
No new dependencies or model/runtime upgrades. Standard-library sysctl reads
the Intel macOS product version. Budget remains one read-only Scout; subsequent
reviews run until the required clean round.

Initial red evidence: UI retry, live cap, image observation, real-worker cached
cap, cache directory move/recreation, evaluator failed timing and provenance all
failed for their stated reasons. Initial green: similarity/hdbscan/evaluator/MSIX
packages pass; expanded actual-worker/UI regressions pass (28.351s), focused UI
race pass (32.243s), make verify-build and shard inventory (681 runnables) pass.

## Local verification before first push

Commands use `/snap/go/current/bin/go` because `/snap/bin/go` invokes a broken
snap confinement wrapper on this host. Go cache writes require sandbox elevation.

- `go test ./internal/similarity ./internal/hdbscan ./scripts/explorereval
  ./scripts/msixstage -count=1`: all four packages pass. Cache-move and same-name
  replacement guards failed against the extracted old pathname-based logic,
  then passed with reads through the opened root. Failed evaluator probes had
  zero decode time before the fix; provenance guards caught the removed engine.
- `go test -tags explorertrial ./internal/ui -run
  '^TestVisualSimilarityExplorer(Local)?$/(configured_file_size_limit|recovery|file_size_changes|active_menu|trial_recording.*|duplicate_distance_changes|favorite_cache.*|presets_cache_backfill|semantic_tags)$'
  -count=1`: PASS, 28.351s. The cached oversized JPEG failed the new guard before
  the fix (2 successful/2 reused instead of 1 successful/1 failed/1 reused).
- `go test -race ./internal/ui -run
  '^TestVisualSimilarityExplorer$/(modal_shortcuts|favorite_shortcuts|setup_unsupported|file_size_changes|recovery|trial_recording.*)$'
  -count=1`: PASS, 47.697s. Modal, ready-assets unsupported admission, retry,
  active/completed limit changes and completed-map image guards were seen red.
- `go test ./internal/ui -run '^Test.*Shortcut.*$' -count=1`: PASS, 2.464s.
- `go test -tags microsoftstore ./internal/similarity ./internal/ui -run
  '^(TestStoreDownloadPolicy|TestVisualSimilarityExplorer)$/(setup_download_retry_cancel)?'
  -count=1`: PASS (similarity 0.004s, UI 28.740s). Both build-tag branches are
  exercised; four Qodana constant-condition warnings are false positives.
- `go test . -run '^TestTranslations_' -count=1`: PASS, 0.072s.
- `make verify-build check-test-shards-direct`: PASS, including format, TUF,
  Qodana exclusions, vet/build and 681 runnable shard assignments.
- A no-cgo darwin/amd64 package cross-build cannot link the existing
  `internal/trash.moveDarwin` cgo implementation. No native macOS success is
  inferred from this attempt. The existing Intel CI job runs the full similarity
  package and actual installer/offline inference with the native sysctl path.

CodeQL alert 6 is a validated false positive: ZIP entries must exactly match
fixed `asset.files()` names before decompression/writes. Existing hostile-name,
symlink, size and checksum tests pass. The alert was dismissed with evidence;
GitHub automatically resolved its thread. Extraction behavior is unchanged.

Qodana's ten post-suppression results: remove an unused vendored-test helper,
use a nil slice, lowercase a generic error; narrowly suppress four build-tag
constant conditions and three correctly capitalized Windows proper names.
Upstream HDBSCAN adaptation notes track its two test-only cleanups. No dependency,
native runtime or model version changed; all existing notices remain shipped.

Lead final diff assessment: all implementation changes map to the ten confirmed
Codex findings and scanner dispositions. One Scout supplied only harness locations;
all specifications, review and fixes stayed with the Lead. No broad local race run.
Fresh remote review and CI evidence will be linked on PR 18 after this push.

## First pushed round: 724a221

All ten Codex threads received commit-linked replies and were resolved; the
CodeQL thread also received a full false-positive disposition. New reviews
started automatically. CodeQL now passes with zero open PR alerts. Validation,
Windows and both macOS CI jobs pass, including Intel installer/offline inference
through the new native sysctl admission check. The full Linux race partitions
and fresh code/security reviews remain in progress at this record's update.

Fresh Qodana run 34609889640 has one post-suppression result, GoResourceLeak at
cache.go's os.OpenRoot. This is a false positive across ownership transfer:
rejected or empty favorites close immediately, admitted handles are retained in
favoriteAnalysis and closed by analysisCache.close. The new
TestAnalysisCacheClosesRoots admits two favorites shared across two source keys
and observes both roots closed. It passes, fails with "cache shutdown left a
directory root open" when the actual Close call is temporarily removed, and
passes after restoration. A narrow GoResourceLeak suppression documents that
ownership; production resource behavior is unchanged. Fresh SARIF is required
after this disposition, along with the fresh code/security/CI acceptance.

## Accepted implementation round

Head `408951fcc430821933df15ed54fb086fa272334f` meets every remote gate:

- [CI run 34610468241](https://github.com/frathe/picfetch/actions/runs/34610468241):
  all eight jobs pass, including four Linux race partitions, Windows, both macOS
  architectures and validation.
- [Qodana run 34610468165](https://github.com/frathe/picfetch/actions/runs/34610468165):
  downloaded complete post-suppression `qodana.sarif.json` contains zero results.
- [CodeQL run 34610468149](https://github.com/frathe/picfetch/actions/runs/34610468149):
  action/Go analyses and the result check pass; code-scanning API returns no
  open alerts for `refs/pull/18/merge`.
- [Fresh review summary](https://github.com/frathe/picfetch/pull/18#issuecomment-5624487413):
  code review completed at 14:39:18 UTC, security at 14:39:13 UTC on September 11,
  both on this head; the bot posted its no-findings thumbs-up at 14:39:21 UTC.
  GraphQL confirms zero unresolved threads, including older commits.
- Commits: `724a221` fixes the ten Codex findings and initial scanner results;
  `408951f` verifies and documents the remaining resource-ownership false positive.
  All eleven addressed threads have commit-linked dispositions.

The accepted implementation record is archived here. Subsequent documentation
commit acceptance is recorded in [PR 18's checks and final loop summary](https://github.com/frathe/picfetch/pull/18).
The PR remains open; no merge, release or hardware/UI acceptance is inferred.
The user's pre-existing AGENTS.md edit is preserved outside these commits.

Final ledger: one read-only Scout (budget one), zero delegated reviews/fixes,
two implementation/disposition commits, focused local tests and race guards,
zero local broad race suites. CI owns the complete suite. The first remote round
found one ownership false positive; the second completed clean. A documentation
round verifies this archival commit under the same latest-head rule.
