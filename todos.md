# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## TODO

### Validate shared dog gaze after the next push

The release-review follow-ups are merged and current main `a5b4c31` passed CI,
CodeQL and Qodana. Unrelated exports preserve loaded derived state, and changing
duplicate sensitivity after closing an unfinished hash pass resumes the work.
Main's only remaining Qodana result is the duplicated Trane/Finis atlas mapping.

The shared gaze extraction is tracked in
`finished_refactorings/2026-09-08-shared-dog-gaze.md`. Its changes remain local until the user
commits and pushes them; rerun remote checks on that commit before release.
The local full gate also exposed an uncontrolled scan-cancellation test; the
test now holds its directory read across replacement explicitly and checks
that cancelled work stops. The plan retains the failed run and retry evidence.

### Investigate intermittent local race-gate failures

The 2026-09-08 follow-up gate retained all four raw streams and recorded a
Docker OOM event. This time comparison printed PASS before its process was
killed; all three root-UI shards passed. A full `make verify` retry passed with
`GOFLAGS=-p=1` inside Docker, preserving all four race partitions while limiting
package concurrency. This resource limit was scoped to the verification run;
the default runner now applies a 16 GiB container memory budget matching CI.
The release assessment keeps
the original failed run and the successful combined retry separate.

The 2026-09-07 Qodana cleanup's `make verify` run reported a package-level failure
in `ui-3` without an individual test failure in the compact log. The isolated
Linux/amd64 race-shard retry passed with raw output preserved. The cause remains
unknown; retain raw streams on the next concurrent run. See
`finished_refactorings/2026-09-07-qodana-findings.md` for commands and evidence.

Later concurrent runs on the unchanged PR head and the mosaic optimization both
captured Docker OOM events for UI shard 3; a 1 GiB Go memory target did not prevent
the latter. The same shard passes alone. Retain the original-run uncertainty,
but investigate Docker's overall memory pressure for this reproduced failure.
Latest evidence: `plans/2026-09-07-mosaic-speed-progress.md`.

The Trane verification on 2026-09-07 also captured a Docker OOM, this time killing
`internal/ui/compare` while all three main-UI shards passed. Logs and the isolated
retry are recorded in `finished_refactorings/2026-09-07-animated-trane.md`.


### Address the 2026-09-06 maintainability audit

Track the reconciled findings in [needs_refactoring.md](needs_refactoring.md) and execute the
[phased implementation plan](plans/2026-09-06-maintainability-plan.md) against the
[specification](.scratch/maintainability/spec.md) and [published tickets](.scratch/maintainability/ticket-breakdown.md). Implementation is in progress: checked TIFF spans, bounded mosaic preparation, deletion identity,
complete cache records and saved JPEG dimensions (MA-001/002/004/006/007) are complete;
clipboard errors, Trash configuration, preview timestamps and RSS arithmetic
(MA-011/013/018/019) are also complete. MA-005 chooser paths and MA-012 Windows
list decoding now pass their actual Windows guards and are complete. Queued animation/picture-frame pacing and chooser lifecycle (MA-003, tickets 12–14), ancillary read cancellation and generation-safe duplicate facts (tickets 15–18) have passed their common gates. The ticket 15 manual-test correction, map lifetime and position-poller work (MA-015/016) pass the next common gate; native poller movement/close/shutdown also passed. Background clipboard encoding and EXIF reads (tickets 21–22) pass their shared common gate; serialized original-file mutations and committed-write/cache reconciliation (ticket 23, completing MA-014) pass their common gate. Duplicate-group reuse/cancellation (ticket 24, MA-008) passes its common gate and measured benchmarks. Native Windows/macOS/Store guards pass; the comparison shutdown crash is fixed and gated. Packaging inputs are pinned, all seven artifacts build with inspected metadata, and the actual macOS package renders/quits cleanly. All eight Windows test-package runs and 14 required native guards pass without skips; the follow-up common gate passes. Ordinary/Store Windows ARM64 graphical startup and clean quit pass; user-operated comparison pan/zoom/swipe/detail pass. MA-017 native/renderer coverage is complete with recorded Retina and Windows 100% scale/DPI 96 environments. Refreshed macOS and both Linux packages render and quit cleanly; Linux uses software GL and amd64 CPU emulation. Native x64 Windows startup and WACK remain deferred to the user’s later Windows testing. Keep remaining correctness, lifecycle,
platform and accepted watch work linked to the stable MA identifiers and ticket evidence.

Tickets 28–30 are complete: preview contention was measured and reduced on small
CPU budgets, and command admission/Escape behavior has an explicit tested matrix.
The user-reported progressive-hide regression is fixed with an independent,
tracked grouping worker. [Phase 6 evidence](.scratch/maintainability/evidence/28-29-preview-contention.md)
and the complete canonical Linux race/golden gate pass. The native macOS Copy
Selection screenshot mismatch remains recorded in the validation evidence.
HEIC/verifier tickets 31/32 retain their separate future-upgrade triggers.

### Complete native x64 Windows packaging tests

User will test later on native x64 systems. Follow the
[Windows test checklist](.scratch/maintainability/windows-test-todo.md) for
ordinary/Store image rendering, comparison, clean quit and SDK/WACK evidence.
Windows VM experiments are deferred; retain their failures as diagnostic
results. Continue non-Windows validation independently.

