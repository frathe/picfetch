# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- Keep Grid View marquee selection aligned with the pointer when the selection
  bar or ranked-search progress changes; update those bars after mouse-up.

- Prevent planted `.new` symlinks from redirecting Unix in-app updates, and
  reclaim stale staging files left by an interrupted update.

- Recheck the selected mosaic wallpaper display after writing its PNG, so a
  monitor connected during export cannot turn a targeted Linux request into
  a global wallpaper change.

- Keep File > Close Files available during an initial scan or sort, then
  disable it again when Escape cancels that work in an empty viewer.

- Fixed "Reveal in file manager" on Linux for filenames containing commas.

#### Internal

- Restricted WinGet publishing to successful Release workflow runs, strengthened
  the manual-dispatch regression guard, and updated failed-publish recovery guidance.
- Hardened `make ci-failures` against command injection from crafted branch names
  and Make variable overrides.
- Include `libglib2.0-bin` in the Docker race, full test, and coverage
  containers so Linux desktop launcher tests can invoke `gio`.

## Open

- **Location Map MVP:** Design interview complete; behavior recorded in
  [the feature design](<next feature.md>). Implementation planning and code are
  pending. Include duplicate-aware GPS fallback, Favorite-owned GPS persistence,
  bounded tile residency and 10k qualification; Ronin owns the 30k stress test.

## Deferred

### Fyne upgrade deferred

Keep Fyne at v2.8.0 in [PR #19](https://github.com/frathe/picfetch/pull/19).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
The four grouped `golang.org/x/*` updates remain in the PR.


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
