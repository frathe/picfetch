# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- Enable Windows HEIC decoding through installed Microsoft extensions, preserving
  primary-image selection, EXIF orientation, metadata and transparency. Keep
  decoder workers hidden and enforce memory, CPU and child-process limits.
- Refresh Explorer test fixtures after the HEIC analysis-facts version change.
- Restore macOS HEIC camera metadata and refresh previously cached image facts
  without repeating similarity inference.

#### Internal

- Create a three-minute 1080p PicFetch feature promo with original electronic
  music, Trane artwork, 22 animated scenes and verified media output. see the
  [production record](finished_refactorings/2026-09-22-picfetch-promo.md).

- Accept Go build diagnostics in the native qualification runner while still
  rejecting build failures and missing or skipped required tests.

## Open

### Complete HEIC PR #50 review loop

[PR #50](https://github.com/frathe/picfetch/pull/50) is open for the system HEIC
implementation. Resolve confirmed review/CI findings and obtain fresh clean
Codex code/security, Qodana/CodeQL and complete CI results on the final head.
The [implementation plan](plans/2026-09-22-system-heic.md) records evidence;
the user's functional testing covers Windows, macOS and Linux. Remaining
qualification items below are not implicitly closed by opening the PR.
The initial three Codex findings have focused regression fixes. Windows cache
regressions now pass in hosted CI; missing Microsoft codecs still prevent its
native HEIC qualification. macOS premultiplied-alpha rendering remains incorrect;
Intel nested-sandbox startup and HEIC analysis now pass on both macOS
architectures. A further regression fixes retained ordering when merge mode
repeats a visible source. Final clean code/security review and complete CI
remain open.

### delegate heic image rendering to the OS

Add back support for HEIC image formats, on start check if the system supports rendering of heic images. if yes save that
information to the settings so we don't have to check that on every launch. when the os does support the rendering we
delegate the rendering to the OS and enable HEIC support.

Design agreed in [the HEIC system-decoder specification](docs/heic-system-decoding.md)
and [ADR 0002](docs/adr/0002-system-provided-heic-decoding.md): include macOS,
Windows and Linux, integrate all existing image consumers, and add Settings
buttons for a support check and the current OS's Markdown installation guide.
Linux uses installed libheif with an HEVC decoder. Implementation is in progress
under [the Deep plan](plans/2026-09-22-system-heic.md), with delegated tickets and
lead-owned review. Linux native slices pass, including ICC-tagged photos;
color correction is best effort and delegated to the system decoder per the
2026-09-22 clarification. Full feature qualification remains
open. Native Apple Silicon verification now passes viewing, clipboard encoding,
PNG/JPEG export, mosaics, retained search, EXIF delivery and cached-fact repair.
ImageIO still reports no images for the two authored premultiplied-alpha
fixtures; those required native tests remain failing. The Go build-diagnostic
handling in the native runner is fixed. See the plan's macOS record for that
host's ARM64 Docker limitation.
Windows 11/amd64 now passes the real HEIC corpus, including primary-order variants,
grids, mirrors and straight/premultiplied alpha. Windows ARM64, older extensions,
packaged opens and Store deployment remain unverified; see the Windows follow-up
record in the plan for final checks and broader native-suite failures.
Windows HEIC focused race tests and native/Linux build checks pass. The
user-authorized 20 GiB WSL memory cap is active after the approved restart;
the full Linux/amd64 race suite completed with the unchanged 16 GiB limit.
Its only failures were stale Explorer facts fixtures; those were corrected,
and all affected tests pass focused race reruns on both Windows and Linux.
No release is qualified.
The implementation spec is published in the local issue tracker at
`.scratch/os-heic/spec.md`; ticket/evidence records are under `.scratch/os-heic/`.

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

The system-provided decoder route is now active under the open HEIC item above.
Bundled decoder alternatives below remain deferred.

Restoring a bundled decoder still needs a documented distribution grant,
containment and platform verification. The previous fork-upgrade watch [MA-023](needs_refactoring.md#ma-023)
is closed by removal.

The [independent alternatives](docs/image-codec-alternatives-2026-09-15.md#heic-alternatives)
include libheif/libde265 with LGPL distribution work and hpvcd with unresolved
table provenance and security/platform qualification. Ronin prefers avoiding
gen2brain replacements. The authorized review of h265 v0.2.3 found an invalid
result invariant, incomplete translated-source provenance and unresolved HEVC
patent obligations. Its still decoder can also fall back to sequence decoding,
so a future adapter must reject sequences explicitly. The local fix does not
qualify that library; the bundled HEIC decoder remains removed.

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
