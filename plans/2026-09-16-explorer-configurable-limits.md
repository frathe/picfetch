# Configurable Similarity Explorer limits (PR #29)

Route: Standard feature extension; cross-package wiring and translation edits
justify exceeding the usual eight-file guideline without a subsystem redesign.

Deliverable: Settings -> Limits -> Similarity Explorer exposes a 512 MB memory
limit and 10,000-item limit; exceeding either reports an actionable error toast.

## Decisions and scope

- Configure the PR's existing per-result serialized memory bound. This is not
  a total process/RSS guarantee; explain the scope in the memory field hint.
  A clarification was offered while independent reconnaissance proceeded.
- Use binary MB consistently with existing memory settings. Accept positive
  integers, cap memory input at the existing 1 TB settings ceiling, normalize
  unset/invalid limits to defaults, and capture changes for the next analysis.
- Keep item admission in both client and worker. Preserve typed resource errors
  after terminating/joining a worker so UI can identify either limit reliably.
- Keep cancellation/stale-result suppression and existing worker isolation.
- No new dependencies, model assets or package moves.
- Ronin additionally requested fixes for P1/P2: allow the qualified 50,655-item
  scale through configurable admission and replace single-message snapshots
  with chunked delivery. Both peers enforce the same aggregate snapshot budget;
  the UI receives only complete snapshots. An explicitly configured aggregate
  budget can still reject a map, with an actionable limit error.
- Ronin authorized committing, pushing to the existing PR branch and checking
  GitHub CI after local completion. No merge or release is requested.

## Acceptance criteria and existing seams

1. Settings contains the two editable fields under a Similarity Explorer
   heading within Limits; invalid edits do not replace accepted values.
   `go test -tags no_emoji,nodynamic ./internal/ui/settingswin`
2. Values round-trip through preferences and root startup/live/shutdown wiring;
   fresh settings show 512 MB and 10000 items.
   `go test -tags no_emoji,nodynamic ./internal/preferences ./internal/ui -run 'Test(SimilarityLimitPreferences|VisualSimilarityExplorer)$'`
3. Client/worker item validation uses captured limits, and bounded decoding
   retains its memory-limit error through subprocess cleanup.
   `go test -tags no_emoji,nodynamic ./internal/similarity`
4. Current limit failures show a toast with the settings path; stale failures
   do not show a toast and limit failures do not invalidate model availability.
   `go test -tags no_emoji,nodynamic ./internal/ui/explorer ./internal/ui -run 'Test(Feature.*Limit|VisualSimilarityExplorer)$'`
5. Translation parity, format, vet/build, GoLand changed-file inspections and
   the native Linux/amd64 Docker verification gate are checked before handoff.
   `make verify`; report unavailable gates as unverified.

## Tasks

### 1. Persistence and settings
Owner: T0 inline. Files: preferences{,_test}.go, settingswin{,_test}.go,
translations/{en,de}.json. Contract: SimilarityMemoryLimitMB and
SimilarityItemLimit in preferences.State; existing positive numeric controls.
Test: saved limits survive invalid/zero snapshots; fields belong to Limits.
Verify: criterion 1 and preference part of criterion 2.
Budget: 0 spawns, 1 review round, no full suite.

### 2. Analysis admission, captured configuration and UI errors
Owner: T0 inline. Files: similarity/{client,analysis_protocol,analyze_test}.go,
explorer/{feature,workflow,feature_test}.go, ui/{features,memlimits,run,explorer_test}.go.
Depends: task 1. Contract: similarity.AnalysisLimits with MemoryMB and Items,
normalization and typed errors for item and memory limits; chunked snapshot
metadata/items/merges/end with atomic delivery and aggregate accounting.
Test: default/custom boundaries, bounded streaming, cleanup error identity,
settings wiring, toast delivery and stale-result suppression.
Verify: criteria 2-4. Budget: 0 spawns, 2 review rounds, no full suite.

