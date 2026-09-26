# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

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
  [specification](.scratch/ma-028/spec.md) is published as `ready-for-agent`,
  with acceptance commands and native-input qualification requirements.
  Implementation planning is next; MA-029 through MA-033 remain proposals.

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
