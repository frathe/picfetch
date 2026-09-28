# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Help -> Privacy policy opens the installed build's policy in a scrollable
  offline window, alongside Release Notes. Menu labels use the existing English
  and German translations; the policy retains its published English text.

#### Bugfix

- Menus and keyboard shortcuts now respect what you’re doing. When a dialog is
  open, commands won’t change the photo behind it, and Copy and Select All work
  on the text you’re editing. Actions that could interrupt copying a selected
  part of a photo are temporarily disabled. Choosing an unavailable action also
  leaves your selection intact.

- Keep Export/Delete keyboard focus on the prompt through Tab and checkbox
  clicks, and release it on dismissal. Windows native qualification found
  Escape swallowed by Export's checkbox or Delete's hidden panel; focused
  regressions and native repeats verify the repair. See the
  [Windows evidence](docs/command-admission-windows-qualification-2026-09-27.md).

- Escape/Close Files now leave ordinary native maximization before restoring
  the compact welcome window. Preserve fixed-size and Grid-owned restore
  behavior. A native-boundary regression and repeated Linux desktop checks
  cover the canvas/native size mismatch; fresh `make verify` and changed-file
  GoLand inspections passed.

#### Internal

- MA-029 browsing ownership is complete: all nine tickets and acceptance points
  are resolved. Focused TDD, all 31 changed-file IDE inspections, full Linux
  race/native, Windows/macOS CI, CodeQL and fresh post-suppression Qodana
  evidence are in the [archived plan](finished_refactorings/2026-09-27-ma-029-browsing-visits.md).
  CI also verified the shutdown-test isolation and HEIC queued-stop fixes.
  PR round 1's nested-search and stale cohort-Grid retirement findings are
  repaired with real-transition regressions; fresh review remains a PR gate.
  [PR 68](https://github.com/frathe/picfetch/pull/68) maintains final review
  dispositions and latest-head checks; merge/release is not authorized.

- MA-028 shared command admission is complete across menus, shortcuts, keys,
  direct actions and open delivery. All ten tickets are resolved. Linux,
  Windows and macOS native qualification passed, with physical editor-input
  provenance retained in the platform records. The clean PR 66 acceptance
  round at `e2d3b30` passed all 17 checks, fresh code/security reviews and
  inspected Qodana SARIF; source tests and GoLand evidence are retained at
  their recorded revisions. See the [completed ticket](docs/ma-028/issues/10-native-qualification.md),
  [archived plan](finished_refactorings/2026-09-27-ma-028-command-admission.md)
  and [review evidence](docs/command-admission-pr-66-review-2026-09-27.md).
  PR 66 records fresh gates for the final documentation-only closure head.

- Stabilize the similarity protocol race test by dispatching its helper before
  the test runner starts and bounding its lifetime from the parent. Preserve
  configuration/error transport and cancellation/exit coverage. See the
  [diagnosis and verification record](plans/2026-09-26-similarity-protocol-timeout.md).

## Open

- **Application architecture:** MA-028's shared command policy and MA-029's
  browsing ownership implementation are complete.
  The [remaining refactoring backlog](needs_refactoring.md) covers collection
  transitions, Favorite ownership, a bounded worker-lifetime pilot and launch
  policy (MA-030 through MA-033).
  [MA-030's accepted design](docs/collection-transitions.md) resolves all eleven
  interview decisions and requires both collection-model and reconciliation
  slices. Its [local specification](.scratch/ma-030/spec.md) is resolved:
  56 user stories and 18 acceptance criteria cover both slices. The approved
  [nine implementation tickets](.scratch/ma-030/issues/README.md) are published
  with blockers and model assignments. The
  [archived Deep SDD record](finished_refactorings/2026-09-28-ma-030-collection-transitions.md)
  records all nine completed tickets: sole collection ownership and ordered
  reconciliation, replay/capture, latest-choice sort, batch removal, unavailable
  recovery, committed write/policy effects and lifecycle convergence. Full CI and
  inspected Qodana SARIF passed on 2722b1d;
  [PR 69](https://github.com/frathe/picfetch/pull/69) records the continuing
  latest-head code/security reviews, dispositions and final checks.
  [MA-031's accepted design](docs/favorite-ownership.md) resolves all twelve
  interview decisions: incremental migration, ownership/retirement, the common
  64 MiB definition limit, validation, partial inventories, freshness, scoped
  retention, bounded handles, UI storage workers, cancellation and confirmation
  conflicts. The [local specification](.scratch/ma-031/spec.md) is resolved,
  with 70 stories, test boundaries and 22 acceptance criteria
  covering both migration stages. The approved
  [nine implementation tickets](.scratch/ma-031/issues/README.md) are published
  with blockers and verification commands. All nine tickets are complete (shared
  bounded ownership, Location Map, scoped similarity/search, maintenance and
  cohorts, asynchronous UI reads/lifecycle, validated saves and identity-bound
  removal, owner-bound previews and convergence/native qualification).
  The [archived Deep SDD record](finished_refactorings/2026-09-28-ma-031-favorite-ownership.md)
  records all ACs, full CI, four-platform native evidence and inspected Qodana
  SARIF on c1f47b4. [PR 70](https://github.com/frathe/picfetch/pull/70) records
  latest-head code/security reviews, dispositions and final checks.
  [MA-032's design record](docs/worker-lifetimes.md) resolves all nine interview
  decisions. Its [local specification](.scratch/ma-032/spec.md) is published as
  `ready-for-agent`, with 60 user stories, test boundaries and 20 acceptance
  criteria. It defines a root sort/display SVG pilot, a conditional root/display
  token migration and a contracts-and-tests fallback. Its
  [nine approved tickets](.scratch/ma-032/issues/README.md) are published as
  with [Deep SDD implementation and evidence](plans/2026-09-28-ma-032-request-lifetimes.md)
  underway. Ticket 01 is complete (shared contract and sorting); ticket 02's verdict
  gates migration or fallback. Each completed ticket gets its own commit;
  final qualification and the latest-head GitHub review loop remain open.
  MA-033 remains a proposal.
  Keep feature state local and preserve explicit composition.

- **Native Location Map gesture timing:** replace hash-only change detection with
  independently verified pan/zoom transforms before enabling formal latency
  qualification again. The current helper rejects these measurements; manual
  trials and stage/RSS observation remain usable. Existing maintainer performance
  acceptance stands separately from measured timing evidence.

## Deferred

<!--
Inactive pause record, retained for reuse after Ronin authorizes another CI
pause. Update the date and restore this section when that happens.

### Qodana CI paused

Disabled at Ronin's request on 2026-09-25 after the trial subscription expired.
Keep its configuration for possible restoration; this is not a passed scan.
GoLand inspections and CodeQL remain in use. The
[local inspection research](docs/local-qodana-inspections-2026-09-25.md) records
the IDE-only Qodana option, licensing distinction and historical inspection advice.
-->

### Fyne upgrade deferred

Keep Fyne at v2.8.0, including [PR #59](https://github.com/frathe/picfetch/pull/59).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
PR #59 retains its grouped Sigstore v1.11.0 update. The earlier hold in
[PR #19](https://github.com/frathe/picfetch/pull/19) retained its four grouped
`golang.org/x/*` updates.


### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS runner, in August 2027. Before then, decide
whether PicFetch will stop shipping an Intel macOS archive or retain it through another build path. If Intel support
remains, replace the `macos-15-intel` release job with a tested alternative; otherwise remove the x86_64 artifact and
update the release and installation documentation. The native Apple-silicon build is not affected by Rosetta's
retirement.

## not deemed worth implementing (edge cases)

- **Verifier size refactoring (MA-024):** Declined by Ronin on 2026-09-13.
  Keep upstream Sigstore/TUF verification. Potential savings of a few megabytes
  do not justify the security risk and maintenance burden of a custom or
  trimmed verifier. The refactoring plan and upgrade watch have been removed.

- **Retained decoded EXIF map tiles (MA-025):** Accepted by the user on 2026-09-09
  for occasional single-photo EXIF lookups. The upstream decoded-tile cache is
  unbounded and its long-session impact remains unmeasured. The new Location Map
  changes the scale assumption: bounded tile residency and sustained browsing
  measurements belong to that feature's open work above. The original decision
  does not qualify collection-scale map browsing.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)
