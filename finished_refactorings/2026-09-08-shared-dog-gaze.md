# Shared dog gaze and final release check

Status: completed and accepted by the user on 2026-09-08. Changes remain
uncommitted; acceptance does not replace the release checks recorded below.

Base: current main `a5b4c31cad8909372b8bb0e24b3649d9a63649e5`.
The already completed release review and final merged PR checks remain evidence
for their recorded commits. Review the small main delta and this new change.

Route: compact Deep, because the extraction crosses ui, help and the existing
widgets package. No new package, dependency, artwork, strings, animation timer,
release operation or Store-policy change. No commit or push.

## Acceptance criteria and decisions

1. Both characters use one atlas decoder and one gaze/portrait implementation.
   Keep artwork loading/caching, Trane fringe correction, hosting, event routing
   and geometry with their current owners. Inspect the final diff and shared
   calls; the Qodana SARIF establishes the original duplicated fragments.
2. Preserve all 16 directions, neutral dead zones, resizing, hover exit, hidden
   Trane reset, Finis window reuse/reopen, and unchanged artwork pixels.
   Oracle: `go test -race ./internal/ui ./internal/ui/help -run 'Test(Trane|Finis|Help_Finis)'`.
   Run these existing characterizations before extraction. Negatively verify
   shared pose selection against these integration tests, then restore it.
3. Keep the app's release behavior intact. Oracle: `make verify`, once after
   focused checks. Record resource or platform limits separately.
4. Check current main CI, security alerts, and Qodana findings without claiming
   those remote results cover an uncommitted refactor. Keep release packaging,
   Windows WACK and live Store checks explicit if still unverified.

## Tasks and ownership

- Lead: review main delta, design and implement `internal/ui/widgets/gaze.go`,
  adapt `internal/ui/trane.go` and `internal/ui/help/finis.go`, preserve current
  surface tests, update architecture and relevant TODO status, and final review.
  Existing per-character wrappers remain responsible for their different layout
  and pointer coordinate systems; common code handles frame extraction and pose
  presentation. No additional UI interaction is introduced by the shared object.
- Scout: read-only extraction of current-main Qodana SARIF and code-scanning
  status into `/private/tmp/picfetch-main-release-check`; no repository edits,
  review judgement, commits, comments, credentials or environment changes.
  Return exact report provenance and finding locations. Verification oracle:
  parsed SARIF plus GitHub JSON, retained in the temporary directory.

## Delegation gate

Scout G1 yes (bounded report query), G2 yes (retained JSON/SARIF parse), G3 yes
(zero repository writes), G4 yes (only report IDs and paths), G5 yes (report
contents not read by Lead). Scripted extraction handles parsing; scout joins
the source provenance and tool results while Lead implements. One scout only.
Review and architecture remain with the Lead per the repository process.

## Cost and verification ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| Main report scout | 1 / 1 | Lead | no | retained JSON/SARIF verified at main; one atlas duplication, zero current CodeQL results |
| Shared gaze extraction | 0 / 0 | 1 | no | baseline and final focused race tests passed; wrong-sector negative verification failed in both host integrations as expected |
| Scan-test synchronization follow-up | 0 / 0 | 1 | no | full-gate race preserved; controlled listing test negatively verified; 20 Linux race repetitions and existing storage-fixture consumers passed |
| Final gate | 0 / 0 | 1 | 2 total: first failed, corrected retry passed | `make verify` exit 0; all four concurrent Linux race partitions passed |

## Implementation and focused evidence

`widgets.Gaze` owns the atlas cell mapping, neutral frame, direction selection
and portrait refresh. Trane retains its cached frames, fringe correction,
fallback artwork and scaled face coordinates. Finis retains its centered
portrait, hover handling and window lifecycle. Neither host gains a worker or
timer. Existing integration tests still check source pixels and surface behavior;
added assertions distinguish Trane's scaled 18-pixel dead zone from Finis's
24-pixel dead zone without adding a new test file or top-level test.

- Baseline: `go test -race ./internal/ui ./internal/ui/help -run
  'Test(Trane|Finis|Help_Finis)'` passed before extraction.
- Negative verification: changing the shared sector wrap from `+16` to `+17`
  made both `TestTraneFollowsCursor` and `TestFinisGazeDirectionsAndRest` fail
  on their expected atlas pixels. The correct mapping was restored immediately.
- Final focused gate: `go test -race ./internal/ui ./internal/ui/help
  ./internal/displays -run 'Test(Trane|Finis|Help_Finis|DisplayLinux_)'` passed.
  This also exercises the XRandR parser tests moved onto all platforms on main.
- `goimports` and `git diff --check` passed. The atlas mapping and sector
  calculation now each occur only once in the shared component.

## Current main review