### Activate and verify approved Microsoft Store updates

PicFetch 1.0.2 is published. The publisher now prepares exact validated artifacts
and frozen notes automatically, then requires frathe's GitHub environment approval
for each rollout. It preserves original artifact provenance, metadata, version
checks, serialization and durable recovery. Cron was removed: manual approved
`check`/`reconcile` observes certification, and a separate approved `submit` handles
a waiting release. See `docs/microsoft-store.md` for the operational sequence.

The existing main-only environment, frathe reviewer, disabled bypass/timer/custom
rules and three secret names were verified through GitHub API metadata. The user
confirmed the linked Developer application and rotated key. Credentials remain
only in GitHub Secrets. The workflow is now on main. Its first automatic `v1.0.3`
publisher run (34220437267) failed in preparation because the Windows artifact
recorded CRLF release notes while the tagged Git blob used LF. The publisher fix
compares notes after CRLF-to-LF conversion only; changed content, whitespace and
bare carriage returns still fail provenance checks. Read-only preparation now
passes against the original producer run 34218808203 and artifact 10053491693;
no replacement build or Store submission was needed. `make verify` and focused
publisher race tests pass, including preparation and approved submission with
simulated Windows notes. Disabling the comparison in a temporary compiler overlay
made all three content/whitespace rejection cases fail as expected.
The line-ending fix is committed as `16abd99`. The next approved publisher run
(34222625099) passed preparation and created draft 1152921505701835817, then stopped
with `pending submission has changes outside the recorded update`. Receipt
6326959682 remains in phase `created`; Partner Center shows the draft's original
notes and unchanged listings/packages. Read access and draft creation are now
confirmed. Preserve this draft and receipt while diagnosing the exact API mismatch.

The local `check` enhancement reports bounded differing field paths and validation
booleans without metadata values or Store/receipt writes. Its publisher race tests
and full `make verify` pass; temporary compiler overlays confirmed the redaction,
size, receipt/base/state selection, missing/null and read-only guards fail when
broken. Land that diagnostic
change and approve a fresh `check` run on main (no tag needed), then use its
`pending_validation` report to finish the fix. Recovery of this existing draft will
use a fresh approved `reconcile`, not a new `submit`. The actual metadata discrepancy
and successful upload/certification remain open. See
`plans/2026-09-08-store-draft-mismatch.md`. Keep the live 1.0.2 tag and artifact unchanged.

Focused tooling race tests, actionlint, formatting, TUF/Qodana checks, host vet/build,
Windows publisher cross-build and Linux shard inventory pass for the approval
amendment. Prior verification:
the combined `make verify` recorded Docker OOM; isolated UI race shards and non-UI
partitions passed. That combined invocation did not succeed. No repeat of the full
race suite was performed for this tooling amendment; fresh check evidence is in
the implementation plan. Nothing was committed, pushed or submitted to Microsoft.

Spec and tickets: `.scratch/microsoft-store-updates/README.md`.
Plan and evidence: `plans/2026-09-07-microsoft-store-updates.md`.

## LATER

### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS runner, in August 2027. Before then, decide
whether PicFetch will stop shipping an Intel macOS archive or retain it through another build path. If Intel support
remains, replace the `macos-15-intel` release job with a tested alternative; otherwise remove the x86_64 artifact and
update the release and installation documentation. The native Apple-silicon build is not affected by Rosetta's
retirement.

## not deemed worth implementing (edge cases)

- Windows releases are not Authenticode-signed. Controlled Folder Access and SmartScreen both judge by signature and
  reputation as well as by which program is writing, so an unsigned `picfetch.exe` can still be blocked even with the
  in-process swap (see Done → Bugfix above, where the block would now name `picfetch.exe` instead of `cmd.exe`). The
  real remaining fix is signing the Windows release build — Azure Trusted Signing or a purchased certificate — in
  `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the IDE's 71. The 8 fragments CI is missing are 7
in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries exactly 3
`#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in db" warnings, naming exactly those two files and no
others, emitted immediately after the line `The Project analysis stage completed in 41s` — so Qodana's own log shows
detection succeeded and serialisation into the report/SARIF failed afterwards. This is an upstream defect, not a
picfetch config problem: nothing here suppresses or excludes those two files, and the drop happens before any
project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes this defect invisible going forward in this
repository, because every dropped fragment happens to live in a test file that the exclusion now removes from the
inspection entirely — recorded here so the defect is not lost along with the rule that used to surface it. Of the
12-fragment CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the other 4 are the
source-suppressed production fragments in the orientation pixel loops recorded above, so nothing about that gap is left
open — only the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte offsets and anchoring detail, and
`plans/2026-08-29-qodana-serialisation-bug-report.md`, Task 8's draft of the upstream report text — as of this writing
not yet submitted to JetBrains; check that file for whether it has been sent since.

- Follow up the pinned fyne.x map widget’s process-global decoded tile cache (`widget/mapcache.go`): PicFetch’s 16 MiB tile cache bounds encoded bytes only; the upstream decoded map has no eviction. Track separately from MA-015 request/failure bounds.
