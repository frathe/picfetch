# PicFetch — TODOs

## Done

### What's Changed

#### New Features

### Find more like this

Choose a picture to find visually similar images in your collection. The Grid
shows your chosen picture first, followed by up to 30 matches, with the closest
matches at the top. Results appear as images are prepared.

You can revisit previously chosen reference pictures, use the usual Grid actions
on results, and save the current result list as a Favorite. PicFetch can remember
image analysis between sessions to speed up future searches. Manage this stored
data in Cache settings.

Interrupted scans are clearly marked as incomplete. Images that cannot be found
remain listed as unavailable, including those that may be on a disconnected drive.
After first-time setup, PicFetch checks that your chosen picture is still available
before starting the search. Turning off stored analysis remains effective even
if PicFetch cannot read the cache.

Results were reviewed using a collection of 446 images and judged satisfactory,
although search accuracy has not been formally measured. Teaching the search
through examples of wanted and unwanted results is planned for later.
Some third-party software distribution requirements also remain open; see LATER
below. Read the [evaluation](docs/find-more-like-this/evaluation.md) for details.

Targeted tests and code checks have been completed. Final approval still depends
on the latest automated reviews and Linux, Windows and macOS checks in
[PR #25](https://github.com/frathe/picfetch/pull/25). The
[development record](finished_refactorings/2026-09-14-find-more-like-this.md)
documents completed checks and remaining limitations.

#### Bugfix

#### Internal

### Comparison test deadline under build contention

Made the automated image-comparison tests more reliable when the computer is
busy building or testing other code. The tests still check that background image
work can be canceled safely and that comparison windows do not get stuck.

The updated tests passed repeated stress runs, and the complete verification
suite passed on Ubuntu. Code inspection found no issues requiring changes.

### Qualify updater notices in native release CI

Improved checks that ensure update packages include the required third-party
license notices.

Checks passed on Linux, Windows and macOS, and unsigned packages passed local
checks for all six supported targets. The final signed Windows packages still
need to pass the new inspection before release.

The license notes also clarify where a third-party component's historical source
version could not be established. See the
[evidence and limits](plans/2026-09-13-updater-notices.md) and the
[completed Linux checks](https://github.com/frathe/picfetch/actions/runs/34755687251).

### Mascot-circle hint for the Hypno Spiral

Added a hidden interaction with Trane and Finis, a wider speech bubble, and a fix
so Escape closes the manual even when its search field has focus.

On the welcome screen, move your pointer in ten circles in the same direction
around Trane's head within twenty seconds. This opens Finis or brings his window
to the front.

Make another ten circles within twenty seconds anywhere inside a Finis window
to reveal a speech bubble that stays visible. Click the bubble to open the manual
with an empty search field. The secret phrase remains in English; the hint in
parentheses appears in your selected language.

Targeted tests, translation checks and local build checks passed. The
[development record](finished_refactorings/2026-09-14-mascot-circle-hint.md)
includes hands-on feedback and identifies checks that could not be completed.

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
