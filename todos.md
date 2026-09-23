# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- *delegate heic image rendering to the OS*
  Add back support for HEIC image formats, on start check if the system supports rendering of heic images. if yes save
  that information to the settings so we don't have to check that on every launch. when the os does support the 
  rendering we delegate the rendering to the OS and enable HEIC support.

#### Bugfix

- Enable Windows HEIC decoding through installed Microsoft extensions, preserving
  primary-image selection, EXIF orientation, metadata and transparency. Keep
  decoder workers hidden and enforce memory, CPU and child-process limits.
- Refresh Explorer test fixtures after the HEIC analysis-facts version change.
- Restore macOS HEIC camera metadata and refresh previously cached image facts
  without repeating similarity inference.

#### Internal

- Close the Windows/Store HEIC pre-release qualification items by maintainer
  decision (2026-09-23). Native x64/ARM64 evidence, installed MSIX behavior,
  codec installation/recheck recovery and final-package qualification are
  accepted as deferred to the maintainer's testing after rollout. This records
  acceptance of the deferral, not successful execution of the outstanding tests.

- Create a three-minute 1080p PicFetch feature promo with original electronic
  music, Trane artwork, 22 animated scenes and verified media output. see the
  [production record](finished_refactorings/2026-09-22-picfetch-promo.md).

- Accept Go build diagnostics in the native qualification runner while still
  rejecting build failures and missing or skipped required tests.

## Open

### Include gio in the Docker test environment

The 2026-09-23 Docker race run found that `scripts/testshards/docker-race.sh`
does not install `libglib2.0-bin`, required by Linux desktop launcher tests.
Both architecture cases failed only because `gio` was missing; installing it
in the disposable container and rerunning `go test -race` for
`./scripts/linuxdesktop` passed. Add the dependency to the Docker test setup;
keep the launch assertions intact. This is separate from the Windows CI codec
exception.

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

- **Retained decoded map tiles (MA-025):** Accepted by the user on 2026-09-09. The map loads only when opened, and
  checking the geolocation of thousands of images is outside expected use. The upstream decoded-tile cache remains
  unbounded; its long-session impact is unmeasured. No further measurement or implementation work is planned.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)
