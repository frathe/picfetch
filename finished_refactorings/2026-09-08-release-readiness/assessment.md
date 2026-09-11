# Release review since v1.0.2

Reviewed immutable range:
`73a5f6381ab8e643de47a6c2038ed1d929f11670...d306654e49cec03432070d6219cc920cac9c2b72`.
All 30 commits are inventoried in `commits.txt`. The independent coverage
records classify 539 paths and distinguish executable changes from generated
assets, documentation and historical validation output. Later fixes are assessed
below; this document does not claim that archived evidence validates a new head.

## Standards

Two P2 findings, both fixed; worst P2.

- Case aliases bypassed complete file transactions and current-source
  reconciliation. Native case-insensitive APFS tests reproduced concurrent
  Save/Strip/Export and stale reset pixels after exporting `Photo.png` to
  `photo.png`. Transaction admission now compares live filesystem identities;
  absent case aliases conservatively serialize, and unrelated existing files
  remain independent. UI reconciliation uses current file identity too.
  `internal/imaging/mutations.go`, `internal/ui/filework.go`.
- A queued timed picture-frame advance could execute immediately after manual
  navigation, before the worker consumed `Kick`. Kick now invalidates the
  countdown synchronously; queued work checks that revision before advancing.
  The regression covers held queue submission and continued playback.
  `internal/ui/slideshow/slideshow.go`.

These violated the documented transaction/reconciliation and queued-staleness
contracts. No additional material baseline-smell findings. See
`standards-coverage.md` for standards sources, reviewed areas and proof limits.

## Spec

Three P2 findings, all fixed; worst P2. No harmful scope creep established.

- The same case-alias defect violated maintainability issue 23's resolved
  identity and post-write refresh requirements.
- An active Save could recreate a source already moved to Trash, while the UI
  removed it from the list as successfully deleted. The deterministic red
  held the real encoder, completed a temporary Trash move and then released
  Save. Trash now participates in the same transaction admission. Shutdown
  cancels waiting admission; an already submitted native move finishes.
  Deleting a live or broken symlink still moves the link itself.
  `internal/ui/deletion/deletion.go`, `imaging.WithFileMutation`.
- Cancelling/resetting/replacing startup scan or capture-date sort left
  `--slideshow` armed for an unrelated later drop, violating launch AC6's
  one-shot requirement. Cancellation, active sort invalidation and reset now
  spend that request. Eleven negative scenarios cover Escape, Close Files,
  reset, replacement and sort-setting changes; ordinary launches still work.
  `internal/ui/drop.go`, `sort.go`, `viewer.go`, `launchoptions.go`.

See `spec-coverage.md` for quoted requirements, original hunk locations and the
full feature/acceptance mapping. Later JPEG dimension policy and Store approval
requirements supersede earlier design documents where they differ.

## Security and Qodana

CodeQL alert 4 reported high-severity `go/zipslip` on PR merge
`a134d00607680a80c9a9976a87010a853b2afcf2` (source head `d306654`). Exact-three-entry
fixtures showed the original allowlist rejected traversal, absolute paths,
duplicates, nonregular entries, oversize and corrupt CRC data. Both production
callers allocate fresh private staging directories; archive-controlled traversal
was not demonstrated. Planted existing output files/symlinks did get overwritten
at the helper boundary: six negative cases reproduced it. Extraction now writes
only fixed trusted names and uses exclusive creation with checked write/close
results. No alert suppression was added. The follow-up read of alert 4 marks
its PR instance fixed; see the exact-head remote evidence below.

Qodana artifact 10035780612 from run 34171123071 names source head `d306654` in its
SARIF provenance. Its post-suppression result set contains 37 findings:

| Inspection | Count | Disposition |
| --- | ---: | --- |
| GoUnhandledErrorResult | 7 | Six HTTP fixture writes now report errors to the test; the exiting CLI explicitly ignores a failed stderr diagnostic write. |
| GoMaybeNil | 1 | `drive` rejects missing reconcile receipts before Microsoft access; a command-boundary regression pins the guard. Exclude only `scripts/storepublish/lifecycle.go` for this inspection. |
| GoReservedWordUsedAsName | 1 | Rename copied receipt to `stable`. |
| RegExpUnnecessaryNonCapturingGroup | 1 | Remove the unnecessary bracket-only group without changing matching. |
| GoErrorStringFormat | 22 | Preserve product-name capitalization. Exclude this inspection only in the four reported Store publisher files: github, lifecycle, microsoft and release. |
| GoRedundantConversion | 5 | Preserve explicit floating-point product rounding; exclude only `internal/mosaic/resample.go` for this inspection. |

