# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Add Settings -> Limits -> Similarity Explorer with persisted 512 MB map-data
  and 10,000-item defaults, configurable for larger libraries. Resource-limit
  errors show wrapped toasts naming the setting location. Stream map snapshots
  in bounded chunks with aggregate accounting instead of one large JSON event.
  Evidence: [PR #29 implementation plan](plans/2026-09-16-explorer-configurable-limits.md).

#### Bugfix

- Reject Save Changes through symlink leaves, preserving both the link and target
  without copying image content into a different directory. Keep regular writes
  at the selected filename and serialize parent-directory aliases. Verification:
  [PR #35](https://github.com/frathe/picfetch/pull/35),
  [review record](finished_refactorings/2026-09-17-pr35-review.md).

- Authenticate staged updates with process-local seals, derive payload hashes
  from verified archive bytes, and retain a write-denying Windows source handle
  through installation. Native Windows guards pass; review and final CI evidence:
  [PR #32](https://github.com/frathe/picfetch/pull/32),
  [review record](finished_refactorings/2026-09-16-pr32-review.md).

- Bound ICO and SVG input processing, enforce WASM AVIF builds, and account for
  GIF frame overhead before animation decoding; preserve static GIF fallback.
- Bound comparison decodes to half the shared image budget per pane and avoid
  retaining animation frames that comparison never displays. Include 16-bit
  pixel admission, actual cached-frame weights, localized refusals and the
  corrected UI shard assignment. Hosted review dispositions and final CI,
  Qodana and CodeQL evidence: [PR #31](https://github.com/frathe/picfetch/pull/31).
- Bound Visual Similarity Explorer collection work and worker event decoding to
  prevent attacker-controlled collections from exhausting CPU or viewer memory.

#### Internal

- Pin the TUF-root workflow's checkout/setup-go actions to verified v7 commits,
  disable checkout credential persistence, and configure GitHub CLI authentication
  before pushing. Focused TUF tests and stubbed publication paths pass; the live
  scheduled write path is not exercised by PR CI. Review dispositions and final
  hosted checks are tracked in [PR #34](https://github.com/frathe/picfetch/pull/34).
- Pin the Qodana action to its reviewed commit, remove source-write permission
  and automatic fix pushes, and disable persisted checkout credentials. Retain
  PR comments, annotations and the Go linter's required project token. Review
  dispositions and hosted verification: [PR #33](https://github.com/frathe/picfetch/pull/33).
- *Windows signing qualification*
  Completed. The Windows release-signing workflow uses the protected
  `release-signing` environment, retains the verified local installer handoff,
  and checks `SIMPLYSIGN_INSTALLER_SHA256` and `SIMPLYSIGN_SIGNER_THUMBPRINT`
  before installation. Setup and test-tag qualification are described in
  [the signing guide](docs/release-signing.md). Fixes, hosted review dispositions
  and CI evidence are tracked in
  [PR #30](https://github.com/frathe/picfetch/pull/30); PR CI does not execute the
  credentialed signing job.


## Open

### Private Favorite storage review and acceptance

PR #39 makes new and re-saved Favorite lists private and isolates the temporary
fallback. Legacy replacement, fallback failure/isolation, and early startup
failure regressions, focused race suites, GoLand inspections and local build
checks pass. Complete the fresh Codex code/security
reviews and platform CI; hosted dispositions and final checks belong on
[PR #39](https://github.com/frathe/picfetch/pull/39).
Scope, limits and local evidence: [review record](plans/2026-09-17-pr39-review.md).

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
is closed by removal. Retained [options and source evidence](docs/find-more-like-this/old-heic-wasm-options.md)
do not authorize restoring a decoder.

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
