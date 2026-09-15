# PicFetch — TODOs

## Open

### Image input hardening

All four safeguards are implemented: bounded ICO selection, SVG expansion limits,
enforced WASM AVIF selection, and GIF memory accounting. Full `make verify` and
both Windows internal-package cross-builds pass. Hosted review, finding
dispositions and final CI/Qodana/CodeQL evidence are tracked in
[PR #27](https://github.com/frathe/picfetch/pull/27). The live GoLand build-tag
refresh and re-inspection remain pending; the committed Qodana configuration
passed hosted analysis. Reload/apply the local `no_emoji nodynamic` module tags
and inspect the imaging import again before closing this remaining local task.
Compatibility limits and evidence:
[implementation plan](finished_refactorings/2026-09-15-image-input-hardening.md).

### Refactoring follow-up review

Implementation and local verification are complete on `feature/refactoring`.
Run fresh Codex code/security reviews and platform CI, assess all findings and
inspect the Qodana/CodeQL reports before acceptance. The user authorized the
GitHub review loop; merging and releasing remain separate actions.
Contracts and evidence workflow: [review record](finished_refactorings/2026-09-15-refactoring-review.md).

## Done

### What's Changed

#### New Features

#### Bugfix

- Bound ICO and SVG input processing, enforce WASM AVIF builds, and account for
  GIF frame overhead before animation decoding; preserve static GIF fallback.

#### Internal

## LATER

### Existing dependency distribution qualification

Resolve the remaining shipped dependency-closure gaps before release: AVIF
native component notices/libyuv pin, Fyne font notices and older x/sys inventory.
HEIC support and its decoder dependency were removed at Ronin's request on
September 14, pending distribution qualification. See the
[audit and current disposition](docs/find-more-like-this/dependency-qualification.md)
and [removal verification](finished_refactorings/2026-09-14-remove-heic-decoder.md).

September 15 research: prefer replacements outside gen2brain. The
[decoder shortlist](docs/image-codec-alternatives-2026-09-15.md) compares direct
libavif builds, HEIC alternatives and small extra formats with their complete
dependency considerations. A later, explicitly authorized
[pure-Go security evaluation](docs/purego-codec-security-evaluation-2026-09-15.md)
found blockers in exact h265 and gav1d snapshots. Its disposable WASM prototype
passed 9/9 scoped isolation checks, with Docker supplying the outer process
limit; no production desktop sandbox was proven. No replacement or new format
is implemented or qualified.

### Fyne upgrade deferred

Keep Fyne at v2.8.0 in [PR #19](https://github.com/frathe/picfetch/pull/19).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
The four grouped `golang.org/x/*` updates remain in the PR.

### Antivirus verdicts on unreleased builds

The September 11 local builds have likely false positives: Microsoft flags both
Linux architectures as `Trojan:Script/Wacatac.C!ml`; Trapmine alone flags Windows
AMD64 as `Malicious.high.ml.score`. Microsoft reports both Windows builds as
undetected. Windows ARM64 is 0/67, but Trapmine cannot process that file type.
All 99 distinct cached dependency directories and archives match the build
metadata and `go.sum`. Vendor review remains pending; no samples have been
submitted by the agent. Record final vendor determinations and rescan the final
release artifacts. See [the investigation](docs/antivirus-triage-2026-09-11.md).

### WinGet package identifier migration

Move the existing `io.github.frathe.picfetch` manifests in `microsoft/winget-pkgs` to the
new identifier `frathe.picfetch` before the next WinGet publication; See the
[maintainer's suggestion](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5639706559).

### Reconsider HEIC support after licensing and security qualification

HEIC/HEIF is currently unsupported. A future restoration needs a documented
distribution grant or another qualified decoder, containment and platform
verification. The previous fork-upgrade watch [MA-023](needs_refactoring.md#ma-023)
is closed by removal. Retained [options and source evidence](docs/find-more-like-this/old-heic-wasm-options.md)
do not authorize restoring a decoder.

Restoration work is now authorized and tracked in
`plans/2026-09-15-isolated-heic-restoration.md`. The bounded parent protocol and
safe-testing boundary are present, but decoding and format exposure remain
disabled until the exact h265 source/guest closure can be retrieved and reviewed
and mandatory native isolation is runtime-qualified. The Codex Cloud checkout
cannot currently retrieve the candidate source (module proxy HTTP 403) or supply
Windows/macOS/native Linux isolation evidence.

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

- Windows releases are not Authenticode-signed. Controlled Folder Access and SmartScreen both judge by signature and
  reputation as well as by which program is writing, so an unsigned `picfetch.exe` can still be blocked even with the
  in-process swap (see Done → Bugfix above, where the block would now name `picfetch.exe` instead of `cmd.exe`). The
  real remaining fix is signing the Windows release build — Azure Trusted Signing or a purchased certificate — in
  `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)
