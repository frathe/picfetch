# Find more like this — implementation evidence

Status: FML-001 tooling implemented; real relevance decision pending
Started: 2026-09-14
Baseline: e6c767e
Authorization: `/implement use tdd and sdd` approves the proposed breakdown and
its pre-agreed test seams. Commit authorization follows the project guide.

## Route and seams

Deep SDD; T0 owns design, implementation, review, fixes and the final gate.
FML-001 tests the evaluation command and its local evidence output. Only native
inference and OS isolation/RSS observations are substituted in ordinary fixtures;
source reads, canonical decoding, ranking and artifact generation remain real.
The tagged native test drives the actual command and network-denied subprocess.
This applies the accepted evaluator seam; no new user approval is needed for it.

The twelve MVP tickets are approved, subject to blockers. The first search UI
still requires a recorded real-image quality decision. MA-026 is complete with
full native Linux/amd64 CI qualification. FML-009 remains
deferred. Existing demo/model assets were reused for the 446-image local
experiment. The [evaluation record](evaluation.md) links its interactive report
and measured evidence. Twenty content judgments and a human proceed/revise
decision remain required; neither timing nor synthetic tests supplies them.

## TDD record

| Cycle | Observed red | Observed green |
| --- | --- | --- |
| Reference corpus admission | `TestSearchEvaluationRequiresReferenceCorpus` rejected the unknown search flag rather than the missing references. | The same test passed after command admission and minimum-reference validation. |
| Ambiguous corpus | Duplicate identities/paths/queries, invalid paths/intents and unknown/self/duplicate relevance reached asset setup or the wrong validation. The version case was tightened to require a corpus error, avoiding a match in its temporary directory name. | `go test ./scripts/explorereval -run '^TestSearchEvaluation' -count=1` passed these cases. |
| Local ranking report | `TestSearchEvaluationReportsRankingAndPendingJudgments` failed at the explicit unimplemented evaluation boundary. | The same test passed with actual fixture decoding, exact ranking, emitted artifacts, native-boundary cleanup and distinct pending judgments. |
| Native command | The tagged command completed the old smoke path without `search-result.json`. | `TestProductionSearchEvaluation` exercised the real encoder in a network-denied subprocess and emitted exact rankings. |
| Initial evidence | Missing source digest and zero first-partial measurement failed the report test. | Captured identities and reference-first preparation now produce `search-initial.json` at 100 processed sources or final flush. |
| Favorite reuse | The tagged command lacked `favorite-profile.json`. | Separate cold/warm production workers prove zero warm inference and clean up the temporary Favorite cache. |
| Large exact ranking | The report lacked its 10,000-vector benchmark fields. | Thirty timed queries after warm-up report separate storage and p50/p95 values. |
| Corpus bounds | A misspelled relevance key silently became unreviewed; subsequent oversized-manifest and vector-budget cases reached the wrong validation. | Strict field decoding, a 16 MiB manifest cap and 256 MiB vector-count admission passed. |
| Judgment controls | The generated report lacked review/download controls and its network policy. | The template provides per-reference review state, relevance selections, P@10 and offline JSON export. |
| Trailing input | A second JSON value was silently ignored. | Manifest parsing now requires EOF after its one object. |
| Changed prepared reference | Changing the first file while the second encoded still published 21 successes and ranked the old reference. | Publication revalidation reports 20 successes/one failure and no precision for the invalid reference. |
| Frozen baseline scope | Editing the source manifest after the worker captured it failed the Favorite baseline with `search corpus version must be 1`. | The baseline reads the retained manifest, and the native command passes after the same edit. The fixture writer deliberately avoids promoted `bytes.Buffer.ReadFrom`, which would otherwise bypass its mutation callback. |

Native first attempt: the agent sandbox prevented nested `sandbox-exec`
(`sandbox_apply: Operation not permitted`). This is infrastructure rejection,
not the intended red test. The retry runs outside the agent sandbox and retains
the application's own network denial; it uses synthetic sources and installed
assets. No model or source image is uploaded or downloaded.

Cancellation and sparse/invalid-source guards were introduced against existing
behavior, then negatively verified with temporary Go overlays. Dropping the
request context let all 21 encodes complete and failed the cancellation guard;
admitting malformed vectors failed with NaN serialization; dividing precision
by available matches failed missing-slot accounting. No mutation was applied to
the working source. Removing only one cancellation check survived because the
canonical decoder independently honors cancellation; the stronger mutation
verified the observable guard. All final guards passed with the real source.

Sparse fixtures cover an unreadable file, failed encodes, wrong vector length,
zero/NaN/infinite vectors, a distinct identical copy, and an eligible negative
cosine match. These checks do not claim real hard-negative relevance labels.

## Lead review and verification

The code-review skill's standards and spec axes were performed by T0, as the
project working agreement requires. Baseline `e6c767e` plus all uncommitted/new
files; no commits were made. Two lead review rounds, zero implementation spawns.

