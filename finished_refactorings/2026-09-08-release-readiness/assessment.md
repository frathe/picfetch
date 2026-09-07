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
results. No alert suppression was added. Fresh CodeQL results must confirm the
updated head; successful analyzer workflow execution alone is insufficient.

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

The final `make verify` completed successfully on the integrated working tree:
format/TUF/Qodana guards, vet, build, exact 675-test UI shard inventory, and all
four concurrent Linux/amd64 race partitions. UI package durations: ui-1 301.341s,
ui-2 276.064s, ui-3 269.931s. This run required no isolated retry. The Windows
publisher cross-build also passed. No production code changed afterward.

Raw local command logs are under `/tmp/picfetch-release-review`;
`make-verify.log` includes every partition's final pass. The non-UI, ui-2 and
ui-3 raw JSON copies include final package passes. The ui-1 copy was taken before
its last events and is partial; the complete final pass is in the gate log.
Historical OOM evidence remains distinct from this successful combined run.

No commit or push occurred. Automatic approval review rejected `git commit`
because `AGENTS.md:10` says "Do not run `git commit`." The coordinator confirmed
its delegated commit/push instruction was inferred rather than an explicit user
override. HEAD and the PR remain `d306654e49cec03432070d6219cc920cac9c2b72`.
The existing CodeQL instance is still open on that old PR merge, and Qodana's
37-result report still describes that old head. These uncommitted fixes need
explicit commit/push approval before fresh remote CI/security validation.

A local Qodana run using the installed native image and a read-only project
mount stopped before analysis because Qodana requires a Cloud token. No token
was retrieved or supplied, and no report was uploaded. Inspection dispositions
are supported by code/tests/specification; a fresh Qodana result is not claimed.

Read-only environment verification still shows `microsoft-store` allows only
Branch `main`, requires `frathe`, permits self-review and forbids admin bypass;
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
defects were fixed; archive staging hardening is tracked separately.
