# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## Open

### Mac App Store preparation

Preserve the full feature set. Active plan: [Mac App Store preparation](plans/2026-09-29-apple-app-store.md).

- [x] Add an immutable Apple Store channel, localized update/repair messages, verified pre-sign runtime staging and a read-only developer-input preflight.
- [x] Preserve captured permissions through image reads, directory scans, and complete session/Favorite serialization; shared folder bookmarks are stored once.
- [ ] Qualify persistent security-scoped permissions for opening, saved sessions, Favorites, writes and workers.
- [x] Implement the Apple Store XPC worker boundary and qualify native fixture TCP/UDP denial, cancellation and broker crash on Apple Silicon.
- [ ] Qualify real HEIC/ONNX execution and granted source/cache access through that worker boundary on both Mac architectures.
- [ ] Bundle pinned native runtimes and notices, and validate signed Mach-O code without comparing its post-sign bytes with upstream archive hashes.
- [ ] Add the signed Store packaging route and validate the final artifact on Intel and Apple Silicon.
- [ ] Complete privacy/dependency/export assessment, listing metadata and Apple submission validation.
- [x] Enable the full Xcode compiler after local license acceptance.
- [ ] Obtain developer Team ID, Store application/installer certificates and provisioning profile for final signed qualification.

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