The five conversions prevent fused multiply-add from changing the resampler's
rounding, as required by the [Go floating-point operator specification](https://go.dev/ref/spec#Floating_point_operators)
and used by `golang.org/x/image/draw` itself. Removing them is not a guaranteed
behavior-preserving cleanup. No test-file-wide unchecked-error suppression and
no general production nil suppression were introduced. Existing exact-file
duplication exclusions still cover every test file; four added top-level UI
tests have explicit shard assignments.

## Validation and operational boundaries

Integrator checks before the final full gate:

- Native race packages: imaging 19.021s; deletion 2.365s; slideshow 1.510s;
  Store publisher 2.981s, all passed.
- Integrated root-UI alias/save-Trash/startup regressions and adjacent
  cancellation paths: race pass, 26.942s.
- Store approval guard and mosaic package race checks passed; formatting,
  Qodana exact-test exclusions and diff checks passed.
- Case-alias negative tests ran on case-insensitive APFS. Distinct-case-file
  behavior is also tested where the filesystem supports it; no native Windows
  execution of these specific new tests is claimed here.
- Initial-head GitHub CI run 34171123243 ultimately passed validation, all four
  Linux race partitions, Windows guards and macOS guards. Its CodeQL security
  check still failed. Codex's existing security review completed with no findings
  on `d306654`; its code review completed at 00:11 UTC with no major issues.
  Neither overrides CodeQL or certifies later edits. The summary's older export
  symlink advisory refers to the already-fixed earlier finding, not alert 4.

The original review’s final `make verify` passed on its integrated working tree:
format/TUF/Qodana guards, vet, build, exact 675-test UI shard inventory, and all
four concurrent Linux/amd64 race partitions. UI package durations: ui-1 301.341s,
ui-2 276.064s, ui-3 269.931s. This run required no isolated retry. The Windows
publisher cross-build also passed. That gate preceded the user’s d7d2f8e commit;
the follow-up diff below has separate validation.

Raw local command logs are under `/tmp/picfetch-release-review`;
`make-verify.log` includes every partition's final pass. The non-UI, ui-2 and
ui-3 raw JSON copies include final package passes. The ui-1 copy was taken before
its last events and is partial; the complete final pass is in the gate log.
Historical OOM evidence remains distinct from this successful combined run.

The user subsequently committed and pushed the original review fixes as
`d7d2f8e68558460ef0a9859b8698b34491e17481`. Verified remote evidence for that head:

- [CI run 34190902492](https://github.com/frathe/picfetch/actions/runs/34190902492)
  passed validation, all four concurrent Linux race partitions, Windows tests
  and macOS native guards; the final shard completed at 05:39:54 UTC on September 8.
- [CodeQL run 34190903241](https://github.com/frathe/picfetch/actions/runs/34190903241)
  passed Go and Actions analysis. The API now reports alert 4's
  `refs/pull/17/merge` instance as `fixed`. The stored instance location still
  names its original merge SHA `a134d006`; its state is no longer open.
- [Qodana run 34190902510](https://github.com/frathe/picfetch/actions/runs/34190902510)
  and check 101949215247 completed successfully on d7d2f8e. The check reports
  “No new problems found by Qodana for Go,” with zero annotations. This was PR
  mode over changed files; it does not claim a clean full-repository scan.
- [Codex security review](https://github.com/frathe/picfetch/pull/17#issuecomment-5579970911)
  explicitly reports no security issues on `d7d2f8e685`, completed at
  05:51 UTC. The summary's lingering export-symlink advisory links the older,
  already-fixed comment and is not a new finding on this head.

The earlier local Qodana attempt stopped before analysis because its image
requires a Cloud token; no token was retrieved or supplied. The remote check
above supersedes the previous pending-CI note without turning that failed local
attempt into evidence.

## Follow-up findings on d7d2f8e

Two new P2 code-review findings were reproduced and fixed locally:

- [Unrelated exports invalidated loaded derived state](https://github.com/frathe/picfetch/pull/17#discussion_r3954798668).
  An ordinary new destination cancelled favorite-preview admission, purged
  thumbnails and cleared duplicate facts. File-work reconciliation now checks
  the committed destination against the immutable loaded-file set on a tracked
  worker before invalidating those features. Filesystem identity preserves
  noncurrent symlink and case aliases. UI delivery retries if the loaded set
  changed. The existing full-image cache writer barrier remains in place.
- [Closed-grid sensitivity changes left missing hashes unfinished](https://github.com/frathe/picfetch/pull/17#discussion_r3954798657).
  Close cancelled a held decode, then changing sensitivity regrouped only the
  surviving facts. Active Hide Duplicates or browsing now admits missing hashes
  through the existing cancellable session; browsing waits for that work before
  its final group check. Terminal Stop still prevents fresh reads.

Both reported regressions failed on the original behavior before the fixes.
The alias and stale-loaded-set guards were also disabled individually: the
noncurrent-alias test failed, and both add/remove-before-delivery cases failed.
The existing unrelated-commit reconciliation test was advanced through the new
queue stage and failed when its stale-writer retry was disabled. All guards were
restored before the final checks. Three new root-UI tests are
assigned to ui-1 (678 root-UI entries total); existing test-file exclusions apply.

Follow-up native race checks passed: root UI export/Save/EXIF/mosaic paths in
104.123s and grid hashing/sensitivity/cancellation paths in 2.345s. The adjusted
reconciliation guard also passed under the native race detector in 3.994s. The
first full `make verify` failed because Docker ran out of memory. The comparison
package printed PASS, then `signal: killed`; its package result failed at
55.717s. Docker recorded an OOM event for container `08662eacf55a` at
07:39:12 UTC. All three UI shards passed in that combined run (ui-1 355.438s,
ui-2 330.843s, ui-3 316.842s). No individual test failure was reported.

The second full `make verify` passed with `GOFLAGS=-p=1` inside Docker,
limiting package concurrency while retaining all four simultaneous race
partitions and every test. Format/TUF/Qodana guards, vet, build and the exact
678-test root-UI inventory passed. All 53 non-UI packages passed, including
comparison (23.507s) and grid (2.393s). UI shard durations were ui-1 307.744s,
ui-2 310.008s and ui-3 296.533s. The case-insensitive export test is the one
root-UI skip on Linux; the native focused suite covers that path on APFS.
No production or test code changed after this gate. Original raw JSON remains under the host-mounted
`.scratch/release-review-followups/`; retry streams go in its `retry/` directory.
The original resource failure remains distinct from the successful combined
retry. Complete command logs are `followup-make-verify.log` and
`followup-make-verify-retry.log` under `/tmp/picfetch-release-review`; the Docker
OOM event is retained as `followup-docker-oom.jsonl`. All eight raw partition
streams include their final package results.

These local changes remain uncommitted as requested. Remote results on d7d2f8e cover the previously
pushed fixes, not this follow-up diff. The user retains ownership of committing
these edits; no agent commit or push was attempted during the follow-up.

Read-only environment verification still shows `microsoft-store` allows only
Branch `main`, requires `REDACTED_REVIEWER`, permits self-review and forbids admin bypass;
no timer/custom rule is installed. Only the three expected environment secret
names/settings were inspected. No secret values were read. Partner Center's
Developer role remains intentional. No merge, tag, release, live Store mutation
or reviewer comment was made during this review.

Remaining verification boundaries:

- Windows x64 packaged startup and native WACK were explicitly deferred; old
  ARM64/other-platform smoke results do not establish either for these fixes.
- Fresh distributable packages must correspond to the eventual release commit.
  The protected Store flow still requires the validated exact package/frozen
  notes and a fresh user approval. Version 1.0.2 must not be retagged or used as
  a submission test.
- Live Publisher authentication/read access is unproven until trusted code
  lands on main and the user approves `check`. Developer submission permission
  is proved only by the next ordinary approved release, not by read-only access.
- Historical combined Docker gates suffered OOM while isolated partitions
  passed. A resource failure and a passing isolated retry are different evidence;
  neither should be described as a successful combined gate.

Standards: 2 findings, worst P2. Spec: 3 findings, worst P2. Four unique runtime
defects were fixed in that review; archive staging hardening is tracked
separately. The subsequent d7d2f8e review added the two P2 fixes above.