### 3. Final verification and evidence
Owner: T0 inline. Files: this plan and todos.md; toast layout only if required
to keep the settings guidance readable. Depends: tasks 1-2.
Verify: criterion 5 and diff against all acceptance criteria; commit/push and
wait for the latest GitHub CI after local completion.
Budget: 1 final review, one full gate attempt.

Task graph: settings reconnaissance (independent) || client reconnaissance;
task 1 -> task 2 -> task 3. All implementation/review stays with the lead.

## Delegation and cost

One read-only settings scout: G1 bounded question, G2 file/line evidence checked
with rg/sed, G3 zero edited files, G4 narrow settings context, G5 lead inspected
client transport instead. No mechanical edit or review delegated. Higher-level
session instructions enable independent reconnaissance alongside lead work.
One follow-up render-only task reused that agent: G1 two named PNGs, G2 one
focused test and artifact existence, G3 one temporary test file, G4 isolated
harness setup, G5 lead inspected production code instead. Lead inspected both
images. Temporary test removed; no delegated review or permanent edits.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Recon | 1/1 | lead evidence check | no |
| 1 | 0/0 | 1 | no |
| 2 | 0/0 | 2 | no |
| 3 | 1/1 render-only follow-up | 2 (including CI golden follow-up) | local platform gate blocked; hosted CI |

## Verification evidence

- Red/green observed for default item admission, preference persistence, Settings
  tree membership, viewer wiring, error toast delivery, chunked serialization
  and narrow-window toast layout.
- Aggregate byte-accounting guard negatively verified by disabling only its
  comparison: the small boundary test failed for partial-map publication, then
  the original source was restored.
- `go test -race -tags no_emoji,nodynamic ./internal/preferences ./internal/similarity ./internal/ui/settingswin ./internal/ui/explorer` passed.
- Focused root UI race suite passed (126.253s): Explorer, startup/preferences,
  apply-settings, toast and overlay regressions.
- Translation/manual guards passed in the root and help packages.
- `make verify-build` passed: formatting, generated assets/notices, TUF,
  Qodana exclusions, vet and build.
- GoLand `get_file_problems(errorsOnly=false)` returned no findings in all 16
  changed Go files, including the new protocol implementation.
- Actual Settings and 360x400 toast renderings inspected:
  `/private/tmp/picfetch-limits-settings.png` and
  `/private/tmp/picfetch-limits-toast.png`. Both defaults and full toast guidance
  are visible. The dedicated temporary render fixture passed and was removed.
- Complete local Docker suite unavailable: `make check-test-platform` reports
  `linux/aarch64`; required native Linux/amd64 verification will run in GitHub CI.
- Final transport race regressions passed (4.474s), including 50,655-item
  subprocess delivery, configured request capture, malformed frames and typed
  memory errors. The historical native 50,655-image inference/cache trial was
  not rerun; this check covers admission and transport at that scale.
- `make check-test-shards` passed through the supported Docker inventory path:
  686 runnables, 3 shards. No new top-level root UI tests or test files were added.
- Final UI/translation/format reruns are checked before commit. Hosted CI is
  checked after push on the exact latest commit; live results:
  [PR #29 checks](https://github.com/frathe/picfetch/pull/29/checks).
- Implementation is complete. No merge or release is performed.


## Hosted verification follow-up

Commit `5b7f8d2` passed native Windows/macOS guards, Linux non-UI and UI-2 race
checks, validation, CodeQL and Qodana (final `/qodana.sarif.json`: zero results).
UI-1 and UI-3 each failed one golden: `bad_drop_after_images.png` and
`bad_drop_fresh.png`. Linux/amd64 `make golden` reproduced both mismatches;
inspection confirmed only the intended wrapped-toast card layout changed.
The two baselines were accepted from those Docker-generated images. No failed
renders are staged. The full Linux/amd64 E2E render suite passed after the
baseline update (`internal/ui`: 0.870s); no other golden changed.

GitHub's managed AI security scan failed before analysis with HTTP 400,
"The requested model is not supported." This is separate from the successful
CodeQL run and produced no security assessment. No repository security policy
was relaxed. The managed service failure remains unverified at handoff unless
GitHub repairs it during the subsequent run.
