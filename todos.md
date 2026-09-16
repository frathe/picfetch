# PicFetch — TODOs

## Open

### Image input hardening

All four safeguards are implemented: bounded ICO selection, SVG expansion limits,
enforced WASM AVIF selection, and GIF memory accounting. Full `make verify` and
both Windows internal-package cross-builds pass. Hosted review, finding
dispositions and final CI/Qodana/CodeQL evidence are tracked in
[PR #27](https://github.com/frathe/picfetch/pull/27). The local `no_emoji nodynamic` module tags were applied in GoLand on
2026-09-16 and the imaging loader re-inspects clear; the committed Qodana
configuration passed hosted analysis.
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

HEIC/HEIF remains default-off. Experimental activation is being implemented
under the accepted September 16 spec; distribution clearance, containment and
platform verification remain required. The previous fork-upgrade watch [MA-023](needs_refactoring.md#ma-023)
is closed by removal. Retained [options and source evidence](docs/find-more-like-this/old-heic-wasm-options.md)
do not authorize restoring a decoder.

Restoration work is authorized and tracked in
[the active plan](plans/2026-09-15-isolated-heic-restoration.md) and
[draft PR #28](https://github.com/frathe/picfetch/pull/28). The signed `52ed2df`
checkpoint publishes the maintained-source WASI guest, native helper boundaries,
shared GUI/analysis admission, canonical imaging integration and authenticated
package/update handling. The current opt-in work now constructs an owner only
after an enabled preference and validated installed package.

- [ ] Complete the [Experimental HEIC opt-in spec](.scratch/experimental-heic-opt-in/spec.md):
  default-off, restart-only activation, immutable session admission and verified
  private Windows helper staging are integrated. The [handoff](.scratch/experimental-heic-opt-in/handoff.md)
  records 48/57 completed checklist items; tickets 01/02/03/04/06/07 are resolved.
  At `c1b6890`, all six standalone native platform/architecture targets, all four
  Linux race partitions, validation, ordinary Windows and Store input construction
  pass on first attempt. New evidence covers uncached duplicate/Spiral pixels,
  active-analysis Settings/source replacement, native directory navigation,
  foreground cancellation, invalid-package recovery and macOS readiness refusal.
  GoLand, local build/provenance/import checks, shard inventory and dependency
  scans pass. Fresh Codex code/security reviews are clean; final Qodana has zero
  results and CodeQL only its two previously dismissed false positives.
  Remaining: Windows application-level private-storage/ACL and failed-query
  recovery plus concurrent application lifetimes (08), installed-MSIX activation
  (05), and final integrated qualification (09). Both MSIX packages install but
  their test processes failed to start with Access denied through `7f5c1da`.
  The CI repair tests a fresh Users-only logon of the actual desktop owner, with
  SID/session and token checks; both native results remain required.
  The separate GitHub AI scanner
  still fails before analysis with an unsupported-model HTTP 400. Keep all failing
  gates and permission checks. The [active plan](plans/2026-09-16-experimental-heic-opt-in.md)
  and [activation record](docs/heic/experimental-opt-in.md) retain exact evidence.
  The earlier ARM64 analysis failure remains unexplained despite the fresh pass.
  Production signing, licensing/distribution, broader camera/color qualification,
  the accepted macOS total-memory limit and the full-reinstallation/qualified-
  bridge requirement remain separate release constraints.

The September 16 [history reconciliation](docs/heic/history-reconciliation.md)
accounts for `fc127b44` and `73cb3c9`: all 107 production/source-license files
were preserved byte for byte. It restores the current app-wide threat model,
Qodana YAML guard, four ordinary fixtures and Fyne metadata ignore rule. The
historical eager WASM allocation option was withdrawn after fresh native Linux
CI failed; existing memory ceilings and sandbox policy remain intact. Focused race/interpreter tests, native Apple Silicon
guards, build/provenance/import checks and GoLand inspections pass. No complete
historical decoder-suite equivalence is claimed.

Remaining qualification:

- At code checkpoint `ee5cc67`, all hosted CI passes: every Linux race shard,
  Linux and Windows amd64/arm64 helper guards, macOS Intel/Apple Silicon native
  guards, ordinary Windows and validation. The signing-job security finding is
  fixed in `c77d9bd` and resolved; subsequent code/security rounds are clean.
  Qodana and CodeQL checks pass. Final evidence and review of the documentation
  follow-up are tracked in [PR #28](https://github.com/frathe/picfetch/pull/28)
  and the active plan. No broad local race suite was duplicated.
- Ronin accepted the absent hard total native-memory cap on macOS. WASM,
  transport, deadline and sandbox controls remain mandatory; helper-only
  executable-memory permission is required. The compiler succeeds on the
  ordinary 12MP fixture; the interpreter reaches the finite deadline.
- Complete distribution-package, Windows Authenticode/MSIX and upgrade
  qualification remain open. Nonadministrator launch-time loopback queries pass
  in standalone Windows native CI on both architectures; installed-MSIX execution
  is still unqualified. The released updater drops the new helper and
  deletes staging, leaving an invalid macOS enclosing signature. The first
  transition requires a complete reinstall or a separately qualified bridge.
- [Compatibility evidence](docs/heic/compatibility-2026-09-16.md) covers ordinary
  synthetic metadata/orientation, alpha and ten-bit transport plus one verified
  Samsung S23 Ultra photo. Broader camera coverage remains open. Faithful
  ICC/wide-gamut/PQ/HLG/gain-map display is absent and those color classes are
  not deliberately rejected; more fixtures alone cannot establish it.
- The initial cloud foundation commit is unsigned; the PR's signing prerequisite
  remains open. No history rewrite, merge or release is
  authorized by this reconciliation.

See [qualification evidence](docs/heic/qualification.md) and the current
[threat model](THREAT-MODEL.md) for controls and accepted limitations.

The [independent alternatives](docs/image-codec-alternatives-2026-09-15.md#heic-alternatives)
include libheif/libde265 with LGPL distribution work and hpvcd with unresolved
table provenance and security/platform qualification. Ronin prefers avoiding
gen2brain replacements. The authorized review of h265 v0.2.3 found an invalid
result invariant, incomplete translated-source provenance and unresolved HEVC
patent obligations. Its still decoder can also fall back to sequence decoding,
so a future adapter must reject sequences explicitly. The local fix does not
qualify the library; experimental activation is tracked separately above.

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