The delta from the already reviewed merged head `6c35f38` to this plan's main
base consists of a README Store badge and moving the existing XRandR parser
and tests into portable files. The parser's split iterator preserves its line
processing. No new actionable Standards or Spec finding was identified in
that delta or the shared-gaze diff.

The full gate then exposed an existing Standards P2 in the scan-cancellation
test: it assumed a newly started scan worker could not run before the next
drop. It created another fixture between those events without any ordering
guarantee. The race report shows that test's own scan progress calling
`fyne.Do`, whose test driver executes inline, overlapping the next drop's
label update. The desktop driver queues this callback onto its main thread.
This evidence establishes a test synchronization problem; no production scan
change is retained.

Exact-main evidence, retained in `/private/tmp/picfetch-main-release-check`:

- [CI](https://github.com/frathe/picfetch/actions/runs/34206976654),
  [Qodana](https://github.com/frathe/picfetch/actions/runs/34206976672) and
  [CodeQL](https://github.com/frathe/picfetch/actions/runs/34206976677) passed.
- Qodana's post-suppression SARIF contains one moderate `DuplicatedCode`
  cluster: the atlas mapping in the original Trane and Finis implementations.
  Those two blocks are replaced by the shared decoder. A fresh remote Qodana
  result is still required after the user commits and pushes this change.
- No open code-scanning alerts on main; both exact-main CodeQL analyses report
  zero results and no analysis error or warning.

This is a bounded follow-up to the completed release review, not a fresh audit
of every feature. Native Windows x64 packaged startup, WACK, and live Microsoft
Store authentication/submission were not exercised by this task. Distributable
packages and remote checks must match the eventual release commit. The Store
listing and protected approval flow are unchanged.

## Full-gate failure and test correction

The first `make verify` passed formatting/TUF/exclusion checks, vet and build.
Its non-ui partition passed, and ui-2/ui-3 passed in 312.233/304.466 seconds.
Ui-1 failed in `TestHandleDrop_SupersededScanGoroutineExits` with a data race.
The full command exited 2; it is not counted as a passing verification.

Preserved evidence:

- `/private/tmp/picfetch-shared-gaze-verify.log`: full first gate output.
- `/private/tmp/picfetch-shared-gaze-first-ui-1.json`: raw failing partition
  stream copied before the original container was removed.
- The isolated original test passed 110 native repetitions and 20 Linux
  repetitions. The race was observed in the full gate, not reliably reproduced
  by that faster loop; the captured conflicting call sites identify the faulty
  scheduling assumption.

The existing test now holds a directory listing until the replacement scan
has loaded its file, then releases the superseded scan and waits for that
generation's completion. It checks both that child directories are no longer
listed and that the replacement file remains selected. A controlled
`DirectoryURI` shares the existing immutable test-storage adapter with
`ReaderURI`; there is no new production seam or mutable global test hook.
No top-level test or test file was added, so the shard manifest and Qodana
test-file exclusions are unchanged.

Negative verification temporarily supplied a non-cancellable context to
`filescan.Images` at the real `handleDrop` call site. The revised test failed
in 0.20 seconds with `superseded scan continued listing child directories`.
That deliberate violation was restored immediately; `internal/ui/drop.go`
has no final diff. Final focused checks and the full retry are recorded below
separately from the failed run.

## Final verification and handoff

- The corrected cancellation test passed 20 Linux race repetitions in 26.053
  seconds. `internal/uitest`, `internal/filesort`, `internal/imaging`, and
  `internal/ui/exifwin` passed focused native race checks after extending the
  existing test-storage adapter.
- The full `make verify` retry exited **0**. Formatting, TUF root consistency,
  exact Qodana test exclusions, vet, build, shard inventory and all four
  concurrent Linux/amd64 race partitions passed. Root UI timings were ui-1
  344.924s, ui-2 321.599s, and ui-3 298.179s; the non-ui partition also passed.
  The formerly failing cancellation test passed within this combined run.
- Retry output: `/private/tmp/picfetch-shared-gaze-verify-retry.log`. There
  were no additional code edits during that gate. The temporary isolated
  Linux reproduction container was removed.
- Final diff review: the deliberate cancellation violation is fully restored,
  shared atlas/angle logic occurs once, and `git diff --check` passes.

| Standards | Intended behavior |
| --- | --- |
| One existing P2 test synchronization defect found and fixed; no remaining actionable finding in this follow-up. | Both characters retain their artwork, gaze, layout and lifecycle. Cancellation is checked while a directory read is held. No new behavior mismatch found. |

No remaining code blocker was found within this bounded follow-up. Changes
remain uncommitted on `codex/shared-dog-gaze`, based on the main revision above.
The user must commit/push and obtain fresh remote checks, including Qodana,
before including this change in a release. The Windows package and live Store
verification boundaries above still apply.

Suggested commit message:
`refactor(ui): share dog gaze and stabilize scan cancellation test`
