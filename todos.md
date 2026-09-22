# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Added Help -> Licenses, showing the release's embedded third-party notices
  as Markdown without an internet connection, including Store builds.

#### Bugfix

- License blocks now wrap and share the document's scroll surface, so mouse
  wheel scrolling works over the text as well as the background.

#### Internal

- Expanded the AVIF/native/WASI notices with full reviewed license, patent and
  attribution texts. Added pinned source/payload checks and release-archive
  guards for both loose notices and the executable's embedded document.

## Open

### Native Windows similarity-cache tests

The full `internal/similarity` suite has three failures on Windows that also
reproduce with the pre-fix search implementation: permission assumptions in
`TestAnalysisCacheMaintenancePartialFailure`, URI/native path comparison in
`TestAnalysisCacheConfinementManagedUsageAndTemps`, and open-directory rename
sharing in `TestFavoriteAnalysisFollowsOpenedDirectory`. Qualify these separately;
the Windows search tests and production search/cache integration pass. See
[the regression evidence](plans/2026-09-19-windows-visual-search-paths.md).

### Finish AVIF notice release qualification

Implementation and local evidence are recorded in
[the plan](plans/2026-09-21-avif-license-viewer.md). Remaining before release:

- Obtain upstream evidence for the historical libyuv checkout and modified
  WASI SDK/libc inputs used in the pinned AVIF payload. The source texts are
  retained, but the floating libyuv branch and SDK `33.0+m` do not establish
  exact historical source correspondence; see
  [provenance limits](scripts/avifnotices/README.md#provenance-limits).
- Verify native offline Licenses UI on Windows/Linux, and final signed Store
  MSIX/bundle plus WACK. Unsigned GitHub archives for both architectures and
  Store executable payloads passed local notice checks; actual MSIX/bundle
  structure has automated fixture coverage, not a signed local build.
- Recheck all final release artifacts against their exact dependency/payload
  versions. AVIF, ONNX and other bundled native/WASM updates must include
  corresponding notice updates in the same change.

## Deferred

### Fyne upgrade deferred

Keep Fyne at v2.8.0 in [PR #19](https://github.com/frathe/picfetch/pull/19).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
The four grouped `golang.org/x/*` updates remain in the PR.

### WinGet package identifier migration

Deferred by Ronin on September 13. Keep `io.github.frathe.picfetch` for now;
the rename to `frathe.picfetch` is not a next-release requirement. Resume only
after Ronin chooses to proceed, following the inventory, Windows upgrade tests
and publication steps in [the migration plan](docs/winget-package-id-migration.md).
The [maintainer's suggestion](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5639706559)
remains background for that deferred work.

### Reconsider HEIC support after licensing and security qualification

HEIC/HEIF is currently unsupported. A future restoration needs a documented
distribution grant or another qualified decoder, containment and platform
verification. The previous fork-upgrade watch [MA-023](needs_refactoring.md#ma-023)
is closed by removal.

The [independent alternatives](docs/image-codec-alternatives-2026-09-15.md#heic-alternatives)
include libheif/libde265 with LGPL distribution work and hpvcd with unresolved
table provenance and security/platform qualification. Ronin prefers avoiding
gen2brain replacements. The authorized review of h265 v0.2.3 found an invalid
result invariant, incomplete translated-source provenance and unresolved HEVC
patent obligations. Its still decoder can also fall back to sequence decoding,
so a future adapter must reject sequences explicitly. The local fix does not
qualify the library; HEIC remains disabled.

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
