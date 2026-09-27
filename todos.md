# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- Escape/Close Files now leave ordinary native maximization before restoring
  the compact welcome window. Preserve fixed-size and Grid-owned restore
  behavior. A native-boundary regression and repeated Linux desktop checks
  cover the canvas/native size mismatch; fresh `make verify` and changed-file
  GoLand inspections passed.

#### Internal

- MA-028 shared command admission is implemented across menus, shortcuts, keys,
  direct actions and open delivery. Pure decisions precede yielding; existing
  features retain payload capture and worker lifetimes. See the
  [implementation and verification record](docs/command-admission-verification-2026-09-27.md).
  `make verify` and all 68 changed-file GoLand inspections passed. The
  [Linux native-input scenarios](docs/command-admission-linux-qualification-2026-09-27.md)
  passed their scoped OS-injected checks. The subsequently found native
  maximized-window reset failure is repaired and locally verified; remaining
  platform/operator and CI acceptance is below.

- Stabilize the similarity protocol race test by dispatching its helper before
  the test runner starts and bounding its lifetime from the parent. Preserve
  configuration/error transport and cancellation/exit coverage. See the
  [diagnosis and verification record](plans/2026-09-26-similarity-protocol-timeout.md).

## Open

- **Application architecture:** the [cross-PR assessment](needs_refactoring.md)
  recommends shared command policy (MA-028), explicit browsing ownership and
  collection transitions (MA-029/030), followed by Favorite ownership, a bounded
  worker-lifetime pilot and launch policy (MA-031 through MA-033). Keep feature
  state local and preserve explicit composition. MA-028's
  [design](docs/command-admission.md) is accepted: full command migration with
  shared pure admission and the agreed input/yield corrections. Its
  [specification](docs/ma-028/spec.md) and
  [published ten-ticket plan](plans/2026-09-27-ma-028-command-admission.md)
  track implementation and acceptance. Tickets 01-09 are done/resolved at
  `9dc3a81`, with their implementation checklists complete; retain
  ticket 10 for the remaining physical/native desktop gates. Linux GNOME/XWayland runtime
  checks passed editor shortcuts, actual menu activation/enablement, modal
  controls and native close during held region copying with OS-injected input.
  The Linux Escape/maximize defect found during qualification is fixed: native
  size/state checks pass repeatedly, including fixed-size controls; the new
  native-boundary regression, changed-file inspections and fresh full Make
  gate pass. Historical diagnosis and final evidence remain in the Linux record.
  Physical input on all three platforms and full Windows/macOS desktop
  acceptance remain unverified. At `4674cca`, fresh PR 66 CI and CodeQL passed,
  Codex code/security reviews have no findings, and Qodana's inspected SARIF has
  zero results after addressing the original eight findings. The
  [review-loop record](docs/command-admission-pr-66-review-2026-09-27.md) retains
  exact evidence; PR 66 records the final documentation-only head's mandatory
  checks/reviews. Local GoLand evidence remains distinct from Qodana CI.
  The case-insensitive export regression now passes on temporary FAT16 storage,
  including five race runs. Windows/macOS CI now selects focused native guards
  for case-alias export and macOS Copy-menu behavior, with command-boundary
  tests rejecting missing/skipped evidence. The actual new runner passes on
  Linux FAT16 and rejects an ext4 skip. The focused guards now also pass in
  native Windows amd64 and macOS arm64/amd64 CI, including the AppKit assertion;
  these isolated tests do not replace desktop/physical-input checks. The user
  authorized the PR 66 review loop. The reviewed planning Markdown is tracked under
  [docs/ma-028](docs/ma-028/README.md), with its personal home path redacted and
  raw evidence left ignored, as recorded in the
  [portability audit](docs/ma-028-portability-audit-2026-09-27.md).
  MA-029 through MA-033 remain proposals.

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
