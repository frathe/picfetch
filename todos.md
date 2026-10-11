# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- [x] Mac App Store release: PicFetch is available on the Mac App Store as a
  sandboxed universal app with the full feature set, persistent folder access
  and isolated HEIC/analysis workers (174 countries/regions; France not yet).
  See [plan and evidence](finished_refactorings/2026-09-29-apple-app-store.md).

- [x] Add `make apple-store-package` for a fresh Intel/Apple Silicon build,
  Store signing and installer validation in one command. Local configuration
  is gitignored; environment overrides remain supported. Orchestration tests
  cover ordering, fresh paths, missing inputs and stopping after build failure.
  Full native execution of the new wrapper remains unverified; existing build
  and signing stages retain their separate qualification evidence.

- [x] Fixed screenshot window mode: four console-selected physical pixel sizes,
  constrained mode transitions and preserved saved geometry. Native measurements,
  actual 1280 x 800 captures, fresh reviews/CI and App Store test build 483 pass.
  See [plan and evidence](finished_refactorings/2026-10-01-fixed-size-mode.md).

#### Bugfix

- [x] Correct the App Store in-app version from 0.0.1 to 1.1.11 by embedding
  Fyne metadata in both native executable slices. New candidate build 484 is
  signed; runtime regression, native About text, bundle/installer guards,
  verify-build and GoLand inspections pass. b69171b has clean code/security
  reviews, full CI, CodeQL/FOSSA and no actionable Qodana findings. Final
  documentation-head gates and candidate delivery are recorded on PR #75. See
  [implementation and qualification](finished_refactorings/2026-10-03-store-runtime-version.md).

- [x] Fix Store installer permissions rejected by Transporter (90255).
  Staged permission normalization and extracted-payload readability guards pass
  regression tests and the real signed installer roundtrip. Ronin reported
  Transporter Verify passing on 2026-10-01 after the corrected candidate was
  supplied. No delivery or App Review submission occurred. See
  `packaging/apple-app-store/ronin-checklist.md` for evidence and candidate path.

- [x] Restore Copy Image in the sandboxed Mac build with direct in-memory AppKit
  PNG publication. Menu Copy Image and Cmd+C produce correct images in Preview,
  including a distinct second image and paste after PicFetch exits. Clipboard
  race/image-selection regressions, vet/build and changed-file inspections pass.
  Universal Store test build 483 is refreshed at source 01b5a18. Latest-head PR
  review/CI evidence is tracked separately in PR #75. See the
  [implementation and native evidence](finished_refactorings/2026-10-01-store-clipboard-image.md).

- [x] Complete PR #75 source follow-ups: all fixes are committed/pushed and
  all 20 threads have dispositions. Fresh code/security-focused reviews, full
  hosted CI, CodeQL and assessed Qodana pass at 3453f1b. Universal Store test
  build 483 is refreshed and its 24 payload hashes verified. Native folder
  approval/button, sibling browsing and grant persistence after full relaunch
  pass with Ronin's explicit test-folder permission. Original cancellation/
  capture qualification remains source-specific.
  Final documentation-head review/checks are tracked in PR #75.
  See [current qualification](finished_refactorings/2026-10-01-metadata-source-binding.md).

- [x] Preserve unsuccessful termination when SIGTERM arrives during animated
  help restoration; independent signal buffers protect SIGTERM from repeated
  Ctrl+C skips. Focused races, negative
  cleanup guard, executable PTY checks and GoLand inspections pass. Latest
  hosted review evidence is tracked in PR #75 and the screenshot record.

- [x] Bind confirmed metadata removal to inspected file identity and contents;
  bind exported metadata to displayed source bytes, including cache and Save.
  Local regressions/race/Store checks and fresh code/security/CI gates pass.
  See [source-binding evidence](finished_refactorings/2026-10-01-metadata-source-binding.md).

- [x] Restore Store folder consent: native panels complete without blocking the
  Fyne dispatcher; moved file bookmarks suggest the current parent. Exact-package
  approval/button, cancel, grant reuse/relaunch and fresh hosted checks pass.
  See [regression evidence](finished_refactorings/2026-10-01-folder-access-regression.md).

- [x] Bound Save Changes and JPEG Export secondary source reads by the encoded
  input limit, including growth after stat. Save leaves oversized sources intact;
  Export preserves pixel-only fallback. Local checks, full hosted CI and CodeQL
  pass; Qodana has no actionable findings and fresh code reviews are clean.
  Accepted as finished by Ronin; separate security-review confirmation is
  recorded in the [verification record](finished_refactorings/2026-09-30-secondary-jpeg-read-budget.md).

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

### App Store follow-ups

- [ ] Automate Mac App Store releases like the Microsoft Store route
  (`microsoft-store.yml` -> `microsoft-store-publish.yml`): a release tag builds
  and qualifies the universal Store package, signing runs in the
  `apple-store-signing` environment only after Ronin's manual approval, and an
  approved publish job uploads the signed installer to App Store Connect and
  prepares the TestFlight build/version. Keep App Review submission and release
  as explicit approved modes. First add the documented Apple CI
  secrets/variables and qualify the first approved hosted run of
  `apple-store.yml` now that it is on main.

- [ ] Enable France after Apple encryption case **102982575995** is answered.
  France is currently excluded from availability (standard-crypto/No-France
  export answers). Confirm whether France needs separate TestFlight
  restrictions or encryption documentation before adding it.

- [ ] Qualify the Store build on a physical Intel Mac and on the minimum
  supported macOS. Intel is so far qualified under Rosetta only.

- [ ] Qualify security-scoped bookmark reuse after moving a granted folder to
  another parent or volume and relaunching (renaming in place passes).

## Deferred

### Missing-file toasts for disconnected drives

- [ ] Improve repeated raw missing-file errors when opening a Favorite on a
  disconnected external drive. Explicitly deferred by Ronin on 2026-10-01;
  this is not a release blocker. Ronin considers the behavior acceptably quick
  even for large collections and reports no problematic memory use (not an
  instrumented measurement). Build 483 stays responsive, and reopening the
  Favorite after reconnecting restores images and navigation. See the
  [acceptance record](packaging/apple-app-store/ronin-checklist.md#6-finish-the-remaining-hands-on-checks).

### Recursive scan work budget

- [ ] Count examined files/directories and batch directory enumeration, including
  rejected files and empty trees. Codex priority advice (PR #75 thread 4148307910)
  treats this as deferrable availability hardening of user-selected trees;
  image admission limits, cancellation and cycle protection already remain.

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
