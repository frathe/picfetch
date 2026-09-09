# PicFetch — TODOs

## Done

### What's Changed

#### New Features

### Keep Random mosaics varied and add Shelf

Mosaic keeps the original seeded **Random** arrangement as the default and adds
an ordered **Shelf** arrangement in the Advanced settings. Both modes use the
same chosen source pool, frame, size, overlap, and shadow settings. Random
retains rotation through 90 degrees; Shelf is deliberately axis-aligned and
hides its rotation control. Missing or unknown saved layout values safely
restore Random.

Shelf carries the strict final-image visibility and overlap-response checks,
including every retained photo occurrence across frames, shadows, and
source-pool sizes. It remains axis-aligned, but Overlap stays available and has
a measured effect. Random retains its original primary-card visibility guard,
varied placement, and rotation checks through 90 degrees. Its gap-repair cards
can still be fully covered, so the strict every-occurrence condition
intentionally does not apply to Random. See the [implementation
record](plans/2026-09-08-mosaic-rotation-overlap.md) for the selected contract
and the remaining limitation.

The user approved the current visual result. Earlier apparent gap-filling or
maze-like output is historical behavior, not a regression introduced here.

#### Bugfix

Grid View now keeps the ring visible while scrolling with the mouse wheel or
trackpad, and live duplicate merges preserve the viewport. Explicit selection
and the displayed image stay unchanged. Native scrollbar movement keeps its
existing behavior; the next duplicate reflow reconciles the ring.

Shift+D now opens the highlighted shot's known duplicate group immediately
while analysis continues, including hidden copies. Live updates follow that
same source, and pending updates preserve the group. Navigation, opening a
copy, exits, reordering, and sensitivity changes have regression coverage.
Both manuals are updated; all SDD/TDD acceptance checks and `make verify` pass.
See the [implementation record](finished_refactorings/2026-09-09-grid-duplicate-browse-during-scan.md).

#### Internal

Local race verification now keeps each attempt's four raw streams, console,
exit status, and available Docker/cgroup memory diagnostics in a unique host
directory under `.scratch/race-runs/`. Failed and interrupted runs retain their
evidence and failure status; diagnostic collection precedes container cleanup.
The existing 16 GiB mitigation and subsequent passing gates close the memory
investigation. The original September 7 package-only failure remains unexplained
in the [implementation and evidence record](finished_refactorings/2026-09-09-local-race-evidence.md).
Focused race tests, deliberate regression checks, and one default `make verify`
pass; its four complete streams and final memory counters survive cleanup.

## TODO

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