Standards: no remaining confirmed code findings. Shared fixture execution
replaced duplicated setup. GoLand inspected every changed Go/HTML/shell file
with `errorsOnly:false`; final results are empty, including weak warnings.
Template JSON now travels through escaped HTML text, avoiding raw template
syntax inside JavaScript. Narrow `HtmlUnknownTarget` suppressions cover only
generated sibling evidence and preview links; their files are checked by the
command/fixture output. Both new test files have exact Qodana exclusions.

Spec: technical FML-001 output is implemented. The required real relevance
judgments and proceed/revise verdict remain open. Production search, general
cache and other-platform qualification are not claimed by this experiment.

Observed checks:

- `make verify-build`: passed formatting, embedded/generated checks, Qodana
  exclusions, vet and build. It required access to the existing Go cache outside
  the agent sandbox; no check was weakened.
- `go test -race ./scripts/explorereval -count=1`: passed after allowing the
  existing worker test to create its own macOS sandbox.
- `go test -tags=explorertrial ./scripts/explorereval -run
  '^(TestProductionSearchEvaluation|TestProductionProfile)$' -count=1`: passed
  with installed assets and actual production workers.
- The real 446-image `TRIAL=search` command exited zero; all measurements and
  their limits are in [evaluation.md](evaluation.md).
- Final GoLand inspections, `make fmt-check` and `git diff --check`: passed.
- Generated synthetic HTML: JavaScript syntax and offline control execution
  passed (pending versus reviewed-zero, live P@10, export and source retention).
  The Browser runtime reported no available browser; visual layout inspection
  remains unverified. Private image previews were not remotely inspected.
- `make check-test-platform`: the daemon reports `linux/aarch64`. The complete
  race gate requires native Linux/amd64 and remains unverified; isolation tests
  were not skipped or relaxed to run under emulation.

## MA-026 prerequisite recon

The source sweep confirmed analysis/setup/cohort work uses one tracked worker
group, presets another, and both share a drainable queue. Move that ownership,
their request lifecycles and their dialogs into `internal/ui/explorer` together.
Root retains collection and duplicate preparation, Grid/image transitions,
launch admission and window policy. Frozen cohort membership and map camera
must remain usable across those transitions. The feature's completion contract
must join both worker groups and drain/repeat; shutdown must end admission
before joining off UI. Keep module behavior tests separate from the retained
root map/Grid/image integration tests. That extraction is complete in
[the archived MA-026 plan](../../finished_refactorings/2026-09-14-explorer-feature.md),
with focused race, native worker, build, GoLand and full native Linux/amd64 CI
evidence. Its [existing tracker anchor](../../needs_refactoring.md#ma-026) remains
available for dependency links.

## Open delivery work

- [x] Complete command dispatch and tagged native search evidence.
- [x] Pin failure/cancellation, malformed-vector, sparse-result and corpus-bound guards.
- [x] Complete first-partial, warm-query, 10,000-vector and Favorite-reuse measurements.
- [x] Finish the local relevance-review controls and judgment export.
- [ ] Collect the required real-image judgments and confirm semantic case coverage.
- [ ] Record FML-001's proceed/revise decision before FML-002.
- [x] Complete the separately tracked MA-026 prerequisite and full native CI qualification.
- [ ] Continue the approved ticket frontier through final qualification.

## Cost and verification

Two bounded read-only tasks reuse the existing Scout: locate approved local
corpus/assets/relevance evidence, then inventory Explorer ownership for MA-026.
No new agent is spawned. Both tasks are factual recon; root retains all design
and review. This is within the two-Scout-task recon budget.

The evaluator adds no dependency, native runtime or model asset. Existing
distribution obligations remain for final feature qualification. Full native
Linux/amd64 verification passed in [PR #24's CI](https://github.com/frathe/picfetch/actions/runs/34829485376)
for `28d65ef`. Human semantic qualification and the remaining tickets are not
complete. Update this record as each cycle closes.

## PR #24 review fixes

Codex's review of `28d65ef` identified two confirmed findings. The download
handler now retains existing relevance IDs without rendered checkboxes while
applying checkbox edits to displayed matches. The canonical specification now
states that the technical evaluator and 446-image experiment are implemented,
with relevance judgments and the proceed/revise decision still pending.

`node --test scripts/explorereval/search_review_test.mjs` executes the shipped
inline handler with a synthetic browser boundary. Its first red lost `s-31`
and `s-32`; all three cases now pass, covering repeat exports, displayed edits,
unreviewed versus reviewed-zero, failed references and unchanged source records.
The latter two guards were also negatively verified with temporary copies.
CI validation runs these tests with the runner's existing Node runtime, without
adding npm or application dependencies. Focused evaluator race tests passed:
`go test -race ./scripts/explorereval -run '^TestSearchEvaluation' -count=1`.
GoLand inspections of the changed HTML, JavaScript and workflow report no issues.

The GitHub review loop requires a fresh clean code/security review and passing
CI for the latest pushed fix; [PR #24](https://github.com/frathe/picfetch/pull/24)
records the review dispositions and current results.
