# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- [x] Bound Save Changes and JPEG Export secondary source reads by the encoded
  input limit, including growth after stat. Save leaves oversized sources intact;
  Export preserves pixel-only fallback. Local race, Store-tagged and static
  checks pass; hosted qualification is pending on PR #75. See the
  [verification record](plans/2026-09-30-secondary-jpeg-read-budget.md).

- [x] Preserve broker cancellation during signal-source startup and retain a
  nonzero exit status when SIGTERM interrupts animated terminal help. Native
  XPC and terminal regressions pass; fresh hosted review evidence is retained
  on PR #75. See the [verification record](plans/2026-09-30-review-cancellation.md).

- [x] Keep Windows Export/Trash filenames out of PowerShell source. Qualified
  with native Windows regression events, the full Linux/amd64 race suite,
  macOS checks, CodeQL, reviewed Qodana SARIF and clean code/security reviews
  on `bb6ce5d`. See the [verification record](finished_refactorings/2026-09-30-windows-path-transport.md);
  final documentation-head review/CI evidence is retained on PR #75.

#### Internal

- [x] Pin every action in the write-enabled release publication job to a
  reviewed full commit. Preserve publisher inputs and signing gates; semantic
  regressions reject mutable refs. Local validation and provenance are in the
  [verification record](plans/2026-09-30-release-action-pins.md); hosted results
  are retained on PR #75.

- [x] Restrict release-workflow contents write to publication and disable
  credential persistence on release checkouts. Semantic policy guards preserve
  signing approval and reject permission/credential regressions. Local checks
  pass; final hosted review/CI evidence is retained on PR #75 and linked from
  the [verification record](plans/2026-09-30-release-permissions.md).

- [x] Add a three-second ASCII Trane turn and PicFetch wordmark to terminal
  `--help`, with plain piped output and a `make trane` developer preview.
  Verification and limitations: [turntable record](finished_refactorings/2026-09-30-trane-ascii.md).

## Open

### Mac App Store preparation

Preserve the full feature set. Active plan: [Mac App Store preparation](plans/2026-09-29-apple-app-store.md).

- [x] Add an immutable Apple Store channel, localized update/repair messages, verified pre-sign runtime staging and a read-only developer-input preflight.
- [x] Preserve captured permissions through image reads, directory scans, and complete session/Favorite serialization; shared folder bookmarks are stored once.
- [x] Carry native Open With/Dock selections through worker-side bookmark capture, cancellation and shutdown; decode native URL escapes once. Signed production opening qualification remains pending.
- [x] Own native save destinations through export/mosaic cancellation, scope image mutations and metadata reads, and stage Apple Store atomic writes in a same-volume replacement directory.
- [ ] Qualify persistent security-scoped permissions for opening, saved sessions, Favorites, writes and workers.
- [x] Implement the Apple Store XPC worker boundary and qualify native fixture TCP/UDP denial, cancellation and broker crash on Apple Silicon.
- [x] Qualify real HEIC/ONNX execution and granted private source/cache access through that worker boundary on ARM and Intel under Rosetta.
- [x] Validate post-sign Apple runtime code and outer bundle seals, use a shared Frameworks path for the app and worker, and qualify real arm64 CPU inference in a network-disabled sandbox fixture.
- [x] Bundle both pinned native runtimes and notices in a universal ad-hoc app; validate signed Mach-O code, architecture/minimum OS, dependencies and nested/outer seals.
- [x] Transfer exact source/model/cache and app-signature access to analysis workers; qualify real HEIC/ONNX, private source/cache access and reuse on ARM and Intel under Rosetta.
- [x] Qualify retained visual search through production ARM/Intel-Rosetta workers; keep scopes through clipboard/Finder/Trash calls and source metadata reads.
- [x] Capture native window-drop URLs before GLFW flattens paths; refresh restored moved/stale bookmarks with immutable persistence coverage.
- [x] Verify production workers deny an ungranted private source while an allowed control image still completes inference, on ARM and Intel under Rosetta.
- [x] Add explicit Store Open-dialog folder consent for single-image sibling navigation; preserve cancellation and the initially selected image. Latest E2E app is in `bin/apple-store-e2e-review-fixes/`.
- [x] Ronin confirmed native Open-dialog folder consent fixes navigation; extend the same consent to shared drop/Open With discovery with cancellation and replacement coverage.
- [ ] Confirm the rebuilt shared-consent app with a single-file window drop and Open With.
- [x] Ronin confirmed multiple-file picker/drop navigation must remain within the selected images; folder sibling discovery applies only to single-file inputs. Automated selection coverage passes.
- [x] Persist explicit sibling-folder approvals in app preferences and validate/reuse bookmarks on fresh single-image opens; keep renewal, cancellation and selection guards.
- [x] Ronin confirmed persistent folder approvals work across real quit/relaunch in `bin/apple-store-e2e-persistent-folders/PicFetch.app`.
- [ ] Confirm moved-folder bookmark reuse in the persistent-folder test app; automated coverage passes, live moved-folder testing remains open.
- [x] Fix PR #75 findings: acquire source authority for metadata sorting and map analysis/search file I/O to resolved bookmark locations while preserving collection/cache identities.
- [x] Complete the application/CI review fixes in [PR #75](https://github.com/frathe/picfetch/pull/75): `0a670e6` has fresh clean code/security reviews, passing platform/race CI and CodeQL, and no actionable local/Qodana findings. All three threads are resolved. Final documentation-commit review/CI evidence lives on the PR.
- [x] Validate copyright/category metadata and reject quarantine attributes in local Store packages.
- [ ] Finish GUI sandbox workflows, moved-folder relaunch, and physical Intel/minimum-OS validation.
- [x] Implement a separate Store distribution-signing/installer route with identity, profile and payload guards.
- [x] Prepare a main-only CI candidate signing workflow with a separate temporary-Keychain signing job and Ronin's required approval environment.
- [ ] Add the documented Apple CI environment secrets/variables and qualify the first approved hosted signing run after the workflow reaches main.
- [ ] Prepare App Store Connect upload/TestFlight and release automation after signed-candidate and SDK/privacy qualification; keep submission/release decisions separate.
- [ ] Run the signed route with real Apple certificates/profiles and validate the resulting candidate on the required distribution path and platform matrix.
- [x] Prepare [listing/review notes](packaging/apple-app-store/submission-draft.md) and [ordered owner checklist](packaging/apple-app-store/ronin-checklist.md); inventory privacy/network/export facts.
- [x] Preserve exact Abseil privacy manifests/license/provenance; verify original Microsoft runtime signatures before local re-signing.
- [ ] Resolve remaining [Protobuf/ONNX privacy coverage and SDK validation](packaging/apple-app-store/privacy-dependency-audit.md), confirm final privacy/export answers and validate Apple submission.
- [x] Enable the full Xcode compiler after local license acceptance.
- [ ] Human inputs last: developer Team ID, Store application/installer certificates, separate app/worker TestFlight provisioning profiles, App Store Connect record and final submission decisions.
- [x] Signed and pushed `62661a1`; full platform/race CI and CodeQL passed. Fresh Qodana SARIF has eight reviewed false positives and no actionable findings.
- [x] Ronin returned from lunch on 2026-09-30 and explicitly reauthorized committing and pushing the current state.

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
