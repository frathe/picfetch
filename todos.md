# PicFetch — TODOs

## Done

### What's Changed

#### New Features

##### Find more like this

Choose a picture to find similar-looking images in your collection. The Grid shows your chosen picture first, followed
by up to 30 matches, with the closest matches at the top. Results appear as PicFetch processes your images.

You can return to pictures you searched from earlier, use the usual Grid actions on results, and save a result list as a
Favorite. PicFetch can save what it learns about your images to make future searches faster. Manage this stored data in
Cache settings.

If a scan stops early, PicFetch clearly marks it as incomplete. Missing images stay listed as unavailable, including
pictures on a disconnected drive. After first-time setup, PicFetch checks that your chosen picture is still available
before searching. Turning off saved analysis works even if the stored data cannot be read.

The search was tried with a collection of 446 images, and the results looked useful. Accuracy has not yet been formally
measured. A way to improve matches by showing PicFetch which results you want and which you do not want is planned for
later.

Some requirements for including third-party software still need to be resolved; see LATER below. Read
the [evaluation](docs/find-more-like-this/evaluation.md) for details.

Tests of the affected features and code checks are complete. Final approval still depends on the latest automated
reviews and checks on Linux, Windows and macOS in [PR #25](https://github.com/frathe/picfetch/pull/25).
The [development record](finished_refactorings/2026-09-14-find-more-like-this.md) lists completed checks and remaining
limitations.

#### Bugfix

#### Internal

##### Find more like this architecture follow-up

Improved how visual search works with Favorites, stored search data and the Grid. Fixes cover saving Favorites while
stored data is being checked, keeping search updates responsive, and returning to your original collection after
removing images or closing a comparison. Pictures that appear more than once keep their original positions when you
return.

PicFetch now reports problems saving search data for Favorites while keeping usable results available. Automatic cleanup
removes temporary files before reusable search data and keeps completed analysis available to the current search.

All 28 review findings were assessed.
The [architecture assessment](docs/find-more-like-this/pr25-architecture-review.md) explains the changes, and
the [implementation record](finished_refactorings/2026-09-15-search-ownership.md) documents the fixes and testing.

The fixes and tests are in place, and local checks passed. Final review and automated check results are recorded
on [PR #25](https://github.com/frathe/picfetch/pull/25).

##### Comparison test deadline under build contention

Made the automated image-comparison tests more reliable when the computer is busy with other development work. These
tests check that background image processing can stop safely and that comparison windows do not get stuck.

The updated tests passed repeated runs under heavy load. The full set of checks also passed on Ubuntu, and code
inspection found no issues requiring changes.

##### Qualify updater notices in native release CI

Improved checks that make sure update packages include the required license information for third-party software.

Checks passed on Linux, Windows and macOS. Packages for all six supported targets also passed local checks before
signing. The final Windows packages still need to pass the new checks after they are digitally signed.

The license notes also explain that the exact historical source-code version of one included component could not be
confirmed. See the [evidence and limits](plans/2026-09-13-updater-notices.md) and
the [completed Linux checks](https://github.com/frathe/picfetch/actions/runs/34755687251).

##### Mascot-circle hint for the Hypno Spiral

Added a hidden interaction with Trane and Finis and made the speech bubble wider. Escape now closes the manual even when
you are typing in its search field.

On the welcome screen, move your pointer around Trane's head ten times in the same direction within twenty seconds. This
opens Finis or brings his window to the front.

Draw another ten circles with your pointer within twenty seconds anywhere inside a Finis window. A speech bubble appears
and stays visible. Click it to open the manual with an empty search field. The secret phrase stays in English, while the
hint in parentheses uses your selected language.

Tests of the affected features, translation checks and local build checks passed.
The [development record](finished_refactorings/2026-09-14-mascot-circle-hint.md) includes feedback from hands-on use and
lists checks that could not be completed.

## LATER

### Existing dependency distribution qualification

Resolve the remaining shipped dependency-closure gaps before release: AVIF
native component notices/libyuv pin, Fyne font notices and older x/sys inventory.
HEIC support and its decoder dependency were removed at Ronin's request on
September 14, pending distribution qualification. See the
[audit and current disposition](docs/find-more-like-this/dependency-qualification.md)
and [removal verification](plans/2026-09-14-remove-heic-decoder.md).

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
