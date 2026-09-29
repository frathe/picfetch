# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- In picture-frame mode on Linux and Windows, the menu bar no longer sits
  on top of your photos. Rest the mouse along the top of the screen for a
  moment and the menu slides in. Move the mouse away and it slides back
  out. While a menu is open, it stays put. On a Mac, the menu bar is
  unchanged.

- Help -> Privacy policy opens the installed build's policy in a scrollable
  offline window, alongside Release Notes. Menu labels use the existing English
  and German translations; the policy retains its published English text.

#### Bugfix

- Location Map now redraws immediately when opened with Shift+L and maximizes
  the window when you enter or return to it.

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

- Improve consistency when switching collections, opening Favorites, saving,
  cancelling background work, and closing PicFetch. The completed architecture
  cleanup also makes future changes easier to maintain.

- Stabilize the similarity protocol race test by dispatching its helper before
  the test runner starts and bounding its lifetime from the parent. Preserve
  configuration/error transport and cancellation/exit coverage. See the
  [diagnosis and verification record](plans/2026-09-26-similarity-protocol-timeout.md).

## Open

No open items.

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
  unbounded and its long-session impact remains unmeasured.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)
