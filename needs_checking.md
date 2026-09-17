# PicFetch — Ronin's checks before the next release

- [ ] [Remove JPEG metadata](#jpeg-metadata-removal-acceptance): Try it on copied photos and check that the pictures still look right.
- [ ] [Save and export JPEGs](#jpeg-save-and-export-acceptance): Check that saved photos keep their wanted details and lose old embedded previews.
- [ ] [Try the updater](#updater-acceptance): Check normal updates and how the app handles a stopped or cancelled download.
- [ ] [Check the refactoring](#refactoring-acceptance): Try browsing, visual search and cache settings, then accept the changes.
- [ ] [Refresh GoLand](#image-input-hardening-goland-check): Reload the build settings and check that the AVIF import error is gone.
- [ ] [Finish dependency notices](#dependency-distribution-qualification): Make sure every shipped library and font has the required license information.
- [ ] [Follow up on antivirus reports](#antivirus-review-and-final-artifact-scan): Get the vendors' answers and scan the actual files being released.

These items are still open. Ronin owns acceptance and release decisions; technical
preparation can be delegated. For each completed item, record the tested version,
platform, result and any accepted limits before ticking its box. PR acceptance
follows the required reviews and CI on the latest commit; those agent tasks remain
in [todos.md](todos.md). This list records existing work, not new verification results.

## Details

### JPEG metadata removal acceptance

**Ronin:** Test Remove Metadata on copies of representative JPEGs, including
upright and rotated photos, progressive JPEGs, and photos with color profiles.
Include a file that has removable metadata but no displayed EXIF tags.

Check that Cancel leaves the file unchanged, successful removal refreshes the
information window, and orientation and colors remain correct. Already-clean
files should be reported as clean; unsupported files should explain the refusal
and remain unchanged. Metadata removal does not erase visible private information
or other copies of a photo.

Implementation and local verification are complete; human acceptance is pending.
The implemented contract covers complete scan validation, metadata and preview
removal throughout the file, supported color-profile cleanup, lossless upright
images, memory limits and refusal without rewriting. Native Windows/macOS UI
behavior remains unverified by the Linux tests. Record which desktop platforms
were checked and any remaining platform gap before accepting this for release.

Evidence: [PR #37](https://github.com/frathe/picfetch/pull/37),
[implementation record](plans/2026-09-17-jpeg-metadata-privacy.md),
[supported inputs and limits](docs/jpeg-metadata-removal-qualification.md),
[design decision](docs/adr/0001-jpeg-metadata-removal-refuses-uncertain-input.md),
and [research](docs/jpeg-metadata-removal-research-2026-09-17.md).

### JPEG save and export acceptance

**Ronin:** Use copies of JPEGs with embedded EXIF thumbnails. Edit a photo, use
Save Changes and JPEG export, then reopen the results. Check the picture's
orientation and dimensions, and that the metadata meant to be kept is still
present. Check the embedded thumbnail separately with a metadata viewer or
inspection tool; displaying the main picture alone does not prove its removal.

Valid old EXIF JPEG thumbnail data should be erased. Malformed thumbnail
descriptions are unlinked conservatively without guessing which bytes to erase.
Ordinary save/export preserves unrelated metadata; this is separate from
Remove Metadata above.

Implementation, focused race tests, GoLand inspections and platform CI are
recorded as passing. Complete the fresh Codex review after the last disposition
before Ronin accepts the PR. Keep its plan active until that acceptance.

Evidence: [PR #36](https://github.com/frathe/picfetch/pull/36) and
[active review record](plans/2026-09-17-pr36-review.md).

### Updater acceptance

**Ronin:** After the hosted review round finishes, try the normal update flow
on a disposable installation. Check that the app stays usable and reports a
failure when a download stalls, and that cancelling an update ends cleanly.
Record the platform, starting version and target version used for the check.

The response timeout and GitHub response-size limits have local regression
coverage. The active plan explicitly leaves the PR open for human acceptance;
its latest hosted checks and reviews still need to be assessed by the agent.

Evidence: [PR #38](https://github.com/frathe/picfetch/pull/38) and
[review record](plans/2026-09-17-pr38-review.md).

### Refactoring acceptance

**Ronin:** After the review and CI finish, try switching between ordinary
browsing and visual search, opening Favorites, and changing the analysis-cache
settings. Check that navigation and search still behave as expected, then
record acceptance of the refactoring branch.

Implementation and local verification are complete on `feature/refactoring`.
Fresh Codex code/security reviews, platform CI and assessment of Qodana/CodeQL
results remain agent work. Acceptance, merging and releasing are separate steps;
the review-loop authorization does not itself authorize a merge or release.

Evidence: [review record](finished_refactorings/2026-09-15-refactoring-review.md).

### Image input hardening: GoLand check

**Ronin:** Apply or reload the GoLand module build tags `no_emoji nodynamic`.
Then re-inspect the imaging package's AVIF policy import, or have the agent
repeat that inspection once the IDE has loaded the settings. Record a clean
result before closing the remaining local check; keep the build guard enabled.

Bounded ICO selection, SVG expansion limits, WASM AVIF selection and GIF memory
accounting are implemented. Full `make verify` and both Windows internal-package
cross-builds passed. Hosted Qodana accepted the committed configuration, but
that does not establish that the running IDE loaded the new tags.

Evidence: [PR #27](https://github.com/frathe/picfetch/pull/27) and
[implementation and IDE record](finished_refactorings/2026-09-15-image-input-hardening.md).

### Dependency distribution qualification

**Before release:** Finish the recorded gaps in the shipped dependency inventory:
AVIF native-component notices and an exact libyuv source pin, Fyne font notices,
and reconciliation of the older `x/sys` inventory. Ronin should arrange the
technical follow-up and check the resulting source/version and notice evidence
before treating the release as ready. Verify notice delivery in the actual
packages being shipped, including the final signed Windows packages.

The September 14 audit records AVIF v0.6.0's embedded components, missing font
notices and an older inventory naming `x/sys` 0.47.0 where the updater gate
covered 0.48.0. Reconcile those historical observations with the release build;
do not assume the dated inventory describes its current contents.

The original audit was removed from the checkout on September 15; its
[historical copy](https://github.com/frathe/picfetch/blob/deca58f1f30304991c5b3e8302f7ff5f8d08ae06/docs/find-more-like-this/dependency-qualification.md)
retains the evidence. HEIC support and its decoder were removed at Ronin's
request; see [removal verification](finished_refactorings/2026-09-14-remove-heic-decoder.md).
That removal did not close the remaining AVIF/font/inventory gaps.

September 15 research preferred replacements outside gen2brain. The
[decoder shortlist](docs/image-codec-alternatives-2026-09-15.md) and
[pure-Go evaluation](docs/purego-codec-security-evaluation-2026-09-15.md)
retain the alternatives and blockers. The disposable WASM prototype passed
9/9 scoped isolation checks with Docker providing the outer process limit;
no production desktop sandbox, replacement decoder or new format was qualified.

### Antivirus review and final artifact scan

**Ronin:** Arrange vendor review of the reported Linux detections with Microsoft
and the Windows AMD64 detection with Trapmine. Record their final answers.
Scan the actual final release artifacts after the normal CI build and Windows
signing; results for older, unsigned or different files do not cover those bytes.

The September 11 local builds were assessed as likely false positives, with
vendor confirmation still pending. Microsoft flagged both Linux architectures
as `Trojan:Script/Wacatac.C!ml`; Trapmine flagged Windows AMD64 as
`Malicious.high.ml.score`. Microsoft reported both Windows builds as undetected.
Windows ARM64 was 0/67, but Trapmine could not process that file type. All 99
checked cached dependency directories and archives matched build metadata and
`go.sum`. No samples had been submitted by the agent; these dated results are
not a clean verdict on the next release.

The original investigation was removed from the checkout on September 15.
Its [historical copy](https://github.com/frathe/picfetch/blob/deca58f1f30304991c5b3e8302f7ff5f8d08ae06/docs/antivirus-triage-2026-09-11.md)
contains the exact sample hashes, vendor follow-up steps and limits of the checks.
